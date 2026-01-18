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

// PutTestProduct is a test entity for PUT operations
type PutTestProduct struct {
	ID          int     `json:"id" gorm:"primarykey" odata:"key"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
}

// PutTestProductCompositeKey is a test entity with composite keys for PUT operations
type PutTestProductCompositeKey struct {
	ProductID   int    `json:"productID" gorm:"primaryKey" odata:"key"`
	LanguageKey string `json:"languageKey" gorm:"primaryKey" odata:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func setupPutTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PutTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(PutTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func setupPutCompositeKeyTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PutTestProductCompositeKey{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(PutTestProductCompositeKey{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestPutEntity_Success(t *testing.T) {
	service, db := setupPutTestService(t)

	// Insert test data
	product := PutTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		Category:    "Electronics",
	}
	db.Create(&product)

	// Replace with completely new data
	replacementData := PutTestProduct{
		Name:        "Gaming Laptop",
		Price:       1499.99,
		Description: "A powerful gaming laptop",
		Category:    "Gaming",
	}
	body, _ := json.Marshal(replacementData)

	req := httptest.NewRequest(http.MethodPut, "/PutTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the entity was replaced in the database
	var updated PutTestProduct
	db.First(&updated, 1)
	if updated.Name != "Gaming Laptop" {
		t.Errorf("Name = %v, want Gaming Laptop", updated.Name)
	}
	if updated.Price != 1499.99 {
		t.Errorf("Price = %v, want 1499.99", updated.Price)
	}
	if updated.Description != "A powerful gaming laptop" {
		t.Errorf("Description = %v, want 'A powerful gaming laptop'", updated.Description)
	}
	if updated.Category != "Gaming" {
		t.Errorf("Category = %v, want Gaming", updated.Category)
	}
}

func TestPutEntity_WithMissingFields(t *testing.T) {
	service, db := setupPutTestService(t)

	// Insert test data
	product := PutTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		Category:    "Electronics",
	}
	db.Create(&product)

	// Replace with data that has missing optional fields
	// According to OData v4, missing fields should be set to default values
	replacementData := map[string]interface{}{
		"name":  "Basic Laptop",
		"price": 599.99,
		// Description and Category are missing - should be set to empty string (default)
	}
	body, _ := json.Marshal(replacementData)

	req := httptest.NewRequest(http.MethodPut, "/PutTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the entity was replaced and missing fields are default values
	var updated PutTestProduct
	db.First(&updated, 1)
	if updated.Name != "Basic Laptop" {
		t.Errorf("Name = %v, want Basic Laptop", updated.Name)
	}
	if updated.Price != 599.99 {
		t.Errorf("Price = %v, want 599.99", updated.Price)
	}
	if updated.Description != "" {
		t.Errorf("Description = %v, want empty string (default)", updated.Description)
	}
	if updated.Category != "" {
		t.Errorf("Category = %v, want empty string (default)", updated.Category)
	}
}

func TestPutEntity_NonExistent(t *testing.T) {
	service, _ := setupPutTestService(t)

	replacementData := PutTestProduct{
		Name:        "New Laptop",
		Price:       999.99,
		Description: "A new laptop",
		Category:    "Electronics",
	}
	body, _ := json.Marshal(replacementData)

	req := httptest.NewRequest(http.MethodPut, "/PutTestProducts(999)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 404 Not Found
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestPutEntity_InvalidJSON(t *testing.T) {
	service, db := setupPutTestService(t)

	// Insert test data
	product := PutTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		Category:    "Electronics",
	}
	db.Create(&product)

	// Invalid JSON
	body := []byte(`{"name": "Laptop", "price": invalid}`)

	req := httptest.NewRequest(http.MethodPut, "/PutTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestPutEntity_InvalidKey(t *testing.T) {
	service, db := setupPutTestService(t)

	// Insert test data
	product := PutTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
		Category:    "Electronics",
	}
	db.Create(&product)

	replacementData := PutTestProduct{
		Name:        "New Laptop",
		Price:       999.99,
		Description: "A new laptop",
		Category:    "Electronics",
	}
	body, _ := json.Marshal(replacementData)

	req := httptest.NewRequest(http.MethodPut, "/PutTestProducts(invalid)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request or 404 Not Found (depending on how invalid keys are handled)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v or %v. Body: %s", w.Code, http.StatusBadRequest, http.StatusNotFound, w.Body.String())
	}
}

func TestPutEntity_GetAfterPut(t *testing.T) {
	service, db := setupPutTestService(t)

	// Insert test data
	product := PutTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "Original description",
		Category:    "Electronics",
	}
	db.Create(&product)

	// Put the entity with completely new data
	replacementData := PutTestProduct{
		Name:        "Updated Laptop",
		Price:       1299.99,
		Description: "Updated description",
		Category:    "Gaming",
	}
	body, _ := json.Marshal(replacementData)

	req := httptest.NewRequest(http.MethodPut, "/PutTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("PUT failed: Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// GET the replaced entity
	req = httptest.NewRequest(http.MethodGet, "/PutTestProducts(1)", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET after PUT failed: Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Parse the response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify all fields were replaced
	if name, ok := response["name"].(string); !ok || name != "Updated Laptop" {
		t.Errorf("GET response name = %v, want 'Updated Laptop'", response["name"])
	}
	if price, ok := response["price"].(float64); !ok || price != 1299.99 {
		t.Errorf("GET response price = %v, want 1299.99", response["price"])
	}
	if desc, ok := response["description"].(string); !ok || desc != "Updated description" {
		t.Errorf("GET response description = %v, want 'Updated description'", response["description"])
	}
	if category, ok := response["category"].(string); !ok || category != "Gaming" {
		t.Errorf("GET response category = %v, want 'Gaming'", response["category"])
	}
}

func TestPutEntity_MultiplePuts(t *testing.T) {
	service, db := setupPutTestService(t)

	// Insert test data
	product := PutTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "Original",
		Category:    "Electronics",
	}
	db.Create(&product)

	// First PUT
	updateData1 := PutTestProduct{
		Name:        "Gaming Laptop",
		Price:       1299.99,
		Description: "First update",
		Category:    "Gaming",
	}
	body1, _ := json.Marshal(updateData1)
	req := httptest.NewRequest(http.MethodPut, "/PutTestProducts(1)", bytes.NewBuffer(body1))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("First PUT: Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Second PUT
	updateData2 := PutTestProduct{
		Name:        "Professional Laptop",
		Price:       1599.99,
		Description: "Second update",
		Category:    "Business",
	}
	body2, _ := json.Marshal(updateData2)
	req = httptest.NewRequest(http.MethodPut, "/PutTestProducts(1)", bytes.NewBuffer(body2))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Second PUT: Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Third PUT - with missing fields
	updateData3 := map[string]interface{}{
		"name":  "Budget Laptop",
		"price": 599.99,
		// Missing description and category - should be set to defaults
	}
	body3, _ := json.Marshal(updateData3)
	req = httptest.NewRequest(http.MethodPut, "/PutTestProducts(1)", bytes.NewBuffer(body3))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Third PUT: Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Verify final state
	var final PutTestProduct
	db.First(&final, 1)
	if final.Name != "Budget Laptop" {
		t.Errorf("Final name = %v, want 'Budget Laptop'", final.Name)
	}
	if final.Price != 599.99 {
		t.Errorf("Final price = %v, want 599.99", final.Price)
	}
	if final.Description != "" {
		t.Errorf("Final description = %v, want empty string", final.Description)
	}
	if final.Category != "" {
		t.Errorf("Final category = %v, want empty string", final.Category)
	}
}

func TestPutEntity_CompositeKey(t *testing.T) {
	service, db := setupPutCompositeKeyTestService(t)

	// Insert test data with composite key
	product := PutTestProductCompositeKey{
		ProductID:   1,
		LanguageKey: "EN",
		Name:        "Laptop",
		Description: "A laptop",
	}
	db.Create(&product)

	// Replace with new data
	replacementData := PutTestProductCompositeKey{
		Name:        "Gaming Laptop",
		Description: "A gaming laptop",
	}
	body, _ := json.Marshal(replacementData)

	req := httptest.NewRequest(http.MethodPut, "/PutTestProductCompositeKeys(productID=1,languageKey='EN')", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the entity was replaced
	var updated PutTestProductCompositeKey
	db.Where("product_id = ? AND language_key = ?", 1, "EN").First(&updated)
	if updated.Name != "Gaming Laptop" {
		t.Errorf("Name = %v, want 'Gaming Laptop'", updated.Name)
	}
	if updated.Description != "A gaming laptop" {
		t.Errorf("Description = %v, want 'A gaming laptop'", updated.Description)
	}
}

func TestPutEntity_DifferenceFromPatch(t *testing.T) {
	service, db := setupPutTestService(t)

	// Insert test data
	product := PutTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "Original description",
		Category:    "Electronics",
	}
	db.Create(&product)

	// PUT with only name and price - description and category should be cleared
	updateData := map[string]interface{}{
		"name":  "Updated Laptop",
		"price": 1299.99,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPut, "/PutTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("PUT failed: Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Verify that omitted fields were set to defaults (this is the difference from PATCH)
	var updated PutTestProduct
	db.First(&updated, 1)
	if updated.Name != "Updated Laptop" {
		t.Errorf("Name = %v, want 'Updated Laptop'", updated.Name)
	}
	if updated.Price != 1299.99 {
		t.Errorf("Price = %v, want 1299.99", updated.Price)
	}
	// These should be empty/default because they were not in the PUT request
	if updated.Description != "" {
		t.Errorf("Description = %v, want empty string (default for PUT)", updated.Description)
	}
	if updated.Category != "" {
		t.Errorf("Category = %v, want empty string (default for PUT)", updated.Category)
	}
}
