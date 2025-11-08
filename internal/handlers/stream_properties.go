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

// HandleStreamProperty handles GET, PUT, and OPTIONS requests for stream properties (e.g., Products(1)/Photo)
// When isValue is true, returns the binary stream content (e.g., Products(1)/Photo/$value)
func (h *EntityHandler) HandleStreamProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertyName string, isValue bool) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetStreamProperty(w, r, entityKey, propertyName, isValue)
	case http.MethodPut:
		if isValue {
			h.handlePutStreamProperty(w, r, entityKey, propertyName)
		} else {
			if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
				"PUT is only supported on stream properties with /$value"); err != nil {
				h.logger.Error("Error writing error response", "error", err)
			}
		}
	case http.MethodOptions:
		h.handleOptionsStreamProperty(w, isValue)
	default:
		h.writeMethodNotAllowedError(w, r.Method, "stream property access")
	}
}

// handleGetStreamProperty handles GET requests for stream properties
func (h *EntityHandler) handleGetStreamProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertyName string, isValue bool) {
	// Find and validate the stream property
	prop := h.findStreamProperty(propertyName)
	if prop == nil {
		h.writePropertyNotFoundError(w, propertyName)
		return
	}

	// Fetch the entity
	entity := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(h.db, entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Apply SELECT clause to fetch only the stream content fields and keys
	db = h.applyStreamPropertySelect(db, prop)

	if err := db.First(entity).Error; err != nil {
		h.handlePropertyFetchError(w, err, entityKey)
		return
	}

	// Get stream content and content type
	entityValue := reflect.ValueOf(entity).Elem()

	var content []byte
	var contentType string

	// Try to get stream content using GetStreamProperty method if available
	if method := entityValue.MethodByName("GetStreamProperty"); method.IsValid() {
		results := method.Call([]reflect.Value{reflect.ValueOf(propertyName)})
		if len(results) == 3 && results[2].Bool() {
			if results[0].Kind() == reflect.Slice && results[0].Type().Elem().Kind() == reflect.Uint8 {
				content = results[0].Bytes()
			}
			if results[1].Kind() == reflect.String {
				contentType = results[1].String()
			}
		}
	} else {
		// Fallback: read from fields directly
		if prop.StreamContentField != "" {
			contentField := entityValue.FieldByName(prop.StreamContentField)
			if contentField.IsValid() && contentField.Kind() == reflect.Slice {
				content = contentField.Bytes()
			}
		}
		if prop.StreamContentTypeField != "" {
			contentTypeField := entityValue.FieldByName(prop.StreamContentTypeField)
			if contentTypeField.IsValid() && contentTypeField.Kind() == reflect.String {
				contentType = contentTypeField.String()
			}
		}
	}

	if isValue {
		// Return binary content with appropriate Content-Type
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			if _, err := w.Write(content); err != nil {
				h.logger.Error("Error writing stream content", "error", err)
			}
		}
	} else {
		// Return metadata about the stream (not the binary content)
		// Return stream property information with read/edit links
		baseURL := response.BuildBaseURL(r)
		entityURL := fmt.Sprintf("%s/%s(%s)", baseURL, h.metadata.EntitySetName, entityKey)
		streamURL := fmt.Sprintf("%s/%s", entityURL, propertyName)

		streamInfo := map[string]interface{}{
			"@odata.context":       fmt.Sprintf("%s/$metadata#%s(%s)/%s", baseURL, h.metadata.EntitySetName, entityKey, propertyName),
			"@odata.mediaReadLink": streamURL + "/$value",
		}

		if contentType != "" {
			streamInfo["@odata.mediaContentType"] = contentType
		}

		w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			if err := json.NewEncoder(w).Encode(streamInfo); err != nil {
				h.logger.Error("Error writing stream metadata", "error", err)
			}
		}
	}
}

// handlePutStreamProperty handles PUT requests to update stream property content
func (h *EntityHandler) handlePutStreamProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertyName string) {
	// Find and validate the stream property
	prop := h.findStreamProperty(propertyName)
	if prop == nil {
		h.writePropertyNotFoundError(w, propertyName)
		return
	}

	// Read binary content from request body
	content := make([]byte, 0)
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		if n > 0 {
			content = append(content, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Get content type from header
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Fetch the entity
	entity := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(h.db, entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Apply SELECT clause to fetch only the stream content fields and keys
	db = h.applyStreamPropertySelect(db, prop)

	if err := db.First(entity).Error; err != nil {
		h.handlePropertyFetchError(w, err, entityKey)
		return
	}

	// Update stream content using SetStreamProperty method if available
	entityValue := reflect.ValueOf(entity).Elem()
	if method := entityValue.MethodByName("SetStreamProperty"); method.IsValid() {
		results := method.Call([]reflect.Value{
			reflect.ValueOf(propertyName),
			reflect.ValueOf(content),
			reflect.ValueOf(contentType),
		})
		if len(results) > 0 && results[0].Kind() == reflect.Bool && !results[0].Bool() {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid stream property",
				fmt.Sprintf("Failed to set stream property '%s'", propertyName)); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
	} else {
		// Fallback: update fields directly
		if prop.StreamContentField != "" {
			contentField := entityValue.FieldByName(prop.StreamContentField)
			if contentField.IsValid() && contentField.CanSet() {
				contentField.SetBytes(content)
			}
		}
		if prop.StreamContentTypeField != "" {
			contentTypeField := entityValue.FieldByName(prop.StreamContentTypeField)
			if contentTypeField.IsValid() && contentTypeField.CanSet() {
				contentTypeField.SetString(contentType)
			}
		}
	}

	// Save the entity
	if err := h.db.Save(entity).Error; err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to update stream property: %v", err)); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleOptionsStreamProperty handles OPTIONS requests for stream properties
func (h *EntityHandler) handleOptionsStreamProperty(w http.ResponseWriter, isValue bool) {
	if isValue {
		w.Header().Set("Allow", "GET, HEAD, PUT, OPTIONS")
	} else {
		w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	}
	w.WriteHeader(http.StatusOK)
}

// applyStreamPropertySelect applies SELECT clause to fetch only the stream content fields and key columns
func (h *EntityHandler) applyStreamPropertySelect(db *gorm.DB, prop *metadata.PropertyMetadata) *gorm.DB {
	// Build select columns list: stream content fields + all key properties
	// Use struct field names - GORM will handle column name conversion
	selectColumns := make([]string, 0)

	// Add stream content field if present
	if prop.StreamContentField != "" {
		selectColumns = append(selectColumns, prop.StreamContentField)
	}

	// Add stream content type field if present
	if prop.StreamContentTypeField != "" {
		selectColumns = append(selectColumns, prop.StreamContentTypeField)
	}

	// Add all key properties
	for _, keyProp := range h.metadata.KeyProperties {
		selectColumns = append(selectColumns, keyProp.Name)
	}

	// If no select columns found, don't apply SELECT (shouldn't happen but safe fallback)
	if len(selectColumns) == 0 {
		return db
	}

	return db.Select(selectColumns)
}
