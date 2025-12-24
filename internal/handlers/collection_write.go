package handlers

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

func (h *EntityHandler) handlePostEntity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start tracing span for create operation
	var span trace.Span
	if h.observability != nil {
		tracer := h.observability.Tracer()
		ctx, span = tracer.StartEntityCreate(ctx, h.metadata.EntitySetName)
		defer span.End()
		r = r.WithContext(ctx)
	}

	// Check if there's an overwrite handler
	if h.overwrite.hasCreate() {
		h.handlePostEntityOverwrite(w, r)
		return
	}

	// Check if this is a virtual entity without overwrite handler
	if h.metadata.IsVirtual {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			"Virtual entities require an overwrite handler for Create operation"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, "", nil), auth.OperationCreate, h.logger) {
		return
	}

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
	if err := h.runInTransaction(ctx, r, func(tx *gorm.DB, hookReq *http.Request) error {
		pendingBindings, err := h.processODataBindAnnotations(ctx, entity, requestData, tx)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid @odata.bind annotation", err.Error())
			return newTransactionHandledError(err)
		}

		if err := h.initializeEntityKeys(ctx, entity); err != nil {
			WriteError(w, http.StatusInternalServerError, ErrMsgInternalError, err.Error())
			return newTransactionHandledError(err)
		}

		if err := h.validateAutoPropertiesNotProvided(requestData); err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid property", err.Error())
			return newTransactionHandledError(err)
		}

		if err := h.validateRequiredProperties(requestData); err != nil {
			WriteError(w, http.StatusBadRequest, "Missing required properties", err.Error())
			return newTransactionHandledError(err)
		}

		if err := h.validateRequiredFieldsNotNull(requestData); err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid null value", err.Error())
			return newTransactionHandledError(err)
		}

		if err := h.callBeforeCreate(entity, hookReq); err != nil {
			WriteError(w, http.StatusForbidden, "Authorization failed", err.Error())
			return newTransactionHandledError(err)
		}

		if err := tx.Create(entity).Error; err != nil {
			WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error())
			return newTransactionHandledError(err)
		}

		// Apply pending collection-valued navigation property bindings after entity is saved
		if err := h.applyPendingCollectionBindings(ctx, tx, entity, pendingBindings); err != nil {
			WriteError(w, http.StatusInternalServerError, "Failed to bind navigation properties", err.Error())
			return newTransactionHandledError(err)
		}

		if err := h.callAfterCreate(entity, hookReq); err != nil {
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

	h.finalizeChangeEvents(ctx, changeEvents)

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
	ctx := r.Context()
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

	if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, "", nil), auth.OperationCreate, h.logger) {
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	entity := reflect.New(h.metadata.EntityType).Interface()

	// Initialize entity keys (e.g., generate UUID for key properties)
	if err := h.initializeEntityKeys(ctx, entity); err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError, err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	entityValue := reflect.ValueOf(entity)

	if method := entityValue.MethodByName("SetMediaContent"); method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(content)})
	}

	if method := entityValue.MethodByName("SetMediaContentType"); method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(contentType)})
	}

	var changeEvents []changeEvent
	if err := h.runInTransaction(ctx, r, func(tx *gorm.DB, hookReq *http.Request) error {
		if err := h.callBeforeCreate(entity, hookReq); err != nil {
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

		if err := h.callAfterCreate(entity, hookReq); err != nil {
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

	h.finalizeChangeEvents(ctx, changeEvents)

	location := h.buildEntityLocation(r, entity)
	w.Header().Set("Location", location)
	SetODataHeader(w, HeaderODataEntityId, location)
	w.WriteHeader(http.StatusCreated)
}

func (h *EntityHandler) validateRequiredProperties(requestData map[string]interface{}) error {
	var missingFields []string
	for _, prop := range h.metadata.Properties {
		// Skip keys and auto fields (server-side fields)
		if !prop.IsRequired || prop.IsKey || prop.IsAuto {
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

func (h *EntityHandler) validateAutoPropertiesNotProvided(requestData map[string]interface{}) error {
	var autoFields []string
	for _, prop := range h.metadata.Properties {
		// Skip non-auto fields
		if !prop.IsAuto {
			continue
		}

		// Check if the auto field exists in the JSON request data
		if _, exists := requestData[prop.JsonName]; exists {
			autoFields = append(autoFields, prop.JsonName)
		}
	}

	if len(autoFields) > 0 {
		return fmt.Errorf("properties marked as 'auto' cannot be provided by clients and are set server-side: %s", strings.Join(autoFields, ", "))
	}

	return nil
}

func (h *EntityHandler) initializeEntityKeys(ctx context.Context, entity interface{}) error {
	if h.metadata == nil {
		return fmt.Errorf("entity metadata is not initialized")
	}

	value := reflect.ValueOf(entity)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	for _, keyProp := range h.metadata.KeyProperties {
		field := value.FieldByName(keyProp.Name)
		if !field.IsValid() {
			continue
		}

		if keyProp.DatabaseGenerated && field.CanSet() {
			zeroField(field)
		}

		if keyProp.KeyGenerator == "" {
			continue
		}

		if !field.CanSet() {
			return fmt.Errorf("cannot set generated key field %s", keyProp.Name)
		}

		generator, err := h.resolveKeyGenerator(keyProp.KeyGenerator)
		if err != nil {
			return err
		}

		generated, err := generator(ctx)
		if err != nil {
			return fmt.Errorf("key generator '%s' failed: %w", keyProp.KeyGenerator, err)
		}

		if err := assignGeneratedValue(field, generated); err != nil {
			return err
		}
	}

	return nil
}

func (h *EntityHandler) resolveKeyGenerator(name string) (func(context.Context) (interface{}, error), error) {
	if h.keyGeneratorResolver == nil {
		return nil, fmt.Errorf("no key generator resolver configured for '%s'", name)
	}

	generator, ok := h.keyGeneratorResolver(name)
	if !ok || generator == nil {
		return nil, fmt.Errorf("key generator '%s' is not registered", name)
	}

	return generator, nil
}

func zeroField(field reflect.Value) {
	if field.CanSet() {
		field.SetZero()
	}
}

func assignGeneratedValue(field reflect.Value, value interface{}) error {
	if !field.CanSet() {
		return fmt.Errorf("cannot set generated value for field of type %s", field.Type())
	}

	if value == nil {
		switch field.Kind() {
		case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice:
			field.SetZero()
			return nil
		default:
			return fmt.Errorf("key generator produced nil for non-nullable field of type %s", field.Type())
		}
	}

	val := reflect.ValueOf(value)
	if !val.IsValid() {
		return fmt.Errorf("key generator produced invalid value")
	}

	targetType := field.Type()

	if val.Type().AssignableTo(targetType) {
		field.Set(val)
		return nil
	}

	if val.Type().ConvertibleTo(targetType) {
		field.Set(val.Convert(targetType))
		return nil
	}

	if uuidBytes, ok := value.([16]byte); ok {
		return assignUUIDBytes(field, uuidBytes)
	}

	if byteSlice, ok := value.([]byte); ok && len(byteSlice) == 16 {
		var uuidBytes [16]byte
		copy(uuidBytes[:], byteSlice)
		return assignUUIDBytes(field, uuidBytes)
	}

	if targetType.Kind() == reflect.Ptr {
		elemType := targetType.Elem()
		if val.Type().AssignableTo(elemType) {
			ptr := reflect.New(elemType)
			ptr.Elem().Set(val)
			field.Set(ptr)
			return nil
		}
		if val.Type().ConvertibleTo(elemType) {
			ptr := reflect.New(elemType)
			ptr.Elem().Set(val.Convert(elemType))
			field.Set(ptr)
			return nil
		}
	}

	if targetType.Kind() == reflect.Interface && targetType.NumMethod() == 0 {
		field.Set(val)
		return nil
	}

	if targetType.Kind() == reflect.String {
		switch v := value.(type) {
		case fmt.Stringer:
			field.SetString(v.String())
			return nil
		case []byte:
			field.SetString(string(v))
			return nil
		}
	}

	return fmt.Errorf("generated key type %s cannot be assigned to field type %s", val.Type(), targetType)
}

func assignUUIDBytes(field reflect.Value, uuidBytes [16]byte) error {
	targetType := field.Type()
	uuidVal := reflect.ValueOf(uuidBytes)

	if uuidVal.Type().AssignableTo(targetType) {
		field.Set(uuidVal)
		return nil
	}

	if uuidVal.Type().ConvertibleTo(targetType) {
		field.Set(uuidVal.Convert(targetType))
		return nil
	}

	switch targetType.Kind() {
	case reflect.Ptr:
		elem := targetType.Elem()
		ptr := reflect.New(elem)
		if err := assignUUIDBytes(ptr.Elem(), uuidBytes); err != nil {
			return err
		}
		field.Set(ptr)
		return nil
	case reflect.Interface:
		field.Set(uuidVal)
		return nil
	case reflect.String:
		field.SetString(formatUUIDFromBytes(uuidBytes))
		return nil
	case reflect.Slice:
		if targetType.Elem().Kind() == reflect.Uint8 {
			bytes := make([]byte, len(uuidBytes))
			copy(bytes, uuidBytes[:])
			field.SetBytes(bytes)
			return nil
		}
	}

	return fmt.Errorf("generated key type %s cannot be assigned to field type %s", uuidVal.Type(), targetType)
}

func formatUUIDFromBytes(b [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%04x%08x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		binary.BigEndian.Uint16(b[10:12]),
		binary.BigEndian.Uint32(b[12:16]))
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

// handlePostEntityOverwrite handles POST entity requests using the overwrite handler
func (h *EntityHandler) handlePostEntityOverwrite(w http.ResponseWriter, r *http.Request) {
	if err := validateContentType(w, r); err != nil {
		return
	}

	pref := preference.ParsePrefer(r)

	// Parse the request body
	var requestData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error()))
		return
	}

	// Convert to entity for type-safe handling
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

	// Create overwrite context with empty query options (CREATE doesn't use query options,
	// but we provide a consistent context interface across all operations)
	ctx := &OverwriteContext{
		QueryOptions: &query.QueryOptions{},
		Request:      r,
	}

	// Call the overwrite handler
	result, err := h.overwrite.create(ctx, entity)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Error creating entity", err.Error())
		return
	}

	if result == nil {
		WriteError(w, http.StatusInternalServerError, "Error creating entity", "handler returned nil entity")
		return
	}

	// Build response
	location := h.buildEntityLocation(r, result)
	w.Header().Set("Location", location)

	if applied := pref.GetPreferenceApplied(); applied != "" {
		w.Header().Set(HeaderPreferenceApplied, applied)
	}

	if pref.ShouldReturnContent(true) {
		SetODataHeader(w, HeaderODataEntityId, location)
		h.writeEntityResponseWithETag(w, r, result, "", http.StatusCreated)
	} else {
		SetODataHeader(w, HeaderODataEntityId, location)
		w.WriteHeader(http.StatusNoContent)
	}
}
