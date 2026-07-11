package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nlstn/go-odata/internal/query"
)

func TestHandleCollectionAppliesFilter(t *testing.T) {
	handler, db := setupTestHandler(t)

	entities := []TestEntity{
		{ID: 1, Name: "Alpha"},
		{ID: 2, Name: "Beta"},
		{ID: 3, Name: "Gamma"},
	}
	for _, entity := range entities {
		if err := db.Create(&entity).Error; err != nil {
			t.Fatalf("failed to seed data: %v", err)
		}
	}

	query := url.Values{}
	query.Set("$filter", "name eq 'Beta'")
	req := httptest.NewRequest(http.MethodGet, "/TestEntities?"+query.Encode(), nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 status, got %d", w.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	values, ok := payload["value"].([]interface{})
	if !ok {
		t.Fatalf("expected value array in response")
	}

	if len(values) != 1 {
		t.Fatalf("expected single entity, got %d", len(values))
	}

	entity, ok := values[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map entry for entity result")
	}

	if entity["name"] != "Beta" {
		t.Fatalf("expected filtered entity 'Beta', got %v", entity["name"])
	}
}

func TestHandleCollectionAppliesPagination(t *testing.T) {
	handler, db := setupTestHandler(t)

	entities := []TestEntity{
		{ID: 1, Name: "One"},
		{ID: 2, Name: "Two"},
		{ID: 3, Name: "Three"},
	}
	for _, entity := range entities {
		if err := db.Create(&entity).Error; err != nil {
			t.Fatalf("failed to seed data: %v", err)
		}
	}

	pagination := url.Values{}
	pagination.Set("$orderby", "id")
	pagination.Set("$top", "1")
	pagination.Set("$skip", "1")
	req := httptest.NewRequest(http.MethodGet, "/TestEntities?"+pagination.Encode(), nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 status, got %d", w.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	values, ok := payload["value"].([]interface{})
	if !ok {
		t.Fatalf("expected value array in response")
	}

	if len(values) != 1 {
		t.Fatalf("expected single entity due to pagination, got %d", len(values))
	}

	entity, ok := values[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map entry for paginated entity")
	}

	if id, ok := entity["id"].(float64); !ok || int(id) != 2 {
		t.Fatalf("expected entity with ID 2 after pagination, got %v", entity["id"])
	}
}

func TestHandleCollectionDeltaTokenWithoutChangeTracking(t *testing.T) {
	handler, db := setupTestHandler(t)

	if err := db.Create(&TestEntity{ID: 1, Name: "Entity"}).Error; err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/TestEntities?$deltatoken=invalid", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when track changes are disabled, got %d", w.Code)
	}
}

func TestNormalizeComputedResultValues(t *testing.T) {
	results := []map[string]interface{}{{
		"Name":        "Laptop",
		"UpperName":   []byte("LAPTOP"),
		"DoublePrice": "39.98",
		"Code":        "123",
	}}
	options := &query.QueryOptions{Compute: &query.ComputeTransformation{Expressions: []query.ComputeExpression{
		{Alias: "UpperName", Expression: &query.FilterExpression{Operator: query.OpToUpper}},
		{Alias: "DoublePrice", Expression: &query.FilterExpression{Operator: query.OpMul}},
	}}}

	normalizeComputedResultValues(results, options)

	if got := results[0]["UpperName"]; got != "LAPTOP" {
		t.Fatalf("UpperName = %#v, want LAPTOP", got)
	}
	if got := results[0]["DoublePrice"]; got != float64(39.98) {
		t.Fatalf("DoublePrice = %#v, want numeric 39.98", got)
	}
	if got := results[0]["Code"]; got != "123" {
		t.Fatalf("unrelated numeric-looking string changed to %#v", got)
	}
}
