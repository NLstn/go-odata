package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test entity with searchable fields
type SearchTestProduct struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name" odata:"searchable"`
	Description string  `json:"Description" odata:"searchable"`
	Category    string  `json:"Category"`
	Price       float64 `json:"Price"`
}

// Test entity without searchable fields (all strings should be searchable by default)
type SearchTestProductNoTags struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name"`
	Description string  `json:"Description"`
	Category    string  `json:"Category"`
	Price       float64 `json:"Price"`
}

// Test entity with custom fuzziness
type SearchTestProductFuzzy struct {
	ID    uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name  string `json:"Name" odata:"searchable,fuzziness=2"`
	Email string `json:"Email" odata:"searchable,fuzziness=3"`
}

func TestIntegrationSearch_WithSearchableFields(t *testing.T) {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&SearchTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	products := []SearchTestProduct{
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
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	_ = service.RegisterEntity(&SearchTestProduct{})

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
			path:           "/SearchTestProducts?$search=laptop",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedIDs:    []uint{1},
			description:    "Should find products with 'laptop' in name or description",
		},
		{
			name:           "Search for 'gaming'",
			path:           "/SearchTestProducts?$search=gaming",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectedIDs:    []uint{2, 4},
			description:    "Should find products with 'gaming' in description",
		},
		{
			name:           "Search for 'wireless'",
			path:           "/SearchTestProducts?$search=wireless",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedIDs:    []uint{3},
			description:    "Should find products with 'wireless' in name or description",
		},
		{
			name:           "Search case-insensitive",
			path:           "/SearchTestProducts?$search=LAPTOP",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedIDs:    []uint{1},
			description:    "Search should be case-insensitive",
		},
		{
			name:           "Search for non-existent term",
			path:           "/SearchTestProducts?$search=smartphone",
			expectedStatus: http.StatusOK,
			expectedCount:  0,
			expectedIDs:    []uint{},
			description:    "Should return empty results for non-matching terms",
		},
		{
			name:           "Search with $top",
			path:           "/SearchTestProducts?$search=gaming&$top=1",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedIDs:    []uint{2},
			description:    "Search should work with $top",
		},
		{
			name:           "Search with $orderby",
			path:           "/SearchTestProducts?$search=gaming&$orderby=Price",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectedIDs:    []uint{4, 2}, // Ordered by price ascending
			description:    "Search should work with $orderby",
		},
		{
			name:           "Search with $filter",
			path:           "/SearchTestProducts?$search=gaming&$filter=Price%20gt%20200",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedIDs:    []uint{2},
			description:    "Search should work with $filter",
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
				}

				// Check that the correct products are returned
				for i, expectedID := range tt.expectedIDs {
					if i >= len(value) {
						break
					}
					product := value[i].(map[string]interface{})
					actualID := uint(product["ID"].(float64))
					if actualID != expectedID {
						t.Errorf("%s: Expected product ID %d at position %d, got %d", tt.description, expectedID, i, actualID)
					}
				}
			}
		})
	}
}

func TestIntegrationSearch_NoSearchableFields(t *testing.T) {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&SearchTestProductNoTags{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	products := []SearchTestProductNoTags{
		{ID: 1, Name: "Laptop", Description: "Gaming laptop", Category: "Electronics", Price: 1299.99},
		{ID: 2, Name: "Mouse", Description: "Wireless mouse", Category: "Accessories", Price: 29.99},
		{ID: 3, Name: "Chair", Description: "Office chair", Category: "Furniture", Price: 249.99},
	}
	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	// Initialize OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	_ = service.RegisterEntity(&SearchTestProductNoTags{})

	tests := []struct {
		name          string
		searchQuery   string
		expectedCount int
		description   string
	}{
		{
			name:          "Search in Name field",
			searchQuery:   "laptop",
			expectedCount: 1,
			description:   "Should search all string fields when no searchable fields defined",
		},
		{
			name:          "Search in Description field",
			searchQuery:   "wireless",
			expectedCount: 1,
			description:   "Should find products with term in description",
		},
		{
			name:          "Search in Category field",
			searchQuery:   "Furniture",
			expectedCount: 1,
			description:   "Should find products with term in category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/SearchTestProductNoTagses?$search="+tt.searchQuery, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("%s: Expected status %d, got %d", tt.description, http.StatusOK, w.Code)
				return
			}

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
			}
		})
	}
}

func TestIntegrationSearch_InvalidQuery(t *testing.T) {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&SearchTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Initialize OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	_ = service.RegisterEntity(&SearchTestProduct{})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "Empty search query",
			path:           "/SearchTestProducts?$search=%20%20%20",
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject empty search query",
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
			}
		})
	}
}
