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

// DeleteTestProduct is a test entity for DELETE operations
type DeleteTestProduct struct {
	ID          int     `json:"id" gorm:"primarykey" odata:"key"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

// DeleteTestProductCompositeKey is a test entity with composite keys for DELETE operations
type DeleteTestProductCompositeKey struct {
	ProductID   int    `json:"productID" gorm:"primaryKey" odata:"key"`
	LanguageKey string `json:"languageKey" gorm:"primaryKey" odata:"key"`
	Name        string `json:"name"`
}

func setupDeleteTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&DeleteTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(DeleteTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func setupDeleteCompositeKeyTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&DeleteTestProductCompositeKey{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(DeleteTestProductCompositeKey{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestDeleteEntity_Success(t *testing.T) {
	service, db := setupDeleteTestService(t)

	// Insert test data
	product := DeleteTestProduct{
		ID:          1,
		Name:        "Laptop",
		Price:       999.99,
		Description: "A high-performance laptop",
	}
	db.Create(&product)

	// Verify entity exists
	var count int64
	db.Model(&DeleteTestProduct{}).Where("id = ?", 1).Count(&count)
	if count != 1 {
		t.Fatalf("Entity not created, count = %d", count)
	}

	// Delete the entity
	req := httptest.NewRequest(http.MethodDelete, "/DeleteTestProducts(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify entity was deleted
	db.Model(&DeleteTestProduct{}).Where("id = ?", 1).Count(&count)
	if count != 0 {
		t.Errorf("Entity not deleted, count = %d", count)
	}
}

func TestDeleteEntity_NotFound(t *testing.T) {
	service, _ := setupDeleteTestService(t)

	// Try to delete non-existent entity
	req := httptest.NewRequest(http.MethodDelete, "/DeleteTestProducts(999)", nil)
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

func TestDeleteEntity_CompositeKey_Success(t *testing.T) {
	service, db := setupDeleteCompositeKeyTestService(t)

	// Insert test data
	product := DeleteTestProductCompositeKey{
		ProductID:   1,
		LanguageKey: "EN",
		Name:        "Laptop",
	}
	db.Create(&product)

	// Verify entity exists
	var count int64
	db.Model(&DeleteTestProductCompositeKey{}).Where("product_id = ? AND language_key = ?", 1, "EN").Count(&count)
	if count != 1 {
		t.Fatalf("Entity not created, count = %d", count)
	}

	// Delete the entity using composite key
	req := httptest.NewRequest(http.MethodDelete, "/DeleteTestProductCompositeKeys(productID=1,languageKey='EN')", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify entity was deleted
	db.Model(&DeleteTestProductCompositeKey{}).Where("product_id = ? AND language_key = ?", 1, "EN").Count(&count)
	if count != 0 {
		t.Errorf("Entity not deleted, count = %d", count)
	}
}

func TestDeleteEntity_CompositeKey_NotFound(t *testing.T) {
	service, _ := setupDeleteCompositeKeyTestService(t)

	// Try to delete non-existent entity with composite key
	req := httptest.NewRequest(http.MethodDelete, "/DeleteTestProductCompositeKeys(productID=999,languageKey='XX')", nil)
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

func TestDeleteEntity_InvalidKey(t *testing.T) {
	service, _ := setupDeleteTestService(t)

	// Try to delete with invalid key format
	req := httptest.NewRequest(http.MethodDelete, "/DeleteTestProducts(invalid)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request or 404 Not Found (depending on how invalid keys are handled)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v or %v. Body: %s", w.Code, http.StatusBadRequest, http.StatusNotFound, w.Body.String())
	}
}

func TestDeleteEntity_MethodNotAllowedOnCollection(t *testing.T) {
	service, _ := setupDeleteTestService(t)

	// Try to delete on collection endpoint (should not be allowed)
	req := httptest.NewRequest(http.MethodDelete, "/DeleteTestProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 405 Method Not Allowed
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestDeleteEntity_MultipleDeletes(t *testing.T) {
	service, db := setupDeleteTestService(t)

	// Insert multiple test products
	products := []DeleteTestProduct{
		{ID: 1, Name: "Product 1", Price: 10.00},
		{ID: 2, Name: "Product 2", Price: 20.00},
		{ID: 3, Name: "Product 3", Price: 30.00},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Delete each product
	for _, p := range products {
		req := httptest.NewRequest(http.MethodDelete, "/DeleteTestProducts("+string(rune(p.ID+'0'))+")", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Delete product %d: Status = %v, want %v", p.ID, w.Code, http.StatusNoContent)
		}
	}

	// Verify all entities were deleted
	var count int64
	db.Model(&DeleteTestProduct{}).Count(&count)
	if count != 0 {
		t.Errorf("Not all entities deleted, remaining count = %d", count)
	}
}

func TestDeleteEntity_GetAfterDelete(t *testing.T) {
	service, db := setupDeleteTestService(t)

	// Insert test data
	product := DeleteTestProduct{
		ID:    1,
		Name:  "Laptop",
		Price: 999.99,
	}
	db.Create(&product)

	// Delete the entity
	req := httptest.NewRequest(http.MethodDelete, "/DeleteTestProducts(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Delete failed: Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Try to GET the deleted entity
	req = httptest.NewRequest(http.MethodGet, "/DeleteTestProducts(1)", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	// Should return 404 Not Found
	if w.Code != http.StatusNotFound {
		t.Errorf("GET after DELETE: Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}
