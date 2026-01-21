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

// TestApplyGroupByColumnNames tests that $apply groupby uses correct database column names
// This ensures that GORM's snake_case column names are used correctly
func TestApplyGroupByColumnNames(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	type Product struct {
		ID         int     `json:"id" gorm:"primaryKey" odata:"key"`
		Name       string  `json:"name"`
		Price      float64 `json:"price"`
		CategoryID int     `json:"CategoryID"` // PascalCase will be converted to category_id in DB
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test products with different categories
	products := []Product{
		{ID: 1, Name: "Laptop", Price: 999.99, CategoryID: 1},
		{ID: 2, Name: "Mouse", Price: 29.99, CategoryID: 1},
		{ID: 3, Name: "Keyboard", Price: 79.99, CategoryID: 1},
		{ID: 4, Name: "Monitor", Price: 299.99, CategoryID: 2},
		{ID: 5, Name: "Desk", Price: 499.99, CategoryID: 3},
	}
	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	service.RegisterEntity(&Product{})

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test groupby with PascalCase property name
	resp, err := http.Get(server.URL + "/Products?$apply=groupby((CategoryID))")
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

	// Check that we have grouped results
	value, ok := result["value"].([]interface{})
	if !ok {
		t.Fatalf("Expected value to be an array")
	}

	// Should have 3 groups (categories 1, 2, 3)
	if len(value) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(value))
	}

	// Verify each group has CategoryID field
	for i, entity := range value {
		entityMap, ok := entity.(map[string]interface{})
		if !ok {
			t.Fatalf("Entity %d is not a map", i)
		}

		if _, ok := entityMap["CategoryID"]; !ok {
			t.Errorf("Entity %d missing CategoryID field", i)
		}
	}
}

// TestApplyGroupByWithAggregate tests groupby with aggregate transformations
func TestApplyGroupByWithAggregate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	type Product struct {
		ID         int     `json:"id" gorm:"primaryKey" odata:"key"`
		Name       string  `json:"name"`
		Price      float64 `json:"price"`
		CategoryID int     `json:"CategoryID"`
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test products
	products := []Product{
		{ID: 1, Name: "Laptop", Price: 999.99, CategoryID: 1},
		{ID: 2, Name: "Mouse", Price: 29.99, CategoryID: 1},
		{ID: 3, Name: "Keyboard", Price: 79.99, CategoryID: 1},
		{ID: 4, Name: "Monitor", Price: 299.99, CategoryID: 2},
		{ID: 5, Name: "Desk", Price: 499.99, CategoryID: 3},
	}
	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	service.RegisterEntity(&Product{})

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test groupby with aggregate count
	resp, err := http.Get(server.URL + "/Products?$apply=groupby((CategoryID),aggregate($count%20as%20Count))")
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

	value, ok := result["value"].([]interface{})
	if !ok {
		t.Fatalf("Expected value to be an array")
	}

	if len(value) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(value))
	}

	// Verify groups have both CategoryID and Count
	foundCategory1 := false
	for _, entity := range value {
		entityMap, ok := entity.(map[string]interface{})
		if !ok {
			t.Fatal("Entity is not a map")
		}

		categoryID, ok := entityMap["CategoryID"].(float64)
		if !ok {
			t.Error("Missing or invalid CategoryID")
			continue
		}

		count, ok := entityMap["Count"].(float64)
		if !ok {
			t.Errorf("Missing or invalid Count for CategoryID %v", categoryID)
			continue
		}

		// Category 1 should have 3 products
		if int(categoryID) == 1 {
			foundCategory1 = true
			if int(count) != 3 {
				t.Errorf("Expected Count=3 for CategoryID=1, got %v", count)
			}
		}
	}

	if !foundCategory1 {
		t.Error("Did not find CategoryID=1 in results")
	}
}
