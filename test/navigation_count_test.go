package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NavCountTestProduct for navigation count testing
type NavCountTestProduct struct {
	ID           uint                      `json:"ID" gorm:"primaryKey" odata:"key"`
	Name         string                    `json:"Name" odata:"required"`
	Price        float64                   `json:"Price"`
	Descriptions []NavCountTestDescription `json:"Descriptions" gorm:"foreignKey:ProductID;references:ID"`
}

// NavCountTestDescription for navigation count testing
type NavCountTestDescription struct {
	ProductID   uint                 `json:"ProductID" gorm:"primaryKey" odata:"key"`
	LanguageKey string               `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key"`
	Description string               `json:"Description" odata:"required"`
	Product     *NavCountTestProduct `json:"Product,omitempty" gorm:"foreignKey:ProductID;references:ID"`
}

func setupNavigationCountTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NavCountTestProduct{}, &NavCountTestDescription{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	products := []NavCountTestProduct{
		{ID: 1, Name: "Laptop", Price: 999.99},
		{ID: 2, Name: "Mouse", Price: 29.99},
		{ID: 3, Name: "Keyboard", Price: 79.99},
	}
	db.Create(&products)

	descriptions := []NavCountTestDescription{
		{ProductID: 1, LanguageKey: "EN", Description: "Laptop in English"},
		{ProductID: 1, LanguageKey: "DE", Description: "Laptop in German"},
		{ProductID: 1, LanguageKey: "FR", Description: "Laptop in French"},
		{ProductID: 2, LanguageKey: "EN", Description: "Mouse in English"},
		{ProductID: 2, LanguageKey: "DE", Description: "Mouse in German"},
		// Product 3 has no descriptions
	}
	db.Create(&descriptions)

	service := odata.NewService(db)
	if err := service.RegisterEntity(&NavCountTestProduct{}); err != nil {
		t.Fatalf("Failed to register NavCountTestProduct: %v", err)
	}
	if err := service.RegisterEntity(&NavCountTestDescription{}); err != nil {
		t.Fatalf("Failed to register NavCountTestDescription: %v", err)
	}

	return service, db
}

// TestNavigationPropertyCountIntegration tests the full integration of navigation property count
func TestNavigationPropertyCountIntegration(t *testing.T) {
	service, _ := setupNavigationCountTestService(t)

	tests := []struct {
		name          string
		url           string
		expectedCode  int
		expectedCount string
		expectError   bool
	}{
		{
			name:          "Product 1 descriptions count",
			url:           "/NavCountTestProducts(1)/Descriptions/$count",
			expectedCode:  http.StatusOK,
			expectedCount: "3",
		},
		{
			name:          "Product 2 descriptions count",
			url:           "/NavCountTestProducts(2)/Descriptions/$count",
			expectedCode:  http.StatusOK,
			expectedCount: "2",
		},
		{
			name:          "Product 3 descriptions count (zero)",
			url:           "/NavCountTestProducts(3)/Descriptions/$count",
			expectedCode:  http.StatusOK,
			expectedCount: "0",
		},
		{
			name:         "Non-existent product",
			url:          "/NavCountTestProducts(999)/Descriptions/$count",
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		{
			name:         "Invalid navigation property",
			url:          "/NavCountTestProducts(1)/InvalidProperty/$count",
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedCode, w.Code, w.Body.String())
			}

			if tt.expectError {
				// Check for error response
				var errorResp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
					t.Fatalf("Failed to parse error response: %v", err)
				}
				if _, ok := errorResp["error"]; !ok {
					t.Error("Expected error field in response")
				}
			} else {
				// Check for count value
				count := w.Body.String()
				if count != tt.expectedCount {
					t.Errorf("Expected count %s, got %s", tt.expectedCount, count)
				}

				// Verify Content-Type
				contentType := w.Header().Get("Content-Type")
				if contentType != "text/plain" {
					t.Errorf("Expected Content-Type text/plain, got %s", contentType)
				}
			}
		})
	}
}

// TestNavigationPropertyCountHEADIntegration tests HEAD requests
func TestNavigationPropertyCountHEADIntegration(t *testing.T) {
	service, _ := setupNavigationCountTestService(t)

	req := httptest.NewRequest(http.MethodHead, "/NavCountTestProducts(1)/Descriptions/$count", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// HEAD should not return body
	if w.Body.Len() > 0 {
		t.Errorf("Expected empty body for HEAD request, got %d bytes", w.Body.Len())
	}

	// But should have correct Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}
}

// TestNavigationPropertyCountOPTIONSIntegration tests OPTIONS requests
func TestNavigationPropertyCountOPTIONSIntegration(t *testing.T) {
	service, _ := setupNavigationCountTestService(t)

	req := httptest.NewRequest(http.MethodOptions, "/NavCountTestProducts(1)/Descriptions/$count", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check Allow header
	allow := w.Header().Get("Allow")
	if allow != "GET, HEAD, OPTIONS" {
		t.Errorf("Expected Allow header 'GET, HEAD, OPTIONS', got %s", allow)
	}
}

// TestNavigationPropertyCountInvalidMethod tests unsupported HTTP methods
func TestNavigationPropertyCountInvalidMethod(t *testing.T) {
	service, _ := setupNavigationCountTestService(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/NavCountTestProducts(1)/Descriptions/$count", nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestNavigationPropertyCountComparison compares count with actual collection size
func TestNavigationPropertyCountComparison(t *testing.T) {
	service, _ := setupNavigationCountTestService(t)

	// First get the navigation property collection
	req := httptest.NewRequest(http.MethodGet, "/NavCountTestProducts(1)/Descriptions", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	var collectionResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &collectionResp); err != nil {
		t.Fatalf("Failed to parse collection response: %v", err)
	}

	value, ok := collectionResp["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}
	actualCount := len(value)

	// Now get the count
	req = httptest.NewRequest(http.MethodGet, "/NavCountTestProducts(1)/Descriptions/$count", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	count := w.Body.String()
	if count != "3" {
		t.Errorf("Expected count 3, got %s", count)
	}

	// Verify count matches actual collection size
	if actualCount != 3 {
		t.Errorf("Expected 3 items in collection, got %d", actualCount)
	}
}
