package response

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/hookerrors"
	"github.com/nlstn/go-odata/internal/odataerrors"
)

// BuildODataErrorResponse maps an error into an HTTP status and OData error payload.
//
// Precedence:
//  1. *odataerrors.ODataError
//  2. *hookerrors.HookError
//  3. generic error fallback
func BuildODataErrorResponse(err error, defaultStatus int, defaultMessage string) (int, *ODataError) {
	status := defaultStatus
	if status == 0 {
		status = http.StatusInternalServerError
	}

	var odataErr *odataerrors.ODataError
	if errors.As(err, &odataErr) {
		if odataErr.StatusCode != 0 {
			status = odataErr.StatusCode
		}

		code := string(odataErr.Code)
		if code == "" {
			code = fmt.Sprintf("%d", status)
		}

		message := odataErr.Message
		if message == "" {
			message = defaultMessage
		}

		respErr := &ODataError{
			Code:    code,
			Message: message,
			Target:  odataErr.Target,
		}

		for _, d := range odataErr.Details {
			respErr.Details = append(respErr.Details, ODataErrorDetail{
				Code:    d.Code,
				Target:  d.Target,
				Message: d.Message,
			})
		}

		return status, respErr
	}

	var hookErr *hookerrors.HookError
	if errors.As(err, &hookErr) {
		if hookErr.StatusCode != 0 {
			status = hookErr.StatusCode
		}

		code := hookErr.Code
		if code == "" {
			code = fmt.Sprintf("%d", status)
		}

		message := hookErr.Message
		if message == "" {
			message = defaultMessage
		}

		respErr := &ODataError{
			Code:    code,
			Message: message,
			Target:  hookErr.Target,
		}

		for _, d := range hookErr.Details {
			respErr.Details = append(respErr.Details, ODataErrorDetail{
				Code:    d.Code,
				Target:  d.Target,
				Message: d.Message,
			})
		}

		if len(respErr.Details) == 0 && hookErr.Err != nil {
			respErr.Details = []ODataErrorDetail{{
				Code:    code,
				Message: hookErr.Err.Error(),
			}}
		}

		return status, respErr
	}

	code := fmt.Sprintf("%d", status)
	message := defaultMessage
	if message == "" {
		message = http.StatusText(status)
		if message == "" {
			message = "Error"
		}
	}

	respErr := &ODataError{
		Code:    code,
		Message: message,
	}
	if err != nil {
		respErr.Details = []ODataErrorDetail{{
			Code:    code,
			Message: err.Error(),
		}}
	}

	return status, respErr
}
