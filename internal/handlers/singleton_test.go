package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SingletonTestSettings is a test singleton entity
type SingletonTestSettings struct {
	ID          uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Theme       string `json:"Theme"`
	Language    string `json:"Language"`
	MaxPageSize int    `json:"MaxPageSize"`
	Version     int    `json:"Version" odata:"etag"`
}

func setupSingletonHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&SingletonTestSettings{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Use AnalyzeSingleton to properly set up singleton metadata
	entityMeta, err := metadata.AnalyzeSingleton(SingletonTestSettings{}, "Settings")
	if err != nil {
		t.Fatalf("Failed to analyze singleton: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	return handler, db
}

func TestHandleSingleton_Get_Success(t *testing.T) {
	handler, db := setupSingletonHandler(t)

	// Insert singleton data
	settings := SingletonTestSettings{
		ID:          1,
		Theme:       "dark",
		Language:    "en",
		MaxPageSize: 50,
		Version:     1,
	}
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/Settings", nil)
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["Theme"] != "dark" {
		t.Errorf("Theme = %v, want dark", response["Theme"])
	}

	if response["Language"] != "en" {
		t.Errorf("Language = %v, want en", response["Language"])
	}

	// Check that @odata.context is present
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Expected @odata.context in response")
	}
}

func TestHandleSingleton_Get_NotFound(t *testing.T) {
	handler, _ := setupSingletonHandler(t)

	// Don't create any data - singleton should not be found
	req := httptest.NewRequest(http.MethodGet, "/Settings", nil)
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

func TestHandleSingleton_Head(t *testing.T) {
	handler, db := setupSingletonHandler(t)

	// Insert singleton data
	settings := SingletonTestSettings{
		ID:          1,
		Theme:       "dark",
		Language:    "en",
		MaxPageSize: 50,
		Version:     1,
	}
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/Settings", nil)
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestHandleSingleton_Patch_Success(t *testing.T) {
	handler, db := setupSingletonHandler(t)

	// Insert singleton data
	settings := SingletonTestSettings{
		ID:          1,
		Theme:       "dark",
		Language:    "en",
		MaxPageSize: 50,
		Version:     1,
	}
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Update theme via PATCH
	updateData := map[string]interface{}{
		"Theme": "light",
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/Settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the update in the database
	var updated SingletonTestSettings
	if err := db.First(&updated).Error; err != nil {
		t.Fatalf("Failed to fetch updated data: %v", err)
	}

	if updated.Theme != "light" {
		t.Errorf("Theme = %v, want light", updated.Theme)
	}

	// Other fields should remain unchanged
	if updated.Language != "en" {
		t.Errorf("Language = %v, want en", updated.Language)
	}
}

func TestHandleSingleton_Patch_NotFound(t *testing.T) {
	handler, _ := setupSingletonHandler(t)

	updateData := map[string]interface{}{
		"Theme": "light",
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/Settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleSingleton_Patch_InvalidJSON(t *testing.T) {
	handler, db := setupSingletonHandler(t)

	// Insert singleton data
	settings := SingletonTestSettings{
		ID:          1,
		Theme:       "dark",
		Language:    "en",
		MaxPageSize: 50,
		Version:     1,
	}
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/Settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleSingleton_Put_Success(t *testing.T) {
	handler, db := setupSingletonHandler(t)

	// Insert singleton data
	settings := SingletonTestSettings{
		ID:          1,
		Theme:       "dark",
		Language:    "en",
		MaxPageSize: 50,
		Version:     1,
	}
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Full replace via PUT
	newSettings := SingletonTestSettings{
		ID:          1,
		Theme:       "light",
		Language:    "de",
		MaxPageSize: 100,
		Version:     2,
	}
	body, _ := json.Marshal(newSettings)

	req := httptest.NewRequest(http.MethodPut, "/Settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the update in the database
	var updated SingletonTestSettings
	if err := db.First(&updated).Error; err != nil {
		t.Fatalf("Failed to fetch updated data: %v", err)
	}

	if updated.Theme != "light" {
		t.Errorf("Theme = %v, want light", updated.Theme)
	}

	if updated.Language != "de" {
		t.Errorf("Language = %v, want de", updated.Language)
	}

	if updated.MaxPageSize != 100 {
		t.Errorf("MaxPageSize = %v, want 100", updated.MaxPageSize)
	}
}

func TestHandleSingleton_Put_NotFound(t *testing.T) {
	handler, _ := setupSingletonHandler(t)

	newSettings := SingletonTestSettings{
		ID:          1,
		Theme:       "light",
		Language:    "de",
		MaxPageSize: 100,
		Version:     1,
	}
	body, _ := json.Marshal(newSettings)

	req := httptest.NewRequest(http.MethodPut, "/Settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleSingleton_Put_InvalidJSON(t *testing.T) {
	handler, db := setupSingletonHandler(t)

	// Insert singleton data
	settings := SingletonTestSettings{
		ID:          1,
		Theme:       "dark",
		Language:    "en",
		MaxPageSize: 50,
		Version:     1,
	}
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/Settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleSingleton_Options(t *testing.T) {
	handler, _ := setupSingletonHandler(t)

	req := httptest.NewRequest(http.MethodOptions, "/Settings", nil)
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, PATCH, PUT, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, PATCH, PUT, OPTIONS'", allowHeader)
	}
}

func TestHandleSingleton_MethodNotAllowed(t *testing.T) {
	handler, _ := setupSingletonHandler(t)

	methods := []string{http.MethodPost, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/Settings", nil)
			w := httptest.NewRecorder()

			handler.HandleSingleton(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestHandleSingleton_RemoveODataAnnotations(t *testing.T) {
	handler, db := setupSingletonHandler(t)

	// Insert singleton data
	settings := SingletonTestSettings{
		ID:          1,
		Theme:       "dark",
		Language:    "en",
		MaxPageSize: 50,
		Version:     1,
	}
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// PATCH with OData annotations that should be stripped
	updateData := map[string]interface{}{
		"Theme":          "light",
		"@odata.context": "should be removed",
		"@odata.etag":    "should be removed",
		"@odata.id":      "should be removed",
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/Settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	// Should succeed because OData annotations are removed
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the update in the database
	var updated SingletonTestSettings
	if err := db.First(&updated).Error; err != nil {
		t.Fatalf("Failed to fetch updated data: %v", err)
	}

	if updated.Theme != "light" {
		t.Errorf("Theme = %v, want light", updated.Theme)
	}
}

func TestHandleSingleton_DisabledMethod(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&SingletonTestSettings{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create singleton data
	settings := SingletonTestSettings{
		ID:          1,
		Theme:       "dark",
		Language:    "en",
		MaxPageSize: 50,
		Version:     1,
	}
	if err := db.Create(&settings).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Create singleton with disabled GET method
	entityMeta, err := metadata.AnalyzeSingleton(SingletonTestSettings{}, "Settings")
	if err != nil {
		t.Fatalf("Failed to analyze singleton: %v", err)
	}
	entityMeta.DisabledMethods = map[string]bool{"GET": true}

	handler := NewEntityHandler(db, entityMeta, nil)

	req := httptest.NewRequest(http.MethodGet, "/Settings", nil)
	w := httptest.NewRecorder()

	handler.HandleSingleton(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}
