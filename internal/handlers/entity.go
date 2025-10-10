package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// EntityHandler handles HTTP requests for entity collections
type EntityHandler struct {
	db       *gorm.DB
	metadata *metadata.EntityMetadata
}

// NewEntityHandler creates a new entity handler
func NewEntityHandler(db *gorm.DB, entityMetadata *metadata.EntityMetadata) *EntityHandler {
	return &EntityHandler{
		db:       db,
		metadata: entityMetadata,
	}
}

// HandleCollection handles GET requests for entity collections
func (h *EntityHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for entity collections", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	// Parse query options
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid query options", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Get the total count if $count=true is specified
	totalCount := h.getTotalCount(queryOptions, w)
	if totalCount == nil && queryOptions.Count {
		return // Error already written
	}

	// Fetch the results
	sliceValue, err := h.fetchResults(queryOptions)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Calculate next link if pagination is active
	nextLink := h.calculateNextLink(queryOptions, sliceValue, r)

	// Get list of expanded properties
	expandedProps := make([]string, len(queryOptions.Expand))
	for i, exp := range queryOptions.Expand {
		expandedProps[i] = exp.NavigationProperty
	}

	// Write the OData response with navigation links
	metadataProvider := &metadataAdapter{metadata: h.metadata}
	if err := response.WriteODataCollectionWithNavigation(w, r, h.metadata.EntitySetName, sliceValue, totalCount, nextLink, metadataProvider, expandedProps); err != nil {
		// If we can't write the response, log the error but don't try to write another response
		fmt.Printf("Error writing OData response: %v\n", err)
	}
}

// getTotalCount retrieves the total count if requested
func (h *EntityHandler) getTotalCount(queryOptions *query.QueryOptions, w http.ResponseWriter) *int64 {
	if !queryOptions.Count {
		return nil
	}

	var count int64
	countDB := h.db.Model(reflect.New(h.metadata.EntityType).Interface())

	// Apply filter to count query if present
	if queryOptions.Filter != nil {
		countDB = query.ApplyFilterOnly(countDB, queryOptions.Filter, h.metadata)
	}

	if err := countDB.Count(&count).Error; err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return nil
	}
	return &count
}

// fetchResults fetches the results from the database
func (h *EntityHandler) fetchResults(queryOptions *query.QueryOptions) (interface{}, error) {
	// Create a slice to hold the results
	sliceType := reflect.SliceOf(h.metadata.EntityType)
	results := reflect.New(sliceType).Interface()

	// Apply query options to the database query
	db := query.ApplyQueryOptions(h.db, queryOptions, h.metadata)

	// Execute the database query
	if err := db.Find(results).Error; err != nil {
		return nil, err
	}

	// Get the actual slice value (results is a pointer to slice)
	sliceValue := reflect.ValueOf(results).Elem().Interface()

	// Apply $select if specified
	if len(queryOptions.Select) > 0 {
		sliceValue = query.ApplySelect(sliceValue, queryOptions.Select, h.metadata)
	}

	return sliceValue, nil
}

// calculateNextLink calculates the next link URL for pagination
func (h *EntityHandler) calculateNextLink(queryOptions *query.QueryOptions, sliceValue interface{}, r *http.Request) *string {
	if queryOptions.Top == nil {
		return nil
	}

	// Get the actual result count
	resultCount := reflect.ValueOf(sliceValue).Len()

	// If we got exactly $top results, check if there are more records
	if resultCount != *queryOptions.Top {
		return nil
	}

	// Calculate the new skip value for the next page
	currentSkip := 0
	if queryOptions.Skip != nil {
		currentSkip = *queryOptions.Skip
	}
	nextSkip := currentSkip + *queryOptions.Top

	// Check if there are more records
	if h.hasMoreRecords(queryOptions, nextSkip) {
		nextURL := response.BuildNextLink(r, nextSkip)
		return &nextURL
	}

	return nil
}

// hasMoreRecords checks if there are more records available
func (h *EntityHandler) hasMoreRecords(queryOptions *query.QueryOptions, nextSkip int) bool {
	checkSliceType := reflect.SliceOf(h.metadata.EntityType)
	checkResults := reflect.New(checkSliceType).Interface()

	checkDB := h.db

	// Apply the same filters
	if queryOptions.Filter != nil {
		checkDB = query.ApplyFilterOnly(checkDB, queryOptions.Filter, h.metadata)
	}

	// Apply the same order by
	if len(queryOptions.OrderBy) > 0 {
		for _, item := range queryOptions.OrderBy {
			// Sanitize: Only allow real property names from metadata
			valid := false
			for _, prop := range h.metadata.Properties {
				if prop.JsonName == item.Property || prop.Name == item.Property {
					valid = true
					break
				}
			}
			if !valid {
				// Skip invalid property
				continue
			}
			fieldName := query.GetPropertyFieldName(item.Property, h.metadata)
			direction := "ASC"
			if item.Descending {
				direction = "DESC"
			}
			// No user input concatenated directly to SQL, fieldName is safe
			checkDB = checkDB.Order(fmt.Sprintf("%s %s", fieldName, direction))
		}
	}

	// Check if there's at least one more record at the next position
	checkDB = checkDB.Offset(nextSkip).Limit(1)
	if err := checkDB.Find(checkResults).Error; err != nil {
		return false
	}

	checkSliceValue := reflect.ValueOf(checkResults).Elem()
	return checkSliceValue.Len() > 0
}

