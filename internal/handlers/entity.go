package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
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
		response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for entity collections", r.Method))
		return
	}

	// Create a slice to hold the results
	sliceType := reflect.SliceOf(h.metadata.EntityType)
	results := reflect.New(sliceType).Interface()

	// Execute the database query
	if err := h.db.Find(results).Error; err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error())
		return
	}

	// Get the actual slice value (results is a pointer to slice)
	sliceValue := reflect.ValueOf(results).Elem().Interface()

	// Write the OData response
	if err := response.WriteODataCollection(w, r, h.metadata.EntitySetName, sliceValue); err != nil {
		// If we can't write the response, log the error but don't try to write another response
		fmt.Printf("Error writing OData response: %v\n", err)
	}
}

// HandleEntity handles GET requests for individual entities
func (h *EntityHandler) HandleEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for individual entities", r.Method))
		return
	}

	// Create an instance to hold the result
	result := reflect.New(h.metadata.EntityType).Interface()

	// Build the query condition using the key property
	keyField := h.metadata.KeyProperty.JsonName
	if err := h.db.Where(fmt.Sprintf("%s = ?", keyField), entityKey).First(result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.WriteError(w, http.StatusNotFound, "Entity not found",
				fmt.Sprintf("Entity with key '%s' not found", entityKey))
		} else {
			response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error())
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
