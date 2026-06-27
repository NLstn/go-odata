package handlers

import (
	"log/slog"
	"net/http"

	"github.com/nlstn/go-odata/internal/response"
)

// WriteError writes an OData error response and logs if the write fails.
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	if err := response.WriteError(w, r, status, code, detail); err != nil {
		slog.Default().Error("Error writing error response", "error", err)
	}
}

// WriteMethodNotAllowed writes a 405 response with the Allow header and logs if the write fails.
func WriteMethodNotAllowed(w http.ResponseWriter, r *http.Request, allow, code, detail string) {
	if err := response.WriteMethodNotAllowed(w, r, allow, code, detail); err != nil {
		slog.Default().Error("Error writing error response", "error", err)
	}
}
