package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// HandleEntity handles GET, HEAD, DELETE, PATCH, PUT, and OPTIONS requests for individual entities
func (h *EntityHandler) HandleEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetEntity(w, r, entityKey)
	case http.MethodDelete:
		h.handleDeleteEntity(w, r, entityKey)
	case http.MethodPatch:
		h.handlePatchEntity(w, r, entityKey)
	case http.MethodPut:
		h.handlePutEntity(w, r, entityKey)
	case http.MethodOptions:
		h.handleOptionsEntity(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for individual entities", r.Method)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
	}
}

// handleOptionsEntity handles OPTIONS requests for individual entities
func (h *EntityHandler) handleOptionsEntity(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, DELETE, PATCH, PUT, OPTIONS")
	w.Header().Set(HeaderODataVersion, "4.0")
	w.WriteHeader(http.StatusOK)
}

// handleGetEntity handles GET requests for individual entities
func (h *EntityHandler) handleGetEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	// Parse query options for $expand and $select
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		h.writeInvalidQueryError(w, err)
		return
	}

	// Fetch the entity
	result, err := h.fetchEntityByKey(entityKey, queryOptions)
	if err != nil {
		h.handleFetchError(w, err, entityKey)
		return
	}

	// Check If-None-Match header if ETag is configured (before applying select)
	var currentETag string
	if h.metadata.ETagProperty != nil {
		ifNoneMatch := r.Header.Get(HeaderIfNoneMatch)
		currentETag = etag.Generate(result, h.metadata)

		// If ETags match, return 304 Not Modified
		if !etag.NoneMatch(ifNoneMatch, currentETag) {
			w.Header().Set(HeaderETag, currentETag)
			w.Header().Set(HeaderODataVersion, "4.0")
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Apply $select if specified (after ETag generation)
	if len(queryOptions.Select) > 0 {
		result = query.ApplySelectToEntity(result, queryOptions.Select, h.metadata, queryOptions.Expand)
	}

	// Build and write response
	h.writeEntityResponseWithETag(w, r, result, currentETag)
}

// fetchEntityByKey fetches an entity by its key with optional expand
func (h *EntityHandler) fetchEntityByKey(entityKey string, queryOptions *query.QueryOptions) (interface{}, error) {
	result := reflect.New(h.metadata.EntityType).Interface()

	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		return nil, err
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
func (h *EntityHandler) writeEntityResponseWithETag(w http.ResponseWriter, r *http.Request, result interface{}, precomputedETag string) {
	// Check if the requested format is supported
	if !response.IsAcceptableFormat(r) {
		if err := response.WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses."); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	contextURL := fmt.Sprintf(ODataContextFormat, response.BuildBaseURL(r), h.metadata.EntitySetName)
	odataResponse := h.buildOrderedEntityResponse(result, contextURL)

	// Use pre-computed ETag if provided, otherwise generate it
	etagValue := precomputedETag
	if etagValue == "" && h.metadata.ETagProperty != nil {
		etagValue = etag.Generate(result, h.metadata)
	}

	if etagValue != "" {
		w.Header().Set(HeaderETag, etagValue)
	}

	// Set Content-Type with dynamic metadata level
	metadataLevel := response.GetODataMetadataLevel(r)
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.Header().Set(HeaderODataVersion, "4.0")
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		fmt.Printf(LogMsgErrorWritingEntityResponse, err)
	}
}

// writeInvalidQueryError writes an invalid query error response
func (h *EntityHandler) writeInvalidQueryError(w http.ResponseWriter, err error) {
	if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error()); writeErr != nil {
		fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
	}
}

// handleDeleteEntity handles DELETE requests for individual entities
func (h *EntityHandler) handleDeleteEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	// Fetch and delete the entity
	entity, err := h.fetchAndVerifyEntity(entityKey, w)
	if err != nil {
		return // Error already written
	}

	// Check If-Match header if ETag is configured
	if h.metadata.ETagProperty != nil {
		ifMatch := r.Header.Get(HeaderIfMatch)
		currentETag := etag.Generate(entity, h.metadata)

		if !etag.Match(ifMatch, currentETag) {
			if writeErr := response.WriteError(w, http.StatusPreconditionFailed, ErrMsgPreconditionFailed,
				ErrDetailPreconditionFailed); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}
	}

	// Delete the entity
	if err := h.db.Delete(entity).Error; err != nil {
		h.writeDatabaseError(w, err)
		return
	}

	// Return 204 No Content according to OData v4 spec
	w.Header().Set(HeaderODataVersion, "4.0")
	w.WriteHeader(http.StatusNoContent)
}

