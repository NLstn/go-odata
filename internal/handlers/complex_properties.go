package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// HandleComplexTypeProperty handles GET, HEAD, and OPTIONS requests for complex type properties (e.g., Products(1)/ShippingAddress)
// propertySegments represents the navigation path segments without $value/$ref/$count (e.g., ["ShippingAddress", "City"]).
func (h *EntityHandler) HandleComplexTypeProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertySegments []string, isValue bool) {
	if len(propertySegments) == 0 {
		h.writePropertyNotFoundError(w, "")
		return
	}

	propertyPath := append([]string(nil), propertySegments...)
	if isValue {
		propertyPath = append(propertyPath, "$value")
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, entityKey, propertyPath), auth.OperationRead, h.logger) {
			return
		}
		h.handleGetComplexTypeProperty(w, r, entityKey, propertySegments, isValue)
	case http.MethodOptions:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, entityKey, propertyPath), auth.OperationRead, h.logger) {
			return
		}
		h.handleOptionsComplexTypeProperty(w)
	default:
		h.writeMethodNotAllowedError(w, r.Method, "complex property access")
	}
}

// handleGetComplexTypeProperty resolves and writes a complex type property response
func (h *EntityHandler) handleGetComplexTypeProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertySegments []string, isValue bool) {
	rootName := propertySegments[0]
	complexProp := h.findComplexTypeProperty(rootName)
	if complexProp == nil {
		h.writePropertyNotFoundError(w, rootName)
		return
	}

	if len(propertySegments) == 1 && isValue {
		if err := response.WriteError(w, r, http.StatusBadRequest, "Invalid request",
			"$value is not supported on complex properties"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	fieldValue, err := h.fetchComplexPropertyValue(w, r, entityKey, complexProp)
	if err != nil {
		return
	}

	// Handle root null complex property
	if isNilPointer(fieldValue) {
		h.writeNoContentForNullComplex(w, r)
		return
	}

	// Prepare traversal state
	contextSegments := []string{complexProp.JsonName}
	currentValue := dereferenceValue(fieldValue)
	currentType := dereferenceType(complexProp.Type)

	// Traverse nested segments, if any
	for idx, segment := range propertySegments[1:] {
		resolved, resolvedType, resolvedJSONName, ok := resolveStructField(currentValue, segment)
		if !ok {
			// Support maps as well as structs
			if currentValue.Kind() == reflect.Map {
				resolved, resolvedType, resolvedJSONName, ok = resolveMapField(currentValue, segment)
			}
		}

		if !ok {
			h.writePropertyNotFoundError(w, segment)
			return
		}

		contextSegments = append(contextSegments, resolvedJSONName)
		currentValue = resolved
		currentType = resolvedType

		// If there are more segments to traverse, ensure we can continue
		if idx < len(propertySegments[1:])-1 {
			if isNilPointer(currentValue) {
				h.writeComplexSegmentNullError(w, r, contextSegments)
				return
			}
			currentValue = dereferenceValue(currentValue)
			currentType = dereferenceType(currentType)
		}
	}

	h.writeResolvedComplexValue(w, r, entityKey, contextSegments, currentValue, currentType, isValue)
}

// handleOptionsComplexTypeProperty handles OPTIONS requests for complex properties
func (h *EntityHandler) handleOptionsComplexTypeProperty(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

// fetchComplexPropertyValue fetches a complex property value from an entity without applying select clauses
func (h *EntityHandler) fetchComplexPropertyValue(w http.ResponseWriter, r *http.Request, entityKey string, prop *metadata.PropertyMetadata) (reflect.Value, error) {
	entity := reflect.New(h.metadata.EntityType).Interface()

	var db *gorm.DB
	var err error

	// Handle singleton case where entityKey is empty
	if h.metadata.IsSingleton && entityKey == "" {
		// For singletons, we don't use a key query, just fetch the first (and only) record
		db = h.db
	} else {
		// For regular entities, build the key query
		db, err = h.buildKeyQuery(h.db, entityKey)
		if err != nil {
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return reflect.Value{}, err
		}
	}

	if err := db.First(entity).Error; err != nil {
		h.handlePropertyFetchError(w, err, entityKey)
		return reflect.Value{}, err
	}

	entityValue := reflect.ValueOf(entity).Elem()
	fieldValue := entityValue.FieldByName(prop.Name)
	if !fieldValue.IsValid() {
		if writeErr := response.WriteError(w, r, http.StatusInternalServerError, ErrMsgInternalError,
			"Could not access complex property"); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return reflect.Value{}, fmt.Errorf("invalid field")
	}

	return fieldValue, nil
}

// writeNoContentForNullComplex writes a 204 No Content response for null complex properties
func (h *EntityHandler) writeNoContentForNullComplex(w http.ResponseWriter, r *http.Request) {
	metadataLevel := response.GetODataMetadataLevel(r)
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusNoContent)
}

// writeComplexSegmentNullError writes an error when a nested complex segment is null
func (h *EntityHandler) writeComplexSegmentNullError(w http.ResponseWriter, r *http.Request, contextSegments []string) {
	path := strings.Join(contextSegments, "/")
	if err := response.WriteError(w, r, http.StatusNotFound, "Property not found",
		fmt.Sprintf("Complex property path '%s' is null", path)); err != nil {
		h.logger.Error("Error writing error response", "error", err)
	}
}

// writeResolvedComplexValue writes the resolved complex property (or nested primitive) response
func (h *EntityHandler) writeResolvedComplexValue(w http.ResponseWriter, r *http.Request, entityKey string, contextSegments []string, value reflect.Value, valueType reflect.Type, isValue bool) {
	if isComplexValue(value, valueType) {
		if isValue {
			if err := response.WriteError(w, r, http.StatusBadRequest, "Invalid request",
				"$value is not supported on complex properties"); err != nil {
				h.logger.Error("Error writing error response", "error", err)
			}
			return
		}

		if isNilPointer(value) {
			h.writeNoContentForNullComplex(w, r)
			return
		}

		resolvedValue := dereferenceValue(value)
		h.writeComplexValueResponse(w, r, entityKey, contextSegments, resolvedValue)
		return
	}

	// Primitive value resolution
	resolvedValue := dereferenceValue(value)

	if isValue {
		h.writeRawPrimitiveValue(w, r, resolvedValue)
		return
	}

	h.writePrimitiveComplexPropertyResponse(w, r, entityKey, contextSegments, value)
}

// writeComplexValueResponse serializes a complex value (struct or map) with @odata.context
func (h *EntityHandler) writeComplexValueResponse(w http.ResponseWriter, r *http.Request, entityKey string, contextSegments []string, value reflect.Value) {
	metadataLevel := response.GetODataMetadataLevel(r)
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	contextPath := strings.Join(contextSegments, "/")
	responseMap := make(map[string]interface{})
	if metadataLevel != "none" {
		contextURL := fmt.Sprintf("%s/$metadata#%s(%s)/%s", response.BuildBaseURL(r), h.metadata.EntitySetName, entityKey, contextPath)
		responseMap[ODataContextProperty] = contextURL
	}

	switch value.Kind() {
	case reflect.Struct:
		structMap := structValueToMap(value)
		for k, v := range structMap {
			responseMap[k] = v
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			key := iter.Key()
			if key.Kind() == reflect.String {
				responseMap[key.String()] = iter.Value().Interface()
			}
		}
	default:
		responseMap["value"] = value.Interface()
	}

	if err := json.NewEncoder(w).Encode(responseMap); err != nil {
		h.logger.Error("Error writing complex property response", "error", err)
	}
}

// writePrimitiveComplexPropertyResponse writes a primitive property (nested within a complex type)
func (h *EntityHandler) writePrimitiveComplexPropertyResponse(w http.ResponseWriter, r *http.Request, entityKey string, contextSegments []string, value reflect.Value) {
	metadataLevel := response.GetODataMetadataLevel(r)
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	var valueInterface interface{}
	if !value.IsValid() {
		valueInterface = nil
	} else if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			valueInterface = nil
		} else {
			valueInterface = value.Elem().Interface()
		}
	} else {
		valueInterface = value.Interface()
	}

	responseBody := map[string]interface{}{
		"value": valueInterface,
	}

	if metadataLevel != "none" {
		contextURL := fmt.Sprintf("%s/$metadata#%s(%s)/%s", response.BuildBaseURL(r), h.metadata.EntitySetName, entityKey, strings.Join(contextSegments, "/"))
		responseBody[ODataContextProperty] = contextURL
	}

	if err := json.NewEncoder(w).Encode(responseBody); err != nil {
		h.logger.Error("Error writing complex primitive property response", "error", err)
	}
}

