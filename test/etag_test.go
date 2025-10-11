package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ETagProduct is a test entity with an ETag field
type ETagProduct struct {
	ID          int       `json:"id" gorm:"primarykey;autoIncrement" odata:"key"`
	Name        string    `json:"name" odata:"required"`
	Price       float64   `json:"price"`
	Version     int       `json:"version" odata:"etag"` // Version field used for ETag
	LastUpdated time.Time `json:"last_updated"`
}

// ETagProductWithTimestamp is a test entity using timestamp for ETag
type ETagProductWithTimestamp struct {
	ID          int       `json:"id" gorm:"primarykey;autoIncrement" odata:"key"`
	Name        string    `json:"name" odata:"required"`
	Price       float64   `json:"price"`
	LastUpdated time.Time `json:"last_updated" odata:"etag"` // Timestamp used for ETag
}

func setupETagTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ETagProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(ETagProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func setupETagTimestampTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ETagProductWithTimestamp{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(ETagProductWithTimestamp{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// Test that GET returns ETag header
func TestGetEntity_WithETag(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Check that ETag header is present
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header is missing")
	}

	// Check that ETag has the correct format (W/"hash")
	if len(etag) < 3 || etag[:3] != "W/\"" {
		t.Errorf("ETag = %v, want format W/\"...\"", etag)
	}
}

// Test that POST returns ETag header
func TestPostEntity_WithETag(t *testing.T) {
	service, _ := setupETagTestService(t)

	newProduct := map[string]interface{}{
		"name":    "Mouse",
		"price":   29.99,
		"version": 1,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/ETagProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	// Check that ETag header is present
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header is missing")
	}
}

// Test PATCH with matching If-Match header
func TestPatchEntity_WithMatchingETag(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	// First, get the ETag
	req := httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	etag := w.Header().Get("ETag")

	// Now try to update with the correct ETag
	update := map[string]interface{}{
		"price": 899.99,
	}
	body, _ := json.Marshal(update)

	req = httptest.NewRequest(http.MethodPatch, "/ETagProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", etag)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}
}

// Test PATCH with non-matching If-Match header
func TestPatchEntity_WithNonMatchingETag(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	// Try to update with an incorrect ETag
	update := map[string]interface{}{
		"price": 899.99,
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest(http.MethodPatch, "/ETagProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "W/\"wrongetag\"")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusPreconditionFailed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusPreconditionFailed)
	}

	// Verify error response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

// Test PATCH without If-Match header (should succeed)
func TestPatchEntity_WithoutIfMatch(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	// Try to update without If-Match header (should succeed)
	update := map[string]interface{}{
		"price": 899.99,
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest(http.MethodPatch, "/ETagProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}
}

// Test PATCH with wildcard If-Match header
func TestPatchEntity_WithWildcardIfMatch(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	// Try to update with wildcard If-Match (should succeed)
	update := map[string]interface{}{
		"price": 899.99,
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest(http.MethodPatch, "/ETagProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "*")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}
}

// Test PUT with matching If-Match header
func TestPutEntity_WithMatchingETag(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	// First, get the ETag
	req := httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	etag := w.Header().Get("ETag")

	// Now try to replace with the correct ETag
	replacement := map[string]interface{}{
		"name":    "Gaming Laptop",
		"price":   1299.99,
		"version": 1,
	}
	body, _ := json.Marshal(replacement)

	req = httptest.NewRequest(http.MethodPut, "/ETagProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", etag)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}
}

// Test PUT with non-matching If-Match header
func TestPutEntity_WithNonMatchingETag(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	// Try to replace with an incorrect ETag
	replacement := map[string]interface{}{
		"name":    "Gaming Laptop",
		"price":   1299.99,
		"version": 1,
	}
	body, _ := json.Marshal(replacement)

	req := httptest.NewRequest(http.MethodPut, "/ETagProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "W/\"wrongetag\"")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusPreconditionFailed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusPreconditionFailed)
	}
}

// Test DELETE with matching If-Match header
func TestDeleteEntity_WithMatchingETag(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	// First, get the ETag
	req := httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	etag := w.Header().Get("ETag")

	// Now try to delete with the correct ETag
	req = httptest.NewRequest(http.MethodDelete, "/ETagProducts(1)", nil)
	req.Header.Set("If-Match", etag)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}
}

// Test DELETE with non-matching If-Match header
func TestDeleteEntity_WithNonMatchingETag(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	// Try to delete with an incorrect ETag
	req := httptest.NewRequest(http.MethodDelete, "/ETagProducts(1)", nil)
	req.Header.Set("If-Match", "W/\"wrongetag\"")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusPreconditionFailed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusPreconditionFailed)
	}
}

// Test that different ETag field values produce different ETags
func TestETag_DifferentValuesProduceDifferentETags(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create two products with different versions
	product1 := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	product2 := ETagProduct{
		ID:          2,
		Name:        "Laptop",
		Price:       999.99,
		Version:     2,
		LastUpdated: time.Now(),
	}
	db.Create(&product1)
	db.Create(&product2)

	// Get ETags for both products
	req := httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	etag1 := w.Header().Get("ETag")

	req = httptest.NewRequest(http.MethodGet, "/ETagProducts(2)", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)
	etag2 := w.Header().Get("ETag")

	if etag1 == etag2 {
		t.Errorf("ETags should be different for different version values, got %v", etag1)
	}
}

// Test that same ETag field values produce same ETags
func TestETag_SameValuesProduceSameETags(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create two products with the same version
	product1 := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	product2 := ETagProduct{
		ID:          2,
		Name:        "Desktop",
		Price:       1299.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product1)
	db.Create(&product2)

	// Get ETags for both products
	req := httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	etag1 := w.Header().Get("ETag")

	req = httptest.NewRequest(http.MethodGet, "/ETagProducts(2)", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)
	etag2 := w.Header().Get("ETag")

	if etag1 != etag2 {
		t.Errorf("ETags should be the same for same version values, got %v and %v", etag1, etag2)
	}
}

// Test ETag with timestamp field
func TestETag_WithTimestampField(t *testing.T) {
	service, db := setupETagTimestampTestService(t)

	// Create a test product
	product := ETagProductWithTimestamp{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		LastUpdated: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/ETagProductWithTimestamps(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Check that ETag header is present
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header is missing")
	}
}

// Test PATCH with Prefer: return=representation includes ETag
func TestPatchEntity_WithPreferReturnRepresentation_IncludesETag(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	// Update with Prefer: return=representation
	update := map[string]interface{}{
		"price": 899.99,
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest(http.MethodPatch, "/ETagProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Check that ETag header is present in response
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header is missing in response")
	}
}

// Test PUT with Prefer: return=representation includes ETag
func TestPutEntity_WithPreferReturnRepresentation_IncludesETag(t *testing.T) {
	service, db := setupETagTestService(t)

	// Create a test product
	product := ETagProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Version:     1,
		LastUpdated: time.Now(),
	}
	db.Create(&product)

	// Replace with Prefer: return=representation
	replacement := map[string]interface{}{
		"name":    "Gaming Laptop",
		"price":   1299.99,
		"version": 1,
	}
	body, _ := json.Marshal(replacement)

	req := httptest.NewRequest(http.MethodPut, "/ETagProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Check that ETag header is present in response
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header is missing in response")
	}
}
