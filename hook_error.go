package odata

import "github.com/nlstn/go-odata/internal/hookerrors"

// HookError is an error type that can be returned from lifecycle hooks
// to specify a custom HTTP status code and error message.
//
// Example usage in a BeforeReadEntity hook:
//
//	func (e *Employee) ODataBeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
//	    if !userHasAccess(ctx) {
//	        return nil, &odata.HookError{
//	            StatusCode: http.StatusUnauthorized,
//	            Message:    "User is not authorized to access this resource",
//	        }
//	    }
//	    return nil, nil
//	}
type HookError = hookerrors.HookError

// NewHookError creates a new HookError with the specified status code and message.
func NewHookError(statusCode int, message string) *HookError {
	return &HookError{
		StatusCode: statusCode,
		Message:    message,
	}
}
