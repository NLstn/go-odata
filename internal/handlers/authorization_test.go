package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
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
