package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestAddNavigationLinksWithNilData tests that addNavigationLinks returns empty slice instead of nil
// when data is nil, ensuring OData v4 compliance (empty collections must be [] not null)
func TestAddNavigationLinksWithNilData(t *testing.T) {
	// Create a mock request
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)

	// Test with nil data
	result := addNavigationLinks(nil, nil, nil, req, "Products")

	// Result should not be nil
	if result == nil {
		t.Error("addNavigationLinks should not return nil, should return empty slice")
	}

	// Marshal to JSON to verify it produces []
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	expected := "[]"
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}

// TestAddNavigationLinksWithEmptySlice tests that addNavigationLinks handles empty slices correctly
func TestAddNavigationLinksWithEmptySlice(t *testing.T) {
	// Create a mock request
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)

	// Test with empty slice
	emptySlice := []interface{}{}
	result := addNavigationLinks(emptySlice, nil, nil, req, "Products")

	// Result should be an empty slice
	if result == nil {
		t.Error("addNavigationLinks should not return nil for empty slice")
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got length %d", len(result))
	}

	// Marshal to JSON to verify it produces []
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	expected := "[]"
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}

// TestAddNavigationLinksWithNonSliceData tests that addNavigationLinks returns empty slice for non-slice data
func TestAddNavigationLinksWithNonSliceData(t *testing.T) {
	// Create a mock request
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)

	// Test with non-slice data (e.g., a single object)
	singleObject := map[string]interface{}{"ID": 1, "Name": "Product"}
	result := addNavigationLinks(singleObject, nil, nil, req, "Products")

	// Result should be an empty slice (not nil)
	if result == nil {
		t.Error("addNavigationLinks should not return nil for non-slice data, should return empty slice")
	}

	// Marshal to JSON to verify it produces []
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	expected := "[]"
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s. addNavigationLinks should return [] for non-slice data to maintain OData v4 compliance", expected, string(data))
	}
}

// TestWriteODataCollectionWithNilData tests that WriteODataCollection handles nil data correctly
// per OData v4 specification (empty collections must be [] not null)
func TestWriteODataCollectionWithNilData(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	w := httptest.NewRecorder()

	// Write response with nil data
	err := WriteODataCollection(w, req, "Products", nil, nil, nil)
	if err != nil {
		t.Fatalf("WriteODataCollection failed: %v", err)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check that value is []
	value := response["value"]
	if value == nil {
		t.Error("Response 'value' should be [], not null")
		return
	}

	arr, ok := value.([]interface{})
	if !ok {
		t.Errorf("Response 'value' should be an array, got %T", value)
		return
	}

	if len(arr) != 0 {
		t.Errorf("Response 'value' should be empty array, got length %d", len(arr))
	}
}

// TestWriteODataCollectionWithNavigationWithNilData tests WriteODataCollectionWithNavigation with nil data
func TestWriteODataCollectionWithNavigationWithNilData(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/Products", nil)
	w := httptest.NewRecorder()

	// Write response with nil data (no metadata provider needed for this test)
	err := WriteODataCollectionWithNavigation(w, req, "Products", nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("WriteODataCollectionWithNavigation failed: %v", err)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check that value is []
	value := response["value"]
	if value == nil {
		t.Error("Response 'value' should be [], not null per OData v4 spec")
		return
	}

	arr, ok := value.([]interface{})
	if !ok {
		t.Errorf("Response 'value' should be an array, got %T", value)
		return
	}

	if len(arr) != 0 {
		t.Errorf("Response 'value' should be empty array, got length %d", len(arr))
	}
}
