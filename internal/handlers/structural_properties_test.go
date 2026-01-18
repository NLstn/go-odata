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

// StructuralPropertyTestEntity is a test entity for structural property tests
type StructuralPropertyTestEntity struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name"`
	Description string  `json:"Description"`
	Price       float64 `json:"Price"`
	Quantity    int     `json:"Quantity"`
	IsActive    bool    `json:"IsActive"`
	BinaryData  []byte  `json:"BinaryData"`
}

func setupStructuralPropertyHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&StructuralPropertyTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(StructuralPropertyTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	return handler, db
}

func TestHandleStructuralProperty_GetString(t *testing.T) {
	handler, db := setupStructuralPropertyHandler(t)

	// Insert test data
	entity := StructuralPropertyTestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "A test product description",
		Price:       99.99,
		Quantity:    10,
		IsActive:    true,
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(1)/Name", nil)
	w := httptest.NewRecorder()

	handler.HandleStructuralProperty(w, req, "1", "Name", false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["value"] != "Test Product" {
		t.Errorf("value = %v, want 'Test Product'", response["value"])
	}

	// Check @odata.context
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Expected @odata.context in response")
	}
}

func TestHandleStructuralProperty_GetNumeric(t *testing.T) {
	handler, db := setupStructuralPropertyHandler(t)

	// Insert test data
	entity := StructuralPropertyTestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "A test product description",
		Price:       99.99,
		Quantity:    42,
		IsActive:    true,
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	t.Run("Float property", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(1)/Price", nil)
		w := httptest.NewRecorder()

		handler.HandleStructuralProperty(w, req, "1", "Price", false)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["value"] != 99.99 {
			t.Errorf("value = %v, want 99.99", response["value"])
		}
	})

	t.Run("Int property", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(1)/Quantity", nil)
		w := httptest.NewRecorder()

		handler.HandleStructuralProperty(w, req, "1", "Quantity", false)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// JSON numbers are float64
		if response["value"] != float64(42) {
			t.Errorf("value = %v, want 42", response["value"])
		}
	})
}

func TestHandleStructuralProperty_GetBool(t *testing.T) {
	handler, db := setupStructuralPropertyHandler(t)

	// Insert test data
	entity := StructuralPropertyTestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "A test product description",
		Price:       99.99,
		Quantity:    10,
		IsActive:    true,
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(1)/IsActive", nil)
	w := httptest.NewRecorder()

	handler.HandleStructuralProperty(w, req, "1", "IsActive", false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["value"] != true {
		t.Errorf("value = %v, want true", response["value"])
	}
}

func TestHandleStructuralProperty_GetRawValue(t *testing.T) {
	handler, db := setupStructuralPropertyHandler(t)

	// Insert test data
	entity := StructuralPropertyTestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "A test product description",
		Price:       99.99,
		Quantity:    42,
		IsActive:    true,
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	t.Run("String $value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(1)/Name/$value", nil)
		w := httptest.NewRecorder()

		handler.HandleStructuralProperty(w, req, "1", "Name", true)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		// Content-Type can include charset
		if contentType != "text/plain" && contentType != "text/plain; charset=utf-8" {
			t.Errorf("Content-Type = %v, want text/plain or text/plain; charset=utf-8", contentType)
		}

		body := w.Body.String()
		if body != "Test Product" {
			t.Errorf("body = %v, want 'Test Product'", body)
		}
	})

	t.Run("Numeric $value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(1)/Price/$value", nil)
		w := httptest.NewRecorder()

		handler.HandleStructuralProperty(w, req, "1", "Price", true)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		// Content-Type can include charset
		if contentType != "text/plain" && contentType != "text/plain; charset=utf-8" {
			t.Errorf("Content-Type = %v, want text/plain or text/plain; charset=utf-8", contentType)
		}

		body := w.Body.String()
		if body != "99.99" {
			t.Errorf("body = %v, want '99.99'", body)
		}
	})

	t.Run("Bool $value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(1)/IsActive/$value", nil)
		w := httptest.NewRecorder()

		handler.HandleStructuralProperty(w, req, "1", "IsActive", true)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		body := w.Body.String()
		if body != "true" {
			t.Errorf("body = %v, want 'true'", body)
		}
	})
}

