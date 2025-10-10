package odata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TestProductForStructuralProp struct {
	ID          int     `json:"id" gorm:"primarykey" odata:"key"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	InStock     bool    `json:"inStock"`
}

func setupStructuralPropTestService(t *testing.T) (*Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestProductForStructuralProp{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := NewService(db)
	if err := service.RegisterEntity(TestProductForStructuralProp{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestStructuralPropertyRead_String(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/name", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check for @odata.context
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Response missing @odata.context")
	}

	// Check the value
	if response["value"] != "Laptop" {
		t.Errorf("value = %v, want 'Laptop'", response["value"])
	}
}

func TestStructuralPropertyRead_Number(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/price", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check for @odata.context
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Response missing @odata.context")
	}

	// Check the value
	if response["value"] != float64(999.99) {
		t.Errorf("value = %v, want 999.99", response["value"])
	}
}

func TestStructuralPropertyRead_Boolean(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/inStock", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check for @odata.context
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Response missing @odata.context")
	}

	// Check the value
	if response["value"] != true {
		t.Errorf("value = %v, want true", response["value"])
	}
}

func TestStructuralPropertyRead_EntityNotFound(t *testing.T) {
	service, _ := setupStructuralPropTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(999)/name", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

func TestStructuralPropertyRead_PropertyNotFound(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/nonexistent", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

func TestStructuralPropertyRead_MethodNotAllowed(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
	}
	db.Create(&product)

	// Test POST method (should be not allowed)
	req := httptest.NewRequest(http.MethodPost, "/TestProductForStructuralProps(1)/name", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestStructuralPropertyRead_UsingFieldName(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
	}
	db.Create(&product)

	// Try using the struct field name instead of JSON name
	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/Name", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check the value
	if response["value"] != "Laptop" {
		t.Errorf("value = %v, want 'Laptop'", response["value"])
	}
}
