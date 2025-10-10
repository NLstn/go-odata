package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

// OrderedMap maintains insertion order of keys
type OrderedMap struct {
	keys   []string
	values map[string]interface{}
}

// NewOrderedMap creates a new OrderedMap
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		keys:   make([]string, 0),
		values: make(map[string]interface{}),
	}
}

// Set adds or updates a key-value pair in the ordered map
func (om *OrderedMap) Set(key string, value interface{}) {
	if _, exists := om.values[key]; !exists {
		om.keys = append(om.keys, key)
	}
	om.values[key] = value
}

// ToMap returns the underlying map (loses ordering)
func (om *OrderedMap) ToMap() map[string]interface{} {
	return om.values
}

// MarshalJSON implements json.Marshaler to maintain field order
func (om *OrderedMap) MarshalJSON() ([]byte, error) {
	buf := []byte("{")
	for i, key := range om.keys {
		if i > 0 {
			buf = append(buf, ',')
		}
		// Marshal the key
		keyBytes, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf = append(buf, keyBytes...)
		buf = append(buf, ':')

		// Marshal the value
		valueBytes, err := json.Marshal(om.values[key])
		if err != nil {
			return nil, err
		}
		buf = append(buf, valueBytes...)
	}
	buf = append(buf, '}')
	return buf, nil
}

// ODataResponse represents the structure of an OData JSON response
type ODataResponse struct {
	Context  string      `json:"@odata.context"`
	Count    *int64      `json:"@odata.count,omitempty"`
	NextLink *string     `json:"@odata.nextLink,omitempty"`
	Value    interface{} `json:"value"`
}

// EntityMetadataProvider is an interface for getting entity metadata
type EntityMetadataProvider interface {
	GetProperties() []PropertyMetadata
	GetKeyProperty() *PropertyMetadata    // Deprecated: Use GetKeyProperties for composite key support
	GetKeyProperties() []PropertyMetadata // Returns all key properties (single or composite)
	GetEntitySetName() string
}

// PropertyMetadata represents metadata about a property
type PropertyMetadata struct {
	Name              string
	JsonName          string
	IsNavigationProp  bool
	NavigationTarget  string
	NavigationIsArray bool
}

// WriteODataCollection writes an OData collection response
func WriteODataCollection(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string) error {
	// Build the context URL
	contextURL := buildContextURL(r, entitySetName)

	response := ODataResponse{
		Context:  contextURL,
		Count:    count,
		NextLink: nextLink,
		Value:    data,
	}

	// Set OData-compliant headers
	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	// Encode and write the response
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(response)
}

// WriteODataCollectionWithNavigation writes an OData collection response with navigation links
func WriteODataCollectionWithNavigation(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string, metadata EntityMetadataProvider, expandedProps []string) error {
	// Build the context URL
	contextURL := buildContextURL(r, entitySetName)

	// Transform the data to add navigation links
	transformedData := addNavigationLinks(data, metadata, expandedProps, r, entitySetName)

	response := ODataResponse{
		Context:  contextURL,
		Count:    count,
		NextLink: nextLink,
		Value:    transformedData,
	}

	// Set OData-compliant headers
	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	// Encode and write the response
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(response)
}

// addNavigationLinks adds @odata.navigationLink annotations for navigation properties
func addNavigationLinks(data interface{}, metadata EntityMetadataProvider, expandedProps []string, r *http.Request, entitySetName string) []interface{} {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		return nil
	}

	result := make([]interface{}, dataValue.Len())
	baseURL := buildBaseURL(r)
	keyProp := metadata.GetKeyProperty()

	for i := 0; i < dataValue.Len(); i++ {
		entity := dataValue.Index(i)
		var entityMap interface{}

		if entity.Kind() == reflect.Map {
			entityMap = processMapEntity(entity, metadata, expandedProps, baseURL, entitySetName, keyProp)
		} else {
			entityMap = processStructEntityOrdered(entity, metadata, expandedProps, baseURL, entitySetName, keyProp)
		}

		if entityMap != nil {
			result[i] = entityMap
		}
	}

	return result
}

