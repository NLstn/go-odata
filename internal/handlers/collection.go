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
	"github.com/nlstn/go-odata/internal/skiptoken"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
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
	w.WriteHeader(http.StatusOK)
}

// handleGetCollection handles GET requests for entity collections
func (h *EntityHandler) handleGetCollection(w http.ResponseWriter, r *http.Request) {

	// Parse Prefer header for server-side preferences
	pref := preference.ParsePrefer(r)

	// Parse query options
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Handle delta requests using $deltatoken
	if queryOptions.DeltaToken != nil {
		h.handleDeltaCollection(w, r, *queryOptions.DeltaToken)
		return
	}

	// Validate skiptoken if present
	if err := h.validateSkipToken(queryOptions); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid $skiptoken", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Validate query options for complex types
	if err := h.validateComplexTypeUsage(queryOptions); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Unsupported query option", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Apply odata.maxpagesize preference if specified
	if pref.MaxPageSize != nil {
		queryOptions = h.applyMaxPageSize(queryOptions, *pref.MaxPageSize)
	}

	// Invoke BeforeReadCollection hooks to obtain scopes
	scopes, err := callBeforeReadCollection(h.metadata, r, queryOptions)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Get the total count if $count=true is specified
	totalCount := h.getTotalCount(queryOptions, w, scopes)
	if totalCount == nil && queryOptions.Count {
		return // Error already written
	}

	// Fetch the results
	sliceValue, err := h.fetchResults(queryOptions, scopes)
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

	// Invoke AfterReadCollection hooks allowing overrides
	if override, hasOverride, hookErr := callAfterReadCollection(h.metadata, r, queryOptions, sliceValue); hookErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", hookErr.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	} else if hasOverride {
		sliceValue = override
	}

	var deltaLink *string
	if pref.TrackChangesRequested {
		if !h.supportsTrackChanges() {
			if writeErr := response.WriteError(w, http.StatusNotImplemented, ErrMsgNotImplemented,
				"Change tracking is not enabled for this entity set"); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}

		token, err := h.tracker.CurrentToken(h.metadata.EntitySetName)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError, err.Error()); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}

		link := response.BuildDeltaLink(r, token)
		deltaLink = &link
		pref.ApplyTrackChanges()
	}

	// Add Preference-Applied header if any preference was applied
	if applied := pref.GetPreferenceApplied(); applied != "" {
		w.Header().Set(HeaderPreferenceApplied, applied)
	}

	// Get list of expanded properties
	expandedProps := make([]string, len(queryOptions.Expand))
	for i, exp := range queryOptions.Expand {
		expandedProps[i] = exp.NavigationProperty
	}

	// Write the OData response with navigation links
	metadataProvider := newMetadataAdapter(h.metadata, h.namespace)
	if err := response.WriteODataCollectionWithNavigationAndDelta(w, r, h.metadata.EntitySetName, sliceValue, totalCount, nextLink, deltaLink, metadataProvider, expandedProps, h.metadata); err != nil {
		// If we can't write the response, log the error but don't try to write another response
		fmt.Printf("Error writing OData response: %v\n", err)
	}
}

