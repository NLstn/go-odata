package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteEntityReference(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/Products(1)/$ref", nil)
	w := httptest.NewRecorder()

	if err := WriteEntityReference(w, req, "Products(1)"); err != nil {
		t.Fatalf("WriteEntityReference failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body["@odata.id"] != "http://example.com/Products(1)" {
		t.Fatalf("unexpected @odata.id: %v", body["@odata.id"])
	}
	if body["@odata.context"] != "http://example.com/$metadata#$ref" {
		t.Fatalf("unexpected @odata.context: %v", body["@odata.context"])
	}
}

func TestWriteEntityReferenceCollection(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/Products/$ref", nil)
	w := httptest.NewRecorder()

	count := int64(2)
	nextLink := "http://example.com/Products/$ref?$skip=2"
	ids := []string{"Products(1)", "Products(2)"}

	if err := WriteEntityReferenceCollection(w, req, ids, &count, &nextLink); err != nil {
		t.Fatalf("WriteEntityReferenceCollection failed: %v", err)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body["@odata.count"] != float64(count) {
		t.Fatalf("unexpected count: %v", body["@odata.count"])
	}
	if body["@odata.nextLink"] != nextLink {
		t.Fatalf("unexpected next link: %v", body["@odata.nextLink"])
	}

	value, ok := body["value"].([]interface{})
	if !ok || len(value) != len(ids) {
		t.Fatalf("unexpected value: %#v", body["value"])
	}
}
