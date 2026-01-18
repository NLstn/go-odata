package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// RealWorldProduct is a test entity for realistic overload tests
type RealWorldProduct struct {
	ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string  `json:"Name"`
	Price    float64 `json:"Price"`
	Category string  `json:"Category"`
}

// setupRealWorldTestService sets up a test service
func setupRealWorldTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&RealWorldProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	products := []RealWorldProduct{
		{ID: 1, Name: "Laptop", Price: 1000.0, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Price: 25.0, Category: "Electronics"},
		{ID: 3, Name: "Keyboard", Price: 75.0, Category: "Electronics"},
		{ID: 4, Name: "Desk", Price: 200.0, Category: "Furniture"},
		{ID: 5, Name: "Chair", Price: 150.0, Category: "Furniture"},
	}
	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to seed database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&RealWorldProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestRealWorldScenario_GetTopProducts demonstrates overloaded functions for filtering products
func TestRealWorldScenario_GetTopProducts(t *testing.T) {
	service, db := setupRealWorldTestService(t)

	// Overload 1: Get top N products by price (no category filter)
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "GetTopProducts",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf([]RealWorldProduct{}),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			count := params["count"].(int64)
			var products []RealWorldProduct
			if err := db.Order("price DESC").Limit(int(count)).Find(&products).Error; err != nil {
				return nil, err
			}
			return products, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register first overload: %v", err)
	}

	// Overload 2: Get top N products by price in a specific category
	err = service.RegisterFunction(odata.FunctionDefinition{
		Name:    "GetTopProducts",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
			{Name: "category", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: reflect.TypeOf([]RealWorldProduct{}),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			count := params["count"].(int64)
			category := params["category"].(string)
			var products []RealWorldProduct
			if err := db.Where("category = ?", category).Order("price DESC").Limit(int(count)).Find(&products).Error; err != nil {
				return nil, err
			}
			return products, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register second overload: %v", err)
	}

	// Test overload 1: Get top 2 products (no category filter)
	req := httptest.NewRequest(http.MethodGet, "/GetTopProducts()?count=2", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Overload 1: Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response1 map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response1); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	products1, ok := response1["value"].([]interface{})
	if !ok || len(products1) != 2 {
		t.Errorf("Expected 2 products, got %v", len(products1))
	}

	// Verify the products are sorted by price (descending)
	if product1, ok := products1[0].(map[string]interface{}); ok {
		if name, ok := product1["Name"].(string); !ok || name != "Laptop" {
			t.Errorf("First product should be Laptop, got %v", name)
		}
	}

	// Test overload 2: Get top 1 product in "Furniture" category
	req = httptest.NewRequest(http.MethodGet, "/GetTopProducts()?count=1&category=Furniture", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Overload 2: Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response2 map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response2); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	products2, ok := response2["value"].([]interface{})
	if !ok || len(products2) != 1 {
		t.Errorf("Expected 1 product, got %v", len(products2))
	}

	// Verify it's a furniture item
	if product2, ok := products2[0].(map[string]interface{}); ok {
		if category, ok := product2["Category"].(string); !ok || category != "Furniture" {
			t.Errorf("Product should be in Furniture category, got %v", category)
		}
		if name, ok := product2["Name"].(string); !ok || name != "Desk" {
			t.Errorf("Top furniture product should be Desk, got %v", name)
		}
	}
}

// TestRealWorldScenario_ApplyDiscount demonstrates overloaded actions for applying discounts
func TestRealWorldScenario_ApplyDiscount(t *testing.T) {
	service, db := setupRealWorldTestService(t)

	// Overload 1: Apply discount to all products
	err := service.RegisterAction(odata.ActionDefinition{
		Name:    "ApplyDiscount",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			percentage := params["percentage"].(float64)
			multiplier := 1.0 - (percentage / 100.0)

			if err := db.Model(&RealWorldProduct{}).Where("1 = 1").
				Update("price", gorm.Expr("price * ?", multiplier)).Error; err != nil {
				return err
			}

			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register first overload: %v", err)
	}

	// Overload 2: Apply discount to products in a specific category
	err = service.RegisterAction(odata.ActionDefinition{
		Name:    "ApplyDiscount",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
			{Name: "category", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			percentage := params["percentage"].(float64)
			category := params["category"].(string)
			multiplier := 1.0 - (percentage / 100.0)

			if err := db.Model(&RealWorldProduct{}).Where("category = ?", category).
				Update("price", gorm.Expr("price * ?", multiplier)).Error; err != nil {
				return err
			}

			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register second overload: %v", err)
	}

	// Test overload 2: Apply 10% discount to Furniture
	body := []byte(`{"percentage": 10.0, "category": "Furniture"}`)
	req := httptest.NewRequest(http.MethodPost, "/ApplyDiscount", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify only furniture prices were changed
	var desk RealWorldProduct
	if err := db.First(&desk, 4).Error; err != nil {
		t.Fatalf("Failed to fetch desk: %v", err)
	}
	expectedDeskPrice := 180.0 // 200 - 10%
	if desk.Price != expectedDeskPrice {
		t.Errorf("Desk price = %v, want %v", desk.Price, expectedDeskPrice)
	}

	var laptop RealWorldProduct
	if err := db.First(&laptop, 1).Error; err != nil {
		t.Fatalf("Failed to fetch laptop: %v", err)
	}
	if laptop.Price != 1000.0 { // Should be unchanged
		t.Errorf("Laptop price should be unchanged, got %v", laptop.Price)
	}

	// Test overload 1: Apply 5% discount to all products
	body = []byte(`{"percentage": 5.0}`)
	req = httptest.NewRequest(http.MethodPost, "/ApplyDiscount", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify all prices were changed
	if err := db.First(&laptop, 1).Error; err != nil {
		t.Fatalf("Failed to fetch laptop: %v", err)
	}
	expectedLaptopPrice := 950.0 // 1000 - 5%
	if laptop.Price != expectedLaptopPrice {
		t.Errorf("Laptop price = %v, want %v", laptop.Price, expectedLaptopPrice)
	}

	if err := db.First(&desk, 4).Error; err != nil {
		t.Fatalf("Failed to fetch desk: %v", err)
	}
	expectedDeskPrice2 := 171.0 // 180 - 5%
	if desk.Price != expectedDeskPrice2 {
		t.Errorf("Desk price = %v, want %v", desk.Price, expectedDeskPrice2)
	}
}

// TestRealWorldScenario_BoundFunctionOverload demonstrates bound function overloads
func TestRealWorldScenario_BoundFunctionOverload(t *testing.T) {
	service, _ := setupRealWorldTestService(t)

	// Bound function overload 1: Calculate discount price with percentage
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:      "CalculatePrice",
		IsBound:   true,
		EntitySet: "RealWorldProducts",
		Parameters: []odata.ParameterDefinition{
			{Name: "discount", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(float64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			product := ctx.(*RealWorldProduct)
			discount := params["discount"].(float64)
			return product.Price * (1.0 - discount/100.0), nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register first overload: %v", err)
	}

	// Bound function overload 2: Calculate price with tax and discount
	err = service.RegisterFunction(odata.FunctionDefinition{
		Name:      "CalculatePrice",
		IsBound:   true,
		EntitySet: "RealWorldProducts",
		Parameters: []odata.ParameterDefinition{
			{Name: "discount", Type: reflect.TypeOf(float64(0)), Required: true},
			{Name: "tax", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(float64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			product := ctx.(*RealWorldProduct)
			discount := params["discount"].(float64)
			tax := params["tax"].(float64)
			discountedPrice := product.Price * (1.0 - discount/100.0)
			return discountedPrice * (1.0 + tax/100.0), nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register second overload: %v", err)
	}

	// Test overload 1: Calculate price with 10% discount
	req := httptest.NewRequest(http.MethodGet, "/RealWorldProducts(1)/CalculatePrice?discount=10", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Overload 1: Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response1 map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response1); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	price1, ok := response1["value"].(float64)
	if !ok {
		t.Fatal("Expected float64 value")
	}
	expectedPrice1 := 900.0 // 1000 - 10%
	if price1 != expectedPrice1 {
		t.Errorf("Price = %v, want %v", price1, expectedPrice1)
	}

	// Test overload 2: Calculate price with 10% discount and 8% tax
	req = httptest.NewRequest(http.MethodGet, "/RealWorldProducts(1)/CalculatePrice?discount=10&tax=8", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Overload 2: Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response2 map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response2); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	price2, ok := response2["value"].(float64)
	if !ok {
		t.Fatal("Expected float64 value")
	}
	expectedPrice2 := 972.0 // (1000 - 10%) + 8% tax
	// Use approximate comparison for floating point
	if price2 < expectedPrice2-0.01 || price2 > expectedPrice2+0.01 {
		t.Errorf("Price = %v, want approximately %v", price2, expectedPrice2)
	}
}
