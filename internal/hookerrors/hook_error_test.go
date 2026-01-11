package hookerrors

import (
	"errors"
	"testing"
)

func TestHookError_Error(t *testing.T) {
	tests := []struct {
		name     string
		hookErr  *HookError
		expected string
	}{
		{
			name: "Message only",
			hookErr: &HookError{
				Message: "Access denied",
			},
			expected: "Access denied",
		},
		{
			name: "Message with wrapped error",
			hookErr: &HookError{
				Message: "Database error",
				Err:     errors.New("connection timeout"),
			},
			expected: "Database error: connection timeout",
		},
		{
			name: "Wrapped error only",
			hookErr: &HookError{
				Err: errors.New("validation failed"),
			},
			expected: "validation failed",
		},
		{
			name:     "No message or error",
			hookErr:  &HookError{},
			expected: "hook error",
		},
		{
			name: "With status code",
			hookErr: &HookError{
				StatusCode: 403,
				Message:    "Forbidden",
			},
			expected: "Forbidden",
		},
		{
			name: "Empty message with status code and wrapped error",
			hookErr: &HookError{
				StatusCode: 500,
				Err:        errors.New("internal error"),
			},
			expected: "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.hookErr.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHookError_Unwrap(t *testing.T) {
	tests := []struct {
		name     string
		hookErr  *HookError
		expected error
	}{
		{
			name: "Has wrapped error",
			hookErr: &HookError{
				Message: "Outer error",
				Err:     errors.New("inner error"),
			},
			expected: errors.New("inner error"),
		},
		{
			name: "No wrapped error",
			hookErr: &HookError{
				Message: "Only message",
			},
			expected: nil,
		},
		{
			name:     "Empty HookError",
			hookErr:  &HookError{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.hookErr.Unwrap()
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Unwrap() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Error("Unwrap() = nil, want error")
				} else if result.Error() != tt.expected.Error() {
					t.Errorf("Unwrap() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestHookError_StatusCode(t *testing.T) {
	tests := []struct {
		name             string
		statusCode       int
		expectedOverride int
	}{
		{"Default 403", 0, 0},
		{"Custom 401", 401, 401},
		{"Custom 404", 404, 404},
		{"Custom 500", 500, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hookErr := &HookError{
				StatusCode: tt.statusCode,
				Message:    "Test error",
			}

			if hookErr.StatusCode != tt.expectedOverride {
				t.Errorf("StatusCode = %d, want %d", hookErr.StatusCode, tt.expectedOverride)
			}
		})
	}
}

func TestHookError_AsError(t *testing.T) {
	// Test that HookError implements error interface
	var err error
	hookErr := &HookError{
		StatusCode: 403,
		Message:    "Forbidden",
	}
	err = hookErr

	if err == nil {
		t.Error("HookError should implement error interface")
	}

	if err.Error() != "Forbidden" {
		t.Errorf("Error() = %q, want 'Forbidden'", err.Error())
	}
}

func TestHookError_ErrorsIs(t *testing.T) {
	innerErr := errors.New("inner error")
	hookErr := &HookError{
		Message: "Outer error",
		Err:     innerErr,
	}

	// Test that errors.Is can unwrap HookError
	if !errors.Is(hookErr, innerErr) {
		t.Error("errors.Is should find the wrapped error")
	}

	otherErr := errors.New("other error")
	if errors.Is(hookErr, otherErr) {
		t.Error("errors.Is should not match unrelated error")
	}
}

func TestHookError_Combinations(t *testing.T) {
	t.Run("Full configuration", func(t *testing.T) {
		innerErr := errors.New("database connection failed")
		hookErr := &HookError{
			StatusCode: 503,
			Message:    "Service unavailable",
			Err:        innerErr,
		}

		if hookErr.StatusCode != 503 {
			t.Errorf("StatusCode = %d, want 503", hookErr.StatusCode)
		}

		expectedMsg := "Service unavailable: database connection failed"
		if hookErr.Error() != expectedMsg {
			t.Errorf("Error() = %q, want %q", hookErr.Error(), expectedMsg)
		}

		if hookErr.Unwrap() != innerErr {
			t.Error("Unwrap() should return the inner error")
		}
	})

	t.Run("Only status code", func(t *testing.T) {
		hookErr := &HookError{
			StatusCode: 401,
		}

		if hookErr.StatusCode != 401 {
			t.Errorf("StatusCode = %d, want 401", hookErr.StatusCode)
		}

		if hookErr.Error() != "hook error" {
			t.Errorf("Error() = %q, want 'hook error'", hookErr.Error())
		}
	})
}
