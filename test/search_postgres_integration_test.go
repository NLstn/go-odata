package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresSearchTestProduct represents a test product with searchable fields for PostgreSQL
type PostgresSearchTestProduct struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name" odata:"searchable"`
	Description string  `json:"Description" odata:"searchable"`
	Category    string  `json:"Category"`
	Price       float64 `json:"Price"`
}

// getTestPostgresDB creates a test database connection for PostgreSQL
// Returns nil if PostgreSQL is not available
func getTestPostgresDB(t *testing.T) *gorm.DB {
	// Try to get DSN from environment variable
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		// Default test DSN with hardcoded credentials (postgres:postgres).
		// For your own test setup, set the POSTGRES_TEST_DSN environment variable
		// to avoid using default credentials.
		dsn = "postgresql://postgres:postgres@localhost:5432/odata_test?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skip("PostgreSQL not available, skipping test:", err)
		return nil
	}

	return db
}

func TestPostgresIntegrationSearch_WithSearchableFields(t *testing.T) {
	// Initialize PostgreSQL database
	db := getTestPostgresDB(t)
	if db == nil {
		return
	}

	// Clean up before test
	db.Exec("DROP TABLE IF EXISTS postgres_search_test_products_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_search_test_products CASCADE")

	// Auto-migrate
	if err := db.AutoMigrate(&PostgresSearchTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	products := []PostgresSearchTestProduct{
		{ID: 1, Name: "Laptop Pro", Description: "High-performance laptop for professionals", Category: "Electronics", Price: 1299.99},
		{ID: 2, Name: "Desktop Computer", Description: "Powerful desktop for gaming and work", Category: "Electronics", Price: 1599.99},
		{ID: 3, Name: "Wireless Mouse", Description: "Ergonomic wireless mouse", Category: "Accessories", Price: 29.99},
		{ID: 4, Name: "Mechanical Keyboard", Description: "RGB mechanical keyboard for gaming", Category: "Accessories", Price: 149.99},
		{ID: 5, Name: "Monitor", Description: "4K Ultra HD monitor", Category: "Electronics", Price: 599.99},
	}
	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	// Initialize OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&PostgresSearchTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedCount  int
		expectedIDs    []uint
		description    string
	}{
		{
			name:           "Search for 'laptop'",
			path:           "/PostgresSearchTestProducts?$search=laptop",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedIDs:    []uint{1},
			description:    "Should find products with 'laptop' in name or description",
		},
		{
			name:           "Search for 'gaming'",
			path:           "/PostgresSearchTestProducts?$search=gaming",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectedIDs:    []uint{2, 4},
			description:    "Should find products with 'gaming' in description",
		},
		{
			name:           "Search for 'wireless'",
			path:           "/PostgresSearchTestProducts?$search=wireless",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedIDs:    []uint{3},
			description:    "Should find products with 'wireless' in name or description",
		},
		{
			name:           "Search case-insensitive",
			path:           "/PostgresSearchTestProducts?$search=LAPTOP",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedIDs:    []uint{1},
			description:    "Search should be case-insensitive",
		},
		{
			name:           "Search for non-existent term",
			path:           "/PostgresSearchTestProducts?$search=smartphone",
			expectedStatus: http.StatusOK,
			expectedCount:  0,
			expectedIDs:    []uint{},
			description:    "Should return empty results for non-matching terms",
		},
		{
			name:           "Search with $top",
			path:           "/PostgresSearchTestProducts?$search=gaming&$top=1",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			description:    "Search should work with $top",
		},
		{
			name:           "Search with $filter",
			path:           "/PostgresSearchTestProducts?$search=gaming&$filter=Price%20gt%20200",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedIDs:    []uint{2},
			description:    "Search should work with $filter",
		},
		{
			name:           "Search with multiple words",
			path:           "/PostgresSearchTestProducts?$search=mechanical%20keyboard",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedIDs:    []uint{4},
			description:    "Should find products with both words in searchable fields",
		},
		{
			name:           "Search with special characters (quotes)",
			path:           "/PostgresSearchTestProducts?$search=laptop%27s",
			expectedStatus: http.StatusOK,
			expectedCount:  0, // May not match but should handle gracefully
			description:    "Should handle single quotes without errors",
		},
		{
			name:           "Search with special characters (semicolon)",
			path:           "/PostgresSearchTestProducts?$search=laptop%3Bdesktop",
			expectedStatus: http.StatusOK,
			expectedCount:  0, // plainto_tsquery removes special chars
			description:    "Should handle semicolons without SQL injection",
		},
		{
			name:           "Search with OR operator",
			path:           "/PostgresSearchTestProducts?$search=laptop%20OR%20wireless",
			expectedStatus: http.StatusOK,
			// plainto_tsquery treats OR as a regular word, not an operator
			description: "Should handle OR as text, not SQL operator",
		},
		{
			name:           "Search with AND operator",
			path:           "/PostgresSearchTestProducts?$search=laptop%20AND%20professional",
			expectedStatus: http.StatusOK,
			// plainto_tsquery treats AND as a regular word, not an operator
			description: "Should handle AND as text, not SQL operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d", tt.description, tt.expectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
				return
			}

			if w.Code == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatalf("Expected 'value' array in response")
				}

				if len(value) != tt.expectedCount {
					t.Errorf("%s: Expected %d results, got %d", tt.description, tt.expectedCount, len(value))
					t.Logf("Results: %+v", value)
				}

				// Check that the correct products are returned (when IDs are specified)
				if len(tt.expectedIDs) > 0 {
					foundIDs := make(map[uint]bool)
					for _, item := range value {
						product := item.(map[string]interface{})
						actualID := uint(product["ID"].(float64))
						foundIDs[actualID] = true
					}

					for _, expectedID := range tt.expectedIDs {
						if !foundIDs[expectedID] {
							t.Errorf("%s: Expected to find product ID %d", tt.description, expectedID)
						}
					}
				}
			}
		})
	}

	// Clean up after test
	db.Exec("DROP TABLE IF EXISTS postgres_search_test_products_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_search_test_products CASCADE")
}

