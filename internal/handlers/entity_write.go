package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
)

var errETagMismatch = errors.New("etag mismatch")

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

	// Call BeforeDelete hook if it exists
	if err := h.callBeforeDelete(entity, r); err != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Delete the entity
	if err := h.db.Delete(entity).Error; err != nil {
		h.writeDatabaseError(w, err)
		return
	}

	// Call AfterDelete hook if it exists
	if err := h.callAfterDelete(entity, r); err != nil {
		// Log the error but don't fail the request since the entity was already deleted
		fmt.Printf("AfterDelete hook failed: %v\n", err)
	}

	h.recordChange(entity, trackchanges.ChangeTypeDeleted)

	// Return 204 No Content according to OData v4 spec
	w.WriteHeader(http.StatusNoContent)
}

// handlePatchEntity handles PATCH requests for individual entities
func (h *EntityHandler) handlePatchEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	// Validate Content-Type header
	if err := validateContentType(w, r); err != nil {
		return
	}

	pref := preference.ParsePrefer(r)

	// Fetch and update the entity
	db, entity, err := h.fetchAndUpdateEntity(w, r, entityKey)
	if err != nil {
		return // Error already written
	}

	if entity != nil {
		if err := db.First(entity).Error; err == nil {
			h.recordChange(entity, trackchanges.ChangeTypeUpdated)
		} else {
			fmt.Printf("Error refreshing entity for change tracking: %v\n", err)
		}
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
			return nil, nil, errETagMismatch
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
	// Skip validation for @odata.bind annotations
	if err := h.validatePropertiesExistForUpdate(updateData, w); err != nil {
		return nil, nil, err
	}

	// Process @odata.bind annotations to update navigation property relationships
	// This also adds the foreign key values to updateData
	if err := h.processODataBindAnnotationsForUpdate(entity, updateData, h.db); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid @odata.bind annotation", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil, nil, err
	}

	// Remove @odata.bind annotations from updateData before updating the entity
	// as they are not actual properties (the foreign keys have been added)
	h.removeODataBindAnnotations(updateData)

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

	// Call BeforeUpdate hook if it exists
	if err := h.callBeforeUpdate(entity, r); err != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil, nil, err
	}

	if err := h.db.Model(entity).Updates(updateData).Error; err != nil {
		h.writeDatabaseError(w, err)
		return nil, nil, err
	}

	// Call AfterUpdate hook if it exists
	if err := h.callAfterUpdate(entity, r); err != nil {
		// Log the error but don't fail the request since the entity was already updated
		fmt.Printf("AfterUpdate hook failed: %v\n", err)
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

	if db != nil {
		replacedEntity := reflect.New(h.metadata.EntityType).Interface()
		if err := db.First(replacedEntity).Error; err == nil {
			h.recordChange(replacedEntity, trackchanges.ChangeTypeUpdated)
		} else {
			fmt.Printf("Error refreshing entity for change tracking: %v\n", err)
		}
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
			return nil, errETagMismatch
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

	// Call BeforeUpdate hook if it exists (PUT is also an update operation)
	if err := h.callBeforeUpdate(entity, r); err != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil, err
	}

	if err := h.db.Model(entity).Select("*").Updates(replacementEntity).Error; err != nil {
		h.writeDatabaseError(w, err)
		return nil, err
	}

	// Call AfterUpdate hook if it exists
	if err := h.callAfterUpdate(entity, r); err != nil {
		// Log the error but don't fail the request since the entity was already updated
		fmt.Printf("AfterUpdate hook failed: %v\n", err)
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

// validatePropertiesExistForUpdate validates that all properties in updateData are valid entity properties
// This version allows @odata.bind annotations for navigation properties
func (h *EntityHandler) validatePropertiesExistForUpdate(updateData map[string]interface{}, w http.ResponseWriter) error {
	// Build a map of valid property names (both JSON names and struct field names)
	validProperties := make(map[string]bool)
	for _, prop := range h.metadata.Properties {
		validProperties[prop.JsonName] = true
		validProperties[prop.Name] = true
		// Allow @odata.bind annotations for navigation properties
		if prop.IsNavigationProp {
			validProperties[prop.JsonName+"@odata.bind"] = true
			validProperties[prop.Name+"@odata.bind"] = true
		}
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

// removeODataBindAnnotations removes @odata.bind annotations from the update data
// as they are not actual entity properties and should not be passed to GORM
func (h *EntityHandler) removeODataBindAnnotations(updateData map[string]interface{}) {
	for key := range updateData {
		if strings.HasSuffix(key, "@odata.bind") {
			delete(updateData, key)
		}
	}
}
