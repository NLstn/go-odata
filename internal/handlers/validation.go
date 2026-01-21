package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/response"
	"go.opentelemetry.io/otel/trace"
)

// validateDataTypes validates that all values in updateData match the expected types
// from the entity metadata. Returns an error describing the first type mismatch found.
func (h *EntityHandler) validateDataTypes(updateData map[string]interface{}) error {
	for propName, propValue := range updateData {
		// Skip nil values - they are allowed for nullable properties
		if propValue == nil {
			continue
		}

		// Find the property metadata
		propMeta := h.metadata.FindProperty(propName)

		// Property validation already handled by validatePropertiesExist
		if propMeta == nil {
			continue
		}

		// Validate the type
		if err := validateValueType(propValue, propMeta.Type, propMeta.JsonName); err != nil {
			return err
		}
	}

	return nil
}

// validateValueType checks if a value matches the expected reflect.Type
func validateValueType(value interface{}, expectedType reflect.Type, fieldName string) error {
	// Get the actual value type
	actualValue := reflect.ValueOf(value)
	actualType := actualValue.Type()

	// Handle pointer types in expected type
	if expectedType.Kind() == reflect.Ptr {
		expectedType = expectedType.Elem()
	}

	// Special handling for numeric types from JSON unmarshaling
	// JSON numbers are unmarshaled as float64
	if actualType.Kind() == reflect.Float64 {
		switch expectedType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			// Numeric types are compatible
			return nil
		}
	}

	// Special handling for time.Time struct type
	// JSON datetime values come as strings but need to be assigned to time.Time fields
	if expectedType.Kind() == reflect.Struct && expectedType.PkgPath() == "time" && expectedType.Name() == "Time" {
		if actualType.Kind() == reflect.String {
			return nil // Strings are valid for time.Time fields (will be parsed by JSON unmarshaler)
		}
		// Reject non-string values for time.Time fields
		return fmt.Errorf("property '%s' expects type Edm.DateTimeOffset but got %s", fieldName, actualType.Kind())
	}

	// Handle string type
	if actualType.Kind() == reflect.String && expectedType.Kind() != reflect.String {
		return fmt.Errorf("property '%s' expects type %s but got string", fieldName, expectedType.Kind())
	}

	// Handle bool type
	if actualType.Kind() == reflect.Bool && expectedType.Kind() != reflect.Bool {
		return fmt.Errorf("property '%s' expects type %s but got bool", fieldName, expectedType.Kind())
	}

	// Handle slice types
	if actualType.Kind() == reflect.Slice && expectedType.Kind() == reflect.Slice {
		// Allow slice type matching
		return nil
	}

	// Handle map types (for complex types)
	if actualType.Kind() == reflect.Map && expectedType.Kind() == reflect.Struct {
		// Allow map to struct conversion (for nested objects)
		return nil
	}

	// If types don't match and we haven't handled the case above, it's an error
	if !actualType.AssignableTo(expectedType) && actualType.Kind() != reflect.Float64 {
		return fmt.Errorf("property '%s' expects type %s but got %s", fieldName, expectedType.Kind(), actualType.Kind())
	}

	return nil
}

// validateRequiredFieldsNotNull validates that required fields are not being set to null
func (h *EntityHandler) validateRequiredFieldsNotNull(updateData map[string]interface{}) error {
	var nullRequiredFields []string

	for propName, propValue := range updateData {
		// Check if value is explicitly null
		if propValue != nil {
			continue
		}

		// Find the property metadata
		propMeta := h.metadata.FindProperty(propName)

		// If property is required and value is null, add to error list
		if propMeta != nil && propMeta.IsRequired {
			nullRequiredFields = append(nullRequiredFields, propMeta.JsonName)
		}
	}

	if len(nullRequiredFields) > 0 {
		return fmt.Errorf("required properties cannot be set to null: %s", strings.Join(nullRequiredFields, ", "))
	}

	return nil
}

