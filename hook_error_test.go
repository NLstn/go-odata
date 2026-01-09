package odata_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Employee entity with hook for testing custom status codes
type EmployeeWithCustomHook struct {
	ID        int    `gorm:"primaryKey" odata:"key"`
	Name      string
	IsBlocked bool `gorm:"-" odata:"-"` // Internal flag for testing
}

func (EmployeeWithCustomHook) TableName() string {
	return "employees"
}

// ODataBeforeReadEntity returns a 401 Unauthorized status code
func (e *EmployeeWithCustomHook) ODataBeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	// Simulate checking if user is authenticated by checking a header
	if r.Header.Get("Authorization") == "" {
		return nil, &odata.HookError{
			StatusCode: http.StatusUnauthorized,
			Message:    "User is not authenticated",
		}
	}
	return nil, nil
}

// ODataBeforeReadCollection returns a 404 Not Found for demonstration
func (e *EmployeeWithCustomHook) ODataBeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	// Simulate a scenario where the collection doesn't exist for this user
	if r.Header.Get("X-Tenant-ID") == "missing" {
		return nil, odata.NewHookError(http.StatusNotFound, "Collection not found for this tenant")
	}
	return nil, nil
}

func TestHookError_CustomStatusCodes(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(&EmployeeWithCustomHook{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	testEmployee := &EmployeeWithCustomHook{ID: 1, Name: "John Doe"}
	if err := db.Create(testEmployee).Error; err != nil {
		t.Fatalf("Failed to create test employee: %v", err)
	}

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&EmployeeWithCustomHook{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	t.Run("BeforeReadEntity returns 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/EmployeeWithCustomHooks(1)", nil)
		// No Authorization header - should trigger 401
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}

		body, _ := io.ReadAll(w.Body)
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		errorObj := errorResp["error"].(map[string]interface{})
		if errorObj["message"] != "User is not authenticated" {
			t.Errorf("Expected message 'User is not authenticated', got %s", errorObj["message"])
		}
	})

	t.Run("BeforeReadEntity allows access with Authorization header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/EmployeeWithCustomHooks(1)", nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("BeforeReadCollection returns 404 Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/EmployeeWithCustomHooks", nil)
		req.Header.Set("X-Tenant-ID", "missing")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
		}

		body, _ := io.ReadAll(w.Body)
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		errorObj := errorResp["error"].(map[string]interface{})
		if errorObj["message"] != "Collection not found for this tenant" {
			t.Errorf("Expected message 'Collection not found for this tenant', got %s", errorObj["message"])
		}
	})

	t.Run("BeforeReadCollection allows access without X-Tenant-ID header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/EmployeeWithCustomHooks", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestHookError_DefaultFallback(t *testing.T) {
	// Test that HookError works correctly
	err := &odata.HookError{
		Message: "Test error",
	}

	if err.Error() != "Test error" {
		t.Errorf("Expected error message 'Test error', got %s", err.Error())
	}

	// Test with StatusCode
	errWithCode := &odata.HookError{
		StatusCode: http.StatusUnauthorized,
		Message:    "Unauthorized access",
	}

	if errWithCode.Error() != "Unauthorized access" {
		t.Errorf("Expected error message 'Unauthorized access', got %s", errWithCode.Error())
	}
}

func TestNewHookError(t *testing.T) {
	err := odata.NewHookError(http.StatusForbidden, "Access forbidden")

	if err.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status code %d, got %d", http.StatusForbidden, err.StatusCode)
	}

	if err.Message != "Access forbidden" {
		t.Errorf("Expected message 'Access forbidden', got %s", err.Message)
	}

	if err.Error() != "Access forbidden" {
		t.Errorf("Expected error string 'Access forbidden', got %s", err.Error())
	}
}
