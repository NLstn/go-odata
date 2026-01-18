package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// WriteTestEntity is a test entity for write handler tests
type WriteTestEntity struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name" odata:"required"`
	Description string  `json:"Description"`
	Price       float64 `json:"Price"`
	Active      bool    `json:"Active"`
}

func setupWriteTestHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&WriteTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(WriteTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	return handler, db
}

func TestHandleCollection_PostWithReturnMinimal(t *testing.T) {
	handler, _ := setupWriteTestHandler(t)

	body := `{"Name": "Test Product", "Price": 99.99, "Active": true}`
	req := httptest.NewRequest(http.MethodPost, "/WriteTestEntities", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Should have Location header
	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Expected Location header")
	}
}

func TestHandleCollection_PostWithReturnRepresentation(t *testing.T) {
	handler, _ := setupWriteTestHandler(t)

	body := `{"Name": "Test Product", "Price": 99.99, "Active": true}`
	req := httptest.NewRequest(http.MethodPost, "/WriteTestEntities", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["Name"] != "Test Product" {
		t.Errorf("Name = %v, want 'Test Product'", response["Name"])
	}
}

func TestHandlePatch_PartialUpdate(t *testing.T) {
	handler, db := setupWriteTestHandler(t)

	// Create entity
	entity := WriteTestEntity{ID: 1, Name: "Original", Description: "Desc", Price: 10.0, Active: true}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Update only Name
	body := `{"Name": "Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/WriteTestEntities(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify update
	var updated WriteTestEntity
	if err := db.First(&updated, 1).Error; err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("Name = %v, want 'Updated'", updated.Name)
	}
	if updated.Description != "Desc" {
		t.Errorf("Description = %v, want 'Desc' (should be unchanged)", updated.Description)
	}
}

func TestHandlePatch_WithPreferMinimal(t *testing.T) {
	handler, db := setupWriteTestHandler(t)

	// Create entity
	entity := WriteTestEntity{ID: 1, Name: "Original", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	body := `{"Name": "Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/WriteTestEntities(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}
}

func TestHandlePatch_WithPreferRepresentation(t *testing.T) {
	handler, db := setupWriteTestHandler(t)

	// Create entity
	entity := WriteTestEntity{ID: 1, Name: "Original", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	body := `{"Name": "Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/WriteTestEntities(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["Name"] != "Updated" {
		t.Errorf("Name = %v, want 'Updated'", response["Name"])
	}
}

func TestHandlePut_FullReplacement(t *testing.T) {
	handler, db := setupWriteTestHandler(t)

	// Create entity
	entity := WriteTestEntity{ID: 1, Name: "Original", Description: "OldDesc", Price: 10.0, Active: true}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// PUT replaces the entire entity
	body := `{"ID": 1, "Name": "Replaced", "Description": "NewDesc", "Price": 99.99, "Active": false}`
	req := httptest.NewRequest(http.MethodPut, "/WriteTestEntities(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify replacement
	var updated WriteTestEntity
	if err := db.First(&updated, 1).Error; err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	if updated.Name != "Replaced" {
		t.Errorf("Name = %v, want 'Replaced'", updated.Name)
	}
	if updated.Description != "NewDesc" {
		t.Errorf("Description = %v, want 'NewDesc'", updated.Description)
	}
	if updated.Active != false {
		t.Errorf("Active = %v, want false", updated.Active)
	}
}

func TestHandlePut_WithPreferRepresentation(t *testing.T) {
	handler, db := setupWriteTestHandler(t)

	// Create entity
	entity := WriteTestEntity{ID: 1, Name: "Original", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	body := `{"ID": 1, "Name": "Replaced", "Price": 99.99}`
	req := httptest.NewRequest(http.MethodPut, "/WriteTestEntities(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["Name"] != "Replaced" {
		t.Errorf("Name = %v, want 'Replaced'", response["Name"])
	}
}

func TestHandleDelete_Success(t *testing.T) {
	handler, db := setupWriteTestHandler(t)

	// Create entity
	entity := WriteTestEntity{ID: 1, Name: "ToDelete", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/WriteTestEntities(1)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify deletion
	var count int64
	db.Model(&WriteTestEntity{}).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 entities, got %d", count)
	}
}

func TestHandleDelete_NotFound(t *testing.T) {
	handler, _ := setupWriteTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/WriteTestEntities(999)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "999")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleCollection_PostMissingRequired(t *testing.T) {
	handler, _ := setupWriteTestHandler(t)

	// Missing Name which is required
	body := `{"Price": 99.99}`
	req := httptest.NewRequest(http.MethodPost, "/WriteTestEntities", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCollection_PostInvalidJSON(t *testing.T) {
	handler, _ := setupWriteTestHandler(t)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/WriteTestEntities", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCollection_PostNoContentType(t *testing.T) {
	handler, _ := setupWriteTestHandler(t)

	body := `{"Name": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/WriteTestEntities", strings.NewReader(body))
	// No Content-Type header
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnsupportedMediaType)
	}
}

func TestHandleCollection_PostWrongContentType(t *testing.T) {
	handler, _ := setupWriteTestHandler(t)

	body := `{"Name": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/WriteTestEntities", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnsupportedMediaType)
	}
}

func TestHandlePatch_NotFound(t *testing.T) {
	handler, _ := setupWriteTestHandler(t)

	body := `{"Name": "Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/WriteTestEntities(999)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "999")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandlePatch_InvalidJSON(t *testing.T) {
	handler, db := setupWriteTestHandler(t)

	// Create entity
	entity := WriteTestEntity{ID: 1, Name: "Original", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPatch, "/WriteTestEntities(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePut_NotFound(t *testing.T) {
	handler, _ := setupWriteTestHandler(t)

	body := `{"ID": 999, "Name": "Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/WriteTestEntities(999)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "999")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandlePut_InvalidJSON(t *testing.T) {
	handler, db := setupWriteTestHandler(t)

	// Create entity
	entity := WriteTestEntity{ID: 1, Name: "Original", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPut, "/WriteTestEntities(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}
