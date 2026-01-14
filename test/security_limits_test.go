package odata_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestEntity for security limits testing
type TestSecurityEntity struct {
	ID       int    `gorm:"primaryKey" json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

func TestMaxInClauseSize(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestSecurityEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create service with custom MaxInClauseSize
	service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{
		MaxInClauseSize: 5, // Set a small limit for testing
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register entity
	if err := service.RegisterEntity(&TestSecurityEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test case 1: IN clause within limit should succeed
	t.Run("WithinLimit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/TestSecurityEntities?$filter=ID%20in%20(1,2,3,4,5)", nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// Test case 2: IN clause exceeding limit should fail
	t.Run("ExceedsLimit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/TestSecurityEntities?$filter=ID%20in%20(1,2,3,4,5,6,7)", nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify error message mentions IN clause size
		body := w.Body.String()
		if !strings.Contains(body, "IN clause") {
			t.Errorf("Expected error message to mention 'IN clause', got: %s", body)
		}
	})

	// Test case 3: Empty IN clause should still work
	t.Run("EmptyIN", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/TestSecurityEntities?$filter=ID%20in%20()", nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		// Empty IN clause is valid (returns no results)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

func TestMaxExpandDepth(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Define entities with relationships
	type Category struct {
		ID            int    `gorm:"primaryKey" json:"id"`
		Name          string `json:"name"`
		ParentID      *int
		Parent        *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
		Subcategories []Category `gorm:"foreignKey:ParentID" json:"subcategories,omitempty"`
	}

	if err := db.AutoMigrate(&Category{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create service with custom MaxExpandDepth
	service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{
		MaxExpandDepth: 3, // Set a limit for testing
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register entity
	if err := service.RegisterEntity(&Category{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test case 1: Expand within depth limit should succeed
	t.Run("WithinDepthLimit", func(t *testing.T) {
		// 1 level of nesting (depth counter 0)
		req := httptest.NewRequest("GET", "/Categories?$expand=Parent", nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for 1 level of nesting, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("NestedExpandWithinLimit", func(t *testing.T) {
		// 2 levels of nesting (depth counter 1): Categories -> Parent -> Parent
		req := httptest.NewRequest("GET", "/Categories?$expand=Parent($expand=Parent)", nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for 2 levels of nesting, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// Test case 2: Expand exceeding depth limit should fail
	t.Run("ExceedsDepthLimit", func(t *testing.T) {
		// 4 levels of nesting (depth counter 3): Categories -> Parent -> Parent -> Parent -> Parent (exceeds limit of 3)
		req := httptest.NewRequest("GET", "/Categories?$expand=Parent($expand=Parent($expand=Parent($expand=Parent)))", nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for 4 levels of nesting, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify error message mentions expand depth
		body := w.Body.String()
		if !strings.Contains(body, "depth") && !strings.Contains(body, "$expand") {
			t.Errorf("Expected error message to mention expand depth, got: %s", body)
		}
	})

	// Test case 3: Multiple expands at same level should work
	t.Run("MultipleExpandsSameLevel", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/Categories?$expand=Parent,Subcategories", nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for multiple expands at same level, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

func TestDefaultLimits(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestSecurityEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create service with default config (should use default limits)
	service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register entity
	if err := service.RegisterEntity(&TestSecurityEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test case: Large IN clause should be rejected with default limit (1000)
	t.Run("DefaultMaxInClauseSize", func(t *testing.T) {
		// Build an IN clause with 1001 values (exceeds default of 1000)
		values := make([]string, 1001)
		for i := range values {
			values[i] = fmt.Sprintf("%d", i+1)
		}
		filterValue := fmt.Sprintf("ID in (%s)", strings.Join(values, ","))

		// Use url.Values to properly encode the query parameter
		params := url.Values{}
		params.Set("$filter", filterValue)
		requestURL := "/TestSecurityEntities?" + params.Encode()

		req := httptest.NewRequest("GET", requestURL, nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for exceeding default IN clause limit, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "IN clause") || !strings.Contains(body, "1000") {
			t.Errorf("Expected error message to mention IN clause and limit 1000, got: %s", body)
		}
	})
}

func TestConfigurableLimits(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestSecurityEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Test case: Custom high limit
	t.Run("CustomHighLimit", func(t *testing.T) {
		service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{
			MaxInClauseSize: 2000, // Custom high limit
		})
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}

		if err := service.RegisterEntity(&TestSecurityEntity{}); err != nil {
			t.Fatalf("Failed to register entity: %v", err)
		}

		// 1500 values should succeed with limit of 2000
		values := make([]string, 1500)
		for i := range values {
			values[i] = fmt.Sprintf("%d", i+1)
		}
		filterValue := fmt.Sprintf("ID in (%s)", strings.Join(values, ","))

		params := url.Values{}
		params.Set("$filter", filterValue)
		requestURL := "/TestSecurityEntities?" + params.Encode()

		req := httptest.NewRequest("GET", requestURL, nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 with custom high limit, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// Test case: Negative values should use defaults
	t.Run("NegativeUsesDefault", func(t *testing.T) {
		service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{
			MaxInClauseSize: -1, // Invalid, should use default
		})
		if err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}

		if err := service.RegisterEntity(&TestSecurityEntity{}); err != nil {
			t.Fatalf("Failed to register entity: %v", err)
		}

		// With negative config, should use default of 1000
		// So 1001 values should fail
		values := make([]string, 1001)
		for i := range values {
			values[i] = fmt.Sprintf("%d", i+1)
		}
		filterValue := fmt.Sprintf("ID in (%s)", strings.Join(values, ","))

		params := url.Values{}
		params.Set("$filter", filterValue)
		requestURL := "/TestSecurityEntities?" + params.Encode()

		req := httptest.NewRequest("GET", requestURL, nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		// Should fail with default limit
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 when negative config uses default, got %d", w.Code)
		}
	})
}
