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

type TestEntity struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
}

func setupTestHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(TestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta)
	return handler, db
}

func TestEntityHandlerCollection(t *testing.T) {
	handler, db := setupTestHandler(t)

	// Insert test data
	testData := []TestEntity{
		{ID: 1, Name: "Test 1"},
		{ID: 2, Name: "Test 2"},
		{ID: 3, Name: "Test 3"},
	}
	for _, entity := range testData {
		db.Create(&entity)
	}

	req := httptest.NewRequest(http.MethodGet, "/TestEntities", nil)
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

	if len(value) != 3 {
		t.Errorf("len(value) = %v, want 3", len(value))
	}
}

func TestEntityHandlerCollectionEmpty(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/TestEntities", nil)
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

	if len(value) != 0 {
		t.Errorf("len(value) = %v, want 0", len(value))
	}
}

func TestEntityHandlerCollectionMethodNotAllowed(t *testing.T) {
	handler, _ := setupTestHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/TestEntities", nil)
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestEntityHandlerEntity(t *testing.T) {
	handler, db := setupTestHandler(t)

	// Insert test data
	entity := TestEntity{ID: 42, Name: "Test Entity"}
	db.Create(&entity)

	req := httptest.NewRequest(http.MethodGet, "/TestEntities(42)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "42")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["name"] != "Test Entity" {
		t.Errorf("name = %v, want Test Entity", response["name"])
	}

	if response["id"] != float64(42) {
		t.Errorf("id = %v, want 42", response["id"])
	}

	// Check context URL
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("@odata.context is not a string")
	}

	if context == "" {
		t.Error("@odata.context should not be empty")
	}
}

func TestEntityHandlerEntityNotFound(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/TestEntities(999)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "999")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

func TestEntityHandlerEntityMethodNotAllowed(t *testing.T) {
	handler, _ := setupTestHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/TestEntities(1)", nil)
			w := httptest.NewRecorder()

			handler.HandleEntity(w, req, "1")

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestMetadataHandler(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	entityMeta, _ := metadata.AnalyzeEntity(TestEntity{})
	entities["TestEntities"] = entityMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/xml" {
		t.Errorf("Content-Type = %v, want application/xml", contentType)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Metadata response body is empty")
	}

	// Check that entity set name appears in metadata
	if !contains(body, "TestEntities") {
		t.Error("Metadata should contain TestEntities")
	}
}

func TestMetadataHandlerMethodNotAllowed(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	handler := NewMetadataHandler(entities)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/$metadata", nil)
			w := httptest.NewRecorder()

			handler.HandleMetadata(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestServiceDocumentHandler(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	entityMeta, _ := metadata.AnalyzeEntity(TestEntity{})
	entities["TestEntities"] = entityMeta

	handler := NewServiceDocumentHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

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

	if len(value) != 1 {
		t.Errorf("len(value) = %v, want 1", len(value))
	}
}

func TestServiceDocumentHandlerEmpty(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	handler := NewServiceDocumentHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

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

	if len(value) != 0 {
		t.Errorf("len(value) = %v, want 0", len(value))
	}
}

func TestServiceDocumentHandlerMethodNotAllowed(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	handler := NewServiceDocumentHandler(entities)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()

			handler.HandleServiceDocument(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr &&
		(s[0:1] == substr || s[len(s)-1:] == substr ||
			len(s) > len(substr) && (s[0:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
