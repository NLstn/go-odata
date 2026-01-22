package odata

import (
	"errors"
	"net/http"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedMatch error
	}{
		{"EntityNotFound", ErrEntityNotFound, ErrEntityNotFound},
		{"ValidationError", ErrValidationError, ErrValidationError},
		{"Unauthorized", ErrUnauthorized, ErrUnauthorized},
		{"Forbidden", ErrForbidden, ErrForbidden},
		{"MethodNotAllowed", ErrMethodNotAllowed, ErrMethodNotAllowed},
		{"Conflict", ErrConflict, ErrConflict},
		{"PreconditionFailed", ErrPreconditionFailed, ErrPreconditionFailed},
		{"UnsupportedMediaType", ErrUnsupportedMediaType, ErrUnsupportedMediaType},
		{"InternalServerError", ErrInternalServerError, ErrInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.expectedMatch) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.err, tt.expectedMatch)
			}
		})
	}
}

func TestODataError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ODataError
		expected string
	}{
		{
			name: "simple error",
			err: &ODataError{
				StatusCode: http.StatusNotFound,
				Code:       ErrorCodeNotFound,
				Message:    "Entity not found",
			},
			expected: "Entity not found",
		},
		{
			name: "error with wrapped error",
			err: &ODataError{
				StatusCode: http.StatusInternalServerError,
				Code:       ErrorCodeInternalServerError,
				Message:    "Failed to process request",
				Err:        errors.New("database connection failed"),
			},
			expected: "Failed to process request: database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("ODataError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestODataError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	odataErr := &ODataError{
		StatusCode: http.StatusInternalServerError,
		Code:       ErrorCodeInternalServerError,
		Message:    "Something went wrong",
		Err:        underlyingErr,
	}

	if !errors.Is(odataErr, underlyingErr) {
		t.Errorf("errors.Is(odataErr, underlyingErr) = false, want true")
	}

	// Test error unwrapping
	unwrapped := odataErr.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("odataErr.Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}
}

func TestMapErrorToHTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{"nil error", nil, http.StatusOK},
		{"EntityNotFound", ErrEntityNotFound, http.StatusNotFound},
		{"ValidationError", ErrValidationError, http.StatusBadRequest},
		{"Unauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"Forbidden", ErrForbidden, http.StatusForbidden},
		{"MethodNotAllowed", ErrMethodNotAllowed, http.StatusMethodNotAllowed},
		{"Conflict", ErrConflict, http.StatusConflict},
		{"PreconditionFailed", ErrPreconditionFailed, http.StatusPreconditionFailed},
		{"UnsupportedMediaType", ErrUnsupportedMediaType, http.StatusUnsupportedMediaType},
		{"InternalServerError", ErrInternalServerError, http.StatusInternalServerError},
		{
			"ODataError",
			&ODataError{StatusCode: http.StatusNotFound, Code: ErrorCodeNotFound, Message: "Not found"},
			http.StatusNotFound,
		},
		{
			"HookError",
			&HookError{StatusCode: http.StatusBadRequest, Message: "Bad request"},
			http.StatusBadRequest,
		},
		{
			"unknown error",
			errors.New("some random error"),
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapErrorToHTTPStatus(tt.err)
			if got != tt.expectedStatus {
				t.Errorf("MapErrorToHTTPStatus(%v) = %d, want %d", tt.err, got, tt.expectedStatus)
			}
		})
	}
}

func TestMapErrorToHTTPStatus_WrappedSentinelError(t *testing.T) {
	// Test that wrapped sentinel errors are correctly identified
	wrappedErr := errors.New("additional context: " + ErrEntityNotFound.Error())
	// This won't match because it's not properly wrapped, but let's test proper wrapping
	properlyWrappedErr := &ODataError{
		StatusCode: http.StatusNotFound,
		Code:       ErrorCodeNotFound,
		Message:    "Entity not found",
		Err:        ErrEntityNotFound,
	}

	status := MapErrorToHTTPStatus(properlyWrappedErr)
	if status != http.StatusNotFound {
		t.Errorf("MapErrorToHTTPStatus(properlyWrappedErr) = %d, want %d", status, http.StatusNotFound)
	}

	// The non-properly wrapped error should default to 500
	status = MapErrorToHTTPStatus(wrappedErr)
	if status != http.StatusInternalServerError {
		t.Errorf("MapErrorToHTTPStatus(wrappedErr) = %d, want %d", status, http.StatusInternalServerError)
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"validation error", ErrValidationError, true},
		{"wrapped validation error", &ODataError{Err: ErrValidationError}, true},
		{"not found error", ErrEntityNotFound, false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidationError(tt.err)
			if got != tt.expected {
				t.Errorf("IsValidationError(%v) = %t, want %t", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"not found error", ErrEntityNotFound, true},
		{"wrapped not found error", &ODataError{Err: ErrEntityNotFound}, true},
		{"validation error", ErrValidationError, false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFoundError(tt.err)
			if got != tt.expected {
				t.Errorf("IsNotFoundError(%v) = %t, want %t", tt.err, got, tt.expected)
			}
		})
	}
}

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrorCodeGeneral, "General"},
		{ErrorCodeNotFound, "NotFound"},
		{ErrorCodeBadRequest, "BadRequest"},
		{ErrorCodeUnauthorized, "Unauthorized"},
		{ErrorCodeForbidden, "Forbidden"},
		{ErrorCodeMethodNotAllowed, "MethodNotAllowed"},
		{ErrorCodeConflict, "Conflict"},
		{ErrorCodePreconditionFailed, "PreconditionFailed"},
		{ErrorCodeUnsupportedMediaType, "UnsupportedMediaType"},
		{ErrorCodeInternalServerError, "InternalServerError"},
		{ErrorCodeNotImplemented, "NotImplemented"},
		{ErrorCodeServiceUnavailable, "ServiceUnavailable"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			got := string(tt.code)
			if got != tt.expected {
				t.Errorf("ErrorCode string = %q, want %q", got, tt.expected)
			}
		})
	}
}