// HandleEntity handles GET requests for individual entities
func (h *EntityHandler) HandleEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	if r.Method != http.MethodGet {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for individual entities", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	// Parse query options for $expand
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid query options", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Create an instance to hold the result
	result := reflect.New(h.metadata.EntityType).Interface()

	// Build the query condition using the key properties
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid key", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Apply expand (preload navigation properties) if specified
	if len(queryOptions.Expand) > 0 {
		db = query.ApplyExpandOnly(db, queryOptions.Expand, h.metadata)
	}

	if err := db.First(result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if writeErr := response.WriteError(w, http.StatusNotFound, "Entity not found",
				fmt.Sprintf("Entity with key '%s' not found", entityKey)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
		} else {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
		}
		return
	}

	// For individual entities, we return the entity directly (not wrapped in a collection)
	contextURL := fmt.Sprintf("%s/$metadata#%s/$entity", response.BuildBaseURL(r), h.metadata.EntitySetName)

	// Build ordered response
	odataResponse := h.buildOrderedEntityResponse(result, contextURL)

	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	// Write the response
	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		fmt.Printf("Error writing entity response: %v\n", err)
	}
}

// HandleNavigationProperty handles GET requests for navigation properties (e.g., Products(1)/Descriptions)
func (h *EntityHandler) HandleNavigationProperty(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string) {
	if r.Method != http.MethodGet {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for navigation properties", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navigationProperty)
	if navProp == nil {
		if err := response.WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navigationProperty, h.metadata.EntitySetName)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	// Fetch the parent entity with the navigation property preloaded
	parent, err := h.fetchParentEntityWithNav(entityKey, navProp.Name)
	if err != nil {
		h.handleFetchError(w, err, entityKey)
		return
	}

	// Extract and write the navigation property value
	navFieldValue := h.extractNavigationField(parent, navProp.Name)
	if !navFieldValue.IsValid() {
		if err := response.WriteError(w, http.StatusInternalServerError, "Internal error",
			"Could not access navigation property"); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	h.writeNavigationResponse(w, r, navProp, navFieldValue)
}

// findNavigationProperty finds a navigation property by name in the entity metadata
func (h *EntityHandler) findNavigationProperty(navigationProperty string) *metadata.PropertyMetadata {
	for _, prop := range h.metadata.Properties {
		if (prop.JsonName == navigationProperty || prop.Name == navigationProperty) && prop.IsNavigationProp {
			return &prop
		}
	}
	return nil
}

// fetchParentEntityWithNav fetches the parent entity and preloads the specified navigation property
func (h *EntityHandler) fetchParentEntityWithNav(entityKey, navPropertyName string) (interface{}, error) {
	parent := reflect.New(h.metadata.EntityType).Interface()

	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		return nil, err
	}

	db = db.Preload(navPropertyName)
	return parent, db.First(parent).Error
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
		if writeErr := response.WriteError(w, http.StatusNotFound, "Entity not found",
			fmt.Sprintf("Entity with key '%s' not found", entityKey)); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
	} else {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
	}
}

// extractNavigationField extracts the navigation property field value from the parent entity
func (h *EntityHandler) extractNavigationField(parent interface{}, navPropertyName string) reflect.Value {
	parentValue := reflect.ValueOf(parent).Elem()
	return parentValue.FieldByName(navPropertyName)
}

// writeNavigationResponse writes the navigation property response (collection or single entity)
func (h *EntityHandler) writeNavigationResponse(w http.ResponseWriter, r *http.Request, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	if navProp.NavigationIsArray {
		h.writeNavigationCollection(w, r, navProp, navFieldValue)
	} else {
		h.writeSingleNavigationEntity(w, r, navProp, navFieldValue)
	}
}

// writeNavigationCollection writes a collection navigation property response
func (h *EntityHandler) writeNavigationCollection(w http.ResponseWriter, r *http.Request, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	navData := navFieldValue.Interface()
	if err := response.WriteODataCollection(w, r, navProp.NavigationTarget+"s", navData, nil, nil); err != nil {
		fmt.Printf("Error writing navigation property collection: %v\n", err)
	}
}

// writeSingleNavigationEntity writes a single navigation property entity response
func (h *EntityHandler) writeSingleNavigationEntity(w http.ResponseWriter, r *http.Request, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	navData := navFieldValue.Interface()
	navValue := reflect.ValueOf(navData)

	// Handle pointer and check for nil
	if navValue.Kind() == reflect.Ptr {
		if navValue.IsNil() {
			w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
			w.Header().Set("OData-Version", "4.0")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		navValue = navValue.Elem()
	}

	// Build the OData response
	contextURL := fmt.Sprintf("%s/$metadata#%s/$entity", response.BuildBaseURL(r), navProp.NavigationTarget+"s")
	odataResponse := h.buildEntityResponse(navValue, contextURL)

	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		fmt.Printf("Error writing navigation property response: %v\n", err)
	}
}

// buildEntityResponse builds an OData entity response from a reflect.Value
func (h *EntityHandler) buildEntityResponse(navValue reflect.Value, contextURL string) map[string]interface{} {
	odataResponse := response.NewOrderedMap()
	odataResponse.Set("@odata.context", contextURL)

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
	odataResponse.Set("@odata.context", contextURL)

	// Merge the entity fields into the response
	entityValue := reflect.ValueOf(result).Elem()
	entityType := entityValue.Type()

	for i := 0; i < entityValue.NumField(); i++ {
		field := entityType.Field(i)
		if field.IsExported() {
			fieldValue := entityValue.Field(i)
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