// validateContentType checks that the Content-Type header is present and valid for write operations
func validateContentType(w http.ResponseWriter, r *http.Request) error {
	contentType := r.Header.Get("Content-Type")

	// Check if Content-Type is missing
	if contentType == "" {
		if writeErr := response.WriteError(w, r, http.StatusUnsupportedMediaType, "Unsupported Media Type",
			"Content-Type header is required for this operation"); writeErr != nil {
			slog.Default().Error("Error writing error response", "error", writeErr)
		}
		return fmt.Errorf("missing Content-Type header")
	}

	// Extract the media type (ignore parameters like charset)
	mediaType := contentType
	if idx := strings.Index(contentType, ";"); idx != -1 {
		mediaType = strings.TrimSpace(contentType[:idx])
	}

	// Check if it's a valid JSON content type
	validTypes := []string{
		"application/json",
		"application/json;odata.metadata=minimal",
		"application/json;odata.metadata=full",
		"application/json;odata.metadata=none",
	}

	isValid := false
	for _, validType := range validTypes {
		if strings.HasPrefix(strings.ToLower(contentType), strings.ToLower(validType)) {
			isValid = true
			break
		}
	}

	// Also accept if just the media type matches
	if !isValid && strings.ToLower(mediaType) == "application/json" {
		isValid = true
	}

	if !isValid {
		if writeErr := response.WriteError(w, r, http.StatusUnsupportedMediaType, "Unsupported Media Type",
			fmt.Sprintf("Content-Type '%s' is not supported. Only application/json is supported for data modifications.", contentType)); writeErr != nil {
			slog.Default().Error("Error writing error response", "error", writeErr)
		}
		return fmt.Errorf("unsupported Content-Type: %s", contentType)
	}

	return nil
}

// validatePropertiesExist validates that all properties in the provided data map are valid entity properties.
// It allows instance annotations (starting with @) and property-level annotations (property@annotation format),
// but rejects unknown property names. When checkAutoProperties is true, it also rejects auto properties.
func (h *EntityHandler) validatePropertiesExist(data map[string]interface{}, w http.ResponseWriter, r *http.Request, checkAutoProperties bool) error {
	// Build a map of valid property names (both JSON names and struct field names)
	validProperties := make(map[string]bool, len(h.metadata.Properties)*2)
	var autoProperties map[string]bool
	if checkAutoProperties {
		autoProperties = make(map[string]bool)
	}

	for _, prop := range h.metadata.Properties {
		validProperties[prop.JsonName] = true
		validProperties[prop.Name] = true

		// Track auto properties to reject client updates if needed
		if checkAutoProperties && prop.IsAuto {
			autoProperties[prop.JsonName] = true
			autoProperties[prop.Name] = true
		}
	}

	// Check each property in data
	for propName := range data {
		// Allow any property starting with @ (instance annotations at entity level)
		// Per OData spec, clients can send instance annotations which should be ignored
		if strings.HasPrefix(propName, "@") {
			continue
		}

		// Allow property-level annotations (property@annotation format)
		// Check if this is a property annotation by looking for @ after the property name
		if idx := strings.Index(propName, "@"); idx > 0 {
			// Extract the property name part before the @
			propertyPart := propName[:idx]
			// Check if the property exists using the precomputed property map for O(1) lookup
			if _, ok := h.propertyMap[propertyPart]; ok {
				continue
			}
			// Property annotation refers to a non-existent property
			err := fmt.Errorf("annotation '%s' refers to non-existent property '%s' on entity type '%s'", propName, propertyPart, h.metadata.EntityName)
			span := trace.SpanFromContext(r.Context())
			span.RecordError(err)
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid annotation", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return err
		}

		if !validProperties[propName] {
			err := fmt.Errorf("property '%s' does not exist on entity type '%s'", propName, h.metadata.EntityName)
			span := trace.SpanFromContext(r.Context())
			span.RecordError(err)
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid property", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return err
		}

		// Reject attempts to update auto properties if checkAutoProperties is true
		if checkAutoProperties && autoProperties[propName] {
			err := fmt.Errorf("property '%s' is automatically set server-side and cannot be modified by clients", propName)
			span := trace.SpanFromContext(r.Context())
			span.RecordError(err)
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Invalid property modification", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return err
		}
	}

	return nil
}

// validateImmutablePropertiesNotUpdated validates that immutable properties are not being updated
func (h *EntityHandler) validateImmutablePropertiesNotUpdated(updateData map[string]interface{}, w http.ResponseWriter, r *http.Request) error {
	for propName := range updateData {
		// Find the property metadata
		propMeta := h.metadata.FindProperty(propName)
		if propMeta == nil {
			continue
		}

		// Check if the property has the Core.Immutable annotation
		if propMeta.Annotations != nil && propMeta.Annotations.Has("Org.OData.Core.V1.Immutable") {
			err := fmt.Errorf("property '%s' is immutable and cannot be modified", propMeta.JsonName)
			if writeErr := response.WriteError(w, r, http.StatusBadRequest, "Cannot update immutable property", err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return err
		}
	}
	return nil
}
