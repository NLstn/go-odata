package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SearchTestProduct represents a test product entity with searchable fields
type SearchTestProduct struct {
	ID          int     `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name" odata:"searchable"`
	Description string  `json:"Description" odata:"searchable"`
	Category    string  `json:"Category"`
	Price       float64 `json:"Price"`
}

func TestSearchWithFTS_Integration(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&SearchTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	testData := []SearchTestProduct{
		{ID: 1, Name: "Laptop Pro", Description: "High-performance laptop for professionals", Category: "Electronics", Price: 1200},
		{ID: 2, Name: "Desktop Computer", Description: "Powerful desktop for gaming", Category: "Electronics", Price: 1500},
		{ID: 3, Name: "Wireless Mouse", Description: "Ergonomic wireless mouse", Category: "Accessories", Price: 25},
		{ID: 4, Name: "Mechanical Keyboard", Description: "RGB mechanical keyboard", Category: "Accessories", Price: 75},
		{ID: 5, Name: "Gaming Laptop", Description: "High-end gaming laptop", Category: "Electronics", Price: 2000},
	}
	if err := db.Create(&testData).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Create metadata
	entityMeta, err := metadata.AnalyzeEntity(SearchTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Create FTS manager
	ftsManager := query.NewFTSManager(db)

	// Create handler
	handler := NewEntityHandler(db, entityMeta, nil)
	handler.SetFTSManager(ftsManager)

	tests := []struct {
		name          string
		searchQuery   string
		expectedIDs   []int
		description   string
	}{
		{
			name:          "Search for 'laptop'",
			searchQuery:   "laptop",
			expectedIDs:   []int{1, 5},
			description:   "Should find products with 'laptop' in name or description",
		},
		{
			name:          "Search for 'gaming'",
			searchQuery:   "gaming",
			expectedIDs:   []int{2, 5},
			description:   "Should find products with 'gaming' in name or description",
		},
		{
			name:          "Search for 'wireless'",
			searchQuery:   "wireless",
			expectedIDs:   []int{3},
			description:   "Should find wireless mouse",
		},
		{
			name:          "Search for 'keyboard'",
			searchQuery:   "keyboard",
			expectedIDs:   []int{4},
			description:   "Should find keyboard",
		},
		{
			name:          "Search for 'professional'",
			searchQuery:   "professionals",
			expectedIDs:   []int{1},
			description:   "Should find laptop with 'professionals' in description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with search parameter
			req := httptest.NewRequest("GET", "/SearchTestProducts?$search="+tt.searchQuery, nil)
			w := httptest.NewRecorder()

			// Execute request
			handler.handleGetCollection(w, req)

			// Check response
			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			// Parse response
			var response struct {
				Value []SearchTestProduct `json:"value"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Verify results
			if len(response.Value) != len(tt.expectedIDs) {
				t.Errorf("%s: Expected %d results, got %d", tt.description, len(tt.expectedIDs), len(response.Value))
				t.Logf("Response: %+v", response.Value)
			}

			// Check IDs
			foundIDs := make(map[int]bool)
			for _, product := range response.Value {
				foundIDs[product.ID] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !foundIDs[expectedID] {
					t.Errorf("%s: Expected to find product with ID %d", tt.description, expectedID)
				}
			}
		})
	}
}

func TestSearchFallback_NoFTS(t *testing.T) {
	// This test verifies that search still works when FTS is not available
	// by falling back to in-memory search

	// Setup database (using a special config that might not have FTS)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&SearchTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	testData := []SearchTestProduct{
		{ID: 1, Name: "Laptop Pro", Description: "High-performance laptop", Category: "Electronics", Price: 1200},
		{ID: 2, Name: "Desktop Computer", Description: "Powerful desktop", Category: "Electronics", Price: 1500},
	}
	if err := db.Create(&testData).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Create metadata
	entityMeta, err := metadata.AnalyzeEntity(SearchTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Create handler WITHOUT FTS manager (simulating no FTS available)
	handler := NewEntityHandler(db, entityMeta, nil)
	// Don't set FTS manager - handler.SetFTSManager(nil)

	// Create request with search parameter
	req := httptest.NewRequest("GET", "/SearchTestProducts?$search=laptop", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.handleGetCollection(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Parse response
	var response struct {
		Value []SearchTestProduct `json:"value"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify results - should still find the laptop using in-memory search
	if len(response.Value) != 1 {
		t.Errorf("Expected 1 result (fallback to in-memory search), got %d", len(response.Value))
	}

	if len(response.Value) > 0 && response.Value[0].ID != 1 {
		t.Errorf("Expected product ID 1, got %d", response.Value[0].ID)
	}
}

func TestSearchWithFTS_CaseInsensitive(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&SearchTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	testData := []SearchTestProduct{
		{ID: 1, Name: "LAPTOP PRO", Description: "High-performance LAPTOP", Category: "Electronics", Price: 1200},
	}
	if err := db.Create(&testData).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// Create metadata
	entityMeta, err := metadata.AnalyzeEntity(SearchTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Create FTS manager
	ftsManager := query.NewFTSManager(db)

	// Create handler
	handler := NewEntityHandler(db, entityMeta, nil)
	handler.SetFTSManager(ftsManager)

	// Test case-insensitive search
	req := httptest.NewRequest("GET", "/SearchTestProducts?$search=laptop", nil)
	w := httptest.NewRecorder()

	handler.handleGetCollection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Value []SearchTestProduct `json:"value"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should find the uppercase LAPTOP with lowercase search
	if len(response.Value) != 1 {
		t.Errorf("Expected 1 result for case-insensitive search, got %d", len(response.Value))
	}
}
