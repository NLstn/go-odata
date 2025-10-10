package odata

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test entity for count integration tests
type CountTestProduct struct {
	ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string  `json:"Name"`
	Price    float64 `json:"Price"`
	Category string  `json:"Category"`
}

func TestIntegrationCountEndpoint(t *testing.T) {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&CountTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	products := []CountTestProduct{
		{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics"},
		{ID: 3, Name: "Keyboard", Price: 149.99, Category: "Electronics"},
		{ID: 4, Name: "Chair", Price: 249.99, Category: "Furniture"},
		{ID: 5, Name: "Desk", Price: 399.99, Category: "Furniture"},
	}
	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	// Initialize OData service
	service := NewService(db)
	service.RegisterEntity(&CountTestProduct{})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
		expectedType   string
	}{
		{
			name:           "Basic count",
			path:           "/CountTestProducts/$count",
			expectedStatus: http.StatusOK,
			expectedBody:   "5",
			expectedType:   "text/plain",
		},
		{
			name:           "Count with filter - Electronics",
			path:           "/CountTestProducts/$count?$filter=Category%20eq%20%27Electronics%27",
			expectedStatus: http.StatusOK,
			expectedBody:   "3",
			expectedType:   "text/plain",
		},
		{
			name:           "Count with filter - Furniture",
			path:           "/CountTestProducts/$count?$filter=Category%20eq%20%27Furniture%27",
			expectedStatus: http.StatusOK,
			expectedBody:   "2",
			expectedType:   "text/plain",
		},
		{
			name:           "Count with filter - Price gt 100",
			path:           "/CountTestProducts/$count?$filter=Price%20gt%20100",
			expectedStatus: http.StatusOK,
			expectedBody:   "4",
			expectedType:   "text/plain",
		},
		{
			name:           "Count with filter - No matches",
			path:           "/CountTestProducts/$count?$filter=Category%20eq%20%27Books%27",
			expectedStatus: http.StatusOK,
			expectedBody:   "0",
			expectedType:   "text/plain",
		},
		{
			name:           "Count with contains filter",
			path:           "/CountTestProducts/$count?$filter=contains(Name,%27Laptop%27)",
			expectedStatus: http.StatusOK,
			expectedBody:   "1",
			expectedType:   "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.expectedStatus)
				t.Logf("Response body: %s", w.Body.String())
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("Content-Type = %v, want %v", contentType, tt.expectedType)
			}

			body, _ := io.ReadAll(w.Body)
			if string(body) != tt.expectedBody {
				t.Errorf("Body = %q, want %q", string(body), tt.expectedBody)
			}

			// Verify OData-Version header
			odataVersion := w.Header().Get("OData-Version")
			if odataVersion != "4.0" {
				t.Errorf("OData-Version = %v, want %v", odataVersion, "4.0")
			}
		})
	}
}

func TestIntegrationCountEndpointVerifyCollectionStillWorks(t *testing.T) {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&CountTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	products := []CountTestProduct{
		{ID: 1, Name: "Test1", Price: 10.0, Category: "Cat1"},
		{ID: 2, Name: "Test2", Price: 20.0, Category: "Cat2"},
	}
	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	// Initialize OData service
	service := NewService(db)
	service.RegisterEntity(&CountTestProduct{})

	// Test that regular collection endpoint still works
	req := httptest.NewRequest(http.MethodGet, "/CountTestProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}

	// Verify it's JSON, not plain text
	body := w.Body.String()
	if len(body) < 10 || body[0] != '{' {
		t.Errorf("Expected JSON response, got: %s", body)
	}
}
