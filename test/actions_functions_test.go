package odata_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ActionTestProduct is a test entity for actions and functions tests
type ActionTestProduct struct {
	ID    uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name  string  `json:"Name"`
	Price float64 `json:"Price"`
}

// setupActionFunctionTestService sets up a test service with actions and functions
func setupActionFunctionTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ActionTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	products := []ActionTestProduct{
		{ID: 1, Name: "Laptop", Price: 1000.0},
		{ID: 2, Name: "Mouse", Price: 25.0},
		{ID: 3, Name: "Keyboard", Price: 75.0},
	}
	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to seed database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&ActionTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestUnboundFunction tests unbound function invocation
func TestUnboundFunction(t *testing.T) {
	service, db := setupActionFunctionTestService(t)

	// Register an unbound function
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:       "GetProductCount",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: reflect.TypeOf(int64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			var count int64
			db.Model(&ActionTestProduct{}).Count(&count)
			return count, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register function: %v", err)
	}

	// Test function invocation
	req := httptest.NewRequest(http.MethodGet, "/GetProductCount", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["@odata.context"]; !ok {
		t.Error("Response missing @odata.context")
	}

	value, ok := response["value"]
	if !ok {
		t.Fatal("Response missing value")
	}

	// The value should be 3 (number of seeded products)
	if valueFloat, ok := value.(float64); ok {
		if int(valueFloat) != 3 {
			t.Errorf("Expected 3 products, got %v", valueFloat)
		}
	}
}

// TestBoundFunction tests bound function invocation
func TestBoundFunction(t *testing.T) {
	service, db := setupActionFunctionTestService(t)

	// Register a bound function
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:      "GetDiscountedPrice",
		IsBound:   true,
		EntitySet: "ActionTestProducts",
		Parameters: []odata.ParameterDefinition{
			{Name: "discount", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(float64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			discount := params["discount"].(float64)

			// For test purposes, get product from path
			var product ActionTestProduct
			if err := db.First(&product, 1).Error; err != nil {
				return nil, err
			}

			return product.Price * (1 - discount/100), nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register function: %v", err)
	}

	// Test function invocation
	req := httptest.NewRequest(http.MethodGet, "/ActionTestProducts(1)/GetDiscountedPrice?discount=10", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify OData-Version header
	if odataVersion := w.Header().Get("OData-Version"); odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", odataVersion)
	}

	// Verify response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["@odata.context"]; !ok {
		t.Error("Response missing @odata.context")
	}
}

// TestUnboundAction tests unbound action invocation
func TestUnboundAction(t *testing.T) {
	service, db := setupActionFunctionTestService(t)

	// Register an unbound action
	err := service.RegisterAction(odata.ActionDefinition{
		Name:       "ResetPrices",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			// Reset all product prices to 100
			if err := db.Model(&ActionTestProduct{}).Where("1 = 1").Update("price", 100.0).Error; err != nil {
				return err
			}

			w.Header()["OData-Version"] = []string{"4.0"}
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register action: %v", err)
	}

	// Test action invocation
	req := httptest.NewRequest(http.MethodPost, "/ResetPrices", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify all prices were reset
	var products []ActionTestProduct
	if err := db.Find(&products).Error; err != nil {
		t.Fatalf("Failed to fetch products: %v", err)
	}

	for _, p := range products {
		if p.Price != 100.0 {
			t.Errorf("Product %d price = %v, want 100.0", p.ID, p.Price)
		}
	}
}

// TestBoundAction tests bound action invocation
func TestBoundAction(t *testing.T) {
	service, db := setupActionFunctionTestService(t)

	// Register a bound action
	err := service.RegisterAction(odata.ActionDefinition{
		Name:      "UpdatePrice",
		IsBound:   true,
		EntitySet: "ActionTestProducts",
		Parameters: []odata.ParameterDefinition{
			{Name: "newPrice", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			newPrice := params["newPrice"].(float64)

			// For test purposes, update product 1
			if err := db.Model(&ActionTestProduct{}).Where("id = ?", 1).Update("price", newPrice).Error; err != nil {
				return err
			}

			w.Header()["OData-Version"] = []string{"4.0"}
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register action: %v", err)
	}

	// Test action invocation
	body := []byte(`{"newPrice": 500.0}`)
	req := httptest.NewRequest(http.MethodPost, "/ActionTestProducts(1)/UpdatePrice", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify price was updated
	var product ActionTestProduct
	if err := db.First(&product, 1).Error; err != nil {
		t.Fatalf("Failed to fetch product: %v", err)
	}

	if product.Price != 500.0 {
		t.Errorf("Product price = %v, want 500.0", product.Price)
	}
}

// TestActionRequiresPost tests that actions require POST method
func TestActionRequiresPost(t *testing.T) {
	service, _ := setupActionFunctionTestService(t)

	// Register an action
	err := service.RegisterAction(odata.ActionDefinition{
		Name:       "TestAction",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register action: %v", err)
	}

	// Try to invoke with GET
	req := httptest.NewRequest(http.MethodGet, "/TestAction", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Actions invoked with GET should return 404 (not found as function) or 405 (method not allowed)
	// Current implementation returns 404 which is acceptable
	if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v or %v", w.Code, http.StatusNotFound, http.StatusMethodNotAllowed)
	}
}

// TestFunctionRequiresGet tests that functions require GET method
func TestFunctionRequiresGet(t *testing.T) {
	service, _ := setupActionFunctionTestService(t)

	// Register a function
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:       "TestFunction",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			return "test", nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register function: %v", err)
	}

	// Try to invoke with POST
	req := httptest.NewRequest(http.MethodPost, "/TestFunction", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Functions invoked with POST should return 404 (not found as action) or 405 (method not allowed)
	// Current implementation returns 404 which is acceptable
	if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v or %v", w.Code, http.StatusNotFound, http.StatusMethodNotAllowed)
	}
}

// TestFunctionWithParameters tests function with parameters
func TestFunctionWithParameters(t *testing.T) {
	service, _ := setupActionFunctionTestService(t)

	// Register a function with parameters
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "AddNumbers",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "a", Type: reflect.TypeOf(int64(0)), Required: true},
			{Name: "b", Type: reflect.TypeOf(int64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(int64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			a := params["a"].(int64)
			b := params["b"].(int64)
			return a + b, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register function: %v", err)
	}

	// Test function invocation
	req := httptest.NewRequest(http.MethodGet, "/AddNumbers?a=5&b=3", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"]
	if !ok {
		t.Fatal("Response missing value")
	}

	if valueFloat, ok := value.(float64); ok {
		if int(valueFloat) != 8 {
			t.Errorf("Expected 8, got %v", valueFloat)
		}
	}
}

// TestActionWithParameters tests action with parameters
func TestActionWithParameters(t *testing.T) {
	service, db := setupActionFunctionTestService(t)

	// Register an action with parameters
	err := service.RegisterAction(odata.ActionDefinition{
		Name:    "SetPrice",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "productId", Type: reflect.TypeOf(int64(0)), Required: true},
			{Name: "price", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			productID := int(params["productId"].(float64))
			price := params["price"].(float64)

			if err := db.Model(&ActionTestProduct{}).Where("id = ?", productID).Update("price", price).Error; err != nil {
				return err
			}

			w.Header()["OData-Version"] = []string{"4.0"}
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register action: %v", err)
	}

	// Test action invocation
	body := []byte(`{"productId": 1, "price": 250.0}`)
	req := httptest.NewRequest(http.MethodPost, "/SetPrice", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify price was updated
	var product ActionTestProduct
	if err := db.First(&product, 1).Error; err != nil {
		t.Fatalf("Failed to fetch product: %v", err)
	}

	if product.Price != 250.0 {
		t.Errorf("Product price = %v, want 250.0", product.Price)
	}
}

// TestActionNotFound tests that invoking a non-existent action returns 404
func TestActionNotFound(t *testing.T) {
	service, _ := setupActionFunctionTestService(t)

	req := httptest.NewRequest(http.MethodPost, "/NonExistentAction", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

// TestFunctionNotFound tests that invoking a non-existent function returns 404
func TestFunctionNotFound(t *testing.T) {
	service, _ := setupActionFunctionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/NonExistentFunction", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

// TestMissingRequiredParameter tests that missing required parameters return error
func TestMissingRequiredParameter(t *testing.T) {
	service, _ := setupActionFunctionTestService(t)

	// Register a function with required parameter
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "TestRequiredParam",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "required", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			return params["required"], nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register function: %v", err)
	}

	// Try to invoke without the required parameter
	req := httptest.NewRequest(http.MethodGet, "/TestRequiredParam", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

// TestBoundActionOnWrongEntitySet tests that bound action on wrong entity set returns error
func TestBoundActionOnWrongEntitySet(t *testing.T) {
	service, _ := setupActionFunctionTestService(t)

	// Register a bound action for ActionTestProducts
	err := service.RegisterAction(odata.ActionDefinition{
		Name:       "TestBoundAction",
		IsBound:    true,
		EntitySet:  "ActionTestProducts",
		Parameters: []odata.ParameterDefinition{},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register action: %v", err)
	}

	// Register another entity set
	type OtherEntity struct {
		ID uint `json:"ID" gorm:"primaryKey" odata:"key"`
	}
	if err := service.RegisterEntity(&OtherEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Try to invoke the action on the wrong entity set
	// Since TestBoundAction is not bound to OtherEntities, it should fail
	// However, since it's not recognized as a navigation property, it will return property not found
	req := httptest.NewRequest(http.MethodPost, "/OtherEntities(1)/TestBoundAction", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 404 or 400 (property not found or invalid binding)
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v or %v", w.Code, http.StatusNotFound, http.StatusBadRequest)
	}
}

// TestActionReturningValue tests action that returns a value
func TestActionReturningValue(t *testing.T) {
	service, _ := setupActionFunctionTestService(t)

	// Register an action that returns a value
	err := service.RegisterAction(odata.ActionDefinition{
		Name:       "GetMessage",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
			w.Header()["OData-Version"] = []string{"4.0"}
			w.WriteHeader(http.StatusOK)

			response := map[string]interface{}{
				"@odata.context": "$metadata#Edm.String",
				"value":          "Hello from action",
			}

			return json.NewEncoder(w).Encode(response)
		},
	})
	if err != nil {
		t.Fatalf("Failed to register action: %v", err)
	}

	// Test action invocation
	req := httptest.NewRequest(http.MethodPost, "/GetMessage", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify response
	body, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if value, ok := response["value"]; !ok || value != "Hello from action" {
		t.Errorf("Expected 'Hello from action', got %v", value)
	}
}
