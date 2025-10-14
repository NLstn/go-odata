package odata_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"

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
	service := odata.NewService(db)
	_ = service.RegisterEntity(&CountTestProduct{})

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
	service := odata.NewService(db)
	_ = service.RegisterEntity(&CountTestProduct{})

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

// Test that $count endpoint works correctly with OData v4 specification
func TestIntegrationCountEndpointODataV4Compliance(t *testing.T) {
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
	service := odata.NewService(db)
	_ = service.RegisterEntity(&CountTestProduct{})

	tests := []struct {
		name         string
		path         string
		expectedBody string
		validateFunc func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:         "Returns plain text, not JSON",
			path:         "/CountTestProducts/$count",
			expectedBody: "5",
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				// Should not be JSON
				if len(body) > 10 || body[0] == '{' || body[0] == '[' {
					t.Errorf("Response should be plain text, got: %s", body)
				}
			},
		},
		{
			name:         "Ignores $top parameter",
			path:         "/CountTestProducts/$count?$top=2",
			expectedBody: "5",
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should return total count, not limited by $top
			},
		},
		{
			name:         "Ignores $skip parameter",
			path:         "/CountTestProducts/$count?$skip=3",
			expectedBody: "5",
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should return total count, not affected by $skip
			},
		},
		{
			name:         "Applies $filter parameter",
			path:         "/CountTestProducts/$count?$filter=Category%20eq%20%27Electronics%27",
			expectedBody: "3",
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should only count filtered items
			},
		},
		{
			name:         "Returns count of zero for empty result",
			path:         "/CountTestProducts/$count?$filter=Category%20eq%20%27Books%27",
			expectedBody: "0",
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should return 0, not error
			},
		},
		{
			name:         "Has correct Content-Type header",
			path:         "/CountTestProducts/$count",
			expectedBody: "5",
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				contentType := w.Header().Get("Content-Type")
				if contentType != "text/plain" {
					t.Errorf("Content-Type = %v, want text/plain", contentType)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
				t.Logf("Response body: %s", w.Body.String())
			}

			body, _ := io.ReadAll(w.Body)
			if string(body) != tt.expectedBody {
				t.Errorf("Body = %q, want %q", string(body), tt.expectedBody)
			}

			// Run additional validation if provided
			if tt.validateFunc != nil {
				// Need to create a new recorder with the same response
				w2 := httptest.NewRecorder()
				w2.Code = w.Code
				w2.Body.WriteString(string(body))
				for k, v := range w.Header() {
					w2.Header()[k] = v
				}
				tt.validateFunc(t, w2)
			}
		})
	}
}
