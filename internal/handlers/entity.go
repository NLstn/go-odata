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

// HandleCollection handles GET and POST requests for entity collections
func (h *EntityHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetCollection(w, r)
	case http.MethodPost:
		h.handlePostEntity(w, r)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for entity collections", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
	}
}

// handleGetCollection handles GET requests for entity collections
func (h *EntityHandler) handleGetCollection(w http.ResponseWriter, r *http.Request) {

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

	// Calculate next link if pagination is active and trim results if needed
	nextLink, needsTrimming := h.calculateNextLink(queryOptions, sliceValue, r)
	if needsTrimming && queryOptions.Top != nil {
		// Trim the results to $top (we fetched $top + 1 to check for more pages)
		sliceValue = h.trimResults(sliceValue, *queryOptions.Top)
	}

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

// handlePostEntity handles POST requests to create new entities in a collection
func (h *EntityHandler) handlePostEntity(w http.ResponseWriter, r *http.Request) {
	// Create a new instance of the entity
	entity := reflect.New(h.metadata.EntityType).Interface()

	// Parse the request body
	if err := json.NewDecoder(r.Body).Decode(entity); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request body",
			fmt.Sprintf("Failed to parse JSON: %v", err.Error())); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Validate required properties
	if err := h.validateRequiredProperties(entity); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Missing required properties", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Create the entity in the database
	if err := h.db.Create(entity).Error; err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Build the Location header with the key(s) of the created entity
	location := h.buildEntityLocation(r, entity)

	// Build the context URL
	contextURL := fmt.Sprintf("%s/$metadata#%s/$entity", response.BuildBaseURL(r), h.metadata.EntitySetName)

	// Build ordered response
	odataResponse := h.buildOrderedEntityResponse(entity, contextURL)

	// Set headers
	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)

	// Write the response
	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		fmt.Printf("Error writing entity response: %v\n", err)
	}
}

// validateRequiredProperties validates that all required properties are provided
func (h *EntityHandler) validateRequiredProperties(entity interface{}) error {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	var missingFields []string
	for _, prop := range h.metadata.Properties {
		if !prop.IsRequired || prop.IsKey {
			continue // Skip non-required and key fields (keys can be auto-generated)
		}

		fieldValue := entityValue.FieldByName(prop.Name)
		if !fieldValue.IsValid() {
			continue
		}

		// Check if the field is zero value
		if fieldValue.IsZero() {
			missingFields = append(missingFields, prop.JsonName)
		}
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required properties: %s", strings.Join(missingFields, ", "))
	}

	return nil
}

// buildEntityLocation builds the Location header URL for a created entity
func (h *EntityHandler) buildEntityLocation(r *http.Request, entity interface{}) string {
	baseURL := response.BuildBaseURL(r)
	entitySetName := h.metadata.EntitySetName

	// Extract key values from the entity
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Handle single key vs composite key
	if len(h.metadata.KeyProperties) == 1 {
		// Single key
		keyProp := h.metadata.KeyProperties[0]
		keyValue := entityValue.FieldByName(keyProp.Name)
		return fmt.Sprintf("%s/%s(%v)", baseURL, entitySetName, keyValue.Interface())
	}

	// Composite key
	var keyParts []string
	for _, keyProp := range h.metadata.KeyProperties {
		keyValue := entityValue.FieldByName(keyProp.Name)
		// Format based on type
		switch keyValue.Kind() {
		case reflect.String:
			keyParts = append(keyParts, fmt.Sprintf("%s='%v'", keyProp.JsonName, keyValue.Interface()))
		default:
			keyParts = append(keyParts, fmt.Sprintf("%s=%v", keyProp.JsonName, keyValue.Interface()))
		}
	}

	return fmt.Sprintf("%s/%s(%s)", baseURL, entitySetName, strings.Join(keyParts, ","))
}

