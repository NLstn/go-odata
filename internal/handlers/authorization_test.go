package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
)

type denyPolicy struct {
	reason string
}

func (p denyPolicy) Authorize(_ auth.AuthContext, _ auth.ResourceDescriptor, _ auth.Operation) auth.Decision {
	return auth.Deny(p.reason)
}

type operationPolicy struct {
	expected auth.Operation
	last     auth.Operation
}

func (p *operationPolicy) Authorize(_ auth.AuthContext, _ auth.ResourceDescriptor, operation auth.Operation) auth.Decision {
	p.last = operation
	if operation != p.expected {
		return auth.Deny("unexpected operation")
	}
	return auth.Allow()
}

type filterPolicy struct {
	filter *query.FilterExpression
}

func (p filterPolicy) Authorize(_ auth.AuthContext, _ auth.ResourceDescriptor, _ auth.Operation) auth.Decision {
	return auth.Allow()
}

func (p filterPolicy) QueryFilter(_ auth.AuthContext, _ auth.ResourceDescriptor, _ auth.Operation) (*query.FilterExpression, error) {
	return p.filter, nil
}

type capturePolicy struct {
	resources  []auth.ResourceDescriptor
	operations []auth.Operation
}

func (p *capturePolicy) Authorize(_ auth.AuthContext, resource auth.ResourceDescriptor, operation auth.Operation) auth.Decision {
	// Make a deep copy of the resource descriptor to capture entity state at authorization time
	captured := resource
	if resource.Entity != nil {
		// Make a copy of the entity to preserve its state at authorization time
		if testEntity, ok := resource.Entity.(*TestEntity); ok {
			copy := *testEntity
			captured.Entity = &copy
		}
	}
	p.resources = append(p.resources, captured)
	p.operations = append(p.operations, operation)
	return auth.Allow()
}

func TestEntityHandlerAuthorizationDenied(t *testing.T) {
	handler, _ := setupTestHandler(t)
	handler.SetPolicy(denyPolicy{reason: "blocked"})

	t.Run("unauthorized without header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/TestEntities", nil)
		w := httptest.NewRecorder()

		handler.HandleCollection(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("forbidden with header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/TestEntities", nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()

		handler.HandleCollection(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusForbidden)
		}
	})
}

