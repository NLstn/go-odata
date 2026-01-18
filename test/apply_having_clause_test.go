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

// Test entity for having clause tests
type HavingProduct struct {
	ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string  `json:"Name"`
	Price    float64 `json:"Price"`
	Category string  `json:"Category"`
}

// TestApply_FilterAfterGroupBy_UsesHavingClause tests that filters after groupby
// use HAVING clause instead of WHERE clause
func TestApply_FilterAfterGroupBy_UsesHavingClause(t *testing.T) {
	db := setupHavingTestDB(t)

	// Create test data
	products := []HavingProduct{
		{ID: 1, Name: "Product 1", Price: 100.0, Category: "Cat1"},
		{ID: 2, Name: "Product 2", Price: 200.0, Category: "Cat1"},
		{ID: 3, Name: "Product 3", Price: 50.0, Category: "Cat2"},
		{ID: 4, Name: "Product 4", Price: 300.0, Category: "Cat3"},
	}

	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	service := setupHavingTestService(t, db)

	// Test: filter($count gt 1) after groupby should use HAVING clause
	// This would fail with SQL error if WHERE clause is used instead of HAVING
	req := httptest.NewRequest("GET", "/HavingProducts?$apply=groupby((Category))/filter($count%20gt%201)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value is not an array")
	}

	// Should have 1 group (Category: Cat1 with 2 products)
	if len(value) != 1 {
		t.Errorf("Expected 1 group, got %d", len(value))
	}

	// Verify the group has $count > 1
	if len(value) > 0 {
		group := value[0].(map[string]interface{})
		count := group["$count"]
		if count == nil {
			t.Error("Expected $count property")
		} else if countVal, ok := count.(float64); !ok || countVal <= 1 {
			t.Errorf("Expected $count > 1, got %v", count)
		}
	}
}

// TestApply_CountDistinct_UsesCorrectColumnNames tests that countdistinct
// uses the correct database column names (snake_case) instead of property names (PascalCase)
func TestApply_CountDistinct_UsesCorrectColumnNames(t *testing.T) {
	db := setupHavingTestDB(t)

	// Create test data
	products := []HavingProduct{
		{ID: 1, Name: "Product 1", Price: 100.0, Category: "Cat1"},
		{ID: 2, Name: "Product 2", Price: 200.0, Category: "Cat2"},
		{ID: 3, Name: "Product 3", Price: 50.0, Category: "Cat3"},
		{ID: 4, Name: "Product 4", Price: 300.0, Category: "Cat1"}, // Duplicate category
	}

	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	service := setupHavingTestService(t, db)

	// Test: countdistinct should use database column name (category), not property name (Category)
	// This would fail with SQL error if property name is used
	req := httptest.NewRequest("GET", "/HavingProducts?$apply=aggregate(Category%20with%20countdistinct%20as%20DistinctCategories)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value is not an array")
	}

	if len(value) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(value))
	}

	result := value[0].(map[string]interface{})
	distinctCount := result["DistinctCategories"]
	if distinctCount == nil {
		t.Error("Expected DistinctCategories property")
	} else if countVal, ok := distinctCount.(float64); !ok || countVal != 3 {
		t.Errorf("Expected DistinctCategories to be 3, got %v", distinctCount)
	}
}

// TestApply_OrderByComputedProperty tests that $orderby can use computed properties from $apply
func TestApply_OrderByComputedProperty(t *testing.T) {
	db := setupHavingTestDB(t)

	// Create test data
	products := []HavingProduct{
		{ID: 1, Name: "Product 1", Price: 50.0, Category: "Cat1"},
		{ID: 2, Name: "Product 2", Price: 100.0, Category: "Cat2"},
		{ID: 3, Name: "Product 3", Price: 200.0, Category: "Cat3"},
	}

	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	service := setupHavingTestService(t, db)

	// Test: $orderby should recognize TotalPrice as a computed property from aggregate
	// This would fail with "property path not supported" error if alias is not recognized
	req := httptest.NewRequest("GET", "/HavingProducts?$apply=groupby((Category),aggregate(Price%20with%20sum%20as%20TotalPrice))&$orderby=TotalPrice%20desc", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value is not an array")
	}

	if len(value) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(value))
	}

	// Verify ordering: should be descending by TotalPrice (200, 100, 50)
	prices := []float64{}
	for _, item := range value {
		group := item.(map[string]interface{})
		if price, ok := group["TotalPrice"].(float64); ok {
			prices = append(prices, price)
		}
	}

	if len(prices) == 3 {
		if prices[0] < prices[1] || prices[1] < prices[2] {
			t.Errorf("Expected descending order, got %v", prices)
		}
	}
}

// TestApply_MultipleAggregations tests that multiple aggregation methods
// all use correct column names
func TestApply_MultipleAggregations(t *testing.T) {
	db := setupHavingTestDB(t)

	// Create test data
	products := []HavingProduct{
		{ID: 1, Name: "Product 1", Price: 100.0, Category: "Cat1"},
		{ID: 2, Name: "Product 2", Price: 200.0, Category: "Cat1"},
		{ID: 3, Name: "Product 3", Price: 50.0, Category: "Cat1"},
	}

	for _, product := range products {
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	service := setupHavingTestService(t, db)

	// Test: aggregate with sum, average, min, max, countdistinct
	// All should use correct column names
	req := httptest.NewRequest("GET", "/HavingProducts?$apply=groupby((Category),aggregate(Price%20with%20sum%20as%20Total,Price%20with%20average%20as%20Avg,Price%20with%20min%20as%20Min,Price%20with%20max%20as%20Max,Price%20with%20countdistinct%20as%20Distinct))", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value is not an array")
	}

	if len(value) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(value))
	}

	result := value[0].(map[string]interface{})

	// Verify all aggregations are present and have correct values
	expectedValues := map[string]float64{
		"Total":    350.0,
		"Avg":      116.666667, // approximately
		"Min":      50.0,
		"Max":      200.0,
		"Distinct": 3.0,
	}

	for key, expectedVal := range expectedValues {
		actualVal, ok := result[key].(float64)
		if !ok {
			t.Errorf("Expected %s to be a number, got %T", key, result[key])
			continue
		}

		// Use tolerance for floating point comparison
		if key == "Avg" {
			if actualVal < 116.0 || actualVal > 117.0 {
				t.Errorf("Expected %s to be approximately %.2f, got %.2f", key, expectedVal, actualVal)
			}
		} else {
			if actualVal != expectedVal {
				t.Errorf("Expected %s to be %.2f, got %.2f", key, expectedVal, actualVal)
			}
		}
	}
}

func setupHavingTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&HavingProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	return db
}

func setupHavingTestService(t *testing.T, db *gorm.DB) http.Handler {
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&HavingProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}
	return service
}
