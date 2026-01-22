package odata_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Employee entity with hook for testing custom status codes
type EmployeeWithCustomHook struct {
	ID        int `gorm:"primaryKey" odata:"key"`
	Name      string
	IsBlocked bool `gorm:"-" odata:"-"` // Internal flag for testing
}

func (EmployeeWithCustomHook) TableName() string {
	return "employees"
}

// ODataBeforeReadEntity returns a 401 Unauthorized status code
func (e *EmployeeWithCustomHook) ODataBeforeReadEntity(ctx context.Context, r *http.Request, opts *odata.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
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
func (e *EmployeeWithCustomHook) ODataBeforeReadCollection(ctx context.Context, r *http.Request, opts *odata.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
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
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
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

// Test entity for write hooks with custom status codes
type ProductWithWriteHook struct {
	ID    int     `gorm:"primaryKey" odata:"key"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func (ProductWithWriteHook) TableName() string {
	return "products"
}

// ODataBeforeCreate returns a 402 Payment Required status code
func (p *ProductWithWriteHook) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
	if r.Header.Get("X-Payment-Token") == "" {
		return &odata.HookError{
			StatusCode: http.StatusPaymentRequired,
			Message:    "Payment token is required to create products",
		}
	}
	return nil
}

// ODataBeforeUpdate returns a 409 Conflict status code
func (p *ProductWithWriteHook) ODataBeforeUpdate(ctx context.Context, r *http.Request) error {
	if r.Header.Get("X-Version") == "outdated" {
		return &odata.HookError{
			StatusCode: http.StatusConflict,
			Message:    "Resource has been modified by another user",
		}
	}
	return nil
}

// ODataBeforeDelete returns a 423 Locked status code
func (p *ProductWithWriteHook) ODataBeforeDelete(ctx context.Context, r *http.Request) error {
	if p.Price > 1000 {
		return &odata.HookError{
			StatusCode: http.StatusLocked,
			Message:    "Cannot delete high-value products",
		}
	}
	return nil
}

func TestHookError_WriteHooks(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(&ProductWithWriteHook{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&ProductWithWriteHook{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	t.Run("BeforeCreate returns 402 Payment Required", func(t *testing.T) {
		body := strings.NewReader(`{"name": "Test Product", "price": 99.99}`)
		req := httptest.NewRequest(http.MethodPost, "/ProductWithWriteHooks", body)
		req.Header.Set("Content-Type", "application/json")
		// No payment token - should trigger 402
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusPaymentRequired {
			bodyBytes, _ := io.ReadAll(w.Body)
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusPaymentRequired, w.Code, string(bodyBytes))
		} else {
			bodyBytes, _ := io.ReadAll(w.Body)
			var errorResp map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &errorResp); err != nil {
				t.Fatalf("Failed to parse error response: %v", err)
			}

			errorObj := errorResp["error"].(map[string]interface{})
			if errorObj["message"] != "Payment token is required to create products" {
				t.Errorf("Expected message 'Payment token is required to create products', got %s", errorObj["message"])
			}
		}
	})

	t.Run("BeforeCreate allows creation with payment token", func(t *testing.T) {
		body := strings.NewReader(`{"ID": 1, "name": "Test Product", "price": 99.99}`)
		req := httptest.NewRequest(http.MethodPost, "/ProductWithWriteHooks", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Payment-Token", "valid-token")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			bodyBytes, _ := io.ReadAll(w.Body)
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, string(bodyBytes))
		}
	})

	t.Run("BeforeUpdate returns 409 Conflict", func(t *testing.T) {
		body := strings.NewReader(`{"name": "Updated Product"}`)
		req := httptest.NewRequest(http.MethodPatch, "/ProductWithWriteHooks(1)", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Version", "outdated")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			bodyBytes, _ := io.ReadAll(w.Body)
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusConflict, w.Code, string(bodyBytes))
		} else {
			bodyBytes, _ := io.ReadAll(w.Body)
			var errorResp map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &errorResp); err != nil {
				t.Fatalf("Failed to parse error response: %v", err)
			}

			errorObj := errorResp["error"].(map[string]interface{})
			if errorObj["message"] != "Resource has been modified by another user" {
				t.Errorf("Expected message 'Resource has been modified by another user', got %s", errorObj["message"])
			}
		}
	})

	t.Run("BeforeUpdate allows update without outdated header", func(t *testing.T) {
		body := strings.NewReader(`{"name": "Updated Product"}`)
		req := httptest.NewRequest(http.MethodPatch, "/ProductWithWriteHooks(1)", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			bodyBytes, _ := io.ReadAll(w.Body)
			t.Errorf("Expected status %d or %d, got %d. Body: %s", http.StatusOK, http.StatusNoContent, w.Code, string(bodyBytes))
		}
	})

	// Insert a high-value product for delete test
	highValueProduct := &ProductWithWriteHook{ID: 2, Name: "Expensive Product", Price: 1500.00}
	if err := db.Create(highValueProduct).Error; err != nil {
		t.Fatalf("Failed to create high-value product: %v", err)
	}

	t.Run("BeforeDelete returns 423 Locked for high-value products", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/ProductWithWriteHooks(2)", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusLocked {
			bodyBytes, _ := io.ReadAll(w.Body)
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusLocked, w.Code, string(bodyBytes))
		} else {
			bodyBytes, _ := io.ReadAll(w.Body)
			var errorResp map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &errorResp); err != nil {
				t.Fatalf("Failed to parse error response: %v", err)
			}

			errorObj := errorResp["error"].(map[string]interface{})
			if errorObj["message"] != "Cannot delete high-value products" {
				t.Errorf("Expected message 'Cannot delete high-value products', got %s", errorObj["message"])
			}
		}
	})

	t.Run("BeforeDelete allows deletion of low-value products", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/ProductWithWriteHooks(1)", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			bodyBytes, _ := io.ReadAll(w.Body)
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNoContent, w.Code, string(bodyBytes))
		}
	})
}

func TestHookError_WrappedError(t *testing.T) {
	baseErr := errors.New("database connection failed")

	// Test HookError with wrapped error
	hookErr := &odata.HookError{
		StatusCode: http.StatusServiceUnavailable,
		Message:    "Service temporarily unavailable",
		Err:        baseErr,
	}

	// Test Error() method includes wrapped error
	expectedMsg := "Service temporarily unavailable: database connection failed"
	if hookErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, hookErr.Error())
	}

	// Test Unwrap() method
	unwrapped := errors.Unwrap(hookErr)
	if unwrapped != baseErr {
		t.Errorf("Expected unwrapped error to be baseErr, got %v", unwrapped)
	}

	// Test errors.Is() works with wrapped errors
	if !errors.Is(hookErr, baseErr) {
		t.Error("errors.Is should return true for wrapped base error")
	}

	// Test with nested wrapping
	wrappedErr := fmt.Errorf("additional context: %w", baseErr)
	hookErr2 := &odata.HookError{
		StatusCode: http.StatusBadGateway,
		Message:    "Gateway error",
		Err:        wrappedErr,
	}

	if !errors.Is(hookErr2, baseErr) {
		t.Error("errors.Is should work with nested wrapped errors")
	}

	// Test HookError without wrapped error
	simpleErr := &odata.HookError{
		StatusCode: http.StatusNotFound,
		Message:    "Resource not found",
	}

	if errors.Unwrap(simpleErr) != nil {
		t.Error("Unwrap should return nil for HookError without wrapped error")
	}

	// Test HookError with empty message uses wrapped error message
	emptyMsgErr := &odata.HookError{
		StatusCode: http.StatusInternalServerError,
		Err:        baseErr,
	}

	if emptyMsgErr.Error() != baseErr.Error() {
		t.Errorf("Expected error message '%s', got '%s'", baseErr.Error(), emptyMsgErr.Error())
	}
}
