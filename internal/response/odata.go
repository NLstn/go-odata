package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

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
	GetKeyProperty() *PropertyMetadata
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
func addNavigationLinks(data interface{}, metadata EntityMetadataProvider, expandedProps []string, r *http.Request, entitySetName string) []map[string]interface{} {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		return nil
	}

	result := make([]map[string]interface{}, dataValue.Len())
	baseURL := buildBaseURL(r)
	keyProp := metadata.GetKeyProperty()

	for i := 0; i < dataValue.Len(); i++ {
		entity := dataValue.Index(i)
		var entityMap map[string]interface{}

		if entity.Kind() == reflect.Map {
			entityMap = processMapEntity(entity, metadata, expandedProps, baseURL, entitySetName, keyProp)
		} else {
			entityMap = processStructEntity(entity, metadata, expandedProps, baseURL, entitySetName, keyProp)
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
			if keyValue := entityMap[keyProp.JsonName]; keyValue != nil {
				navLink := fmt.Sprintf("%s/%s(%v)/%s", baseURL, entitySetName, keyValue, prop.JsonName)
				entityMap[prop.JsonName+"@odata.navigationLink"] = navLink
			}
		}
	}

	return entityMap
}

// processStructEntity processes an entity that is a struct and converts it to a map with navigation links
func processStructEntity(entity reflect.Value, metadata EntityMetadataProvider, expandedProps []string, baseURL, entitySetName string, keyProp *PropertyMetadata) map[string]interface{} {
	entityMap := make(map[string]interface{})
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
			processNavigationProperty(entityMap, entity, propMeta, fieldValue, jsonName, expandedProps, baseURL, entitySetName, keyProp)
		} else {
			// Regular property - include its value
			entityMap[jsonName] = fieldValue.Interface()
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

// processNavigationProperty handles navigation properties by either including expanded data or adding a navigation link
func processNavigationProperty(entityMap map[string]interface{}, entity reflect.Value, propMeta *PropertyMetadata, fieldValue reflect.Value, jsonName string, expandedProps []string, baseURL, entitySetName string, keyProp *PropertyMetadata) {
	if isPropertyExpanded(*propMeta, expandedProps) {
		// Include the expanded data
		entityMap[jsonName] = fieldValue.Interface()
	} else {
		// Add navigation link instead of null value
		keyFieldValue := entity.FieldByName(keyProp.Name)
		if keyFieldValue.IsValid() {
			keyValue := fmt.Sprintf("%v", keyFieldValue.Interface())
			navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keyValue, propMeta.JsonName)
			entityMap[jsonName+"@odata.navigationLink"] = navLink
		}
	}
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
	EntityKey          string
	NavigationProperty string // For paths like Products(1)/Descriptions
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

	components := &ODataURLComponents{}

	// Extract entity set and key
	pathParts := strings.Split(u.Path, "/")
	if len(pathParts) > 0 {
		entitySet := pathParts[0]

		// Check for key in parentheses: Products(1)
		if idx := strings.Index(entitySet, "("); idx != -1 {
			if strings.HasSuffix(entitySet, ")") {
				components.EntityKey = entitySet[idx+1 : len(entitySet)-1]
				components.EntitySet = entitySet[:idx]
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
