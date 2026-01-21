package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// AdditionalCoverageTestEntity is a test entity for additional coverage tests
type AdditionalCoverageTestEntity struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name" odata:"required"`
	Price       float64 `json:"Price"`
	Category    string  `json:"Category"`
	Description string  `json:"Description"`
}

func setupAdditionalCoverageHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&AdditionalCoverageTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(AdditionalCoverageTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	tracker, err := trackchanges.NewTracker()
	if err != nil {
		t.Fatalf("Failed to create tracker: %v", err)
	}
	tracker.RegisterEntity(entityMeta.EntitySetName)
	handler.SetDeltaTracker(tracker)
	return handler, db
}

// Test DELETE entity
func TestHandleDeleteEntity_Success(t *testing.T) {
	handler, db := setupAdditionalCoverageHandler(t)

	// Create entity
	entity := AdditionalCoverageTestEntity{ID: 1, Name: "Test", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/AdditionalCoverageTestEntities(1)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify deletion
	var deleted AdditionalCoverageTestEntity
	err := db.First(&deleted, 1).Error
	if err == nil {
		t.Error("Expected entity to be deleted")
	}
}

func TestHandleDeleteEntity_NotFound(t *testing.T) {
	handler, _ := setupAdditionalCoverageHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/AdditionalCoverageTestEntities(999)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "999")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

// Test PATCH entity
func TestHandlePatchEntity_Success(t *testing.T) {
	handler, db := setupAdditionalCoverageHandler(t)

	// Create entity
	entity := AdditionalCoverageTestEntity{ID: 1, Name: "Original", Price: 10.0, Category: "Test"}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	body := `{"Name": "Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/AdditionalCoverageTestEntities(1)", requestBody(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify update
	var updated AdditionalCoverageTestEntity
	if err := db.First(&updated, 1).Error; err != nil {
		t.Fatalf("Failed to fetch updated entity: %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("Name = %v, want 'Updated'", updated.Name)
	}

	// Category should remain unchanged
	if updated.Category != "Test" {
		t.Errorf("Category = %v, want 'Test'", updated.Category)
	}
}

// Test PUT entity
func TestHandlePutEntity_Success(t *testing.T) {
	handler, db := setupAdditionalCoverageHandler(t)

	// Create entity
	entity := AdditionalCoverageTestEntity{ID: 1, Name: "Original", Price: 10.0, Category: "Test"}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	body := `{"ID": 1, "Name": "Replaced", "Price": 20.0, "Category": "NewCategory"}`
	req := httptest.NewRequest(http.MethodPut, "/AdditionalCoverageTestEntities(1)", requestBody(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify update
	var updated AdditionalCoverageTestEntity
	if err := db.First(&updated, 1).Error; err != nil {
		t.Fatalf("Failed to fetch updated entity: %v", err)
	}

	if updated.Name != "Replaced" {
		t.Errorf("Name = %v, want 'Replaced'", updated.Name)
	}

	if updated.Category != "NewCategory" {
		t.Errorf("Category = %v, want 'NewCategory'", updated.Category)
	}
}

// Test collection count
func TestHandleCollectionCount_Success(t *testing.T) {
	handler, db := setupAdditionalCoverageHandler(t)

	// Create entities
	entities := []AdditionalCoverageTestEntity{
		{ID: 1, Name: "One", Price: 10.0},
		{ID: 2, Name: "Two", Price: 20.0},
		{ID: 3, Name: "Three", Price: 30.0},
	}
	for _, e := range entities {
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/AdditionalCoverageTestEntities/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleCount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body != "3" {
		t.Errorf("Count = %v, want '3'", body)
	}
}

func TestHandleCollectionCount_Empty(t *testing.T) {
	handler, _ := setupAdditionalCoverageHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/AdditionalCoverageTestEntities/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleCount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body != "0" {
		t.Errorf("Count = %v, want '0'", body)
	}
}

func TestHandleCollectionCount_Head(t *testing.T) {
	handler, db := setupAdditionalCoverageHandler(t)

	// Create entities
	entity := AdditionalCoverageTestEntity{ID: 1, Name: "One", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/AdditionalCoverageTestEntities/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleCount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestHandleCollectionCount_MethodNotAllowed(t *testing.T) {
	handler, _ := setupAdditionalCoverageHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/AdditionalCoverageTestEntities/$count", nil)
			w := httptest.NewRecorder()

			handler.HandleCount(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// Test HEAD requests
func TestHandleCollection_Head(t *testing.T) {
	handler, db := setupAdditionalCoverageHandler(t)

	// Create entity
	entity := AdditionalCoverageTestEntity{ID: 1, Name: "Test", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/AdditionalCoverageTestEntities", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestHandleEntity_Head(t *testing.T) {
	handler, db := setupAdditionalCoverageHandler(t)

	// Create entity
	entity := AdditionalCoverageTestEntity{ID: 1, Name: "Test", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/AdditionalCoverageTestEntities(1)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

// Test POST create entity
func TestHandlePostEntity_Success(t *testing.T) {
	handler, db := setupAdditionalCoverageHandler(t)

	body := `{"Name": "New Entity", "Price": 99.99, "Category": "Electronics"}`
	req := httptest.NewRequest(http.MethodPost, "/AdditionalCoverageTestEntities", requestBody(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["Name"] != "New Entity" {
		t.Errorf("Name = %v, want 'New Entity'", response["Name"])
	}

	// Verify in database
	var entities []AdditionalCoverageTestEntity
	if err := db.Find(&entities).Error; err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if len(entities) != 1 {
		t.Errorf("Expected 1 entity in database, got %d", len(entities))
	}
}

func TestHandlePostEntity_MissingRequired(t *testing.T) {
	handler, _ := setupAdditionalCoverageHandler(t)

	// Missing Name which is required
	body := `{"Price": 99.99}`
	req := httptest.NewRequest(http.MethodPost, "/AdditionalCoverageTestEntities", requestBody(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePostEntity_InvalidContentType(t *testing.T) {
	handler, _ := setupAdditionalCoverageHandler(t)

	body := `{"Name": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/AdditionalCoverageTestEntities", requestBody(body))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnsupportedMediaType)
	}
}

func TestHandlePostEntity_NoContentType(t *testing.T) {
	handler, _ := setupAdditionalCoverageHandler(t)

	body := `{"Name": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/AdditionalCoverageTestEntities", requestBody(body))
	// No Content-Type header
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnsupportedMediaType)
	}
}

// Test inline count ($count=true)
func TestHandleCollection_InlineCount(t *testing.T) {
	handler, db := setupAdditionalCoverageHandler(t)

	// Create entities
	entities := []AdditionalCoverageTestEntity{
		{ID: 1, Name: "One", Price: 10.0},
		{ID: 2, Name: "Two", Price: 20.0},
		{ID: 3, Name: "Three", Price: 30.0},
	}
	for _, e := range entities {
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/AdditionalCoverageTestEntities?$count=true", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check inline count
	count, ok := response["@odata.count"]
	if !ok {
		t.Error("Expected @odata.count in response")
	}

	if count != float64(3) {
		t.Errorf("@odata.count = %v, want 3", count)
	}
}

// Test $top and $skip
func TestHandleCollection_TopAndSkip(t *testing.T) {
	handler, db := setupAdditionalCoverageHandler(t)

	// Create entities
	entities := []AdditionalCoverageTestEntity{
		{ID: 1, Name: "One", Price: 10.0},
		{ID: 2, Name: "Two", Price: 20.0},
		{ID: 3, Name: "Three", Price: 30.0},
		{ID: 4, Name: "Four", Price: 40.0},
		{ID: 5, Name: "Five", Price: 50.0},
	}
	for _, e := range entities {
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/AdditionalCoverageTestEntities?$top=2&$skip=1&$orderby=ID", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 2 {
		t.Errorf("Expected 2 results (top=2), got %d", len(value))
	}

	// First item should be Two (after skip=1)
	if len(value) > 0 {
		first, ok := value[0].(map[string]interface{})
		if !ok {
			t.Fatal("First item is not a map")
		}
		if first["Name"] != "Two" {
			t.Errorf("First item Name = %v, want 'Two'", first["Name"])
		}
	}
}

// requestBody is a helper to create a request body
func requestBody(body string) *strings.Reader {
	return strings.NewReader(body)
}