// processMapEntity processes an entity that is already a map and adds navigation links
func processMapEntity(entity reflect.Value, metadata EntityMetadataProvider, expandedProps []string, baseURL, entitySetName string, keyProp *PropertyMetadata) map[string]interface{} {
	entityMap, ok := entity.Interface().(map[string]interface{})
	if !ok {
		return nil
	}

	// Add navigation links for properties that are not in the map and not expanded
	for _, prop := range metadata.GetProperties() {
		if !prop.IsNavigationProp {
			continue
		}

		if isPropertyExpanded(prop, expandedProps) {
			continue
		}

		// If property doesn't exist in map, add navigation link
		if _, exists := entityMap[prop.JsonName]; !exists {
			keySegment := buildKeySegmentFromMap(entityMap, metadata)
			if keySegment != "" {
				navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, prop.JsonName)
				entityMap[prop.JsonName+"@odata.navigationLink"] = navLink
			}
		}
	}

	return entityMap
}

// processStructEntityOrdered processes an entity that is a struct and returns an OrderedMap
func processStructEntityOrdered(entity reflect.Value, metadata EntityMetadataProvider, expandedProps []string, baseURL, entitySetName string, keyProp *PropertyMetadata) *OrderedMap {
	entityMap := NewOrderedMap()
	entityType := entity.Type()

	for j := 0; j < entity.NumField(); j++ {
		field := entityType.Field(j)
		if !field.IsExported() {
			continue
		}

		fieldValue := entity.Field(j)
		jsonName := getJsonFieldName(field)
		propMeta := findPropertyMetadata(field.Name, metadata)

		if propMeta != nil && propMeta.IsNavigationProp {
			processNavigationPropertyOrderedWithMetadata(entityMap, entity, propMeta, fieldValue, jsonName, expandedProps, baseURL, entitySetName, metadata)
		} else {
			// Regular property - include its value
			entityMap.Set(jsonName, fieldValue.Interface())
		}
	}

	return entityMap
}

// isPropertyExpanded checks if a property is in the expanded properties list
func isPropertyExpanded(prop PropertyMetadata, expandedProps []string) bool {
	for _, expanded := range expandedProps {
		if expanded == prop.Name || expanded == prop.JsonName {
			return true
		}
	}
	return false
}

// findPropertyMetadata finds the metadata for a property by its field name
func findPropertyMetadata(fieldName string, metadata EntityMetadataProvider) *PropertyMetadata {
	for _, prop := range metadata.GetProperties() {
		if prop.Name == fieldName {
			return &prop
		}
	}
	return nil
}

// processNavigationPropertyOrdered handles navigation properties for ordered maps
func processNavigationPropertyOrdered(entityMap *OrderedMap, entity reflect.Value, propMeta *PropertyMetadata, fieldValue reflect.Value, jsonName string, expandedProps []string, baseURL, entitySetName string, keyProp *PropertyMetadata) {
	if isPropertyExpanded(*propMeta, expandedProps) {
		// Include the expanded data
		entityMap.Set(jsonName, fieldValue.Interface())
	} else {
		// Add navigation link instead of null value
		keySegment := buildKeySegmentFromEntity(entity, keyProp)
		if keySegment != "" {
			navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, propMeta.JsonName)
			entityMap.Set(jsonName+"@odata.navigationLink", navLink)
		}
	}
}

// processNavigationPropertyOrderedWithMetadata handles navigation properties with full metadata support
func processNavigationPropertyOrderedWithMetadata(entityMap *OrderedMap, entity reflect.Value, propMeta *PropertyMetadata, fieldValue reflect.Value, jsonName string, expandedProps []string, baseURL, entitySetName string, metadata EntityMetadataProvider) {
	if isPropertyExpanded(*propMeta, expandedProps) {
		// Include the expanded data
		entityMap.Set(jsonName, fieldValue.Interface())
	} else {
		// Add navigation link instead of null value
		keySegment := BuildKeySegmentFromEntity(entity, metadata)
		if keySegment != "" {
			navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, propMeta.JsonName)
			entityMap.Set(jsonName+"@odata.navigationLink", navLink)
		}
	}
}

