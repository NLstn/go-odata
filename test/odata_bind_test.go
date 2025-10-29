package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test entities for @odata.bind functionality

// BindTestCategory represents a category entity
type BindTestCategory struct {
	ID       int               `json:"ID" gorm:"primaryKey;autoIncrement" odata:"key"`
	Name     string            `json:"Name" odata:"required"`
	Products []BindTestProduct `json:"Products,omitempty" gorm:"foreignKey:CategoryID"`
}

// BindTestProduct represents a product with a single-valued navigation to category
type BindTestProduct struct {
	ID         int               `json:"ID" gorm:"primaryKey;autoIncrement" odata:"key"`
	Name       string            `json:"Name" odata:"required"`
	Price      float64           `json:"Price"`
	CategoryID *int              `json:"CategoryID,omitempty"`
	Category   *BindTestCategory `json:"Category,omitempty" gorm:"foreignKey:CategoryID"`
}

// BindTestOrder represents an order with collection-valued navigation
type BindTestOrder struct {
	ID         int                 `json:"ID" gorm:"primaryKey;autoIncrement" odata:"key"`
	OrderDate  string              `json:"OrderDate" odata:"required"`
	TotalPrice float64             `json:"TotalPrice"`
	Items      []BindTestOrderItem `json:"Items,omitempty" gorm:"foreignKey:OrderID"`
}

// BindTestOrderItem represents an order item
type BindTestOrderItem struct {
	ID        int              `json:"ID" gorm:"primaryKey;autoIncrement" odata:"key"`
	OrderID   *int             `json:"OrderID,omitempty"`
	ProductID int              `json:"ProductID" odata:"required"`
	Quantity  int              `json:"Quantity" odata:"required"`
	Order     *BindTestOrder   `json:"Order,omitempty" gorm:"foreignKey:OrderID"`
	Product   *BindTestProduct `json:"Product,omitempty" gorm:"foreignKey:ProductID"`
}

func setupBindTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&BindTestCategory{}, &BindTestProduct{}, &BindTestOrder{}, &BindTestOrderItem{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	categories := []BindTestCategory{
		{ID: 1, Name: "Electronics"},
		{ID: 2, Name: "Books"},
		{ID: 3, Name: "Clothing"},
	}
	db.Create(&categories)

	products := []BindTestProduct{
		{ID: 1, Name: "Laptop", Price: 999.99},
		{ID: 2, Name: "Mouse", Price: 29.99},
		{ID: 3, Name: "Keyboard", Price: 79.99},
	}
	db.Create(&products)

	service := odata.NewService(db)
	if err := service.RegisterEntity(&BindTestCategory{}); err != nil {
		t.Fatalf("Failed to register BindTestCategory: %v", err)
	}
	if err := service.RegisterEntity(&BindTestProduct{}); err != nil {
		t.Fatalf("Failed to register BindTestProduct: %v", err)
	}
	if err := service.RegisterEntity(&BindTestOrder{}); err != nil {
		t.Fatalf("Failed to register BindTestOrder: %v", err)
	}
	if err := service.RegisterEntity(&BindTestOrderItem{}); err != nil {
		t.Fatalf("Failed to register BindTestOrderItem: %v", err)
	}

	return service, db
}