// writeRawPrimitiveValue writes a primitive value in raw form for $value requests
func (h *EntityHandler) writeRawPrimitiveValue(w http.ResponseWriter, r *http.Request, value reflect.Value) {
	if !value.IsValid() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch value.Kind() {
	case reflect.String:
		w.Header().Set(HeaderContentType, ContentTypePlainText)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		w.Header().Set(HeaderContentType, ContentTypePlainText)
	default:
		w.Header().Set(HeaderContentType, "application/octet-stream")
	}

	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	if _, err := fmt.Fprintf(w, "%v", value.Interface()); err != nil {
		h.logger.Error("Error writing raw primitive value", "error", err)
	}
}

// resolveStructField resolves a field from a struct by segment name, returning the reflect.Value, type, and canonical JSON name
func resolveStructField(value reflect.Value, segment string) (reflect.Value, reflect.Type, string, bool) {
	if value.Kind() != reflect.Struct {
		return reflect.Value{}, nil, "", false
	}

	valueType := value.Type()
	lowerSegment := strings.ToLower(segment)
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonName := getJSONFieldName(field)
		candidates := []string{field.Name, jsonName}
		for _, candidate := range candidates {
			if candidate == "" || candidate == "-" {
				continue
			}
			if candidate == segment || strings.ToLower(candidate) == lowerSegment {
				return value.Field(i), field.Type, jsonName, true
			}
		}
	}

	return reflect.Value{}, nil, "", false
}

