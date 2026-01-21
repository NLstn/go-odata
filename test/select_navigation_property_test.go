package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestSelectWithNavigationProperty tests that $select with navigation properties works correctly
// This is a regression test for the bug where $select with navigation properties caused a server crash
func TestSelectWithNavigationProperty(t *testing.T) {
	db := setupTestDBForSelectNav(t)
	service := setupServiceWithRelations(db, t)

	// Create test products with descriptions
	products := createProductsWithDescriptions(db, t)
	if len(products) == 0 {
		t.Fatal("Failed to create test products")
	}

	// Test: $select with navigation property should not crash
	req := httptest.NewRequest(http.MethodGet, "/TestProductWithDescs?$select=Name,Descriptions", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		return
	}

	// Verify response is valid JSON
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
		return
	}

	// Verify the response contains value
	if _, ok := response["value"]; !ok {
		t.Error("Response missing 'value' property")
		return
	}

	// Verify response includes selected properties
	values, ok := response["value"].([]interface{})
	if !ok || len(values) == 0 {
		t.Error("Response value is not an array or is empty")
		return
	}

	firstItem, ok := values[0].(map[string]interface{})
	if !ok {
		t.Error("First item is not an object")
		return
	}

	// Should have Name (selected)
	if _, ok := firstItem["Name"]; !ok {
		t.Error("Response missing 'Name' property")
	}

	// Should have ID (always included as key)
	if _, ok := firstItem["ID"]; !ok {
		t.Error("Response missing 'ID' property (key should always be included)")
	}

	// Should NOT have Price (not selected)
	if _, ok := firstItem["Price"]; ok {
		t.Error("Response should not include 'Price' property (not selected)")
	}
}

// TestSelectWithNavigationPropertyAndExpand tests $select combined with $expand
func TestSelectWithNavigationPropertyAndExpand(t *testing.T) {
	db := setupTestDBForSelectNav(t)
	service := setupServiceWithRelations(db, t)

	createProductsWithDescriptions(db, t)

	// Test: $select with $expand should work
	req := httptest.NewRequest(http.MethodGet, "/TestProductWithDescs?$select=Name&$expand=Descriptions", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
		return
	}

	values, ok := response["value"].([]interface{})
	if !ok || len(values) == 0 {
		t.Error("Response value is not an array or is empty")
		return
	}

	firstItem, ok := values[0].(map[string]interface{})
	if !ok {
		t.Error("First item is not an object")
		return
	}

	// Should have expanded Descriptions
	if _, ok := firstItem["Descriptions"]; !ok {
		t.Error("Response missing expanded 'Descriptions' navigation property")
	}
}

