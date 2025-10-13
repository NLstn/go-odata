package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
)

// HandleCollection handles GET, HEAD, POST, and OPTIONS requests for entity collections
func (h *EntityHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetCollection(w, r)
	case http.MethodPost:
		h.handlePostEntity(w, r)
	case http.MethodOptions:
		h.handleOptionsCollection(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for entity collections", r.Method)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
	}
}

// handleOptionsCollection handles OPTIONS requests for entity collections
func (h *EntityHandler) handleOptionsCollection(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, POST, OPTIONS")
	w.Header().Set(HeaderODataVersion, "4.0")
	w.WriteHeader(http.StatusOK)
}

// handleGetCollection handles GET requests for entity collections
func (h *EntityHandler) handleGetCollection(w http.ResponseWriter, r *http.Request) {

	// Parse query options
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
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
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
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
	// Parse Prefer header
	pref := preference.ParsePrefer(r)

	// Create a new instance of the entity
	entity := reflect.New(h.metadata.EntityType).Interface()

	// Parse the request body
	if err := json.NewDecoder(r.Body).Decode(entity); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error())); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Validate required properties
	if err := h.validateRequiredProperties(entity); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Missing required properties", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Create the entity in the database
	if err := h.db.Create(entity).Error; err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Build the Location header with the key(s) of the created entity
	location := h.buildEntityLocation(r, entity)

	// Set common headers
	w.Header().Set(HeaderODataVersion, "4.0")
	w.Header().Set("Location", location)

	// Add Preference-Applied header if a preference was specified
	if applied := pref.GetPreferenceApplied(); applied != "" {
		w.Header().Set(HeaderPreferenceApplied, applied)
	}

	// Determine whether to return content based on preferences
	if pref.ShouldReturnContent(true) {
		// Return representation (default for POST)
		contextURL := fmt.Sprintf(ODataContextFormat, response.BuildBaseURL(r), h.metadata.EntitySetName)
		odataResponse := h.buildOrderedEntityResponse(entity, contextURL)

		// Generate and set ETag header if entity has an ETag property
		if etagValue := etag.Generate(entity, h.metadata); etagValue != "" {
			w.Header().Set(HeaderETag, etagValue)
		}

		w.Header().Set(HeaderContentType, ContentTypeJSON)
		w.WriteHeader(http.StatusCreated)

		if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
			fmt.Printf(LogMsgErrorWritingEntityResponse, err)
		}
	} else {
		// Return minimal (204 No Content)
		w.WriteHeader(http.StatusNoContent)
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

// HandleCount handles GET, HEAD, and OPTIONS requests for entity collection count (e.g., /Products/$count)
func (h *EntityHandler) HandleCount(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetCount(w, r)
	case http.MethodOptions:
		h.handleOptionsCount(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for $count", r.Method)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
	}
}

// handleGetCount handles GET requests for entity collection count
func (h *EntityHandler) handleGetCount(w http.ResponseWriter, r *http.Request) {
	// Parse query options (primarily for $filter support)
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
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
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Write the count as plain text according to OData v4 spec
	w.Header().Set(HeaderContentType, "text/plain")
	w.Header().Set(HeaderODataVersion, "4.0")
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

	if _, err := fmt.Fprintf(w, "%d", count); err != nil {
		fmt.Printf("Error writing count response: %v\n", err)
	}
}

// handleOptionsCount handles OPTIONS requests for $count endpoint
func (h *EntityHandler) handleOptionsCount(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.Header().Set(HeaderODataVersion, "4.0")
	w.WriteHeader(http.StatusOK)
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
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return nil
	}
	return &count
}

// fetchResults fetches the results from the database
func (h *EntityHandler) fetchResults(queryOptions *query.QueryOptions) (interface{}, error) {
	// If $top is specified, fetch $top + 1 records to check if there are more results
	// This avoids an extra database query for pagination
	modifiedOptions := *queryOptions
	if queryOptions.Top != nil {
		topPlusOne := *queryOptions.Top + 1
		modifiedOptions.Top = &topPlusOne
	}

	// Apply query options to the database query
	db := query.ApplyQueryOptions(h.db, &modifiedOptions, h.metadata)

	// Check if we need to use map results (for $apply transformations)
	if query.ShouldUseMapResults(queryOptions) {
		// Use maps for aggregated/transformed results
		var results []map[string]interface{}
		if err := db.Find(&results).Error; err != nil {
			return nil, err
		}
		return results, nil
	}

	// Create a slice to hold the results
	sliceType := reflect.SliceOf(h.metadata.EntityType)
	results := reflect.New(sliceType).Interface()

	// Execute the database query
	if err := db.Find(results).Error; err != nil {
		return nil, err
	}

	// Get the actual slice value (results is a pointer to slice)
	sliceValue := reflect.ValueOf(results).Elem().Interface()

	// Apply $search if specified (database-agnostic in-memory filtering)
	if queryOptions.Search != "" {
		sliceValue = query.ApplySearch(sliceValue, queryOptions.Search, h.metadata)
	}

	// Apply $select if specified
	if len(queryOptions.Select) > 0 {
		sliceValue = query.ApplySelect(sliceValue, queryOptions.Select, h.metadata, queryOptions.Expand)
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
