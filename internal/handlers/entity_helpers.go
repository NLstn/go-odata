package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

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
		db = query.ApplyExpandOnly(db, queryOptions.Expand, h.metadata)
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
		if err := response.WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
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

	// Set Content-Type with dynamic metadata level
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(status)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		h.logger.Error("Error writing entity response", "error", err)
	}
}

// writeInvalidQueryError writes an invalid query error response
func (h *EntityHandler) writeInvalidQueryError(w http.ResponseWriter, err error) {
	if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error()); writeErr != nil {
		h.logger.Error("Error writing error response", "error", writeErr)
	}
}

// fetchAndVerifyEntity fetches an entity by key and handles errors
func (h *EntityHandler) fetchAndVerifyEntity(db *gorm.DB, entityKey string, w http.ResponseWriter) (interface{}, error) {
	entity := reflect.New(h.metadata.EntityType).Interface()

	query, err := h.buildKeyQuery(db, entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return nil, err
	}

	if err := query.First(entity).Error; err != nil {
		h.handleFetchError(w, err, entityKey)
		return nil, err
	}

	return entity, nil
}

// writeDatabaseError writes a database error response
func (h *EntityHandler) writeDatabaseError(w http.ResponseWriter, err error) {
	if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
		h.logger.Error("Error writing error response", "error", writeErr)
	}
}
