package response

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type TestEntity struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TestProduct mimics a real entity with multiple fields for ordering tests
type TestProduct struct {
	ID          uint    `json:"ID"`
	Name        string  `json:"Name"`
	Price       float64 `json:"Price"`
	Category    string  `json:"Category"`
	Description string  `json:"Description"`
}

// TestEntityWithNav is a test entity with navigation properties
type TestEntityWithNav struct {
	ID       uint        `json:"ID"`
	Title    string      `json:"Title"`
	AuthorID uint        `json:"AuthorID"`
	Author   *TestAuthor `json:"Author,omitempty"`
	Tags     []TestTag   `json:"Tags"`
}

type TestAuthor struct {
	ID   uint   `json:"ID"`
	Name string `json:"Name"`
}

type TestTag struct {
	ID   uint   `json:"ID"`
	Name string `json:"Name"`
}

func TestWriteODataCollection(t *testing.T) {
	data := []TestEntity{
		{ID: 1, Name: "Test 1"},
		{ID: 2, Name: "Test 2"},
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/Products", nil)
	w := httptest.NewRecorder()

	err := WriteODataCollection(w, req, "Products", data, nil, nil)
	if err != nil {
		t.Fatalf("WriteODataCollection() error = %v", err)
	}

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Check headers
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", odataVersion)
	}

	// Check response body
	var response ODataResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Context != "http://localhost:8080/$metadata#Products" {
		t.Errorf("Context = %v, want http://localhost:8080/$metadata#Products", response.Context)
	}

	// Check value array
	valueArray, ok := response.Value.([]interface{})
	if !ok {
		t.Fatal("Value is not an array")
	}

	if len(valueArray) != 2 {
		t.Errorf("len(Value) = %v, want 2", len(valueArray))
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	err := WriteError(w, http.StatusNotFound, "Not found", "Entity not found")
	if err != nil {
		t.Fatalf("WriteError() error = %v", err)
	}

	// Check status code
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	// Check headers
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorData, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error field is not an object")
	}

	if errorData["message"] != "Not found" {
		t.Errorf("error.message = %v, want Not found", errorData["message"])
	}

	if errorData["details"] != "Entity not found" {
		t.Errorf("error.details = %v, want Entity not found", errorData["details"])
	}
}

