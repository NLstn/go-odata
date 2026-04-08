package handlers

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/nlstn/go-odata/internal/observability"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/version"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// BatchHandler handles $batch requests for OData v4
type BatchHandler struct {
	db            *gorm.DB
	handlers      map[string]*EntityHandler
	service       http.Handler
	logger        *slog.Logger
	observability *observability.Config
	// preRequestHook is called before each sub-request is processed.
	preRequestHook func(r *http.Request) (context.Context, error)
	// maxBatchSize limits the maximum number of sub-requests allowed in a batch
	maxBatchSize int
}

// NewBatchHandler creates a new batch handler
func NewBatchHandler(db *gorm.DB, handlers map[string]*EntityHandler, service http.Handler, maxBatchSize int) *BatchHandler {
	return &BatchHandler{
		db:           db,
		handlers:     handlers,
		service:      service,
		logger:       slog.Default(),
		maxBatchSize: maxBatchSize,
	}
}

// SetLogger sets the logger for the batch handler.
func (h *BatchHandler) SetLogger(logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	h.logger = logger
}

// SetObservability configures observability for the batch handler.
func (h *BatchHandler) SetObservability(cfg *observability.Config) {
	h.observability = cfg
}

// SetPreRequestHook sets a hook that is called before each sub-request is processed.
// This enables authentication and context enrichment for all batch operations.
func (h *BatchHandler) SetPreRequestHook(hook func(r *http.Request) (context.Context, error)) {
	h.preRequestHook = hook
}

// batchRequest represents a single request within a batch
type batchRequest struct {
	Method    string
	URL       string
	Headers   http.Header
	Body      []byte
	ContentID string // Content-ID from the MIME part envelope (to be echoed in response)
}

// batchResponse represents a single response within a batch
type batchResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	ContentID  string // Content-ID to include in the response MIME part envelope
}

// jsonBatchRequestItem represents a single request object in a JSON batch envelope.
// Defined in OData JSON Format v4.01 §19.2.
type jsonBatchRequestItem struct {
	ID             string            `json:"id"`
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers,omitempty"`
	Body           json.RawMessage   `json:"body,omitempty"`
	AtomicityGroup string            `json:"atomicityGroup,omitempty"`
	DependsOn      []string          `json:"dependsOn,omitempty"`
}

// jsonBatchEnvelope is the top-level structure of a JSON batch request body.
type jsonBatchEnvelope struct {
	Requests []jsonBatchRequestItem `json:"requests"`
}

