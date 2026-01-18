package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// PatchTestProduct is a test entity for PATCH operations
type PatchTestProduct struct {
	ID          int     `json:"id" gorm:"primarykey" odata:"key"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
}

// PatchTestProductCompositeKey is a test entity with composite keys for PATCH operations
type PatchTestProductCompositeKey struct {
	ProductID   int    `json:"productID" gorm:"primaryKey" odata:"key"`
	LanguageKey string `json:"languageKey" gorm:"primaryKey" odata:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func setupPatchTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PatchTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(PatchTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func setupPatchCompositeKeyTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PatchTestProductCompositeKey{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(PatchTestProductCompositeKey{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestPatchEntity_Success(t *testing.T) {
	service, db := setupPatchTestService(t)

	// Insert test data
	product := PatchTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		Category:    "Electronics",
	}
	db.Create(&product)

	// Update only the price
	updateData := map[string]interface{}{
		"price": 899.99,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify entity was updated
	var updated PatchTestProduct
	db.First(&updated, 1)
	if updated.Price != 899.99 {
		t.Errorf("Price = %v, want 899.99", updated.Price)
	}
	// Verify other fields were not changed
	if updated.Name != "Laptop" {
		t.Errorf("Name = %v, want Laptop", updated.Name)
	}
	if updated.Description != "A high-performance laptop" {
		t.Errorf("Description = %v, want 'A high-performance laptop'", updated.Description)
	}
	if updated.Category != "Electronics" {
		t.Errorf("Category = %v, want Electronics", updated.Category)
	}
}

func TestPatchEntity_MultipleFields(t *testing.T) {
	service, db := setupPatchTestService(t)

	// Insert test data
	product := PatchTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		Category:    "Electronics",
	}
	db.Create(&product)

	// Update multiple fields
	updateData := map[string]interface{}{
		"name":        "Gaming Laptop",
		"price":       1299.99,
		"description": "A high-performance gaming laptop",
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify entity was updated
	var updated PatchTestProduct
	db.First(&updated, 1)
	if updated.Name != "Gaming Laptop" {
		t.Errorf("Name = %v, want 'Gaming Laptop'", updated.Name)
	}
	if updated.Price != 1299.99 {
		t.Errorf("Price = %v, want 1299.99", updated.Price)
	}
	if updated.Description != "A high-performance gaming laptop" {
		t.Errorf("Description = %v, want 'A high-performance gaming laptop'", updated.Description)
	}
	// Verify category was not changed
	if updated.Category != "Electronics" {
		t.Errorf("Category = %v, want Electronics", updated.Category)
	}
}

func TestPatchEntity_NotFound(t *testing.T) {
	service, _ := setupPatchTestService(t)

	// Try to patch non-existent entity
	updateData := map[string]interface{}{
		"price": 899.99,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(999)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 404 Not Found
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}

	// Verify error response format
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

func TestPatchEntity_InvalidJSON(t *testing.T) {
	service, db := setupPatchTestService(t)

	// Insert test data
	product := PatchTestProduct{
		ID:    1,
		Name:  "Laptop",
		Price: 999.99,
	}
	db.Create(&product)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(1)", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	// Verify error response format
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

func TestPatchEntity_CannotUpdateKeyProperty(t *testing.T) {
	service, db := setupPatchTestService(t)

	// Insert test data
	product := PatchTestProduct{
		ID:    1,
		Name:  "Laptop",
		Price: 999.99,
	}
	db.Create(&product)

	// Try to update the key property
	updateData := map[string]interface{}{
		"id":   2,
		"name": "Updated Laptop",
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	// Verify error response format
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}

	// Verify entity ID was not changed
	var unchanged PatchTestProduct
	db.First(&unchanged, 1)
	if unchanged.ID != 1 {
		t.Errorf("ID = %v, want 1", unchanged.ID)
	}
}

func TestPatchEntity_MethodNotAllowedOnCollection(t *testing.T) {
	service, _ := setupPatchTestService(t)

	// Try to patch on collection endpoint (should not be allowed)
	updateData := map[string]interface{}{
		"price": 899.99,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 405 Method Not Allowed
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestPatchEntity_CompositeKey_Success(t *testing.T) {
	service, db := setupPatchCompositeKeyTestService(t)

	// Insert test data
	product := PatchTestProductCompositeKey{
		ProductID:   1,
		LanguageKey: "EN",
		Name:        "Laptop",
		Description: "A high-performance laptop",
	}
	db.Create(&product)

	// Update the description
	updateData := map[string]interface{}{
		"description": "An updated description",
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProductCompositeKeys(productID=1,languageKey='EN')", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify entity was updated
	var updated PatchTestProductCompositeKey
	db.Where("product_id = ? AND language_key = ?", 1, "EN").First(&updated)
	if updated.Description != "An updated description" {
		t.Errorf("Description = %v, want 'An updated description'", updated.Description)
	}
	// Verify name was not changed
	if updated.Name != "Laptop" {
		t.Errorf("Name = %v, want Laptop", updated.Name)
	}
}

func TestPatchEntity_CompositeKey_CannotUpdateKeyProperty(t *testing.T) {
	service, db := setupPatchCompositeKeyTestService(t)

	// Insert test data
	product := PatchTestProductCompositeKey{
		ProductID:   1,
		LanguageKey: "EN",
		Name:        "Laptop",
		Description: "A high-performance laptop",
	}
	db.Create(&product)

	// Try to update a key property
	updateData := map[string]interface{}{
		"languageKey": "DE",
		"description": "Updated description",
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProductCompositeKeys(productID=1,languageKey='EN')", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	// Verify entity was not updated
	var unchanged PatchTestProductCompositeKey
	db.Where("product_id = ? AND language_key = ?", 1, "EN").First(&unchanged)
	if unchanged.LanguageKey != "EN" {
		t.Errorf("LanguageKey = %v, want EN", unchanged.LanguageKey)
	}
}

func TestPatchEntity_GetAfterPatch(t *testing.T) {
	service, db := setupPatchTestService(t)

	// Insert test data
	product := PatchTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "Original description",
		Category:    "Electronics",
	}
	db.Create(&product)

	// Patch the entity
	updateData := map[string]interface{}{
		"price":       799.99,
		"description": "Updated description",
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("PATCH failed: Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// GET the patched entity
	req = httptest.NewRequest(http.MethodGet, "/PatchTestProducts(1)", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("GET after PATCH: Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify the response contains updated data
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["price"] != 799.99 {
		t.Errorf("price = %v, want 799.99", response["price"])
	}
	if response["description"] != "Updated description" {
		t.Errorf("description = %v, want 'Updated description'", response["description"])
	}
	if response["name"] != "Laptop" {
		t.Errorf("name = %v, want Laptop", response["name"])
	}
	if response["category"] != "Electronics" {
		t.Errorf("category = %v, want Electronics", response["category"])
	}
}

func TestPatchEntity_MultiplePatches(t *testing.T) {
	service, db := setupPatchTestService(t)

	// Insert test data
	product := PatchTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "Original description",
		Category:    "Electronics",
	}
	db.Create(&product)

	// First PATCH: update price
	updateData1 := map[string]interface{}{
		"price": 899.99,
	}
	body1, _ := json.Marshal(updateData1)
	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(1)", bytes.NewBuffer(body1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("First PATCH: Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Second PATCH: update name
	updateData2 := map[string]interface{}{
		"name": "Gaming Laptop",
	}
	body2, _ := json.Marshal(updateData2)
	req = httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(1)", bytes.NewBuffer(body2))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Second PATCH: Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Third PATCH: update category
	updateData3 := map[string]interface{}{
		"category": "Gaming",
	}
	body3, _ := json.Marshal(updateData3)
	req = httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(1)", bytes.NewBuffer(body3))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Third PATCH: Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Verify all updates were applied
	var updated PatchTestProduct
	db.First(&updated, 1)
	if updated.Price != 899.99 {
		t.Errorf("Price = %v, want 899.99", updated.Price)
	}
	if updated.Name != "Gaming Laptop" {
		t.Errorf("Name = %v, want 'Gaming Laptop'", updated.Name)
	}
	if updated.Category != "Gaming" {
		t.Errorf("Category = %v, want Gaming", updated.Category)
	}
	if updated.Description != "Original description" {
		t.Errorf("Description = %v, want 'Original description'", updated.Description)
	}
}

func TestPatchEntity_EmptyBody(t *testing.T) {
	service, db := setupPatchTestService(t)

	// Insert test data
	product := PatchTestProduct{
		ID:    1,
		Name:  "Laptop",
		Price: 999.99,
	}
	db.Create(&product)

	// Send empty update data
	updateData := map[string]interface{}{}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should still return 204 No Content (no changes made)
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify entity was not changed
	var unchanged PatchTestProduct
	db.First(&unchanged, 1)
	if unchanged.Name != "Laptop" {
		t.Errorf("Name = %v, want Laptop", unchanged.Name)
	}
	if unchanged.Price != 999.99 {
		t.Errorf("Price = %v, want 999.99", unchanged.Price)
	}
}

func TestPatchEntity_NullValue(t *testing.T) {
	service, db := setupPatchTestService(t)

	// Insert test data
	product := PatchTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A description",
	}
	db.Create(&product)

	// Set description to empty string (null-like behavior)
	updateData := map[string]interface{}{
		"description": "",
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PatchTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify description was cleared
	var updated PatchTestProduct
	db.First(&updated, 1)
	if updated.Description != "" {
		t.Errorf("Description = %v, want empty string", updated.Description)
	}
	// Verify other fields were not changed
	if updated.Name != "Laptop" {
		t.Errorf("Name = %v, want Laptop", updated.Name)
	}
	if updated.Price != 999.99 {
		t.Errorf("Price = %v, want 999.99", updated.Price)
	}
}