// TestComplexTypeDirectAccess tests that direct access to complex types returns the serialized complex object
func TestComplexTypeDirectAccess(t *testing.T) {
	db := setupTestDBForSelectNav(t)
	service := setupServiceWithComplexTypes(db, t)

	createProductWithComplexType(db, t)

	// Test: Direct access to complex type should return the complex object
	req := httptest.NewRequest(http.MethodGet, "/TestProductWithComplexes(1)/Address", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for complex type access, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if context, ok := response["@odata.context"].(string); !ok || !strings.Contains(context, "Address") {
		t.Fatalf("Expected @odata.context to reference Address, got %v", response["@odata.context"])
	}

	if response["Street"] != "123 Main St" {
		t.Fatalf("Expected Street '123 Main St', got %v", response["Street"])
	}
	if response["City"] != "Seattle" {
		t.Fatalf("Expected City 'Seattle', got %v", response["City"])
	}
	if response["State"] != "WA" {
		t.Fatalf("Expected State 'WA', got %v", response["State"])
	}
}

// TestComplexTypeInFilter tests that filtering by complex types returns 400
func TestComplexTypeInFilter(t *testing.T) {
	db := setupTestDBForSelectNav(t)
	service := setupServiceWithComplexTypes(db, t)

	createProductWithComplexType(db, t)

	// Test: Filtering by complex type should return 400
	req := httptest.NewRequest(http.MethodGet, "/TestProductWithComplexes?$filter=Address%20eq%20null", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for complex type filter, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestComplexTypeInSelect tests that selecting complex types works without SQL errors
func TestComplexTypeInSelect(t *testing.T) {
	db := setupTestDBForSelectNav(t)
	service := setupServiceWithComplexTypes(db, t)

	createProductWithComplexType(db, t)

	// Test: Selecting complex type should return 200 (complex type is skipped)
	req := httptest.NewRequest(http.MethodGet, "/TestProductWithComplexes?$select=Name,Address", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
		return
	}

	values, ok := response["value"].([]interface{})
	if !ok || len(values) == 0 {
		t.Error("Response value is not an array or is empty")
		return
	}

	firstItem, ok := values[0].(map[string]interface{})
	if !ok {
		t.Error("First item is not an object")
		return
	}

	// Should have Name (selected)
	if _, ok := firstItem["Name"]; !ok {
		t.Error("Response missing 'Name' property")
	}

	// Complex type should be skipped (not cause SQL error)
	// The response should still be valid
}

// Helper types for testing
type TestProductWithDesc struct {
	ID           uint                          `json:"ID" gorm:"primaryKey" odata:"key"`
	Name         string                        `json:"Name" odata:"required"`
	Price        float64                       `json:"Price" odata:"required"`
	Descriptions []TestProductDescForNavSelect `json:"Descriptions" gorm:"foreignKey:ProductID;references:ID"`
}

type TestProductDescForNavSelect struct {
	ID          uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	ProductID   uint   `json:"ProductID" odata:"required"`
	Description string `json:"Description" odata:"required"`
}

type TestProductWithComplex struct {
	ID      uint         `json:"ID" gorm:"primaryKey" odata:"key"`
	Name    string       `json:"Name" odata:"required"`
	Price   float64      `json:"Price" odata:"required"`
	Address *TestAddress `json:"Address,omitempty" gorm:"embedded;embeddedPrefix:addr_" odata:"nullable"`
}

type TestAddress struct {
	Street string `json:"Street"`
	City   string `json:"City"`
	State  string `json:"State"`
}

func setupServiceWithRelations(db *gorm.DB, t *testing.T) *odata.Service {
	if err := db.AutoMigrate(&TestProductWithDesc{}, &TestProductDescForNavSelect{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(&TestProductWithDesc{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	if err := service.RegisterEntity(&TestProductDescForNavSelect{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service
}

func setupServiceWithComplexTypes(db *gorm.DB, t *testing.T) *odata.Service {
	if err := db.AutoMigrate(&TestProductWithComplex{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(&TestProductWithComplex{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service
}

func createProductsWithDescriptions(db *gorm.DB, t *testing.T) []TestProductWithDesc {
	products := []TestProductWithDesc{
		{ID: 1, Name: "Laptop", Price: 999.99},
		{ID: 2, Name: "Mouse", Price: 29.99},
	}

	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	descriptions := []TestProductDescForNavSelect{
		{ID: 1, ProductID: 1, Description: "Gaming laptop"},
		{ID: 2, ProductID: 1, Description: "Work laptop"},
		{ID: 3, ProductID: 2, Description: "Wireless mouse"},
	}

	for _, d := range descriptions {
		if err := db.Create(&d).Error; err != nil {
			t.Fatalf("Failed to create description: %v", err)
		}
	}

	return products
}

func createProductWithComplexType(db *gorm.DB, t *testing.T) {
	product := TestProductWithComplex{
		ID:    1,
		Name:  "Test Product",
		Price: 99.99,
		Address: &TestAddress{
			Street: "123 Main St",
			City:   "Seattle",
			State:  "WA",
		},
	}

	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("Failed to create product: %v", err)
	}
}

func setupTestDBForSelectNav(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	return db
}
