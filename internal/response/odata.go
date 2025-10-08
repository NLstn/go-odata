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

		// Check if the entity is already a map (from $select processing)
		if entity.Kind() == reflect.Map {
			// Entity is already a map, convert it
			entityMap = entity.Interface().(map[string]interface{})

			// Add navigation links for properties that are not in the map
			// and are not expanded
			for _, prop := range metadata.GetProperties() {
				if prop.IsNavigationProp {
					// Check if this property was expanded
					isExpanded := false
					for _, expanded := range expandedProps {
						if expanded == prop.Name || expanded == prop.JsonName {
							isExpanded = true
							break
						}
					}

					// If not expanded and not in the map, add navigation link
					if !isExpanded {
						// Check if the property is already in the map
						if _, exists := entityMap[prop.JsonName]; !exists {
							// Get the entity key value from the map
							keyValue := entityMap[keyProp.JsonName]
							if keyValue != nil {
								navLink := fmt.Sprintf("%s/%s(%v)/%s", baseURL, entitySetName, keyValue, prop.JsonName)
								entityMap[prop.JsonName+"@odata.navigationLink"] = navLink
							}
						}
					}
				}
			}
		} else {
			// Entity is a struct, process fields
			entityMap = make(map[string]interface{})
			entityType := entity.Type()

			for j := 0; j < entity.NumField(); j++ {
				field := entityType.Field(j)
				if !field.IsExported() {
					continue
				}

				fieldValue := entity.Field(j)
				jsonName := getJsonFieldName(field)

				// Find metadata for this property
				var propMeta *PropertyMetadata
				for _, prop := range metadata.GetProperties() {
					if prop.Name == field.Name {
						propMeta = &prop
						break
					}
				}

				// Check if this is a navigation property
				if propMeta != nil && propMeta.IsNavigationProp {
					// Check if this property was expanded
					isExpanded := false
					for _, expanded := range expandedProps {
						if expanded == propMeta.Name || expanded == propMeta.JsonName {
							isExpanded = true
							break
						}
					}

					if isExpanded {
						// Include the expanded data
						entityMap[jsonName] = fieldValue.Interface()
					} else {
						// Add navigation link instead of null value
						// Get the entity key value
						keyFieldValue := entity.FieldByName(keyProp.Name)
						if keyFieldValue.IsValid() {
							keyValue := fmt.Sprintf("%v", keyFieldValue.Interface())
							navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keyValue, propMeta.JsonName)
							entityMap[jsonName+"@odata.navigationLink"] = navLink
						}
					}
				} else {
					// Regular property - include its value
					entityMap[jsonName] = fieldValue.Interface()
				}
			}
		}

		result[i] = entityMap
	}

	return result
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
