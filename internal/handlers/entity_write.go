package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

var errETagMismatch = errors.New("etag mismatch")
var errAuthorizationDenied = errors.New("authorization denied")

// handleDeleteEntity handles DELETE requests for individual entities
func (h *EntityHandler) handleDeleteEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	ctx := r.Context()

	// Start tracing span for delete operation
	var span trace.Span
	if h.observability != nil {
		tracer := h.observability.Tracer()
		ctx, span = tracer.StartEntityDelete(ctx, h.metadata.EntitySetName, entityKey)
		defer span.End()
		r = r.WithContext(ctx)
	}

	// Check if there's an overwrite handler
	if h.overwrite.hasDelete() {
		h.handleDeleteEntityOverwrite(w, r, entityKey)
		return
	}

	// Check if this is a virtual entity without overwrite handler
	if h.metadata.IsVirtual {
		if err := response.WriteError(w, r, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			"Virtual entities require an overwrite handler for Delete operation"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	if !h.enforceDeleteRestrictions(w, r) {
		return
	}

	var (
		entity       interface{}
		changeEvents []changeEvent
	)

	if err := h.runInTransaction(ctx, r, func(tx *gorm.DB, hookReq *http.Request) error {
		fetched, err := h.fetchAndVerifyEntity(tx, entityKey, w, r)
		if err != nil {
			return newTransactionHandledError(err)
		}
		entity = fetched

		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptorWithEntity(h.metadata, entityKey, entity, nil), auth.OperationDelete, h.logger) {
			return newTransactionHandledError(errAuthorizationDenied)
		}

		if h.metadata.ETagProperty != nil {
			ifMatch := r.Header.Get(HeaderIfMatch)
			currentETag := etag.Generate(entity, h.metadata)

			if !etag.Match(ifMatch, currentETag) {
				if writeErr := response.WriteError(w, r, http.StatusPreconditionFailed, ErrMsgPreconditionFailed,
					ErrDetailPreconditionFailed); writeErr != nil {
					h.logger.Error("Error writing error response", "error", writeErr)
				}
				return newTransactionHandledError(errETagMismatch)
			}
		}

		if err := h.callBeforeDelete(entity, hookReq); err != nil {
			h.writeHookError(w, r, err, http.StatusForbidden, "Authorization failed")
			return newTransactionHandledError(err)
		}

		if err := tx.Delete(entity).Error; err != nil {
			h.writeDatabaseError(w, r, err)
			return newTransactionHandledError(err)
		}

		if err := h.callAfterDelete(entity, hookReq); err != nil {
			h.logger.Error("AfterDelete hook failed", "error", err)
		}

		changeEvents = append(changeEvents, changeEvent{entity: entity, changeType: trackchanges.ChangeTypeDeleted})
		return nil
	}); err != nil {
		if isTransactionHandled(err) {
			return
		}
		h.writeDatabaseError(w, r, err)
		return
	}

	h.finalizeChangeEvents(ctx, changeEvents)

	w.WriteHeader(http.StatusNoContent)
}

