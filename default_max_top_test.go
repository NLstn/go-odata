package odata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TestProduct struct {
	ID          int     `json:"id" gorm:"primarykey" odata:"key"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type TestOrder struct {
	ID         int    `json:"id" gorm:"primarykey" odata:"key"`
	CustomerID int    `json:"customerId"`
	Total      float64 `json:"total"`
}

func setupTestServiceWithProducts(t *testing.T, productCount int) (*Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestProduct{}, &TestOrder{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Insert test products
	for i := 1; i <= productCount; i++ {
		product := TestProduct{
			ID:          i,
			Name:        fmt.Sprintf("Product %d", i),
			Description: fmt.Sprintf("Description %d", i),
			Price:       float64(i) * 10.0,
		}
		if err := db.Create(&product).Error; err != nil {
			t.Fatalf("Failed to create test product: %v", err)
		}
	}

	// Insert test orders
	for i := 1; i <= productCount; i++ {
		order := TestOrder{
			ID:         i,
			CustomerID: i % 5,
			Total:      float64(i) * 100.0,
		}
		if err := db.Create(&order).Error; err != nil {
			t.Fatalf("Failed to create test order: %v", err)
		}
	}

	service := NewService(db)
	if err := service.RegisterEntity(&TestProduct{}); err != nil {
		t.Fatalf("Failed to register TestProduct: %v", err)
	}
	if err := service.RegisterEntity(&TestOrder{}); err != nil {
		t.Fatalf("Failed to register TestOrder: %v", err)
	}

	return service, db
}

func TestSetDefaultMaxTop_ServiceLevel(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 50)

	// Set service-level default max top
	service.SetDefaultMaxTop(10)

	// Test without explicit $top - should return 10 results
	req := httptest.NewRequest(http.MethodGet, "/TestProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 10 {
		t.Errorf("Expected 10 results, got %d", len(value))
	}

	// Verify @odata.nextLink is present
	if _, exists := response["@odata.nextLink"]; !exists {
		t.Error("Expected @odata.nextLink to be present")
	}
}

func TestSetDefaultMaxTop_WithExplicitTop(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 50)

	// Set service-level default max top
	service.SetDefaultMaxTop(10)

	// Test with explicit $top=5 - should return 5 results
	req := httptest.NewRequest(http.MethodGet, "/TestProducts?$top=5", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 5 {
		t.Errorf("Expected 5 results (explicit $top should override default), got %d", len(value))
	}
}

func TestSetDefaultMaxTop_RemoveDefault(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 20)

	// Set and then remove default max top
	service.SetDefaultMaxTop(10)
	service.SetDefaultMaxTop(0) // Remove the default

	// Test without explicit $top - should return all 20 results
	req := httptest.NewRequest(http.MethodGet, "/TestProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 20 {
		t.Errorf("Expected 20 results (no limit), got %d", len(value))
	}

	// Verify @odata.nextLink is NOT present
	if _, exists := response["@odata.nextLink"]; exists {
		t.Error("Expected @odata.nextLink to be absent when all results are returned")
	}
}

func TestSetEntityDefaultMaxTop_EntityLevel(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 50)

	// Set entity-level default max top for TestProducts
	if err := service.SetEntityDefaultMaxTop("TestProducts", 7); err != nil {
		t.Fatalf("Failed to set entity default max top: %v", err)
	}

	// Test TestProducts - should return 7 results
	req := httptest.NewRequest(http.MethodGet, "/TestProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 7 {
		t.Errorf("Expected 7 results, got %d", len(value))
	}

	// Test TestOrders - should return all results (no default)
	req = httptest.NewRequest(http.MethodGet, "/TestOrders", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	response = map[string]interface{}{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok = response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 50 {
		t.Errorf("Expected 50 results for TestOrders (no limit), got %d", len(value))
	}
}

func TestSetEntityDefaultMaxTop_EntityOverridesService(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 50)

	// Set service-level default
	service.SetDefaultMaxTop(20)

	// Set entity-level default that overrides service default
	if err := service.SetEntityDefaultMaxTop("TestProducts", 5); err != nil {
		t.Fatalf("Failed to set entity default max top: %v", err)
	}

	// Test TestProducts - should return 5 results (entity default)
	req := httptest.NewRequest(http.MethodGet, "/TestProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 5 {
		t.Errorf("Expected 5 results (entity-level default should override service), got %d", len(value))
	}

	// Test TestOrders - should return 20 results (service default)
	req = httptest.NewRequest(http.MethodGet, "/TestOrders", nil)
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	response = map[string]interface{}{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok = response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 20 {
		t.Errorf("Expected 20 results for TestOrders (service default), got %d", len(value))
	}
}

func TestSetEntityDefaultMaxTop_ExplicitTopOverridesAll(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 50)

	// Set both service and entity defaults
	service.SetDefaultMaxTop(20)
	if err := service.SetEntityDefaultMaxTop("TestProducts", 10); err != nil {
		t.Fatalf("Failed to set entity default max top: %v", err)
	}

	// Test with explicit $top=3 - should return 3 results
	req := httptest.NewRequest(http.MethodGet, "/TestProducts?$top=3", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 3 {
		t.Errorf("Expected 3 results (explicit $top overrides all defaults), got %d", len(value))
	}
}

func TestSetEntityDefaultMaxTop_InvalidEntitySet(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 10)

	err := service.SetEntityDefaultMaxTop("NonExistent", 10)
	if err == nil {
		t.Fatal("Expected error when setting default for non-existent entity set")
	}

	expectedMsg := "entity set 'NonExistent' is not registered"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestSetDefaultMaxTop_WithMaxPageSizePreference(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 50)

	// Set service-level default max top
	service.SetDefaultMaxTop(20)

	// Test with Prefer: odata.maxpagesize=5 header
	req := httptest.NewRequest(http.MethodGet, "/TestProducts", nil)
	req.Header.Set("Prefer", "odata.maxpagesize=5")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	// MaxPageSize preference should take precedence
	if len(value) != 5 {
		t.Errorf("Expected 5 results (MaxPageSize preference should override default), got %d", len(value))
	}
}

func TestSetEntityDefaultMaxTop_RemoveEntityDefault(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 50)

	// Set service-level default
	service.SetDefaultMaxTop(20)

	// Set entity-level default
	if err := service.SetEntityDefaultMaxTop("TestProducts", 5); err != nil {
		t.Fatalf("Failed to set entity default max top: %v", err)
	}

	// Remove entity-level default (should fall back to service default)
	if err := service.SetEntityDefaultMaxTop("TestProducts", 0); err != nil {
		t.Fatalf("Failed to remove entity default max top: %v", err)
	}

	// Test - should return 20 results (service default)
	req := httptest.NewRequest(http.MethodGet, "/TestProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 20 {
		t.Errorf("Expected 20 results (should fall back to service default), got %d", len(value))
	}
}

func TestSetDefaultMaxTop_WithSkipAndTop(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 50)

	// Set service-level default max top
	service.SetDefaultMaxTop(10)

	// Test with $skip=5 (no explicit $top, should use default)
	req := httptest.NewRequest(http.MethodGet, "/TestProducts?$skip=5", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	// Should return 10 results (default) starting from the 6th product
	if len(value) != 10 {
		t.Errorf("Expected 10 results with $skip=5, got %d", len(value))
	}

	// Verify the first product ID is 6 (after skipping 5)
	firstProduct := value[0].(map[string]interface{})
	firstID := int(firstProduct["id"].(float64))
	if firstID != 6 {
		t.Errorf("Expected first product ID to be 6, got %d", firstID)
	}
}

func TestSetEntityDefaultMaxTop_ServiceDefaultChangesAfterEntitySet(t *testing.T) {
	service, _ := setupTestServiceWithProducts(t, 50)

	// Step 1: Set service-level default
	service.SetDefaultMaxTop(20)

	// Step 2: Set entity-level default
	if err := service.SetEntityDefaultMaxTop("TestProducts", 10); err != nil {
		t.Fatalf("Failed to set entity default max top: %v", err)
	}

	// Step 3: Change service-level default
	service.SetDefaultMaxTop(30)

	// Step 4: Remove entity-level default - should fall back to current service default (30)
	if err := service.SetEntityDefaultMaxTop("TestProducts", 0); err != nil {
		t.Fatalf("Failed to remove entity default max top: %v", err)
	}

	// Test - should return 30 results (current service default, not the old 20)
	req := httptest.NewRequest(http.MethodGet, "/TestProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 30 {
		t.Errorf("Expected 30 results (should use current service default), got %d", len(value))
	}
}