// buildKeySegmentFromEntity builds the key segment for URLs from an entity reflection value
// For single keys: returns "1" or "ID=1"
// For composite keys: returns "ProductID=1,LanguageKey='EN'"
func buildKeySegmentFromEntity(entity reflect.Value, keyProp *PropertyMetadata) string {
	// This function is kept for backwards compatibility but won't work properly for composite keys
	// since it only receives a single keyProp
	if keyProp == nil {
		return ""
	}

	keyFieldValue := entity.FieldByName(keyProp.Name)
	if keyFieldValue.IsValid() {
		return fmt.Sprintf("%v", keyFieldValue.Interface())
	}

	return ""
}

// BuildKeySegmentFromEntity builds the key segment for URLs from an entity and metadata
// For single keys: returns "1"
// For composite keys: returns "ProductID=1,LanguageKey='EN'"
func BuildKeySegmentFromEntity(entity reflect.Value, metadata EntityMetadataProvider) string {
	keyProps := metadata.GetKeyProperties()
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

// buildKeySegmentFromMap builds the key segment for URLs from a map
// For single keys: returns "1"
// For composite keys: returns "ProductID=1,LanguageKey='EN'"
func buildKeySegmentFromMap(entityMap map[string]interface{}, metadata EntityMetadataProvider) string {
	keyProps := metadata.GetKeyProperties()
	if len(keyProps) == 0 {
		return ""
	}

	// Single key - return just the value
	if len(keyProps) == 1 {
		if keyValue := entityMap[keyProps[0].JsonName]; keyValue != nil {
			return fmt.Sprintf("%v", keyValue)
		}
		return ""
	}

	// Composite keys - return key1=value1,key2=value2
	var parts []string
	for _, keyProp := range keyProps {
		if keyValue := entityMap[keyProp.JsonName]; keyValue != nil {
			// Quote string values
			if strVal, ok := keyValue.(string); ok {
				parts = append(parts, fmt.Sprintf("%s='%s'", keyProp.JsonName, strVal))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%v", keyProp.JsonName, keyValue))
			}
		}
	}

	return strings.Join(parts, ",")
}

// getJsonFieldName extracts the JSON field name from struct tags
func getJsonFieldName(field reflect.StructField) string {
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

// WriteError writes an OData error response
func WriteError(w http.ResponseWriter, code int, message string, details string) error {
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"details": details,
		},
	}

	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(code)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(errorResponse)
}

// WriteServiceDocument writes the OData service document
func WriteServiceDocument(w http.ResponseWriter, r *http.Request, entitySets []string) error {
	baseURL := buildBaseURL(r)

	entities := make([]map[string]interface{}, 0, len(entitySets))
	for _, entitySet := range entitySets {
		entities = append(entities, map[string]interface{}{
			"name": entitySet,
			"kind": "EntitySet",
			"url":  entitySet,
		})
	}

	serviceDoc := map[string]interface{}{
		"@odata.context": baseURL + "/$metadata",
		"value":          entities,
	}

	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(serviceDoc)
}

// buildContextURL builds the @odata.context URL for a response
func buildContextURL(r *http.Request, entitySetName string) string {
	baseURL := buildBaseURL(r)
	return baseURL + "/$metadata#" + entitySetName
}

// BuildBaseURL builds the base URL for the service (exported for use in handlers)
func BuildBaseURL(r *http.Request) string {
	return buildBaseURL(r)
}

// buildBaseURL builds the base URL for the service
func buildBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	// Handle X-Forwarded-Proto header for reverse proxies
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}

	return scheme + "://" + host
}

// BuildNextLink builds the next link URL for pagination
func BuildNextLink(r *http.Request, skipValue int) string {
	baseURL := buildBaseURL(r)

	// Clone the URL to avoid modifying the original
	nextURL := *r.URL

	// Get existing query parameters
	query := nextURL.Query()

	// Update the $skip parameter
	query.Set("$skip", fmt.Sprintf("%d", skipValue))

	// Rebuild the URL with updated query
	nextURL.RawQuery = query.Encode()

	return baseURL + nextURL.Path + "?" + nextURL.RawQuery
}