// handlePatchEntity handles PATCH requests for individual entities
func (h *EntityHandler) handlePatchEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	ctx := r.Context()

	// Start tracing span for patch operation
	var span trace.Span
	if h.observability != nil {
		tracer := h.observability.Tracer()
		ctx, span = tracer.StartEntityPatch(ctx, h.metadata.EntitySetName, entityKey)
		defer span.End()
		r = r.WithContext(ctx)
	}

	// Check if there's an overwrite handler
	if h.overwrite.hasUpdate() {
		h.handleUpdateEntityOverwrite(w, r, entityKey, false)
		return
	}

	// Check if this is a virtual entity without overwrite handler
	if h.metadata.IsVirtual {
		if err := response.WriteError(w, r, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			"Virtual entities require an overwrite handler for Update operation"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	if !h.enforceUpdateRestrictions(w, r, "PATCH") {
		return
	}

	// Validate Content-Type header
	if err := validateContentType(w, r); err != nil {
		return
	}

	pref := preference.ParsePrefer(r)

	var (
		entity       interface{}
		changeEvents []changeEvent
	)

	if err := h.runInTransaction(ctx, r, func(tx *gorm.DB, hookReq *http.Request) error {
		entity = reflect.New(h.metadata.EntityType).Interface()

		db, err := h.buildKeyQuery(tx, entityKey)
		if err != nil {
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return newTransactionHandledError(err)
		}

		if err := db.First(entity).Error; err != nil {
			h.handleFetchError(w, r, err, entityKey)
			return newTransactionHandledError(err)
		}

		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptorWithEntity(h.metadata, entityKey, entity, nil), auth.OperationUpdate, h.logger) {
			return newTransactionHandledError(errAuthorizationDenied)
		}

		if h.metadata.ETagProperty != nil {
			ifMatch := r.Header.Get(HeaderIfMatch)
			currentETag := etag.Generate(entity, h.metadata)

			if !etag.Match(ifMatch, currentETag) {
				if writeErr := response.WriteError(w, r, http.StatusPreconditionFailed, ErrMsgPreconditionFailed,
					ErrDetailPreconditionFailed); writeErr != nil {
					h.logger.Error("Error writing error response", "error", writeErr)
				}
				return newTransactionHandledError(errETagMismatch)
			}
		}

		updateData, err := h.parsePatchRequestBody(r, w)
		if err != nil {
			return newTransactionHandledError(err)
		}

		if err := h.validateKeyPropertiesNotUpdated(updateData, w, r); err != nil {
			return newTransactionHandledError(err)
		}

		if err := h.validatePropertiesExistForUpdate(updateData, w, r); err != nil {
			return newTransactionHandledError(err)
		}

		pendingBindings, err := h.processODataBindAnnotationsForUpdate(ctx, entity, updateData, tx)
		if err != nil {
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid @odata.bind annotation", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return newTransactionHandledError(err)
		}

		h.removeODataBindAnnotations(updateData)

		if err := h.validateDataTypes(updateData); err != nil {
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid data type", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return newTransactionHandledError(err)
		}

		if err := h.validateRequiredFieldsNotNull(updateData); err != nil {
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid value for required property", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return newTransactionHandledError(err)
		}

		if err := h.callBeforeUpdate(entity, hookReq); err != nil {
			h.writeHookError(w, r, err, http.StatusForbidden, "Authorization failed")
			return newTransactionHandledError(err)
		}

		// Manually increment ETag property (Version field) for PATCH operations
		// GORM's BeforeUpdate hook is not triggered when using Updates() with a map
		if h.metadata.ETagProperty != nil {
			h.incrementETagProperty(entity)
			// Add the incremented version to updateData so it gets persisted
			entityVal := reflect.ValueOf(entity).Elem()
			etagField := entityVal.FieldByName(h.metadata.ETagProperty.FieldName)
			if etagField.IsValid() {
				// Use the column name from metadata for the database update
				updateData[h.metadata.ETagProperty.ColumnName] = etagField.Interface()
			}
		}

		if err := tx.Model(entity).Updates(updateData).Error; err != nil {
			h.writeDatabaseError(w, r, err)
			return newTransactionHandledError(err)
		}

		if err := h.applyPendingCollectionBindings(ctx, tx, entity, pendingBindings); err != nil {
			if writeErr := response.WriteError(w, r, http.StatusInternalServerError, "Failed to bind navigation properties", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return newTransactionHandledError(err)
		}

		if err := h.callAfterUpdate(entity, hookReq); err != nil {
			h.logger.Error("AfterUpdate hook failed", "error", err)
		}

		if err := tx.First(entity).Error; err != nil {
			h.logger.Error("Error refreshing entity for change tracking", "error", err)
		} else {
			changeEvents = append(changeEvents, changeEvent{entity: entity, changeType: trackchanges.ChangeTypeUpdated})
		}

		return nil
	}); err != nil {
		if isTransactionHandled(err) {
			return
		}
		h.writeDatabaseError(w, r, err)
		return
	}

	h.finalizeChangeEvents(ctx, changeEvents)

	db, err := h.buildKeyQuery(h.db.WithContext(ctx), entityKey)
	if err != nil {
		h.writeDatabaseError(w, r, err)
		return
	}

	h.writeUpdateResponse(w, r, pref, db)
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
		h.writeDatabaseError(w, r, err)
		return
	}

	h.writeEntityResponseWithETag(w, r, updatedEntity, "", http.StatusOK, nil)
}

