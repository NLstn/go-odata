package response

import (
	"encoding/json"
	"testing"
)

// TestOrderedMapMarshalJSON tests that OrderedMap maintains field order
func TestOrderedMapMarshalJSON(t *testing.T) {
	om := NewOrderedMap()
	om.Set("first", "value1")
	om.Set("second", 42)
	om.Set("third", true)
	om.Set("fourth", []string{"a", "b", "c"})

	data, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("Failed to marshal OrderedMap: %v", err)
	}

	// Verify order is maintained
	expected := `{"first":"value1","second":42,"third":true,"fourth":["a","b","c"]}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// TestOrderedMapMarshalJSONEmpty tests empty OrderedMap
func TestOrderedMapMarshalJSONEmpty(t *testing.T) {
	om := NewOrderedMap()

	data, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("Failed to marshal empty OrderedMap: %v", err)
	}

	expected := `{}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// TestOrderedMapMarshalJSONWithEscaping tests keys that need escaping
func TestOrderedMapMarshalJSONWithEscaping(t *testing.T) {
	om := NewOrderedMap()
	om.Set("normal", "value1")
	om.Set("key\"with\"quotes", "value2")
	om.Set("key\nwith\nnewlines", "value3")
	om.Set("key\\with\\backslashes", "value4")

	data, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("Failed to marshal OrderedMap with escaping: %v", err)
	}

	// Verify it can be unmarshaled back
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Verify values are preserved
	if result["normal"] != "value1" {
		t.Errorf("Expected 'value1', got %v", result["normal"])
	}
	if result["key\"with\"quotes"] != "value2" {
		t.Errorf("Expected 'value2', got %v", result["key\"with\"quotes"])
	}
	if result["key\nwith\nnewlines"] != "value3" {
		t.Errorf("Expected 'value3', got %v", result["key\nwith\nnewlines"])
	}
	if result["key\\with\\backslashes"] != "value4" {
		t.Errorf("Expected 'value4', got %v", result["key\\with\\backslashes"])
	}
}

// TestOrderedMapMarshalJSONWithODataFields tests OData-specific fields
func TestOrderedMapMarshalJSONWithODataFields(t *testing.T) {
	om := NewOrderedMap()
	om.Set("@odata.context", "$metadata#Products")
	om.Set("@odata.id", "Products(1)")
	om.Set("@odata.etag", "W/\"123\"")
	om.Set("ID", 1)
	om.Set("Name", "Test")

	data, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("Failed to marshal OrderedMap: %v", err)
	}

	// Verify order is maintained (OData annotations should come first)
	expected := `{"@odata.context":"$metadata#Products","@odata.id":"Products(1)","@odata.etag":"W/\"123\"","ID":1,"Name":"Test"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// TestOrderedMapSet tests the Set operation
func TestOrderedMapSet(t *testing.T) {
	om := NewOrderedMap()

	// Test adding new keys
	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	if len(om.keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(om.keys))
	}
	if len(om.values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(om.values))
	}

	// Test updating existing key (should not duplicate)
	om.Set("second", 22)
	if len(om.keys) != 3 {
		t.Errorf("Expected 3 keys after update, got %d", len(om.keys))
	}
	if om.values["second"] != 22 {
		t.Errorf("Expected updated value 22, got %v", om.values["second"])
	}
}

// TestOrderedMapDelete tests the Delete operation
func TestOrderedMapDelete(t *testing.T) {
	om := NewOrderedMap()
	om.Set("first", 1)
	om.Set("second", 2)
	om.Set("third", 3)

	om.Delete("second")

	if len(om.keys) != 2 {
		t.Errorf("Expected 2 keys after delete, got %d", len(om.keys))
	}
	if len(om.values) != 2 {
		t.Errorf("Expected 2 values after delete, got %d", len(om.values))
	}

	// Verify order is maintained
	expectedKeys := []string{"first", "third"}
	for i, key := range om.keys {
		if key != expectedKeys[i] {
			t.Errorf("Expected key %s at position %d, got %s", expectedKeys[i], i, key)
		}
	}
}

// TestOrderedMapInsertAfter tests the InsertAfter operation
func TestOrderedMapInsertAfter(t *testing.T) {
	om := NewOrderedMap()
	om.Set("first", 1)
	om.Set("third", 3)

	om.InsertAfter("first", "second", 2)

	expectedKeys := []string{"first", "second", "third"}
	for i, key := range om.keys {
		if key != expectedKeys[i] {
			t.Errorf("Expected key %s at position %d, got %s", expectedKeys[i], i, key)
		}
	}

	if om.values["second"] != 2 {
		t.Errorf("Expected value 2, got %v", om.values["second"])
	}
}

// TestOrderedMapWithCapacity tests capacity pre-allocation
func TestOrderedMapWithCapacity(t *testing.T) {
	om := NewOrderedMapWithCapacity(20)

	if cap(om.keys) < 20 {
		t.Errorf("Expected keys capacity >= 20, got %d", cap(om.keys))
	}

	// Add items and verify it works
	for i := 0; i < 15; i++ {
		om.Set(string(rune('a'+i)), i)
	}

	if len(om.keys) != 15 {
		t.Errorf("Expected 15 keys, got %d", len(om.keys))
	}
}

// TestNeedsEscaping tests the needsEscaping helper function
func TestNeedsEscaping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"simple", "simple", false},
		{"with_underscore", "field_name", false},
		{"with_at", "@odata.id", false},
		{"with_dot", "odata.context", false},
		{"with_quotes", "field\"name", true},
		{"with_backslash", "field\\name", true},
		{"with_newline", "field\nname", true},
		{"with_tab", "field\tname", true},
		{"with_carriage_return", "field\rname", true},
		{"unicode", "field_名前", false},
		{"emoji", "field_😀", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsEscaping(tt.input)
			if result != tt.expected {
				t.Errorf("needsEscaping(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
