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

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", odataVersion)
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

	// Check OData v4 compliant structure
	if errorData["code"] != "404" {
		t.Errorf("error.code = %v, want 404", errorData["code"])
	}

	if errorData["message"] != "Not found" {
		t.Errorf("error.message = %v, want Not found", errorData["message"])
	}

	// Check that details are in the details array
	details, ok := errorData["details"].([]interface{})
	if !ok {
		t.Fatal("error.details is not an array")
	}

	if len(details) != 1 {
		t.Fatalf("len(error.details) = %v, want 1", len(details))
	}

	firstDetail, ok := details[0].(map[string]interface{})
	if !ok {
		t.Fatal("error.details[0] is not an object")
	}

	if firstDetail["message"] != "Entity not found" {
		t.Errorf("error.details[0].message = %v, want Entity not found", firstDetail["message"])
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

func TestWriteODataError_WithTarget(t *testing.T) {
	w := httptest.NewRecorder()

	odataErr := &ODataError{
		Code:    "ValidationError",
		Message: "Validation failed",
		Target:  "Products(1)/Price",
		Details: []ODataErrorDetail{
			{
				Code:    "FieldTooLarge",
				Target:  "Price",
				Message: "Price cannot exceed 10000",
			},
		},
	}

	err := WriteODataError(w, http.StatusBadRequest, odataErr)
	if err != nil {
		t.Fatalf("WriteODataError() error = %v", err)
	}

	// Check status code
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
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

	if errorData["code"] != "ValidationError" {
		t.Errorf("error.code = %v, want ValidationError", errorData["code"])
	}

	if errorData["message"] != "Validation failed" {
		t.Errorf("error.message = %v, want Validation failed", errorData["message"])
	}

	if errorData["target"] != "Products(1)/Price" {
		t.Errorf("error.target = %v, want Products(1)/Price", errorData["target"])
	}

	details, ok := errorData["details"].([]interface{})
	if !ok {
		t.Fatal("error.details is not an array")
	}

	if len(details) != 1 {
		t.Fatalf("len(error.details) = %v, want 1", len(details))
	}

	detail := details[0].(map[string]interface{})
	if detail["code"] != "FieldTooLarge" {
		t.Errorf("error.details[0].code = %v, want FieldTooLarge", detail["code"])
	}

	if detail["target"] != "Price" {
		t.Errorf("error.details[0].target = %v, want Price", detail["target"])
	}
}

func TestWriteODataError_MultipleDetails(t *testing.T) {
	w := httptest.NewRecorder()

	odataErr := &ODataError{
		Code:    "MultipleErrors",
		Message: "Multiple validation errors occurred",
		Details: []ODataErrorDetail{
			{
				Code:    "RequiredField",
				Target:  "Name",
				Message: "Name is required",
			},
			{
				Code:    "InvalidFormat",
				Target:  "Email",
				Message: "Email format is invalid",
			},
			{
				Code:    "OutOfRange",
				Target:  "Age",
				Message: "Age must be between 0 and 150",
			},
		},
	}

	err := WriteODataError(w, http.StatusBadRequest, odataErr)
	if err != nil {
		t.Fatalf("WriteODataError() error = %v", err)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorData := response["error"].(map[string]interface{})
	details := errorData["details"].([]interface{})

	if len(details) != 3 {
		t.Fatalf("len(error.details) = %v, want 3", len(details))
	}

	// Verify first detail
	detail0 := details[0].(map[string]interface{})
	if detail0["code"] != "RequiredField" {
		t.Errorf("details[0].code = %v, want RequiredField", detail0["code"])
	}

	// Verify second detail
	detail1 := details[1].(map[string]interface{})
	if detail1["target"] != "Email" {
		t.Errorf("details[1].target = %v, want Email", detail1["target"])
	}

	// Verify third detail
	detail2 := details[2].(map[string]interface{})
	if detail2["message"] != "Age must be between 0 and 150" {
		t.Errorf("details[2].message = %v, want 'Age must be between 0 and 150'", detail2["message"])
	}
}

func TestWriteODataError_WithInnerError(t *testing.T) {
	w := httptest.NewRecorder()

	odataErr := &ODataError{
		Code:    "InternalError",
		Message: "An internal error occurred",
		InnerError: &ODataInnerError{
			Message:  "Database connection failed",
			TypeName: "System.Data.SqlClient.SqlException",
			InnerError: &ODataInnerError{
				Message:    "Network timeout",
				StackTrace: "at System.Net.Sockets.TcpClient.Connect()\n   at Database.Connect()",
			},
		},
	}

	err := WriteODataError(w, http.StatusInternalServerError, odataErr)
	if err != nil {
		t.Fatalf("WriteODataError() error = %v", err)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorData := response["error"].(map[string]interface{})
	innerError, ok := errorData["innererror"].(map[string]interface{})
	if !ok {
		t.Fatal("error.innererror is not an object")
	}

	if innerError["message"] != "Database connection failed" {
		t.Errorf("innererror.message = %v, want 'Database connection failed'", innerError["message"])
	}

	if innerError["type"] != "System.Data.SqlClient.SqlException" {
		t.Errorf("innererror.type = %v, want System.Data.SqlClient.SqlException", innerError["type"])
	}

	// Check nested inner error
	nestedInner, ok := innerError["innererror"].(map[string]interface{})
	if !ok {
		t.Fatal("innererror.innererror is not an object")
	}

	if nestedInner["message"] != "Network timeout" {
		t.Errorf("nested innererror.message = %v, want 'Network timeout'", nestedInner["message"])
	}

	if !strings.Contains(nestedInner["stacktrace"].(string), "TcpClient.Connect") {
		t.Errorf("nested innererror.stacktrace doesn't contain expected stack trace")
	}
}

func TestWriteODataError_MinimalError(t *testing.T) {
	w := httptest.NewRecorder()

	odataErr := &ODataError{
		Code:    "NotFound",
		Message: "Resource not found",
	}

	err := WriteODataError(w, http.StatusNotFound, odataErr)
	if err != nil {
		t.Fatalf("WriteODataError() error = %v", err)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorData := response["error"].(map[string]interface{})

	// Verify minimal structure
	if errorData["code"] != "NotFound" {
		t.Errorf("error.code = %v, want NotFound", errorData["code"])
	}

	if errorData["message"] != "Resource not found" {
		t.Errorf("error.message = %v, want 'Resource not found'", errorData["message"])
	}

	// Ensure optional fields are omitted
	if _, exists := errorData["target"]; exists {
		t.Error("error.target should be omitted when empty")
	}

	if _, exists := errorData["details"]; exists {
		t.Error("error.details should be omitted when empty")
	}

	if _, exists := errorData["innererror"]; exists {
		t.Error("error.innererror should be omitted when nil")
	}
}

func TestWriteErrorWithTarget(t *testing.T) {
	w := httptest.NewRecorder()

	err := WriteErrorWithTarget(w, http.StatusNotFound, "Entity not found", "Products(999)", "The specified entity does not exist")
	if err != nil {
		t.Fatalf("WriteErrorWithTarget() error = %v", err)
	}

	// Check status code
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorData := response["error"].(map[string]interface{})

	if errorData["code"] != "404" {
		t.Errorf("error.code = %v, want 404", errorData["code"])
	}

	if errorData["message"] != "Entity not found" {
		t.Errorf("error.message = %v, want 'Entity not found'", errorData["message"])
	}

	if errorData["target"] != "Products(999)" {
		t.Errorf("error.target = %v, want Products(999)", errorData["target"])
	}

	details := errorData["details"].([]interface{})
	if len(details) != 1 {
		t.Fatalf("len(error.details) = %v, want 1", len(details))
	}

	detail := details[0].(map[string]interface{})
	if detail["message"] != "The specified entity does not exist" {
		t.Errorf("error.details[0].message = %v, want 'The specified entity does not exist'", detail["message"])
	}

	if detail["target"] != "Products(999)" {
		t.Errorf("error.details[0].target = %v, want Products(999)", detail["target"])
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
	result := processStructEntityOrdered(entityValue, metadata, []string{}, "http://localhost:8080", "Products")

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
	keyProps      []PropertyMetadata
	entitySetName string
}

func (m *mockMetadata) GetProperties() []PropertyMetadata {
	return m.props
}

func (m *mockMetadata) GetKeyProperty() *PropertyMetadata {
	return m.keyProp
}

func (m *mockMetadata) GetKeyProperties() []PropertyMetadata {
	if m.keyProps != nil {
		return m.keyProps
	}
	// For backwards compatibility, if keyProps is not set but keyProp is, return it as a slice
	if m.keyProp != nil {
		return []PropertyMetadata{*m.keyProp}
	}
	return nil
}

func (m *mockMetadata) GetEntitySetName() string {
	return m.entitySetName
}
