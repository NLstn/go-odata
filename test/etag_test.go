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

// Test GET with If-None-Match header when ETag matches (304)
func TestGetEntity_WithIfNoneMatch_Matching(t *testing.T) {
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

	// Now try to GET with If-None-Match using the same ETag
	req = httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	req.Header.Set("If-None-Match", etag)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 304 Not Modified
	if w.Code != http.StatusNotModified {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotModified)
	}

	// Should still have ETag header
	if w.Header().Get("ETag") == "" {
		t.Error("ETag header should be present in 304 response")
	}

	// Body should be empty for 304
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty for 304 response, got %d bytes", w.Body.Len())
	}
}

// Test GET with If-None-Match header when ETag doesn't match (200)
func TestGetEntity_WithIfNoneMatch_NotMatching(t *testing.T) {
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

	// Try to GET with If-None-Match using a different ETag
	req := httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	req.Header.Set("If-None-Match", "W/\"differentetag\"")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 200 OK with full entity
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Should have ETag header
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header is missing")
	}

	// Body should contain the entity
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["name"] != "Laptop" {
		t.Errorf("Expected name=Laptop, got %v", response["name"])
	}
}

// Test GET without If-None-Match header (normal behavior)
func TestGetEntity_WithoutIfNoneMatch(t *testing.T) {
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

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Should have ETag header
	if w.Header().Get("ETag") == "" {
		t.Error("ETag header is missing")
	}

	// Body should contain the entity
	if w.Body.Len() == 0 {
		t.Error("Body should not be empty")
	}
}

// Test GET with If-None-Match wildcard (*)
func TestGetEntity_WithIfNoneMatch_Wildcard(t *testing.T) {
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

	// Try to GET with If-None-Match using wildcard
	req := httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	req.Header.Set("If-None-Match", "*")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 304 Not Modified (entity exists)
	if w.Code != http.StatusNotModified {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotModified)
	}
}

// Test GET with If-None-Match after entity modification
func TestGetEntity_WithIfNoneMatch_AfterModification(t *testing.T) {
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
	originalETag := w.Header().Get("ETag")

	// Modify the product (change version which is the ETag field)
	db.Model(&ETagProduct{}).Where("id = ?", 1).Update("version", 2)

	// Now try to GET with If-None-Match using the old ETag
	req = httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	req.Header.Set("If-None-Match", originalETag)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 200 OK since ETag changed
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// New ETag should be different
	newETag := w.Header().Get("ETag")
	if newETag == originalETag {
		t.Error("ETag should have changed after modification")
	}

	// Body should contain the modified entity
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify the version was updated
	if response["version"] != float64(2) {
		t.Errorf("Expected version=2, got %v", response["version"])
	}
}

// Test HEAD with If-None-Match (should also return 304)
func TestHeadEntity_WithIfNoneMatch_Matching(t *testing.T) {
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

	// First, get the ETag using GET
	req := httptest.NewRequest(http.MethodGet, "/ETagProducts(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	etag := w.Header().Get("ETag")

	// Now try HEAD with If-None-Match using the same ETag
	req = httptest.NewRequest(http.MethodHead, "/ETagProducts(1)", nil)
	req.Header.Set("If-None-Match", etag)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 304 Not Modified
	if w.Code != http.StatusNotModified {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotModified)
	}

	// Should still have ETag header
	if w.Header().Get("ETag") == "" {
		t.Error("ETag header should be present in 304 response")
	}

	// Body should be empty for HEAD request
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty for HEAD request, got %d bytes", w.Body.Len())
	}
}
