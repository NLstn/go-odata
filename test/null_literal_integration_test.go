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

// NullableProduct represents a product entity with nullable fields for testing null literal filters
type NullableProduct struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name" gorm:"not null"`
	Price       float64 `json:"Price" gorm:"not null"`
	Description *string `json:"Description"` // Nullable field
	Notes       *string `json:"Notes"`       // Nullable field
}

func setupNullLiteralTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the models
	if err := db.AutoMigrate(&NullableProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data with null and non-null values
	desc1 := "High-performance laptop"
	desc2 := "Wireless mouse"
	notes1 := "Special edition"

	products := []NullableProduct{
		{ID: 1, Name: "Laptop Pro", Price: 1200.00, Description: &desc1, Notes: &notes1},
		{ID: 2, Name: "Laptop Basic", Price: 600.00, Description: &desc2, Notes: nil},
		{ID: 3, Name: "Mouse", Price: 29.99, Description: nil, Notes: nil},
		{ID: 4, Name: "Keyboard", Price: 79.99, Description: nil, Notes: &notes1},
	}

	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to seed products: %v", err)
	}

	return db
}

// TestNullLiteralInFilter tests filter expressions with null literal
func TestNullLiteralInFilter(t *testing.T) {
	db := setupNullLiteralTestDB(t)

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&NullableProduct{}); err != nil {
		t.Fatalf("Failed to register NullableProduct entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		expectedIDs   []uint
		description   string
	}{
		{
			name:          "Filter eq null",
			filter:        "Description eq null",
			expectedCount: 2,
			expectedIDs:   []uint{3, 4}, // Mouse and Keyboard have null Description
			description:   "Should return products where Description is NULL",
		},
		{
			name:          "Filter ne null",
			filter:        "Description ne null",
			expectedCount: 2,
			expectedIDs:   []uint{1, 2}, // Laptop Pro and Laptop Basic have non-null Description
			description:   "Should return products where Description is NOT NULL",
		},
		{
			name:          "Multiple null checks with AND",
			filter:        "Description eq null and Notes eq null",
			expectedCount: 1,
			expectedIDs:   []uint{3}, // Only Mouse has both null
			description:   "Should support AND with multiple null checks",
		},
		{
			name:          "Multiple null checks with OR",
			filter:        "Description eq null or Notes eq null",
			expectedCount: 3,
			expectedIDs:   []uint{2, 3, 4}, // Laptop Basic, Mouse, and Keyboard
			description:   "Should support OR with multiple null checks",
		},
		{
			name:          "Combine null check with other conditions",
			filter:        "Description eq null and Price lt 100",
			expectedCount: 2,
			expectedIDs:   []uint{3, 4}, // Mouse and Keyboard (both have null Description and Price < 100)
			description:   "Should support combining null checks with other filter conditions",
		},
		{
			name:          "NOT with null check",
			filter:        "not (Description eq null)",
			expectedCount: 2,
			expectedIDs:   []uint{1, 2}, // Same as Description ne null
			description:   "Should support NOT operator with null checks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/NullableProducts?$filter="+url.QueryEscape(tt.filter), nil)
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

			// Verify correct IDs were returned
			if len(tt.expectedIDs) > 0 {
				returnedIDs := make(map[uint]bool)
				for _, item := range value {
					itemMap := item.(map[string]interface{})
					id := uint(itemMap["ID"].(float64))
					returnedIDs[id] = true
				}

				for _, expectedID := range tt.expectedIDs {
					if !returnedIDs[expectedID] {
						t.Errorf("Expected ID %d in results but it was not found", expectedID)
					}
				}
			}
		})
	}
}
