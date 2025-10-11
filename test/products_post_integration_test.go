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

// Product mimics the entity from cmd/devserver for integration testing
type Product struct {
	ID           uint                 `json:"ID" gorm:"primaryKey" odata:"key"`
	Name         string               `json:"Name" gorm:"not null" odata:"required,maxlength=100"`
	Price        float64              `json:"Price" gorm:"not null" odata:"required,precision=10,scale=2"`
	Category     string               `json:"Category" gorm:"not null" odata:"required,maxlength=50"`
	Version      int                  `json:"Version" gorm:"default:1" odata:"etag"`
	Descriptions []ProductDescription `json:"Descriptions" gorm:"foreignKey:ProductID;references:ID"`
}

// ProductDescription mimics the entity from cmd/devserver for integration testing
type ProductDescription struct {
	ProductID   uint     `json:"ProductID" gorm:"primaryKey" odata:"key"`
	LanguageKey string   `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key,maxlength=2"`
	Description string   `json:"Description" gorm:"not null" odata:"required,maxlength=500"`
	LongText    string   `json:"LongText" gorm:"type:text" odata:"maxlength=2000,nullable"`
	Product     *Product `json:"Product,omitempty" gorm:"foreignKey:ProductID;references:ID"`
}

func setupProductsTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal("Failed to connect to database:", err)
	}

	if err := db.AutoMigrate(&Product{}, &ProductDescription{}); err != nil {
		t.Fatal("Failed to migrate database:", err)
	}

	// Seed some test data
	sampleProducts := []Product{
		{
			ID:       1,
			Name:     "Laptop",
			Price:    999.99,
			Category: "Electronics",
			Version:  1,
		},
		{
			ID:       2,
			Name:     "Wireless Mouse",
			Price:    29.99,
			Category: "Electronics",
			Version:  1,
		},
	}
	db.Create(&sampleProducts)

	service := odata.NewService(db)
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatal("Failed to register Product entity:", err)
	}
	if err := service.RegisterEntity(&ProductDescription{}); err != nil {
		t.Fatal("Failed to register ProductDescription entity:", err)
	}

	return service, db
}

// TestProductsPOST_ToCollectionEndpoint verifies POST to /Products returns 201 Created
func TestProductsPOST_ToCollectionEndpoint(t *testing.T) {
	service, _ := setupProductsTestService(t)

	newProduct := map[string]interface{}{
		"Name":     "Gaming Laptop",
		"Price":    1499.99,
		"Category": "Electronics",
		"Version":  1,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/Products", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify 201 Created status according to OData v4 spec
	if w.Code != http.StatusCreated {
		t.Errorf("POST /Products returned status %d, expected 201 Created. Body: %s",
			w.Code, w.Body.String())
	}

	// Verify OData-Version header
	if version := w.Header().Get("OData-Version"); version != "4.0" {
		t.Errorf("OData-Version header = %q, expected \"4.0\"", version)
	}

	// Verify Location header is present
	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Location header is missing in POST response")
	}

	// Verify response body contains the created entity
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if response["Name"] != "Gaming Laptop" {
		t.Errorf("Response Name = %v, expected 'Gaming Laptop'", response["Name"])
	}

	if response["Price"] != 1499.99 {
		t.Errorf("Response Price = %v, expected 1499.99", response["Price"])
	}
}

// TestProductsPOST_ToIndividualEntityEndpoint verifies POST to /Products(1) returns 405
func TestProductsPOST_ToIndividualEntityEndpoint(t *testing.T) {
	service, _ := setupProductsTestService(t)

	newProduct := map[string]interface{}{
		"Name":  "Should Not Work",
		"Price": 99.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/Products(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify 405 Method Not Allowed according to OData v4 spec
	// POST is only allowed on collections, not individual entities
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /Products(1) returned status %d, expected 405 Method Not Allowed. Body: %s",
			w.Code, w.Body.String())
	}
}

// TestProductsPOST_WithTrailingSlash verifies POST to /Products/ works
func TestProductsPOST_WithTrailingSlash(t *testing.T) {
	service, _ := setupProductsTestService(t)

	newProduct := map[string]interface{}{
		"Name":     "Test Product",
		"Price":    99.99,
		"Category": "Test",
		"Version":  1,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/Products/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should handle trailing slash and return 201 Created
	if w.Code != http.StatusCreated {
		t.Errorf("POST /Products/ returned status %d, expected 201 Created. Body: %s",
			w.Code, w.Body.String())
	}
}

// TestProductsPOST_WithMissingRequiredField verifies validation
func TestProductsPOST_WithMissingRequiredField(t *testing.T) {
	service, _ := setupProductsTestService(t)

	// Missing required "Name" field
	invalidProduct := map[string]interface{}{
		"Price":    99.99,
		"Category": "Test",
		"Version":  1,
	}
	body, _ := json.Marshal(invalidProduct)

	req := httptest.NewRequest(http.MethodPost, "/Products", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request for missing required field
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST with missing required field returned status %d, expected 400 Bad Request. Body: %s",
			w.Code, w.Body.String())
	}
}

// TestProductsPOST_WithInvalidJSON verifies JSON parsing error handling
func TestProductsPOST_WithInvalidJSON(t *testing.T) {
	service, _ := setupProductsTestService(t)

	req := httptest.NewRequest(http.MethodPost, "/Products",
		bytes.NewBuffer([]byte("invalid json content")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request for invalid JSON
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST with invalid JSON returned status %d, expected 400 Bad Request. Body: %s",
			w.Code, w.Body.String())
	}
}

