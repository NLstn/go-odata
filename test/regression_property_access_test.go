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

// TestRegressionProductNameAccess tests the specific scenario from the problem statement:
// "Property access is not working anymore, calls to /Products(1)/Name return 404 
// saying the Name is not a valid navigation property"
//
// This test verifies that structural properties (like Name) are NOT treated as navigation properties
func TestRegressionProductNameAccess(t *testing.T) {
	// Setup test entities that mirror the devserver structure
	type Product struct {
		ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
		Name     string  `json:"Name" gorm:"not null" odata:"required"`
		Price    float64 `json:"Price" gorm:"not null" odata:"required"`
		Category string  `json:"Category" odata:"required"`
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	product := Product{
		ID:       1,
		Name:     "Laptop",
		Price:    999.99,
		Category: "Electronics",
	}
	db.Create(&product)

	// Test the exact scenario from the problem statement: /Products(1)/Name
	req := httptest.NewRequest(http.MethodGet, "/Products(1)/Name", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	// Should NOT return 404
	if w.Code == http.StatusNotFound {
		t.Errorf("REGRESSION: /Products(1)/Name returned 404. Body: %s", w.Body.String())
		
		// Check if the error mentions "navigation property"
		if strings.Contains(w.Body.String(), "navigation property") {
			t.Error("REGRESSION: Name is being incorrectly treated as a navigation property")
		}
	}

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify the response structure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have @odata.context
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Response missing @odata.context")
	}

	// Should have value wrapper (structural property)
	if _, ok := response["value"]; !ok {
		t.Error("Response missing 'value' field - structural properties should have value wrapper")
	}

	// Verify the actual value
	if response["value"] != "Laptop" {
		t.Errorf("Expected value 'Laptop', got %v", response["value"])
	}
}

// TestRegressionNavigationPropertyDistinction verifies that navigation properties
// are correctly identified and NOT confused with structural properties
func TestRegressionNavigationPropertyDistinction(t *testing.T) {
	type Category struct {
		ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
		Name string `json:"Name" odata:"required"`
	}

	type ProductWithNav struct {
		ID         uint      `json:"ID" gorm:"primaryKey" odata:"key"`
		Name       string    `json:"Name" odata:"required"`
		CategoryID uint      `json:"CategoryID"`
		Category   *Category `json:"Category" gorm:"foreignKey:CategoryID"`
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ProductWithNav{}, &Category{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&ProductWithNav{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}
	if err := service.RegisterEntity(&Category{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	category := Category{ID: 1, Name: "Electronics"}
	db.Create(&category)
	product := ProductWithNav{ID: 1, Name: "Laptop", CategoryID: 1}
	db.Create(&product)

	tests := []struct {
		name              string
		url               string
		expectValueWrapper bool
		expectEntityProps  bool
		description       string
	}{
		{
			name:               "Structural property Name",
			url:                "/ProductWithNavs(1)/Name",
			expectValueWrapper: true,
			expectEntityProps:  false,
			description:        "Name is a structural property and should have value wrapper",
		},
		{
			name:               "Navigation property Category",
			url:                "/ProductWithNavs(1)/Category",
			expectValueWrapper: false,
			expectEntityProps:  true,
			description:        "Category is a navigation property and should return entity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check for value wrapper
			_, hasValue := response["value"]
			if tt.expectValueWrapper && !hasValue {
				t.Errorf("%s: Expected value wrapper but not found", tt.description)
			}
			if !tt.expectValueWrapper && hasValue {
				// For navigation properties, if there's a "value" key, it might be empty
				// The key point is that entity properties should be present
				if !tt.expectEntityProps {
					t.Errorf("%s: Unexpected value wrapper found", tt.description)
				}
			}

			// Check for entity properties (like ID)
			_, hasID := response["ID"]
			if tt.expectEntityProps && !hasID {
				t.Errorf("%s: Expected entity properties but not found", tt.description)
			}
			if !tt.expectEntityProps && hasID {
				t.Errorf("%s: Unexpected entity properties found", tt.description)
			}
		})
	}
}

// TestRegressionErrorMessageAccuracy verifies that error messages are accurate
// when accessing nonexistent properties
func TestRegressionErrorMessageAccuracy(t *testing.T) {
	type Product struct {
		ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
		Name string `json:"Name" odata:"required"`
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	product := Product{ID: 1, Name: "Test"}
	db.Create(&product)

	// Try to access a nonexistent property
	req := httptest.NewRequest(http.MethodGet, "/Products(1)/NonExistent", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Response should have error object")
	}

	message, ok := errorObj["message"].(string)
	if !ok {
		t.Fatal("Error should have message string")
	}

	// The error should say "Property not found", NOT "Navigation property not found"
	// This prevents confusion when accessing structural properties
	if message != "Property not found" {
		t.Errorf("Error message should be 'Property not found', got: %s", message)
	}

	// The error should NOT specifically mention "navigation property" since
	// we don't know if the user intended a navigation or structural property
	if strings.Contains(strings.ToLower(w.Body.String()), "navigation property not found") {
		t.Error("Error should not specifically mention 'navigation property' for ambiguous cases")
	}
}
