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
