package handlers

import (
	"errors"
	"net/http"

	"github.com/nlstn/go-odata/internal/hookerrors"
	"github.com/nlstn/go-odata/internal/response"
)

// extractHookErrorDetails checks if an error is a HookError and extracts the status code,
// message, and details. If it's a HookError, returns true with the extracted values.
// Otherwise returns false with default values.
func extractHookErrorDetails(err error, defaultStatus int, defaultCode string) (isHookErr bool, status int, message string, details string) {
	var hookErr *hookerrors.HookError
	if !errors.As(err, &hookErr) {
		return false, defaultStatus, defaultCode, err.Error()
	}

	status = hookErr.StatusCode
	if status == 0 {
		status = defaultStatus
	}

	message = hookErr.Message
	if message == "" {
		message = defaultCode
	}

	if hookErr.Err != nil {
		details = hookErr.Err.Error()
	}

	return true, status, message, details
}

// writeHookError writes a hook error response, checking for custom HookError types
// with custom status codes. If the error is a HookError with a StatusCode set,
// uses that status code; otherwise falls back to the defaultStatus.
//
//nolint:unparam // defaultStatus is kept as parameter for potential future use with different defaults
func (h *EntityHandler) writeHookError(w http.ResponseWriter, r *http.Request, err error, defaultStatus int, defaultCode string) {
	if err == nil {
		return
	}

	status, odataErr := response.BuildODataErrorResponse(err, defaultStatus, defaultCode)

	if writeErr := response.WriteODataError(w, r, status, odataErr); writeErr != nil {
		h.logger.Error("Error writing error response", "error", writeErr)
	}
}
