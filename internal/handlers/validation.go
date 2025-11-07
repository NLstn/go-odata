package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/response"
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
		if writeErr := response.WriteError(w, http.StatusUnsupportedMediaType, "Unsupported Media Type",
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
		if writeErr := response.WriteError(w, http.StatusUnsupportedMediaType, "Unsupported Media Type",
			fmt.Sprintf("Content-Type '%s' is not supported. Only application/json is supported for data modifications.", contentType)); writeErr != nil {
			slog.Default().Error("Error writing error response", "error", writeErr)
		}
		return fmt.Errorf("unsupported Content-Type: %s", contentType)
	}

	return nil
}
