package operations_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/service/operations"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type odataErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Message string `json:"message"`
		} `json:"details"`
	} `json:"error"`
}

type noopLogger struct{}

func (noopLogger) Error(string, ...any) {}

func decodeODataError(t *testing.T, body []byte) odataErrorResponse {
	t.Helper()

	var resp odataErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to decode error response: %v. body: %s", err, string(body))
	}
	return resp
}

func newBaseHandler() *operations.Handler {
	return operations.NewHandler(
		make(map[string][]*actions.ActionDefinition),
		make(map[string][]*actions.FunctionDefinition),
		make(map[string]*handlers.EntityHandler),
		make(map[string]*metadata.EntityMetadata),
		"",
		noopLogger{},
	)
}

type boundTestEntity struct {
	ID uint `gorm:"primaryKey" odata:"key"`
}

func newBoundDependencies(t *testing.T) (map[string]*handlers.EntityHandler, map[string]*metadata.EntityMetadata, string) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&boundTestEntity{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(&boundTestEntity{})
	if err != nil {
		t.Fatalf("failed to analyze entity: %v", err)
	}

	handler := handlers.NewEntityHandler(db, entityMeta, nil)
	handler.SetNamespace("ODataService")
	handler.SetEntitiesMetadata(map[string]*metadata.EntityMetadata{entityMeta.EntitySetName: entityMeta})

	handlersMap := map[string]*handlers.EntityHandler{entityMeta.EntitySetName: handler}
	entities := map[string]*metadata.EntityMetadata{entityMeta.EntitySetName: entityMeta}

	return handlersMap, entities, entityMeta.EntitySetName
}

func TestHandleActionOrFunction_ActionNotFound(t *testing.T) {
	handler := newBaseHandler()
	req := httptest.NewRequest(http.MethodPost, "/UnknownAction", nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "UnknownAction", "", false, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	resp := decodeODataError(t, rec.Body.Bytes())
	if resp.Error.Code != "404" {
		t.Errorf("error code = %s, want 404", resp.Error.Code)
	}
	if resp.Error.Message != "Action not found" {
		t.Errorf("message = %q, want %q", resp.Error.Message, "Action not found")
	}
	if len(resp.Error.Details) == 0 || resp.Error.Details[0].Message != "Action 'UnknownAction' is not registered" {
		t.Fatalf("unexpected details: %#v", resp.Error.Details)
	}
}

func TestHandleActionOrFunction_ActionInvalidParameters(t *testing.T) {
	actionName := "UpdateName"
	handler := operations.NewHandler(
		map[string][]*actions.ActionDefinition{
			actionName: {
				{
					Name: actionName,
					Parameters: []actions.ParameterDefinition{
						{Name: "name", Type: reflect.TypeOf(""), Required: true},
					},
					Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
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

	req := httptest.NewRequest(http.MethodPost, "/UpdateName", bytes.NewBufferString(`{"name":123}`))
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, actionName, "", false, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	resp := decodeODataError(t, rec.Body.Bytes())
	if resp.Error.Message != "Invalid parameters" {
		t.Errorf("message = %q, want %q", resp.Error.Message, "Invalid parameters")
	}
	if len(resp.Error.Details) == 0 || resp.Error.Details[0].Message == "" {
		t.Fatalf("expected parameter validation details, got %#v", resp.Error.Details)
	}
}

func TestHandleActionOrFunction_BoundActionEntityNotFound(t *testing.T) {
	handlersMap, entities, entitySet := newBoundDependencies(t)

	handler := operations.NewHandler(
		map[string][]*actions.ActionDefinition{
			"DoBoundAction": {
				{
					Name:      "DoBoundAction",
					IsBound:   true,
					EntitySet: entitySet,
					Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
						return nil
					},
				},
			},
		},
		make(map[string][]*actions.FunctionDefinition),
		handlersMap,
		entities,
		"",
		noopLogger{},
	)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/%s(999)/DoBoundAction", entitySet), nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "DoBoundAction", "999", true, entitySet)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	resp := decodeODataError(t, rec.Body.Bytes())
	if len(resp.Error.Details) == 0 || resp.Error.Details[0].Message != "Entity with key '999' not found" {
		t.Fatalf("unexpected details: %#v", resp.Error.Details)
	}
}

func TestHandleActionOrFunction_ActionHandlerError(t *testing.T) {
	handler := operations.NewHandler(
		map[string][]*actions.ActionDefinition{
			"FailAction": {
				{
					Name: "FailAction",
					Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
						return errors.New("boom")
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

	req := httptest.NewRequest(http.MethodPost, "/FailAction", nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "FailAction", "", false, "")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleActionOrFunction_FunctionHandlerError(t *testing.T) {
	handler := operations.NewHandler(
		make(map[string][]*actions.ActionDefinition),
		map[string][]*actions.FunctionDefinition{
			"FailFunction": {
				{
					Name: "FailFunction",
					Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
						return nil, errors.New("boom")
					},
				},
			},
		},
		make(map[string]*handlers.EntityHandler),
		make(map[string]*metadata.EntityMetadata),
		"",
		noopLogger{},
	)

	req := httptest.NewRequest(http.MethodGet, "/FailFunction", nil)
	rec := httptest.NewRecorder()

	handler.HandleActionOrFunction(rec, req, "FailFunction", "", false, "")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
