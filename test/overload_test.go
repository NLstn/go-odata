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

// OverloadTestProduct is a test entity for overload tests
type OverloadTestProduct struct {
	ID    uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name  string  `json:"Name"`
	Price float64 `json:"Price"`
}

// setupOverloadTestService sets up a test service
func setupOverloadTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&OverloadTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	products := []OverloadTestProduct{
		{ID: 1, Name: "Laptop", Price: 1000.0},
		{ID: 2, Name: "Mouse", Price: 25.0},
		{ID: 3, Name: "Keyboard", Price: 75.0},
	}
	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to seed database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&OverloadTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestFunctionOverloadDifferentParameterCount tests function overloads with different parameter counts
func TestFunctionOverloadDifferentParameterCount(t *testing.T) {
	service, _ := setupOverloadTestService(t)

	// Register first overload: no parameters
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:       "Calculate",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: reflect.TypeOf(int64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			return int64(0), nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register first function overload: %v", err)
	}

	// Register second overload: one parameter
	err = service.RegisterFunction(odata.FunctionDefinition{
		Name:    "Calculate",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "value", Type: reflect.TypeOf(int64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(int64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			value := params["value"].(int64)
			return value * 2, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register second function overload: %v", err)
	}

	// Register third overload: two parameters
	err = service.RegisterFunction(odata.FunctionDefinition{
		Name:    "Calculate",
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
		t.Fatalf("Failed to register third function overload: %v", err)
	}

	// Test first overload (no parameters)
	req := httptest.NewRequest(http.MethodGet, "/Calculate()", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Overload 1: Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if value, ok := response["value"].(float64); !ok || int(value) != 0 {
		t.Errorf("Expected 0, got %v", value)
	}

	// Test second overload (one parameter)
	req = httptest.NewRequest(http.MethodGet, "/Calculate()?value=5", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Overload 2: Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if value, ok := response["value"].(float64); !ok || int(value) != 10 {
		t.Errorf("Expected 10, got %v", value)
	}

	// Test third overload (two parameters)
	req = httptest.NewRequest(http.MethodGet, "/Calculate()?a=3&b=7", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Overload 3: Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if value, ok := response["value"].(float64); !ok || int(value) != 10 {
		t.Errorf("Expected 10, got %v", value)
	}
}

// TestActionOverloadDifferentParameterCount tests action overloads with different parameter counts
func TestActionOverloadDifferentParameterCount(t *testing.T) {
	service, db := setupOverloadTestService(t)

	// Register first overload: no parameters
	err := service.RegisterAction(odata.ActionDefinition{
		Name:       "Process",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			// Reset all prices to 100
			if err := db.Model(&OverloadTestProduct{}).Where("1 = 1").Update("price", 100.0).Error; err != nil {
				return err
			}
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register first action overload: %v", err)
	}

	// Register second overload: one parameter
	err = service.RegisterAction(odata.ActionDefinition{
		Name:    "Process",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "price", Type: reflect.TypeOf(float64(0)), Required: true},
		},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			price := params["price"].(float64)
			// Set all prices to specified value
			if err := db.Model(&OverloadTestProduct{}).Where("1 = 1").Update("price", price).Error; err != nil {
				return err
			}
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register second action overload: %v", err)
	}

	// Test first overload (no parameters)
	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/Process", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Overload 1: Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify all prices are 100
	var products []OverloadTestProduct
	if err := db.Find(&products).Error; err != nil {
		t.Fatalf("Failed to fetch products: %v", err)
	}
	for _, p := range products {
		if p.Price != 100.0 {
			t.Errorf("Product %d price = %v, want 100.0", p.ID, p.Price)
		}
	}

	// Test second overload (one parameter)
	body = []byte(`{"price": 250.0}`)
	req = httptest.NewRequest(http.MethodPost, "/Process", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Overload 2: Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify all prices are 250
	if err := db.Find(&products).Error; err != nil {
		t.Fatalf("Failed to fetch products: %v", err)
	}
	for _, p := range products {
		if p.Price != 250.0 {
			t.Errorf("Product %d price = %v, want 250.0", p.ID, p.Price)
		}
	}
}

// TestBoundFunctionOverload tests bound function overloads on different entity sets
func TestBoundFunctionOverload(t *testing.T) {
	service, _ := setupOverloadTestService(t)

	// Define another entity type
	type OtherEntity struct {
		ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
		Name string `json:"Name"`
	}

	// Register the other entity
	if err := service.RegisterEntity(&OtherEntity{}); err != nil {
		t.Fatalf("Failed to register OtherEntity: %v", err)
	}

	// Register bound function for OverloadTestProducts
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:      "GetInfo",
		IsBound:   true,
		EntitySet: "OverloadTestProducts",
		Parameters: []odata.ParameterDefinition{
			{Name: "format", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			format := params["format"].(string)
			return "Product info in " + format, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register bound function for OverloadTestProducts: %v", err)
	}

	// Register bound function with same name for OtherEntities
	err = service.RegisterFunction(odata.FunctionDefinition{
		Name:      "GetInfo",
		IsBound:   true,
		EntitySet: "OtherEntities",
		Parameters: []odata.ParameterDefinition{
			{Name: "format", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			format := params["format"].(string)
			return "Other entity info in " + format, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register bound function for OtherEntities: %v", err)
	}

	// Test bound function on OverloadTestProducts
	req := httptest.NewRequest(http.MethodGet, "/OverloadTestProducts(1)/GetInfo?format=json", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if value, ok := response["value"].(string); !ok || value != "Product info in json" {
		t.Errorf("Expected 'Product info in json', got %v", value)
	}
}

// TestDuplicateOverloadRejection tests that registering duplicate overloads is rejected
func TestDuplicateOverloadRejection(t *testing.T) {
	service, _ := setupOverloadTestService(t)

	// Register a function
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "Duplicate",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "value", Type: reflect.TypeOf(int64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(int64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			return int64(1), nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register function: %v", err)
	}

	// Try to register the same function again (same signature)
	err = service.RegisterFunction(odata.FunctionDefinition{
		Name:    "Duplicate",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "value", Type: reflect.TypeOf(int64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(int64(0)),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			return int64(2), nil
		},
	})
	if err == nil {
		t.Error("Expected error when registering duplicate function, got nil")
	}

	// Register an action
	err = service.RegisterAction(odata.ActionDefinition{
		Name:    "DuplicateAction",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "name", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register action: %v", err)
	}

	// Try to register the same action again (same signature)
	err = service.RegisterAction(odata.ActionDefinition{
		Name:    "DuplicateAction",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "name", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
	})
	if err == nil {
		t.Error("Expected error when registering duplicate action, got nil")
	}
}

// TestFunctionOverloadDifferentParameterTypes tests function overloads with different parameter types
func TestFunctionOverloadDifferentParameterTypes(t *testing.T) {
	service, _ := setupOverloadTestService(t)

	// Register first overload: string parameter
	err := service.RegisterFunction(odata.FunctionDefinition{
		Name:    "Convert",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "input", Type: reflect.TypeOf(""), Required: true},
		},
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			input := params["input"].(string)
			return "String: " + input, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register first function overload: %v", err)
	}

	// Register second overload: int parameter
	err = service.RegisterFunction(odata.FunctionDefinition{
		Name:    "Convert",
		IsBound: false,
		Parameters: []odata.ParameterDefinition{
			{Name: "number", Type: reflect.TypeOf(int64(0)), Required: true},
		},
		ReturnType: reflect.TypeOf(""),
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
			number := params["number"].(int64)
			return "Number: " + string(rune(number+48)), nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to register second function overload: %v", err)
	}

	// Test first overload (string parameter)
	req := httptest.NewRequest(http.MethodGet, "/Convert()?input=hello", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Overload 1: Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if value, ok := response["value"].(string); !ok || value != "String: hello" {
		t.Errorf("Expected 'String: hello', got %v", value)
	}

	// Test second overload (int parameter)
	req = httptest.NewRequest(http.MethodGet, "/Convert()?number=5", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Overload 2: Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if value, ok := response["value"].(string); !ok || value != "Number: 5" {
		t.Errorf("Expected 'Number: 5', got %v", value)
	}
}
