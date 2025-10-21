package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ConcatTestProduct represents a product entity for testing concat functionality
type ConcatTestProduct struct {
	ID       uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string `json:"Name" gorm:"not null"`
	Category string `json:"Category" gorm:"not null"`
	Prefix   string `json:"Prefix"`
	Suffix   string `json:"Suffix"`
}

func setupConcatTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the models
	if err := db.AutoMigrate(&ConcatTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	products := []ConcatTestProduct{
		{ID: 1, Name: "Laptop", Category: "Electronics", Prefix: "PRO", Suffix: "2024"},
		{ID: 2, Name: "Mouse", Category: "Accessories", Prefix: "STD", Suffix: "2023"},
		{ID: 3, Name: "Keyboard", Category: "Accessories", Prefix: "MECH", Suffix: "2024"},
		{ID: 4, Name: "Monitor", Category: "Electronics", Prefix: "HD", Suffix: "2024"},
		{ID: 5, Name: "Cable", Category: "Accessories", Prefix: "", Suffix: ""},
	}

	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to seed products: %v", err)
	}

	return db
}

func TestConcat_TwoLiterals(t *testing.T) {
	db := setupConcatTestDB(t)
	service := odata.NewService(db)

	err := service.RegisterEntity(ConcatTestProduct{})
	if err != nil {
		t.Fatalf("Failed to register entity set: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test concat with two empty strings
	encodedFilter := url.QueryEscape("concat('','') eq ''")
	resp, err := http.Get(server.URL + "/ConcatTestProducts?$filter=" + encodedFilter)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should match all products since concat('','') eq '' is always true
	value := result["value"].([]interface{})
	if len(value) != 5 {
		t.Errorf("Expected 5 products, got %d", len(value))
	}
}

func TestConcat_LiteralAndProperty(t *testing.T) {
	db := setupConcatTestDB(t)
	service := odata.NewService(db)

	err := service.RegisterEntity(ConcatTestProduct{})
	if err != nil {
		t.Fatalf("Failed to register entity set: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test concat with literal prefix and property
	encodedFilter := url.QueryEscape("concat('Product: ', Name) eq 'Product: Laptop'")
	resp, err := http.Get(server.URL + "/ConcatTestProducts?$filter=" + encodedFilter)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value := result["value"].([]interface{})
	if len(value) != 1 {
		t.Errorf("Expected 1 product, got %d", len(value))
	}

	if len(value) > 0 {
		product := value[0].(map[string]interface{})
		if product["Name"] != "Laptop" {
			t.Errorf("Expected Name 'Laptop', got '%v'", product["Name"])
		}
	}
}

func TestConcat_TwoLiteralsNonEmpty(t *testing.T) {
	db := setupConcatTestDB(t)
	service := odata.NewService(db)

	err := service.RegisterEntity(ConcatTestProduct{})
	if err != nil {
		t.Fatalf("Failed to register entity set: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test concat with two non-empty literal strings
	encodedFilter := url.QueryEscape("concat('Hello', ' World') eq 'Hello World'")
	resp, err := http.Get(server.URL + "/ConcatTestProducts?$filter=" + encodedFilter)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should match all products since the condition is always true
	value := result["value"].([]interface{})
	if len(value) != 5 {
		t.Errorf("Expected 5 products (condition is always true), got %d", len(value))
	}
}

func TestConcat_LiteralFirstEmptyString(t *testing.T) {
	db := setupConcatTestDB(t)
	service := odata.NewService(db)

	err := service.RegisterEntity(ConcatTestProduct{})
	if err != nil {
		t.Fatalf("Failed to register entity set: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test concat with empty literal and property - should equal the property value
	encodedFilter := url.QueryEscape("concat('', Name) eq 'Mouse'")
	resp, err := http.Get(server.URL + "/ConcatTestProducts?$filter=" + encodedFilter)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value := result["value"].([]interface{})
	if len(value) != 1 {
		t.Errorf("Expected 1 product, got %d", len(value))
	}

	if len(value) > 0 {
		product := value[0].(map[string]interface{})
		if product["Name"] != "Mouse" {
			t.Errorf("Expected Name 'Mouse', got '%v'", product["Name"])
		}
	}
}

func TestConcat_PropertyAndProperty(t *testing.T) {
	db := setupConcatTestDB(t)
	service := odata.NewService(db)

	err := service.RegisterEntity(ConcatTestProduct{})
	if err != nil {
		t.Fatalf("Failed to register entity set: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test concat with two properties
	encodedFilter := url.QueryEscape("concat(Prefix, Suffix) eq 'PRO2024'")
	resp, err := http.Get(server.URL + "/ConcatTestProducts?$filter=" + encodedFilter)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value := result["value"].([]interface{})
	if len(value) != 1 {
		t.Errorf("Expected 1 product, got %d", len(value))
	}

	if len(value) > 0 {
		product := value[0].(map[string]interface{})
		if product["Name"] != "Laptop" {
			t.Errorf("Expected Name 'Laptop', got '%v'", product["Name"])
		}
	}
}

func TestConcat_NestedWithPropertyAndLiteral(t *testing.T) {
	db := setupConcatTestDB(t)
	service := odata.NewService(db)

	err := service.RegisterEntity(ConcatTestProduct{})
	if err != nil {
		t.Fatalf("Failed to register entity set: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test nested concat with property in inner concat and literal in outer
	encodedFilter := url.QueryEscape("concat(concat(Name, ' - '), Category) eq 'Laptop - Electronics'")
	resp, err := http.Get(server.URL + "/ConcatTestProducts?$filter=" + encodedFilter)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value := result["value"].([]interface{})
	if len(value) != 1 {
		t.Errorf("Expected 1 product, got %d", len(value))
	}

	if len(value) > 0 {
		product := value[0].(map[string]interface{})
		if product["Name"] != "Laptop" {
			t.Errorf("Expected Name 'Laptop', got '%v'", product["Name"])
		}
	}
}

func TestConcat_LiteralWithFunctionCall(t *testing.T) {
	db := setupConcatTestDB(t)
	service := odata.NewService(db)

	err := service.RegisterEntity(ConcatTestProduct{})
	if err != nil {
		t.Fatalf("Failed to register entity set: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test concat with literal and function call
	encodedFilter := url.QueryEscape("concat('Category: ', tolower(Category)) eq 'Category: electronics'")
	resp, err := http.Get(server.URL + "/ConcatTestProducts?$filter=" + encodedFilter)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value := result["value"].([]interface{})
	if len(value) != 2 {
		t.Errorf("Expected 2 products (Electronics category), got %d", len(value))
	}
}

func TestConcat_SpecialCharactersInLiterals(t *testing.T) {
	db := setupConcatTestDB(t)
	service := odata.NewService(db)

	err := service.RegisterEntity(ConcatTestProduct{})
	if err != nil {
		t.Fatalf("Failed to register entity set: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test concat with special characters in literals
	encodedFilter := url.QueryEscape("concat('Test!', '@#$') eq 'Test!@#$'")
	resp, err := http.Get(server.URL + "/ConcatTestProducts?$filter=" + encodedFilter)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should match all products since the condition is always true
	value := result["value"].([]interface{})
	if len(value) != 5 {
		t.Errorf("Expected 5 products (condition is always true), got %d", len(value))
	}
}

func TestConcat_ComplexExpression(t *testing.T) {
	db := setupConcatTestDB(t)
	service := odata.NewService(db)

	err := service.RegisterEntity(ConcatTestProduct{})
	if err != nil {
		t.Fatalf("Failed to register entity set: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(service.ServeHTTP))
	defer server.Close()

	// Test concat in complex expression with OR
	encodedFilter := url.QueryEscape("concat('test', 'value') eq 'testvalue' or Name eq 'NonExistent'")
	resp, err := http.Get(server.URL + "/ConcatTestProducts?$filter=" + encodedFilter)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should match all products since first condition is always true
	value := result["value"].([]interface{})
	if len(value) != 5 {
		t.Errorf("Expected 5 products (first condition is always true), got %d", len(value))
	}
}
