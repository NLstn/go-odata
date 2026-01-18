package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestODataIDInCollectionMinimalMetadata tests that @odata.id is included for entities in collections
// with minimal metadata (default metadata level)
func TestODataIDInCollectionMinimalMetadata(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test products
	products := []Product{
		{ID: 1, Name: "Product 1", Price: 10.99},
		{ID: 2, Name: "Product 2", Price: 20.99},
		{ID: 3, Name: "Product 3", Price: 30.99},
	}
	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&Product{})

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test with minimal metadata (default)
	resp, err := http.Get(server.URL + "/Products?$top=3")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that @odata.context is present
	if _, ok := result["@odata.context"]; !ok {
		t.Error("Expected @odata.context in response")
	}

	// Get the value array
	value, ok := result["value"].([]interface{})
	if !ok {
		t.Fatalf("Expected value to be an array")
	}

	if len(value) != 3 {
		t.Fatalf("Expected 3 entities, got %d", len(value))
	}

	// Check that each entity has @odata.id
	for i, entity := range value {
		entityMap, ok := entity.(map[string]interface{})
		if !ok {
			t.Fatalf("Entity %d is not a map", i)
		}

		odataID, ok := entityMap["@odata.id"].(string)
		if !ok {
			t.Errorf("Entity %d missing @odata.id field", i)
			continue
		}

		// Verify @odata.id format matches entity set and key pattern
		expectedPattern := server.URL + "/Products("
		if len(odataID) < len(expectedPattern) || odataID[:len(expectedPattern)] != expectedPattern {
			t.Errorf("Entity %d @odata.id has wrong format: %s", i, odataID)
		}

		// Verify @odata.id ends with closing parenthesis
		if odataID[len(odataID)-1] != ')' {
			t.Errorf("Entity %d @odata.id does not end with ')': %s", i, odataID)
		}
	}
}

// TestODataIDInCollectionFullMetadata tests that @odata.id is included for entities in collections
// with full metadata
func TestODataIDInCollectionFullMetadata(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test products
	products := []Product{
		{ID: 1, Name: "Product 1", Price: 10.99},
		{ID: 2, Name: "Product 2", Price: 20.99},
	}
	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&Product{})

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test with full metadata
	req, err := http.NewRequest("GET", server.URL+"/Products?$top=2", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "application/json;odata.metadata=full")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Get the value array
	value, ok := result["value"].([]interface{})
	if !ok {
		t.Fatalf("Expected value to be an array")
	}

	if len(value) != 2 {
		t.Fatalf("Expected 2 entities, got %d", len(value))
	}

	// Check that each entity has @odata.id
	for i, entity := range value {
		entityMap, ok := entity.(map[string]interface{})
		if !ok {
			t.Fatalf("Entity %d is not a map", i)
		}

		if _, ok := entityMap["@odata.id"]; !ok {
			t.Errorf("Entity %d missing @odata.id field in full metadata", i)
		}
	}
}

// TestODataIDNotInCollectionNoneMetadata tests that @odata.id is NOT included for entities in collections
// with none metadata
func TestODataIDNotInCollectionNoneMetadata(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test products
	products := []Product{
		{ID: 1, Name: "Product 1", Price: 10.99},
		{ID: 2, Name: "Product 2", Price: 20.99},
	}
	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&Product{})

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test with none metadata
	req, err := http.NewRequest("GET", server.URL+"/Products?$top=2", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "application/json;odata.metadata=none")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// @odata.context should NOT be present in none metadata
	if _, ok := result["@odata.context"]; ok {
		t.Error("Did not expect @odata.context in none metadata response")
	}

	// Get the value array
	value, ok := result["value"].([]interface{})
	if !ok {
		t.Fatalf("Expected value to be an array")
	}

	if len(value) != 2 {
		t.Fatalf("Expected 2 entities, got %d", len(value))
	}

	// Check that entities do NOT have @odata.id in none metadata
	for i, entity := range value {
		entityMap, ok := entity.(map[string]interface{})
		if !ok {
			t.Fatalf("Entity %d is not a map", i)
		}

		if _, ok := entityMap["@odata.id"]; ok {
			t.Errorf("Entity %d should not have @odata.id field in none metadata", i)
		}
	}
}

// TestODataIDInSingleEntity tests that @odata.id is included for single entity responses
func TestODataIDInSingleEntity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test product
	product := Product{ID: 1, Name: "Test Product", Price: 99.99}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("Failed to create product: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&Product{})

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test single entity request
	resp, err := http.Get(server.URL + "/Products(1)")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that @odata.id is present
	odataID, ok := result["@odata.id"].(string)
	if !ok {
		t.Fatal("Expected @odata.id in single entity response")
	}

	// Verify @odata.id format
	expectedID := server.URL + "/Products(1)"
	if odataID != expectedID {
		t.Errorf("Expected @odata.id=%s, got %s", expectedID, odataID)
	}
}
