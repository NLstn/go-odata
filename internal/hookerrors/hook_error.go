package hookerrors

import "fmt"

// HookError is an error type that can be returned from lifecycle hooks
// to specify a custom HTTP status code and error message.
type HookError struct {
	// StatusCode is the HTTP status code to return (e.g., 401, 403, 404).
	// If not specified, defaults to 403 Forbidden.
	StatusCode int

	// Message is the error message to return in the response.
	Message string

	// Err is an optional wrapped error for additional context.
	Err error
}

// Error implements the error interface.
func (e *HookError) Error() string {
	if e.Message != "" {
		if e.Err != nil {
			return fmt.Sprintf("%s: %v", e.Message, e.Err)
		}
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "hook error"
}

// Unwrap returns the wrapped error, if any.
func (e *HookError) Unwrap() error {
	return e.Err
}
