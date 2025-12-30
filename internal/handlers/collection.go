package handlers

import (
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/query"
)

// HandleCollection handles GET, HEAD, POST, and OPTIONS requests for entity collections
func (h *EntityHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	// Check if the method is disabled
	// Map HEAD to GET for method checking since HEAD is semantically equivalent to GET
	methodToCheck := r.Method
	if r.Method == http.MethodHead {
		methodToCheck = http.MethodGet
	}
	if h.isMethodDisabled(methodToCheck) {
		WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not allowed for this entity", r.Method))
		return
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, "", nil), auth.OperationQuery, h.logger) {
			return
		}
		h.handleGetCollection(w, r)
	case http.MethodPost:
		h.handlePostEntity(w, r)
	case http.MethodOptions:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, "", nil), auth.OperationRead, h.logger) {
			return
		}
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
	// Check if GET method is disabled (applies to both GET and HEAD)
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		if h.isMethodDisabled(http.MethodGet) {
			WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
				fmt.Sprintf("Method %s is not allowed for this entity", r.Method))
			return
		}
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, "", []string{"$count"}), auth.OperationQuery, h.logger) {
			return
		}
		h.handleGetCount(w, r)
	case http.MethodOptions:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, "", []string{"$count"}), auth.OperationRead, h.logger) {
			return
		}
		h.handleOptionsCount(w)
	default:
		WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for $count", r.Method))
	}
}

// handleGetCount handles GET requests for entity collection count
func (h *EntityHandler) handleGetCount(w http.ResponseWriter, r *http.Request) {
	// Check if there's an overwrite handler
	if h.overwrite.hasGetCount() {
		h.handleGetCountOverwrite(w, r)
		return
	}

	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error())
		return
	}

	if err := applyPolicyFilter(r, h.policy, buildEntityResourceDescriptor(h.metadata, "", []string{"$count"}), queryOptions); err != nil {
		WriteError(w, http.StatusForbidden, "Authorization failed", err.Error())
		return
	}

	scopes, hookErr := callBeforeReadCollection(h.metadata, r, queryOptions)
	if hookErr != nil {
		WriteError(w, http.StatusForbidden, "Authorization failed", hookErr.Error())
		return
	}

	count, countErr := h.countEntities(r.Context(), queryOptions, scopes)
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
		h.logger.Error("Error writing count response", "error", writeErr)
	}
}

// handleGetCountOverwrite handles GET count requests using the overwrite handler
func (h *EntityHandler) handleGetCountOverwrite(w http.ResponseWriter, r *http.Request) {
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error())
		return
	}

	if err := applyPolicyFilter(r, h.policy, buildEntityResourceDescriptor(h.metadata, "", []string{"$count"}), queryOptions); err != nil {
		WriteError(w, http.StatusForbidden, "Authorization failed", err.Error())
		return
	}

	// Create overwrite context
	ctx := &OverwriteContext{
		QueryOptions: queryOptions,
		Request:      r,
	}

	// Call the overwrite handler
	count, err := h.overwrite.getCount(ctx)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Error getting count", err.Error())
		return
	}

	w.Header().Set(HeaderContentType, "text/plain")
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	if _, writeErr := fmt.Fprintf(w, "%d", count); writeErr != nil {
		h.logger.Error("Error writing count response", "error", writeErr)
	}
}

// handleOptionsCount handles OPTIONS requests for $count endpoint
func (h *EntityHandler) handleOptionsCount(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}
