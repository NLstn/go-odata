package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// AdvancedProduct represents a product entity for testing advanced filter features
type AdvancedProduct struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name" gorm:"not null"`
	Price       float64 `json:"Price" gorm:"not null"`
	Category    string  `json:"Category" gorm:"not null"`
	IsAvailable bool    `json:"IsAvailable" gorm:"not null"`
	Quantity    int     `json:"Quantity" gorm:"not null"`
}

func setupAdvancedFilterTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the models
	if err := db.AutoMigrate(&AdvancedProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	products := []AdvancedProduct{
		{ID: 1, Name: "Laptop Pro", Price: 1200.00, Category: "Electronics", IsAvailable: true, Quantity: 10},
		{ID: 2, Name: "Laptop Basic", Price: 600.00, Category: "Electronics", IsAvailable: true, Quantity: 25},
		{ID: 3, Name: "Mouse Wireless", Price: 29.99, Category: "Electronics", IsAvailable: true, Quantity: 100},
		{ID: 4, Name: "Book: Go Programming", Price: 45.00, Category: "Books", IsAvailable: true, Quantity: 50},
		{ID: 5, Name: "Book: OData Guide", Price: 35.00, Category: "Books", IsAvailable: false, Quantity: 0},
		{ID: 6, Name: "Luxury Watch", Price: 5000.00, Category: "Luxury", IsAvailable: true, Quantity: 2},
	}

	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to seed products: %v", err)
	}

	return db
}

// TestAdvancedFilterWithParentheses tests filter expressions with parentheses
func TestAdvancedFilterWithParentheses(t *testing.T) {
	db := setupAdvancedFilterTestDB(t)

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&AdvancedProduct{}); err != nil {
		t.Fatalf("Failed to register AdvancedProduct entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "Simple parentheses",
			filter:        "(Price gt 100)",
			expectedCount: 3, // Laptop Pro ($1200), Laptop Basic ($600), Luxury Watch ($5000)
			description:   "Should support simple parentheses",
		},
		{
			name:          "Complex boolean grouping",
			filter:        "(Price gt 100 and Category eq 'Electronics') or (Price lt 50 and Category eq 'Books')",
			expectedCount: 4, // Laptop Pro, Laptop Basic (>100, Electronics) + Go Programming, OData Guide (<50, Books)
			description:   "Should support complex boolean combinations with parentheses",
		},
		{
			name:          "Multiple levels of grouping",
			filter:        "((Price gt 1000 or Price lt 50) and IsAvailable eq true)",
			expectedCount: 4, // Laptop Pro ($1200), Mouse ($29.99), Go Programming ($45), Luxury Watch ($5000) - all available
			description:   "Should support multiple levels of parentheses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/AdvancedProducts?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("Response does not contain value array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d for filter: %s", tt.expectedCount, len(value), tt.filter)
			}
		})
	}
}

// TestAdvancedFilterWithNOT tests filter expressions with NOT operator
func TestAdvancedFilterWithNOT(t *testing.T) {
	db := setupAdvancedFilterTestDB(t)

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&AdvancedProduct{}); err != nil {
		t.Fatalf("Failed to register AdvancedProduct entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "Simple NOT",
			filter:        "not (Category eq 'Books')",
			expectedCount: 4, // All except books
			description:   "Should support NOT operator",
		},
		{
			name:          "NOT with complex expression (basic support)",
			filter:        "not (Category eq 'Luxury')",
			expectedCount: 5, // All except Luxury Watch
			description:   "Should support NOT operator on simple expressions",
		},
		{
			name:          "Multiple NOT operators",
			filter:        "not (Price gt 1000) and not (Category eq 'Books')",
			expectedCount: 2, // Laptop Basic, Mouse (Price <= 1000 AND Category != 'Books')
			description:   "Should support multiple NOT operators",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/AdvancedProducts?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("Response does not contain value array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d for filter: %s", tt.expectedCount, len(value), tt.filter)
				t.Logf("Response: %+v", value)
			}
		})
	}
}

// TestAdvancedFilterWithFunctions tests filter expressions with functions and complex logic
func TestAdvancedFilterWithFunctions(t *testing.T) {
	db := setupAdvancedFilterTestDB(t)

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&AdvancedProduct{}); err != nil {
		t.Fatalf("Failed to register AdvancedProduct entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "Function with boolean logic",
			filter:        "contains(Name,'Laptop') and Price gt 500",
			expectedCount: 2, // Laptop Pro and Laptop Basic (both > 500)
			description:   "Should support functions with AND logic",
		},
		{
			name:          "Function in parentheses with OR",
			filter:        "(contains(Name,'Laptop') or contains(Name,'Mouse')) and Price lt 1000",
			expectedCount: 2, // Laptop Basic and Mouse
			description:   "Should support functions in complex expressions",
		},
		{
			name:          "Function with NOT",
			filter:        "contains(Name,'Book') and not (IsAvailable eq false)",
			expectedCount: 1, // Only Go Programming book (available)
			description:   "Should support functions with NOT operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/AdvancedProducts?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("Response does not contain value array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d for filter: %s", tt.expectedCount, len(value), tt.filter)
			}
		})
	}
}

// TestAdvancedFilterWithLiterals tests filter expressions with different literal types
func TestAdvancedFilterWithLiterals(t *testing.T) {
	db := setupAdvancedFilterTestDB(t)

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&AdvancedProduct{}); err != nil {
		t.Fatalf("Failed to register AdvancedProduct entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "Boolean literal true",
			filter:        "IsAvailable eq true",
			expectedCount: 5, // All available products
			description:   "Should support boolean literal true",
		},
		{
			name:          "Boolean literal false",
			filter:        "IsAvailable eq false",
			expectedCount: 1, // OData Guide book
			description:   "Should support boolean literal false",
		},
		{
			name:          "Numeric literal with decimal",
			filter:        "Price eq 29.99",
			expectedCount: 1, // Mouse
			description:   "Should support numeric literals with decimals",
		},
		{
			name:          "Multiple conditions with different literal types",
			filter:        "IsAvailable eq true and Price gt 100.0 and Category eq 'Electronics'",
			expectedCount: 2, // Laptop Pro and Laptop Basic
			description:   "Should support multiple literal types in one expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/AdvancedProducts?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("Response does not contain value array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d for filter: %s", tt.expectedCount, len(value), tt.filter)
			}
		})
	}
}

// TestCombinedAdvancedFeatures tests combining all advanced features
func TestCombinedAdvancedFeatures(t *testing.T) {
	db := setupAdvancedFilterTestDB(t)

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&AdvancedProduct{}); err != nil {
		t.Fatalf("Failed to register AdvancedProduct entity: %v", err)
	}

	// Test combining multiple features in one query
	filter := "((Price gt 100 and not (Category eq 'Books')) or contains(Name,'Mouse')) and IsAvailable eq true"

	req := httptest.NewRequest("GET", "/AdvancedProducts?$filter="+url.QueryEscape(filter), nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Response does not contain value array")
	}

	// Expected: Laptop Pro, Laptop Basic, Mouse, Luxury Watch (not books, available, and match conditions)
	expectedCount := 4
	if len(value) != expectedCount {
		t.Errorf("Expected %d results, got %d", expectedCount, len(value))
		t.Logf("Results: %+v", value)
	}
}
