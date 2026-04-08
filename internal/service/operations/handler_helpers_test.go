package operations_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/service/operations"
)

func decodeError(t *testing.T, body []byte) odataErrorResponse {
	t.Helper()

	var resp odataErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to decode error response: %v. body: %s", err, string(body))
	}
	return resp
}

func newHandler() *operations.Handler {
	return operations.NewHandler(
		make(map[string][]*actions.ActionDefinition),
		make(map[string][]*actions.FunctionDefinition),
		make(map[string]*handlers.EntityHandler),
		make(map[string]*metadata.EntityMetadata),
		"",
		noopLogger{},
	)
}

func TestHandler_SetLogger(t *testing.T) {
	handler := newHandler()
	handler.SetLogger(nil) // Should not panic
}

func TestHandler_SetNamespace(t *testing.T) {
	handler := newHandler()
	handler.SetNamespace("TestNamespace")
	// Cannot directly check namespace since it's unexported, but should not panic
}

func TestHandler_HandleActionOrFunction_FunctionNotFound(t *testing.T) {
	handler := newHandler()
	req := httptest.NewRequest(http.MethodGet, "/UnknownFunction", nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "UnknownFunction", "", false, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	resp := decodeError(t, rec.Body.Bytes())
	if resp.Error.Code != "404" {
		t.Errorf("error code = %s, want 404", resp.Error.Code)
	}
	if resp.Error.Message != "Function not found" {
		t.Errorf("message = %q, want %q", resp.Error.Message, "Function not found")
	}
}

func TestHandler_HandleActionOrFunction_MethodNotAllowed(t *testing.T) {
	handler := newHandler()

	methods := []string{http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/TestOp", nil)
			rec := httptest.NewRecorder()

			handler.HandleActionOrFunction(rec, req, "TestOp", "", false, "")

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// TestHandler_QueryOptionsInjectedIntoFunctionHandler verifies that parsed OData
// query options are available inside a function handler via
// actions.QueryOptionsFromRequest.
func TestHandler_QueryOptionsInjectedIntoFunctionHandler(t *testing.T) {
	var capturedTop *int

	handler := operations.NewHandler(
		make(map[string][]*actions.ActionDefinition),
		map[string][]*actions.FunctionDefinition{
			"ListItems": {
				{
					Name: "ListItems",
					Handler: func(_ http.ResponseWriter, r *http.Request, _ interface{}, _ map[string]interface{}) (interface{}, error) {
						opts := actions.QueryOptionsFromRequest(r)
						if opts != nil {
							capturedTop = opts.Top
						}
						return []string{"a", "b", "c"}, nil
					},
				},
			},
		},
		make(map[string]*handlers.EntityHandler),
		make(map[string]*metadata.EntityMetadata),
		"",
		noopLogger{},
	)

	req := httptest.NewRequest(http.MethodGet, "/ListItems?$top=2", nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "ListItems", "", false, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if capturedTop == nil {
		t.Fatal("expected Top query option to be captured, got nil")
	}
	if *capturedTop != 2 {
		t.Errorf("Top = %d, want 2", *capturedTop)
	}
}

// TestHandler_QueryOptionsInjectedIntoActionHandler verifies that parsed OData
// query options are available inside an action handler via
// actions.QueryOptionsFromRequest.
func TestHandler_QueryOptionsInjectedIntoActionHandler(t *testing.T) {
	var capturedOrderBy string

	handler := operations.NewHandler(
		map[string][]*actions.ActionDefinition{
			"DoSomething": {
				{
					Name: "DoSomething",
					Handler: func(w http.ResponseWriter, r *http.Request, _ interface{}, _ map[string]interface{}) error {
						opts := actions.QueryOptionsFromRequest(r)
						if opts != nil && len(opts.OrderBy) > 0 {
							capturedOrderBy = opts.OrderBy[0].Property
						}
						w.WriteHeader(http.StatusNoContent)
						return nil
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

	req := httptest.NewRequest(http.MethodPost, "/DoSomething?$orderby=Name", nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "DoSomething", "", false, "")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	if capturedOrderBy != "Name" {
		t.Errorf("OrderBy property = %q, want %q", capturedOrderBy, "Name")
	}
}

// TestHandler_InvalidQueryOptionsReturn400 verifies that malformed OData system
// query options (e.g. $top=abc) cause the framework to return 400 Bad Request
// before the action or function handler is called.
func TestHandler_InvalidQueryOptionsReturn400(t *testing.T) {
	called := false

	handler := operations.NewHandler(
		make(map[string][]*actions.ActionDefinition),
		map[string][]*actions.FunctionDefinition{
			"ListItems": {
				{
					Name: "ListItems",
					Handler: func(_ http.ResponseWriter, _ *http.Request, _ interface{}, _ map[string]interface{}) (interface{}, error) {
						called = true
						return nil, nil
					},
				},
			},
		},
		make(map[string]*handlers.EntityHandler),
		make(map[string]*metadata.EntityMetadata),
		"",
		noopLogger{},
	)

	req := httptest.NewRequest(http.MethodGet, "/ListItems?$top=notanumber", nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "ListItems", "", false, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if called {
		t.Error("expected function handler NOT to be called when query options are invalid")
	}
}

// TestHandler_NonSystemParametersAreExcludedFromQueryParsing verifies that
// non-$ query parameters are not interpreted as OData system query options.
func TestHandler_NonSystemParametersAreExcludedFromQueryParsing(t *testing.T) {
	var (
		capturedParamCount string
		capturedCount      bool
		hasOpts            bool
	)

	handler := operations.NewHandler(
		make(map[string][]*actions.ActionDefinition),
		map[string][]*actions.FunctionDefinition{
			"ListItems": {
				{
					Name: "ListItems",
					Parameters: []actions.ParameterDefinition{
						{Name: "count", Type: reflect.TypeOf(""), Required: true},
					},
					Handler: func(_ http.ResponseWriter, r *http.Request, _ interface{}, params map[string]interface{}) (interface{}, error) {
						capturedParamCount = params["count"].(string)

						opts := actions.QueryOptionsFromRequest(r)
						if opts != nil {
							hasOpts = true
							capturedCount = opts.Count
						}

						return []string{"ok"}, nil
					},
				},
			},
		},
		make(map[string]*handlers.EntityHandler),
		make(map[string]*metadata.EntityMetadata),
		"",
		noopLogger{},
	)

	req := httptest.NewRequest(http.MethodGet, "/ListItems(count='paramValue')?$count=true", nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "ListItems", "", false, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if capturedParamCount != "paramValue" {
		t.Fatalf("function parameter count = %q, want %q", capturedParamCount, "paramValue")
	}

	if !hasOpts {
		t.Fatal("expected query options to be injected")
	}

	if !capturedCount {
		t.Fatalf("query option $count = %v, want true", capturedCount)
	}
}
