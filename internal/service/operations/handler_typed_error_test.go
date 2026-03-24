package operations_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/hookerrors"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/odataerrors"
	"github.com/nlstn/go-odata/internal/service/operations"
)

func decodeJSONInto(t *testing.T, body []byte, v interface{}) error {
	t.Helper()
	return json.Unmarshal(body, v)
}

func TestHandleActionOrFunction_ActionODataError(t *testing.T) {
	tests := []struct {
		name           string
		returnErr      error
		expectedStatus int
		expectedCode   string
		expectedMsg    string
	}{
		{
			name: "ODataError with 400",
			returnErr: &odataerrors.ODataError{
				StatusCode: http.StatusBadRequest,
				Code:       odataerrors.ErrorCodeBadRequest,
				Message:    "validation failed",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "BadRequest",
			expectedMsg:    "validation failed",
		},
		{
			name: "ODataError with 403",
			returnErr: &odataerrors.ODataError{
				StatusCode: http.StatusForbidden,
				Code:       odataerrors.ErrorCodeForbidden,
				Message:    "access denied",
			},
			expectedStatus: http.StatusForbidden,
			expectedCode:   "Forbidden",
			expectedMsg:    "access denied",
		},
		{
			name: "ODataError with 409",
			returnErr: &odataerrors.ODataError{
				StatusCode: http.StatusConflict,
				Code:       odataerrors.ErrorCodeConflict,
				Message:    "resource already exists",
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "Conflict",
			expectedMsg:    "resource already exists",
		},
		{
			name: "ODataError with zero StatusCode defaults to 500",
			returnErr: &odataerrors.ODataError{
				Message: "no status code set",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "no status code set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := operations.NewHandler(
				map[string][]*actions.ActionDefinition{
					"TestAction": {
						{
							Name: "TestAction",
							Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
								return tt.returnErr
							},
						},
					},
				},
				make(map[string][]*actions.FunctionDefinition),
				make(map[string]*handlers.EntityHandler),
				make(map[string]*metadata.EntityMetadata),
				"",
				noopLogger{},
			)

			req := httptest.NewRequest(http.MethodPost, "/TestAction", nil)
			rec := httptest.NewRecorder()

			handler.HandleActionOrFunction(rec, req, "TestAction", "", false, "")

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			resp := decodeODataError(t, rec.Body.Bytes())
			if resp.Error.Message != tt.expectedMsg {
				t.Errorf("message = %q, want %q", resp.Error.Message, tt.expectedMsg)
			}
			if tt.expectedCode != "" && resp.Error.Code != tt.expectedCode {
				t.Errorf("code = %q, want %q", resp.Error.Code, tt.expectedCode)
			}
		})
	}
}

func TestHandleActionOrFunction_ActionHookError(t *testing.T) {
	tests := []struct {
		name           string
		returnErr      error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "HookError with 400",
			returnErr: &hookerrors.HookError{
				StatusCode: http.StatusBadRequest,
				Message:    "bad input from hook",
			},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "bad input from hook",
		},
		{
			name: "HookError with 429",
			returnErr: &hookerrors.HookError{
				StatusCode: http.StatusTooManyRequests,
				Message:    "rate limit exceeded",
			},
			expectedStatus: http.StatusTooManyRequests,
			expectedMsg:    "rate limit exceeded",
		},
		{
			name: "HookError with zero StatusCode defaults to 500",
			returnErr: &hookerrors.HookError{
				Message: "hook error no status",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "hook error no status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := operations.NewHandler(
				map[string][]*actions.ActionDefinition{
					"TestAction": {
						{
							Name: "TestAction",
							Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
								return tt.returnErr
							},
						},
					},
				},
				make(map[string][]*actions.FunctionDefinition),
				make(map[string]*handlers.EntityHandler),
				make(map[string]*metadata.EntityMetadata),
				"",
				noopLogger{},
			)

			req := httptest.NewRequest(http.MethodPost, "/TestAction", nil)
			rec := httptest.NewRecorder()

			handler.HandleActionOrFunction(rec, req, "TestAction", "", false, "")

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			resp := decodeODataError(t, rec.Body.Bytes())
			if resp.Error.Message != tt.expectedMsg {
				t.Errorf("message = %q, want %q", resp.Error.Message, tt.expectedMsg)
			}
		})
	}
}

