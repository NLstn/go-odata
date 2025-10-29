package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestAddNavigationLinksWithNilData(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	result := addNavigationLinks(nil, nil, nil, req, "Products", "minimal", nil)

	if result == nil {
		t.Fatal("addNavigationLinks should not return nil for nil data")
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if string(data) != "[]" {
		t.Fatalf("expected [], got %s", string(data))
	}
}

func TestAddNavigationLinksWithEmptySlice(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	result := addNavigationLinks([]interface{}{}, nil, nil, req, "Products", "minimal", nil)

	if result == nil {
		t.Fatal("addNavigationLinks should not return nil for empty slice")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d", len(result))
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if string(data) != "[]" {
		t.Fatalf("expected [], got %s", string(data))
	}
}

func TestAddNavigationLinksWithNonSliceData(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	single := map[string]interface{}{"ID": 1}
	result := addNavigationLinks(single, nil, nil, req, "Products", "minimal", nil)

	if result == nil {
		t.Fatal("addNavigationLinks should return empty slice for non-slice data")
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if string(data) != "[]" {
		t.Fatalf("expected [], got %s", string(data))
	}
}
