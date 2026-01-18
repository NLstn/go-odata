package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NonMediaEntity is a test entity that is NOT a media entity
type NonMediaEntity struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name"`
}

func setupNonMediaHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NonMediaEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(NonMediaEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	return handler, db
}

func TestHandleMediaEntityValue_NotMediaEntity(t *testing.T) {
	handler, _ := setupNonMediaHandler(t)

	// Accessing $value on a non-media entity should fail
	req := httptest.NewRequest(http.MethodGet, "/NonMediaEntities(1)/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleMediaEntityValue(w, req, "1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleMediaEntityValue_GetNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NonMediaEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(NonMediaEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Manually mark as media entity for testing purposes
	entityMeta.HasStream = true

	handler := NewEntityHandler(db, entityMeta, nil)

	req := httptest.NewRequest(http.MethodGet, "/NonMediaEntities(999)/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleMediaEntityValue(w, req, "999")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleMediaEntityValue_Options(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NonMediaEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(NonMediaEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Manually mark as media entity
	entityMeta.HasStream = true

	handler := NewEntityHandler(db, entityMeta, nil)

	req := httptest.NewRequest(http.MethodOptions, "/NonMediaEntities(1)/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleMediaEntityValue(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, PUT, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, PUT, OPTIONS'", allowHeader)
	}
}

func TestHandleMediaEntityValue_MethodNotAllowed(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NonMediaEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(NonMediaEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Manually mark as media entity
	entityMeta.HasStream = true

	handler := NewEntityHandler(db, entityMeta, nil)

	methods := []string{http.MethodPost, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/NonMediaEntities(1)/$value", nil)
			w := httptest.NewRecorder()

			handler.HandleMediaEntityValue(w, req, "1")

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestHandleMediaEntityValue_InvalidKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NonMediaEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(NonMediaEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Manually mark as media entity
	entityMeta.HasStream = true

	handler := NewEntityHandler(db, entityMeta, nil)

	req := httptest.NewRequest(http.MethodGet, "/NonMediaEntities(invalid)/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleMediaEntityValue(w, req, "invalid")

	// SQLite will treat invalid as a string and try to query, resulting in not found
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want 404 or 400", w.Code)
	}
}