// fetchAndVerifyEntity fetches an entity by key and handles errors
func (h *EntityHandler) fetchAndVerifyEntity(entityKey string, w http.ResponseWriter) (interface{}, error) {
	entity := reflect.New(h.metadata.EntityType).Interface()

	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil, err
	}

	if err := db.First(entity).Error; err != nil {
		h.handleFetchError(w, err, entityKey)
		return nil, err
	}

	return entity, nil
}

// writeDatabaseError writes a database error response
func (h *EntityHandler) writeDatabaseError(w http.ResponseWriter, err error) {
	if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
		fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
	}
}

// handlePatchEntity handles PATCH requests for individual entities
func (h *EntityHandler) handlePatchEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	pref := preference.ParsePrefer(r)

	// Fetch and update the entity
	db, _, err := h.fetchAndUpdateEntity(w, r, entityKey)
	if err != nil {
		return // Error already written
	}

	// Write response based on preference
	h.writeUpdateResponse(w, r, pref, db)
}

// fetchAndUpdateEntity fetches an entity and applies PATCH updates
func (h *EntityHandler) fetchAndUpdateEntity(w http.ResponseWriter, r *http.Request, entityKey string) (*gorm.DB, interface{}, error) {
	entity := reflect.New(h.metadata.EntityType).Interface()

	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil, nil, err
	}

	if err := db.First(entity).Error; err != nil {
		h.handleFetchError(w, err, entityKey)
		return nil, nil, err
	}

	// Check If-Match header if ETag is configured
	if h.metadata.ETagProperty != nil {
		ifMatch := r.Header.Get(HeaderIfMatch)
		currentETag := etag.Generate(entity, h.metadata)

		if !etag.Match(ifMatch, currentETag) {
			if writeErr := response.WriteError(w, http.StatusPreconditionFailed, ErrMsgPreconditionFailed,
				ErrDetailPreconditionFailed); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return nil, nil, err
		}
	}

	// Parse the request body to get the update data
	updateData, err := h.parsePatchRequestBody(r, w)
	if err != nil {
		return nil, nil, err
	}

	if err := h.validateKeyPropertiesNotUpdated(updateData, w); err != nil {
		return nil, nil, err
	}

	if err := h.db.Model(entity).Updates(updateData).Error; err != nil {
		h.writeDatabaseError(w, err)
		return nil, nil, err
	}

	return db, entity, nil
}

// writeUpdateResponse writes the response for PATCH/PUT operations based on preferences
func (h *EntityHandler) writeUpdateResponse(w http.ResponseWriter, r *http.Request, pref *preference.Preference, db *gorm.DB) {
	w.Header().Set(HeaderODataVersion, "4.0")

	if applied := pref.GetPreferenceApplied(); applied != "" {
		w.Header().Set(HeaderPreferenceApplied, applied)
	}

	if pref.ShouldReturnContent(false) {
		h.returnUpdatedEntity(w, r, db)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// returnUpdatedEntity fetches and returns the updated entity
func (h *EntityHandler) returnUpdatedEntity(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	updatedEntity := reflect.New(h.metadata.EntityType).Interface()
	if err := db.First(updatedEntity).Error; err != nil {
		h.writeDatabaseError(w, err)
		return
	}

	contextURL := fmt.Sprintf(ODataContextFormat, response.BuildBaseURL(r), h.metadata.EntitySetName)
	odataResponse := h.buildOrderedEntityResponse(updatedEntity, contextURL)

	// Generate and set ETag header if entity has an ETag property
	if etagValue := etag.Generate(updatedEntity, h.metadata); etagValue != "" {
		w.Header().Set(HeaderETag, etagValue)
	}

	w.Header().Set(HeaderContentType, ContentTypeJSON)
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		fmt.Printf(LogMsgErrorWritingEntityResponse, err)
	}
}

// handlePutEntity handles PUT requests for individual entities
// PUT performs a complete replacement according to OData v4 spec
func (h *EntityHandler) handlePutEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	pref := preference.ParsePrefer(r)

	// Fetch and replace the entity
	db, err := h.fetchAndReplaceEntity(w, r, entityKey)
	if err != nil {
		return // Error already written
	}

	// Write response based on preference
	h.writeUpdateResponse(w, r, pref, db)
}

