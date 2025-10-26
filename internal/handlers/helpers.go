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

// SetODataHeader sets an HTTP header preserving the exact capitalization.
// This is needed for OData-specific headers like "OData-Version" and "OData-EntityId"
// which do not follow Go's standard HTTP header canonicalization (which would turn them into "Odata-Version").
// The OData v4 specification requires these headers to have exact capitalization with a capital 'D'.
//
// This function sets the header using direct map assignment to preserve the exact key,
// ensuring the header is sent over the wire with the correct capitalization as required by OData spec.
//
// Note: Due to the non-canonical key, Header.Get() will not work for accessing these headers.
// Use direct map access (w.Header()["OData-Version"]) instead when you need to read them.
func SetODataHeader(w http.ResponseWriter, key, value string) {
	// Set with exact capitalization for wire format (OData spec compliance)
	w.Header()[key] = []string{value}
}

// SetODataVersionHeader sets the OData-Version header with the correct version value.
// This centralizes the version header setting to ensure consistency across all responses.
func SetODataVersionHeader(w http.ResponseWriter) {
	SetODataHeader(w, HeaderODataVersion, ODataVersionValue)
}

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

// buildEntityResponseWithMetadata builds an OData entity response with metadata level support
func (h *EntityHandler) buildEntityResponseWithMetadata(navValue reflect.Value, contextURL string, metadataLevel string) map[string]interface{} {
	odataResponse := response.NewOrderedMap()

	// Only include @odata.context for minimal and full metadata (not for none)
	if metadataLevel != "none" {
		odataResponse.Set(ODataContextProperty, contextURL)
	}

	// Add @odata.type for full metadata
	if metadataLevel == "full" {
		odataResponse.Set("@odata.type", "#ODataService."+h.metadata.EntityName)
	}

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

// buildOrderedEntityResponseWithMetadata builds an ordered OData entity response with metadata level support
func (h *EntityHandler) buildOrderedEntityResponseWithMetadata(result interface{}, contextURL string, metadataLevel string, r *http.Request, etagValue string) *response.OrderedMap {
	odataResponse := response.NewOrderedMap()

	// Only include @odata.context for minimal and full metadata (not for none)
	if metadataLevel != "none" {
		odataResponse.Set(ODataContextProperty, contextURL)
	}

	// Add @odata.etag annotation if ETag value is available
	// Per OData v4 spec, exclude all control information for metadata=none
	if etagValue != "" && metadataLevel != "none" {
		odataResponse.Set("@odata.etag", etagValue)
	}

	// Check if result is a map (from $select) or a struct
	resultValue := reflect.ValueOf(result)

	// Build the entity ID for @odata.id
	var entityID string
	switch resultValue.Kind() {
	case reflect.Ptr:
		entityID = h.buildEntityIDFromValue(resultValue.Elem(), r)
	case reflect.Struct:
		entityID = h.buildEntityIDFromValue(resultValue, r)
	case reflect.Map:
		if mapResult, ok := result.(map[string]interface{}); ok {
			entityID = h.buildEntityIDFromMap(mapResult, r)
		}
	}

	// Add @odata.id based on metadata level
	if entityID != "" {
		switch metadataLevel {
		case "full":
			// Always include @odata.id in full metadata
			odataResponse.Set("@odata.id", entityID)
		case "minimal":
			// For minimal metadata, check later after processing fields
			odataResponse.Set("__temp_entity_id", entityID)
		}
		// For "none" metadata level, never include @odata.id
	}

	// Add @odata.type for full metadata
	if metadataLevel == "full" {
		odataResponse.Set("@odata.type", "#ODataService."+h.metadata.EntityName)
	}

	// Handle map[string]interface{} (from $select filtering)
	if resultValue.Kind() == reflect.Map {
		// Iterate over map keys in a consistent order
		for _, key := range resultValue.MapKeys() {
			keyStr := key.String()
			value := resultValue.MapIndex(key)
			odataResponse.Set(keyStr, value.Interface())
		}

		// For minimal metadata with map, check if all key fields are present
		if metadataLevel == "minimal" {
			if tempID, exists := odataResponse.ToMap()["__temp_entity_id"]; exists {
				// Remove the temporary ID
				odataResponse.Delete("__temp_entity_id")

				// Per OData v4 spec: For individual entity responses (with /$entity in context URL),
				// @odata.id MUST be included even if all key fields are present
				// For collection items, @odata.id is only needed when key fields are omitted
				isIndividualEntity := strings.Contains(contextURL, "/$entity")

				if isIndividualEntity {
					// Always add @odata.id for individual entity responses
					odataResponse.InsertAfter(ODataContextProperty, "@odata.id", tempID)
				} else {
					// For collection items, check if all key fields are present
					allKeysPresent := true
					for _, keyProp := range h.metadata.KeyProperties {
						if _, exists := odataResponse.ToMap()[keyProp.JsonName]; !exists {
							allKeysPresent = false
							break
						}
					}

					if !allKeysPresent {
						// Add @odata.id after @odata.context
						odataResponse.InsertAfter(ODataContextProperty, "@odata.id", tempID)
					}
				}
			}
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
		if !field.IsExported() {
			continue
		}

		fieldValue := resultValue.Field(i)
		jsonName := getJsonName(field)

		// Check if this field is a navigation property
		propMeta := h.findPropertyMetadata(field.Name)
		if propMeta != nil && propMeta.IsNavigationProp {
			// Check if the navigation property is populated (expanded)
			isExpanded := false
			if fieldValue.Kind() == reflect.Ptr {
				// For pointer types, check if not nil
				isExpanded = !fieldValue.IsNil()
			} else if fieldValue.Kind() == reflect.Slice {
				// For slices, check if not nil and not empty
				isExpanded = !fieldValue.IsNil() && fieldValue.Len() > 0
			}

			if isExpanded {
				// Include the expanded data
				odataResponse.Set(jsonName, fieldValue.Interface())
			} else {
				// Handle navigation property based on metadata level
				// Only add navigation link for full metadata (per OData v4 spec)
				// Minimal and none metadata levels do not include navigation links
				if metadataLevel == "full" && r != nil {
					// Build the key segment for the navigation link
					keySegment := h.buildKeySegmentFromEntity(resultValue)
					if keySegment != "" {
						baseURL := response.BuildBaseURL(r)
						navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, h.metadata.EntitySetName, keySegment, propMeta.JsonName)
						odataResponse.Set(jsonName+"@odata.navigationLink", navLink)
					}
				}
			}
		} else {
			// Regular property - include its value
			odataResponse.Set(jsonName, fieldValue.Interface())
		}
	}

	// For minimal metadata, check if all key fields are present in struct
	if metadataLevel == "minimal" {
		if tempID, exists := odataResponse.ToMap()["__temp_entity_id"]; exists {
			// Remove the temporary ID
			odataResponse.Delete("__temp_entity_id")

			// Per OData v4 spec: For individual entity responses (with /$entity in context URL),
			// @odata.id MUST be included even if all key fields are present
			// For collection items, @odata.id is only needed when key fields are omitted
			isIndividualEntity := strings.Contains(contextURL, "/$entity")
			if isIndividualEntity {
				// Always add @odata.id for individual entity responses
				odataResponse.InsertAfter(ODataContextProperty, "@odata.id", tempID)
			}
			// For collection items (not individual entities), @odata.id is only added when key fields are omitted
			// This is already handled in the map case above (lines 230-248)
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

// findPropertyMetadata finds metadata for a property by field name
func (h *EntityHandler) findPropertyMetadata(fieldName string) *metadata.PropertyMetadata {
	for i := range h.metadata.Properties {
		if h.metadata.Properties[i].Name == fieldName || h.metadata.Properties[i].FieldName == fieldName {
			return &h.metadata.Properties[i]
		}
	}
	return nil
}

// buildKeySegmentFromEntity builds the key segment for URLs from an entity value
func (h *EntityHandler) buildKeySegmentFromEntity(entity reflect.Value) string {
	keyProps := h.metadata.KeyProperties
	if len(keyProps) == 0 {
		return ""
	}

	// Single key - return just the value
	if len(keyProps) == 1 {
		keyFieldValue := entity.FieldByName(keyProps[0].Name)
		if keyFieldValue.IsValid() {
			return fmt.Sprintf("%v", keyFieldValue.Interface())
		}
		return ""
	}

	// Composite keys - return key1=value1,key2=value2
	var parts []string
	for _, keyProp := range keyProps {
		keyFieldValue := entity.FieldByName(keyProp.Name)
		if keyFieldValue.IsValid() {
			keyValue := keyFieldValue.Interface()
			// Quote string values
			if keyFieldValue.Kind() == reflect.String {
				parts = append(parts, fmt.Sprintf("%s='%v'", keyProp.JsonName, keyValue))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%v", keyProp.JsonName, keyValue))
			}
		}
	}

	return strings.Join(parts, ",")
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

func (a *metadataAdapter) GetETagProperty() *response.PropertyMetadata {
	if a.metadata.ETagProperty == nil {
		return nil
	}
	return &response.PropertyMetadata{
		Name:              a.metadata.ETagProperty.Name,
		JsonName:          a.metadata.ETagProperty.JsonName,
		IsNavigationProp:  a.metadata.ETagProperty.IsNavigationProp,
		NavigationTarget:  a.metadata.ETagProperty.NavigationTarget,
		NavigationIsArray: a.metadata.ETagProperty.NavigationIsArray,
	}
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

// buildEntityIDFromValue builds the @odata.id value from an entity value
func (h *EntityHandler) buildEntityIDFromValue(entityValue reflect.Value, r *http.Request) string {
	if r == nil {
		return ""
	}
	baseURL := response.BuildBaseURL(r)
	keySegment := h.buildKeySegmentFromEntity(entityValue)
	if keySegment == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s(%s)", baseURL, h.metadata.EntitySetName, keySegment)
}

// buildEntityIDFromMap builds the @odata.id value from a map entity
func (h *EntityHandler) buildEntityIDFromMap(entityMap map[string]interface{}, r *http.Request) string {
	if r == nil {
		return ""
	}
	baseURL := response.BuildBaseURL(r)

	// Build key segment from map
	keyProps := h.metadata.KeyProperties
	if len(keyProps) == 0 {
		return ""
	}

	// Single key
	if len(keyProps) == 1 {
		if keyValue, exists := entityMap[keyProps[0].JsonName]; exists && keyValue != nil {
			return fmt.Sprintf("%s/%s(%v)", baseURL, h.metadata.EntitySetName, keyValue)
		}
		return ""
	}

	// Composite keys
	var parts []string
	for _, keyProp := range keyProps {
		if keyValue, exists := entityMap[keyProp.JsonName]; exists && keyValue != nil {
			// Quote string values
			if strVal, ok := keyValue.(string); ok {
				parts = append(parts, fmt.Sprintf("%s='%s'", keyProp.JsonName, strVal))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%v", keyProp.JsonName, keyValue))
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("%s/%s(%s)", baseURL, h.metadata.EntitySetName, strings.Join(parts, ","))
}
