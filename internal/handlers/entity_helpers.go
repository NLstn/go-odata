package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

type requestError struct {
	StatusCode int
	ErrorCode  string
	Message    string
}

func (e *requestError) Error() string {
	return e.Message
}

func (h *EntityHandler) writeRequestError(w http.ResponseWriter, r *http.Request, err error, defaultStatus int, defaultCode string) {
	if err == nil {
		return
	}

	// Check for GeospatialNotEnabledError first
	if IsGeospatialNotEnabledError(err) {
		if writeErr := response.WriteError(w, r, http.StatusNotImplemented, "Geospatial features not enabled", err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	var reqErr *requestError
	if errors.As(err, &reqErr) {
		status := reqErr.StatusCode
		if status == 0 {
			status = defaultStatus
		}

		code := reqErr.ErrorCode
		if code == "" {
			code = defaultCode
		}

		message := reqErr.Message
		if message == "" {
			message = err.Error()
		}

		if writeErr := response.WriteError(w, r, status, code, message); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	if writeErr := response.WriteError(w, r, defaultStatus, defaultCode, err.Error()); writeErr != nil {
		h.logger.Error("Error writing error response", "error", writeErr)
	}
}

func (h *EntityHandler) parseSingleEntityQueryOptions(r *http.Request) (*query.QueryOptions, error) {
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		return nil, &requestError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  ErrMsgInvalidQueryOptions,
			Message:    err.Error(),
		}
	}

	// Check if geospatial operations are used but not enabled
	if queryOptions.Filter != nil && query.ContainsGeospatialOperations(queryOptions.Filter) {
		if !h.geospatialEnabled {
			return nil, &GeospatialNotEnabledError{}
		}
	}

	if queryOptions.Top != nil {
		return nil, &requestError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  ErrMsgInvalidQueryOptions,
			Message:    "$top query option is not applicable to individual entities",
		}
	}

	if queryOptions.Skip != nil {
		return nil, &requestError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  ErrMsgInvalidQueryOptions,
			Message:    "$skip query option is not applicable to individual entities",
		}
	}

	if queryOptions.Index {
		return nil, &requestError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  ErrMsgInvalidQueryOptions,
			Message:    "$index query option is not applicable to individual entities",
		}
	}

	if err := applyPolicyFiltersToExpand(r, h.policy, h.metadata, queryOptions.Expand); err != nil {
		return nil, &requestError{
			StatusCode: http.StatusForbidden,
			ErrorCode:  "Authorization failed",
			Message:    err.Error(),
		}
	}

	return queryOptions, nil
}

// fetchEntityByKey fetches an entity by its key with optional expand
func (h *EntityHandler) fetchEntityByKey(ctx context.Context, entityKey string, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	result := reflect.New(h.metadata.EntityType).Interface()

	db, err := h.buildKeyQuery(h.db.WithContext(ctx), entityKey)
	if err != nil {
		return nil, err
	}

	if len(scopes) > 0 {
		db = db.Scopes(scopes...)
	}

	// Apply expand (preload navigation properties) if specified
	if len(queryOptions.Expand) > 0 {
		db = query.ApplyExpandOnly(db, queryOptions.Expand, h.metadata, h.logger)
	}

	if err := db.First(result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

// writeEntityResponseWithETag writes an entity response with an optional pre-computed ETag
// and customizable success status codes while handling common response requirements.
func (h *EntityHandler) writeEntityResponseWithETag(w http.ResponseWriter, r *http.Request, result interface{}, precomputedETag string, status int) {
	// Check if the requested format is supported
	if !response.IsAcceptableFormat(r) {
		if err := response.WriteError(w, r, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses."); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	// Get metadata level
	metadataLevel := response.GetODataMetadataLevel(r)
	contextURL := fmt.Sprintf(ODataContextFormat, response.BuildBaseURL(r), h.metadata.EntitySetName)

	// Use pre-computed ETag if provided, otherwise generate it
	etagValue := precomputedETag
	if etagValue == "" && h.metadata.ETagProperty != nil {
		etagValue = etag.Generate(result, h.metadata)
	}

	odataResponse := h.buildOrderedEntityResponseWithMetadata(result, contextURL, metadataLevel, r, etagValue)

	if etagValue != "" {
		w.Header().Set(HeaderETag, etagValue)
	}
	if status == 0 {
		status = http.StatusOK
	}

	if r.Method == http.MethodHead {
		jsonBytes, err := json.Marshal(odataResponse)
		if err != nil {
			h.logger.Error("Error marshaling entity response for HEAD request", "error", err)
			if writeErr := response.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to marshal response for HEAD request."); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
		w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))
		w.WriteHeader(status)
		return
	}

	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		h.logger.Error("Error writing entity response", "error", err)
	}
}

// writeInvalidQueryError writes an invalid query error response
func (h *EntityHandler) writeInvalidQueryError(w http.ResponseWriter, r *http.Request, err error) {
	if writeErr := response.WriteError(w, r, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error()); writeErr != nil {
		h.logger.Error("Error writing error response", "error", writeErr)
	}
}

// fetchAndVerifyEntity fetches an entity by key and handles errors
func (h *EntityHandler) fetchAndVerifyEntity(db *gorm.DB, entityKey string, w http.ResponseWriter, r *http.Request) (interface{}, error) {
	entity := reflect.New(h.metadata.EntityType).Interface()

	query, err := h.buildKeyQuery(db, entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, r, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return nil, err
	}

	if err := query.First(entity).Error; err != nil {
		h.handleFetchError(w, r, err, entityKey)
		return nil, err
	}

	return entity, nil
}

// writeDatabaseError writes a database error response
func (h *EntityHandler) writeDatabaseError(w http.ResponseWriter, r *http.Request, err error) {
	if writeErr := response.WriteError(w, r, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
		h.logger.Error("Error writing error response", "error", writeErr)
	}
}