// resolveMapField resolves a value from a map with string keys
func resolveMapField(value reflect.Value, segment string) (reflect.Value, reflect.Type, string, bool) {
	if value.Kind() != reflect.Map || value.Type().Key().Kind() != reflect.String {
		return reflect.Value{}, nil, "", false
	}

	mapValue := value.MapIndex(reflect.ValueOf(segment))
	if mapValue.IsValid() {
		return mapValue, mapValue.Type(), segment, true
	}

	lowerSegment := strings.ToLower(segment)
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key()
		if strings.ToLower(key.String()) == lowerSegment {
			val := iter.Value()
			return val, val.Type(), key.String(), true
		}
	}

	return reflect.Value{}, nil, "", false
}

// isNilPointer checks if a value is a nil pointer or interface
func isNilPointer(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	switch value.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// dereferenceValue dereferences pointer values until a non-pointer is reached
func dereferenceValue(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

// dereferenceType dereferences pointer types until a non-pointer is reached
func dereferenceType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// isComplexValue determines if a resolved value should be treated as a complex object
func isComplexValue(value reflect.Value, valueType reflect.Type) bool {
	if !value.IsValid() {
		if valueType == nil {
			return false
		}
		value = reflect.New(dereferenceType(valueType)).Elem()
	}

	if value.Kind() == reflect.Map {
		return true
	}

	resolvedType := valueType
	if resolvedType == nil {
		resolvedType = value.Type()
	}

	resolvedType = dereferenceType(resolvedType)
	if resolvedType == nil {
		return false
	}

	if resolvedType.Kind() != reflect.Struct {
		return false
	}

	// Treat time.Time (and similar) as primitive despite being structs
	if resolvedType.PkgPath() == "time" && resolvedType.Name() == "Time" {
		return false
	}

	return true
}

// structValueToMap converts a struct to a map using JSON tag names
func structValueToMap(value reflect.Value) map[string]interface{} {
	result := make(map[string]interface{})
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonName := getJSONFieldName(field)
		if jsonName == "-" {
			continue
		}
		if jsonName == "" {
			jsonName = field.Name
		}

		result[jsonName] = value.Field(i).Interface()
	}
	return result
}

// getJSONFieldName extracts the JSON field name from a struct field
func getJSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}

	parts := strings.Split(tag, ",")
	if len(parts) == 0 || parts[0] == "" {
		return field.Name
	}

	return parts[0]
}
