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

// ProductStatus represents product status as a flags enum
type ProductStatus int32

const (
	// ProductStatusNone represents no status
	ProductStatusNone ProductStatus = 0
	// ProductStatusInStock represents that the product is in stock
	ProductStatusInStock ProductStatus = 1
	// ProductStatusOnSale represents that the product is on sale
	ProductStatusOnSale ProductStatus = 2
	// ProductStatusDiscontinued represents that the product is discontinued
	ProductStatusDiscontinued ProductStatus = 4
	// ProductStatusFeatured represents that the product is featured
	ProductStatusFeatured ProductStatus = 8
)

// EnumMembers exposes the enum mapping for metadata generation.
func (ProductStatus) EnumMembers() map[string]int {
	return map[string]int{
		"None":         int(ProductStatusNone),
		"InStock":      int(ProductStatusInStock),
		"OnSale":       int(ProductStatusOnSale),
		"Discontinued": int(ProductStatusDiscontinued),
		"Featured":     int(ProductStatusFeatured),
	}
}

// EnumProduct represents a product entity with enum status for testing
type EnumProduct struct {
	ID     uint          `json:"ID" gorm:"primaryKey" odata:"key"`
	Name   string        `json:"Name" gorm:"not null"`
	Price  float64       `json:"Price" gorm:"not null"`
	Status ProductStatus `json:"Status" gorm:"not null" odata:"enum=ProductStatus,flags"`
}

func setupEnumTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the models
	if err := db.AutoMigrate(&EnumProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data with different flag combinations
	products := []EnumProduct{
		{ID: 1, Name: "Laptop", Price: 999.99, Status: ProductStatusInStock | ProductStatusFeatured},                        // 1 | 8 = 9
		{ID: 2, Name: "Mouse", Price: 29.99, Status: ProductStatusInStock | ProductStatusOnSale},                            // 1 | 2 = 3
		{ID: 3, Name: "Keyboard", Price: 79.99, Status: ProductStatusInStock},                                               // 1
		{ID: 4, Name: "Monitor", Price: 299.99, Status: ProductStatusInStock | ProductStatusOnSale | ProductStatusFeatured}, // 1 | 2 | 8 = 11
		{ID: 5, Name: "Chair", Price: 249.99, Status: ProductStatusDiscontinued},                                            // 4
		{ID: 6, Name: "Desk", Price: 449.99, Status: ProductStatusOnSale},                                                   // 2
	}

	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to seed products: %v", err)
	}

	return db
}

// TestEnumHasFunction tests the 'has' function for enum flags
func TestEnumHasFunction(t *testing.T) {
	db := setupEnumTestDB(t)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&EnumProduct{}); err != nil {
		t.Fatalf("Failed to register EnumProduct entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		expectedIDs   []uint
		description   string
	}{
		{
			name:          "Has InStock flag",
			filter:        "has(Status, 1)",
			expectedCount: 4,
			expectedIDs:   []uint{1, 2, 3, 4},
			description:   "Should return products with InStock flag set",
		},
		{
			name:          "Has OnSale flag",
			filter:        "has(Status, 2)",
			expectedCount: 3,
			expectedIDs:   []uint{2, 4, 6},
			description:   "Should return products with OnSale flag set",
		},
		{
			name:          "Has Discontinued flag",
			filter:        "has(Status, 4)",
			expectedCount: 1,
			expectedIDs:   []uint{5},
			description:   "Should return products with Discontinued flag set",
		},
		{
			name:          "Has Featured flag",
			filter:        "has(Status, 8)",
			expectedCount: 2,
			expectedIDs:   []uint{1, 4},
			description:   "Should return products with Featured flag set",
		},
		{
			name:          "Has multiple flags with AND",
			filter:        "has(Status, 1) and has(Status, 2)",
			expectedCount: 2,
			expectedIDs:   []uint{2, 4},
			description:   "Should return products with both InStock and OnSale flags set",
		},
		{
			name:          "Has multiple flags with OR",
			filter:        "has(Status, 2) or has(Status, 4)",
			expectedCount: 4,
			expectedIDs:   []uint{2, 4, 5, 6},
			description:   "Should return products with either OnSale or Discontinued flags set",
		},
		{
			name:          "Has flag combined with other filters",
			filter:        "has(Status, 1) and Price gt 100",
			expectedCount: 2,
			expectedIDs:   []uint{1, 4},
			description:   "Should return products with InStock flag and price > 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/EnumProducts?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
				return
			}

			var response struct {
				Value []EnumProduct `json:"value"`
			}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(response.Value) != tt.expectedCount {
				t.Errorf("Expected %d products, got %d", tt.expectedCount, len(response.Value))
			}

			// Verify the expected IDs
			resultIDs := make(map[uint]bool)
			for _, product := range response.Value {
				resultIDs[product.ID] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !resultIDs[expectedID] {
					t.Errorf("Expected product with ID %d, but not found", expectedID)
				}
			}
		})
	}
}

// TestEnumHasFunctionWithNot tests the 'has' function with NOT operator
func TestEnumHasFunctionWithNot(t *testing.T) {
	db := setupEnumTestDB(t)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&EnumProduct{}); err != nil {
		t.Fatalf("Failed to register EnumProduct entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		expectedIDs   []uint
		description   string
	}{
		{
			name:          "NOT has InStock flag",
			filter:        "not has(Status, 1)",
			expectedCount: 2,
			expectedIDs:   []uint{5, 6},
			description:   "Should return products without InStock flag set",
		},
		{
			name:          "NOT has Featured flag",
			filter:        "not has(Status, 8)",
			expectedCount: 4,
			expectedIDs:   []uint{2, 3, 5, 6},
			description:   "Should return products without Featured flag set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/EnumProducts?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
				return
			}

			var response struct {
				Value []EnumProduct `json:"value"`
			}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(response.Value) != tt.expectedCount {
				t.Errorf("Expected %d products, got %d", tt.expectedCount, len(response.Value))
			}

			// Verify the expected IDs
			resultIDs := make(map[uint]bool)
			for _, product := range response.Value {
				resultIDs[product.ID] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !resultIDs[expectedID] {
					t.Errorf("Expected product with ID %d, but not found", expectedID)
				}
			}
		})
	}
}