func TestHandleActionOrFunction_FunctionODataError(t *testing.T) {
	tests := []struct {
		name           string
		returnErr      error
		expectedStatus int
		expectedCode   string
		expectedMsg    string
	}{
		{
			name: "ODataError with 400",
			returnErr: &odataerrors.ODataError{
				StatusCode: http.StatusBadRequest,
				Code:       odataerrors.ErrorCodeBadRequest,
				Message:    "invalid parameter",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "BadRequest",
			expectedMsg:    "invalid parameter",
		},
		{
			name: "ODataError with 404",
			returnErr: &odataerrors.ODataError{
				StatusCode: http.StatusNotFound,
				Code:       odataerrors.ErrorCodeNotFound,
				Message:    "resource not found",
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NotFound",
			expectedMsg:    "resource not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := operations.NewHandler(
				make(map[string][]*actions.ActionDefinition),
				map[string][]*actions.FunctionDefinition{
					"TestFunction": {
						{
							Name: "TestFunction",
							Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
								return nil, tt.returnErr
							},
						},
					},
				},
				make(map[string]*handlers.EntityHandler),
				make(map[string]*metadata.EntityMetadata),
				"",
				noopLogger{},
			)

			req := httptest.NewRequest(http.MethodGet, "/TestFunction", nil)
			rec := httptest.NewRecorder()

			handler.HandleActionOrFunction(rec, req, "TestFunction", "", false, "")

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			resp := decodeODataError(t, rec.Body.Bytes())
			if resp.Error.Message != tt.expectedMsg {
				t.Errorf("message = %q, want %q", resp.Error.Message, tt.expectedMsg)
			}
			if tt.expectedCode != "" && resp.Error.Code != tt.expectedCode {
				t.Errorf("code = %q, want %q", resp.Error.Code, tt.expectedCode)
			}
		})
	}
}

func TestHandleActionOrFunction_FunctionHookError(t *testing.T) {
	tests := []struct {
		name           string
		returnErr      error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "HookError with 403",
			returnErr: &hookerrors.HookError{
				StatusCode: http.StatusForbidden,
				Message:    "not allowed",
			},
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "not allowed",
		},
		{
			name: "HookError with 503",
			returnErr: &hookerrors.HookError{
				StatusCode: http.StatusServiceUnavailable,
				Message:    "service unavailable",
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedMsg:    "service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := operations.NewHandler(
				make(map[string][]*actions.ActionDefinition),
				map[string][]*actions.FunctionDefinition{
					"TestFunction": {
						{
							Name: "TestFunction",
							Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
								return nil, tt.returnErr
							},
						},
					},
				},
				make(map[string]*handlers.EntityHandler),
				make(map[string]*metadata.EntityMetadata),
				"",
				noopLogger{},
			)

			req := httptest.NewRequest(http.MethodGet, "/TestFunction", nil)
			rec := httptest.NewRecorder()

			handler.HandleActionOrFunction(rec, req, "TestFunction", "", false, "")

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			resp := decodeODataError(t, rec.Body.Bytes())
			if resp.Error.Message != tt.expectedMsg {
				t.Errorf("message = %q, want %q", resp.Error.Message, tt.expectedMsg)
			}
		})
	}
}

func TestHandleActionOrFunction_ODataErrorWithDetails(t *testing.T) {
	handler := operations.NewHandler(
		map[string][]*actions.ActionDefinition{
			"ValidateAction": {
				{
					Name: "ValidateAction",
					Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
						return &odataerrors.ODataError{
							StatusCode: http.StatusBadRequest,
							Code:       odataerrors.ErrorCodeBadRequest,
							Message:    "Validation failed",
							Details: []odataerrors.ErrorDetail{
								{Code: "Required", Target: "Name", Message: "Name is required"},
								{Code: "Range", Target: "Price", Message: "Price must be positive"},
							},
						}
					},
				},
			},
		},
		make(map[string][]*actions.FunctionDefinition),
		make(map[string]*handlers.EntityHandler),
		make(map[string]*metadata.EntityMetadata),
		"",
		noopLogger{},
	)

	req := httptest.NewRequest(http.MethodPost, "/ValidateAction", nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "ValidateAction", "", false, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	type detailedErrorResponse struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Details []struct {
				Code    string `json:"code"`
				Target  string `json:"target"`
				Message string `json:"message"`
			} `json:"details"`
		} `json:"error"`
	}

	var resp detailedErrorResponse
	if err := decodeJSONInto(t, rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Message != "Validation failed" {
		t.Errorf("message = %q, want %q", resp.Error.Message, "Validation failed")
	}
	if len(resp.Error.Details) != 2 {
		t.Fatalf("details count = %d, want 2", len(resp.Error.Details))
	}
	if resp.Error.Details[0].Target != "Name" {
		t.Errorf("details[0].Target = %q, want %q", resp.Error.Details[0].Target, "Name")
	}
	if resp.Error.Details[1].Target != "Price" {
		t.Errorf("details[1].Target = %q, want %q", resp.Error.Details[1].Target, "Price")
	}
}
