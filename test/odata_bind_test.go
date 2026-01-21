package odata_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
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

// Test POST with @odata.bind - Collection-valued navigation property
func TestPostWithODataBind_CollectionValuedRelativeURL(t *testing.T) {
	service, db := setupBindTestService(t)

	// Create order items to bind to
	orderItems := []BindTestOrderItem{
		{ProductID: 1, Quantity: 2},
		{ProductID: 2, Quantity: 3},
		{ProductID: 3, Quantity: 1},
	}
	db.Create(&orderItems)

	// Create a new order with items binding using relative URLs
	newOrder := map[string]interface{}{
		"OrderDate":        "2024-01-15",
		"TotalPrice":       129.99,
		"Items@odata.bind": []interface{}{"BindTestOrderItems(1)", "BindTestOrderItems(2)"},
	}

	body, _ := json.Marshal(newOrder)
	req := httptest.NewRequest(http.MethodPost, "/BindTestOrders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify the order was created with the correct items
	var order BindTestOrder
	if err := db.Preload("Items").Where("order_date = ?", "2024-01-15").First(&order).Error; err != nil {
		t.Fatalf("Failed to fetch created order: %v", err)
	}

	if len(order.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(order.Items))
	}

	// Verify the items are the correct ones
	foundItems := make(map[int]bool)
	for _, item := range order.Items {
		foundItems[item.ID] = true
	}
	if !foundItems[1] || !foundItems[2] {
		t.Errorf("Expected items 1 and 2, got items: %v", order.Items)
	}
}

// Test POST with @odata.bind - Collection-valued with absolute URLs
func TestPostWithODataBind_CollectionValuedAbsoluteURL(t *testing.T) {
	service, db := setupBindTestService(t)

	// Create order items to bind to
	orderItems := []BindTestOrderItem{
		{ProductID: 1, Quantity: 2},
		{ProductID: 2, Quantity: 3},
	}
	db.Create(&orderItems)

	// Create a new order with items binding using absolute URLs
	newOrder := map[string]interface{}{
		"OrderDate":  "2024-01-16",
		"TotalPrice": 89.99,
		"Items@odata.bind": []interface{}{
			"http://localhost:8080/BindTestOrderItems(1)",
			"http://localhost:8080/BindTestOrderItems(2)",
		},
	}

	body, _ := json.Marshal(newOrder)
	req := httptest.NewRequest(http.MethodPost, "/BindTestOrders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify the order was created with the correct items
	var order BindTestOrder
	if err := db.Preload("Items").Where("order_date = ?", "2024-01-16").First(&order).Error; err != nil {
		t.Fatalf("Failed to fetch created order: %v", err)
	}

	if len(order.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(order.Items))
	}
}

// Test POST with @odata.bind - Empty collection clears relationships
func TestPostWithODataBind_CollectionValuedEmpty(t *testing.T) {
	service, db := setupBindTestService(t)

	// Create a new order with empty items array
	newOrder := map[string]interface{}{
		"OrderDate":        "2024-01-17",
		"TotalPrice":       0.0,
		"Items@odata.bind": []interface{}{},
	}

	body, _ := json.Marshal(newOrder)
	req := httptest.NewRequest(http.MethodPost, "/BindTestOrders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify the order was created with no items
	var order BindTestOrder
	if err := db.Preload("Items").Where("order_date = ?", "2024-01-17").First(&order).Error; err != nil {
		t.Fatalf("Failed to fetch created order: %v", err)
	}

	if len(order.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(order.Items))
	}
}

