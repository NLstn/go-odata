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

// NavTestParent is a parent entity for navigation tests
type NavTestParent struct {
	ID       uint           `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string         `json:"Name"`
	Children []NavTestChild `json:"Children" gorm:"foreignKey:ParentID"`
}

// NavTestChild is a child entity for navigation tests
type NavTestChild struct {
	ID       uint           `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string         `json:"Name"`
	ParentID uint           `json:"ParentID"`
	Parent   *NavTestParent `json:"Parent,omitempty" gorm:"foreignKey:ParentID"`
}

func setupNavTestHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NavTestParent{}, &NavTestChild{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	parentMeta, err := metadata.AnalyzeEntity(NavTestParent{})
	if err != nil {
		t.Fatalf("Failed to analyze parent entity: %v", err)
	}

	childMeta, err := metadata.AnalyzeEntity(NavTestChild{})
	if err != nil {
		t.Fatalf("Failed to analyze child entity: %v", err)
	}

	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"NavTestParents":  parentMeta,
		"NavTestChildren": childMeta,
	}

	handler := NewEntityHandler(db, parentMeta, nil)
	handler.SetEntitiesMetadata(entitiesMetadata)
	return handler, db
}

func TestHandleNavigationProperty_Get(t *testing.T) {
	handler, db := setupNavTestHandler(t)

	// Create parent with children
	parent := NavTestParent{ID: 1, Name: "Parent 1"}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("Failed to create parent: %v", err)
	}

	children := []NavTestChild{
		{ID: 1, Name: "Child 1", ParentID: 1},
		{ID: 2, Name: "Child 2", ParentID: 1},
	}
	if err := db.Create(&children).Error; err != nil {
		t.Fatalf("Failed to create children: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/NavTestParents(1)/Children", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationProperty(w, req, "1", "Children", false)

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

	if len(value) != 2 {
		t.Errorf("Expected 2 children, got %d", len(value))
	}
}

func TestHandleNavigationProperty_GetRef(t *testing.T) {
	handler, db := setupNavTestHandler(t)

	// Create parent with children
	parent := NavTestParent{ID: 1, Name: "Parent 1"}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("Failed to create parent: %v", err)
	}

	children := []NavTestChild{
		{ID: 1, Name: "Child 1", ParentID: 1},
		{ID: 2, Name: "Child 2", ParentID: 1},
	}
	if err := db.Create(&children).Error; err != nil {
		t.Fatalf("Failed to create children: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/NavTestParents(1)/Children/$ref", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationProperty(w, req, "1", "Children", true)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have a value array with refs
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(value) != 2 {
		t.Errorf("Expected 2 refs, got %d", len(value))
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

func TestHandleNavigationProperty_ParentNotFound(t *testing.T) {
	handler, _ := setupNavTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/NavTestParents(999)/Children", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationProperty(w, req, "999", "Children", false)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleNavigationProperty_InvalidProperty(t *testing.T) {
	handler, db := setupNavTestHandler(t)

	// Create parent
	parent := NavTestParent{ID: 1, Name: "Parent 1"}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("Failed to create parent: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/NavTestParents(1)/InvalidNav", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationProperty(w, req, "1", "InvalidNav", false)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleNavigationProperty_Head(t *testing.T) {
	handler, db := setupNavTestHandler(t)

	// Create parent with children
	parent := NavTestParent{ID: 1, Name: "Parent 1"}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("Failed to create parent: %v", err)
	}

	child := NavTestChild{ID: 1, Name: "Child 1", ParentID: 1}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("Failed to create child: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/NavTestParents(1)/Children", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationProperty(w, req, "1", "Children", false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestHandleNavigationProperty_Options(t *testing.T) {
	handler, _ := setupNavTestHandler(t)

	req := httptest.NewRequest(http.MethodOptions, "/NavTestParents(1)/Children", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationProperty(w, req, "1", "Children", false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader == "" {
		t.Error("Expected Allow header to be set")
	}
}

func TestHandleNavigationProperty_OptionsRef(t *testing.T) {
	handler, _ := setupNavTestHandler(t)

	req := httptest.NewRequest(http.MethodOptions, "/NavTestParents(1)/Children/$ref", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationProperty(w, req, "1", "Children", true)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader == "" {
		t.Error("Expected Allow header to be set")
	}
}

func TestHandleNavigationPropertyCount_Get(t *testing.T) {
	handler, db := setupNavTestHandler(t)

	// Create parent with children
	parent := NavTestParent{ID: 1, Name: "Parent 1"}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("Failed to create parent: %v", err)
	}

	children := []NavTestChild{
		{ID: 1, Name: "Child 1", ParentID: 1},
		{ID: 2, Name: "Child 2", ParentID: 1},
		{ID: 3, Name: "Child 3", ParentID: 1},
	}
	if err := db.Create(&children).Error; err != nil {
		t.Fatalf("Failed to create children: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/NavTestParents(1)/Children/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationPropertyCount(w, req, "1", "Children")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	body := w.Body.String()
	if body != "3" {
		t.Errorf("Count = %v, want '3'", body)
	}
}

func TestHandleNavigationPropertyCount_ParentNotFound(t *testing.T) {
	handler, _ := setupNavTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/NavTestParents(999)/Children/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationPropertyCount(w, req, "999", "Children")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleNavigationPropertyCount_Head(t *testing.T) {
	handler, db := setupNavTestHandler(t)

	// Create parent with child
	parent := NavTestParent{ID: 1, Name: "Parent 1"}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("Failed to create parent: %v", err)
	}

	child := NavTestChild{ID: 1, Name: "Child 1", ParentID: 1}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("Failed to create child: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/NavTestParents(1)/Children/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationPropertyCount(w, req, "1", "Children")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestHandleNavigationPropertyCount_Options(t *testing.T) {
	handler, _ := setupNavTestHandler(t)

	req := httptest.NewRequest(http.MethodOptions, "/NavTestParents(1)/Children/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationPropertyCount(w, req, "1", "Children")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}
}

func TestHandleNavigationPropertyCount_MethodNotAllowed(t *testing.T) {
	handler, _ := setupNavTestHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/NavTestParents(1)/Children/$count", nil)
			w := httptest.NewRecorder()

			handler.HandleNavigationPropertyCount(w, req, "1", "Children")

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}