// HandleCount handles GET requests for entity collection count (e.g., /Products/$count)
func (h *EntityHandler) HandleCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for $count", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	// Parse query options (primarily for $filter support)
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid query options", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
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
		return
	}

	// Write the count as plain text according to OData v4 spec
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, "%d", count); err != nil {
		fmt.Printf("Error writing count response: %v\n", err)
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

	// If $top is specified, fetch $top + 1 records to check if there are more results
	// This avoids an extra database query for pagination
	modifiedOptions := *queryOptions
	if queryOptions.Top != nil {
		topPlusOne := *queryOptions.Top + 1
		modifiedOptions.Top = &topPlusOne
	}

	// Apply query options to the database query
	db := query.ApplyQueryOptions(h.db, &modifiedOptions, h.metadata)

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
// Returns the nextLink and a boolean indicating if results need trimming
func (h *EntityHandler) calculateNextLink(queryOptions *query.QueryOptions, sliceValue interface{}, r *http.Request) (*string, bool) {
	if queryOptions.Top == nil {
		return nil, false
	}

	// Get the actual result count
	resultCount := reflect.ValueOf(sliceValue).Len()

	// If we got more than $top results, it means there are more pages
	// We fetched $top + 1 to determine this without an extra query
	if resultCount > *queryOptions.Top {
		// Calculate the new skip value for the next page
		currentSkip := 0
		if queryOptions.Skip != nil {
			currentSkip = *queryOptions.Skip
		}
		nextSkip := currentSkip + *queryOptions.Top

		nextURL := response.BuildNextLink(r, nextSkip)
		return &nextURL, true // true indicates we need to trim the results
	}

	return nil, false
}

// trimResults trims a slice to the specified length
func (h *EntityHandler) trimResults(sliceValue interface{}, maxLen int) interface{} {
	v := reflect.ValueOf(sliceValue)
	if v.Kind() != reflect.Slice {
		return sliceValue
	}

	if v.Len() <= maxLen {
		return sliceValue
	}

	// Check if sliceValue is a slice of maps (from $select)
	if v.Len() > 0 && v.Index(0).Kind() == reflect.Map {
		// Handle slice of maps
		mapSlice, ok := sliceValue.([]map[string]interface{})
		if ok {
			return mapSlice[:maxLen]
		}
	}

	// Handle regular slice
	return v.Slice(0, maxLen).Interface()
}

// HandleEntity handles GET, DELETE, and PATCH requests for individual entities
func (h *EntityHandler) HandleEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetEntity(w, r, entityKey)
	case http.MethodDelete:
		h.handleDeleteEntity(w, r, entityKey)
	case http.MethodPatch:
		h.handlePatchEntity(w, r, entityKey)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for individual entities", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
	}
}

// handleGetEntity handles GET requests for individual entities
func (h *EntityHandler) handleGetEntity(w http.ResponseWriter, r *http.Request, entityKey string) {

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

// handleDeleteEntity handles DELETE requests for individual entities
func (h *EntityHandler) handleDeleteEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	_ = r // Reserved for future use (e.g., conditional deletes with If-Match header)
	// Create an instance to hold the entity to be deleted
	entity := reflect.New(h.metadata.EntityType).Interface()

	// Build the query condition using the key properties
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid key", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// First, check if the entity exists
	if err := db.First(entity).Error; err != nil {
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

	// Delete the entity
	if err := h.db.Delete(entity).Error; err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Return 204 No Content according to OData v4 spec
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusNoContent)
}

// handlePatchEntity handles PATCH requests for individual entities
func (h *EntityHandler) handlePatchEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	// Create an instance to hold the entity to be updated
	entity := reflect.New(h.metadata.EntityType).Interface()

	// Build the query condition using the key properties
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid key", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// First, check if the entity exists
	if err := db.First(entity).Error; err != nil {
		h.handleFetchError(w, err, entityKey)
		return
	}

	// Parse the request body to get the update data
	updateData, err := h.parsePatchRequestBody(r, w)
	if err != nil {
		return // Error already written by parsePatchRequestBody
	}

	// Validate that we're not trying to update key properties
	if err := h.validateKeyPropertiesNotUpdated(updateData, w); err != nil {
		return // Error already written by validateKeyPropertiesNotUpdated
	}

	// Apply updates using GORM's Updates method which only updates provided fields
	if err := h.db.Model(entity).Updates(updateData).Error; err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Return 204 No Content according to OData v4 spec
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusNoContent)
}

// parsePatchRequestBody parses the JSON request body for PATCH operations
func (h *EntityHandler) parsePatchRequestBody(r *http.Request, w http.ResponseWriter) (map[string]interface{}, error) {
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request body",
			fmt.Sprintf("Failed to parse JSON: %v", err.Error())); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return nil, err
	}
	return updateData, nil
}

