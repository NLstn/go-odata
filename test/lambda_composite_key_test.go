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

// OrderHeader is a parent entity with composite keys
type OrderHeader struct {
	OrderID  uint        `json:"OrderID" gorm:"primaryKey" odata:"key"`
	Version  string      `json:"Version" gorm:"primaryKey;size:10" odata:"key"`
	Customer string      `json:"Customer" gorm:"not null"`
	Lines    []OrderLine `json:"Lines" gorm:"foreignKey:OrderID,Version;references:OrderID,Version"`
}

// OrderLine is a child entity that references the parent's composite key
type OrderLine struct {
	LineID      uint         `json:"LineID" gorm:"primaryKey" odata:"key"`
	OrderID     uint         `json:"OrderID" gorm:"not null"`
	Version     string       `json:"Version" gorm:"size:10;not null"`
	ProductName string       `json:"ProductName" gorm:"not null"`
	Quantity    int          `json:"Quantity" gorm:"not null"`
	UnitPrice   float64      `json:"UnitPrice" gorm:"not null"`
	Order       *OrderHeader `json:"Order,omitempty" gorm:"foreignKey:OrderID,Version;references:OrderID,Version"`
}

func setupLambdaCompositeKeyTest(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&OrderHeader{}, &OrderLine{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	// Order 1, Version v1 with 2 lines
	order1v1 := OrderHeader{OrderID: 1, Version: "v1", Customer: "Alice"}
	db.Create(&order1v1)
	db.Create(&OrderLine{LineID: 1, OrderID: 1, Version: "v1", ProductName: "Laptop", Quantity: 1, UnitPrice: 1000.0})
	db.Create(&OrderLine{LineID: 2, OrderID: 1, Version: "v1", ProductName: "Mouse", Quantity: 2, UnitPrice: 25.0})

	// Order 1, Version v2 with 1 line (different version of same order)
	order1v2 := OrderHeader{OrderID: 1, Version: "v2", Customer: "Alice"}
	db.Create(&order1v2)
	db.Create(&OrderLine{LineID: 3, OrderID: 1, Version: "v2", ProductName: "Keyboard", Quantity: 1, UnitPrice: 75.0})

	// Order 2, Version v1 with 1 line
	order2v1 := OrderHeader{OrderID: 2, Version: "v1", Customer: "Bob"}
	db.Create(&order2v1)
	db.Create(&OrderLine{LineID: 4, OrderID: 2, Version: "v1", ProductName: "Monitor", Quantity: 1, UnitPrice: 300.0})

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(&OrderHeader{}); err != nil {
		t.Fatalf("Failed to register OrderHeader entity: %v", err)
	}

	if err := service.RegisterEntity(&OrderLine{}); err != nil {
		t.Fatalf("Failed to register OrderLine entity: %v", err)
	}

	return service, db
}

// TestLambdaAny_CompositeKeyParent tests lambda any with parent entity having composite keys
func TestLambdaAny_CompositeKeyParent(t *testing.T) {
	service, _ := setupLambdaCompositeKeyTest(t)

	// Test: Find orders that have a line with ProductName = "Laptop"
	// Only Order 1 Version v1 should match (not Order 1 Version v2, not Order 2 Version v1)
	req := httptest.NewRequest(http.MethodGet, "/OrderHeaders?$filter=Lines/any(l:%20l/ProductName%20eq%20'Laptop')", nil)
	rec := httptest.NewRecorder()
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Response missing 'value' array")
	}

	if len(value) != 1 {
		t.Fatalf("Expected 1 order, got %d", len(value))
	}

	order := value[0].(map[string]interface{})
	if order["OrderID"].(float64) != 1 {
		t.Errorf("Expected OrderID=1, got %v", order["OrderID"])
	}
	if order["Version"].(string) != "v1" {
		t.Errorf("Expected Version=v1, got %v", order["Version"])
	}
}

// TestLambdaAny_CompositeKeyParent_DifferentVersions tests that different versions are distinguished
func TestLambdaAny_CompositeKeyParent_DifferentVersions(t *testing.T) {
	service, _ := setupLambdaCompositeKeyTest(t)

	// Test: Find orders that have a line with ProductName = "Keyboard"
	// Only Order 1 Version v2 should match
	req := httptest.NewRequest(http.MethodGet, "/OrderHeaders?$filter=Lines/any(l:%20l/ProductName%20eq%20'Keyboard')", nil)
	rec := httptest.NewRecorder()
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Response missing 'value' array")
	}

	if len(value) != 1 {
		t.Fatalf("Expected 1 order, got %d", len(value))
	}

	order := value[0].(map[string]interface{})
	if order["OrderID"].(float64) != 1 {
		t.Errorf("Expected OrderID=1, got %v", order["OrderID"])
	}
	if order["Version"].(string) != "v2" {
		t.Errorf("Expected Version=v2, got %v", order["Version"])
	}
}

// TestLambdaAll_CompositeKeyParent tests lambda all with composite key parent
func TestLambdaAll_CompositeKeyParent(t *testing.T) {
	service, _ := setupLambdaCompositeKeyTest(t)

	// Test: Find orders where ALL lines have Quantity > 0
	// All orders should match (all lines have positive quantity)
	req := httptest.NewRequest(http.MethodGet, "/OrderHeaders?$filter=Lines/all(l:%20l/Quantity%20gt%200)", nil)
	rec := httptest.NewRecorder()
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Response missing 'value' array")
	}

	// Should return all 3 orders (Order 1 v1, Order 1 v2, Order 2 v1)
	if len(value) != 3 {
		t.Fatalf("Expected 3 orders, got %d", len(value))
	}
}

// TestLambdaAny_CompositeKeyParent_NoMatch tests lambda any with no matching results
func TestLambdaAny_CompositeKeyParent_NoMatch(t *testing.T) {
	service, _ := setupLambdaCompositeKeyTest(t)

	// Test: Find orders that have a line with ProductName = "NonExistent"
	// No orders should match
	req := httptest.NewRequest(http.MethodGet, "/OrderHeaders?$filter=Lines/any(l:%20l/ProductName%20eq%20'NonExistent')", nil)
	rec := httptest.NewRecorder()
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Response missing 'value' array")
	}

	if len(value) != 0 {
		t.Fatalf("Expected 0 orders, got %d", len(value))
	}
}