// Test POST with @odata.bind - Single-valued navigation property with relative URL
func TestPostWithODataBind_SingleValuedRelativeURL(t *testing.T) {
	service, db := setupBindTestService(t)

	// Create a new product with category binding using relative URL
	newProduct := map[string]interface{}{
		"Name":                "Tablet",
		"Price":               399.99,
		"Category@odata.bind": "BindTestCategories(1)",
	}

	body, _ := json.Marshal(newProduct)
	req := httptest.NewRequest(http.MethodPost, "/BindTestProducts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify the product was created with the correct category
	var product BindTestProduct
	if err := db.Preload("Category").Where("name = ?", "Tablet").First(&product).Error; err != nil {
		t.Fatalf("Failed to fetch created product: %v", err)
	}

	if product.CategoryID == nil {
		t.Error("Expected CategoryID to be set")
	} else if *product.CategoryID != 1 {
		t.Errorf("Expected CategoryID to be 1, got %d", *product.CategoryID)
	}

	if product.Category == nil {
		t.Error("Expected Category to be loaded")
	} else if product.Category.Name != "Electronics" {
		t.Errorf("Expected Category.Name to be 'Electronics', got '%s'", product.Category.Name)
	}
}

// Test POST with @odata.bind - Single-valued navigation property with absolute URL
func TestPostWithODataBind_SingleValuedAbsoluteURL(t *testing.T) {
	service, db := setupBindTestService(t)

	// Create a new product with category binding using absolute URL
	newProduct := map[string]interface{}{
		"Name":                "Smartphone",
		"Price":               699.99,
		"Category@odata.bind": "http://localhost:8080/BindTestCategories(2)",
	}

	body, _ := json.Marshal(newProduct)
	req := httptest.NewRequest(http.MethodPost, "/BindTestProducts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify the product was created with the correct category
	var product BindTestProduct
	if err := db.Preload("Category").Where("name = ?", "Smartphone").First(&product).Error; err != nil {
		t.Fatalf("Failed to fetch created product: %v", err)
	}

	if product.CategoryID == nil {
		t.Error("Expected CategoryID to be set")
	} else if *product.CategoryID != 2 {
		t.Errorf("Expected CategoryID to be 2, got %d", *product.CategoryID)
	}
}

// Test POST with @odata.bind - Invalid entity reference
func TestPostWithODataBind_InvalidEntityReference(t *testing.T) {
	service, _ := setupBindTestService(t)

	// Try to create a product with a non-existent category
	newProduct := map[string]interface{}{
		"Name":                "Invalid Product",
		"Price":               99.99,
		"Category@odata.bind": "BindTestCategories(999)",
	}

	body, _ := json.Marshal(newProduct)
	req := httptest.NewRequest(http.MethodPost, "/BindTestProducts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test POST with @odata.bind - Wrong entity set
func TestPostWithODataBind_WrongEntitySet(t *testing.T) {
	service, _ := setupBindTestService(t)

	// Try to create a product with a binding to wrong entity set
	newProduct := map[string]interface{}{
		"Name":                "Wrong Binding",
		"Price":               99.99,
		"Category@odata.bind": "BindTestProducts(1)", // Should be BindTestCategories
	}

	body, _ := json.Marshal(newProduct)
	req := httptest.NewRequest(http.MethodPost, "/BindTestProducts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test PATCH with @odata.bind - Update single-valued navigation property
func TestPatchWithODataBind_UpdateNavigation(t *testing.T) {
	service, db := setupBindTestService(t)

	// Create a product with a category
	product := BindTestProduct{
		Name:       "Test Product",
		Price:      49.99,
		CategoryID: intPtr(1),
	}
	db.Create(&product)

	// Update the product's category using @odata.bind
	updateData := map[string]interface{}{
		"Category@odata.bind": "BindTestCategories(3)",
	}

	body, _ := json.Marshal(updateData)
	req := httptest.NewRequest(http.MethodPatch, "/BindTestProducts("+string(rune(product.ID+48))+")", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("Expected status 204 or 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify the product's category was updated
	var updatedProduct BindTestProduct
	if err := db.Preload("Category").First(&updatedProduct, product.ID).Error; err != nil {
		t.Fatalf("Failed to fetch updated product: %v", err)
	}

	if updatedProduct.CategoryID == nil {
		t.Error("Expected CategoryID to be set")
	} else if *updatedProduct.CategoryID != 3 {
		t.Errorf("Expected CategoryID to be 3, got %d", *updatedProduct.CategoryID)
	}
}

// Test PATCH with @odata.bind and regular properties together
func TestPatchWithODataBind_MixedUpdate(t *testing.T) {
	service, db := setupBindTestService(t)

	// Create a product
	product := BindTestProduct{
		Name:       "Mixed Update Test",
		Price:      19.99,
		CategoryID: intPtr(1),
	}
	db.Create(&product)

	// Update both regular properties and navigation property
	updateData := map[string]interface{}{
		"Price":               29.99,
		"Category@odata.bind": "BindTestCategories(2)",
	}

	body, _ := json.Marshal(updateData)
	req := httptest.NewRequest(http.MethodPatch, "/BindTestProducts("+string(rune(product.ID+48))+")", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("Expected status 204 or 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify both updates were applied
	var updatedProduct BindTestProduct
	if err := db.Preload("Category").First(&updatedProduct, product.ID).Error; err != nil {
		t.Fatalf("Failed to fetch updated product: %v", err)
	}

	if updatedProduct.Price != 29.99 {
		t.Errorf("Expected Price to be 29.99, got %f", updatedProduct.Price)
	}

	if updatedProduct.CategoryID == nil {
		t.Error("Expected CategoryID to be set")
	} else if *updatedProduct.CategoryID != 2 {
		t.Errorf("Expected CategoryID to be 2, got %d", *updatedProduct.CategoryID)
	}
}

// Test @odata.bind with invalid format
func TestODataBind_InvalidFormat(t *testing.T) {
	service, _ := setupBindTestService(t)

	testCases := []struct {
		name        string
		bindValue   interface{}
		description string
	}{
		{
			name:        "Missing parentheses",
			bindValue:   "BindTestCategories",
			description: "Entity reference without key",
		},
		{
			name:        "Invalid type",
			bindValue:   123,
			description: "Numeric value instead of string",
		},
		{
			name:        "Empty string",
			bindValue:   "",
			description: "Empty binding value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newProduct := map[string]interface{}{
				"Name":                "Test Product",
				"Price":               99.99,
				"Category@odata.bind": tc.bindValue,
			}

			body, _ := json.Marshal(newProduct)
			req := httptest.NewRequest(http.MethodPost, "/BindTestProducts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: Expected status 400, got %d: %s", tc.description, w.Code, w.Body.String())
			}
		})
	}
}

// Test @odata.bind with non-existent navigation property
func TestODataBind_NonExistentNavigationProperty(t *testing.T) {
	service, _ := setupBindTestService(t)

	newProduct := map[string]interface{}{
		"Name":                       "Test Product",
		"Price":                      99.99,
		"NonExistentProp@odata.bind": "BindTestCategories(1)",
	}

	body, _ := json.Marshal(newProduct)
	req := httptest.NewRequest(http.MethodPost, "/BindTestProducts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for non-existent navigation property, got %d: %s", w.Code, w.Body.String())
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
