package response

import (
	"fmt"
	"reflect"
	"testing"
)

func TestExtractJsonFieldName(t *testing.T) {
	tests := []struct {
		name     string
		field    reflect.StructField
		expected string
	}{
		{
			name: "No JSON tag",
			field: reflect.StructField{
				Name: "FieldName",
				Tag:  "",
			},
			expected: "FieldName",
		},
		{
			name: "JSON tag with name only",
			field: reflect.StructField{
				Name: "FieldName",
				Tag:  `json:"fieldName"`,
			},
			expected: "fieldName",
		},
		{
			name: "JSON tag with omitempty",
			field: reflect.StructField{
				Name: "FieldName",
				Tag:  `json:"fieldName,omitempty"`,
			},
			expected: "fieldName",
		},
		{
			name: "JSON tag with multiple options",
			field: reflect.StructField{
				Name: "FieldName",
				Tag:  `json:"fieldName,omitempty,string"`,
			},
			expected: "fieldName",
		},
		{
			name: "JSON tag with dash (ignore field)",
			field: reflect.StructField{
				Name: "FieldName",
				Tag:  `json:"-"`,
			},
			expected: "-",
		},
		{
			name: "JSON tag with comma only",
			field: reflect.StructField{
				Name: "FieldName",
				Tag:  `json:",omitempty"`,
			},
			expected: "FieldName",
		},
		{
			name: "Empty JSON tag",
			field: reflect.StructField{
				Name: "FieldName",
				Tag:  `json:""`,
			},
			expected: "FieldName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJsonFieldName(tt.field)
			if result != tt.expected {
				t.Errorf("extractJsonFieldName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetFieldInfos(t *testing.T) {
	type TestStruct struct {
		ID          int    `json:"id"`
		Name        string `json:"name,omitempty"`
		Description string
		//nolint:unused // unexported field is intentionally included for testing field info extraction
		unexported string
	}

	testType := reflect.TypeOf(TestStruct{})

	// First call should cache the field information
	infos := getFieldInfos(testType)

	if len(infos) != 4 {
		t.Errorf("Expected 4 field infos, got %d", len(infos))
	}

	// Verify field info for each field
	expectedFields := []struct {
		jsonName   string
		isExported bool
	}{
		{"id", true},
		{"name", true},
		{"Description", true},
		{"unexported", false},
	}

	for i, expected := range expectedFields {
		if infos[i].JsonName != expected.jsonName {
			t.Errorf("Field %d: JsonName = %q, want %q", i, infos[i].JsonName, expected.jsonName)
		}
		if infos[i].IsExported != expected.isExported {
			t.Errorf("Field %d: IsExported = %v, want %v", i, infos[i].IsExported, expected.isExported)
		}
	}

	// Second call should return cached result
	infos2 := getFieldInfos(testType)
	if len(infos2) != len(infos) {
		t.Error("Cached result has different length")
	}

	// Test with different struct type
	type AnotherStruct struct {
		Value int `json:"value"`
	}
	anotherType := reflect.TypeOf(AnotherStruct{})
	infos3 := getFieldInfos(anotherType)

	if len(infos3) != 1 {
		t.Errorf("Expected 1 field info for AnotherStruct, got %d", len(infos3))
	}
	if infos3[0].JsonName != "value" {
		t.Errorf("JsonName = %q, want 'value'", infos3[0].JsonName)
	}
}

func TestGetFieldInfosConcurrency(t *testing.T) {
	type TestStruct struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	testType := reflect.TypeOf(TestStruct{})

	// Test concurrent access to the cache
	errors := make(chan error, 10)
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			infos := getFieldInfos(testType)
			if len(infos) != 2 {
				errors <- fmt.Errorf("Expected 2 field infos, got %d", len(infos))
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check for any errors
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

// MockEntityMetadataProvider implements EntityMetadataProvider for testing
type MockEntityMetadataProvider struct {
	namespace     string
	properties    []PropertyMetadata
	entitySetName string
	keyProperty   *PropertyMetadata
	keyProperties []PropertyMetadata
	etagProperty  *PropertyMetadata
}

func (m *MockEntityMetadataProvider) GetNamespace() string {
	return m.namespace
}

func (m *MockEntityMetadataProvider) GetProperties() []PropertyMetadata {
	return m.properties
}

func (m *MockEntityMetadataProvider) GetKeyProperty() *PropertyMetadata {
	return m.keyProperty
}

func (m *MockEntityMetadataProvider) GetKeyProperties() []PropertyMetadata {
	return m.keyProperties
}

func (m *MockEntityMetadataProvider) GetEntitySetName() string {
	return m.entitySetName
}

func (m *MockEntityMetadataProvider) GetETagProperty() *PropertyMetadata {
	return m.etagProperty
}

func TestGetCachedPropertyMetadataMap(t *testing.T) {
	// Create a mock metadata provider
	mockProvider := &MockEntityMetadataProvider{
		namespace: "Test.Namespace",
		properties: []PropertyMetadata{
			{Name: "ID", JsonName: "id", IsNavigationProp: false},
			{Name: "Name", JsonName: "name", IsNavigationProp: false},
			{Name: "Category", JsonName: "category", IsNavigationProp: true},
		},
	}

	// First call should build the cache
	propMap := getCachedPropertyMetadataMap(mockProvider)

	if len(propMap) != 3 {
		t.Errorf("Expected 3 properties in map, got %d", len(propMap))
	}

	// Verify properties are correctly mapped
	if prop, ok := propMap["ID"]; !ok || prop.JsonName != "id" {
		t.Error("ID property not correctly cached")
	}
	if prop, ok := propMap["Name"]; !ok || prop.JsonName != "name" {
		t.Error("Name property not correctly cached")
	}
	if prop, ok := propMap["Category"]; !ok || prop.JsonName != "category" {
		t.Error("Category property not correctly cached")
	}

	// Second call should return cached result
	propMap2 := getCachedPropertyMetadataMap(mockProvider)
	if len(propMap2) != len(propMap) {
		t.Error("Cached result has different length")
	}

	// Create a different mock provider
	mockProvider2 := &MockEntityMetadataProvider{
		namespace: "Test.Namespace2",
		properties: []PropertyMetadata{
			{Name: "Value", JsonName: "value", IsNavigationProp: false},
		},
	}

	propMap3 := getCachedPropertyMetadataMap(mockProvider2)
	if len(propMap3) != 1 {
		t.Errorf("Expected 1 property in map for second provider, got %d", len(propMap3))
	}
	if prop, ok := propMap3["Value"]; !ok || prop.JsonName != "value" {
		t.Error("Value property not correctly cached for second provider")
	}
}

func TestGetCachedPropertyMetadataMapConcurrency(t *testing.T) {
	mockProvider := &MockEntityMetadataProvider{
		namespace: "Test.Namespace",
		properties: []PropertyMetadata{
			{Name: "ID", JsonName: "id", IsNavigationProp: false},
			{Name: "Name", JsonName: "name", IsNavigationProp: false},
		},
	}

	// Test concurrent access to the cache
	errors := make(chan error, 10)
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			propMap := getCachedPropertyMetadataMap(mockProvider)
			if len(propMap) != 2 {
				errors <- fmt.Errorf("Expected 2 properties in map, got %d", len(propMap))
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check for any errors
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}