// TestEnumMetadata tests that enum types are properly exposed in metadata
func TestEnumMetadata(t *testing.T) {
	db := setupEnumTestDB(t)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&EnumProduct{}); err != nil {
		t.Fatalf("Failed to register EnumProduct entity: %v", err)
	}

	tests := []struct {
		name         string
		format       string
		contentType  string
		checkContent func(t *testing.T, body string)
		description  string
	}{
		{
			name:        "XML metadata contains EnumType",
			format:      "",
			contentType: "application/xml",
			checkContent: func(t *testing.T, body string) {
				// Check for EnumType definition
				if !containsEnum(body, `<EnumType Name="ProductStatus"`) {
					t.Error("Expected EnumType definition for ProductStatus in metadata")
				}
				// Check for IsFlags attribute
				if !containsEnum(body, `IsFlags="true"`) {
					t.Error("Expected IsFlags attribute on ProductStatus enum")
				}
				// Check for enum members
				if !containsEnum(body, `<Member Name="InStock" Value="1"`) {
					t.Error("Expected InStock member in ProductStatus enum")
				}
				if !containsEnum(body, `<Member Name="OnSale" Value="2"`) {
					t.Error("Expected OnSale member in ProductStatus enum")
				}
				if !containsEnum(body, `<Member Name="Featured" Value="8"`) {
					t.Error("Expected Featured member in ProductStatus enum")
				}
				// Check that Status property uses the enum type
				if !containsEnum(body, `Type="ODataService.ProductStatus"`) {
					t.Error("Expected Status property to use ProductStatus enum type")
				}
			},
			description: "XML metadata should contain enum type definitions",
		},
		{
			name:        "JSON metadata contains EnumType",
			format:      "json",
			contentType: "application/json",
			checkContent: func(t *testing.T, body string) {
				var metadata map[string]interface{}
				if err := json.Unmarshal([]byte(body), &metadata); err != nil {
					t.Fatalf("Failed to parse JSON metadata: %v", err)
				}

				// Check ODataService namespace exists
				odataService, ok := metadata["ODataService"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected ODataService namespace in metadata")
				}

				// Check ProductStatus enum type exists
				productStatus, ok := odataService["ProductStatus"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected ProductStatus enum type in metadata")
				}

				// Check enum type properties
				if kind, ok := productStatus["$Kind"].(string); !ok || kind != "EnumType" {
					t.Error("Expected ProductStatus to be an EnumType")
				}

				if isFlags, ok := productStatus["$IsFlags"].(bool); !ok || !isFlags {
					t.Error("Expected ProductStatus to have IsFlags set to true")
				}

				// Check enum members
				if inStock, ok := productStatus["InStock"].(float64); !ok || inStock != 1 {
					t.Error("Expected InStock member with value 1")
				}
				if onSale, ok := productStatus["OnSale"].(float64); !ok || onSale != 2 {
					t.Error("Expected OnSale member with value 2")
				}
				if featured, ok := productStatus["Featured"].(float64); !ok || featured != 8 {
					t.Error("Expected Featured member with value 8")
				}

				// Check that Status property uses the enum type
				enumProduct, ok := odataService["EnumProduct"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected EnumProduct entity type in metadata")
				}

				statusProp, ok := enumProduct["Status"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected Status property in EnumProduct")
				}

				if statusType, ok := statusProp["$Type"].(string); !ok || statusType != "ODataService.ProductStatus" {
					t.Errorf("Expected Status property to use ProductStatus enum type, got: %v", statusType)
				}
			},
			description: "JSON metadata should contain enum type definitions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.format == "" {
				req = httptest.NewRequest(http.MethodGet, "/$metadata", nil)
			} else {
				req = httptest.NewRequest(http.MethodGet, "/$metadata?$format="+tt.format, nil)
			}
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
				return
			}

			if contentType := w.Header().Get("Content-Type"); contentType != tt.contentType {
				t.Errorf("Expected Content-Type %s, got %s", tt.contentType, contentType)
			}

			tt.checkContent(t, w.Body.String())
		})
	}
}

// TestEnumStatusValues tests that enum status values are returned correctly in responses
func TestEnumStatusValues(t *testing.T) {
	db := setupEnumTestDB(t)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&EnumProduct{}); err != nil {
		t.Fatalf("Failed to register EnumProduct entity: %v", err)
	}

	// Get a specific product with combined flags
	req := httptest.NewRequest(http.MethodGet, "/EnumProducts(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		return
	}

	var product EnumProduct
	if err := json.NewDecoder(w.Body).Decode(&product); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Product 1 should have InStock | Featured = 9
	expectedStatus := ProductStatusInStock | ProductStatusFeatured
	if product.Status != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, product.Status)
	}

	// Verify the flags are set correctly
	if product.Status&ProductStatusInStock == 0 {
		t.Error("Expected InStock flag to be set")
	}
	if product.Status&ProductStatusFeatured == 0 {
		t.Error("Expected Featured flag to be set")
	}
	if product.Status&ProductStatusOnSale != 0 {
		t.Error("Expected OnSale flag to NOT be set")
	}
}

// Helper function to check if a string contains a substring
func containsEnum(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsEnumAt(s, substr))
}

func containsEnumAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