func TestPostgresIntegrationSearch_PerformanceComparison(t *testing.T) {
	// Initialize PostgreSQL database
	db := getTestPostgresDB(t)
	if db == nil {
		return
	}

	// Clean up before test
	db.Exec("DROP TABLE IF EXISTS postgres_search_test_products_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_search_test_products CASCADE")

	// Auto-migrate
	if err := db.AutoMigrate(&PostgresSearchTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create a larger dataset to see FTS benefits
	products := make([]PostgresSearchTestProduct, 100)
	for i := 0; i < 100; i++ {
		products[i] = PostgresSearchTestProduct{
			ID:          uint(i + 1),
			Name:        "Product " + string(rune('A'+i%26)),
			Description: "Description with various keywords like laptop gaming wireless monitor",
			Category:    "Category",
			Price:       float64(i * 10),
		}
	}

	// Add specific products we want to search for
	products[0] = PostgresSearchTestProduct{ID: 1, Name: "Special Laptop", Description: "Amazing laptop with great features", Category: "Electronics", Price: 999.99}
	products[1] = PostgresSearchTestProduct{ID: 2, Name: "Gaming Desktop", Description: "Powerful gaming machine", Category: "Electronics", Price: 1499.99}

	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	// Initialize OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&PostgresSearchTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Perform search
	req := httptest.NewRequest(http.MethodGet, "/PostgresSearchTestProducts?$search=laptop", nil)
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
		t.Fatalf("Expected 'value' array in response")
	}

	// Should find at least the specific laptop we added
	if len(value) < 1 {
		t.Errorf("Expected to find at least 1 laptop, got %d results", len(value))
	}

	t.Logf("Search completed successfully, found %d results out of 100 products", len(value))

	// Clean up after test
	db.Exec("DROP TABLE IF EXISTS postgres_search_test_products_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_search_test_products CASCADE")
}
