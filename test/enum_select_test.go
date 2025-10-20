package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ProductStatus represents product status as a flags enum
type ProductStatusEnum int

const (
	ProductStatusEnumNone         ProductStatusEnum = 0
	ProductStatusEnumInStock      ProductStatusEnum = 1
	ProductStatusEnumOnSale       ProductStatusEnum = 2
	ProductStatusEnumDiscontinued ProductStatusEnum = 4
	ProductStatusEnumFeatured     ProductStatusEnum = 8
)

// ProductWithEnum represents a product entity with enum status
type ProductWithEnum struct {
	ID        uint              `json:"ID" gorm:"primaryKey" odata:"key"`
	Name      string            `json:"Name" gorm:"not null" odata:"required,maxlength=100"`
	Price     float64           `json:"Price" gorm:"not null" odata:"required"`
	Status    ProductStatusEnum `json:"Status" gorm:"not null" odata:"enum=ProductStatus,flags"`
	Version   int               `json:"Version" gorm:"default:1" odata:"etag"`
	CreatedAt time.Time         `json:"CreatedAt" gorm:"not null"`
}

func TestSelectWithEnumAndETag(t *testing.T) {
	// Initialize in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the ProductWithEnum model
	if err := db.AutoMigrate(&ProductWithEnum{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed the database with test data
	testProducts := []ProductWithEnum{
		{
			ID:        1,
			Name:      "Test Product 1",
			Price:     99.99,
			Status:    ProductStatusEnumInStock | ProductStatusEnumFeatured,
			Version:   1,
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			Name:      "Test Product 2",
			Price:     49.99,
			Status:    ProductStatusEnumOnSale,
			Version:   1,
			CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        3,
			Name:      "Test Product 3",
			Price:     29.99,
			Status:    ProductStatusEnumDiscontinued,
			Version:   2,
			CreatedAt: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
		},
	}

	if err := db.Create(&testProducts).Error; err != nil {
		t.Fatalf("Failed to seed database: %v", err)
	}

	// Create OData service
	service := odata.NewService(db)

	// Register the ProductWithEnum entity
	if err := service.RegisterEntity(&ProductWithEnum{}); err != nil {
		t.Fatalf("Failed to register ProductWithEnum entity: %v", err)
	}

	// Test 1: $select with enum property and ETag property
	t.Run("$select with enum and ETag", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ProductWithEnums?$select=Name,Status,Version", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check that we have values
		values, ok := response["value"].([]interface{})
		if !ok {
			t.Fatal("Expected value array in response")
		}

		if len(values) == 0 {
			t.Fatal("Expected at least one result")
		}

		// Check first item
		firstItem, ok := values[0].(map[string]interface{})
		if !ok {
			t.Fatal("Expected first value to be an object")
		}

		// Verify that the selected properties are present
		if _, ok := firstItem["Name"]; !ok {
			t.Error("Expected Name property in response")
		}

		if _, ok := firstItem["Status"]; !ok {
			t.Error("Expected Status property in response")
		}

		if _, ok := firstItem["Version"]; !ok {
			t.Error("Expected Version property in response")
		}

		// Most importantly: verify that ETag is present
		// This was the bug - ETag generation would crash with map[string]interface{}
		if _, ok := firstItem["@odata.etag"]; !ok {
			t.Error("Expected @odata.etag in response when Version is selected")
		}
	})

	// Test 2: $select with only enum property (no ETag)
	t.Run("$select with only enum", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ProductWithEnums?$select=Name,Status", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		values, ok := response["value"].([]interface{})
		if !ok || len(values) == 0 {
			t.Fatal("Expected at least one result")
		}

		firstItem := values[0].(map[string]interface{})

		// Verify that the selected properties are present
		if _, ok := firstItem["Name"]; !ok {
			t.Error("Expected Name property in response")
		}

		if _, ok := firstItem["Status"]; !ok {
			t.Error("Expected Status (enum) property in response")
		}

		// ID should always be included (key property)
		if _, ok := firstItem["ID"]; !ok {
			t.Error("Expected ID (key) property in response")
		}
	})

	// Test 3: $orderby with enum property
	t.Run("$orderby with enum", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ProductWithEnums?$orderby=Status", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// Test 4: $filter with enum property
	t.Run("$filter with enum numeric value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ProductWithEnums?$filter=Status%20eq%202", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		values, ok := response["value"].([]interface{})
		if !ok {
			t.Fatal("Expected value array in response")
		}

		// Should find the product with Status = ProductStatusEnumOnSale (2)
		if len(values) != 1 {
			t.Errorf("Expected 1 result, got %d", len(values))
		}
	})

	// Test 5: Combine $select with enum and $filter
	t.Run("$select and $filter with enum", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ProductWithEnums?$select=Name,Status&$filter=Status%20gt%200", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		values, ok := response["value"].([]interface{})
		if !ok {
			t.Fatal("Expected value array in response")
		}

		// Should find all products with Status > 0 (all of them)
		if len(values) != 3 {
			t.Errorf("Expected 3 results, got %d", len(values))
		}
	})
}

func TestSelectWithEnumInSingleEntity(t *testing.T) {
	// Initialize in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate and seed
	if err := db.AutoMigrate(&ProductWithEnum{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	testProduct := ProductWithEnum{
		ID:        1,
		Name:      "Test Product",
		Price:     99.99,
		Status:    ProductStatusEnumInStock | ProductStatusEnumFeatured,
		Version:   1,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&testProduct).Error; err != nil {
		t.Fatalf("Failed to seed database: %v", err)
	}

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&ProductWithEnum{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test $select on single entity with enum
	t.Run("Single entity $select with enum", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ProductWithEnums(1)?$select=Name,Status,Version", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify selected properties
		if _, ok := response["Name"]; !ok {
			t.Error("Expected Name property in response")
		}

		if _, ok := response["Status"]; !ok {
			t.Error("Expected Status (enum) property in response")
		}

		if _, ok := response["Version"]; !ok {
			t.Error("Expected Version property in response")
		}

		// Verify ETag is present (this was the bug)
		if _, ok := response["@odata.etag"]; !ok {
			t.Error("Expected @odata.etag in response")
		}
	})
}
