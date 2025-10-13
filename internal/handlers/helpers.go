package handlers

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// buildKeyQuery builds a GORM query with WHERE conditions for the entity key(s)
// Supports both single keys and composite keys
func (h *EntityHandler) buildKeyQuery(entityKey string) (*gorm.DB, error) {
	db := h.db

	// Parse the key - could be single value or composite key format
	components := &response.ODataURLComponents{
		EntityKeyMap: make(map[string]string),
	}

	// Try to parse as composite key format first
	keyPart := entityKey
	if err := parseCompositeKey(keyPart, components); err != nil {
		// If parsing fails, treat as simple single key
		components.EntityKey = entityKey
		components.EntityKeyMap = nil
	}

	// If we have a composite key map, use it
	if len(components.EntityKeyMap) > 0 {
		// Build WHERE clause for each key property
		for _, keyProp := range h.metadata.KeyProperties {
			keyValue, found := components.EntityKeyMap[keyProp.JsonName]
			if !found {
				// Also try the field name
				keyValue, found = components.EntityKeyMap[keyProp.Name]
			}
			if !found {
				return nil, fmt.Errorf("missing key property: %s", keyProp.JsonName)
			}
			// Use snake_case for database column name (GORM convention)
			columnName := toSnakeCase(keyProp.JsonName)
			db = db.Where(fmt.Sprintf("%s = ?", columnName), keyValue)
		}
	} else {
		// Single key - use backwards compatible logic
		if len(h.metadata.KeyProperties) != 1 {
			return nil, fmt.Errorf("entity has composite keys, please use composite key format: key1=value1,key2=value2")
		}
		// Use snake_case for database column name (GORM convention)
		columnName := toSnakeCase(h.metadata.KeyProperties[0].JsonName)
		db = db.Where(fmt.Sprintf("%s = ?", columnName), entityKey)
	}

	return db, nil
}

// toSnakeCase converts a camelCase or PascalCase string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if the previous character was lowercase or if this is the start of a new word
			// For "ProductID", we want "product_id" not "product_i_d"
			prevRune := rune(s[i-1])
			if prevRune >= 'a' && prevRune <= 'z' {
				result.WriteRune('_')
			} else if i < len(s)-1 {
				// Check if next character is lowercase (e.g., "XMLParser" -> "xml_parser")
				nextRune := rune(s[i+1])
				if nextRune >= 'a' && nextRune <= 'z' {
					result.WriteRune('_')
				}
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// parseCompositeKey attempts to parse a composite key string
func parseCompositeKey(keyPart string, components *response.ODataURLComponents) error {
	// Check if it contains '=' - if not, it's a simple single key value
	if !strings.Contains(keyPart, "=") {
		return fmt.Errorf("not a composite key")
	}

	// Parse composite key format: key1=value1,key2=value2
	pairs := strings.Split(keyPart, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key-value pair: %s", pair)
		}

		keyName := strings.TrimSpace(parts[0])
		keyValue := strings.TrimSpace(parts[1])

		// Remove quotes from value if present
		if len(keyValue) > 0 && (keyValue[0] == '\'' || keyValue[0] == '"') {
			keyValue = strings.Trim(keyValue, "'\"")
		}

		components.EntityKeyMap[keyName] = keyValue
	}

	return nil
}

// handleFetchError writes appropriate error responses based on the fetch error type
func (h *EntityHandler) handleFetchError(w http.ResponseWriter, err error, entityKey string) {
	if err == gorm.ErrRecordNotFound {
		target := fmt.Sprintf(ODataEntityKeyFormat, h.metadata.EntitySetName, entityKey)
		if writeErr := response.WriteErrorWithTarget(w, http.StatusNotFound, ErrMsgEntityNotFound,
			target, fmt.Sprintf(EntityKeyNotExistFmt, entityKey)); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
	} else {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
	}
}

// buildEntityResponse builds an OData entity response from a reflect.Value
func (h *EntityHandler) buildEntityResponse(navValue reflect.Value, contextURL string) map[string]interface{} {
	odataResponse := response.NewOrderedMap()
	odataResponse.Set(ODataContextProperty, contextURL)

	navType := navValue.Type()
	for i := 0; i < navValue.NumField(); i++ {
		field := navType.Field(i)
		if field.IsExported() {
			fieldValue := navValue.Field(i)
			jsonName := getJsonName(field)
			odataResponse.Set(jsonName, fieldValue.Interface())
		}
	}

	return odataResponse.ToMap()
}

