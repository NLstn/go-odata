package handlers

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"time"

	"github.com/nlstn/go-odata/internal/observability"
	"github.com/nlstn/go-odata/internal/response"
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
}

// NewBatchHandler creates a new batch handler
func NewBatchHandler(db *gorm.DB, handlers map[string]*EntityHandler, service http.Handler) *BatchHandler {
	return &BatchHandler{
		db:       db,
		handlers: handlers,
		service:  service,
		logger:   slog.Default(),
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
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			"Only POST method is supported for $batch requests"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	// Parse Content-Type header to extract boundary
	contentType := r.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid Content-Type",
			fmt.Sprintf("Failed to parse Content-Type header: %v", err)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid Content-Type",
			"$batch requests must use multipart/mixed Content-Type"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	boundary, ok := params["boundary"]
	if !ok {
		if err := response.WriteError(w, http.StatusBadRequest, "Missing boundary",
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
			if err := response.WriteError(w, http.StatusBadRequest, "Invalid batch request",
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
			changesetResponses := h.processChangeset(part, changesetBoundary)
			responses = append(responses, changesetResponses...)
		} else if partMediaType == "application/http" {
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
			resp := h.executeRequest(req)
			resp.ContentID = req.ContentID // Echo Content-ID in response
			responses = append(responses, resp)
		} else {
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
func (h *BatchHandler) processChangeset(r io.Reader, boundary string) []batchResponse {
	reader := multipart.NewReader(r, boundary)
	responses := []batchResponse{}

	// Start a transaction for the changeset
	tx := h.db.Begin()
	if tx.Error != nil {
		return []batchResponse{h.createErrorResponse(http.StatusInternalServerError, "Failed to start transaction")}
	}

	pendingEvents := make([]pendingChangeEvent, 0)

	var hasError bool
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

		// Execute request within transaction
		resp := h.executeRequestInTransaction(req, tx, &pendingEvents)
		resp.ContentID = req.ContentID // Echo Content-ID in response
		responses = append(responses, resp)

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
			return []batchResponse{h.createErrorResponse(http.StatusInternalServerError, "Failed to commit transaction")}
		}
		flushPendingChangeEvents(pendingEvents)
	}

	return responses
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

	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
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
		Method:  parts[0],
		URL:     parts[1],
		Headers: headers,
		Body:    bytes.TrimSpace(body),
	}, nil
}

// executeRequest executes a single batch request.
// Per the OData specification, each sub-request should be treated as an independent request.
// Sub-requests are routed through the service handler which invokes the PreRequestHook.
func (h *BatchHandler) executeRequest(req *batchRequest) batchResponse {
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
func (h *BatchHandler) executeRequestInTransaction(req *batchRequest, tx *gorm.DB, pendingEvents *[]pendingChangeEvent) batchResponse {
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
			if err := response.WriteError(w, http.StatusNotFound, "Property not found",
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
				if err := response.WriteError(w, http.StatusBadRequest, "Invalid request",
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
				if err := response.WriteError(w, http.StatusBadRequest, "Invalid request",
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

		if err := response.WriteError(w, http.StatusNotFound, "Property not found",
			fmt.Sprintf("'%s' is not a valid property for %s", property, components.EntitySet)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
	}

	serviceHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			if err := response.WriteError(w, http.StatusNotFound, "Resource not found",
				"Requested resource is not available in transactional batch requests"); err != nil {
				h.logger.Error("Error writing error response", "error", err)
			}
			return
		}

		switch path {
		case "$metadata":
			if err := response.WriteError(w, http.StatusNotFound, "Resource not found",
				"Metadata is not accessible inside transactional batch requests"); err != nil {
				h.logger.Error("Error writing error response", "error", err)
			}
			return
		case "$batch":
			if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
				"Nested $batch requests are not supported within transactional batch requests"); err != nil {
				h.logger.Error("Error writing error response", "error", err)
			}
			return
		}

		components, err := response.ParseODataURLComponents(path)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid URL", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}

		handler, exists := txHandlers[components.EntitySet]
		if !exists {
			if writeErr := response.WriteError(w, http.StatusNotFound, "Entity set not found",
				fmt.Sprintf("Entity set '%s' is not registered", components.EntitySet)); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}

		if components.TypeCast != "" {
			parts := strings.Split(components.TypeCast, ".")
			if len(parts) < 2 {
				if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid type cast",
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
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
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
				if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
					"$value is not supported on entity collections. Use $value on individual properties: EntitySet(key)/PropertyName/$value"); writeErr != nil {
					h.logger.Error("Error writing error response", "error", writeErr)
				}
				return
			}
			if components.NavigationProperty != "" {
				if writeErr := response.WriteError(w, http.StatusNotFound, "Property or operation not found",
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
