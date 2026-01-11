package operations_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