// handlePutEntity handles PUT requests for individual entities
// PUT performs a complete replacement according to OData v4 spec
func (h *EntityHandler) handlePutEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	// Check if there's an overwrite handler
	if h.overwrite.hasUpdate() {
		h.handleUpdateEntityOverwrite(w, r, entityKey, true)
		return
	}

	// Check if this is a virtual entity without overwrite handler
	if h.metadata.IsVirtual {
		if err := response.WriteError(w, r, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			"Virtual entities require an overwrite handler for Update operation"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	if !h.enforceUpdateRestrictions(w, r, "PUT") {
		return
	}

	// Validate Content-Type header
	if err := validateContentType(w, r); err != nil {
		return
	}

	pref := preference.ParsePrefer(r)

	var changeEvents []changeEvent

	ctx := r.Context()
	if err := h.runInTransaction(ctx, r, func(tx *gorm.DB, hookReq *http.Request) error {
		entity := reflect.New(h.metadata.EntityType).Interface()

		db, err := h.buildKeyQuery(tx, entityKey)
		if err != nil {
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return newTransactionHandledError(err)
		}

		if err := db.First(entity).Error; err != nil {
			h.handleFetchError(w, r, err, entityKey)
			return newTransactionHandledError(err)
		}

		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptorWithEntity(h.metadata, entityKey, entity, nil), auth.OperationUpdate, h.logger) {
			return newTransactionHandledError(errAuthorizationDenied)
		}

		if h.metadata.ETagProperty != nil {
			ifMatch := r.Header.Get(HeaderIfMatch)
			currentETag := etag.Generate(entity, h.metadata)

			if !etag.Match(ifMatch, currentETag) {
				if writeErr := response.WriteError(w, r, http.StatusPreconditionFailed, ErrMsgPreconditionFailed,
					ErrDetailPreconditionFailed); writeErr != nil {
					h.logger.Error("Error writing error response", "error", writeErr)
				}
				return newTransactionHandledError(errETagMismatch)
			}
		}

		replacementEntity := reflect.New(h.metadata.EntityType).Interface()
		if err := json.NewDecoder(r.Body).Decode(replacementEntity); err != nil {
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, ErrMsgInvalidRequestBody,
				fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error())); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return newTransactionHandledError(err)
		}

		if err := h.preserveKeyProperties(entity, replacementEntity); err != nil {
			if writeErr := response.WriteError(w, r, http.StatusInternalServerError, ErrMsgInternalError, err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return newTransactionHandledError(err)
		}

		// Preserve server-managed timestamp fields (like CreatedAt) to avoid MySQL zero datetime issues
		h.preserveTimestampFields(entity, replacementEntity)

		if err := h.callBeforeUpdate(entity, hookReq); err != nil {
			h.writeHookError(w, r, err, http.StatusForbidden, "Authorization failed")
			return newTransactionHandledError(err)
		}

		// Manually increment ETag property (Version field) for PUT operations
		// GORM's BeforeUpdate hook is not triggered reliably with Updates()
		if h.metadata.ETagProperty != nil {
			h.incrementETagProperty(entity)
			// Copy the incremented version to the replacement entity
			h.copyETagProperty(entity, replacementEntity)
		}

		if err := tx.Model(entity).Select("*").Updates(replacementEntity).Error; err != nil {
			h.writeDatabaseError(w, r, err)
			return newTransactionHandledError(err)
		}

		if err := h.callAfterUpdate(entity, hookReq); err != nil {
			h.logger.Error("AfterUpdate hook failed", "error", err)
		}

		if err := tx.First(entity).Error; err != nil {
			h.logger.Error("Error refreshing entity for change tracking", "error", err)
		} else {
			changeEvents = append(changeEvents, changeEvent{entity: entity, changeType: trackchanges.ChangeTypeUpdated})
		}

		return nil
	}); err != nil {
		if isTransactionHandled(err) {
			return
		}
		h.writeDatabaseError(w, r, err)
		return
	}

	h.finalizeChangeEvents(ctx, changeEvents)

	db, err := h.buildKeyQuery(h.db.WithContext(ctx), entityKey)
	if err != nil {
		h.writeDatabaseError(w, r, err)
		return
	}

	h.writeUpdateResponse(w, r, pref, db)
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

