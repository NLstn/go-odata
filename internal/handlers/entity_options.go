package handlers

import "net/http"

// handleOptionsEntity handles OPTIONS requests for individual entities
func (h *EntityHandler) handleOptionsEntity(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, DELETE, PATCH, PUT, OPTIONS")
	w.WriteHeader(http.StatusOK)
}
