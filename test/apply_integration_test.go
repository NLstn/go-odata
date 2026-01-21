package odata_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test entity for apply integration tests
type ApplyTestProduct struct {
	ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string  `json:"Name"`
	Price    float64 `json:"Price"`
	Category string  `json:"Category"`
	Quantity int     `json:"Quantity"`
}

func TestIntegrationApplyGroupBy(t *testing.T) {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&ApplyTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	products := []ApplyTestProduct{
		{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics", Quantity: 10},
		{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics", Quantity: 50},
		{ID: 3, Name: "Keyboard", Price: 149.99, Category: "Electronics", Quantity: 30},
		{ID: 4, Name: "Chair", Price: 249.99, Category: "Furniture", Quantity: 20},
		{ID: 5, Name: "Desk", Price: 399.99, Category: "Furniture", Quantity: 15},
		{ID: 6, Name: "Book", Price: 19.99, Category: "Books", Quantity: 100},
		{ID: 7, Name: "Pen", Price: 4.99, Category: "Books", Quantity: 200},
	}
	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	// Initialize OData service
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = service.RegisterEntity(&ApplyTestProduct{})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		validate       func(*testing.T, map[string]interface{})
	}{
		{
			name:           "GroupBy single property",
			path:           "/ApplyTestProducts?$apply=groupby((Category))",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				// Should have 3 groups: Electronics, Furniture, Books
				if len(value) != 3 {
					t.Errorf("Expected 3 groups, got %d", len(value))
				}
			},
		},
		{
			name:           "GroupBy with aggregate sum",
			path:           "/ApplyTestProducts?$apply=groupby((Category)%2Caggregate(Price%20with%20sum%20as%20TotalPrice))",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) != 3 {
					t.Errorf("Expected 3 groups, got %d", len(value))
					return
				}
				// Verify each group has Category and TotalPrice
				for i, item := range value {
					group, ok := item.(map[string]interface{})
					if !ok {
						t.Errorf("Group %d is not a map", i)
						continue
					}
					if _, hasCategory := group["Category"]; !hasCategory {
						t.Errorf("Group %d missing Category", i)
					}
					if _, hasTotal := group["TotalPrice"]; !hasTotal {
						t.Errorf("Group %d missing TotalPrice", i)
					}
				}
			},
		},
		{
			name:           "GroupBy with aggregate count",
			path:           "/ApplyTestProducts?$apply=groupby((Category)%2Caggregate($count%20as%20Total))",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) != 3 {
					t.Errorf("Expected 3 groups, got %d", len(value))
				}
				// Find Electronics group and verify count
				for _, item := range value {
					group, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					if category, ok := group["Category"].(string); ok && category == "Electronics" {
						if total, ok := group["Total"].(float64); !ok || total != 3 {
							t.Errorf("Expected Electronics count of 3, got %v", total)
						}
					}
				}
			},
		},
		{
			name:           "GroupBy with multiple aggregates",
			path:           "/ApplyTestProducts?$apply=groupby((Category)%2Caggregate(Price%20with%20sum%20as%20TotalPrice%2CQuantity%20with%20sum%20as%20TotalQuantity))",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) != 3 {
					t.Errorf("Expected 3 groups, got %d", len(value))
				}
				for _, item := range value {
					group, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					if _, hasTotalPrice := group["TotalPrice"]; !hasTotalPrice {
						t.Error("Missing TotalPrice")
					}
					if _, hasTotalQuantity := group["TotalQuantity"]; !hasTotalQuantity {
						t.Error("Missing TotalQuantity")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			// Handle request
			service.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response: %s", w.Body.String())
				return
			}

			// Parse response body
			body, _ := io.ReadAll(w.Body)
			var response map[string]interface{}
			if err := json.Unmarshal(body, &response); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Validate response
			if tt.validate != nil {
				tt.validate(t, response)
			}
		})
	}
}