func (h *EntityHandler) handleDeltaCollection(w http.ResponseWriter, r *http.Request, token string) {
	if !h.supportsTrackChanges() {
		if writeErr := response.WriteError(w, http.StatusNotImplemented, ErrMsgNotImplemented,
			"Change tracking is not enabled for this entity set"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	entitySet, err := h.tracker.EntitySetFromToken(token)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"Invalid $deltatoken value"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	if entitySet != h.metadata.EntitySetName {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"Delta token does not match the requested entity set"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	events, newToken, err := h.tracker.ChangesSince(token)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	entries := h.buildDeltaEntries(r, events)
	deltaLink := response.BuildDeltaLink(r, newToken)

	if err := response.WriteODataDeltaResponse(w, r, h.metadata.EntitySetName, entries, &deltaLink); err != nil {
		fmt.Printf("Error writing delta response: %v\n", err)
	}
}

func (h *EntityHandler) buildDeltaEntries(r *http.Request, events []trackchanges.ChangeEvent) []map[string]interface{} {
	metadataLevel := response.GetODataMetadataLevel(r)
	includeMetadata := metadataLevel != "none"
	baseURL := response.BuildBaseURL(r)
	entityTypeAnnotation := ""
	if metadataLevel == "full" {
		entityTypeAnnotation = fmt.Sprintf("#ODataService.%s", h.metadata.EntityName)
	}

	entries := make([]map[string]interface{}, 0, len(events))

	for _, event := range events {
		entityID := response.BuildEntityID(h.metadata.EntitySetName, event.KeyValues)
		resourceID := baseURL + "/" + entityID

		switch event.Type {
		case trackchanges.ChangeTypeAdded, trackchanges.ChangeTypeUpdated:
			entry := make(map[string]interface{})
			for k, v := range event.Data {
				entry[k] = v
			}
			if includeMetadata {
				entry["@odata.id"] = resourceID
				if h.metadata.ETagProperty != nil {
					if etagValue := etag.Generate(event.Data, h.metadata); etagValue != "" {
						entry["@odata.etag"] = etagValue
					}
				}
				if entityTypeAnnotation != "" {
					entry["@odata.type"] = entityTypeAnnotation
				}
			}
			entries = append(entries, entry)
		case trackchanges.ChangeTypeDeleted:
			entry := make(map[string]interface{})
			if includeMetadata {
				entry["@odata.id"] = resourceID
				if entityTypeAnnotation != "" {
					entry["@odata.type"] = entityTypeAnnotation
				}
			}
			entry["@odata.removed"] = map[string]string{"reason": "deleted"}
			for k, v := range event.KeyValues {
				entry[k] = v
			}
			entries = append(entries, entry)
		}
	}

	return entries
}

// handlePostEntity handles POST requests to create new entities in a collection
func (h *EntityHandler) handlePostEntity(w http.ResponseWriter, r *http.Request) {
	// Validate Content-Type header
	if err := validateContentType(w, r); err != nil {
		return
	}

	// Parse Prefer header
	pref := preference.ParsePrefer(r)

	// Parse the request body as a map first to detect @odata.bind annotations
	var requestData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error())); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Create a new instance of the entity
	entity := reflect.New(h.metadata.EntityType).Interface()

	// Convert the map back to JSON and decode into the entity structure
	// This properly handles all the type conversions
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, "Failed to process request data", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}
	if err := json.Unmarshal(jsonData, entity); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error())); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Process @odata.bind annotations to establish navigation property relationships
	if err := h.processODataBindAnnotations(entity, requestData, h.db); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid @odata.bind annotation", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Clear auto-increment key properties (ignore client-provided values)
	// This ensures the database auto-generates the ID
	h.clearAutoIncrementKeys(entity)

	// Validate required properties
	if err := h.validateRequiredProperties(entity); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Missing required properties", err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Call BeforeCreate hook if it exists
	if err := h.callBeforeCreate(entity, r); err != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", err.Error()); writeErr != nil {
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

	// Call AfterCreate hook if it exists
	if err := h.callAfterCreate(entity, r); err != nil {
		// Log the error but don't fail the request since the entity was already created
		fmt.Printf("AfterCreate hook failed: %v\n", err)
	}

	h.recordChange(entity, trackchanges.ChangeTypeAdded)

	// Build the Location header with the key(s) of the created entity
	location := h.buildEntityLocation(r, entity)

	// Set common headers
	w.Header().Set("Location", location)

	// Add Preference-Applied header if a preference was specified
	if applied := pref.GetPreferenceApplied(); applied != "" {
		w.Header().Set(HeaderPreferenceApplied, applied)
	}

	// Determine whether to return content based on preferences
	if pref.ShouldReturnContent(true) {
		// Return representation (default for POST)
		// Set OData-EntityId header as per OData v4 spec
		// Using helper function to preserve exact capitalization
		SetODataHeader(w, HeaderODataEntityId, location)

		h.writeEntityResponseWithETag(w, r, entity, "", http.StatusCreated)
	} else {
		// Return minimal (204 No Content)
		// Set OData-EntityId header as per OData v4 spec
		// Using helper function to preserve exact capitalization
		SetODataHeader(w, HeaderODataEntityId, location)
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

// clearAutoIncrementKeys clears the values of auto-increment key properties
// This ensures that client-provided ID values are ignored and the database
// auto-generates the ID value as per OData spec requirements
func (h *EntityHandler) clearAutoIncrementKeys(entity interface{}) {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// For composite keys, we don't clear any values - composite keys are typically not auto-increment
	if len(h.metadata.KeyProperties) > 1 {
		return
	}

	// For single keys, check if it should be auto-incremented
	for _, keyProp := range h.metadata.KeyProperties {
		// If explicitly marked as autoIncrement:false, don't clear
		if strings.Contains(keyProp.GormTag, "autoIncrement:false") {
			continue
		}

		field := entityValue.FieldByName(keyProp.Name)
		if !field.IsValid() || !field.CanSet() {
			continue
		}

		// In GORM, unsigned integer primary keys (uint, uint32, uint64) default to auto-increment
		// Signed integers (int, int32, int64) do NOT auto-increment by default
		// We only clear unsigned integer keys as those are auto-increment by convention
		switch field.Kind() {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			// Unsigned integers are auto-increment in GORM unless marked otherwise
			field.SetZero()
		}
		// For signed integers, strings, or other types, don't clear them
		// They typically require client-provided values
	}
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

	// Invoke BeforeReadCollection hooks for count requests
	scopes, hookErr := callBeforeReadCollection(h.metadata, r, queryOptions)
	if hookErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", hookErr.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	var count int64
	countDB := h.db.Model(reflect.New(h.metadata.EntityType).Interface())
	if len(scopes) > 0 {
		countDB = countDB.Scopes(scopes...)
	}

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
	w.WriteHeader(http.StatusOK)
}

// getTotalCount retrieves the total count if requested
func (h *EntityHandler) getTotalCount(queryOptions *query.QueryOptions, w http.ResponseWriter, scopes []func(*gorm.DB) *gorm.DB) *int64 {
	if !queryOptions.Count {
		return nil
	}

	var count int64
	countDB := h.db.Model(reflect.New(h.metadata.EntityType).Interface())
	if len(scopes) > 0 {
		countDB = countDB.Scopes(scopes...)
	}

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
func (h *EntityHandler) fetchResults(queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	// If $top is specified, fetch $top + 1 records to check if there are more results
	// This avoids an extra database query for pagination
	modifiedOptions := *queryOptions
	if queryOptions.Top != nil {
		topPlusOne := *queryOptions.Top + 1
		modifiedOptions.Top = &topPlusOne
	}

	// Start with the base database query
	db := h.db
	if len(scopes) > 0 {
		db = db.Scopes(scopes...)
	}

	// Apply skiptoken filter if present (must be done before other query options)
	if queryOptions.SkipToken != nil {
		db = h.applySkipTokenFilter(db, queryOptions)
	}

	// Apply query options to the database query
	db = query.ApplyQueryOptions(db, &modifiedOptions, h.metadata)

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
		// Always use $skiptoken for server-driven paging
		// This follows OData v4 best practices
		nextURL := h.buildNextLinkWithSkipToken(queryOptions, sliceValue, r)
		if nextURL != nil {
			return nextURL, true
		}

		// Fall back to $skip-based pagination only if skiptoken creation fails
		currentSkip := 0
		if queryOptions.Skip != nil {
			currentSkip = *queryOptions.Skip
		}
		nextSkip := currentSkip + *queryOptions.Top

		fallbackURL := response.BuildNextLink(r, nextSkip)
		return &fallbackURL, true // true indicates we need to trim the results
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

// applyMaxPageSize applies the odata.maxpagesize preference to query options
// If $top is not specified or is greater than maxpagesize, set $top to maxpagesize
func (h *EntityHandler) applyMaxPageSize(queryOptions *query.QueryOptions, maxPageSize int) *query.QueryOptions {
	if queryOptions.Top == nil || *queryOptions.Top > maxPageSize {
		queryOptions.Top = &maxPageSize
	}
	return queryOptions
}

// buildNextLinkWithSkipToken builds a nextLink using $skiptoken for server-driven paging
func (h *EntityHandler) buildNextLinkWithSkipToken(queryOptions *query.QueryOptions, sliceValue interface{}, r *http.Request) *string {
	// Get the last entity in the result set (which should be at index $top)
	v := reflect.ValueOf(sliceValue)
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return nil
	}

	// The last entity we want to return is at index $top - 1
	// (we fetched $top + 1 to check for more pages)
	lastIndex := *queryOptions.Top - 1
	if lastIndex < 0 || lastIndex >= v.Len() {
		return nil
	}

	lastEntity := v.Index(lastIndex).Interface()

	// Extract key property names
	keyProps := make([]string, len(h.metadata.KeyProperties))
	for i, kp := range h.metadata.KeyProperties {
		keyProps[i] = kp.JsonName
	}

	// Extract orderby property names
	orderByProps := make([]string, len(queryOptions.OrderBy))
	for i, ob := range queryOptions.OrderBy {
		orderByProps[i] = ob.Property
		if ob.Descending {
			orderByProps[i] += " desc"
		}
	}

	// Create skip token from the last entity
	token, err := skiptoken.ExtractFromEntity(lastEntity, keyProps, orderByProps)
	if err != nil {
		// If we can't create a skiptoken, fall back to $skip
		return nil
	}

	// Encode the skip token
	encoded, err := skiptoken.Encode(token)
	if err != nil {
		return nil
	}

	// Build the next link with $skiptoken
	nextURL := response.BuildNextLinkWithSkipToken(r, encoded)
	return &nextURL
}

// applySkipTokenFilter decodes the skiptoken and applies appropriate WHERE clauses
// to skip to the correct position in the result set
func (h *EntityHandler) applySkipTokenFilter(db *gorm.DB, queryOptions *query.QueryOptions) *gorm.DB {
	if queryOptions.SkipToken == nil {
		return db
	}

	// Decode the skip token
	token, err := skiptoken.Decode(*queryOptions.SkipToken)
	if err != nil {
		// Invalid skiptoken - just return db as-is
		// The query will return from the beginning
		return db
	}

	// Build WHERE clause based on orderby and key values
	// For ordered queries, we need to filter based on the orderby columns
	// and use the key as a tiebreaker

	if len(queryOptions.OrderBy) > 0 {
		// Build a compound WHERE clause for ordered results
		// For example, with ORDER BY Price DESC, Name ASC, ID:
		// WHERE (Price < ? OR (Price = ? AND Name > ?) OR (Price = ? AND Name = ? AND ID > ?))

		// This is a simplified implementation that handles single orderby column
		// A full implementation would handle multiple orderby columns with proper logic

		orderByProp := queryOptions.OrderBy[0]
		orderByValue, ok := token.OrderByValues[orderByProp.Property]
		if !ok {
			return db
		}

		// Get the key property value
		var keyValue interface{}
		for keyProp := range token.KeyValues {
			keyValue = token.KeyValues[keyProp]
			break
		}

		// Build the WHERE clause
		// Find the database column name for the orderby property
		var orderByColumnName string
		if orderByMetadata := h.metadata.FindProperty(orderByProp.Property); orderByMetadata != nil {
			orderByColumnName = toSnakeCase(orderByMetadata.Name)
		}

		if orderByColumnName == "" {
			return db
		}

		// Find the database column name for the key property
		var keyColumnName string
		for _, keyProp := range h.metadata.KeyProperties {
			keyColumnName = toSnakeCase(keyProp.Name)
			break
		}

		if orderByProp.Descending {
			// For descending order: WHERE col < ? OR (col = ? AND key > ?)
			db = db.Where(fmt.Sprintf("(%s < ? OR (%s = ? AND %s > ?))",
				orderByColumnName, orderByColumnName, keyColumnName),
				orderByValue, orderByValue, keyValue)
		} else {
			// For ascending order: WHERE col > ? OR (col = ? AND key > ?)
			db = db.Where(fmt.Sprintf("(%s > ? OR (%s = ? AND %s > ?))",
				orderByColumnName, orderByColumnName, keyColumnName),
				orderByValue, orderByValue, keyValue)
		}
	} else {
		// No orderby - just filter by key
		// WHERE key > ?
		var keyColumnName string
		var keyValue interface{}
		for _, keyProp := range h.metadata.KeyProperties {
			keyColumnName = toSnakeCase(keyProp.Name)
			keyValue = token.KeyValues[keyProp.JsonName]
			break
		}

		if keyColumnName != "" && keyValue != nil {
			db = db.Where(fmt.Sprintf("%s > ?", keyColumnName), keyValue)
		}
	}

	return db
}

// validateSkipToken validates that a skiptoken can be decoded properly
func (h *EntityHandler) validateSkipToken(queryOptions *query.QueryOptions) error {
	if queryOptions.SkipToken == nil {
		return nil
	}

	// Try to decode the skiptoken
	_, err := skiptoken.Decode(*queryOptions.SkipToken)
	if err != nil {
		return fmt.Errorf("invalid skiptoken: %w", err)
	}

	return nil
}

// validateComplexTypeUsage validates that complex types are not used in unsupported operations
func (h *EntityHandler) validateComplexTypeUsage(queryOptions *query.QueryOptions) error {
	// Build a map of computed property aliases
	computedAliases := make(map[string]bool)
	if queryOptions.Compute != nil {
		for _, expr := range queryOptions.Compute.Expressions {
			computedAliases[expr.Alias] = true
		}
	}

	// Check filter for complex type usage
	if queryOptions.Filter != nil {
		if err := h.validateFilterForComplexTypes(queryOptions.Filter, false, computedAliases); err != nil {
			return err
		}
	}

	// Check orderby for complex type usage
	for _, orderBy := range queryOptions.OrderBy {
		// Skip validation for computed properties
		if computedAliases[orderBy.Property] {
			continue
		}

		prop, _, err := h.metadata.ResolvePropertyPath(orderBy.Property)
		if err != nil {
			return fmt.Errorf("property path '%s' is not supported", orderBy.Property)
		}
		if prop.IsNavigationProp {
			return fmt.Errorf("ordering by navigation property '%s' is not supported", orderBy.Property)
		}
		if prop.IsComplexType {
			return fmt.Errorf("ordering by complex type property '%s' is not supported", orderBy.Property)
		}
	}

	return nil
}

// validateFilterForComplexTypes recursively validates a filter expression for complex type usage
// The insideLambda parameter indicates if we're validating properties inside a lambda predicate
// The computedAliases parameter contains aliases of computed properties that should be skipped
func (h *EntityHandler) validateFilterForComplexTypes(filter *query.FilterExpression, insideLambda bool, computedAliases map[string]bool) error {
	if filter == nil {
		return nil
	}

	// Skip property validation if we're inside a lambda predicate
	// Properties inside lambda predicates refer to the related entity, not the current entity
	if !insideLambda && filter.Property != "" && !strings.HasPrefix(filter.Property, "_") {
		// Allow $it (current instance reference) - used in isof() per OData v4 spec 5.1.1.11.4
		// $it can appear when isof() is used with a single argument to check entity type
		if filter.Property == "$it" {
			// $it is valid when used with isof operator or when part of a comparison involving isof
			if filter.Operator != query.OpIsOf && filter.Operator != query.OpEqual && filter.Operator != query.OpNotEqual {
				return fmt.Errorf("property path '$it' can only be used with isof() function")
			}
			// No further validation needed for $it
			goto validateChildren
		}

		// Skip validation for computed properties
		if computedAliases[filter.Property] {
			goto validateChildren
		}

		// Allow lambda operators (any/all) on navigation properties - OData v4 spec 5.1.1.10
		if filter.Operator == query.OpAny || filter.Operator == query.OpAll {
			// For lambda operators, the property is the navigation property
			// The predicate is stored in filter.Left
			prop, _, err := h.metadata.ResolvePropertyPath(filter.Property)
			if err != nil {
				return fmt.Errorf("property path '%s' is not supported", filter.Property)
			}
			if !prop.IsNavigationProp {
				return fmt.Errorf("lambda operator '%s' can only be used with navigation properties", filter.Operator)
			}
			// Lambda operators on navigation properties are valid
			// Note: We don't validate filter.Left here because it refers to properties
			// of the related entity, not the current entity. The properties in the predicate
			// will be validated at SQL generation time against the related entity's metadata.
			goto validateChildren
		}

		prop, _, err := h.metadata.ResolvePropertyPath(filter.Property)
		if err != nil {
			return fmt.Errorf("property path '%s' is not supported", filter.Property)
		}
		if prop.IsNavigationProp {
			return fmt.Errorf("filtering by navigation property '%s' is not supported (use any/all operators)", filter.Property)
		}
		if prop.IsComplexType {
			return fmt.Errorf("filtering by complex type property '%s' is not supported", filter.Property)
		}
	}

validateChildren:
	// When we encounter a lambda operator, mark that its children (the predicate) are inside a lambda
	isLambda := filter.Operator == query.OpAny || filter.Operator == query.OpAll

	if filter.Left != nil {
		// If this filter is a lambda, its Left contains the predicate for the related entity
		if err := h.validateFilterForComplexTypes(filter.Left, insideLambda || isLambda, computedAliases); err != nil {
			return err
		}
	}

	if filter.Right != nil {
		if err := h.validateFilterForComplexTypes(filter.Right, insideLambda, computedAliases); err != nil {
			return err
		}
	}

	return nil
}
