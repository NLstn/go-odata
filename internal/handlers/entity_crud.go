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

	// Validate that $top and $skip are not used on individual entities
	// Per OData v4 spec, these query options only apply to collections
	if queryOptions.Top != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$top query option is not applicable to individual entities"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}
	if queryOptions.Skip != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$skip query option is not applicable to individual entities"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
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
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Apply $select if specified (after ETag generation)
	if len(queryOptions.Select) > 0 {
		result = query.ApplySelectToEntity(result, queryOptions.Select, h.metadata, queryOptions.Expand)
	}

	// Build and write response
	h.writeEntityResponseWithETag(w, r, result, currentETag, http.StatusOK)
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
// and customizable success status codes while handling common response requirements.
func (h *EntityHandler) writeEntityResponseWithETag(w http.ResponseWriter, r *http.Request, result interface{}, precomputedETag string, status int) {
	// Check if the requested format is supported
	if !response.IsAcceptableFormat(r) {
		if err := response.WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses."); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
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
	// Validate Content-Type header
	if err := validateContentType(w, r); err != nil {
		return
	}

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

	// Validate that all properties in updateData are valid entity properties
	if err := h.validatePropertiesExist(updateData, w); err != nil {
		return nil, nil, err
	}

	// Validate data types
	if err := h.validateDataTypes(updateData); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid data type", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil, nil, err
	}

	// Validate that required fields are not being set to null
	if err := h.validateRequiredFieldsNotNull(updateData); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid value for required property", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
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

	if applied := pref.GetPreferenceApplied(); applied != "" {
		w.Header().Set(HeaderPreferenceApplied, applied)
	}

	if pref.ShouldReturnContent(false) {
		h.returnUpdatedEntity(w, r, db)
	} else {
		// For 204 No Content responses, we need to include OData-EntityId header
		// Fetch the entity to build its entity-id
		if db != nil {
			entity := reflect.New(h.metadata.EntityType).Interface()
			if err := db.First(entity).Error; err == nil {
				entityId := h.buildEntityLocation(r, entity)
				// Using helper function to preserve exact capitalization
				SetODataHeader(w, HeaderODataEntityId, entityId)
			}
		}
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

	h.writeEntityResponseWithETag(w, r, updatedEntity, "", http.StatusOK)
}

// handlePutEntity handles PUT requests for individual entities
// PUT performs a complete replacement according to OData v4 spec
func (h *EntityHandler) handlePutEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	// Validate Content-Type header
	if err := validateContentType(w, r); err != nil {
		return
	}

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

// validatePropertiesExist validates that all properties in updateData are valid entity properties
func (h *EntityHandler) validatePropertiesExist(updateData map[string]interface{}, w http.ResponseWriter) error {
	// Build a map of valid property names (both JSON names and struct field names)
	validProperties := make(map[string]bool)
	for _, prop := range h.metadata.Properties {
		validProperties[prop.JsonName] = true
		validProperties[prop.Name] = true
	}

	// Check each property in updateData
	for propName := range updateData {
		if !validProperties[propName] {
			err := fmt.Errorf("property '%s' does not exist on entity type '%s'", propName, h.metadata.EntityName)
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid property", err.Error()); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return err
		}
	}
	return nil
}

// HandleEntityRef handles GET requests for entity references (e.g., Products(1)/$ref)
func (h *EntityHandler) HandleEntityRef(w http.ResponseWriter, r *http.Request, entityKey string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for entity references", r.Method)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// Validate that $expand and $select are not used with $ref
	// According to OData v4 spec, $ref does not support $expand or $select
	queryParams := r.URL.Query()
	if queryParams.Get("$expand") != "" {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$expand is not supported with $ref"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}
	if queryParams.Get("$select") != "" {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$select is not supported with $ref"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Fetch the entity to ensure it exists
	entity := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	if err := db.First(entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if writeErr := response.WriteError(w, http.StatusNotFound, ErrMsgEntityNotFound,
				fmt.Sprintf("Entity with key '%s' not found", entityKey)); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
		} else {
			h.writeDatabaseError(w, err)
		}
		return
	}

	// Extract key values and build entity ID
	keyValues := response.ExtractEntityKeys(entity, h.metadata.KeyProperties)
	entityID := response.BuildEntityID(h.metadata.EntitySetName, keyValues)

	if err := response.WriteEntityReference(w, r, entityID); err != nil {
		fmt.Printf("Error writing entity reference: %v\n", err)
	}
}

// HandleCollectionRef handles GET requests for collection references (e.g., Products/$ref)
func (h *EntityHandler) HandleCollectionRef(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for collection references", r.Method)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// Validate that $expand and $select are not used with $ref
	// According to OData v4 spec, $ref only supports $filter, $top, $skip, $orderby, and $count
	queryParams := r.URL.Query()
	if queryParams.Get("$expand") != "" {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$expand is not supported with $ref"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}
	if queryParams.Get("$select") != "" {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$select is not supported with $ref"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Parse query options (support filtering, ordering, pagination for references)
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Get the total count if $count=true is specified
	totalCount := h.getTotalCount(queryOptions, w)
	if totalCount == nil && queryOptions.Count {
		return // Error already written
	}

	// Fetch the results
	results, err := h.fetchResults(queryOptions)
	if err != nil {
		h.writeDatabaseError(w, err)
		return
	}

	// Calculate next link if pagination is active and trim results if needed
	nextLink, needsTrimming := h.calculateNextLink(queryOptions, results, r)
	if needsTrimming && queryOptions.Top != nil {
		// Trim the results to $top (we fetched $top + 1 to check for more pages)
		results = h.trimResults(results, *queryOptions.Top)
	}

	// Build entity IDs for each entity
	var entityIDs []string
	sliceValue := reflect.ValueOf(results)
	if sliceValue.Kind() == reflect.Slice {
		for i := 0; i < sliceValue.Len(); i++ {
			entity := sliceValue.Index(i).Interface()
			keyValues := response.ExtractEntityKeys(entity, h.metadata.KeyProperties)
			entityID := response.BuildEntityID(h.metadata.EntitySetName, keyValues)
			entityIDs = append(entityIDs, entityID)
		}
	}

	if err := response.WriteEntityReferenceCollection(w, r, entityIDs, totalCount, nextLink); err != nil {
		fmt.Printf("Error writing entity reference collection: %v\n", err)
	}
}