// buildOrderedEntityResponse builds an ordered OData entity response
func (h *EntityHandler) buildOrderedEntityResponse(result interface{}, contextURL string) *response.OrderedMap {
	odataResponse := response.NewOrderedMap()
	odataResponse.Set(ODataContextProperty, contextURL)

	// Check if result is a map (from $select) or a struct
	resultValue := reflect.ValueOf(result)

	// Handle map[string]interface{} (from $select filtering)
	if resultValue.Kind() == reflect.Map {
		// Iterate over map keys in a consistent order
		for _, key := range resultValue.MapKeys() {
			keyStr := key.String()
			value := resultValue.MapIndex(key)
			odataResponse.Set(keyStr, value.Interface())
		}
		return odataResponse
	}

	// Handle struct/pointer to struct (original entity)
	if resultValue.Kind() == reflect.Ptr {
		resultValue = resultValue.Elem()
	}

	entityType := resultValue.Type()
	for i := 0; i < resultValue.NumField(); i++ {
		field := entityType.Field(i)
		if field.IsExported() {
			fieldValue := resultValue.Field(i)
			jsonName := getJsonName(field)
			odataResponse.Set(jsonName, fieldValue.Interface())
		}
	}

	return odataResponse
}

// getJsonName extracts the JSON field name from struct tags
func getJsonName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return field.Name
	}

	// Handle json:",omitempty" or json:"fieldname,omitempty"
	parts := strings.Split(jsonTag, ",")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return field.Name
}

// metadataAdapter adapts metadata.EntityMetadata to response.EntityMetadataProvider
type metadataAdapter struct {
	metadata *metadata.EntityMetadata
}

func (a *metadataAdapter) GetProperties() []response.PropertyMetadata {
	props := make([]response.PropertyMetadata, len(a.metadata.Properties))
	for i, p := range a.metadata.Properties {
		props[i] = response.PropertyMetadata{
			Name:              p.Name,
			JsonName:          p.JsonName,
			IsNavigationProp:  p.IsNavigationProp,
			NavigationTarget:  p.NavigationTarget,
			NavigationIsArray: p.NavigationIsArray,
		}
	}
	return props
}

func (a *metadataAdapter) GetKeyProperty() *response.PropertyMetadata {
	if a.metadata.KeyProperty == nil {
		return nil
	}
	return &response.PropertyMetadata{
		Name:              a.metadata.KeyProperty.Name,
		JsonName:          a.metadata.KeyProperty.JsonName,
		IsNavigationProp:  a.metadata.KeyProperty.IsNavigationProp,
		NavigationTarget:  a.metadata.KeyProperty.NavigationTarget,
		NavigationIsArray: a.metadata.KeyProperty.NavigationIsArray,
	}
}

func (a *metadataAdapter) GetKeyProperties() []response.PropertyMetadata {
	props := make([]response.PropertyMetadata, len(a.metadata.KeyProperties))
	for i, p := range a.metadata.KeyProperties {
		props[i] = response.PropertyMetadata{
			Name:              p.Name,
			JsonName:          p.JsonName,
			IsNavigationProp:  p.IsNavigationProp,
			NavigationTarget:  p.NavigationTarget,
			NavigationIsArray: p.NavigationIsArray,
		}
	}
	return props
}

func (a *metadataAdapter) GetEntitySetName() string {
	return a.metadata.EntitySetName
}

// ValidateODataVersion checks if the OData-MaxVersion header is compatible with OData v4.0
// Returns true if the version is acceptable, false if it should be rejected
func ValidateODataVersion(r *http.Request) bool {
	maxVersion := r.Header.Get(HeaderODataMaxVersion)

	// If no OData-MaxVersion header is present, accept the request
	if maxVersion == "" {
		return true
	}

	// Parse the version string (e.g., "4.0", "3.0", "4.01")
	// We need to check if it's below 4.0
	majorVersion, _ := parseVersion(maxVersion)

	// Accept if major version is >= 4
	if majorVersion >= 4 {
		return true
	}

	// Reject if major version is < 4
	return false
}

// parseVersion parses a version string like "4.0" or "3.0" into major and minor components
func parseVersion(version string) (int, int) {
	version = strings.TrimSpace(version)
	parts := strings.Split(version, ".")

	if len(parts) == 0 {
		return 0, 0
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0
	}

	minor := 0
	if len(parts) > 1 {
		// Ignore error - if minor version can't be parsed, default to 0
		minor, _ = strconv.Atoi(parts[1]) //nolint:errcheck
	}

	return major, minor
}
