package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ServiceDocProduct is a test entity for service document tests
type ServiceDocProduct struct {
	ID    int     `json:"id" gorm:"primarykey" odata:"key"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// ServiceDocCategory is another test entity
type ServiceDocCategory struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
}

func setupServiceDocTestService(t *testing.T) *odata.Service {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ServiceDocProduct{}, &ServiceDocCategory{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(ServiceDocProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}
	if err := service.RegisterEntity(ServiceDocCategory{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service
}

func TestServiceDocument_GET(t *testing.T) {
	service := setupServiceDocTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify @odata.context contains $metadata
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("@odata.context is missing or not a string")
	}
	// Context should end with /$metadata
	if len(context) < 10 || context[len(context)-10:] != "/$metadata" {
		t.Errorf("@odata.context = %v, should end with /$metadata", context)
	}

	// Verify value array exists
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value is missing or not an array")
	}

	// Verify entity sets are listed
	foundProducts := false
	foundCategories := false
	for _, item := range value {
		entity, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := entity["name"].(string)
		kind, _ := entity["kind"].(string)
		url, _ := entity["url"].(string)

		if name == "ServiceDocProducts" {
			foundProducts = true
			if kind != "EntitySet" {
				t.Errorf("ServiceDocProducts kind = %v, want EntitySet", kind)
			}
			if url != "ServiceDocProducts" {
				t.Errorf("ServiceDocProducts url = %v, want ServiceDocProducts", url)
			}
		}
		if name == "ServiceDocCategories" {
			foundCategories = true
			if kind != "EntitySet" {
				t.Errorf("ServiceDocCategories kind = %v, want EntitySet", kind)
			}
		}
	}

	if !foundProducts {
		t.Error("ServiceDocProducts not found in service document")
	}
	if !foundCategories {
		t.Error("ServiceDocCategories not found in service document")
	}
}

func TestServiceDocument_HEAD(t *testing.T) {
	service := setupServiceDocTestService(t)

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}

	// Verify body is empty for HEAD request
	if w.Body.Len() != 0 {
		t.Errorf("Body should be empty for HEAD request, got: %s", w.Body.String())
	}

	// Verify Content-Length header is set
	contentLength := w.Header().Get("Content-Length")
	if contentLength == "" || contentLength == "0" {
		t.Errorf("Content-Length should be set to body size for HEAD request")
	}
}

func TestServiceDocument_POST_NotAllowed(t *testing.T) {
	service := setupServiceDocTestService(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestServiceDocument_PUT_NotAllowed(t *testing.T) {
	service := setupServiceDocTestService(t)

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestServiceDocument_DELETE_NotAllowed(t *testing.T) {
	service := setupServiceDocTestService(t)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestServiceDocument_PATCH_NotAllowed(t *testing.T) {
	service := setupServiceDocTestService(t)

	req := httptest.NewRequest(http.MethodPatch, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestServiceDocument_WithHost(t *testing.T) {
	service := setupServiceDocTestService(t)

	req := httptest.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify @odata.context includes full URL
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("@odata.context is missing or not a string")
	}
	expected := "http://example.com:8080/$metadata"
	if context != expected {
		t.Errorf("@odata.context = %v, want %v", context, expected)
	}
}

func TestServiceDocument_MetadataLevelFull(t *testing.T) {
	service := setupServiceDocTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json;odata.metadata=full")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify Content-Type reflects metadata level
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=full" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=full", contentType)
	}
}

func TestServiceDocument_MetadataLevelNone(t *testing.T) {
	service := setupServiceDocTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json;odata.metadata=none")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify Content-Type reflects metadata level
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=none" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=none", contentType)
	}
}

func TestServiceDocument_EmptyService(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify value is empty array
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value is missing or not an array")
	}
	if len(value) != 0 {
		t.Errorf("Expected empty value array for service with no entities, got %d items", len(value))
	}
}

func TestServiceDocument_EntitySetStructure(t *testing.T) {
	service := setupServiceDocTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value is missing or not an array")
	}

	// Verify each entity set has required fields
	for _, item := range value {
		entity, ok := item.(map[string]interface{})
		if !ok {
			t.Fatal("entity set entry is not an object")
		}

		// Verify required fields per OData spec
		if _, ok := entity["name"]; !ok {
			t.Error("Entity set missing 'name' field")
		}
		if _, ok := entity["kind"]; !ok {
			t.Error("Entity set missing 'kind' field")
		}
		if _, ok := entity["url"]; !ok {
			t.Error("Entity set missing 'url' field")
		}
	}
}
