package odata_test

import (
	"encoding/base64"
	"encoding/json"
	odata "github.com/nlstn/go-odata"
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
	Image       []byte  `json:"image" gorm:"type:blob" odata:"nullable"`                      // Binary data field
	Icon        []byte  `json:"icon" gorm:"type:blob" odata:"nullable,contenttype=image/png"` // Binary data with custom MIME type
}

func setupStructuralPropTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestProductForStructuralProp{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
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

// Additional test types with navigation properties
type TestProductWithNav struct {
	ID         int              `json:"id" gorm:"primarykey" odata:"key"`
	Name       string           `json:"name"`
	CategoryID int              `json:"categoryId"`
	Category   *TestCategoryNav `json:"category" gorm:"foreignKey:CategoryID"`
}

type TestCategoryNav struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
}

func setupStructuralPropWithNavTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestProductWithNav{}, &TestCategoryNav{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(TestProductWithNav{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}
	if err := service.RegisterEntity(TestCategoryNav{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestStructuralPropertyRead_WithNavigationProperty(t *testing.T) {
	service, db := setupStructuralPropWithNavTestService(t)

	// Insert test data
	category := TestCategoryNav{ID: 1, Name: "Electronics"}
	db.Create(&category)
	product := TestProductWithNav{ID: 1, Name: "Laptop", CategoryID: 1}
	db.Create(&product)

	// Test structural property (should return value wrapper)
	req1 := httptest.NewRequest(http.MethodGet, "/TestProductWithNavs(1)/name", nil)
	w1 := httptest.NewRecorder()
	service.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v for structural property. Body: %s", w1.Code, http.StatusOK, w1.Body.String())
	}

	var response1 map[string]interface{}
	if err := json.NewDecoder(w1.Body).Decode(&response1); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check structural property returns value wrapper
	if response1["value"] != "Laptop" {
		t.Errorf("value = %v, want 'Laptop'", response1["value"])
	}

	// Test navigation property (should return full entity, not value wrapper)
	req2 := httptest.NewRequest(http.MethodGet, "/TestProductWithNavs(1)/Category", nil)
	w2 := httptest.NewRecorder()
	service.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v for navigation property. Body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	var response2 map[string]interface{}
	if err := json.NewDecoder(w2.Body).Decode(&response2); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check navigation property returns full entity (not value wrapper)
	if _, hasValue := response2["value"]; hasValue {
		// Navigation properties should not have a "value" wrapper, just the entity properties
		if _, hasId := response2["id"]; !hasId {
			t.Error("Navigation property response has 'value' wrapper but should return full entity")
		}
	}

	// Navigation property should have the entity's properties
	if response2["name"] != "Electronics" {
		t.Errorf("Navigation property name = %v, want 'Electronics'", response2["name"])
	}
}

func TestStructuralPropertyValue_String(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/name/$value", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %v, want 'text/plain; charset=utf-8'", contentType)
	}

	// Check the raw value (should not be JSON)
	body := w.Body.String()
	if body != "Laptop" {
		t.Errorf("Body = %v, want 'Laptop'", body)
	}
}

func TestStructuralPropertyValue_Number(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/price/$value", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %v, want 'text/plain; charset=utf-8'", contentType)
	}

	// Check the raw value (should not be JSON)
	body := w.Body.String()
	if body != "999.99" {
		t.Errorf("Body = %v, want '999.99'", body)
	}
}

func TestStructuralPropertyValue_Boolean(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/inStock/$value", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %v, want 'text/plain; charset=utf-8'", contentType)
	}

	// Check the raw value (should not be JSON)
	body := w.Body.String()
	if body != "true" {
		t.Errorf("Body = %v, want 'true'", body)
	}
}

