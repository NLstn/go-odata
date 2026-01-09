package handlers

import (
	"errors"
	"net/http"

	"github.com/nlstn/go-odata/internal/hookerrors"
	"github.com/nlstn/go-odata/internal/response"
)

// writeHookError writes a hook error response, checking for custom HookError types
// with custom status codes. If the error is a HookError with a StatusCode set,
// uses that status code; otherwise falls back to the defaultStatus.
func (h *EntityHandler) writeHookError(w http.ResponseWriter, err error, defaultStatus int, defaultCode string) {
	if err == nil {
		return
	}

	var hookErr *hookerrors.HookError
	if errors.As(err, &hookErr) {
		status := hookErr.StatusCode
		if status == 0 {
			status = defaultStatus
		}
		message := hookErr.Message
		if message == "" {
			message = defaultCode
		}
		// Use the custom message as the main error message, with the error details in the details field
		details := ""
		if hookErr.Err != nil {
			details = hookErr.Err.Error()
		}
		if writeErr := response.WriteError(w, status, message, details); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Fall back to default error handling
	if writeErr := response.WriteError(w, defaultStatus, defaultCode, err.Error()); writeErr != nil {
		h.logger.Error("Error writing error response", "error", writeErr)
	}
}
