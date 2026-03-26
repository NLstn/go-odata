package hookerrors

import "fmt"

// ErrorDetail represents additional error information in a HookError.
type ErrorDetail struct {
	// Code is a service-defined error code for this detail.
	Code string

	// Target identifies the specific part of the request causing this error.
	Target string

	// Message is a human-readable description of this specific error.
	Message string
}

// HookError is an error type that can be returned from lifecycle hooks
// to specify a custom HTTP status code and error message.
type HookError struct {
	// StatusCode is the HTTP status code to return (e.g., 401, 403, 404).
	// If not specified, defaults to 403 Forbidden.
	StatusCode int

	// Code is an optional OData error code.
	// When empty, handlers fall back to using the HTTP status code as a string.
	Code string

	// Message is the error message to return in the response.
	Message string

	// Target optionally identifies the request part causing this error.
	Target string

	// Details provides additional OData error detail entries.
	Details []ErrorDetail

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