// preserveTimestampFields copies time.Time and *time.Time fields from source to destination if destination has zero value
// This prevents MySQL/MariaDB errors with zero datetime values ('0000-00-00')
func (h *EntityHandler) preserveTimestampFields(source, destination interface{}) {
	sourceVal := reflect.ValueOf(source).Elem()
	destVal := reflect.ValueOf(destination).Elem()

	// Iterate over all fields in the destination struct
	for i := 0; i < destVal.NumField(); i++ {
		destField := destVal.Field(i)
		fieldType := destVal.Type().Field(i)

		// Handle time.Time fields
		if destField.Type() == reflect.TypeOf(time.Time{}) && destField.CanSet() {
			// Check if destination field is zero (safe type assertion)
			destTime, ok := destField.Interface().(time.Time)
			if ok && destTime.IsZero() {
				// Get corresponding source field
				sourceField := sourceVal.FieldByName(fieldType.Name)
				if sourceField.IsValid() && sourceField.Type() == reflect.TypeOf(time.Time{}) {
					// Copy the non-zero value from source (safe type assertion)
					sourceTime, ok := sourceField.Interface().(time.Time)
					if ok && !sourceTime.IsZero() {
						destField.Set(sourceField)
					}
				}
			}
		}

		// Handle *time.Time fields
		if destField.Type() == reflect.TypeOf((*time.Time)(nil)) && destField.CanSet() {
			// Check if destination field is nil or points to zero time
			shouldPreserve := false
			if destField.IsNil() {
				shouldPreserve = true
			} else {
				destTimePtr, ok := destField.Interface().(*time.Time)
				if ok && destTimePtr != nil && destTimePtr.IsZero() {
					shouldPreserve = true
				}
			}

			if shouldPreserve {
				// Get corresponding source field
				sourceField := sourceVal.FieldByName(fieldType.Name)
				if sourceField.IsValid() && sourceField.Type() == reflect.TypeOf((*time.Time)(nil)) {
					// Copy the non-nil, non-zero value from source
					if !sourceField.IsNil() {
						sourceTimePtr, ok := sourceField.Interface().(*time.Time)
						if ok && sourceTimePtr != nil && !sourceTimePtr.IsZero() {
							destField.Set(sourceField)
						}
					}
				}
			}
		}
	}
}

// parsePatchRequestBody parses the JSON request body for PATCH operations
func (h *EntityHandler) parsePatchRequestBody(r *http.Request, w http.ResponseWriter) (map[string]interface{}, error) {
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		if writeErr := response.WriteError(w, r, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error())); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return nil, err
	}
	return updateData, nil
}

// validateKeyPropertiesNotUpdated validates that key properties are not being updated
func (h *EntityHandler) validateKeyPropertiesNotUpdated(updateData map[string]interface{}, w http.ResponseWriter, r *http.Request) error {
	for _, keyProp := range h.metadata.KeyProperties {
		if _, exists := updateData[keyProp.JsonName]; exists {
			err := fmt.Errorf("key property '%s' cannot be modified", keyProp.JsonName)
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Cannot update key property", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return err
		}
		// Also check using the struct field name
		if _, exists := updateData[keyProp.Name]; exists {
			err := fmt.Errorf("key property '%s' cannot be modified", keyProp.Name)
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Cannot update key property", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return err
		}
	}
	return nil
}

// validatePropertiesExistForUpdate validates that all properties in updateData are valid entity properties
// This version allows @odata.bind annotations for navigation properties
func (h *EntityHandler) validatePropertiesExistForUpdate(updateData map[string]interface{}, w http.ResponseWriter, r *http.Request) error {
	// Use shared validation function with checkAutoProperties=true
	// (auto properties cannot be updated by clients)
	return h.validatePropertiesExist(updateData, w, r, true)
}

// removeODataBindAnnotations removes @odata.bind annotations and all other instance annotations
// from the update data as they are not actual entity properties and should not be passed to GORM
func (h *EntityHandler) removeODataBindAnnotations(updateData map[string]interface{}) {
	// Pre-allocate with small initial capacity; annotations are typically a small fraction of update data
	keysToRemove := make([]string, 0, 8)
	for key := range updateData {
		// Remove entity-level annotations (starting with @)
		if strings.HasPrefix(key, "@") {
			keysToRemove = append(keysToRemove, key)
			continue
		}
		// Remove property-level annotations (property@annotation format)
		// Check if @ appears and the part before @ is a valid property name
		// Use pre-built propertyMap for O(1) property name lookup instead of scanning all properties
		if idx := strings.Index(key, "@"); idx > 0 {
			propertyPart := key[:idx]
			if _, ok := h.propertyMap[propertyPart]; ok {
				keysToRemove = append(keysToRemove, key)
			}
		}
	}
	for _, key := range keysToRemove {
		delete(updateData, key)
	}
}

