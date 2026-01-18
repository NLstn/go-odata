package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// CollectionRefTestEntity is a test entity for collection ref tests
type CollectionRefTestEntity struct {
	ID       uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string `json:"Name"`
	Category string `json:"Category"`
}

func setupCollectionRefTestHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&CollectionRefTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(CollectionRefTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	return handler, db
}

func TestHandleCollectionRef_Get(t *testing.T) {
	handler, db := setupCollectionRefTestHandler(t)

	// Create entities
	entities := []CollectionRefTestEntity{
		{ID: 1, Name: "Item 1", Category: "A"},
		{ID: 2, Name: "Item 2", Category: "B"},
		{ID: 3, Name: "Item 3", Category: "A"},
	}
	for _, e := range entities {
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("Failed to create entity: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/CollectionRefTestEntities/$ref", nil)
	w := httptest.NewRecorder()

	handler.HandleCollectionRef(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have a value array
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(value) != 3 {
		t.Errorf("Expected 3 refs, got %d", len(value))
	}

	// Each item should have @odata.id
	for _, item := range value {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok := itemMap["@odata.id"]; !ok {
			t.Error("Expected @odata.id in ref response")
		}
	}
}

func TestHandleCollectionRef_Head(t *testing.T) {
	handler, db := setupCollectionRefTestHandler(t)

	// Create entity
	entity := CollectionRefTestEntity{ID: 1, Name: "Item 1"}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/CollectionRefTestEntities/$ref", nil)
	w := httptest.NewRecorder()

	handler.HandleCollectionRef(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestHandleCollectionRef_Options(t *testing.T) {
	handler, _ := setupCollectionRefTestHandler(t)

	req := httptest.NewRequest(http.MethodOptions, "/CollectionRefTestEntities/$ref", nil)
	w := httptest.NewRecorder()

	handler.HandleCollectionRef(w, req)

	// Check if OPTIONS is supported or returns Method Not Allowed
	// Both are acceptable depending on implementation
	if w.Code != http.StatusOK && w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want 200 or 405", w.Code)
	}
}

func TestHandleCollectionRef_Empty(t *testing.T) {
	handler, _ := setupCollectionRefTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/CollectionRefTestEntities/$ref", nil)
	w := httptest.NewRecorder()

	handler.HandleCollectionRef(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have empty value array
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(value) != 0 {
		t.Errorf("Expected 0 refs, got %d", len(value))
	}
}

func TestHandleEntityRef_Get(t *testing.T) {
	handler, db := setupCollectionRefTestHandler(t)

	// Create entity
	entity := CollectionRefTestEntity{ID: 1, Name: "Item 1"}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/CollectionRefTestEntities(1)/$ref", nil)
	w := httptest.NewRecorder()

	handler.HandleEntityRef(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have @odata.id
	if _, ok := response["@odata.id"]; !ok {
		t.Error("Expected @odata.id in ref response")
	}
}

func TestHandleEntityRef_NotFound(t *testing.T) {
	handler, _ := setupCollectionRefTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/CollectionRefTestEntities(999)/$ref", nil)
	w := httptest.NewRecorder()

	handler.HandleEntityRef(w, req, "999")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleEntityRef_Head(t *testing.T) {
	handler, db := setupCollectionRefTestHandler(t)

	// Create entity
	entity := CollectionRefTestEntity{ID: 1, Name: "Item 1"}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/CollectionRefTestEntities(1)/$ref", nil)
	w := httptest.NewRecorder()

	handler.HandleEntityRef(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestHandleEntityRef_Options(t *testing.T) {
	handler, _ := setupCollectionRefTestHandler(t)

	req := httptest.NewRequest(http.MethodOptions, "/CollectionRefTestEntities(1)/$ref", nil)
	w := httptest.NewRecorder()

	handler.HandleEntityRef(w, req, "1")

	// Check if OPTIONS is supported or returns Method Not Allowed
	// Both are acceptable depending on implementation
	if w.Code != http.StatusOK && w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want 200 or 405", w.Code)
	}
}

func TestHandleEntity_Get_WithSelect(t *testing.T) {
	handler, db := setupCollectionRefTestHandler(t)

	// Create entity
	entity := CollectionRefTestEntity{ID: 1, Name: "Test Item", Category: "Electronics"}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/CollectionRefTestEntities(1)?$select=Name", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have Name
	if _, ok := response["Name"]; !ok {
		t.Error("Expected Name in response")
	}
}

func TestHandleCollection_Get_WithFilter(t *testing.T) {
	handler, db := setupCollectionRefTestHandler(t)

	// Create entities
	entities := []CollectionRefTestEntity{
		{ID: 1, Name: "Item 1", Category: "A"},
		{ID: 2, Name: "Item 2", Category: "B"},
		{ID: 3, Name: "Item 3", Category: "A"},
	}
	for _, e := range entities {
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("Failed to create entity: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/CollectionRefTestEntities?$filter=Category%20eq%20%27A%27", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have filtered results
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(value) != 2 {
		t.Errorf("Expected 2 items with Category 'A', got %d", len(value))
	}
}

func TestHandleCollection_Get_WithOrderBy(t *testing.T) {
	handler, db := setupCollectionRefTestHandler(t)

	// Create entities
	entities := []CollectionRefTestEntity{
		{ID: 1, Name: "Zebra", Category: "A"},
		{ID: 2, Name: "Apple", Category: "B"},
		{ID: 3, Name: "Mango", Category: "A"},
	}
	for _, e := range entities {
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("Failed to create entity: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/CollectionRefTestEntities?$orderby=Name", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have ordered results
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(value) != 3 {
		t.Errorf("Expected 3 items, got %d", len(value))
	}

	// First item should be Apple
	if len(value) > 0 {
		first, ok := value[0].(map[string]interface{})
		if !ok {
			t.Fatal("First item is not a map")
		}
		if first["Name"] != "Apple" {
			t.Errorf("First item Name = %v, want 'Apple'", first["Name"])
		}
	}
}
