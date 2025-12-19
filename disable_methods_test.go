package odata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
}

func setupDisableMethodsTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create test data
	db.Create(&User{ID: 1, Name: "Alice"})
	db.Create(&User{ID: 2, Name: "Bob"})

	return db
}

func TestDisableHTTPMethods_POST(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Disable POST for Users
	if err := service.DisableHTTPMethods("Users", "POST"); err != nil {
		t.Fatalf("Failed to disable POST method: %v", err)
	}

	// Test that POST returns 405
	reqBody := `{"name": "Charlie"}`
	req := httptest.NewRequest(http.MethodPost, "/Users", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}

	// Test that GET still works
	req = httptest.NewRequest(http.MethodGet, "/Users", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for GET, got %d", w.Code)
	}
}

func TestDisableHTTPMethods_DELETE(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Disable DELETE for Users
	if err := service.DisableHTTPMethods("Users", "DELETE"); err != nil {
		t.Fatalf("Failed to disable DELETE method: %v", err)
	}

	// Test that DELETE returns 405
	req := httptest.NewRequest(http.MethodDelete, "/Users(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}

	// Test that GET still works
	req = httptest.NewRequest(http.MethodGet, "/Users(1)", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for GET, got %d", w.Code)
	}
}

func TestDisableHTTPMethods_PATCH(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Disable PATCH for Users
	if err := service.DisableHTTPMethods("Users", "PATCH"); err != nil {
		t.Fatalf("Failed to disable PATCH method: %v", err)
	}

	// Test that PATCH returns 405
	reqBody := `{"name": "Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/Users(1)", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestDisableHTTPMethods_PUT(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Disable PUT for Users
	if err := service.DisableHTTPMethods("Users", "PUT"); err != nil {
		t.Fatalf("Failed to disable PUT method: %v", err)
	}

	// Test that PUT returns 405
	reqBody := `{"id": 1, "name": "Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/Users(1)", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestDisableHTTPMethods_GET(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Disable GET for Users
	if err := service.DisableHTTPMethods("Users", "GET"); err != nil {
		t.Fatalf("Failed to disable GET method: %v", err)
	}

	// Test that GET returns 405 for collection
	req := httptest.NewRequest(http.MethodGet, "/Users", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for GET collection, got %d", w.Code)
	}

	// Test that GET returns 405 for single entity
	req = httptest.NewRequest(http.MethodGet, "/Users(1)", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for GET entity, got %d", w.Code)
	}

	// Test that GET returns 405 for $count
	req = httptest.NewRequest(http.MethodGet, "/Users/$count", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for GET $count, got %d", w.Code)
	}
}

func TestDisableHTTPMethods_HEAD_BlockedWhenGETDisabled(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Disable GET for Users
	if err := service.DisableHTTPMethods("Users", "GET"); err != nil {
		t.Fatalf("Failed to disable GET method: %v", err)
	}

	// Test that HEAD returns 405 for collection (should be blocked when GET is disabled)
	req := httptest.NewRequest(http.MethodHead, "/Users", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for HEAD collection when GET is disabled, got %d", w.Code)
	}

	// Test that HEAD returns 405 for single entity
	req = httptest.NewRequest(http.MethodHead, "/Users(1)", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for HEAD entity when GET is disabled, got %d", w.Code)
	}

	// Test that HEAD returns 405 for $count
	req = httptest.NewRequest(http.MethodHead, "/Users/$count", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for HEAD $count when GET is disabled, got %d", w.Code)
	}
}

func TestDisableHTTPMethods_HEAD_AllowedWhenGETEnabled(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Don't disable GET - HEAD should work

	// Test that HEAD returns 200 for collection
	req := httptest.NewRequest(http.MethodHead, "/Users", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for HEAD collection when GET is enabled, got %d", w.Code)
	}

	// Test that HEAD returns 200 for single entity
	req = httptest.NewRequest(http.MethodHead, "/Users(1)", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for HEAD entity when GET is enabled, got %d", w.Code)
	}

	// Test that HEAD returns 200 for $count
	req = httptest.NewRequest(http.MethodHead, "/Users/$count", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for HEAD $count when GET is enabled, got %d", w.Code)
	}
}

func TestDisableHTTPMethods_Multiple(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Disable POST and DELETE for Users
	if err := service.DisableHTTPMethods("Users", "POST", "DELETE"); err != nil {
		t.Fatalf("Failed to disable methods: %v", err)
	}

	// Test that POST returns 405
	reqBody := `{"name": "Charlie"}`
	req := httptest.NewRequest(http.MethodPost, "/Users", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for POST, got %d", w.Code)
	}

	// Test that DELETE returns 405
	req = httptest.NewRequest(http.MethodDelete, "/Users(1)", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for DELETE, got %d", w.Code)
	}

	// Test that GET still works
	req = httptest.NewRequest(http.MethodGet, "/Users", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for GET, got %d", w.Code)
	}

	// Test that PATCH still works
	reqBody = `{"name": "Updated"}`
	req = httptest.NewRequest(http.MethodPatch, "/Users(1)", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Errorf("Expected status 200 or 204 for PATCH, got %d", w.Code)
	}
}

func TestDisableHTTPMethods_InvalidEntitySet(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	// Try to disable methods for non-existent entity set
	err := service.DisableHTTPMethods("NonExistent", "POST")
	if err == nil {
		t.Error("Expected error when disabling methods for non-existent entity set")
	}

	expected := "entity set 'NonExistent' is not registered"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestDisableHTTPMethods_InvalidMethod(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Try to disable an invalid HTTP method
	err := service.DisableHTTPMethods("Users", "INVALID")
	if err == nil {
		t.Error("Expected error when disabling invalid HTTP method")
	}

	if !strings.Contains(err.Error(), "unsupported HTTP method") {
		t.Errorf("Expected error about unsupported method, got: %s", err.Error())
	}
}

func TestDisableHTTPMethods_CaseInsensitive(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Disable POST using lowercase
	if err := service.DisableHTTPMethods("Users", "post"); err != nil {
		t.Fatalf("Failed to disable POST method: %v", err)
	}

	// Test that POST returns 405
	reqBody := `{"name": "Charlie"}`
	req := httptest.NewRequest(http.MethodPost, "/Users", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestDisableHTTPMethods_ErrorResponse(t *testing.T) {
	db := setupDisableMethodsTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(&User{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Disable POST for Users
	if err := service.DisableHTTPMethods("Users", "POST"); err != nil {
		t.Fatalf("Failed to disable POST method: %v", err)
	}

	// Test that POST returns proper error response
	reqBody := `{"name": "Charlie"}`
	req := httptest.NewRequest(http.MethodPost, "/Users", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}

	// Parse the error response
	var errorResponse map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	// Check that error message is present
	if errorObj, ok := errorResponse["error"].(map[string]interface{}); ok {
		if message, ok := errorObj["message"].(string); ok {
			if !strings.Contains(message, "not allowed") {
				t.Errorf("Expected error message to contain 'not allowed', got: %s", message)
			}
		} else {
			t.Error("Expected error message in response")
		}
	} else {
		t.Error("Expected error object in response")
	}
}