// Test POST with @odata.bind - Invalid collection reference
func TestPostWithODataBind_CollectionValuedInvalidReference(t *testing.T) {
	service, _ := setupBindTestService(t)

	// Try to create an order with a non-existent item
	newOrder := map[string]interface{}{
		"OrderDate":        "2024-01-18",
		"TotalPrice":       99.99,
		"Items@odata.bind": []interface{}{"BindTestOrderItems(999)"},
	}

	body, _ := json.Marshal(newOrder)
	req := httptest.NewRequest(http.MethodPost, "/BindTestOrders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test POST with @odata.bind - Mixed entity sets in collection
func TestPostWithODataBind_CollectionValuedMixedEntitySets(t *testing.T) {
	service, _ := setupBindTestService(t)

	// Try to create an order with items from different entity sets
	newOrder := map[string]interface{}{
		"OrderDate":  "2024-01-19",
		"TotalPrice": 99.99,
		"Items@odata.bind": []interface{}{
			"BindTestOrderItems(1)",
			"BindTestProducts(1)", // Wrong entity set
		},
	}

	body, _ := json.Marshal(newOrder)
	req := httptest.NewRequest(http.MethodPost, "/BindTestOrders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test POST with @odata.bind - Collection with invalid type (not array)
func TestPostWithODataBind_CollectionValuedInvalidType(t *testing.T) {
	service, _ := setupBindTestService(t)

	// Try to create an order with items binding as string instead of array
	newOrder := map[string]interface{}{
		"OrderDate":        "2024-01-20",
		"TotalPrice":       99.99,
		"Items@odata.bind": "BindTestOrderItems(1)", // Should be array
	}

	body, _ := json.Marshal(newOrder)
	req := httptest.NewRequest(http.MethodPost, "/BindTestOrders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test PATCH with @odata.bind - Replace collection-valued navigation property
func TestPatchWithODataBind_ReplaceCollection(t *testing.T) {
	service, db := setupBindTestService(t)

	// Create order items
	orderItems := []BindTestOrderItem{
		{ProductID: 1, Quantity: 2},
		{ProductID: 2, Quantity: 3},
		{ProductID: 3, Quantity: 1},
	}
	db.Create(&orderItems)

	// Create an order with initial items
	order := BindTestOrder{
		OrderDate:  "2024-01-21",
		TotalPrice: 99.99,
	}
	db.Create(&order)
	// Add initial items
	db.Model(&order).Association("Items").Append([]BindTestOrderItem{orderItems[0], orderItems[1]})

	// Update the order to replace items with a different set
	updateData := map[string]interface{}{
		"Items@odata.bind": []interface{}{"BindTestOrderItems(2)", "BindTestOrderItems(3)"},
	}

	body, _ := json.Marshal(updateData)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/BindTestOrders(%d)", order.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("Expected status 204 or 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify the order's items were replaced
	var updatedOrder BindTestOrder
	if err := db.Preload("Items").First(&updatedOrder, order.ID).Error; err != nil {
		t.Fatalf("Failed to fetch updated order: %v", err)
	}

	if len(updatedOrder.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(updatedOrder.Items))
	}

	// Verify we have items 2 and 3, not 1 and 2
	foundItems := make(map[int]bool)
	for _, item := range updatedOrder.Items {
		foundItems[item.ID] = true
	}
	if !foundItems[2] || !foundItems[3] {
		t.Errorf("Expected items 2 and 3, got items: %v", updatedOrder.Items)
	}
	if foundItems[1] {
		t.Errorf("Item 1 should have been removed but is still present")
	}
}

// Test PATCH with @odata.bind - Clear collection with empty array
func TestPatchWithODataBind_ClearCollection(t *testing.T) {
	service, db := setupBindTestService(t)

	// Create order items
	orderItems := []BindTestOrderItem{
		{ProductID: 1, Quantity: 2},
		{ProductID: 2, Quantity: 3},
	}
	db.Create(&orderItems)

	// Create an order with initial items
	order := BindTestOrder{
		OrderDate:  "2024-01-22",
		TotalPrice: 99.99,
	}
	db.Create(&order)
	db.Model(&order).Association("Items").Append(orderItems)

	// Update the order to clear all items
	updateData := map[string]interface{}{
		"Items@odata.bind": []interface{}{},
	}

	body, _ := json.Marshal(updateData)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/BindTestOrders(%d)", order.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("Expected status 204 or 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify the order's items were cleared
	var updatedOrder BindTestOrder
	if err := db.Preload("Items").First(&updatedOrder, order.ID).Error; err != nil {
		t.Fatalf("Failed to fetch updated order: %v", err)
	}

	if len(updatedOrder.Items) != 0 {
		t.Errorf("Expected 0 items after clearing, got %d", len(updatedOrder.Items))
	}
}

// Test PATCH with @odata.bind - Mixed update with regular properties and collection binding
func TestPatchWithODataBind_MixedUpdateWithCollection(t *testing.T) {
	service, db := setupBindTestService(t)

	// Create order items
	orderItems := []BindTestOrderItem{
		{ProductID: 1, Quantity: 2},
		{ProductID: 2, Quantity: 3},
	}
	db.Create(&orderItems)

	// Create an order
	order := BindTestOrder{
		OrderDate:  "2024-01-23",
		TotalPrice: 50.00,
	}
	db.Create(&order)

	// Update both regular properties and collection navigation property
	updateData := map[string]interface{}{
		"TotalPrice":       199.99,
		"Items@odata.bind": []interface{}{"BindTestOrderItems(1)", "BindTestOrderItems(2)"},
	}

	body, _ := json.Marshal(updateData)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/BindTestOrders(%d)", order.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("Expected status 204 or 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify both updates were applied
	var updatedOrder BindTestOrder
	if err := db.Preload("Items").First(&updatedOrder, order.ID).Error; err != nil {
		t.Fatalf("Failed to fetch updated order: %v", err)
	}

	if updatedOrder.TotalPrice != 199.99 {
		t.Errorf("Expected TotalPrice to be 199.99, got %f", updatedOrder.TotalPrice)
	}

	if len(updatedOrder.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(updatedOrder.Items))
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
