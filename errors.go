package odata

import (
	"errors"
	"net/http"

	"github.com/nlstn/go-odata/internal/odataerrors"
)

// Sentinel errors for common OData error conditions.
// These can be used with errors.Is() for error handling.
var (
	// ErrEntityNotFound indicates the requested entity does not exist.
	// Maps to HTTP 404 Not Found.
	ErrEntityNotFound = errors.New("odata: entity not found")

	// ErrValidationError indicates the request data failed validation.
	// Maps to HTTP 400 Bad Request.
	ErrValidationError = errors.New("odata: validation error")

	// ErrUnauthorized indicates the request lacks valid authentication.
	// Maps to HTTP 401 Unauthorized.
	ErrUnauthorized = errors.New("odata: unauthorized")

	// ErrForbidden indicates the authenticated user lacks permission.
	// Maps to HTTP 403 Forbidden.
	ErrForbidden = errors.New("odata: forbidden")

	// ErrMethodNotAllowed indicates the operation is not supported for this entity.
	// Maps to HTTP 405 Method Not Allowed.
	ErrMethodNotAllowed = errors.New("odata: method not allowed")

	// ErrConflict indicates a conflict with the current state (e.g., duplicate key, concurrent modification).
	// Maps to HTTP 409 Conflict.
	ErrConflict = errors.New("odata: conflict")

	// ErrPreconditionFailed indicates an ETag precondition check failed.
	// Maps to HTTP 412 Precondition Failed.
	ErrPreconditionFailed = errors.New("odata: precondition failed")

	// ErrUnsupportedMediaType indicates the request content type is not supported.
	// Maps to HTTP 415 Unsupported Media Type.
	ErrUnsupportedMediaType = errors.New("odata: unsupported media type")

	// ErrInternalServerError indicates an unexpected server error.
	// Maps to HTTP 500 Internal Server Error.
	ErrInternalServerError = errors.New("odata: internal server error")
)

// ErrorCode represents standard OData error codes.
// These codes provide semantic information about the error type
// and follow the OData specification conventions.
type ErrorCode = odataerrors.ErrorCode

// Standard OData error codes as defined in the OData specification.
const (
	// ErrorCodeGeneral is a general, unspecified error.
	ErrorCodeGeneral = odataerrors.ErrorCodeGeneral

	// ErrorCodeNotFound indicates the requested resource was not found.
	ErrorCodeNotFound = odataerrors.ErrorCodeNotFound

	// ErrorCodeBadRequest indicates malformed or invalid request syntax.
	ErrorCodeBadRequest = odataerrors.ErrorCodeBadRequest

	// ErrorCodeUnauthorized indicates missing or invalid authentication.
	ErrorCodeUnauthorized = odataerrors.ErrorCodeUnauthorized

	// ErrorCodeForbidden indicates insufficient permissions.
	ErrorCodeForbidden = odataerrors.ErrorCodeForbidden

	// ErrorCodeMethodNotAllowed indicates the HTTP method is not supported.
	ErrorCodeMethodNotAllowed = odataerrors.ErrorCodeMethodNotAllowed

	// ErrorCodeConflict indicates a conflict with current resource state.
	ErrorCodeConflict = odataerrors.ErrorCodeConflict

	// ErrorCodePreconditionFailed indicates an ETag precondition failed.
	ErrorCodePreconditionFailed = odataerrors.ErrorCodePreconditionFailed

	// ErrorCodeUnsupportedMediaType indicates unsupported Content-Type.
	ErrorCodeUnsupportedMediaType = odataerrors.ErrorCodeUnsupportedMediaType

	// ErrorCodeInternalServerError indicates an internal server error.
	ErrorCodeInternalServerError = odataerrors.ErrorCodeInternalServerError

	// ErrorCodeNotImplemented indicates the operation is not implemented.
	ErrorCodeNotImplemented = odataerrors.ErrorCodeNotImplemented

	// ErrorCodeServiceUnavailable indicates the service is temporarily unavailable.
	ErrorCodeServiceUnavailable = odataerrors.ErrorCodeServiceUnavailable
)

// ODataError provides a structured error that includes an HTTP status code,
// OData error code, and descriptive message. This type can be returned from
// hooks, overwrite handlers, and custom operations to provide precise error responses.
//
// Example usage in an action or function handler:
//
//	Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
//	    if !isAuthorized(r) {
//	        return &odata.ODataError{
//	            StatusCode: http.StatusForbidden,
//	            Code:       odata.ErrorCodeForbidden,
//	            Message:    "You do not have permission to perform this action",
//	        }
//	    }
//	    return nil
//	}
//
// Example usage in an overwrite handler:
//
//	func (ctx *OverwriteContext) (interface{}, error) {
//	    product, err := externalAPI.GetProduct(ctx.EntityKey)
//	    if err != nil {
//	        if errors.Is(err, externalAPI.ErrNotFound) {
//	            return nil, &odata.ODataError{
//	                StatusCode: http.StatusNotFound,
//	                Code:       odata.ErrorCodeNotFound,
//	                Message:    fmt.Sprintf("Product %s not found", ctx.EntityKey),
//	            }
//	        }
//	        return nil, err
//	    }
//	    return product, nil
//	}
type ODataError = odataerrors.ODataError

// ErrorDetail represents additional error information in an OData error response.
type ErrorDetail = odataerrors.ErrorDetail

// MapErrorToHTTPStatus returns the appropriate HTTP status code for common errors.
// This helper can be used in custom handlers to determine status codes.
//
// Example usage:
//
//	status := odata.MapErrorToHTTPStatus(err)
//	w.WriteHeader(status)
func MapErrorToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Check for ODataError first
	var odataErr *ODataError
	if errors.As(err, &odataErr) {
		return odataErr.StatusCode
	}

	// Check for HookError
	var hookErr *HookError
	if errors.As(err, &hookErr) {
		return hookErr.StatusCode
	}

	// Check for sentinel errors
	switch {
	case errors.Is(err, ErrEntityNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrValidationError):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrMethodNotAllowed):
		return http.StatusMethodNotAllowed
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrPreconditionFailed):
		return http.StatusPreconditionFailed
	case errors.Is(err, ErrUnsupportedMediaType):
		return http.StatusUnsupportedMediaType
	case errors.Is(err, ErrInternalServerError):
		return http.StatusInternalServerError
	}

	// Default to internal server error for unknown errors
	return http.StatusInternalServerError
}

// IsValidationError returns true if the error is a validation error.
//
// Example usage:
//
//	if odata.IsValidationError(err) {
//	    log.Printf("Validation failed: %v", err)
//	    // Handle validation error
//	}
func IsValidationError(err error) bool {
	return errors.Is(err, ErrValidationError)
}

// IsNotFoundError returns true if the error indicates an entity was not found.
//
// Example usage:
//
//	entity, err := getEntity(id)
//	if odata.IsNotFoundError(err) {
//	    return nil, nil // Entity doesn't exist, return nil without error
//	}
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrEntityNotFound)
}
