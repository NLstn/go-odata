package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteODataCollectionWithNilData(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	w := httptest.NewRecorder()

	if err := WriteODataCollection(w, req, "Products", nil, nil, nil); err != nil {
		t.Fatalf("WriteODataCollection failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatalf("expected value to be []interface{}, got %T", response["value"])
	}
	if len(value) != 0 {
		t.Fatalf("expected empty collection, got %d entries", len(value))
	}
}

func TestWriteODataCollectionWithNavigationWithNilData(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	w := httptest.NewRecorder()

	if err := WriteODataCollectionWithNavigation(w, req, "Products", nil, nil, nil, nil, nil, nil, nil, false); err != nil {
		t.Fatalf("WriteODataCollectionWithNavigation failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatalf("expected value to be []interface{}, got %T", response["value"])
	}
	if len(value) != 0 {
		t.Fatalf("expected empty collection, got %d entries", len(value))
	}
}

func TestWriteODataDeltaResponse(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	w := httptest.NewRecorder()

	deltaLink := "http://example.com/Products?$deltatoken=abc"
	entries := []map[string]interface{}{{"ID": 1}}

	if err := WriteODataDeltaResponse(w, req, "Products", entries, &deltaLink); err != nil {
		t.Fatalf("WriteODataDeltaResponse failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if _, ok := body["@odata.context"].(string); !ok {
		t.Fatalf("expected @odata.context in response")
	}
	if body["@odata.deltaLink"] != deltaLink {
		t.Fatalf("expected delta link %s, got %v", deltaLink, body["@odata.deltaLink"])
	}
	value, ok := body["value"].([]interface{})
	if !ok || len(value) != 1 {
		t.Fatalf("expected single entry in delta response")
	}
}
