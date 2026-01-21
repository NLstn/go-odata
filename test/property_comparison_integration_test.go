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

// ProductWithCost represents a product with both price and cost for comparison testing
type ProductWithCost struct {
	ID    uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name  string  `json:"Name" gorm:"not null"`
	Price float64 `json:"Price" gorm:"not null"`
	Cost  float64 `json:"Cost" gorm:"not null"`
}

func TestPropertyToPropertyComparisonIntegration(t *testing.T) {
	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&ProductWithCost{}); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	// Create test data with various price/cost relationships
	testProducts := []ProductWithCost{
		{ID: 1, Name: "Product A", Price: 100.0, Cost: 50.0},  // Price > Cost
		{ID: 2, Name: "Product B", Price: 75.0, Cost: 75.0},   // Price = Cost
		{ID: 3, Name: "Product C", Price: 60.0, Cost: 80.0},   // Price < Cost
		{ID: 4, Name: "Product D", Price: 200.0, Cost: 120.0}, // Price > Cost
		{ID: 5, Name: "Product E", Price: 90.0, Cost: 100.0},  // Price < Cost
	}

	for _, product := range testProducts {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create test product: %v", err)
		}
	}

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(&ProductWithCost{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedIDs   []uint
		expectedCount int
	}{
		{
			name:          "Price gt Cost - should return products where price > cost",
			filter:        "Price gt Cost",
			expectedIDs:   []uint{1, 4},
			expectedCount: 2,
		},
		{
			name:          "Price eq Cost - should return products where price = cost",
			filter:        "Price eq Cost",
			expectedIDs:   []uint{2},
			expectedCount: 1,
		},
		{
			name:          "Price lt Cost - should return products where price < cost",
			filter:        "Price lt Cost",
			expectedIDs:   []uint{3, 5},
			expectedCount: 2,
		},
		{
			name:          "Price ge Cost - should return products where price >= cost",
			filter:        "Price ge Cost",
			expectedIDs:   []uint{1, 2, 4},
			expectedCount: 3,
		},
		{
			name:          "Price le Cost - should return products where price <= cost",
			filter:        "Price le Cost",
			expectedIDs:   []uint{2, 3, 5},
			expectedCount: 3,
		},
		{
			name:          "Price ne Cost - should return products where price != cost",
			filter:        "Price ne Cost",
			expectedIDs:   []uint{1, 3, 4, 5},
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with filter (URL-encode the filter parameter)
			reqURL := "/ProductWithCosts?$filter=" + url.QueryEscape(tt.filter)
			req := httptest.NewRequest(http.MethodGet, reqURL, nil)
			w := httptest.NewRecorder()

			// Handle request
			service.ServeHTTP(w, req)

			// Check response
			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			// Parse response
			var response struct {
				Value []ProductWithCost `json:"value"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v. Body: %s", err, w.Body.String())
			}

			// Verify count
			if len(response.Value) != tt.expectedCount {
				t.Errorf("Expected %d products, got %d", tt.expectedCount, len(response.Value))
			}

			// Verify IDs
			actualIDs := make([]uint, len(response.Value))
			for i, product := range response.Value {
				actualIDs[i] = product.ID
			}

			// Check that we got the expected IDs (order may vary)
			expectedIDMap := make(map[uint]bool)
			for _, id := range tt.expectedIDs {
				expectedIDMap[id] = true
			}

			actualIDMap := make(map[uint]bool)
			for _, id := range actualIDs {
				actualIDMap[id] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !actualIDMap[expectedID] {
					t.Errorf("Expected product ID %d in results, but it was not found. Got IDs: %v", expectedID, actualIDs)
				}
			}

			for _, actualID := range actualIDs {
				if !expectedIDMap[actualID] {
					t.Errorf("Got unexpected product ID %d in results. Expected IDs: %v", actualID, tt.expectedIDs)
				}
			}

			// Log for debugging
			t.Logf("Filter: %s, Got %d products with IDs: %v", tt.filter, len(actualIDs), actualIDs)
		})
	}
}
