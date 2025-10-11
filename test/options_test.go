package odata_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestOptionsProduct is a simple test entity for OPTIONS tests
type TestOptionsProduct struct {
	ID    int     `json:"id" gorm:"primarykey" odata:"key"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func setupOptionsTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestOptionsProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(TestOptionsProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestOptionsServiceDocument tests OPTIONS request on the service document endpoint
func TestOptionsServiceDocument(t *testing.T) {
	service, _ := setupOptionsTestService(t)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version header = %v, want '4.0'", odataVersion)
	}

	// OPTIONS should return no body
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty, got %v bytes", w.Body.Len())
	}
}

// TestOptionsMetadata tests OPTIONS request on the metadata endpoint
func TestOptionsMetadata(t *testing.T) {
	service, _ := setupOptionsTestService(t)

	req := httptest.NewRequest(http.MethodOptions, "/$metadata", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version header = %v, want '4.0'", odataVersion)
	}

	// OPTIONS should return no body
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty, got %v bytes", w.Body.Len())
	}
}

// TestOptionsEntityCollection tests OPTIONS request on an entity collection endpoint
func TestOptionsEntityCollection(t *testing.T) {
	service, _ := setupOptionsTestService(t)

	req := httptest.NewRequest(http.MethodOptions, "/TestOptionsProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, POST, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, POST, OPTIONS'", allowHeader)
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version header = %v, want '4.0'", odataVersion)
	}

	// OPTIONS should return no body
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty, got %v bytes", w.Body.Len())
	}
}

// TestOptionsEntity tests OPTIONS request on an individual entity endpoint
func TestOptionsEntity(t *testing.T) {
	service, db := setupOptionsTestService(t)

	// Insert a test product
	product := TestOptionsProduct{ID: 1, Name: "Test Product", Price: 99.99}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodOptions, "/TestOptionsProducts(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, DELETE, PATCH, PUT, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, DELETE, PATCH, PUT, OPTIONS'", allowHeader)
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version header = %v, want '4.0'", odataVersion)
	}

	// OPTIONS should return no body
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty, got %v bytes", w.Body.Len())
	}
}

// TestOptionsCount tests OPTIONS request on the $count endpoint
func TestOptionsCount(t *testing.T) {
	service, _ := setupOptionsTestService(t)

	req := httptest.NewRequest(http.MethodOptions, "/TestOptionsProducts/$count", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version header = %v, want '4.0'", odataVersion)
	}

	// OPTIONS should return no body
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty, got %v bytes", w.Body.Len())
	}
}

// TestOptionsInvalidEntitySet tests OPTIONS on a non-existent entity set
func TestOptionsInvalidEntitySet(t *testing.T) {
	service, _ := setupOptionsTestService(t)

	req := httptest.NewRequest(http.MethodOptions, "/NonExistentEntitySet", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 404 Not Found for non-existent entity set
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

// TestOptionsCorsCompatibility tests that OPTIONS works for CORS preflight
func TestOptionsCorsCompatibility(t *testing.T) {
	service, _ := setupOptionsTestService(t)

	tests := []struct {
		name string
		path string
	}{
		{"Service document", "/"},
		{"Metadata", "/$metadata"},
		{"Entity collection", "/TestOptionsProducts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, tt.path, nil)
			// Simulate CORS preflight headers
			req.Header.Set("Origin", "http://example.com")
			req.Header.Set("Access-Control-Request-Method", "GET")

			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			// Should include Allow header
			if w.Header().Get("Allow") == "" {
				t.Error("Missing Allow header")
			}

			// Should include OData-Version header
			if w.Header().Get("OData-Version") != "4.0" {
				t.Errorf("OData-Version = %v, want '4.0'", w.Header().Get("OData-Version"))
			}
		})
	}
}
