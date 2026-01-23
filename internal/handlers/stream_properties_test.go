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

// StreamPropertyTestEntity is a test entity with stream properties.
// Stream property convention in this library:
// - The field marked with `odata:"stream"` (Photo) indicates this is a stream property
// - The actual binary content is stored in {PropertyName}Content (PhotoContent)
// - The content type is stored in {PropertyName}ContentType (PhotoContentType)
// This allows the library to automatically handle media uploads and downloads.
type StreamPropertyTestEntity struct {
	ID               uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name             string `json:"Name"`
	Photo            []byte `json:"-" odata:"stream"`   // Stream property marker
	PhotoContent     []byte `json:"-" gorm:"type:blob"` // Actual binary content
	PhotoContentType string `json:"-"`                  // Content type following convention
}

func setupStreamPropertyHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&StreamPropertyTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(StreamPropertyTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	return handler, db
}

func TestHandleStreamProperty_GetStreamWithValue(t *testing.T) {
	handler, db := setupStreamPropertyHandler(t)

	// Insert test data with binary content
	entity := StreamPropertyTestEntity{
		ID:               1,
		Name:             "Test Entity",
		PhotoContent:     []byte("binary photo data"),
		PhotoContentType: "image/jpeg",
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Request with $value to get binary content
	req := httptest.NewRequest(http.MethodGet, "/StreamPropertyTestEntities(1)/Photo/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleStreamProperty(w, req, "1", "Photo", true)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "image/jpeg" {
		t.Errorf("Content-Type = %v, want 'image/jpeg'", contentType)
	}

	body := w.Body.Bytes()
	if string(body) != "binary photo data" {
		t.Errorf("body = %v, want 'binary photo data'", string(body))
	}
}

func TestHandleStreamProperty_GetStreamMetadata(t *testing.T) {
	handler, db := setupStreamPropertyHandler(t)

	// Insert test data
	entity := StreamPropertyTestEntity{
		ID:               1,
		Name:             "Test Entity",
		PhotoContent:     []byte("binary photo data"),
		PhotoContentType: "image/jpeg",
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Request without $value to get stream metadata
	req := httptest.NewRequest(http.MethodGet, "/StreamPropertyTestEntities(1)/Photo", nil)
	w := httptest.NewRecorder()

	handler.HandleStreamProperty(w, req, "1", "Photo", false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should contain media link info
	if _, ok := response["@odata.mediaReadLink"]; !ok {
		t.Error("Expected @odata.mediaReadLink in response")
	}

	if response["@odata.mediaContentType"] != "image/jpeg" {
		t.Errorf("@odata.mediaContentType = %v, want 'image/jpeg'", response["@odata.mediaContentType"])
	}
}

func TestHandleStreamProperty_GetDefaultContentType(t *testing.T) {
	handler, db := setupStreamPropertyHandler(t)

	// Insert test data with no content type
	entity := StreamPropertyTestEntity{
		ID:               1,
		Name:             "Test Entity",
		PhotoContent:     []byte("binary photo data"),
		PhotoContentType: "", // Empty content type
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/StreamPropertyTestEntities(1)/Photo/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleStreamProperty(w, req, "1", "Photo", true)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Should default to application/octet-stream
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/octet-stream" {
		t.Errorf("Content-Type = %v, want 'application/octet-stream'", contentType)
	}
}

func TestHandleStreamProperty_EntityNotFound(t *testing.T) {
	handler, _ := setupStreamPropertyHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/StreamPropertyTestEntities(999)/Photo/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleStreamProperty(w, req, "999", "Photo", true)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleStreamProperty_PropertyNotFound(t *testing.T) {
	handler, db := setupStreamPropertyHandler(t)

	// Insert test data
	entity := StreamPropertyTestEntity{
		ID:               1,
		Name:             "Test Entity",
		PhotoContent:     []byte("binary photo data"),
		PhotoContentType: "image/jpeg",
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/StreamPropertyTestEntities(1)/NonExistent/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleStreamProperty(w, req, "1", "NonExistent", true)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleStreamProperty_Head(t *testing.T) {
	handler, db := setupStreamPropertyHandler(t)

	// Insert test data
	entity := StreamPropertyTestEntity{
		ID:               1,
		Name:             "Test Entity",
		PhotoContent:     []byte("binary photo data"),
		PhotoContentType: "image/jpeg",
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/StreamPropertyTestEntities(1)/Photo/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleStreamProperty(w, req, "1", "Photo", true)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestHandleStreamProperty_Options(t *testing.T) {
	handler, _ := setupStreamPropertyHandler(t)

	t.Run("Options with $value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/StreamPropertyTestEntities(1)/Photo/$value", nil)
		w := httptest.NewRecorder()

		handler.HandleStreamProperty(w, req, "1", "Photo", true)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		allowHeader := w.Header().Get("Allow")
		if allowHeader != "GET, HEAD, PUT, OPTIONS" {
			t.Errorf("Allow header = %v, want 'GET, HEAD, PUT, OPTIONS'", allowHeader)
		}
	})

	t.Run("Options without $value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/StreamPropertyTestEntities(1)/Photo", nil)
		w := httptest.NewRecorder()

		handler.HandleStreamProperty(w, req, "1", "Photo", false)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		allowHeader := w.Header().Get("Allow")
		if allowHeader != "GET, HEAD, OPTIONS" {
			t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
		}
	})
}

func TestHandleStreamProperty_PutWithoutValue(t *testing.T) {
	handler, _ := setupStreamPropertyHandler(t)

	// PUT without /$value should fail
	req := httptest.NewRequest(http.MethodPut, "/StreamPropertyTestEntities(1)/Photo", bytes.NewReader([]byte("data")))
	w := httptest.NewRecorder()

	handler.HandleStreamProperty(w, req, "1", "Photo", false)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleStreamProperty_MethodNotAllowed(t *testing.T) {
	handler, _ := setupStreamPropertyHandler(t)

	methods := []string{http.MethodPost, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/StreamPropertyTestEntities(1)/Photo/$value", nil)
			w := httptest.NewRecorder()

			handler.HandleStreamProperty(w, req, "1", "Photo", true)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

// TestHandleStreamProperty_Put tests uploading stream property content via PUT
func TestHandleStreamProperty_Put(t *testing.T) {
	handler, db := setupStreamPropertyHandler(t)

	// Create an entity first
	entity := StreamPropertyTestEntity{
		ID:   1,
		Name: "Test Entity",
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Upload new photo content
	newContent := []byte("new photo binary data")
	req := httptest.NewRequest(http.MethodPut, "/StreamPropertyTestEntities(1)/Photo/$value", bytes.NewReader(newContent))
	req.Header.Set("Content-Type", "image/png")
	w := httptest.NewRecorder()

	handler.HandleStreamProperty(w, req, "1", "Photo", true)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the content was updated
	var updated StreamPropertyTestEntity
	if err := db.First(&updated, 1).Error; err != nil {
		t.Fatalf("Failed to fetch updated entity: %v", err)
	}

	if string(updated.PhotoContent) != string(newContent) {
		t.Errorf("PhotoContent = %v, want %v", string(updated.PhotoContent), string(newContent))
	}

	if updated.PhotoContentType != "image/png" {
		t.Errorf("PhotoContentType = %v, want 'image/png'", updated.PhotoContentType)
	}
}

// TestHandleStreamProperty_PutDefaultContentType tests uploading stream property without content type
func TestHandleStreamProperty_PutDefaultContentType(t *testing.T) {
	handler, db := setupStreamPropertyHandler(t)

	// Create an entity first
	entity := StreamPropertyTestEntity{
		ID:   1,
		Name: "Test Entity",
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Upload without Content-Type header
	newContent := []byte("binary data")
	req := httptest.NewRequest(http.MethodPut, "/StreamPropertyTestEntities(1)/Photo/$value", bytes.NewReader(newContent))
	w := httptest.NewRecorder()

	handler.HandleStreamProperty(w, req, "1", "Photo", true)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Verify default content type was used
	var updated StreamPropertyTestEntity
	if err := db.First(&updated, 1).Error; err != nil {
		t.Fatalf("Failed to fetch updated entity: %v", err)
	}

	if updated.PhotoContentType != "application/octet-stream" {
		t.Errorf("PhotoContentType = %v, want 'application/octet-stream'", updated.PhotoContentType)
	}
}

// TestHandleStreamProperty_PutNonExistentEntity tests PUT on non-existent entity
func TestHandleStreamProperty_PutNonExistentEntity(t *testing.T) {
	handler, _ := setupStreamPropertyHandler(t)

	req := httptest.NewRequest(http.MethodPut, "/StreamPropertyTestEntities(999)/Photo/$value", bytes.NewReader([]byte("data")))
	w := httptest.NewRecorder()

	handler.HandleStreamProperty(w, req, "999", "Photo", true)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}