func TestHandleStructuralProperty_GetBinaryValue(t *testing.T) {
	handler, db := setupStructuralPropertyHandler(t)

	// Insert test data with binary content
	entity := StructuralPropertyTestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "A test product description",
		Price:       99.99,
		Quantity:    10,
		IsActive:    true,
		BinaryData:  []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}, // "Hello"
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(1)/BinaryData/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleStructuralProperty(w, req, "1", "BinaryData", true)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/octet-stream" {
		t.Errorf("Content-Type = %v, want application/octet-stream", contentType)
	}

	body := w.Body.Bytes()
	expectedBody := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}
	if string(body) != string(expectedBody) {
		t.Errorf("body = %v, want %v", body, expectedBody)
	}
}

func TestHandleStructuralProperty_EntityNotFound(t *testing.T) {
	handler, _ := setupStructuralPropertyHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(999)/Name", nil)
	w := httptest.NewRecorder()

	handler.HandleStructuralProperty(w, req, "999", "Name", false)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleStructuralProperty_PropertyNotFound(t *testing.T) {
	handler, db := setupStructuralPropertyHandler(t)

	// Insert test data
	entity := StructuralPropertyTestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "A test product description",
		Price:       99.99,
		Quantity:    10,
		IsActive:    true,
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(1)/NonExistentProperty", nil)
	w := httptest.NewRecorder()

	handler.HandleStructuralProperty(w, req, "1", "NonExistentProperty", false)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleStructuralProperty_InvalidKey(t *testing.T) {
	handler, db := setupStructuralPropertyHandler(t)

	// Insert test data
	entity := StructuralPropertyTestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "A test product description",
		Price:       99.99,
		Quantity:    10,
		IsActive:    true,
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(invalid)/Name", nil)
	w := httptest.NewRecorder()

	handler.HandleStructuralProperty(w, req, "invalid", "Name", false)

	// SQLite returns 404 for invalid keys since it tries to query and fails
	// This behavior is acceptable as the entity is not found
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want 400 or 404. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleStructuralProperty_Head(t *testing.T) {
	handler, db := setupStructuralPropertyHandler(t)

	// Insert test data
	entity := StructuralPropertyTestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "A test product description",
		Price:       99.99,
		Quantity:    10,
		IsActive:    true,
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/StructuralPropertyTestEntities(1)/Name", nil)
	w := httptest.NewRecorder()

	handler.HandleStructuralProperty(w, req, "1", "Name", false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestHandleStructuralProperty_Options(t *testing.T) {
	handler, _ := setupStructuralPropertyHandler(t)

	req := httptest.NewRequest(http.MethodOptions, "/StructuralPropertyTestEntities(1)/Name", nil)
	w := httptest.NewRecorder()

	handler.HandleStructuralProperty(w, req, "1", "Name", false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}
}

func TestHandleStructuralProperty_MethodNotAllowed(t *testing.T) {
	handler, _ := setupStructuralPropertyHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/StructuralPropertyTestEntities(1)/Name", nil)
			w := httptest.NewRecorder()

			handler.HandleStructuralProperty(w, req, "1", "Name", false)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestHandleStructuralProperty_MetadataLevel_None(t *testing.T) {
	handler, db := setupStructuralPropertyHandler(t)

	// Insert test data
	entity := StructuralPropertyTestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "A test product description",
		Price:       99.99,
		Quantity:    10,
		IsActive:    true,
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/StructuralPropertyTestEntities(1)/Name", nil)
	req.Header.Set("Accept", "application/json;odata.metadata=none")
	w := httptest.NewRecorder()

	handler.HandleStructuralProperty(w, req, "1", "Name", false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// With metadata=none, @odata.context should not be present
	if _, ok := response["@odata.context"]; ok {
		t.Error("Expected @odata.context to NOT be present with metadata=none")
	}

	// But value should still be present
	if response["value"] != "Test Product" {
		t.Errorf("value = %v, want 'Test Product'", response["value"])
	}
}
