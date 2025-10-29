package odata

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

type boundTestEntity struct {
	ID uint `gorm:"primaryKey" odata:"key"`
}

func decodeODataError(t *testing.T, body []byte) odataErrorResponse {
	t.Helper()

	var resp odataErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to decode error response: %v. body: %s", err, string(body))
	}
	return resp
}

func newTestService() *Service {
	return &Service{
		actions:   make(map[string][]*actions.ActionDefinition),
		functions: make(map[string][]*actions.FunctionDefinition),
		handlers:  make(map[string]*handlers.EntityHandler),
		entities:  make(map[string]*metadata.EntityMetadata),
	}
}

func newBoundService(t *testing.T) (*Service, string) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&boundTestEntity{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	svc := NewService(db)
	if err := svc.RegisterEntity(&boundTestEntity{}); err != nil {
		t.Fatalf("failed to register entity: %v", err)
	}

	var entitySet string
	for name := range svc.entities {
		entitySet = name
		break
	}

	if entitySet == "" {
		t.Fatal("no entity set registered")
	}

	return svc, entitySet
}

func TestHandleActionOrFunction_ActionNotFound(t *testing.T) {
	svc := newTestService()
	req := httptest.NewRequest(http.MethodPost, "/UnknownAction", nil)
	rec := httptest.NewRecorder()

	svc.handleActionOrFunction(rec, req, "UnknownAction", "", false, "")

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
	svc := newTestService()

	actionName := "UpdateName"
	svc.actions[actionName] = []*actions.ActionDefinition{
		&actions.ActionDefinition{
			Name: actionName,
			Parameters: []actions.ParameterDefinition{
				{Name: "name", Type: reflect.TypeOf(""), Required: true},
			},
			Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/UpdateName", bytes.NewBufferString(`{"name":123}`))
	rec := httptest.NewRecorder()

	svc.handleActionOrFunction(rec, req, actionName, "", false, "")

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
	svc, entitySet := newBoundService(t)

	if err := svc.RegisterAction(ActionDefinition{
		Name:      "DoBoundAction",
		IsBound:   true,
		EntitySet: entitySet,
		Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
			return nil
		},
	}); err != nil {
		t.Fatalf("failed to register action: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/%s(999)/DoBoundAction", entitySet), nil)
	rec := httptest.NewRecorder()

	svc.handleActionOrFunction(rec, req, "DoBoundAction", "999", true, entitySet)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	resp := decodeODataError(t, rec.Body.Bytes())
	if resp.Error.Message != "Entity not found" {
		t.Errorf("message = %q, want %q", resp.Error.Message, "Entity not found")
	}
	if len(resp.Error.Details) == 0 || resp.Error.Details[0].Message != "Entity with key '999' not found" {
		t.Fatalf("unexpected details: %#v", resp.Error.Details)
	}
}

func TestHandleActionOrFunction_ActionHandlerError(t *testing.T) {
	svc := newTestService()

	actionName := "FailingAction"
	svc.actions[actionName] = []*actions.ActionDefinition{
		&actions.ActionDefinition{
			Name: actionName,
			Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
				return errors.New("boom")
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/FailingAction", nil)
	rec := httptest.NewRecorder()

	svc.handleActionOrFunction(rec, req, actionName, "", false, "")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	resp := decodeODataError(t, rec.Body.Bytes())
	if resp.Error.Message != "Action failed" {
		t.Errorf("message = %q, want %q", resp.Error.Message, "Action failed")
	}
	if len(resp.Error.Details) == 0 || resp.Error.Details[0].Message != "boom" {
		t.Fatalf("unexpected details: %#v", resp.Error.Details)
	}
}

func TestHandleActionOrFunction_FunctionNotFound(t *testing.T) {
	svc := newTestService()
	req := httptest.NewRequest(http.MethodGet, "/UnknownFunction()", nil)
	rec := httptest.NewRecorder()

	svc.handleActionOrFunction(rec, req, "UnknownFunction", "", false, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	resp := decodeODataError(t, rec.Body.Bytes())
	if resp.Error.Message != "Function not found" {
		t.Errorf("message = %q, want %q", resp.Error.Message, "Function not found")
	}
	if len(resp.Error.Details) == 0 || resp.Error.Details[0].Message != "Function 'UnknownFunction' is not registered" {
		t.Fatalf("unexpected details: %#v", resp.Error.Details)
	}
}

func TestHandleActionOrFunction_FunctionInvalidParameters(t *testing.T) {
	svc := newTestService()

	functionName := "Compute"
	svc.functions[functionName] = []*actions.FunctionDefinition{
		&actions.FunctionDefinition{
			Name: functionName,
			Parameters: []actions.ParameterDefinition{
				{Name: "value", Type: reflect.TypeOf(int64(0)), Required: true},
			},
			ReturnType: reflect.TypeOf(int64(0)),
			Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
				return nil, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/Compute?value=abc", nil)
	rec := httptest.NewRecorder()

	svc.handleActionOrFunction(rec, req, functionName, "", false, "")

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

func TestHandleActionOrFunction_BoundFunctionEntityNotFound(t *testing.T) {
	svc, entitySet := newBoundService(t)

	if err := svc.RegisterFunction(FunctionDefinition{
		Name:      "BoundFunction",
		IsBound:   true,
		EntitySet: entitySet,
		Parameters: []ParameterDefinition{
			{Name: "value", Type: reflect.TypeOf(int64(0)), Required: false},
		},
		ReturnType: reflect.TypeOf(int64(0)),
		Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
			return int64(0), nil
		},
	}); err != nil {
		t.Fatalf("failed to register function: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s(999)/BoundFunction", entitySet), nil)
	rec := httptest.NewRecorder()

	svc.handleActionOrFunction(rec, req, "BoundFunction", "999", true, entitySet)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	resp := decodeODataError(t, rec.Body.Bytes())
	if resp.Error.Message != "Entity not found" {
		t.Errorf("message = %q, want %q", resp.Error.Message, "Entity not found")
	}
	if len(resp.Error.Details) == 0 || resp.Error.Details[0].Message != "Entity with key '999' not found" {
		t.Fatalf("unexpected details: %#v", resp.Error.Details)
	}
}

func TestHandleActionOrFunction_FunctionHandlerError(t *testing.T) {
	svc := newTestService()

	functionName := "FailingFunction"
	svc.functions[functionName] = []*actions.FunctionDefinition{
		&actions.FunctionDefinition{
			Name:       functionName,
			ReturnType: reflect.TypeOf(int64(0)),
			Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
				return nil, errors.New("explode")
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/FailingFunction", nil)
	rec := httptest.NewRecorder()

	svc.handleActionOrFunction(rec, req, functionName, "", false, "")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	resp := decodeODataError(t, rec.Body.Bytes())
	if resp.Error.Message != "Function failed" {
		t.Errorf("message = %q, want %q", resp.Error.Message, "Function failed")
	}
	if len(resp.Error.Details) == 0 || resp.Error.Details[0].Message != "explode" {
		t.Fatalf("unexpected details: %#v", resp.Error.Details)
	}
}