func TestIntegrationApplyAggregate(t *testing.T) {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&ApplyTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	products := []ApplyTestProduct{
		{ID: 1, Name: "Product A", Price: 100.00, Category: "Electronics", Quantity: 10},
		{ID: 2, Name: "Product B", Price: 200.00, Category: "Electronics", Quantity: 5},
		{ID: 3, Name: "Product C", Price: 50.00, Category: "Electronics", Quantity: 20},
		{ID: 4, Name: "Product D", Price: 150.00, Category: "Furniture", Quantity: 8},
	}
	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	// Initialize OData service
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = service.RegisterEntity(&ApplyTestProduct{})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		validate       func(*testing.T, map[string]interface{})
	}{
		{
			name:           "Aggregate sum",
			path:           "/ApplyTestProducts?$apply=aggregate(Price%20with%20sum%20as%20TotalPrice)",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) != 1 {
					t.Errorf("Expected 1 result, got %d", len(value))
					return
				}
				result, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("Result is not a map")
				}
				totalPrice, ok := result["TotalPrice"].(float64)
				if !ok {
					t.Fatal("TotalPrice is not a number")
				}
				expectedTotal := 500.00 // 100 + 200 + 50 + 150
				if totalPrice != expectedTotal {
					t.Errorf("Expected TotalPrice %v, got %v", expectedTotal, totalPrice)
				}
			},
		},
		{
			name:           "Aggregate count",
			path:           "/ApplyTestProducts?$apply=aggregate($count%20as%20Total)",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) != 1 {
					t.Errorf("Expected 1 result, got %d", len(value))
					return
				}
				result, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("Result is not a map")
				}
				total, ok := result["Total"].(float64)
				if !ok {
					t.Fatal("Total is not a number")
				}
				if total != 4 {
					t.Errorf("Expected Total 4, got %v", total)
				}
			},
		},
		{
			name:           "Aggregate average",
			path:           "/ApplyTestProducts?$apply=aggregate(Price%20with%20average%20as%20AvgPrice)",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) != 1 {
					t.Errorf("Expected 1 result, got %d", len(value))
					return
				}
				result, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("Result is not a map")
				}
				avgPrice, ok := result["AvgPrice"].(float64)
				if !ok {
					t.Fatal("AvgPrice is not a number")
				}
				expectedAvg := 125.00 // (100 + 200 + 50 + 150) / 4
				if avgPrice != expectedAvg {
					t.Errorf("Expected AvgPrice %v, got %v", expectedAvg, avgPrice)
				}
			},
		},
		{
			name:           "Aggregate min",
			path:           "/ApplyTestProducts?$apply=aggregate(Price%20with%20min%20as%20MinPrice)",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) != 1 {
					t.Errorf("Expected 1 result, got %d", len(value))
					return
				}
				result, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("Result is not a map")
				}
				minPrice, ok := result["MinPrice"].(float64)
				if !ok {
					t.Fatal("MinPrice is not a number")
				}
				if minPrice != 50.00 {
					t.Errorf("Expected MinPrice 50.00, got %v", minPrice)
				}
			},
		},
		{
			name:           "Aggregate max",
			path:           "/ApplyTestProducts?$apply=aggregate(Price%20with%20max%20as%20MaxPrice)",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) != 1 {
					t.Errorf("Expected 1 result, got %d", len(value))
					return
				}
				result, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("Result is not a map")
				}
				maxPrice, ok := result["MaxPrice"].(float64)
				if !ok {
					t.Fatal("MaxPrice is not a number")
				}
				if maxPrice != 200.00 {
					t.Errorf("Expected MaxPrice 200.00, got %v", maxPrice)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			// Handle request
			service.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response: %s", w.Body.String())
				return
			}

			// Parse response body
			body, _ := io.ReadAll(w.Body)
			var response map[string]interface{}
			if err := json.Unmarshal(body, &response); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Validate response
			if tt.validate != nil {
				tt.validate(t, response)
			}
		})
	}
}

func TestIntegrationApplyFilter(t *testing.T) {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&ApplyTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	products := []ApplyTestProduct{
		{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics", Quantity: 10},
		{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics", Quantity: 50},
		{ID: 3, Name: "Keyboard", Price: 149.99, Category: "Electronics", Quantity: 30},
		{ID: 4, Name: "Chair", Price: 249.99, Category: "Furniture", Quantity: 20},
		{ID: 5, Name: "Desk", Price: 399.99, Category: "Furniture", Quantity: 15},
	}
	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	// Initialize OData service
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = service.RegisterEntity(&ApplyTestProduct{})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		validate       func(*testing.T, map[string]interface{})
	}{
		{
			name:           "Filter then groupby",
			path:           "/ApplyTestProducts?$apply=filter(Price%20gt%20100)/groupby((Category))",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				// Should have 2 groups after filtering: Electronics and Furniture
				if len(value) != 2 {
					t.Errorf("Expected 2 groups, got %d", len(value))
				}
			},
		},
		{
			name:           "Filter then groupby with aggregate",
			path:           "/ApplyTestProducts?$apply=filter(Price%20gt%20100)/groupby((Category)%2Caggregate(Price%20with%20sum%20as%20TotalPrice))",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) != 2 {
					t.Errorf("Expected 2 groups, got %d", len(value))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			// URL encode the query
			path := strings.ReplaceAll(tt.path, " ", "%20")
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			// Handle request
			service.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Logf("Response: %s", w.Body.String())
				return
			}

			// Parse response body
			body, _ := io.ReadAll(w.Body)
			var response map[string]interface{}
			if err := json.Unmarshal(body, &response); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Validate response
			if tt.validate != nil {
				tt.validate(t, response)
			}
		})
	}
}