// TestProductsPOST_WithETagField verifies ETag header generation
func TestProductsPOST_WithETagField(t *testing.T) {
	service, _ := setupProductsTestService(t)

	newProduct := map[string]interface{}{
		"Name":     "Laptop Pro",
		"Price":    2499.99,
		"Category": "Electronics",
		"Version":  1,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/Products", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST returned status %d, expected 201. Body: %s", w.Code, w.Body.String())
	}

	// Verify ETag header is present (Product has Version field with odata:"etag" tag)
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header is missing")
	}

	// ETag should be in weak format: W/"..."
	if len(etag) < 4 || etag[:3] != "W/\"" {
		t.Errorf("ETag format is incorrect: %s", etag)
	}
}

// TestProductsPOST_AndVerifyCreation verifies created entity can be retrieved
func TestProductsPOST_AndVerifyCreation(t *testing.T) {
	service, _ := setupProductsTestService(t)

	newProduct := map[string]interface{}{
		"Name":     "Mechanical Keyboard",
		"Price":    149.99,
		"Category": "Accessories",
		"Version":  1,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/Products", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST returned status %d, expected 201. Body: %s", w.Code, w.Body.String())
	}

	// Extract ID from response
	var postResponse map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&postResponse); err != nil {
		t.Fatalf("Failed to decode POST response: %v", err)
	}

	id, ok := postResponse["ID"].(float64)
	if !ok {
		t.Fatal("ID not found in POST response")
	}

	// Now GET the created entity to verify it exists
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/Products(%d)", int(id)), nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET created entity returned status %d, expected 200 OK. Body: %s",
			w.Code, w.Body.String())
	}

	var getResponse map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&getResponse); err != nil {
		t.Fatalf("Failed to decode GET response: %v", err)
	}

	if getResponse["Name"] != "Mechanical Keyboard" {
		t.Errorf("Retrieved entity Name = %v, expected 'Mechanical Keyboard'", getResponse["Name"])
	}
}

// TestProductsPOST_MultipleEntities verifies multiple sequential POST requests
func TestProductsPOST_MultipleEntities(t *testing.T) {
	service, db := setupProductsTestService(t)

	products := []map[string]interface{}{
		{
			"Name":     "Product A",
			"Price":    10.00,
			"Category": "Category A",
			"Version":  1,
		},
		{
			"Name":     "Product B",
			"Price":    20.00,
			"Category": "Category B",
			"Version":  1,
		},
		{
			"Name":     "Product C",
			"Price":    30.00,
			"Category": "Category C",
			"Version":  1,
		},
	}

	for _, product := range products {
		body, _ := json.Marshal(product)

		req := httptest.NewRequest(http.MethodPost, "/Products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("POST for product %v returned status %d, expected 201. Body: %s",
				product["Name"], w.Code, w.Body.String())
		}
	}

	// Verify all products were created
	var count int64
	db.Model(&Product{}).Count(&count)
	expectedCount := int64(2 + 3) // 2 seeded + 3 created
	if count != expectedCount {
		t.Errorf("Total products in database = %d, expected %d", count, expectedCount)
	}
}

// TestProductsGET_VerifyCollectionEndpoint ensures GET still works correctly
func TestProductsGET_VerifyCollectionEndpoint(t *testing.T) {
	service, _ := setupProductsTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify 200 OK status
	if w.Code != http.StatusOK {
		t.Errorf("GET /Products returned status %d, expected 200 OK. Body: %s",
			w.Code, w.Body.String())
	}

	// Verify response structure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Response does not contain 'value' array")
	}

	// Should have 2 seeded products
	if len(value) != 2 {
		t.Errorf("Expected 2 products in collection, got %d", len(value))
	}
}

// TestProductsPOST_WithPreferReturnMinimal verifies Prefer: return=minimal header
func TestProductsPOST_WithPreferReturnMinimal(t *testing.T) {
	service, _ := setupProductsTestService(t)

	newProduct := map[string]interface{}{
		"Name":     "Minimal Response Product",
		"Price":    59.99,
		"Category": "Test",
		"Version":  1,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/Products", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content with Prefer: return=minimal
	if w.Code != http.StatusNoContent {
		t.Errorf("POST with Prefer: return=minimal returned status %d, expected 204 No Content. Body: %s",
			w.Code, w.Body.String())
	}

	// Location header should still be present
	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Location header is missing")
	}

	// Preference-Applied header should be present
	preferenceApplied := w.Header().Get("Preference-Applied")
	if preferenceApplied != "return=minimal" {
		t.Errorf("Preference-Applied header = %q, expected 'return=minimal'", preferenceApplied)
	}
}

// TestProductsPOST_WithPreferReturnRepresentation verifies Prefer: return=representation header
func TestProductsPOST_WithPreferReturnRepresentation(t *testing.T) {
	service, _ := setupProductsTestService(t)

	newProduct := map[string]interface{}{
		"Name":     "Full Response Product",
		"Price":    79.99,
		"Category": "Test",
		"Version":  1,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/Products", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 201 Created with full representation (default behavior)
	if w.Code != http.StatusCreated {
		t.Errorf("POST with Prefer: return=representation returned status %d, expected 201 Created. Body: %s",
			w.Code, w.Body.String())
	}

	// Response body should contain the entity
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["Name"] != "Full Response Product" {
		t.Errorf("Response Name = %v, expected 'Full Response Product'", response["Name"])
	}

	// Preference-Applied header should be present
	preferenceApplied := w.Header().Get("Preference-Applied")
	if preferenceApplied != "return=representation" {
		t.Errorf("Preference-Applied header = %q, expected 'return=representation'", preferenceApplied)
	}
}
