package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
)

func (h *EntityHandler) handlePostEntity(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if h.metadata.HasStream && !strings.Contains(contentType, "application/json") {
		h.handlePostMediaEntity(w, r)
		return
	}

	if err := validateContentType(w, r); err != nil {
		return
	}

	pref := preference.ParsePrefer(r)

	var requestData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error()))
		return
	}

	entity := reflect.New(h.metadata.EntityType).Interface()

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to process request data", err.Error())
		return
	}
	if err := json.Unmarshal(jsonData, entity); err != nil {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error()))
		return
	}

	var changeEvents []changeEvent
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		pendingBindings, err := h.processODataBindAnnotations(entity, requestData, tx)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid @odata.bind annotation", err.Error())
			return newTransactionHandledError(err)
		}

		h.clearAutoIncrementKeys(entity)

		if err := h.validateRequiredProperties(requestData); err != nil {
			WriteError(w, http.StatusBadRequest, "Missing required properties", err.Error())
			return newTransactionHandledError(err)
		}

		if err := h.validateRequiredFieldsNotNull(requestData); err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid null value", err.Error())
			return newTransactionHandledError(err)
		}

		if err := h.callBeforeCreate(entity, r); err != nil {
			WriteError(w, http.StatusForbidden, "Authorization failed", err.Error())
			return newTransactionHandledError(err)
		}

		if err := tx.Create(entity).Error; err != nil {
			WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error())
			return newTransactionHandledError(err)
		}

		// Apply pending collection-valued navigation property bindings after entity is saved
		if err := h.applyPendingCollectionBindings(tx, entity, pendingBindings); err != nil {
			WriteError(w, http.StatusInternalServerError, "Failed to bind navigation properties", err.Error())
			return newTransactionHandledError(err)
		}

		if err := h.callAfterCreate(entity, r); err != nil {
			h.logger.Error("AfterCreate hook failed", "error", err)
		}

		changeEvents = append(changeEvents, changeEvent{entity: entity, changeType: trackchanges.ChangeTypeAdded})
		return nil
	}); err != nil {
		if isTransactionHandled(err) {
			return
		}
		h.writeDatabaseError(w, err)
		return
	}

	for _, event := range changeEvents {
		h.recordChange(event.entity, event.changeType)
	}

	location := h.buildEntityLocation(r, entity)
	w.Header().Set("Location", location)

	if applied := pref.GetPreferenceApplied(); applied != "" {
		w.Header().Set(HeaderPreferenceApplied, applied)
	}

	if pref.ShouldReturnContent(true) {
		SetODataHeader(w, HeaderODataEntityId, location)
		h.writeEntityResponseWithETag(w, r, entity, "", http.StatusCreated)
	} else {
		SetODataHeader(w, HeaderODataEntityId, location)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *EntityHandler) handlePostMediaEntity(w http.ResponseWriter, r *http.Request) {
	content := make([]byte, 0)
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		if n > 0 {
			content = append(content, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	entity := reflect.New(h.metadata.EntityType).Interface()

	entityValue := reflect.ValueOf(entity)

	if method := entityValue.MethodByName("SetMediaContent"); method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(content)})
	}

	if method := entityValue.MethodByName("SetMediaContentType"); method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(contentType)})
	}

	var changeEvents []changeEvent
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := h.callBeforeCreate(entity, r); err != nil {
			if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return newTransactionHandledError(err)
		}

		if err := tx.Create(entity).Error; err != nil {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return newTransactionHandledError(err)
		}

		if err := h.callAfterCreate(entity, r); err != nil {
			h.logger.Error("AfterCreate hook failed", "error", err)
		}

		changeEvents = append(changeEvents, changeEvent{entity: entity, changeType: trackchanges.ChangeTypeAdded})
		return nil
	}); err != nil {
		if isTransactionHandled(err) {
			return
		}
		h.writeDatabaseError(w, err)
		return
	}

	for _, event := range changeEvents {
		h.recordChange(event.entity, event.changeType)
	}

	location := h.buildEntityLocation(r, entity)
	w.Header().Set("Location", location)
	SetODataHeader(w, HeaderODataEntityId, location)
	w.WriteHeader(http.StatusCreated)
}

func (h *EntityHandler) validateRequiredProperties(requestData map[string]interface{}) error {
	var missingFields []string
	for _, prop := range h.metadata.Properties {
		if !prop.IsRequired || prop.IsKey {
			continue
		}

		// Check if the field exists in the JSON request data
		// This distinguishes between "field not provided" vs "field provided with zero value"
		if _, exists := requestData[prop.JsonName]; !exists {
			missingFields = append(missingFields, prop.JsonName)
		}
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required properties: %s", strings.Join(missingFields, ", "))
	}

	return nil
}

func (h *EntityHandler) clearAutoIncrementKeys(entity interface{}) {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	if len(h.metadata.KeyProperties) > 1 {
		return
	}

	for _, keyProp := range h.metadata.KeyProperties {
		if strings.Contains(keyProp.GormTag, "autoIncrement:false") {
			continue
		}

		field := entityValue.FieldByName(keyProp.Name)
		if !field.IsValid() || !field.CanSet() {
			continue
		}

		switch field.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			field.SetZero()
		}
	}
}

func (h *EntityHandler) buildEntityLocation(r *http.Request, entity interface{}) string {
	baseURL := response.BuildBaseURL(r)
	entitySetName := h.metadata.EntitySetName

	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	if len(h.metadata.KeyProperties) == 1 {
		keyProp := h.metadata.KeyProperties[0]
		keyValue := entityValue.FieldByName(keyProp.Name)
		return fmt.Sprintf("%s/%s(%v)", baseURL, entitySetName, keyValue.Interface())
	}

	var keyParts []string
	for _, keyProp := range h.metadata.KeyProperties {
		keyValue := entityValue.FieldByName(keyProp.Name)
		switch keyValue.Kind() {
		case reflect.String:
			keyParts = append(keyParts, fmt.Sprintf("%s='%v'", keyProp.JsonName, keyValue.Interface()))
		default:
			keyParts = append(keyParts, fmt.Sprintf("%s=%v", keyProp.JsonName, keyValue.Interface()))
		}
	}

	return fmt.Sprintf("%s/%s(%s)", baseURL, entitySetName, strings.Join(keyParts, ","))
}
