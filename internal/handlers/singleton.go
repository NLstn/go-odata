package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/response"
)

// HandleSingleton handles GET, PATCH, PUT, and OPTIONS requests for singleton entities
// Singletons are single instances of an entity type accessed directly by name (e.g., /Me)
func (h *EntityHandler) HandleSingleton(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetSingleton(w, r)
	case http.MethodPatch:
		h.handlePatchSingleton(w, r)
	case http.MethodPut:
		h.handlePutSingleton(w, r)
	case http.MethodOptions:
		h.handleOptionsSingleton(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for singleton", r.Method)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
	}
}

// handleGetSingleton handles GET requests for singleton entities
func (h *EntityHandler) handleGetSingleton(w http.ResponseWriter, r *http.Request) {
	// Create a new instance of the entity
	entityInstance := reflect.New(h.metadata.EntityType).Interface()

	// Query the database for the singleton
	// Singletons typically have a single row in the database, so we use First()
	if err := h.db.First(entityInstance).Error; err != nil {
		if err.Error() == "record not found" {
			// If no record exists for the singleton, return 404
			if writeErr := response.WriteError(w, http.StatusNotFound, ErrMsgEntityNotFound,
				fmt.Sprintf("Singleton '%s' not found", h.metadata.SingletonName)); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}
		// Other database errors
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Write the singleton entity response
	h.writeEntityResponseWithETag(w, r, entityInstance, "", http.StatusOK)
}

// handlePatchSingleton handles PATCH requests for singleton entities (partial update)
func (h *EntityHandler) handlePatchSingleton(w http.ResponseWriter, r *http.Request) {
	// Create a new instance of the entity
	entityInstance := reflect.New(h.metadata.EntityType).Interface()

	// Query the database for the existing singleton
	if err := h.db.First(entityInstance).Error; err != nil {
		if err.Error() == "record not found" {
			// If no record exists, return 404
			if writeErr := response.WriteError(w, http.StatusNotFound, ErrMsgEntityNotFound,
				fmt.Sprintf("Singleton '%s' not found", h.metadata.SingletonName)); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}
		// Other database errors
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Handle ETag if present
	if h.metadata.ETagProperty != nil {
		ifMatch := r.Header.Get(HeaderIfMatch)
		currentETag := etag.Generate(entityInstance, h.metadata)

		if !etag.Match(ifMatch, currentETag) {
			if writeErr := response.WriteError(w, http.StatusPreconditionFailed, ErrMsgPreconditionFailed,
				ErrDetailPreconditionFailed); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}
	}

	// Parse the update data from request body
	var updateData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error())); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Apply updates to the entity
	if err := h.db.Model(entityInstance).Updates(updateData).Error; err != nil {
		h.writeDatabaseError(w, err)
		return
	}

	// Reload the entity to get the updated values
	if err := h.db.First(entityInstance).Error; err != nil {
		h.writeDatabaseError(w, err)
		return
	}

	// Handle Prefer header for response
	pref := preference.ParsePrefer(r)
	h.writeUpdateResponse(w, r, pref, h.db)
}

// handlePutSingleton handles PUT requests for singleton entities (full replace)
func (h *EntityHandler) handlePutSingleton(w http.ResponseWriter, r *http.Request) {
	// Create a new instance of the entity
	existingEntity := reflect.New(h.metadata.EntityType).Interface()

	// Query the database for the existing singleton
	if err := h.db.First(existingEntity).Error; err != nil {
		if err.Error() == "record not found" {
			// If no record exists, return 404
			if writeErr := response.WriteError(w, http.StatusNotFound, ErrMsgEntityNotFound,
				fmt.Sprintf("Singleton '%s' not found", h.metadata.SingletonName)); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}
		// Other database errors
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Handle ETag if present
	if h.metadata.ETagProperty != nil {
		ifMatch := r.Header.Get(HeaderIfMatch)
		currentETag := etag.Generate(existingEntity, h.metadata)

		if !etag.Match(ifMatch, currentETag) {
			if writeErr := response.WriteError(w, http.StatusPreconditionFailed, ErrMsgPreconditionFailed,
				ErrDetailPreconditionFailed); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}
	}

	// Parse the new entity data from request body
	newEntity := reflect.New(h.metadata.EntityType).Interface()
	if err := json.NewDecoder(r.Body).Decode(newEntity); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidRequestBody,
			fmt.Sprintf(ErrDetailFailedToParseJSON, err.Error())); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Update the entity in the database (replace all fields)
	if err := h.db.Model(existingEntity).Select("*").Updates(newEntity).Error; err != nil {
		h.writeDatabaseError(w, err)
		return
	}

	// Reload the entity to get the updated values
	if err := h.db.First(existingEntity).Error; err != nil {
		h.writeDatabaseError(w, err)
		return
	}

	// Handle Prefer header for response
	pref := preference.ParsePrefer(r)
	h.writeUpdateResponse(w, r, pref, h.db)
}

// handleOptionsSingleton handles OPTIONS requests for singleton endpoint
func (h *EntityHandler) handleOptionsSingleton(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, PATCH, PUT, OPTIONS")
	w.WriteHeader(http.StatusOK)
}
