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

func TestHandleActionOrFunction_FunctionNotAcceptable(t *testing.T) {
	handler := operations.NewHandler(
		nil,
		map[string][]*actions.FunctionDefinition{
			"GetStatus": {
				{
					Name: "GetStatus",
					Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
						return "ok", nil
					},
				},
			},
		},
		nil,
		map[string]*metadata.EntityMetadata{},
		"",
		noopLogger{},
	)

	req := httptest.NewRequest(http.MethodGet, "/GetStatus", nil)
	req.Header.Set("Accept", "application/xml")
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "GetStatus", "", false, "")

	if rec.Code != http.StatusNotAcceptable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotAcceptable)
	}

	resp := decodeODataError(t, rec.Body.Bytes())
	if resp.Error.Message != "Not Acceptable" {
		t.Fatalf("message = %q, want %q", resp.Error.Message, "Not Acceptable")
	}
}

func TestHandleActionOrFunction_FunctionMetadataNone(t *testing.T) {
	handler := operations.NewHandler(
		nil,
		map[string][]*actions.FunctionDefinition{
			"GetStatus": {
				{
					Name: "GetStatus",
					Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
						return "ok", nil
					},
				},
			},
		},
		nil,
		map[string]*metadata.EntityMetadata{},
		"",
		noopLogger{},
	)

	req := httptest.NewRequest(http.MethodGet, "/GetStatus", nil)
	req.Header.Set("Accept", "application/json;odata.metadata=none")
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "GetStatus", "", false, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json;odata.metadata=none" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json;odata.metadata=none")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := payload["@odata.context"]; ok {
		t.Fatalf("expected no @odata.context for metadata=none, got %v", payload["@odata.context"])
	}
	if payload["value"] != "ok" {
		t.Fatalf("value = %v, want %q", payload["value"], "ok")
	}
}

func TestHandleActionOrFunction_BoundEntitySetMissing(t *testing.T) {
	handler := operations.NewHandler(
		map[string][]*actions.ActionDefinition{
			"DoThing": {
				{
					Name:      "DoThing",
					IsBound:   true,
					EntitySet: "Missing",
					Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
						return nil
					},
				},
			},
		},
		nil,
		map[string]*handlers.EntityHandler{},
		map[string]*metadata.EntityMetadata{},
		"",
		noopLogger{},
	)

	req := httptest.NewRequest(http.MethodPost, "/Missing(1)/DoThing", nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "DoThing", "1", true, "Missing")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	resp := decodeODataError(t, rec.Body.Bytes())
	if len(resp.Error.Details) == 0 || resp.Error.Details[0].Message != "Entity set 'Missing' is not registered" {
		t.Fatalf("unexpected details: %#v", resp.Error.Details)
	}
}