func TestMetadataHandlerAuthorizationUsesMetadataOperation(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	entityMeta, err := metadata.AnalyzeEntity(TestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	entities[entityMeta.EntitySetName] = entityMeta

	policy := &operationPolicy{expected: auth.OperationMetadata}
	handler := NewMetadataHandler(entities)
	handler.SetPolicy(policy)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if policy.last != auth.OperationMetadata {
		t.Fatalf("Expected operation %v, got %v", auth.OperationMetadata, policy.last)
	}
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestServiceDocumentAuthorizationUsesReadOperation(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	entityMeta, err := metadata.AnalyzeEntity(TestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	entities[entityMeta.EntitySetName] = entityMeta

	policy := &operationPolicy{expected: auth.OperationRead}
	handler := NewServiceDocumentHandler(entities, nil)
	handler.SetPolicy(policy)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if policy.last != auth.OperationRead {
		t.Fatalf("Expected operation %v, got %v", auth.OperationRead, policy.last)
	}
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestPolicyFilterConstrainsCollectionResults(t *testing.T) {
	handler, db := setupTestHandler(t)

	testData := []TestEntity{
		{ID: 1, Name: "Allowed"},
		{ID: 2, Name: "Denied"},
	}
	for _, entity := range testData {
		db.Create(&entity)
	}

	handler.SetPolicy(filterPolicy{
		filter: &query.FilterExpression{
			Property: "id",
			Operator: query.OpEqual,
			Value:    1,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/TestEntities", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 1 {
		t.Fatalf("len(value) = %v, want 1", len(value))
	}

	countReq := httptest.NewRequest(http.MethodGet, "/TestEntities/$count", nil)
	countReq.Header.Set("Authorization", "Bearer token")
	countW := httptest.NewRecorder()

	handler.HandleCount(countW, countReq)

	if countW.Code != http.StatusOK {
		t.Fatalf("Count status = %v, want %v", countW.Code, http.StatusOK)
	}

	if strings.TrimSpace(countW.Body.String()) != "1" {
		t.Fatalf("Count body = %q, want %q", countW.Body.String(), "1")
	}
}

func TestEntityMutationAuthorizationIncludesEntityData(t *testing.T) {
	handler, db := setupTestHandler(t)

	db.Create(&TestEntity{ID: 1, Name: "Original"})
	db.Create(&TestEntity{ID: 2, Name: "Delete Me"})

	policy := &capturePolicy{}
	handler.SetPolicy(policy)

	t.Run("patch", func(t *testing.T) {
		body := strings.NewReader(`{"name":"Updated"}`)
		req := httptest.NewRequest(http.MethodPatch, "/TestEntities(1)", body)
		req.Header.Set("Authorization", "Bearer token")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.HandleEntity(w, req, "1")

		if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
			t.Fatalf("Status = %v, want %v or %v", w.Code, http.StatusNoContent, http.StatusOK)
		}

		resource := findAuthorizedResource(t, policy, auth.OperationUpdate)
		assertResourceKey(t, resource, "id", 1)

		// Verify entity data is included and has correct field values at authorization time
		if resource.Entity == nil {
			t.Fatal("Expected Entity to be present in resource descriptor")
		}
		entity, ok := resource.Entity.(*TestEntity)
		if !ok {
			t.Fatalf("Expected Entity to be *TestEntity, got %T", resource.Entity)
		}
		if entity.ID != 1 {
			t.Errorf("Entity ID = %v, want 1", entity.ID)
		}
		if entity.Name != "Original" {
			t.Errorf("Entity Name = %q, want %q (entity should have original data at authorization time)", entity.Name, "Original")
		}
	})

	t.Run("delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/TestEntities(2)", nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()

		handler.HandleEntity(w, req, "2")

		if w.Code != http.StatusNoContent {
			t.Fatalf("Status = %v, want %v", w.Code, http.StatusNoContent)
		}

		resource := findAuthorizedResource(t, policy, auth.OperationDelete)
		assertResourceKey(t, resource, "id", 2)

		// Verify entity data is included and has correct field values
		if resource.Entity == nil {
			t.Fatal("Expected Entity to be present in resource descriptor")
		}
		entity, ok := resource.Entity.(*TestEntity)
		if !ok {
			t.Fatalf("Expected Entity to be *TestEntity, got %T", resource.Entity)
		}
		if entity.ID != 2 {
			t.Errorf("Entity ID = %v, want 2", entity.ID)
		}
		if entity.Name != "Delete Me" {
			t.Errorf("Entity Name = %q, want %q", entity.Name, "Delete Me")
		}
	})
}

func findAuthorizedResource(t *testing.T, policy *capturePolicy, operation auth.Operation) auth.ResourceDescriptor {
	t.Helper()

	for i := len(policy.operations) - 1; i >= 0; i-- {
		if policy.operations[i] != operation {
			continue
		}
		resource := policy.resources[i]
		if resource.Entity != nil {
			return resource
		}
	}

	t.Fatalf("Expected authorization call with entity data for operation %v", operation)
	return auth.ResourceDescriptor{}
}

func assertResourceKey(t *testing.T, resource auth.ResourceDescriptor, key string, expected int) {
	t.Helper()

	if resource.KeyValues == nil {
		t.Fatalf("Expected key values on resource descriptor")
	}
	value, ok := resource.KeyValues[key]
	if !ok {
		t.Fatalf("Expected key %q on resource descriptor", key)
	}
	intValue, ok := value.(int)
	if !ok {
		t.Fatalf("Expected key %q to be int, got %T", key, value)
	}
	if intValue != expected {
		t.Fatalf("Key %q = %v, want %v", key, intValue, expected)
	}
}