// jsonBatchResponseItem represents a single response object in a JSON batch response.
// Defined in OData JSON Format v4.01 §19.5.
type jsonBatchResponseItem struct {
	ID      string            `json:"id"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// jsonBatchResponseEnvelope is the top-level structure of a JSON batch response body.
type jsonBatchResponseEnvelope struct {
	Responses []jsonBatchResponseItem `json:"responses"`
}

// jsonGroupState holds the transaction state for a single atomicity group.
type jsonGroupState struct {
	tx                 *gorm.DB
	pendingEvents      []pendingChangeEvent
	contentIDLocations map[string]string
	failed             bool
	committed          bool
	// responseIndices tracks the indices of already-appended response items
	// in the outer responses slice, so they can be retroactively updated to
	// 424 Failed Dependency if the group transaction is rolled back.
	responseIndices []int
}

// HandleBatch handles the $batch endpoint
func (h *BatchHandler) HandleBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start tracing span for batch operation
	var batchSpan trace.Span
	if h.observability != nil {
		tracer := h.observability.Tracer()
		ctx, batchSpan = tracer.StartBatch(ctx, 0) // Size will be updated later
		defer batchSpan.End()
		r = r.WithContext(ctx)
	}

	if r.Method != http.MethodPost {
		if err := response.WriteError(w, r, http.StatusMethodNotAllowed, "Method not allowed",
			"Only POST method is supported for $batch requests"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	// Parse Content-Type header to extract boundary
	contentType := r.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		if err := response.WriteError(w, r, http.StatusBadRequest, "Invalid Content-Type",
			fmt.Sprintf("Failed to parse Content-Type header: %v", err)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	// Route to JSON batch handler for application/json Content-Type (OData 4.01).
	if mediaType == "application/json" {
		h.handleJSONBatch(w, r)
		return
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		if err := response.WriteError(w, r, http.StatusBadRequest, "Invalid Content-Type",
			"$batch requests must use multipart/mixed or application/json Content-Type"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	boundary, ok := params["boundary"]
	if !ok {
		if err := response.WriteError(w, r, http.StatusBadRequest, "Missing boundary",
			"Content-Type must include boundary parameter"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	// Parse multipart request
	reader := multipart.NewReader(r.Body, boundary)
	responses := []batchResponse{}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			if err := response.WriteError(w, r, http.StatusBadRequest, "Invalid batch request",
				fmt.Sprintf("Failed to read batch part: %v", err)); err != nil {
				h.logger.Error("Error writing error response", "error", err)
			}
			return
		}

		partContentType := part.Header.Get("Content-Type")
		partMediaType, partParams, err := mime.ParseMediaType(partContentType)
		if err != nil {
			responses = append(responses, h.createErrorResponse(http.StatusBadRequest, "Invalid part Content-Type"))
			continue
		}

		// Check if this is a changeset (nested multipart)
		if strings.HasPrefix(partMediaType, "multipart/") {
			changesetBoundary, ok := partParams["boundary"]
			if !ok {
				responses = append(responses, h.createErrorResponse(http.StatusBadRequest, "Missing changeset boundary"))
				continue
			}

			// Process changeset (atomic operations)
			// Pass remaining capacity to ensure changeset doesn't exceed batch limit
			remainingCapacity := h.maxBatchSize - len(responses)
			changesetResponses, exceeded := h.processChangeset(part, changesetBoundary, remainingCapacity, r)
			// Check if batch size limit was exceeded
			if exceeded {
				if err := response.WriteError(w, r, http.StatusRequestEntityTooLarge, "Batch size limit exceeded",
					fmt.Sprintf("Batch request contains too many sub-requests. Maximum allowed: %d", h.maxBatchSize)); err != nil {
					h.logger.Error("Error writing error response", "error", err)
				}
				return
			}
			responses = append(responses, changesetResponses...)
		} else if partMediaType == "application/http" {
			// Check batch size limit before processing single request
			if len(responses)+1 > h.maxBatchSize {
				if err := response.WriteError(w, r, http.StatusRequestEntityTooLarge, "Batch size limit exceeded",
					fmt.Sprintf("Batch request contains too many sub-requests. Maximum allowed: %d", h.maxBatchSize)); err != nil {
					h.logger.Error("Error writing error response", "error", err)
				}
				return
			}

			// Process single request
			// Capture Content-ID from MIME part envelope headers (per OData v4 spec, must be echoed in response)
			contentID := part.Header.Get("Content-ID")

			req, err := h.parseHTTPRequest(part)
			if err != nil {
				errResp := h.createErrorResponse(http.StatusBadRequest, fmt.Sprintf("Failed to parse request: %v", err))
				errResp.ContentID = contentID
				responses = append(responses, errResp)
				continue
			}

			req.ContentID = contentID
			resp := h.executeRequest(req, r)
			resp.ContentID = req.ContentID // Echo Content-ID in response
			responses = append(responses, resp)
		} else {
			// Check batch size limit before appending error response
			if len(responses)+1 > h.maxBatchSize {
				if err := response.WriteError(w, r, http.StatusRequestEntityTooLarge, "Batch size limit exceeded",
					fmt.Sprintf("Batch request contains too many sub-requests. Maximum allowed: %d", h.maxBatchSize)); err != nil {
					h.logger.Error("Error writing error response", "error", err)
				}
				return
			}
			responses = append(responses, h.createErrorResponse(http.StatusBadRequest, "Invalid part Content-Type"))
		}
	}

	// Write batch response
	h.writeBatchResponse(w, responses)

	// Update batch span with actual size and record batch metrics
	if h.observability != nil {
		batchSpan.SetAttributes(observability.BatchSizeAttr(len(responses)))
		h.observability.Metrics().RecordBatchSize(ctx, len(responses))
	}
}

// processChangeset processes a changeset (atomic operations)
// Returns responses and a boolean indicating if batch size limit was exceeded
func (h *BatchHandler) processChangeset(r io.Reader, boundary string, remainingCapacity int, parentReq *http.Request) ([]batchResponse, bool) {
	reader := multipart.NewReader(r, boundary)
	responses := []batchResponse{}

	// Start a transaction for the changeset
	tx := h.db.Begin()
	if tx.Error != nil {
		return []batchResponse{h.createErrorResponse(http.StatusInternalServerError, "Failed to start transaction")}, false
	}

	pendingEvents := make([]pendingChangeEvent, 0)

	// contentIDLocations maps a Content-ID value to the URL path of the entity created by
	// that request. This enables $<contentID> URL referencing per OData v4 spec §11.4.9.3:
	// subsequent requests in the same changeset may use "$<contentID>" as a URL prefix to
	// refer to the entity created by the request bearing that Content-ID.
	contentIDLocations := make(map[string]string)

	var hasError bool
	var sizeExceeded bool
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			hasError = true
			responses = append(responses, h.createErrorResponse(http.StatusBadRequest, fmt.Sprintf("Failed to read changeset part: %v", err)))
			break
		}

		// Check if adding this request would exceed the batch size limit
		if len(responses)+1 > remainingCapacity {
			sizeExceeded = true
			hasError = true
			break
		}

		// Capture Content-ID from MIME part envelope headers (per OData v4 spec, must be echoed in response)
		contentID := part.Header.Get("Content-ID")

		req, err := h.parseHTTPRequest(part)
		if err != nil {
			hasError = true
			errResp := h.createErrorResponse(http.StatusBadRequest, fmt.Sprintf("Failed to parse request: %v", err))
			errResp.ContentID = contentID
			responses = append(responses, errResp)
			break
		}

		req.ContentID = contentID

		// Resolve any $<contentID> URL reference before dispatching the request.
		// Per OData v4 spec §11.4.9.3, a request may use "$<contentID>" as a prefix in
		// its URL to refer to the entity created by the earlier request with that Content-ID.
		req.URL = resolveContentIDReference(req.URL, contentIDLocations)

		// Execute request within transaction
		resp := h.executeRequestInTransaction(req, tx, &pendingEvents, parentReq)
		resp.ContentID = req.ContentID // Echo Content-ID in response
		responses = append(responses, resp)

		// If the request succeeded and carried a Content-ID, record the Location of the
		// newly created entity so subsequent requests can reference it via $<contentID>.
		if contentID != "" && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if location := resp.Headers.Get("Location"); location != "" {
				contentIDLocations[contentID] = extractLocationPath(location)
			}
		}

		// If any request fails, mark as error
		if resp.StatusCode >= 400 {
			hasError = true
			break
		}
	}

	// Commit or rollback transaction
	if hasError {
		tx.Rollback()
	} else {
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			return []batchResponse{h.createErrorResponse(http.StatusInternalServerError, "Failed to commit transaction")}, false
		}
		flushPendingChangeEvents(pendingEvents)
	}

	return responses, sizeExceeded
}

// parseHTTPRequest parses an HTTP request from a multipart part
func (h *BatchHandler) parseHTTPRequest(r io.Reader) (*batchRequest, error) {
	reader := bufio.NewReader(r)

	requestLine, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read request line: %w", err)
	}
	requestLine = strings.TrimRight(requestLine, "\r\n")
	if requestLine == "" {
		return nil, fmt.Errorf("empty request")
	}

	// Parse HTTP request line: METHOD SP Request-URI [SP HTTP-Version]
	// Cannot use strings.Fields because OData URLs may contain spaces in query parameters
	// (e.g., $filter=Name eq 'John'), which would be incorrectly split.
	firstSpace := strings.IndexByte(requestLine, ' ')
	if firstSpace == -1 {
		return nil, fmt.Errorf("invalid request line: %s", requestLine)
	}
	method := requestLine[:firstSpace]
	rest := requestLine[firstSpace+1:]

	// Strip optional HTTP version suffix (e.g., " HTTP/1.1")
	reqURL := rest
	if lastSpace := strings.LastIndexByte(rest, ' '); lastSpace != -1 {
		if strings.HasPrefix(rest[lastSpace+1:], "HTTP/") {
			reqURL = rest[:lastSpace]
		}
	}

	if reqURL == "" {
		return nil, fmt.Errorf("invalid request line: %s", requestLine)
	}

	tp := textproto.NewReader(reader)
	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to read headers: %w", err)
	}

	headers := make(http.Header, len(mimeHeader))
	for key, values := range mimeHeader {
		for _, value := range values {
			headers.Add(key, value)
		}
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return &batchRequest{
		Method:  method,
		URL:     reqURL,
		Headers: headers,
		Body:    bytes.TrimSpace(body),
	}, nil
}

// executeRequest executes a single batch request.
// Per the OData specification, each sub-request should be treated as an independent request.
// Sub-requests are routed through the service handler which invokes the PreRequestHook.
// Cookies from the parent batch request are forwarded to enable cookie-based authentication.
func (h *BatchHandler) executeRequest(req *batchRequest, parentReq *http.Request) batchResponse {
	// Ensure URL has a leading slash to avoid httptest.NewRequest panic
	url := req.URL
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}

	// Encode spaces in query parameters to prevent httptest.NewRequest panic.
	// OData filter expressions contain spaces (e.g., "$filter=Name eq 'John'")
	// which cause httptest.NewRequest to misparse the URL as an HTTP request line.
	if idx := strings.IndexByte(url, '?'); idx != -1 {
		url = url[:idx] + "?" + strings.ReplaceAll(url[idx+1:], " ", "%20")
	}

	// Create an HTTP request
	httpReq := httptest.NewRequest(req.Method, url, bytes.NewReader(req.Body))
	for key, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	// Forward cookies from parent request to enable cookie-based authentication
	for _, cookie := range parentReq.Cookies() {
		httpReq.AddCookie(cookie)
	}

	// Execute request using the service handler.
	// The service handler invokes PreRequestHook for context enrichment.
	recorder := httptest.NewRecorder()
	h.service.ServeHTTP(recorder, httpReq)

	return batchResponse{
		StatusCode: recorder.Code,
		Headers:    recorder.Header(),
		Body:       recorder.Body.Bytes(),
	}
}

// executeRequestInTransaction executes a request within a transaction
func (h *BatchHandler) executeRequestInTransaction(req *batchRequest, tx *gorm.DB, pendingEvents *[]pendingChangeEvent, parentReq *http.Request) batchResponse {
	// Create temporary handlers that use the transaction
	txHandlers := make(map[string]*EntityHandler)
	for name, handler := range h.handlers {
		txHandler := NewEntityHandler(tx, handler.metadata, handler.logger)
		txHandler.SetNamespace(handler.namespace)
		txHandler.SetDeltaTracker(handler.tracker)
		txHandler.SetPolicy(handler.policy)
		if handler.entitiesMetadata != nil {
			txHandler.SetEntitiesMetadata(handler.entitiesMetadata)
		}
		if handler.keyGeneratorResolver != nil {
			txHandler.SetKeyGeneratorResolver(handler.keyGeneratorResolver)
		}
		txHandlers[name] = txHandler
	}

	// Create a service handler for the transaction
	getKeyString := func(components *response.ODataURLComponents) string {
		if components == nil {
			return ""
		}
		if components.EntityKey != "" {
			return components.EntityKey
		}
		if len(components.EntityKeyMap) == 0 {
			return ""
		}

		parts := make([]string, 0, len(components.EntityKeyMap))
		for key, value := range components.EntityKeyMap {
			isNumeric := true
			for _, ch := range value {
				if ch < '0' || ch > '9' {
					isNumeric = false
					break
				}
			}

			if isNumeric {
				parts = append(parts, fmt.Sprintf("%s=%s", key, value))
			} else {
				parts = append(parts, fmt.Sprintf("%s='%s'", key, value))
			}
		}

		return strings.Join(parts, ",")
	}

	handlePropertyRequest := func(w http.ResponseWriter, r *http.Request, handler *EntityHandler,
		components *response.ODataURLComponents, key string) {
		property := components.NavigationProperty
		if property == "" {
			if err := response.WriteError(w, r, http.StatusNotFound, "Property not found",
				"Requested property was not found on the target entity"); err != nil {
				h.logger.Error("Error writing error response", "error", err)
			}
			return
		}

		if components.IsCount {
			handler.HandleNavigationPropertyCount(w, r, key, property)
			return
		}

		if handler.IsNavigationProperty(property) {
			if components.IsValue {
				if err := response.WriteError(w, r, http.StatusBadRequest, "Invalid request",
					"$value is not supported on navigation properties"); err != nil {
					h.logger.Error("Error writing error response", "error", err)
				}
				return
			}
			handler.HandleNavigationProperty(w, r, key, property, components.IsRef)
			return
		}

		if handler.IsStreamProperty(property) {
			if components.IsRef {
				if err := response.WriteError(w, r, http.StatusBadRequest, "Invalid request",
					"$ref is not supported on stream properties"); err != nil {
					h.logger.Error("Error writing error response", "error", err)
				}
				return
			}
			handler.HandleStreamProperty(w, r, key, property, components.IsValue)
			return
		}

		if handler.IsStructuralProperty(property) {
			handler.HandleStructuralProperty(w, r, key, property, components.IsValue)
			return
		}

		if handler.IsComplexTypeProperty(property) {
			segments := components.PropertySegments
			if len(segments) == 0 {
				segments = []string{property}
			}
			handler.HandleComplexTypeProperty(w, r, key, segments, components.IsValue)
			return
		}

		if err := response.WriteError(w, r, http.StatusNotFound, "Property not found",
			fmt.Sprintf("'%s' is not a valid property for %s", property, components.EntitySet)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
	}

	serviceHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			if err := response.WriteError(w, r, http.StatusNotFound, "Resource not found",
				"Requested resource is not available in transactional batch requests"); err != nil {
				h.logger.Error("Error writing error response", "error", err)
			}
			return
		}

		switch path {
		case "$metadata":
			if err := response.WriteError(w, r, http.StatusNotFound, "Resource not found",
				"Metadata is not accessible inside transactional batch requests"); err != nil {
				h.logger.Error("Error writing error response", "error", err)
			}
			return
		case "$batch":
			if err := response.WriteError(w, r, http.StatusMethodNotAllowed, "Method not allowed",
				"Nested $batch requests are not supported within transactional batch requests"); err != nil {
				h.logger.Error("Error writing error response", "error", err)
			}
			return
		}

		components, err := response.ParseODataURLComponents(path)
		if err != nil {
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid URL", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}

		handler, exists := txHandlers[components.EntitySet]
		if !exists {
			if writeErr := response.WriteError(w, r, http.StatusNotFound, "Entity set not found",
				fmt.Sprintf("Entity set '%s' is not registered", components.EntitySet)); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}

		if components.TypeCast != "" {
			parts := strings.Split(components.TypeCast, ".")
			if len(parts) < 2 {
				if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid type cast",
					fmt.Sprintf("Type cast '%s' is not in the correct format (Namespace.TypeName)", components.TypeCast)); writeErr != nil {
					h.logger.Error("Error writing error response", "error", writeErr)
				}
				return
			}

			typeName := parts[len(parts)-1]
			ctx := WithTypeCast(r.Context(), typeName)
			r = r.WithContext(ctx)
		}

		hasKey := components.EntityKey != "" || len(components.EntityKeyMap) > 0
		keyString := getKeyString(components)
		isSingleton := handler.IsSingleton()

		switch {
		case components.IsCount:
			if hasKey && components.NavigationProperty != "" {
				handler.HandleNavigationPropertyCount(w, r, keyString, components.NavigationProperty)
				return
			}
			if !hasKey && components.NavigationProperty == "" {
				handler.HandleCount(w, r)
				return
			}
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid request",
				"$count is not supported on individual entities. Use $count on collections or navigation properties."); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
		case components.IsRef:
			if hasKey && components.NavigationProperty == "" {
				handler.HandleEntityRef(w, r, keyString)
				return
			}
			if !hasKey && components.NavigationProperty == "" {
				handler.HandleCollectionRef(w, r)
				return
			}
			handlePropertyRequest(w, r, handler, components, keyString)
		case isSingleton:
			if components.NavigationProperty != "" {
				handlePropertyRequest(w, r, handler, components, keyString)
			} else {
				handler.HandleSingleton(w, r)
			}
		case !hasKey:
			if components.IsValue {
				if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid request",
					"$value is not supported on entity collections. Use $value on individual properties: EntitySet(key)/PropertyName/$value"); writeErr != nil {
					h.logger.Error("Error writing error response", "error", writeErr)
				}
				return
			}
			if components.NavigationProperty != "" {
				if writeErr := response.WriteError(w, r, http.StatusNotFound, "Property or operation not found",
					fmt.Sprintf("'%s' is not a valid property, action, or function for %s", components.NavigationProperty, components.EntitySet)); writeErr != nil {
					h.logger.Error("Error writing error response", "error", writeErr)
				}
				return
			}
			handler.HandleCollection(w, r)
		default:
			if components.NavigationProperty != "" {
				handlePropertyRequest(w, r, handler, components, keyString)
				return
			}
			if components.IsValue {
				handler.HandleMediaEntityValue(w, r, keyString)
			} else {
				handler.HandleEntity(w, r, keyString)
			}
		}
	})

	// Ensure URL has a leading slash to avoid httptest.NewRequest panic
	url := req.URL
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}

	// Create an HTTP request
	httpReq := httptest.NewRequest(req.Method, url, bytes.NewReader(req.Body))
	for key, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	// Forward cookies from parent request to enable cookie-based authentication
	for _, cookie := range parentReq.Cookies() {
		httpReq.AddCookie(cookie)
	}

	// Call the pre-request hook if configured (for changeset sub-requests)
	ctx := httpReq.Context()
	if h.preRequestHook != nil {
		hookCtx, err := h.preRequestHook(httpReq)
		if err != nil {
			return h.createErrorResponse(http.StatusForbidden, err.Error())
		}
		if hookCtx != nil {
			ctx = hookCtx
		}
	}

	httpReq = httpReq.WithContext(withTransactionAndEvents(ctx, tx, pendingEvents))

	// Execute request
	recorder := httptest.NewRecorder()
	serviceHandler.ServeHTTP(recorder, httpReq)

	return batchResponse{
		StatusCode: recorder.Code,
		Headers:    recorder.Header(),
		Body:       recorder.Body.Bytes(),
	}
}

// createErrorResponse creates an error response
func (h *BatchHandler) createErrorResponse(statusCode int, message string) batchResponse {
	errorBody := fmt.Sprintf(`{"error":{"code":"%d","message":"%s"}}`, statusCode, message)
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	// Use version 4.01 for batch error responses (no request context available)
	headers[HeaderODataVersion] = []string{"4.01"}

	return batchResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       []byte(errorBody),
	}
}

// writeBatchResponse writes the batch response
func (h *BatchHandler) writeBatchResponse(w http.ResponseWriter, responses []batchResponse) {
	// Generate a boundary for the response
	boundary := fmt.Sprintf("batchresponse_%s", generateBoundary())

	w.Header().Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w.WriteHeader(http.StatusOK)

	// Write each response as a multipart part
	for _, resp := range responses {
		if _, err := fmt.Fprintf(w, "--%s\r\n", boundary); err != nil {
			h.logger.Error("Error writing boundary", "error", err)
			return
		}
		if _, err := fmt.Fprintf(w, "Content-Type: application/http\r\n"); err != nil {
			h.logger.Error("Error writing content type", "error", err)
			return
		}
		if _, err := fmt.Fprintf(w, "Content-Transfer-Encoding: binary\r\n"); err != nil {
			h.logger.Error("Error writing encoding", "error", err)
			return
		}
		// Echo Content-ID in the response MIME part envelope if it was present in the request
		// Per OData v4 spec section 11.7.4, the Content-ID MUST be echoed back
		if resp.ContentID != "" {
			if _, err := fmt.Fprintf(w, "Content-ID: %s\r\n", resp.ContentID); err != nil {
				h.logger.Error("Error writing Content-ID", "error", err)
				return
			}
		}
		if _, err := fmt.Fprintf(w, "\r\n"); err != nil {
			h.logger.Error("Error writing newline", "error", err)
			return
		}

		// Write status line
		if _, err := fmt.Fprintf(w, "HTTP/1.1 %d %s\r\n", resp.StatusCode, http.StatusText(resp.StatusCode)); err != nil {
			h.logger.Error("Error writing status line", "error", err)
			return
		}

		// Write headers
		for key, values := range resp.Headers {
			for _, value := range values {
				if _, err := fmt.Fprintf(w, "%s: %s\r\n", key, value); err != nil {
					h.logger.Error("Error writing header", "error", err)
					return
				}
			}
		}

		if _, err := fmt.Fprintf(w, "\r\n"); err != nil {
			h.logger.Error("Error writing newline", "error", err)
			return
		}

		// Write body
		if _, err := w.Write(resp.Body); err != nil {
			h.logger.Error("Error writing body", "error", err)
			return
		}
		if _, err := fmt.Fprintf(w, "\r\n"); err != nil {
			h.logger.Error("Error writing newline", "error", err)
			return
		}
	}

	// Write final boundary
	if _, err := fmt.Fprintf(w, "--%s--\r\n", boundary); err != nil {
		h.logger.Error("Error writing final boundary", "error", err)
	}
}

// handleJSONBatch processes a JSON-encoded batch request per OData JSON Format v4.01 §19.
//
// The request body must contain a top-level "requests" array; the response is a top-level
// "responses" array.  Key semantics implemented:
//   - dependsOn  – if any listed dependency has failed the request is skipped with 424.
//   - atomicityGroup – all requests sharing a group name run in a single database
//     transaction; if any individual request fails the transaction is rolled back and all
//     responses for that group (including ones already written) are changed to 424.
//   - Prefer: continue-on-error – when absent the handler stops at the first failure of a
//     standalone (non-group) request; when present it continues processing.
func (h *BatchHandler) handleJSONBatch(w http.ResponseWriter, r *http.Request) {
	// JSON batch is an OData 4.01-only feature.  Reject requests when the
	// negotiated version is 4.0 (i.e. the client sent OData-MaxVersion: 4.0).
	negotiated := version.GetVersion(r.Context())
	if negotiated.Major == 4 && negotiated.Minor == 0 {
		if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Not supported",
			"JSON batch ($batch with Content-Type: application/json) requires OData 4.01. "+
				"Use multipart/mixed for OData 4.0 batch requests."); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid batch request",
			fmt.Sprintf("Failed to read request body: %v", err)); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	var envelope jsonBatchEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid JSON batch request",
			fmt.Sprintf("Failed to parse JSON batch envelope: %v", err)); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Validate: check for duplicate IDs
	allIDs := make(map[string]bool, len(envelope.Requests))
	for _, item := range envelope.Requests {
		if item.ID == "" {
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid batch request",
				"Each request in a JSON batch must have a non-empty 'id' field"); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
		if allIDs[item.ID] {
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid batch request",
				fmt.Sprintf("Duplicate request id: %q", item.ID)); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
		allIDs[item.ID] = true
	}

	// Enforce batch size limit
	if len(envelope.Requests) > h.maxBatchSize {
		if writeErr := response.WriteError(w, r, http.StatusRequestEntityTooLarge, "Batch size limit exceeded",
			fmt.Sprintf("Batch request contains too many sub-requests. Maximum allowed: %d", h.maxBatchSize)); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Check Prefer: continue-on-error
	prefer := r.Header.Get("Prefer")
	continueOnError := strings.Contains(prefer, "continue-on-error")

	// Pre-scan: record the last array index for each atomicityGroup so we know
	// when to commit the group transaction.
	groupLastIdx := make(map[string]int)
	for i, item := range envelope.Requests {
		if item.AtomicityGroup != "" {
			groupLastIdx[item.AtomicityGroup] = i
		}
	}

	// Processing state
	failedIDs := make(map[string]bool)
	groups := make(map[string]*jsonGroupState)
	responses := make([]jsonBatchResponseItem, 0, len(envelope.Requests))

	for i, item := range envelope.Requests {
		// --- Check dependsOn ---
		depFailed := false
		for _, dep := range item.DependsOn {
			if failedIDs[dep] {
				depFailed = true
				break
			}
		}

		gs := groups[item.AtomicityGroup]
		groupAlreadyFailed := item.AtomicityGroup != "" && gs != nil && gs.failed

		if depFailed || groupAlreadyFailed {
			failedIDs[item.ID] = true
			resp := h.makeJSONFailedDependencyResponse(item.ID)
			if item.AtomicityGroup != "" {
				if gs == nil {
					gs = &jsonGroupState{contentIDLocations: map[string]string{}}
					groups[item.AtomicityGroup] = gs
				}
				gs.responseIndices = append(gs.responseIndices, len(responses))
			}
			responses = append(responses, resp)
			continue
		}

		// --- Build the internal batchRequest from the JSON item ---
		req, buildErr := h.jsonItemToBatchRequest(item)
		if buildErr != nil {
			failedIDs[item.ID] = true
			errResp := jsonBatchResponseItem{
				ID:     item.ID,
				Status: http.StatusBadRequest,
				Body:   jsonRawError(http.StatusBadRequest, buildErr.Error()),
			}
			if item.AtomicityGroup != "" {
				gs = h.getOrCreateJSONGroupState(groups, item.AtomicityGroup)
				if gs.tx != nil {
					gs.tx.Rollback()
				}
				gs.failed = true
				for _, idx := range gs.responseIndices {
					id := responses[idx].ID
					responses[idx] = h.makeJSONFailedDependencyResponse(id)
				}
				gs.responseIndices = append(gs.responseIndices, len(responses))
			}
			responses = append(responses, errResp)
			if !continueOnError {
				break
			}
			continue
		}

		// --- Execute the request ---
		var resp batchResponse

		if item.AtomicityGroup != "" {
			gs = h.getOrCreateJSONGroupState(groups, item.AtomicityGroup)

			// Start a transaction for the group if this is the first request.
			if gs.tx == nil {
				tx := h.db.Begin()
				if tx.Error != nil {
					gs.failed = true
					failedIDs[item.ID] = true
					errResp := jsonBatchResponseItem{
						ID:     item.ID,
						Status: http.StatusInternalServerError,
						Body:   jsonRawError(http.StatusInternalServerError, "Failed to start transaction"),
					}
					gs.responseIndices = append(gs.responseIndices, len(responses))
					responses = append(responses, errResp)
					if !continueOnError {
						break
					}
					continue
				}
				gs.tx = tx
			}

			// Resolve $<id> content-ID URL references within the group.
			req.URL = resolveContentIDReference(req.URL, gs.contentIDLocations)

			resp = h.executeRequestInTransaction(req, gs.tx, &gs.pendingEvents, r)

			if resp.StatusCode >= 400 {
				// Failure → rollback entire group.
				gs.tx.Rollback()
				gs.failed = true
				failedIDs[item.ID] = true

				// Retroactively mark all earlier group responses as 424.
				for _, idx := range gs.responseIndices {
					id := responses[idx].ID
					responses[idx] = h.makeJSONFailedDependencyResponse(id)
				}
				gs.responseIndices = append(gs.responseIndices, len(responses))
				responses = append(responses, h.batchResponseToJSONItem(item.ID, resp))
			} else {
				// Record content-ID location for later $<id> URL references.
				if item.ID != "" {
					if loc := resp.Headers.Get("Location"); loc != "" {
						gs.contentIDLocations[item.ID] = extractLocationPath(loc)
					}
				}

				gs.responseIndices = append(gs.responseIndices, len(responses))
				responses = append(responses, h.batchResponseToJSONItem(item.ID, resp))

				// Commit when we reach the last request in the group.
				if i == groupLastIdx[item.AtomicityGroup] {
					if commitErr := gs.tx.Commit().Error; commitErr != nil {
						gs.tx.Rollback()
						gs.failed = true
						failedIDs[item.ID] = true
						// Retroactively mark all group responses as 424.
						for _, idx := range gs.responseIndices {
							id := responses[idx].ID
							responses[idx] = h.makeJSONFailedDependencyResponse(id)
						}
					} else {
						gs.committed = true
						flushPendingChangeEvents(gs.pendingEvents)
					}
				}
			}
		} else {
			// Standalone (non-group) request.
			resp = h.executeRequest(req, r)
			responses = append(responses, h.batchResponseToJSONItem(item.ID, resp))

			if resp.StatusCode >= 400 {
				failedIDs[item.ID] = true
				if !continueOnError {
					// Fatal error: per OData JSON Format v4.01 §19.5, the service MUST
					// return a response for every request.  Fill remaining unprocessed
					// requests with 424 Failed Dependency and stop.
					for j := i + 1; j < len(envelope.Requests); j++ {
						rem := envelope.Requests[j]
						failedIDs[rem.ID] = true
						if rem.AtomicityGroup != "" {
							remGS := h.getOrCreateJSONGroupState(groups, rem.AtomicityGroup)
							remGS.responseIndices = append(remGS.responseIndices, len(responses))
						}
						responses = append(responses, h.makeJSONFailedDependencyResponse(rem.ID))
					}
					break
				}
			}
		}
	}

	// Roll back any group transaction that was never committed (early exit).
	for _, gs := range groups {
		if !gs.committed && !gs.failed && gs.tx != nil {
			gs.tx.Rollback()
		}
	}

	h.writeJSONBatchResponse(w, responses)

	// Update batch span and metrics (reuse observability infrastructure).
	if h.observability != nil {
		h.observability.Metrics().RecordBatchSize(r.Context(), len(responses))
	}
}

// getOrCreateJSONGroupState returns the existing group state or creates a new one.
func (h *BatchHandler) getOrCreateJSONGroupState(groups map[string]*jsonGroupState, name string) *jsonGroupState {
	if gs, ok := groups[name]; ok {
		return gs
	}
	gs := &jsonGroupState{
		contentIDLocations: make(map[string]string),
	}
	groups[name] = gs
	return gs
}

// jsonItemToBatchRequest converts a jsonBatchRequestItem into the internal batchRequest type.
func (h *BatchHandler) jsonItemToBatchRequest(item jsonBatchRequestItem) (*batchRequest, error) {
	if item.Method == "" {
		return nil, fmt.Errorf("request %q is missing required field 'method'", item.ID)
	}
	if item.URL == "" {
		return nil, fmt.Errorf("request %q is missing required field 'url'", item.ID)
	}

	// Strip an absolute base URL prefix, keeping only the path (and query string).
	reqURL := item.URL
	if u, parseErr := url.Parse(reqURL); parseErr == nil && u.IsAbs() {
		reqURL = u.RequestURI() // path + query + fragment
	}

	headers := make(http.Header, len(item.Headers))
	for k, v := range item.Headers {
		headers.Set(k, v)
	}

	// Marshal the body back to JSON bytes (it was parsed as json.RawMessage).
	var bodyBytes []byte
	if len(item.Body) > 0 && string(item.Body) != "null" {
		bodyBytes = item.Body
	}

	return &batchRequest{
		Method:    item.Method,
		URL:       reqURL,
		Headers:   headers,
		Body:      bodyBytes,
		ContentID: item.ID,
	}, nil
}

// batchResponseToJSONItem converts an internal batchResponse into a jsonBatchResponseItem.
func (h *BatchHandler) batchResponseToJSONItem(id string, resp batchResponse) jsonBatchResponseItem {
	headers := make(map[string]string, len(resp.Headers))
	for k, vals := range resp.Headers {
		if len(vals) > 0 {
			headers[k] = vals[0]
		}
	}
	if len(headers) == 0 {
		headers = nil
	}

	var bodyMsg json.RawMessage
	trimmed := bytes.TrimSpace(resp.Body)
	if len(trimmed) > 0 && json.Valid(trimmed) {
		bodyMsg = json.RawMessage(trimmed)
	}

	return jsonBatchResponseItem{
		ID:      id,
		Status:  resp.StatusCode,
		Headers: headers,
		Body:    bodyMsg,
	}
}

// makeJSONFailedDependencyResponse returns a 424 Failed Dependency response item.
func (h *BatchHandler) makeJSONFailedDependencyResponse(id string) jsonBatchResponseItem {
	return jsonBatchResponseItem{
		ID:     id,
		Status: http.StatusFailedDependency,
		Body:   jsonRawError(http.StatusFailedDependency, "Failed Dependency"),
	}
}

// writeJSONBatchResponse serialises the responses slice as a JSON batch response envelope.
func (h *BatchHandler) writeJSONBatchResponse(w http.ResponseWriter, responses []jsonBatchResponseItem) {
	envelope := jsonBatchResponseEnvelope{Responses: responses}
	out, err := json.Marshal(envelope)
	if err != nil {
		h.logger.Error("Error marshalling JSON batch response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(out); err != nil {
		h.logger.Error("Error writing JSON batch response", "error", err)
	}
}

// jsonRawError returns a json.RawMessage containing an OData-format error object.
func jsonRawError(code int, message string) json.RawMessage {
	// Defensive: marshal can theoretically fail with exotic string values, so
	// handle the error path even though in practice it will not occur.
	b, err := json.Marshal(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    fmt.Sprintf("%d", code),
			"message": message,
		},
	})
	if err != nil {
		// Fallback: return a safe static error payload.
		return json.RawMessage(`{"error":{"code":"500","message":"internal error"}}`)
	}
	return json.RawMessage(b)
}

// resolveContentIDReference replaces a leading $<contentID> token in rawURL with the
// corresponding entity path recorded in contentIDLocations.
//
// Per OData v4 spec §11.4.9.3, within a changeset a request may use "$<contentID>" as a
// URL prefix to refer to the resource created by the earlier request that bore that
// Content-ID header. Examples:
//
//	"$1"               →  "/Products(9)"
//	"/$1"              →  "/Products(9)"
//	"$1/Descriptions"  →  "/Products(9)/Descriptions"
//	"/$1/Descriptions" →  "/Products(9)/Descriptions"
func resolveContentIDReference(rawURL string, contentIDLocations map[string]string) string {
	// Strip an optional leading slash so we always work with "$<id>…" form.
	stripped := strings.TrimPrefix(rawURL, "/")
	if !strings.HasPrefix(stripped, "$") {
		return rawURL
	}

	for id, locPath := range contentIDLocations {
		prefix := "$" + id
		// Match exactly "$<id>" or "$<id>/" to avoid "$10" matching "$1".
		if stripped == prefix || strings.HasPrefix(stripped, prefix+"/") {
			suffix := stripped[len(prefix):]
			return locPath + suffix
		}
	}

	return rawURL
}

// extractLocationPath returns the path component of an absolute or relative URL.
// If the input cannot be parsed as a URL, it is returned unchanged.
func extractLocationPath(rawURL string) string {
	if parsed, err := url.Parse(rawURL); err == nil && parsed.Path != "" {
		return parsed.Path
	}
	return rawURL
}

// generateBoundary generates a random boundary string
func generateBoundary() string {
	const boundaryBytes = 18

	buf := make([]byte, boundaryBytes)
	if _, err := rand.Read(buf); err != nil {
		// Fallback to time-based boundary if the crypto reader fails
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(buf)
}
