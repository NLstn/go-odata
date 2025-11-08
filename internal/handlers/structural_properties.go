package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// HandleStructuralProperty handles GET and OPTIONS requests for structural properties (e.g., Products(1)/Name)
// When isValue is true, returns the raw property value without JSON wrapper (e.g., Products(1)/Name/$value)
func (h *EntityHandler) HandleStructuralProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertyName string, isValue bool) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetStructuralProperty(w, r, entityKey, propertyName, isValue)
	case http.MethodOptions:
		h.handleOptionsStructuralProperty(w)
	default:
		h.writeMethodNotAllowedError(w, r.Method, "property access")
	}
}

// handleGetStructuralProperty handles GET requests for structural properties
func (h *EntityHandler) handleGetStructuralProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertyName string, isValue bool) {
	// Find and validate the structural property
	prop := h.findStructuralProperty(propertyName)
	if prop == nil {
		h.writePropertyNotFoundError(w, propertyName)
		return
	}

	// Fetch property value
	fieldValue, err := h.fetchPropertyValue(w, entityKey, prop)
	if err != nil {
		return // Error already written
	}

	// Write response
	if isValue {
		h.writeRawPropertyValue(w, r, prop, fieldValue)
	} else {
		h.writePropertyResponse(w, r, entityKey, prop, fieldValue)
	}
}

// handleOptionsStructuralProperty handles OPTIONS requests for structural properties
func (h *EntityHandler) handleOptionsStructuralProperty(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

// fetchPropertyValue fetches a property value from an entity
func (h *EntityHandler) fetchPropertyValue(w http.ResponseWriter, entityKey string, prop *metadata.PropertyMetadata) (reflect.Value, error) {
	entity := reflect.New(h.metadata.EntityType).Interface()

	var db *gorm.DB
	var err error

	// Handle singleton case where entityKey is empty
	if h.metadata.IsSingleton && entityKey == "" {
		// For singletons, we don't use a key query, just fetch the first (and only) record
		db = h.db
	} else {
		// For regular entities, build the key query
		db, err = h.buildKeyQuery(h.db, entityKey)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return reflect.Value{}, err
		}
	}

	db = h.applyStructuralPropertySelect(db, prop)

	if err := db.First(entity).Error; err != nil {
		h.handlePropertyFetchError(w, err, entityKey)
		return reflect.Value{}, err
	}

	entityValue := reflect.ValueOf(entity).Elem()
	fieldValue := entityValue.FieldByName(prop.Name)
	if !fieldValue.IsValid() {
		if err := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			"Could not access property"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return reflect.Value{}, fmt.Errorf("invalid field")
	}

	return fieldValue, nil
}

// handlePropertyFetchError handles errors when fetching a property
func (h *EntityHandler) handlePropertyFetchError(w http.ResponseWriter, err error, entityKey string) {
	if err == gorm.ErrRecordNotFound {
		var errorMessage string
		if h.metadata.IsSingleton {
			errorMessage = fmt.Sprintf("Singleton '%s' not found", h.metadata.SingletonName)
		} else {
			errorMessage = fmt.Sprintf("Entity with key '%s' not found", entityKey)
		}

		if writeErr := response.WriteError(w, http.StatusNotFound, ErrMsgEntityNotFound, errorMessage); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
	} else {
		h.writeDatabaseError(w, err)
	}
}

// writePropertyResponse writes a property response with OData context
func (h *EntityHandler) writePropertyResponse(w http.ResponseWriter, r *http.Request, entityKey string, prop *metadata.PropertyMetadata, fieldValue reflect.Value) {
	// Get metadata level to determine which fields to include
	metadataLevel := response.GetODataMetadataLevel(r)

	odataResponse := map[string]interface{}{
		"value": fieldValue.Interface(),
	}

	// Only include @odata.context for minimal and full metadata (not for none)
	if metadataLevel != "none" {
		contextURL := fmt.Sprintf("%s/$metadata#%s(%s)/%s", response.BuildBaseURL(r), h.metadata.EntitySetName, entityKey, prop.JsonName)
		odataResponse[ODataContextProperty] = contextURL
	}

	// Set Content-Type with dynamic metadata level
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		h.logger.Error("Error writing property response", "error", err)
	}
}

// writeRawPropertyValue writes a property value in raw format for /$value requests
func (h *EntityHandler) writeRawPropertyValue(w http.ResponseWriter, r *http.Request, prop *metadata.PropertyMetadata, fieldValue reflect.Value) {
	// Set appropriate content type based on the value type
	valueInterface := fieldValue.Interface()

	// Check for binary data ([]byte) first
	if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.Uint8 {
		// Binary data - set appropriate content type and write raw bytes
		// Use custom content type if specified, otherwise default to application/octet-stream
		contentType := "application/octet-stream"
		if prop.ContentType != "" {
			contentType = prop.ContentType
		}
		w.Header().Set(HeaderContentType, contentType)
		w.WriteHeader(http.StatusOK)

		// For HEAD requests, don't write the body
		if r.Method == http.MethodHead {
			return
		}

		// Write raw binary data
		if byteData, ok := valueInterface.([]byte); ok {
			if _, err := w.Write(byteData); err != nil {
				h.logger.Error("Error writing binary value", "error", err)
			}
		}
		return
	}

	// Determine content type based on the property type
	switch fieldValue.Kind() {
	case reflect.String:
		w.Header().Set(HeaderContentType, ContentTypePlainText)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		w.Header().Set(HeaderContentType, ContentTypePlainText)
	case reflect.Bool:
		w.Header().Set(HeaderContentType, ContentTypePlainText)
	default:
		// For other types, use application/octet-stream
		w.Header().Set(HeaderContentType, "application/octet-stream")
	}

	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

	// Write the raw value
	if _, err := fmt.Fprintf(w, "%v", valueInterface); err != nil {
		h.logger.Error("Error writing raw value", "error", err)
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
