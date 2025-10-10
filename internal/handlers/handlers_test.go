package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TestEntity struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
}

func setupTestHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(TestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta)
	return handler, db
}

func TestEntityHandlerCollection(t *testing.T) {
	handler, db := setupTestHandler(t)

	// Insert test data
	testData := []TestEntity{
		{ID: 1, Name: "Test 1"},
		{ID: 2, Name: "Test 2"},
		{ID: 3, Name: "Test 3"},
	}
	for _, entity := range testData {
		db.Create(&entity)
	}

	req := httptest.NewRequest(http.MethodGet, "/TestEntities", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 3 {
		t.Errorf("len(value) = %v, want 3", len(value))
	}
}

func TestEntityHandlerCollectionEmpty(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/TestEntities", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 0 {
		t.Errorf("len(value) = %v, want 0", len(value))
	}
}

func TestEntityHandlerCollectionMethodNotAllowed(t *testing.T) {
	handler, _ := setupTestHandler(t)

	// POST is now supported for collections, so only test PUT, DELETE, and PATCH
	methods := []string{http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/TestEntities", nil)
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestEntityHandlerEntity(t *testing.T) {
	handler, db := setupTestHandler(t)

	// Insert test data
	entity := TestEntity{ID: 42, Name: "Test Entity"}
	db.Create(&entity)

	req := httptest.NewRequest(http.MethodGet, "/TestEntities(42)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "42")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["name"] != "Test Entity" {
		t.Errorf("name = %v, want Test Entity", response["name"])
	}

	if response["id"] != float64(42) {
		t.Errorf("id = %v, want 42", response["id"])
	}

	// Check context URL
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("@odata.context is not a string")
	}

	if context == "" {
		t.Error("@odata.context should not be empty")
	}
}

func TestEntityHandlerEntityNotFound(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/TestEntities(999)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "999")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

func TestEntityHandlerEntityMethodNotAllowed(t *testing.T) {
	handler, _ := setupTestHandler(t)

	// DELETE and PATCH are now supported, so only test POST and PUT
	methods := []string{http.MethodPost, http.MethodPut}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/TestEntities(1)", nil)
			w := httptest.NewRecorder()

			handler.HandleEntity(w, req, "1")

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestMetadataHandler(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	entityMeta, _ := metadata.AnalyzeEntity(TestEntity{})
	entities["TestEntities"] = entityMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/xml" {
		t.Errorf("Content-Type = %v, want application/xml", contentType)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Metadata response body is empty")
	}

	// Check that entity set name appears in metadata
	if !contains(body, "TestEntities") {
		t.Error("Metadata should contain TestEntities")
	}
}

func TestMetadataHandlerMethodNotAllowed(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	handler := NewMetadataHandler(entities)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/$metadata", nil)
			w := httptest.NewRecorder()

			handler.HandleMetadata(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestMetadataHandlerJSON(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	entityMeta, _ := metadata.AnalyzeEntity(TestEntity{})
	entities["TestEntities"] = entityMeta

	handler := NewMetadataHandler(entities)

	t.Run("JSON format via query parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", contentType)
		}

		odataVersion := w.Header().Get("OData-Version")
		if odataVersion != "4.0" {
			t.Errorf("OData-Version = %v, want 4.0", odataVersion)
		}

		// Parse JSON response
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		// Validate JSON structure
		if version, ok := response["$Version"].(string); !ok || version != "4.0" {
			t.Errorf("Expected $Version to be 4.0, got %v", response["$Version"])
		}

		odataService, ok := response["ODataService"].(map[string]interface{})
		if !ok {
			t.Fatal("ODataService not found in response")
		}

		// Check for entity type
		entityType, ok := odataService["TestEntity"].(map[string]interface{})
		if !ok {
			t.Fatal("TestEntity not found in metadata")
		}

		// Validate entity type structure
		if kind, ok := entityType["$Kind"].(string); !ok || kind != "EntityType" {
			t.Errorf("Expected $Kind to be EntityType, got %v", entityType["$Kind"])
		}

		// Check for key property
		key, ok := entityType["$Key"].([]interface{})
		if !ok || len(key) == 0 {
			t.Error("$Key not found or empty")
		}

		// Check for properties
		if _, ok := entityType["id"]; !ok {
			t.Error("id property not found in entity type")
		}
		if _, ok := entityType["name"]; !ok {
			t.Error("name property not found in entity type")
		}

		// Check for container
		container, ok := odataService["Container"].(map[string]interface{})
		if !ok {
			t.Fatal("Container not found in metadata")
		}

		// Validate container structure
		if kind, ok := container["$Kind"].(string); !ok || kind != "EntityContainer" {
			t.Errorf("Expected $Kind to be EntityContainer, got %v", container["$Kind"])
		}

		// Check for entity set
		if _, ok := container["TestEntities"]; !ok {
			t.Error("TestEntities entity set not found in container")
		}
	})

	t.Run("JSON format via Accept header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", contentType)
		}

		// Parse JSON response
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		if _, ok := response["$Version"]; !ok {
			t.Error("$Version not found in JSON response")
		}
	})

	t.Run("XML format by default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/xml" {
			t.Errorf("Content-Type = %v, want application/xml", contentType)
		}

		body := w.Body.String()
		if !contains(body, "<?xml") {
			t.Error("Response should contain XML declaration")
		}
	})
}