// ODataURLComponents represents the parsed components of an OData URL
type ODataURLComponents struct {
	EntitySet          string
	EntityKey          string            // For single keys: the value, for composite keys: empty (use EntityKeyMap)
	EntityKeyMap       map[string]string // For composite keys: map of key names to values
	NavigationProperty string            // For paths like Products(1)/Descriptions
}

// ParseODataURL parses an OData URL and extracts components (exported for use in main package)
func ParseODataURL(path string) (entitySet string, entityKey string, err error) {
	components, err := ParseODataURLComponents(path)
	if err != nil {
		return "", "", err
	}
	return components.EntitySet, components.EntityKey, err
}

// ParseODataURLComponents parses an OData URL and returns detailed components
func ParseODataURLComponents(path string) (*ODataURLComponents, error) {
	// Remove leading slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Parse URL
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	components := &ODataURLComponents{
		EntityKeyMap: make(map[string]string),
	}

	// Extract entity set and key
	pathParts := strings.Split(u.Path, "/")
	if len(pathParts) > 0 {
		entitySet := pathParts[0]

		// Check for key in parentheses: Products(1) or ProductDescriptions(ProductID=1,LanguageKey='EN')
		if idx := strings.Index(entitySet, "("); idx != -1 {
			if strings.HasSuffix(entitySet, ")") {
				keyPart := entitySet[idx+1 : len(entitySet)-1]
				components.EntitySet = entitySet[:idx]

				// Parse the key part - could be single key or composite
				if err := parseKeyPart(keyPart, components); err != nil {
					return nil, fmt.Errorf("invalid key format: %w", err)
				}
			}
		} else {
			components.EntitySet = entitySet
		}

		// Check for navigation property: Products(1)/Descriptions
		if len(pathParts) > 1 {
			components.NavigationProperty = pathParts[1]
		}
	}

	return components, nil
}

// parseKeyPart parses the key portion of an OData URL
// Supports both single keys: (1) or (ID=1)
// and composite keys: (ProductID=1,LanguageKey='EN')
func parseKeyPart(keyPart string, components *ODataURLComponents) error {
	// Check if it contains '=' - if not, it's a simple single key value
	if !strings.Contains(keyPart, "=") {
		components.EntityKey = keyPart
		return nil
	}

	// Parse composite key format: key1=value1,key2=value2
	// Split by comma, but be careful of quoted values
	pairs, err := splitKeyPairs(keyPart)
	if err != nil {
		return err
	}

	for _, pair := range pairs {
		// Split by '='
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key-value pair: %s", pair)
		}

		keyName := strings.TrimSpace(parts[0])
		keyValue := strings.TrimSpace(parts[1])

		// Remove quotes from value if present (OData allows 'value' or just value)
		if len(keyValue) > 0 && (keyValue[0] == '\'' || keyValue[0] == '"') {
			keyValue = strings.Trim(keyValue, "'\"")
		}

		components.EntityKeyMap[keyName] = keyValue
	}

	// If only one key-value pair, also set EntityKey for backwards compatibility
	if len(components.EntityKeyMap) == 1 {
		for _, v := range components.EntityKeyMap {
			components.EntityKey = v
			break
		}
	}

	return nil
}

// splitKeyPairs splits key pairs by comma, respecting quoted values
func splitKeyPairs(input string) ([]string, error) {
	var pairs []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, ch := range input {
		switch {
		case (ch == '\'' || ch == '"') && !inQuote:
			inQuote = true
			quoteChar = ch
			current.WriteRune(ch)
		case ch == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
			current.WriteRune(ch)
		case ch == ',' && !inQuote:
			pairs = append(pairs, current.String())
			current.Reset()
		default:
			current.WriteRune(ch)
		}

		// Check for unclosed quote at end
		if i == len(input)-1 && inQuote {
			return nil, fmt.Errorf("unclosed quote in key part")
		}
	}

	// Add the last pair
	if current.Len() > 0 {
		pairs = append(pairs, current.String())
	}

	return pairs, nil
}