// fetchAndReplaceEntity fetches an entity and performs PUT replacement
func (h *EntityHandler) fetchAndReplaceEntity(w http.ResponseWriter, r *http.Request, entityKey string) (*gorm.DB, error) {
	entity := reflect.New(h.metadata.EntityType).Interface()

	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil, err
	}

	if err := db.First(entity).Error; err != nil {
		h.handleFetchError(w, err, entityKey)
		return nil, err
	}

	// Check If-Match header if ETag is configured
	if h.metadata.ETagProperty != nil {
		ifMatch := r.Header.Get(HeaderIfMatch)
		currentETag := etag.Generate(entity, h.metadata)

		if !etag.Match(ifMatch, currentETag) {
			if writeErr := response.WriteError(w, http.StatusPreconditionFailed, ErrMsgPreconditionFailed,
				ErrDetailPreconditionFailed); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return nil, err
		}
	}

	// Create a new instance for the replacement data
	replacementEntity := reflect.New(h.metadata.EntityType).Interface()
	if err := json.NewDecoder(r.Body).Decode(replacementEntity); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error())); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil, err
	}

	if err := h.preserveKeyProperties(entity, replacementEntity); err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil, err
	}

	if err := h.db.Model(entity).Select("*").Updates(replacementEntity).Error; err != nil {
		h.writeDatabaseError(w, err)
		return nil, err
	}

	return db, nil
}

// preserveKeyProperties copies key property values from source to destination
func (h *EntityHandler) preserveKeyProperties(source, destination interface{}) error {
	sourceVal := reflect.ValueOf(source).Elem()
	destVal := reflect.ValueOf(destination).Elem()

	for _, keyProp := range h.metadata.KeyProperties {
		sourceField := sourceVal.FieldByName(keyProp.Name)
		destField := destVal.FieldByName(keyProp.Name)

		if !sourceField.IsValid() || !destField.IsValid() {
			return fmt.Errorf("key property '%s' not found", keyProp.Name)
		}

		if !destField.CanSet() {
			return fmt.Errorf("cannot set key property '%s'", keyProp.Name)
		}

		destField.Set(sourceField)
	}

	return nil
}

// parsePatchRequestBody parses the JSON request body for PATCH operations
func (h *EntityHandler) parsePatchRequestBody(r *http.Request, w http.ResponseWriter) (map[string]interface{}, error) {
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error())); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil, err
	}
	return updateData, nil
}

// validateKeyPropertiesNotUpdated validates that key properties are not being updated
func (h *EntityHandler) validateKeyPropertiesNotUpdated(updateData map[string]interface{}, w http.ResponseWriter) error {
	for _, keyProp := range h.metadata.KeyProperties {
		if _, exists := updateData[keyProp.JsonName]; exists {
			err := fmt.Errorf("key property '%s' cannot be modified", keyProp.JsonName)
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Cannot update key property", err.Error()); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return err
		}
		// Also check using the struct field name
		if _, exists := updateData[keyProp.Name]; exists {
			err := fmt.Errorf("key property '%s' cannot be modified", keyProp.Name)
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Cannot update key property", err.Error()); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return err
		}
	}
	return nil
}