func TestStructuralPropertyValue_EntityNotFound(t *testing.T) {
	service, _ := setupStructuralPropTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(999)/name/$value", nil)
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

func TestStructuralPropertyValue_PropertyNotFound(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/nonexistent/$value", nil)
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

func TestStructuralPropertyValue_OnNavigationProperty(t *testing.T) {
	service, db := setupStructuralPropWithNavTestService(t)

	// Insert test data
	category := TestCategoryNav{ID: 1, Name: "Electronics"}
	db.Create(&category)
	product := TestProductWithNav{ID: 1, Name: "Laptop", CategoryID: 1}
	db.Create(&product)

	// Try to use $value on a navigation property (should fail)
	req := httptest.NewRequest(http.MethodGet, "/TestProductWithNavs(1)/Category/$value", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v for $value on navigation property. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

// Binary (Edm.Binary) type tests

func TestStructuralPropertyRead_Binary(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data with binary content
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
		Image:       binaryData,
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/image", nil)
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

	// Check the value is base64 encoded (Go's json.Marshal automatically encodes []byte as base64)
	if valueStr, ok := response["value"].(string); ok {
		// Decode base64 and verify it matches original data
		decoded, err := base64.StdEncoding.DecodeString(valueStr)
		if err != nil {
			t.Errorf("Failed to decode base64 value: %v", err)
		}
		if string(decoded) != string(binaryData) {
			t.Errorf("Decoded binary data = %v, want %v", decoded, binaryData)
		}
	} else {
		t.Errorf("value is not a string, got type %T", response["value"])
	}
}

func TestStructuralPropertyValue_Binary(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data with binary content
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
		Image:       binaryData,
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/image/$value", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check content type for binary data
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/octet-stream" {
		t.Errorf("Content-Type = %v, want 'application/octet-stream'", contentType)
	}

	// Check the raw binary value
	body := w.Body.Bytes()
	if string(body) != string(binaryData) {
		t.Errorf("Body = %v, want %v", body, binaryData)
	}
}

func TestStructuralPropertyRead_EmptyBinary(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data with empty binary content
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
		Image:       []byte{}, // Empty binary data
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/image", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Empty binary should be encoded as empty string in JSON
	if valueStr, ok := response["value"].(string); ok {
		if valueStr != "" {
			t.Errorf("Empty binary value = %v, want empty string", valueStr)
		}
	} else {
		t.Errorf("value is not a string, got type %T", response["value"])
	}
}

func TestStructuralPropertyValue_EmptyBinary(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data with empty binary content
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
		Image:       []byte{}, // Empty binary data
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/image/$value", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check content type for binary data
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/octet-stream" {
		t.Errorf("Content-Type = %v, want 'application/octet-stream'", contentType)
	}

	// Check the raw binary value is empty
	body := w.Body.Bytes()
	if len(body) != 0 {
		t.Errorf("Body length = %v, want 0", len(body))
	}
}

func TestStructuralPropertyRead_NullBinary(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data with null binary content
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
		Image:       nil, // Null binary data
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/image", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Null binary should be encoded as null in JSON
	if response["value"] != nil {
		t.Errorf("Null binary value = %v, want nil", response["value"])
	}
}

func TestStructuralPropertyValue_BinaryHEAD(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data with binary content
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
		Image:       binaryData,
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodHead, "/TestProductForStructuralProps(1)/image/$value", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check content type for binary data
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/octet-stream" {
		t.Errorf("Content-Type = %v, want 'application/octet-stream'", contentType)
	}

	// HEAD request should have no body
	body := w.Body.Bytes()
	if len(body) != 0 {
		t.Errorf("HEAD request should have no body, got %d bytes", len(body))
	}
}

func TestStructuralPropertyValue_BinaryCustomContentType(t *testing.T) {
	service, db := setupStructuralPropTestService(t)

	// Insert test data with binary content for Icon field (which has custom content type)
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	product := TestProductForStructuralProp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		InStock:     true,
		Icon:        pngData,
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductForStructuralProps(1)/icon/$value", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check content type - should use custom MIME type from odata tag
	contentType := w.Header().Get("Content-Type")
	if contentType != "image/png" {
		t.Errorf("Content-Type = %v, want 'image/png'", contentType)
	}

	// Check the raw binary value
	body := w.Body.Bytes()
	if string(body) != string(pngData) {
		t.Errorf("Body = %v, want %v", body, pngData)
	}
}
