package operations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TestEntity struct {
	ID   uint   `gorm:"primarykey" json:"id"`
	Name string `json:"name"`
}

func (TestEntity) TableName() string {
	return "test_entities"
}

type authContextCapturePolicy struct {
	lastAuthContext auth.AuthContext
	allowed         bool
}

func (p *authContextCapturePolicy) Authorize(ctx auth.AuthContext, _ auth.ResourceDescriptor, _ auth.Operation) auth.Decision {
	p.lastAuthContext = ctx
	if p.allowed {
		return auth.Allow()
	}
	return auth.Deny("test denial")
}

type denyPolicy struct{}

func (p denyPolicy) Authorize(_ auth.AuthContext, _ auth.ResourceDescriptor, _ auth.Operation) auth.Decision {
	return auth.Deny("access denied")
}

type operationCapturePolicy struct {
	lastOperation auth.Operation
	lastResource  auth.ResourceDescriptor
}

func (p *operationCapturePolicy) Authorize(_ auth.AuthContext, resource auth.ResourceDescriptor, operation auth.Operation) auth.Decision {
	p.lastOperation = operation
	p.lastResource = resource
	return auth.Allow()
}

func setupTestHandler(t *testing.T) *Handler {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test entity
	testEntity := &TestEntity{ID: 1, Name: "Test"}
	if err := db.Create(testEntity).Error; err != nil {
		t.Fatalf("Failed to create test entity: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(&TestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	entities := map[string]*metadata.EntityMetadata{
		"TestEntities": entityMeta,
	}

	entityHandler := handlers.NewEntityHandler(db, entityMeta, nil)

	handlersMap := map[string]*handlers.EntityHandler{
		"TestEntities": entityHandler,
	}

	testAction := &actions.ActionDefinition{
		Name:       "TestAction",
		IsBound:    false,
		EntitySet:  "",
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			w.WriteHeader(http.StatusOK)
			return nil
		},
	}

	testBoundAction := &actions.ActionDefinition{
		Name:       "TestBoundAction",
		IsBound:    true,
		EntitySet:  "TestEntities",
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			w.WriteHeader(http.StatusOK)
			return nil
		},
	}

	testFunction := &actions.FunctionDefinition{
		Name:       "TestFunction",
		IsBound:    false,
		EntitySet:  "",
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			return "test result", nil
		},
	}

	testBoundFunction := &actions.FunctionDefinition{
		Name:       "TestBoundFunction",
		IsBound:    true,
		EntitySet:  "TestEntities",
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			return "bound result", nil
		},
	}

	actionsMap := map[string][]*actions.ActionDefinition{
		"TestAction":      {testAction},
		"TestBoundAction": {testBoundAction},
	}

	functionsMap := map[string][]*actions.FunctionDefinition{
		"TestFunction":      {testFunction},
		"TestBoundFunction": {testBoundFunction},
	}

	return NewHandler(actionsMap, functionsMap, handlersMap, entities, "Test", nil)
}

func TestHandler_Authorization_Action(t *testing.T) {
	handler := setupTestHandler(t)
	policy := &operationCapturePolicy{}
	handler.SetPolicy(policy)

	req := httptest.NewRequest(http.MethodPost, "/TestAction", nil)
	w := httptest.NewRecorder()

	handler.HandleActionOrFunction(w, req, "TestAction", "", false, "")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	if policy.lastOperation != auth.OperationAction {
		t.Errorf("Operation = %v, want %v", policy.lastOperation, auth.OperationAction)
	}
}

func TestHandler_Authorization_Function(t *testing.T) {
	handler := setupTestHandler(t)
	policy := &operationCapturePolicy{}
	handler.SetPolicy(policy)

	req := httptest.NewRequest(http.MethodGet, "/TestFunction", nil)
	w := httptest.NewRecorder()

	handler.HandleActionOrFunction(w, req, "TestFunction", "", false, "")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	if policy.lastOperation != auth.OperationFunction {
		t.Errorf("Operation = %v, want %v", policy.lastOperation, auth.OperationFunction)
	}
}

func TestHandler_Authorization_Denied(t *testing.T) {
	handler := setupTestHandler(t)
	handler.SetPolicy(denyPolicy{})

	t.Run("Action denied without auth header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/TestAction", nil)
		w := httptest.NewRecorder()

		handler.HandleActionOrFunction(w, req, "TestAction", "", false, "")

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("Action denied with auth header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/TestAction", nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()

		handler.HandleActionOrFunction(w, req, "TestAction", "", false, "")

		if w.Code != http.StatusForbidden {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusForbidden)
		}
	})

	t.Run("Function denied without auth header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/TestFunction", nil)
		w := httptest.NewRecorder()

		handler.HandleActionOrFunction(w, req, "TestFunction", "", false, "")

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})
}

func TestHandler_Authorization_BoundAction(t *testing.T) {
	handler := setupTestHandler(t)
	policy := &operationCapturePolicy{}
	handler.SetPolicy(policy)

	req := httptest.NewRequest(http.MethodPost, "/TestEntities(1)/TestBoundAction", nil)
	w := httptest.NewRecorder()

	handler.HandleActionOrFunction(w, req, "TestBoundAction", "1", true, "TestEntities")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	if policy.lastOperation != auth.OperationAction {
		t.Errorf("Operation = %v, want %v", policy.lastOperation, auth.OperationAction)
	}

	if policy.lastResource.EntitySetName != "TestEntities" {
		t.Errorf("EntitySetName = %v, want %v", policy.lastResource.EntitySetName, "TestEntities")
	}
}

func TestHandler_Authorization_AuthContextExtraction(t *testing.T) {
	handler := setupTestHandler(t)
	policy := &authContextCapturePolicy{allowed: true}
	handler.SetPolicy(policy)

	// Create a request with auth data in context
	req := httptest.NewRequest(http.MethodPost, "/TestAction", nil)
	ctx := context.WithValue(req.Context(), auth.PrincipalContextKey, "user@example.com")
	ctx = context.WithValue(ctx, auth.RolesContextKey, []string{"admin", "user"})
	ctx = context.WithValue(ctx, auth.ClaimsContextKey, map[string]interface{}{"tenant_id": "123"})
	ctx = context.WithValue(ctx, auth.ScopesContextKey, []string{"read", "write"})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.HandleActionOrFunction(w, req, "TestAction", "", false, "")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify auth context was extracted
	if policy.lastAuthContext.Principal != "user@example.com" {
		t.Errorf("Principal = %v, want %v", policy.lastAuthContext.Principal, "user@example.com")
	}

	if len(policy.lastAuthContext.Roles) != 2 {
		t.Errorf("Roles length = %v, want %v", len(policy.lastAuthContext.Roles), 2)
	}

	if policy.lastAuthContext.Claims["tenant_id"] != "123" {
		t.Errorf("Claims[tenant_id] = %v, want %v", policy.lastAuthContext.Claims["tenant_id"], "123")
	}

	if len(policy.lastAuthContext.Scopes) != 2 {
		t.Errorf("Scopes length = %v, want %v", len(policy.lastAuthContext.Scopes), 2)
	}
}

func TestHandler_Authorization_NoPolicy(t *testing.T) {
	handler := setupTestHandler(t)
	// No policy set - should allow all requests

	req := httptest.NewRequest(http.MethodPost, "/TestAction", nil)
	w := httptest.NewRecorder()

	handler.HandleActionOrFunction(w, req, "TestAction", "", false, "")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v (expected to allow when no policy)", w.Code, http.StatusOK)
	}
}