func TestWriteServiceDocument(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	w := httptest.NewRecorder()

	entitySets := []string{"Products", "Categories"}

	err := WriteServiceDocument(w, req, entitySets)
	if err != nil {
		t.Fatalf("WriteServiceDocument() error = %v", err)
	}

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["@odata.context"] != "http://localhost:8080/$metadata" {
		t.Errorf("@odata.context = %v, want http://localhost:8080/$metadata", response["@odata.context"])
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 2 {
		t.Errorf("len(value) = %v, want 2", len(value))
	}
}

func TestBuildBaseURL(t *testing.T) {
	tests := []struct {
		name string
		req  *http.Request
		want string
	}{
		{
			name: "http request",
			req: &http.Request{
				Host:   "localhost:8080",
				Header: http.Header{},
			},
			want: "http://localhost:8080",
		},
		{
			name: "https request",
			req: &http.Request{
				Host:   "example.com",
				TLS:    &tls.ConnectionState{},
				Header: http.Header{},
			},
			want: "https://example.com",
		},
		{
			name: "request with X-Forwarded-Proto",
			req: &http.Request{
				Host: "example.com",
				Header: http.Header{
					"X-Forwarded-Proto": []string{"https"},
				},
			},
			want: "https://example.com",
		},
		{
			name: "request without host",
			req: &http.Request{
				Header: http.Header{},
			},
			want: "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildBaseURL(tt.req)
			if got != tt.want {
				t.Errorf("BuildBaseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseODataURL(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantEntitySet string
		wantEntityKey string
		wantErr       bool
	}{
		{
			name:          "collection",
			path:          "Products",
			wantEntitySet: "Products",
			wantEntityKey: "",
			wantErr:       false,
		},
		{
			name:          "entity with key",
			path:          "Products(1)",
			wantEntitySet: "Products",
			wantEntityKey: "1",
			wantErr:       false,
		},
		{
			name:          "entity with string key",
			path:          "Products('ABC')",
			wantEntitySet: "Products",
			wantEntityKey: "'ABC'",
			wantErr:       false,
		},
		{
			name:          "path with leading slash",
			path:          "/Products",
			wantEntitySet: "Products",
			wantEntityKey: "",
			wantErr:       false,
		},
		{
			name:          "empty path",
			path:          "",
			wantEntitySet: "",
			wantEntityKey: "",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEntitySet, gotEntityKey, err := ParseODataURL(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseODataURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotEntitySet != tt.wantEntitySet {
				t.Errorf("ParseODataURL() entitySet = %v, want %v", gotEntitySet, tt.wantEntitySet)
			}
			if gotEntityKey != tt.wantEntityKey {
				t.Errorf("ParseODataURL() entityKey = %v, want %v", gotEntityKey, tt.wantEntityKey)
			}
		})
	}
}

func TestOrderedMap_MarshalJSON(t *testing.T) {
	om := NewOrderedMap()
	om.Set("@odata.context", "http://example.com/$metadata#Products")
	om.Set("ID", 1)
	om.Set("Name", "Test Product")
	om.Set("Price", 99.99)
	om.Set("Category", "Electronics")

	data, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	jsonStr := string(data)

	// Verify the JSON is valid
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify all fields are present
	if result["@odata.context"] != "http://example.com/$metadata#Products" {
		t.Errorf("@odata.context = %v, want http://example.com/$metadata#Products", result["@odata.context"])
	}
	if result["ID"] != float64(1) { // JSON numbers are float64
		t.Errorf("ID = %v, want 1", result["ID"])
	}
	if result["Name"] != "Test Product" {
		t.Errorf("Name = %v, want Test Product", result["Name"])
	}

	// Verify field order by checking the JSON string
	// The fields should appear in the order they were added
	contextIdx := strings.Index(jsonStr, `"@odata.context"`)
	idIdx := strings.Index(jsonStr, `"ID"`)
	nameIdx := strings.Index(jsonStr, `"Name"`)
	priceIdx := strings.Index(jsonStr, `"Price"`)
	categoryIdx := strings.Index(jsonStr, `"Category"`)

	if contextIdx == -1 || idIdx == -1 || nameIdx == -1 || priceIdx == -1 || categoryIdx == -1 {
		t.Fatal("Not all fields found in JSON")
	}

	// Check ordering
	if contextIdx >= idIdx || idIdx >= nameIdx || nameIdx >= priceIdx || priceIdx >= categoryIdx {
		t.Errorf("Fields are not in the correct order. Indices: context=%d, ID=%d, Name=%d, Price=%d, Category=%d",
			contextIdx, idIdx, nameIdx, priceIdx, categoryIdx)
	}
}

func TestOrderedMap_SetAndToMap(t *testing.T) {
	om := NewOrderedMap()
	om.Set("field1", "value1")
	om.Set("field2", 123)
	om.Set("field3", true)

	m := om.ToMap()

	if len(m) != 3 {
		t.Errorf("ToMap() returned map with %d entries, want 3", len(m))
	}

	if m["field1"] != "value1" {
		t.Errorf("field1 = %v, want value1", m["field1"])
	}
	if m["field2"] != 123 {
		t.Errorf("field2 = %v, want 123", m["field2"])
	}
	if m["field3"] != true {
		t.Errorf("field3 = %v, want true", m["field3"])
	}
}

func TestOrderedMap_UpdateExistingKey(t *testing.T) {
	om := NewOrderedMap()
	om.Set("key1", "value1")
	om.Set("key2", "value2")
	om.Set("key1", "updated")

	// Verify the value was updated
	m := om.ToMap()
	if m["key1"] != "updated" {
		t.Errorf("key1 = %v, want updated", m["key1"])
	}

	// Verify the key appears only once and in the original position
	data, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	jsonStr := string(data)
	key1Idx := strings.Index(jsonStr, `"key1"`)
	key2Idx := strings.Index(jsonStr, `"key2"`)

	// key1 should still come before key2
	if key1Idx >= key2Idx {
		t.Errorf("key1 should appear before key2 in JSON, but key1Idx=%d, key2Idx=%d", key1Idx, key2Idx)
	}

	// Verify key1 appears only once
	if strings.Count(jsonStr, `"key1"`) != 1 {
		t.Errorf("key1 appears %d times, want 1", strings.Count(jsonStr, `"key1"`))
	}
}

func TestProcessStructEntityOrdered_FieldOrder(t *testing.T) {
	// Create a test entity
	entity := TestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Category:    "Electronics",
		Description: "A high-performance laptop",
	}

	// Create mock metadata
	metadata := &mockMetadata{
		props: []PropertyMetadata{
			{Name: "ID", JsonName: "ID", IsNavigationProp: false},
			{Name: "Name", JsonName: "Name", IsNavigationProp: false},
			{Name: "Price", JsonName: "Price", IsNavigationProp: false},
			{Name: "Category", JsonName: "Category", IsNavigationProp: false},
			{Name: "Description", JsonName: "Description", IsNavigationProp: false},
		},
		keyProp: &PropertyMetadata{Name: "ID", JsonName: "ID"},
	}

	entityValue := reflect.ValueOf(entity)
	result := processStructEntityOrdered(entityValue, metadata, []string{}, "http://localhost:8080", "Products", metadata.GetKeyProperty())

	// Marshal to JSON to check order
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	jsonStr := string(data)

	// Check that fields appear in struct order
	idIdx := strings.Index(jsonStr, `"ID"`)
	nameIdx := strings.Index(jsonStr, `"Name"`)
	priceIdx := strings.Index(jsonStr, `"Price"`)
	categoryIdx := strings.Index(jsonStr, `"Category"`)
	descIdx := strings.Index(jsonStr, `"Description"`)

	if idIdx >= nameIdx || nameIdx >= priceIdx || priceIdx >= categoryIdx || categoryIdx >= descIdx {
		t.Errorf("Fields are not in struct order. Indices: ID=%d, Name=%d, Price=%d, Category=%d, Description=%d",
			idIdx, nameIdx, priceIdx, categoryIdx, descIdx)
	}
}

func TestWriteODataCollectionWithNavigation_FieldOrder(t *testing.T) {
	// Create test data with specific field order
	data := []TestProduct{
		{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics", Description: "High-performance"},
		{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics", Description: "Wireless"},
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/Products", nil)
	w := httptest.NewRecorder()

	metadata := &mockMetadata{
		props: []PropertyMetadata{
			{Name: "ID", JsonName: "ID", IsNavigationProp: false},
			{Name: "Name", JsonName: "Name", IsNavigationProp: false},
			{Name: "Price", JsonName: "Price", IsNavigationProp: false},
			{Name: "Category", JsonName: "Category", IsNavigationProp: false},
			{Name: "Description", JsonName: "Description", IsNavigationProp: false},
		},
		keyProp:       &PropertyMetadata{Name: "ID", JsonName: "ID"},
		entitySetName: "Products",
	}

	err := WriteODataCollectionWithNavigation(w, req, "Products", data, nil, nil, metadata, []string{})
	if err != nil {
		t.Fatalf("WriteODataCollectionWithNavigation() error = %v", err)
	}

	// Parse the response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Get the raw JSON to check field order
	w2 := httptest.NewRecorder()
	if err := WriteODataCollectionWithNavigation(w2, req, "Products", data, nil, nil, metadata, []string{}); err != nil {
		t.Fatalf("WriteODataCollectionWithNavigation() error = %v", err)
	}

	jsonStr := w2.Body.String()

	// Find the first entity in the value array and check field order
	// Look for the pattern after "value": [
	valueStart := strings.Index(jsonStr, `"value": [`)
	if valueStart == -1 {
		t.Fatal("Could not find 'value' array in JSON")
	}

	// Get a substring starting from the first entity
	entityJSON := jsonStr[valueStart:]

	// Find field indices within the first entity
	idIdx := strings.Index(entityJSON, `"ID"`)
	nameIdx := strings.Index(entityJSON, `"Name"`)
	priceIdx := strings.Index(entityJSON, `"Price"`)
	categoryIdx := strings.Index(entityJSON, `"Category"`)
	descIdx := strings.Index(entityJSON, `"Description"`)

	if idIdx == -1 || nameIdx == -1 || priceIdx == -1 || categoryIdx == -1 || descIdx == -1 {
		t.Fatal("Not all fields found in JSON")
	}

	// Check ordering matches struct field order
	if idIdx >= nameIdx || nameIdx >= priceIdx || priceIdx >= categoryIdx || categoryIdx >= descIdx {
		t.Errorf("Fields in collection are not in struct order. Indices: ID=%d, Name=%d, Price=%d, Category=%d, Description=%d",
			idIdx, nameIdx, priceIdx, categoryIdx, descIdx)
	}
}

// mockMetadata is a mock implementation of EntityMetadataProvider for testing
type mockMetadata struct {
	props         []PropertyMetadata
	keyProp       *PropertyMetadata
	entitySetName string
}

func (m *mockMetadata) GetProperties() []PropertyMetadata {
	return m.props
}

func (m *mockMetadata) GetKeyProperty() *PropertyMetadata {
	return m.keyProp
}

func (m *mockMetadata) GetEntitySetName() string {
	return m.entitySetName
}
