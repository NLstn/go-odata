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

	// Write the OData response
	if err := response.WriteODataCollection(w, r, h.metadata.EntitySetName, sliceValue, totalCount, nextLink); err != nil {
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
			fieldName := query.GetPropertyFieldName(item.Property, h.metadata)
			direction := "ASC"
			if item.Descending {
				direction = "DESC"
			}
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

	// Create an instance to hold the result
	result := reflect.New(h.metadata.EntityType).Interface()

	// Build the query condition using the key property
	keyField := h.metadata.KeyProperty.JsonName
	if err := h.db.Where(fmt.Sprintf("%s = ?", keyField), entityKey).First(result).Error; err != nil {
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

	odataResponse := map[string]interface{}{
		"@odata.context": contextURL,
	}

	// Merge the entity fields into the response
	entityValue := reflect.ValueOf(result).Elem()
	entityType := entityValue.Type()

	for i := 0; i < entityValue.NumField(); i++ {
		field := entityType.Field(i)
		if field.IsExported() {
			fieldValue := entityValue.Field(i)
			jsonName := getJsonName(field)
			odataResponse[jsonName] = fieldValue.Interface()
		}
	}

	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	// Write the response
	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		fmt.Printf("Error writing entity response: %v\n", err)
	}
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