func TestServiceDocumentHandler(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	entityMeta, _ := metadata.AnalyzeEntity(TestEntity{})
	entities["TestEntities"] = entityMeta

	handler := NewServiceDocumentHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 1 {
		t.Errorf("len(value) = %v, want 1", len(value))
	}
}

func TestServiceDocumentHandlerEmpty(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	handler := NewServiceDocumentHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 0 {
		t.Errorf("len(value) = %v, want 0", len(value))
	}
}

func TestServiceDocumentHandlerMethodNotAllowed(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	handler := NewServiceDocumentHandler(entities)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()

			handler.HandleServiceDocument(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr &&
		(s[0:1] == substr || s[len(s)-1:] == substr ||
			len(s) > len(substr) && (s[0:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Additional entity type for testing query options
type Product struct {
	ID          int     `json:"ID" gorm:"primarykey" odata:"key"`
	Name        string  `json:"Name"`
	Description string  `json:"Description"`
	Price       float64 `json:"Price"`
	Category    string  `json:"Category"`
}

func setupProductHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(Product{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta)
	return handler, db
}

func TestEntityHandlerCollectionWithFilter(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert test data
	products := []Product{
		{ID: 1, Name: "Laptop", Description: "High-performance laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Description: "Wireless mouse", Price: 29.99, Category: "Electronics"},
		{ID: 3, Name: "Coffee Mug", Description: "Ceramic mug", Price: 15.50, Category: "Kitchen"},
		{ID: 4, Name: "Office Chair", Description: "Ergonomic chair", Price: 249.99, Category: "Furniture"},
	}
	for _, product := range products {
		db.Create(&product)
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{
			name:          "Filter by category",
			query:         "$filter=Category eq 'Electronics'",
			expectedCount: 2,
		},
		{
			name:          "Filter by price greater than",
			query:         "$filter=Price gt 100",
			expectedCount: 2,
		},
		{
			name:          "Filter by price less than or equal",
			query:         "$filter=Price le 30",
			expectedCount: 2,
		},
		{
			name:          "Filter with contains",
			query:         "$filter=contains(Name,'Laptop')",
			expectedCount: 1,
		},
		{
			name:          "Filter with startswith",
			query:         "$filter=startswith(Category,'Elec')",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products", nil)
			q := req.URL.Query()
			// Parse and add query parameters manually to handle spaces properly
			if tt.query != "" {
				parts := strings.SplitN(tt.query, "=", 2)
				if len(parts) == 2 {
					q.Add(parts[0], parts[1])
				}
			}
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
				t.Logf("Response body: %s", w.Body.String())
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("value field is not an array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(value))
			}
		})
	}
}

func TestEntityHandlerCollectionWithSelect(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert test data
	products := []Product{
		{ID: 1, Name: "Laptop", Description: "High-performance laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Description: "Wireless mouse", Price: 29.99, Category: "Electronics"},
	}
	for _, product := range products {
		db.Create(&product)
	}

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	q := req.URL.Query()
	q.Add("$select", "Name,Price")
	req.URL.RawQuery = q.Encode()
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 2 {
		t.Errorf("Expected 2 results, got %d", len(value))
	}

	// Check that only selected properties are present
	if len(value) > 0 {
		item, ok := value[0].(map[string]interface{})
		if !ok {
			t.Fatal("Item is not a map")
		}

		if _, hasName := item["Name"]; !hasName {
			t.Error("Expected Name property to be present")
		}

		if _, hasPrice := item["Price"]; !hasPrice {
			t.Error("Expected Price property to be present")
		}

		if _, hasDescription := item["Description"]; hasDescription {
			t.Error("Did not expect Description property to be present")
		}

		if _, hasCategory := item["Category"]; hasCategory {
			t.Error("Did not expect Category property to be present")
		}
	}
}

func TestEntityHandlerCollectionWithOrderBy(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert test data
	products := []Product{
		{ID: 1, Name: "Laptop", Description: "High-performance laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Description: "Wireless mouse", Price: 29.99, Category: "Electronics"},
		{ID: 3, Name: "Coffee Mug", Description: "Ceramic mug", Price: 15.50, Category: "Kitchen"},
		{ID: 4, Name: "Office Chair", Description: "Ergonomic chair", Price: 249.99, Category: "Furniture"},
	}
	for _, product := range products {
		db.Create(&product)
	}

	tests := []struct {
		name          string
		query         string
		expectedFirst string
		expectedLast  string
	}{
		{
			name:          "Order by name ascending",
			query:         "$orderby=Name asc",
			expectedFirst: "Coffee Mug",
			expectedLast:  "Office Chair",
		},
		{
			name:          "Order by name descending",
			query:         "$orderby=Name desc",
			expectedFirst: "Office Chair",
			expectedLast:  "Coffee Mug",
		},
		{
			name:          "Order by price ascending",
			query:         "$orderby=Price asc",
			expectedFirst: "Coffee Mug",
			expectedLast:  "Laptop",
		},
		{
			name:          "Order by price descending",
			query:         "$orderby=Price desc",
			expectedFirst: "Laptop",
			expectedLast:  "Coffee Mug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products", nil)
			q := req.URL.Query()
			if tt.query != "" {
				parts := strings.SplitN(tt.query, "=", 2)
				if len(parts) == 2 {
					q.Add(parts[0], parts[1])
				}
			}
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("value field is not an array")
			}

			if len(value) != 4 {
				t.Errorf("Expected 4 results, got %d", len(value))
			}

			// Check first item
			if len(value) > 0 {
				firstItem, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("First item is not a map")
				}

				firstName, ok := firstItem["Name"].(string)
				if !ok {
					t.Fatal("Name is not a string")
				}

				if firstName != tt.expectedFirst {
					t.Errorf("Expected first item to be %s, got %s", tt.expectedFirst, firstName)
				}
			}

			// Check last item
			if len(value) > 0 {
				lastItem, ok := value[len(value)-1].(map[string]interface{})
				if !ok {
					t.Fatal("Last item is not a map")
				}

				lastName, ok := lastItem["Name"].(string)
				if !ok {
					t.Fatal("Name is not a string")
				}

				if lastName != tt.expectedLast {
					t.Errorf("Expected last item to be %s, got %s", tt.expectedLast, lastName)
				}
			}
		})
	}
}

func TestEntityHandlerCollectionWithCombinedOptions(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert test data
	products := []Product{
		{ID: 1, Name: "Laptop", Description: "High-performance laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Description: "Wireless mouse", Price: 29.99, Category: "Electronics"},
		{ID: 3, Name: "Keyboard", Description: "Mechanical keyboard", Price: 149.99, Category: "Electronics"},
		{ID: 4, Name: "Coffee Mug", Description: "Ceramic mug", Price: 15.50, Category: "Kitchen"},
	}
	for _, product := range products {
		db.Create(&product)
	}

	// Test filter + orderby + select
	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	q := req.URL.Query()
	q.Add("$filter", "Category eq 'Electronics'")
	q.Add("$orderby", "Price desc")
	q.Add("$select", "Name,Price")
	req.URL.RawQuery = q.Encode()
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	// Should have 3 Electronics items
	if len(value) != 3 {
		t.Errorf("Expected 3 results, got %d", len(value))
	}

	// First item should be Laptop (highest price)
	if len(value) > 0 {
		firstItem, ok := value[0].(map[string]interface{})
		if !ok {
			t.Fatal("First item is not a map")
		}

		firstName, _ := firstItem["Name"].(string)
		if firstName != "Laptop" {
			t.Errorf("Expected first item to be Laptop, got %s", firstName)
		}

		// Should only have Name and Price
		if len(firstItem) > 2 {
			t.Errorf("Expected only 2 properties, got %d", len(firstItem))
		}
	}
}

func TestEntityHandlerCollectionWithInvalidFilter(t *testing.T) {
	handler, _ := setupProductHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	q := req.URL.Query()
	q.Add("$filter", "InvalidProperty eq 'value'")
	req.URL.RawQuery = q.Encode()
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}
