package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/nlstn/go-odata/internal/hookerrors"
)

func TestExtractHookErrorDetails(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		defaultStatus int
		defaultCode   string
		wantIsHookErr bool
		wantStatus    int
		wantMessage   string
		wantDetails   string
	}{
		{
			name:          "Regular error, not a HookError",
			err:           errors.New("regular error"),
			defaultStatus: http.StatusInternalServerError,
			defaultCode:   "InternalError",
			wantIsHookErr: false,
			wantStatus:    http.StatusInternalServerError,
			wantMessage:   "InternalError",
			wantDetails:   "regular error",
		},
		{
			name: "HookError with status code and message",
			err: &hookerrors.HookError{
				StatusCode: http.StatusForbidden,
				Message:    "Access denied",
			},
			defaultStatus: http.StatusInternalServerError,
			defaultCode:   "InternalError",
			wantIsHookErr: true,
			wantStatus:    http.StatusForbidden,
			wantMessage:   "Access denied",
			wantDetails:   "",
		},
		{
			name: "HookError with wrapped error",
			err: &hookerrors.HookError{
				StatusCode: http.StatusUnauthorized,
				Message:    "Authentication failed",
				Err:        errors.New("invalid token"),
			},
			defaultStatus: http.StatusInternalServerError,
			defaultCode:   "InternalError",
			wantIsHookErr: true,
			wantStatus:    http.StatusUnauthorized,
			wantMessage:   "Authentication failed",
			wantDetails:   "invalid token",
		},
		{
			name: "HookError with zero status code uses default",
			err: &hookerrors.HookError{
				StatusCode: 0,
				Message:    "Custom message",
			},
			defaultStatus: http.StatusBadRequest,
			defaultCode:   "BadRequest",
			wantIsHookErr: true,
			wantStatus:    http.StatusBadRequest,
			wantMessage:   "Custom message",
			wantDetails:   "",
		},
		{
			name: "HookError with empty message uses default code",
			err: &hookerrors.HookError{
				StatusCode: http.StatusNotFound,
				Message:    "",
			},
			defaultStatus: http.StatusInternalServerError,
			defaultCode:   "NotFound",
			wantIsHookErr: true,
			wantStatus:    http.StatusNotFound,
			wantMessage:   "NotFound",
			wantDetails:   "",
		},
		{
			name: "HookError with all defaults",
			err: &hookerrors.HookError{
				StatusCode: 0,
				Message:    "",
			},
			defaultStatus: http.StatusForbidden,
			defaultCode:   "Forbidden",
			wantIsHookErr: true,
			wantStatus:    http.StatusForbidden,
			wantMessage:   "Forbidden",
			wantDetails:   "",
		},
		{
			name: "HookError with only wrapped error",
			err: &hookerrors.HookError{
				Err: errors.New("validation error"),
			},
			defaultStatus: http.StatusBadRequest,
			defaultCode:   "ValidationFailed",
			wantIsHookErr: true,
			wantStatus:    http.StatusBadRequest,
			wantMessage:   "ValidationFailed",
			wantDetails:   "validation error",
		},
		{
			name: "Wrapped HookError",
			err: func() error {
				innerErr := &hookerrors.HookError{
					StatusCode: http.StatusConflict,
					Message:    "Conflict occurred",
				}
				// Return it wrapped in another error
				return errors.Join(errors.New("outer error"), innerErr)
			}(),
			defaultStatus: http.StatusInternalServerError,
			defaultCode:   "InternalError",
			wantIsHookErr: true,
			wantStatus:    http.StatusConflict,
			wantMessage:   "Conflict occurred",
			wantDetails:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsHookErr, gotStatus, gotMessage, gotDetails := extractHookErrorDetails(
				tt.err,
				tt.defaultStatus,
				tt.defaultCode,
			)

			if gotIsHookErr != tt.wantIsHookErr {
				t.Errorf("isHookErr = %v, want %v", gotIsHookErr, tt.wantIsHookErr)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("status = %v, want %v", gotStatus, tt.wantStatus)
			}
			if gotMessage != tt.wantMessage {
				t.Errorf("message = %q, want %q", gotMessage, tt.wantMessage)
			}
			if gotDetails != tt.wantDetails {
				t.Errorf("details = %q, want %q", gotDetails, tt.wantDetails)
			}
		})
	}
}
