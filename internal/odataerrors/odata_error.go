package odataerrors

import "fmt"

// ErrorCode represents standard OData error codes.
// These codes provide semantic information about the error type
// and follow the OData specification conventions.
type ErrorCode string

// Standard OData error codes as defined in the OData specification.
const (
	// ErrorCodeGeneral is a general, unspecified error.
	ErrorCodeGeneral ErrorCode = "General"

	// ErrorCodeNotFound indicates the requested resource was not found.
	ErrorCodeNotFound ErrorCode = "NotFound"

	// ErrorCodeBadRequest indicates malformed or invalid request syntax.
	ErrorCodeBadRequest ErrorCode = "BadRequest"

	// ErrorCodeUnauthorized indicates missing or invalid authentication.
	ErrorCodeUnauthorized ErrorCode = "Unauthorized"

	// ErrorCodeForbidden indicates insufficient permissions.
	ErrorCodeForbidden ErrorCode = "Forbidden"

	// ErrorCodeMethodNotAllowed indicates the HTTP method is not supported.
	ErrorCodeMethodNotAllowed ErrorCode = "MethodNotAllowed"

	// ErrorCodeConflict indicates a conflict with current resource state.
	ErrorCodeConflict ErrorCode = "Conflict"

	// ErrorCodePreconditionFailed indicates an ETag precondition failed.
	ErrorCodePreconditionFailed ErrorCode = "PreconditionFailed"

	// ErrorCodeUnsupportedMediaType indicates unsupported Content-Type.
	ErrorCodeUnsupportedMediaType ErrorCode = "UnsupportedMediaType"

	// ErrorCodeInternalServerError indicates an internal server error.
	ErrorCodeInternalServerError ErrorCode = "InternalServerError"

	// ErrorCodeNotImplemented indicates the operation is not implemented.
	ErrorCodeNotImplemented ErrorCode = "NotImplemented"

	// ErrorCodeServiceUnavailable indicates the service is temporarily unavailable.
	ErrorCodeServiceUnavailable ErrorCode = "ServiceUnavailable"
)

// ErrorDetail represents additional error information in an OData error response.
type ErrorDetail struct {
	// Code is a service-defined error code for this detail.
	Code string

	// Target identifies the specific part of the request causing this error.
	Target string

	// Message is a human-readable description of this specific error.
	Message string
}

// ODataError provides a structured error that includes an HTTP status code,
// OData error code, and descriptive message. This type can be returned from
// hooks, overwrite handlers, and custom operations to provide precise error responses.
type ODataError struct {
	// StatusCode is the HTTP status code to return (e.g., 400, 404, 500).
	StatusCode int

	// Code is the OData-specific error code.
	Code ErrorCode

	// Message is a human-readable error description.
	Message string

	// Target optionally identifies the part of the request that caused the error.
	// For example, "Products(1)/Name" for a validation error on the Name property.
	Target string

	// Details provides additional error information for complex validation scenarios.
	Details []ErrorDetail

	// Err is the underlying error, if any. This allows error wrapping while
	// maintaining compatibility with errors.Is() and errors.As().
	Err error
}

// Error implements the error interface.
func (e *ODataError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap implements error unwrapping for errors.Is() and errors.As().
func (e *ODataError) Unwrap() error {
	return e.Err
}
