package odata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TestProduct struct {
	ID    int     `json:"id" gorm:"primarykey" odata:"key"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := NewService(db)
	if err := service.RegisterEntity(TestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestServeHTTPServiceDocument(t *testing.T) {
	service, _ := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["@odata.context"]; !ok {
		t.Error("Response missing @odata.context")
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) == 0 {
		t.Error("Service document value array is empty")
	}
}

func TestServeHTTPMetadata(t *testing.T) {
	service, _ := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/xml" {
		t.Errorf("Content-Type = %v, want application/xml", contentType)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Metadata response body is empty")
	}
}

func TestServeHTTPCollection(t *testing.T) {
	service, db := setupTestService(t)

	// Insert test data
	testProducts := []TestProduct{
		{ID: 1, Name: "Product 1", Price: 10.99},
		{ID: 2, Name: "Product 2", Price: 20.99},
	}
	for _, product := range testProducts {
		db.Create(&product)
	}

	req := httptest.NewRequest(http.MethodGet, "/TestProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 2 {
		t.Errorf("len(value) = %v, want 2", len(value))
	}
}

func TestServeHTTPEntity(t *testing.T) {
	service, db := setupTestService(t)

	// Insert test data
	product := TestProduct{ID: 1, Name: "Test Product", Price: 99.99}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProducts(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["name"] != "Test Product" {
		t.Errorf("name = %v, want Test Product", response["name"])
	}

	if response["id"] != float64(1) {
		t.Errorf("id = %v, want 1", response["id"])
	}
}

func TestServeHTTPEntityNotFound(t *testing.T) {
	service, _ := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/TestProducts(999)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

func TestServeHTTPInvalidEntitySet(t *testing.T) {
	service, _ := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/InvalidEntitySet", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestServeHTTPInvalidURL(t *testing.T) {
	service, _ := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/Products(", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// The URL parser doesn't fail on this, so it will be treated as entity set "Products("
	// which doesn't exist, resulting in 404 Not Found
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestServeHTTPMethodNotAllowed(t *testing.T) {
	service, _ := setupTestService(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"POST to service document", http.MethodPost, "/"},
		{"POST to metadata", http.MethodPost, "/$metadata"},
		{"DELETE to collection", http.MethodDelete, "/TestProducts"},
		// PUT to entity is now supported, so removed from this test
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}