// validateKeyPropertiesNotUpdated validates that key properties are not being updated
func (h *EntityHandler) validateKeyPropertiesNotUpdated(updateData map[string]interface{}, w http.ResponseWriter) error {
	for _, keyProp := range h.metadata.KeyProperties {
		if _, exists := updateData[keyProp.JsonName]; exists {
			err := fmt.Errorf("key property '%s' cannot be modified", keyProp.JsonName)
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Cannot update key property", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return err
		}
		// Also check using the struct field name
		if _, exists := updateData[keyProp.Name]; exists {
			err := fmt.Errorf("key property '%s' cannot be modified", keyProp.Name)
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Cannot update key property", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return err
		}
	}
	return nil
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

	h.writeNavigationResponse(w, r, entityKey, navProp, navFieldValue)
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

// findStructuralProperty finds a structural (non-navigation) property by name in the entity metadata
func (h *EntityHandler) findStructuralProperty(propertyName string) *metadata.PropertyMetadata {
	for _, prop := range h.metadata.Properties {
		if (prop.JsonName == propertyName || prop.Name == propertyName) && !prop.IsNavigationProp {
			return &prop
		}
	}
	return nil
}

// IsNavigationProperty checks if a property name is a navigation property
func (h *EntityHandler) IsNavigationProperty(propertyName string) bool {
	return h.findNavigationProperty(propertyName) != nil
}

// IsStructuralProperty checks if a property name is a structural property
func (h *EntityHandler) IsStructuralProperty(propertyName string) bool {
	return h.findStructuralProperty(propertyName) != nil
}

// HandleStructuralProperty handles GET requests for structural properties (e.g., Products(1)/Name)
// When isValue is true, returns the raw property value without JSON wrapper (e.g., Products(1)/Name/$value)
func (h *EntityHandler) HandleStructuralProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertyName string, isValue bool) {
	if r.Method != http.MethodGet {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for property access", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	// Find and validate the structural property
	prop := h.findStructuralProperty(propertyName)
	if prop == nil {
		if err := response.WriteError(w, http.StatusNotFound, "Property not found",
			fmt.Sprintf("'%s' is not a valid property for %s", propertyName, h.metadata.EntitySetName)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	// Fetch the entity with only the needed property and key columns
	entity := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid key", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Apply select to fetch only the needed property and key columns
	db = h.applyStructuralPropertySelect(db, prop)

	if err := db.First(entity).Error; err != nil {
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

	// Extract the property value
	entityValue := reflect.ValueOf(entity).Elem()
	fieldValue := entityValue.FieldByName(prop.Name)
	if !fieldValue.IsValid() {
		if err := response.WriteError(w, http.StatusInternalServerError, "Internal error",
			"Could not access property"); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	// If $value is requested, return the raw value without JSON wrapper
	if isValue {
		h.writeRawPropertyValue(w, fieldValue)
		return
	}

	// Build the OData response according to OData v4 spec
	contextURL := fmt.Sprintf("%s/$metadata#%s(%s)/%s", response.BuildBaseURL(r), h.metadata.EntitySetName, entityKey, prop.JsonName)
	odataResponse := map[string]interface{}{
		"@odata.context": contextURL,
		"value":          fieldValue.Interface(),
	}

	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		fmt.Printf("Error writing property response: %v\n", err)
	}
}

// writeRawPropertyValue writes a property value in raw format for /$value requests
func (h *EntityHandler) writeRawPropertyValue(w http.ResponseWriter, fieldValue reflect.Value) {
	// Set appropriate content type based on the value type
	valueInterface := fieldValue.Interface()

	// Determine content type based on the property type
	switch fieldValue.Kind() {
	case reflect.String:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case reflect.Bool:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	default:
		// For other types, use application/octet-stream
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	// Write the raw value
	if _, err := fmt.Fprintf(w, "%v", valueInterface); err != nil {
		fmt.Printf("Error writing raw value: %v\n", err)
	}
}

// applyStructuralPropertySelect applies SELECT clause to fetch only the structural property and key columns
func (h *EntityHandler) applyStructuralPropertySelect(db *gorm.DB, prop *metadata.PropertyMetadata) *gorm.DB {
	// Build select columns list: property + all key properties
	// Use struct field names - GORM will handle column name conversion
	selectColumns := []string{prop.Name}
	for _, keyProp := range h.metadata.KeyProperties {
		// Avoid duplicates if the property itself is a key
		if keyProp.Name != prop.Name {
			selectColumns = append(selectColumns, keyProp.Name)
		}
	}
	return db.Select(selectColumns)
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
func (h *EntityHandler) writeNavigationResponse(w http.ResponseWriter, r *http.Request, entityKey string, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	if navProp.NavigationIsArray {
		h.writeNavigationCollection(w, r, entityKey, navProp, navFieldValue)
	} else {
		h.writeSingleNavigationEntity(w, r, entityKey, navProp, navFieldValue)
	}
}

// writeNavigationCollection writes a collection navigation property response
func (h *EntityHandler) writeNavigationCollection(w http.ResponseWriter, r *http.Request, entityKey string, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	navData := navFieldValue.Interface()
	// Build the navigation path according to OData V4 spec: EntitySet(key)/NavigationProperty
	navigationPath := fmt.Sprintf("%s(%s)/%s", h.metadata.EntitySetName, entityKey, navProp.JsonName)
	if err := response.WriteODataCollection(w, r, navigationPath, navData, nil, nil); err != nil {
		fmt.Printf("Error writing navigation property collection: %v\n", err)
	}
}

// writeSingleNavigationEntity writes a single navigation property entity response
func (h *EntityHandler) writeSingleNavigationEntity(w http.ResponseWriter, r *http.Request, entityKey string, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
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

	// Build the OData response with navigation path according to OData V4 spec: EntitySet(key)/NavigationProperty/$entity
	navigationPath := fmt.Sprintf("%s(%s)/%s", h.metadata.EntitySetName, entityKey, navProp.JsonName)
	contextURL := fmt.Sprintf("%s/$metadata#%s/$entity", response.BuildBaseURL(r), navigationPath)
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
