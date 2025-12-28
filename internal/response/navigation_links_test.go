package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

type testMetadataProvider struct{}

func (testMetadataProvider) GetProperties() []PropertyMetadata {
	return []PropertyMetadata{
		{Name: "ID", JsonName: "ID"},
		{Name: "Name", JsonName: "Name"},
		{Name: "Orders", JsonName: "Orders", IsNavigationProp: true},
	}
}

func (testMetadataProvider) GetKeyProperty() *PropertyMetadata {
	return &PropertyMetadata{Name: "ID", JsonName: "ID"}
}

func (testMetadataProvider) GetKeyProperties() []PropertyMetadata {
	return []PropertyMetadata{{Name: "ID", JsonName: "ID"}}
}

func (testMetadataProvider) GetEntitySetName() string {
	return "Products"
}

func (testMetadataProvider) GetETagProperty() *PropertyMetadata {
	return nil
}

func (testMetadataProvider) GetNamespace() string {
	return "Default"
}

func TestAddNavigationLinksWithNilData(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	result := addNavigationLinks(nil, nil, nil, req, "Products", "minimal", nil, nil, false)

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
	result := addNavigationLinks([]interface{}{}, nil, nil, req, "Products", "minimal", nil, nil, false)

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
	result := addNavigationLinks(single, nil, nil, req, "Products", "minimal", nil, nil, false)

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

func TestAddNavigationLinksRespectsSelectProjection(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	data := []map[string]interface{}{
		{
			"ID":   1,
			"Name": "Sample",
		},
	}

	result := addNavigationLinks(data, testMetadataProvider{}, nil, req, "Products", "full", nil, []string{"Name"}, true)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	item, ok := result[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result[0])
	}

	if _, exists := item["Orders@odata.navigationLink"]; exists {
		t.Fatal("unexpected navigation link for unselected property")
	}
}