// incrementETagProperty increments the ETag property (Version field) on an entity
// This is needed because GORM's BeforeUpdate hook is not triggered when using Updates() with a map
func (h *EntityHandler) incrementETagProperty(entity interface{}) {
	if h.metadata.ETagProperty == nil {
		return
	}

	entityVal := reflect.ValueOf(entity).Elem()
	etagField := entityVal.FieldByName(h.metadata.ETagProperty.FieldName)

	if !etagField.IsValid() || !etagField.CanSet() {
		h.logger.Error("Cannot increment ETag property", "property", h.metadata.ETagProperty.FieldName)
		return
	}

	// Handle int-based version fields
	if etagField.Kind() == reflect.Int || etagField.Kind() == reflect.Int32 || etagField.Kind() == reflect.Int64 {
		currentValue := etagField.Int()
		etagField.SetInt(currentValue + 1)
	}
}

// copyETagProperty copies the ETag property value from source to destination
func (h *EntityHandler) copyETagProperty(source, destination interface{}) {
	if h.metadata.ETagProperty == nil {
		return
	}

	sourceVal := reflect.ValueOf(source).Elem()
	destVal := reflect.ValueOf(destination).Elem()

	sourceField := sourceVal.FieldByName(h.metadata.ETagProperty.FieldName)
	destField := destVal.FieldByName(h.metadata.ETagProperty.FieldName)

	if !sourceField.IsValid() || !destField.IsValid() || !destField.CanSet() {
		h.logger.Error("Cannot copy ETag property", "property", h.metadata.ETagProperty.FieldName)
		return
	}

	destField.Set(sourceField)
}

// handleDeleteEntityOverwrite handles DELETE entity requests using the overwrite handler
func (h *EntityHandler) handleDeleteEntityOverwrite(w http.ResponseWriter, r *http.Request, entityKey string) {
	// Create overwrite context with empty query options (DELETE doesn't use query options,
	// but we provide a consistent context interface across all operations)
	ctx := &OverwriteContext{
		QueryOptions: &query.QueryOptions{},
		EntityKey:    entityKey,
		Request:      r,
	}

	// Call the overwrite handler
	if err := h.overwrite.delete(ctx); err != nil {
		if IsNotFoundError(err) {
			WriteError(w, r, http.StatusNotFound, ErrMsgEntityNotFound,
				fmt.Sprintf("Entity with key '%s' not found", entityKey))
			return
		}
		WriteError(w, r, http.StatusInternalServerError, "Error deleting entity", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateEntityOverwrite handles PATCH/PUT entity requests using the overwrite handler
func (h *EntityHandler) handleUpdateEntityOverwrite(w http.ResponseWriter, r *http.Request, entityKey string, isFullReplace bool) {
	if err := validateContentType(w, r); err != nil {
		return
	}

	pref := preference.ParsePrefer(r)

	// Parse the request body
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		WriteError(w, r, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error()))
		return
	}

	// Create overwrite context
	ctx := &OverwriteContext{
		QueryOptions: &query.QueryOptions{},
		EntityKey:    entityKey,
		Request:      r,
	}

	// Call the overwrite handler
	result, err := h.overwrite.update(ctx, updateData, isFullReplace)
	if err != nil {
		if IsNotFoundError(err) {
			WriteError(w, r, http.StatusNotFound, ErrMsgEntityNotFound,
				fmt.Sprintf("Entity with key '%s' not found", entityKey))
			return
		}
		WriteError(w, r, http.StatusInternalServerError, "Error updating entity", err.Error())
		return
	}

	// Build response
	if applied := pref.GetPreferenceApplied(); applied != "" {
		w.Header().Set(HeaderPreferenceApplied, applied)
	}

	if pref.ShouldReturnContent(false) {
		if result != nil {
			h.writeEntityResponseWithETag(w, r, result, "", http.StatusOK, nil)
		} else {
			// If no result was returned but content was requested, return 204
			w.WriteHeader(http.StatusNoContent)
		}
	} else {
		// For 204 No Content, include OData-EntityId if we have the result
		if result != nil {
			entityId := h.buildEntityLocation(r, result)
			SetODataHeader(w, HeaderODataEntityId, entityId)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
