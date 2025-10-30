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

type etagTestEntity struct {
	ID      int    `json:"id" gorm:"primarykey" odata:"key"`
	Version int    `json:"version" odata:"etag"`
	Name    string `json:"name"`
}

func setupETagTestHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&etagTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(etagTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta)
	return handler, db
}

func TestHandlePatchEntityIfMatchMismatch(t *testing.T) {
	handler, db := setupETagTestHandler(t)

	entity := etagTestEntity{ID: 1, Version: 1, Name: "Original"}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/etagTestEntities(1)", strings.NewReader(`{"name":"Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	req.Header.Set(HeaderIfMatch, "W/\"mismatched\"")

	w := httptest.NewRecorder()

	handler.handlePatchEntity(w, req, "1")

	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusPreconditionFailed)
	}

	if header := w.Header().Get(HeaderPreferenceApplied); header != "" {
		t.Fatalf("Preference-Applied header was set: %q", header)
	}

	if header := w.Header().Get(HeaderODataEntityId); header != "" {
		t.Fatalf("OData-EntityId header was set: %q", header)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if _, ok := body["error"]; !ok {
		t.Fatalf("Response does not contain error object: %v", body)
	}
}

func TestHandlePutEntityIfMatchMismatch(t *testing.T) {
	handler, db := setupETagTestHandler(t)

	entity := etagTestEntity{ID: 1, Version: 1, Name: "Original"}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	reqBody := `{"id":1,"version":1,"name":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/etagTestEntities(1)", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	req.Header.Set(HeaderIfMatch, "W/\"mismatched\"")

	w := httptest.NewRecorder()

	handler.handlePutEntity(w, req, "1")

	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusPreconditionFailed)
	}

	if header := w.Header().Get(HeaderPreferenceApplied); header != "" {
		t.Fatalf("Preference-Applied header was set: %q", header)
	}

	if header := w.Header().Get(HeaderODataEntityId); header != "" {
		t.Fatalf("OData-EntityId header was set: %q", header)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if _, ok := body["error"]; !ok {
		t.Fatalf("Response does not contain error object: %v", body)
	}
}
