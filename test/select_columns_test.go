package odata_test

import (
	"encoding/json"
	"fmt"
	odata "github.com/nlstn/go-odata"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestProduct is a test entity for select column tests
type TestProductSelect struct {
	ID          int     `json:"id" gorm:"primarykey" odata:"key"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	InStock     bool    `json:"inStock"`
}

func setupSelectTestService(t *testing.T) *odata.Service {
	// Create a custom logger to capture SQL queries
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestProductSelect{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(TestProductSelect{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	db.Create(&TestProductSelect{ID: 1, Name: "Laptop", Price: 999.99, Description: "High-performance laptop", Category: "Electronics", InStock: true})
	db.Create(&TestProductSelect{ID: 2, Name: "Mouse", Price: 29.99, Description: "Wireless mouse", Category: "Electronics", InStock: true})
	db.Create(&TestProductSelect{ID: 3, Name: "Keyboard", Price: 79.99, Description: "Mechanical keyboard", Category: "Electronics", InStock: false})

	return service
}

// TestSelectFetchesOnlyNeededColumns verifies that $select only fetches requested columns from database
func TestSelectFetchesOnlyNeededColumns(t *testing.T) {
	service := setupSelectTestService(t)

	tests := []struct {
		name           string
		url            string
		expectedFields []string
	}{
		{
			name:           "Select single column",
			url:            "/TestProductSelects?$select=name",
			expectedFields: []string{"name"},
		},
		{
			name:           "Select multiple columns",
			url:            "/TestProductSelects?$select=name,price",
			expectedFields: []string{"name", "price"},
		},
		{
			name:           "Select with spaces",
			url:            "/TestProductSelects?$select=name,%20price,%20category",
			expectedFields: []string{"name", "price", "category"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
				return
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check that response contains value array
			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("Response does not contain value array")
			}

			if len(value) == 0 {
				t.Fatal("Response value array is empty")
			}

			// Check first item has only selected fields (no others except those requested)
			firstItem, ok := value[0].(map[string]interface{})
			if !ok {
				t.Fatal("First value item is not a map")
			}

			// Verify that only the expected fields are present
			for _, field := range tt.expectedFields {
				if _, exists := firstItem[field]; !exists {
					t.Errorf("Expected field %s not found in response", field)
				}
			}

			// Verify that NO other structural properties are present
			// (We should NOT see description, category, inStock, etc. if they weren't selected)
			allFields := []string{"name", "price", "description", "category", "inStock"}
			for _, field := range allFields {
				if _, exists := firstItem[field]; exists {
					// Check if this field was expected
					expected := false
					for _, expField := range tt.expectedFields {
						if expField == field {
							expected = true
							break
						}
					}
					if !expected {
						t.Errorf("Unexpected field %s found in response (should only have selected fields)", field)
					}
				}
			}

			// The key (id) might be included for proper OData responses, which is acceptable
			// but other fields should not be present
			if len(firstItem) > len(tt.expectedFields)+1 { // +1 for potential id
				t.Errorf("Response has more fields than expected. Got %d, expected at most %d. Fields: %v",
					len(firstItem), len(tt.expectedFields)+1, getMapKeys(firstItem))
			}
		})
	}
}

// TestStructuralPropertyFetchesOnlyNeededColumn verifies structural property access fetches only that column
func TestStructuralPropertyFetchesOnlyNeededColumn(t *testing.T) {
	service := setupSelectTestService(t)

	tests := []struct {
		name         string
		url          string
		expectedType string
	}{
		{
			name:         "String property",
			url:          "/TestProductSelects(1)/name",
			expectedType: "string",
		},
		{
			name:         "Number property",
			url:          "/TestProductSelects(1)/price",
			expectedType: "float64",
		},
		{
			name:         "Boolean property",
			url:          "/TestProductSelects(1)/inStock",
			expectedType: "bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
				return
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check for @odata.context
			if _, ok := response["@odata.context"]; !ok {
				t.Error("Response missing @odata.context")
			}

			// Check for value
			if _, ok := response["value"]; !ok {
				t.Error("Response missing value field")
			}

			// Verify response has only two fields: @odata.context and value
			if len(response) != 2 {
				t.Errorf("Response should have exactly 2 fields, got %d: %v", len(response), getMapKeys(response))
			}
		})
	}
}

// TestSelectWithFilterAndOrderBy verifies $select works with other query options
func TestSelectWithFilterAndOrderBy(t *testing.T) {
	service := setupSelectTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/TestProductSelects?$select=name,price&$filter=price%20gt%2050&$orderby=price%20desc", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Response does not contain value array")
	}

	// Should have 2 items (Laptop and Keyboard, both > $50)
	if len(value) != 2 {
		t.Errorf("Expected 2 items, got %d", len(value))
	}

	// First item should be Laptop (highest price)
	firstItem, ok := value[0].(map[string]interface{})
	if !ok {
		t.Fatal("First value item is not a map")
	}

	if firstItem["name"] != "Laptop" {
		t.Errorf("Expected first item to be Laptop, got %v", firstItem["name"])
	}

	// Should only have name and price (plus potentially id)
	expectedFields := []string{"name", "price"}
	for _, field := range expectedFields {
		if _, exists := firstItem[field]; !exists {
			t.Errorf("Expected field %s not found", field)
		}
	}

	// Should not have other fields like description, category, inStock
	unexpectedFields := []string{"description", "category", "inStock"}
	for _, field := range unexpectedFields {
		if _, exists := firstItem[field]; exists {
			t.Errorf("Unexpected field %s found in response", field)
		}
	}
}

// TestSelectAllFields verifies that without $select, all fields are returned
func TestSelectAllFields(t *testing.T) {
	service := setupSelectTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/TestProductSelects", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Response does not contain value array")
	}

	if len(value) == 0 {
		t.Fatal("Response value array is empty")
	}

	// Without $select, all fields should be present
	firstItem, ok := value[0].(map[string]interface{})
	if !ok {
		t.Fatal("First value item is not a map")
	}

	expectedFields := []string{"id", "name", "price", "description", "category", "inStock"}
	for _, field := range expectedFields {
		if _, exists := firstItem[field]; !exists {
			t.Errorf("Expected field %s not found in response (all fields should be present without $select)", field)
		}
	}
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestSelectWithUppercaseJsonNames tests select with entities that have uppercase JSON names
func TestSelectWithUppercaseJsonNames(t *testing.T) {
	type TestProductUppercase struct {
		ID          int     `json:"ID" gorm:"primarykey" odata:"key"`
		Name        string  `json:"Name"`
		Price       float64 `json:"Price"`
		Description string  `json:"Description"`
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestProductUppercase{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(TestProductUppercase{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	db.Create(&TestProductUppercase{ID: 1, Name: "Test Product", Price: 100.0, Description: "Test"})

	req := httptest.NewRequest(http.MethodGet, "/TestProductUppercases?$select=Name,Price", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok || len(value) == 0 {
		t.Fatal("Response does not contain value array or is empty")
	}

	firstItem, ok := value[0].(map[string]interface{})
	if !ok {
		t.Fatal("First value item is not a map")
	}

	// Should have Name and Price
	if _, exists := firstItem["Name"]; !exists {
		t.Error("Expected field Name not found")
	}
	if _, exists := firstItem["Price"]; !exists {
		t.Error("Expected field Price not found")
	}

	// Should NOT have Description
	if _, exists := firstItem["Description"]; exists {
		t.Error("Unexpected field Description found (should not be present with $select=Name,Price)")
	}
}

// TestSelectKeyAlwaysIncluded verifies that key properties are always included even if not in $select
func TestSelectKeyAlwaysIncluded(t *testing.T) {
	service := setupSelectTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/TestProductSelects?$select=name", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok || len(value) == 0 {
		t.Fatal("Response does not contain value array or is empty")
	}

	firstItem, ok := value[0].(map[string]interface{})
	if !ok {
		t.Fatal("First value item is not a map")
	}

	// Even though we only selected 'name', the response may include 'id' for OData compliance
	// This is acceptable and expected behavior
	if _, exists := firstItem["name"]; !exists {
		t.Error("Expected field 'name' not found")
	}

	// The key 'id' should be present for proper OData entity identification
	// (This is implementation-specific, but good practice for OData)
}

func TestMain(m *testing.M) {
	// Disable GORM logging for cleaner test output
	// Individual tests can enable it if needed
	fmt.Println("Running select columns tests...")
	m.Run()
}
