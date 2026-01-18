package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestODataMetadataLevelIntegration tests the odata.metadata parameter support end-to-end
func TestODataMetadataLevelIntegration(t *testing.T) {
	// Setup test database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	type Product struct {
		ID    int     `json:"id" gorm:"primaryKey" odata:"key"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	db.Create(&Product{ID: 1, Name: "Laptop", Price: 999.99})
	db.Create(&Product{ID: 2, Name: "Mouse", Price: 29.99})

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&Product{})

	tests := []struct {
		name         string
		acceptHeader string
		formatParam  string
		expectedCT   string
		description  string
	}{
		{
			name:        "Default (no parameters) should use minimal",
			expectedCT:  "application/json;odata.metadata=minimal",
			description: "Default behavior should be minimal metadata",
		},
		{
			name:         "Accept header with odata.metadata=full",
			acceptHeader: "application/json;odata.metadata=full",
			expectedCT:   "application/json;odata.metadata=full",
			description:  "Should respect full metadata from Accept header",
		},
		{
			name:         "Accept header with odata.metadata=none",
			acceptHeader: "application/json;odata.metadata=none",
			expectedCT:   "application/json;odata.metadata=none",
			description:  "Should respect none metadata from Accept header",
		},
		{
			name:         "Accept header with odata.metadata=minimal",
			acceptHeader: "application/json;odata.metadata=minimal",
			expectedCT:   "application/json;odata.metadata=minimal",
			description:  "Should respect minimal metadata from Accept header",
		},
		{
			name:        "$format parameter with odata.metadata=full",
			formatParam: "application/json;odata.metadata=full",
			expectedCT:  "application/json;odata.metadata=full",
			description: "Should respect full metadata from $format parameter",
		},
		{
			name:        "$format parameter with odata.metadata=none",
			formatParam: "application/json;odata.metadata=none",
			expectedCT:  "application/json;odata.metadata=none",
			description: "Should respect none metadata from $format parameter",
		},
		{
			name:         "$format overrides Accept header",
			acceptHeader: "application/json;odata.metadata=full",
			formatParam:  "application/json;odata.metadata=none",
			expectedCT:   "application/json;odata.metadata=none",
			description:  "$format should take precedence over Accept",
		},
		{
			name:        "$format shorthand with metadata",
			formatParam: "json;odata.metadata=full",
			expectedCT:  "application/json;odata.metadata=full",
			description: "Should handle 'json' shorthand with metadata parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with collection endpoint
			t.Run("Collection", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/Products", nil)
				if tt.formatParam != "" {
					q := req.URL.Query()
					q.Set("$format", tt.formatParam)
					req.URL.RawQuery = q.Encode()
				}
				if tt.acceptHeader != "" {
					req.Header.Set("Accept", tt.acceptHeader)
				}
				w := httptest.NewRecorder()

				service.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", w.Code)
				}

				contentType := w.Header().Get("Content-Type")
				if contentType != tt.expectedCT {
					t.Errorf("Content-Type = %v, want %v (test: %s)",
						contentType, tt.expectedCT, tt.description)
				}

				// Verify response is valid JSON
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to parse JSON response: %v", err)
				}

				// Check @odata.context presence based on metadata level
				hasContext := false
				if _, ok := response["@odata.context"]; ok {
					hasContext = true
				}

				if strings.Contains(tt.expectedCT, "none") {
					// For metadata=none, @odata.context should NOT be present
					if hasContext {
						t.Error("@odata.context should NOT be present for metadata=none")
					}
				} else {
					// For minimal and full metadata, @odata.context should be present
					if !hasContext {
						t.Error("Expected @odata.context in response for minimal/full metadata")
					}
				}
			})

			// Test with individual entity endpoint
			t.Run("Individual Entity", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/Products(1)", nil)
				if tt.formatParam != "" {
					q := req.URL.Query()
					q.Set("$format", tt.formatParam)
					req.URL.RawQuery = q.Encode()
				}
				if tt.acceptHeader != "" {
					req.Header.Set("Accept", tt.acceptHeader)
				}
				w := httptest.NewRecorder()

				service.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", w.Code)
				}

				contentType := w.Header().Get("Content-Type")
				if contentType != tt.expectedCT {
					t.Errorf("Content-Type = %v, want %v (test: %s)",
						contentType, tt.expectedCT, tt.description)
				}
			})

			// Test with service document
			t.Run("Service Document", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				if tt.formatParam != "" {
					q := req.URL.Query()
					q.Set("$format", tt.formatParam)
					req.URL.RawQuery = q.Encode()
				}
				if tt.acceptHeader != "" {
					req.Header.Set("Accept", tt.acceptHeader)
				}
				w := httptest.NewRecorder()

				service.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", w.Code)
				}

				contentType := w.Header().Get("Content-Type")
				if contentType != tt.expectedCT {
					t.Errorf("Content-Type = %v, want %v (test: %s)",
						contentType, tt.expectedCT, tt.description)
				}
			})
		})
	}
}

// TestODataTypeAnnotation tests that @odata.type field is included when full metadata is requested
func TestODataTypeAnnotation(t *testing.T) {
	// Setup test database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	type Product struct {
		ID    int     `json:"id" gorm:"primaryKey" odata:"key"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	db.Create(&Product{ID: 1, Name: "Laptop", Price: 999.99})
	db.Create(&Product{ID: 2, Name: "Mouse", Price: 29.99})

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&Product{})

	tests := []struct {
		name                string
		url                 string
		acceptHeader        string
		formatParam         string
		shouldHaveODataType bool
		description         string
	}{
		{
			name:                "Full metadata via Accept header - collection",
			url:                 "/Products",
			acceptHeader:        "application/json;odata.metadata=full",
			shouldHaveODataType: true,
			description:         "Should include @odata.type in collection with full metadata",
		},
		{
			name:                "Full metadata via $format - collection",
			url:                 "/Products",
			formatParam:         "application/json;odata.metadata=full",
			shouldHaveODataType: true,
			description:         "Should include @odata.type in collection with $format=full",
		},
		{
			name:                "Full metadata via Accept header - single entity",
			url:                 "/Products(1)",
			acceptHeader:        "application/json;odata.metadata=full",
			shouldHaveODataType: true,
			description:         "Should include @odata.type in single entity with full metadata",
		},
		{
			name:                "Full metadata via $format - single entity",
			url:                 "/Products(1)",
			formatParam:         "application/json;odata.metadata=full",
			shouldHaveODataType: true,
			description:         "Should include @odata.type in single entity with $format=full",
		},
		{
			name:                "Minimal metadata - collection",
			url:                 "/Products",
			acceptHeader:        "application/json;odata.metadata=minimal",
			shouldHaveODataType: false,
			description:         "Should NOT include @odata.type with minimal metadata",
		},
		{
			name:                "Minimal metadata - single entity",
			url:                 "/Products(1)",
			acceptHeader:        "application/json;odata.metadata=minimal",
			shouldHaveODataType: false,
			description:         "Should NOT include @odata.type with minimal metadata",
		},
		{
			name:                "None metadata - collection",
			url:                 "/Products",
			acceptHeader:        "application/json;odata.metadata=none",
			shouldHaveODataType: false,
			description:         "Should NOT include @odata.type with none metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.formatParam != "" {
				q := req.URL.Query()
				q.Set("$format", tt.formatParam)
				req.URL.RawQuery = q.Encode()
			}
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			if strings.Contains(tt.url, "Products(") {
				// Single entity response - check root level
				_, hasODataType := response["@odata.type"]
				if tt.shouldHaveODataType && !hasODataType {
					t.Errorf("Expected @odata.type in response (test: %s)", tt.description)
				}
				if !tt.shouldHaveODataType && hasODataType {
					t.Errorf("Did not expect @odata.type in response (test: %s)", tt.description)
				}
				if hasODataType {
					// Verify the format is correct
					odataType := response["@odata.type"].(string)
					if odataType != "#ODataService.Product" {
						t.Errorf("Expected @odata.type=#ODataService.Product, got %s", odataType)
					}
				}
			} else {
				// Collection response - check in value array
				value, ok := response["value"].([]interface{})
				if !ok || len(value) == 0 {
					t.Fatal("Expected value array in response")
				}

				firstEntity, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("Expected first entity to be a map")
				}

				_, hasODataType := firstEntity["@odata.type"]
				if tt.shouldHaveODataType && !hasODataType {
					t.Errorf("Expected @odata.type in entity (test: %s)", tt.description)
				}
				if !tt.shouldHaveODataType && hasODataType {
					t.Errorf("Did not expect @odata.type in entity (test: %s)", tt.description)
				}
				if hasODataType {
					// Verify the format is correct
					odataType := firstEntity["@odata.type"].(string)
					if odataType != "#ODataService.Product" {
						t.Errorf("Expected @odata.type=#ODataService.Product, got %s", odataType)
					}
				}
			}
		})
	}
}

