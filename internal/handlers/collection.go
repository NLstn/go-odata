package handlers

import (
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/query"
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
		WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for entity collections", r.Method))
	}
}

// handleOptionsCollection handles OPTIONS requests for entity collections
func (h *EntityHandler) handleOptionsCollection(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, POST, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

// HandleCount handles GET, HEAD, and OPTIONS requests for entity collection count (e.g., /Products/$count)
func (h *EntityHandler) HandleCount(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetCount(w, r)
	case http.MethodOptions:
		h.handleOptionsCount(w)
	default:
		WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for $count", r.Method))
	}
}

// handleGetCount handles GET requests for entity collection count
func (h *EntityHandler) handleGetCount(w http.ResponseWriter, r *http.Request) {
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error())
		return
	}

	scopes, hookErr := callBeforeReadCollection(h.metadata, r, queryOptions)
	if hookErr != nil {
		WriteError(w, http.StatusForbidden, "Authorization failed", hookErr.Error())
		return
	}

	count, countErr := h.countEntities(queryOptions, scopes)
	if countErr != nil {
		WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, countErr.Error())
		return
	}

	w.Header().Set(HeaderContentType, "text/plain")
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	if _, writeErr := fmt.Fprintf(w, "%d", count); writeErr != nil {
		fmt.Printf("Error writing count response: %v\n", writeErr)
	}
}

// handleOptionsCount handles OPTIONS requests for $count endpoint
func (h *EntityHandler) handleOptionsCount(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}
