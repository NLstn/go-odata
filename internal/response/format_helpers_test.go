package response

import (
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

func TestBuildEntityIDSingleStringKey(t *testing.T) {
	keyValues := map[string]interface{}{"ID": "ALFKI"}
	if id := BuildEntityID("Customers", keyValues); id != "Customers('ALFKI')" {
		t.Fatalf("expected Customers('ALFKI'), got %s", id)
	}
}

func TestBuildEntityIDCompositeOrdering(t *testing.T) {
	keyValues := map[string]interface{}{"LanguageKey": "EN", "ProductID": 1}
	expected := "ProductDescriptions(LanguageKey='EN',ProductID=1)"
	if id := BuildEntityID("ProductDescriptions", keyValues); id != expected {
		t.Fatalf("expected %s, got %s", expected, id)
	}
}

func TestBuildEntityIDSingleIntKey(t *testing.T) {
	keyValues := map[string]interface{}{"ID": 42}
	if id := BuildEntityID("Products", keyValues); id != "Products(42)" {
		t.Fatalf("expected Products(42), got %s", id)
	}
}

func TestBuildEntityIDStringWithQuotes(t *testing.T) {
	keyValues := map[string]interface{}{"Name": "O'Brien"}
	if id := BuildEntityID("People", keyValues); id != "People('O''Brien')" {
		t.Fatalf("expected People('O''Brien'), got %s", id)
	}
}

func TestFormatKeyValueLiteral(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"String", "test", "'test'"},
		{"String with single quote", "O'Brien", "'O''Brien'"},
		{"Integer", 42, "42"},
		{"Float", 3.14, "3.14"},
		{"Boolean", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatKeyValueLiteral(tt.value)
			if result != tt.expected {
				t.Errorf("formatKeyValueLiteral(%v) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestExtractEntityKeys(t *testing.T) {
	type TestEntity struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	entity := TestEntity{ID: 123, Name: "Test"}
	keyProperties := []metadata.PropertyMetadata{
		{Name: "ID", JsonName: "id"},
	}

	keyValues := ExtractEntityKeys(entity, keyProperties)

	if len(keyValues) != 1 {
		t.Fatalf("Expected 1 key value, got %d", len(keyValues))
	}

	if keyValues["id"] != 123 {
		t.Errorf("Expected ID=123, got %v", keyValues["id"])
	}
}

func TestExtractEntityKeysPointer(t *testing.T) {
	type TestEntity struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	entity := &TestEntity{ID: 456, Name: "Test"}
	keyProperties := []metadata.PropertyMetadata{
		{Name: "ID", JsonName: "id"},
	}

	keyValues := ExtractEntityKeys(entity, keyProperties)

	if len(keyValues) != 1 {
		t.Fatalf("Expected 1 key value, got %d", len(keyValues))
	}

	if keyValues["id"] != 456 {
		t.Errorf("Expected ID=456, got %v", keyValues["id"])
	}
}

func TestExtractEntityKeysComposite(t *testing.T) {
	type TestEntity struct {
		ProductID   int    `json:"productId"`
		LanguageKey string `json:"languageKey"`
		Description string `json:"description"`
	}

	entity := TestEntity{ProductID: 1, LanguageKey: "EN", Description: "Test"}
	keyProperties := []metadata.PropertyMetadata{
		{Name: "ProductID", JsonName: "productId"},
		{Name: "LanguageKey", JsonName: "languageKey"},
	}

	keyValues := ExtractEntityKeys(entity, keyProperties)

	if len(keyValues) != 2 {
		t.Fatalf("Expected 2 key values, got %d", len(keyValues))
	}

	if keyValues["productId"] != 1 {
		t.Errorf("Expected ProductID=1, got %v", keyValues["productId"])
	}

	if keyValues["languageKey"] != "EN" {
		t.Errorf("Expected LanguageKey=EN, got %v", keyValues["languageKey"])
	}
}

func TestExtractEntityKeysInvalidField(t *testing.T) {
	type TestEntity struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	entity := TestEntity{ID: 123, Name: "Test"}
	keyProperties := []metadata.PropertyMetadata{
		{Name: "NonExistentField", JsonName: "nonexistent"},
	}

	keyValues := ExtractEntityKeys(entity, keyProperties)

	// The function should still work but the key won't be found
	// Check that it doesn't panic
	if keyValues == nil {
		t.Error("ExtractEntityKeys returned nil")
	}
}

func TestExtractEntityKeysEmptyKeyProperties(t *testing.T) {
	type TestEntity struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	entity := TestEntity{ID: 123, Name: "Test"}
	keyProperties := []metadata.PropertyMetadata{}

	keyValues := ExtractEntityKeys(entity, keyProperties)

	if len(keyValues) != 0 {
		t.Errorf("Expected 0 key values, got %d", len(keyValues))
	}
}

func TestExtractEntityKeysNilEntity(t *testing.T) {
	keyProperties := []metadata.PropertyMetadata{
		{Name: "ID", JsonName: "id"},
	}

	// Test with nil value - should panic (expected behavior)
	defer func() {
		if r := recover(); r == nil {
			t.Error("ExtractEntityKeys should panic with nil entity")
		}
	}()

	ExtractEntityKeys(nil, keyProperties)
}

func TestExtractEntityKeysReflection(t *testing.T) {
	type TestEntity struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Private string // no json tag, private
	}

	entity := TestEntity{ID: 789, Name: "Reflection Test", Private: "private"}
	keyProperties := []metadata.PropertyMetadata{
		{Name: "ID", JsonName: "id"},
		{Name: "Name", JsonName: "name"},
	}

	keyValues := ExtractEntityKeys(entity, keyProperties)

	if len(keyValues) != 2 {
		t.Fatalf("Expected 2 key values, got %d", len(keyValues))
	}

	// Verify values are correctly extracted
	idVal := keyValues["id"]
	if !reflect.DeepEqual(idVal, 789) {
		t.Errorf("ID key value mismatch: got %v, want 789", idVal)
	}

	nameVal := keyValues["name"]
	if !reflect.DeepEqual(nameVal, "Reflection Test") {
		t.Errorf("Name key value mismatch: got %v, want 'Reflection Test'", nameVal)
	}
}