// TestODataMetadataWithQueryOptions tests metadata level with other query options
func TestODataMetadataWithQueryOptions(t *testing.T) {
	// Setup test database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	type Product struct {
		ID    int     `json:"id" gorm:"primaryKey" odata:"key"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	db.Create(&Product{ID: 1, Name: "Laptop", Price: 999.99})
	db.Create(&Product{ID: 2, Name: "Mouse", Price: 29.99})
	db.Create(&Product{ID: 3, Name: "Keyboard", Price: 79.99})

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&Product{})

	tests := []struct {
		name        string
		path        string
		params      map[string]string
		expectedCT  string
		description string
	}{
		{
			name: "Metadata with $filter",
			path: "/Products",
			params: map[string]string{
				"$filter": "Price gt 50",
				"$format": "application/json;odata.metadata=full",
			},
			expectedCT:  "application/json;odata.metadata=full",
			description: "Should work with $filter query option",
		},
		{
			name: "Metadata with $top and $skip",
			path: "/Products",
			params: map[string]string{
				"$top":    "2",
				"$skip":   "1",
				"$format": "application/json;odata.metadata=none",
			},
			expectedCT:  "application/json;odata.metadata=none",
			description: "Should work with pagination options",
		},
		{
			name: "Metadata with $orderby",
			path: "/Products",
			params: map[string]string{
				"$orderby": "Price desc",
				"$format":  "application/json;odata.metadata=full",
			},
			expectedCT:  "application/json;odata.metadata=full",
			description: "Should work with $orderby option",
		},
		{
			name: "Metadata with $select",
			path: "/Products",
			params: map[string]string{
				"$select": "name,price",
				"$format": "application/json;odata.metadata=minimal",
			},
			expectedCT:  "application/json;odata.metadata=minimal",
			description: "Should work with $select option",
		},
		{
			name: "Metadata with $count",
			path: "/Products",
			params: map[string]string{
				"$count":  "true",
				"$format": "application/json;odata.metadata=full",
			},
			expectedCT:  "application/json;odata.metadata=full",
			description: "Should work with $count option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			// Set query parameters properly
			q := req.URL.Query()
			for key, value := range tt.params {
				q.Set(key, value)
			}
			req.URL.RawQuery = q.Encode()

			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedCT {
				t.Errorf("Content-Type = %v, want %v (test: %s)",
					contentType, tt.expectedCT, tt.description)
			}

			// Verify response is valid JSON
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to parse JSON response: %v", err)
			}
		})
	}
}
