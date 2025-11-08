package handlers

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// HandleMediaEntityValue handles GET, PUT, and OPTIONS requests for media entity binary content (e.g., MediaItems(1)/$value)
func (h *EntityHandler) HandleMediaEntityValue(w http.ResponseWriter, r *http.Request, entityKey string) {
	// Check if this is actually a media entity
	if !h.metadata.HasStream {
		if err := response.WriteError(w, http.StatusBadRequest, "Not a media entity",
			fmt.Sprintf("%s is not a media entity. Only media entities support /$value access", h.metadata.EntitySetName)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetMediaEntityValue(w, r, entityKey)
	case http.MethodPut:
		h.handlePutMediaEntityValue(w, r, entityKey)
	case http.MethodOptions:
		h.handleOptionsMediaEntityValue(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for media entity $value", r.Method)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
	}
}

// handleGetMediaEntityValue handles GET requests for media entity binary content
func (h *EntityHandler) handleGetMediaEntityValue(w http.ResponseWriter, r *http.Request, entityKey string) {
	// Fetch the entity
	entity := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(h.db, entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	if err := db.First(entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if writeErr := response.WriteError(w, http.StatusNotFound, ErrMsgEntityNotFound,
				fmt.Sprintf("Entity with key %s not found", entityKey)); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
		} else {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
				fmt.Sprintf("Database error: %v", err)); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
		}
		return
	}

	// Get media content using GetMediaContent method
	var content []byte
	var contentType string

	entityValue := reflect.ValueOf(entity)
	if method := entityValue.MethodByName("GetMediaContent"); method.IsValid() {
		results := method.Call(nil)
		if len(results) > 0 && results[0].Kind() == reflect.Slice {
			content = results[0].Bytes()
		}
	}

	if method := entityValue.MethodByName("GetMediaContentType"); method.IsValid() {
		results := method.Call(nil)
		if len(results) > 0 && results[0].Kind() == reflect.String {
			contentType = results[0].String()
		}
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Write binary response
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		if _, err := w.Write(content); err != nil {
			h.logger.Error("Error writing media content", "error", err)
		}
	}
}

// handlePutMediaEntityValue handles PUT requests to update media entity binary content
func (h *EntityHandler) handlePutMediaEntityValue(w http.ResponseWriter, r *http.Request, entityKey string) {
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

	if err := db.First(entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if writeErr := response.WriteError(w, http.StatusNotFound, ErrMsgEntityNotFound,
				fmt.Sprintf("Entity with key %s not found", entityKey)); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
		} else {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
				fmt.Sprintf("Database error: %v", err)); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
		}
		return
	}

	// Update media content using SetMediaContent and SetMediaContentType methods
	entityValue := reflect.ValueOf(entity)

	if method := entityValue.MethodByName("SetMediaContent"); method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(content)})
	}

	if method := entityValue.MethodByName("SetMediaContentType"); method.IsValid() {
		method.Call([]reflect.Value{reflect.ValueOf(contentType)})
	}

	// Save the entity
	if err := h.db.Save(entity).Error; err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to update media entity: %v", err)); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleOptionsMediaEntityValue handles OPTIONS requests for media entity $value
func (h *EntityHandler) handleOptionsMediaEntityValue(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, PUT, OPTIONS")
	w.WriteHeader(http.StatusOK)
}
