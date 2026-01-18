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

// Test entities with composite keys for navigation
type ProductWithCompositeKey struct {
	ID           uint                             `json:"ID" gorm:"primaryKey" odata:"key"`
	Name         string                           `json:"Name"`
	Descriptions []ProductDescriptionCompositeKey `json:"Descriptions" gorm:"foreignKey:ProductID"`
}

type ProductDescriptionCompositeKey struct {
	ProductID   uint                     `json:"ProductID" gorm:"primaryKey" odata:"key"`
	LanguageKey string                   `json:"LanguageKey" gorm:"primaryKey" odata:"key"`
	Description string                   `json:"Description"`
	Product     *ProductWithCompositeKey `json:"Product,omitempty" gorm:"foreignKey:ProductID"`
}

func setupCompositeKeyTest(t *testing.T) *odata.Service {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ProductWithCompositeKey{}, &ProductDescriptionCompositeKey{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	products := []ProductWithCompositeKey{
		{ID: 1, Name: "Laptop"},
		{ID: 2, Name: "Mouse"},
	}
	db.Create(&products)

	descriptions := []ProductDescriptionCompositeKey{
		{ProductID: 1, LanguageKey: "EN", Description: "A portable computer"},
		{ProductID: 1, LanguageKey: "FR", Description: "Un ordinateur portable"},
		{ProductID: 1, LanguageKey: "DE", Description: "Ein tragbarer Computer"},
		{ProductID: 2, LanguageKey: "EN", Description: "A computer pointing device"},
		{ProductID: 2, LanguageKey: "FR", Description: "Un dispositif de pointage"},
	}
	db.Create(&descriptions)

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	_ = service.RegisterEntity(&ProductWithCompositeKey{})
	_ = service.RegisterEntity(&ProductDescriptionCompositeKey{})

	return service
}

// TestNavigationCompositeKey_SingleItem tests accessing a specific item from a collection
// navigation property using a composite key
func TestNavigationCompositeKey_SingleItem(t *testing.T) {
	service := setupCompositeKeyTest(t)

	// Access ProductWithCompositeKeys(1)/Descriptions(ProductID=1,LanguageKey='EN')
	req := httptest.NewRequest(http.MethodGet, "/ProductWithCompositeKeys(1)/Descriptions(ProductID=1,LanguageKey='EN')", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should have a value array with exactly one item
	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(values) != 1 {
		t.Errorf("Expected 1 description, got %d", len(values))
		return
	}

	// Verify the correct description was returned
	desc := values[0].(map[string]interface{})
	if desc["ProductID"] != float64(1) {
		t.Errorf("Expected ProductID 1, got %v", desc["ProductID"])
	}
	if desc["LanguageKey"] != "EN" {
		t.Errorf("Expected LanguageKey EN, got %v", desc["LanguageKey"])
	}
	if desc["Description"] != "A portable computer" {
		t.Errorf("Expected description 'A portable computer', got %v", desc["Description"])
	}
}

// TestNavigationCompositeKey_DifferentLanguage tests accessing a different language description
func TestNavigationCompositeKey_DifferentLanguage(t *testing.T) {
	service := setupCompositeKeyTest(t)

	// Access ProductWithCompositeKeys(1)/Descriptions(ProductID=1,LanguageKey='FR')
	req := httptest.NewRequest(http.MethodGet, "/ProductWithCompositeKeys(1)/Descriptions(ProductID=1,LanguageKey='FR')", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(values) != 1 {
		t.Errorf("Expected 1 description, got %d", len(values))
		return
	}

	desc := values[0].(map[string]interface{})
	if desc["LanguageKey"] != "FR" {
		t.Errorf("Expected LanguageKey FR, got %v", desc["LanguageKey"])
	}
	if desc["Description"] != "Un ordinateur portable" {
		t.Errorf("Expected description 'Un ordinateur portable', got %v", desc["Description"])
	}
}

// TestNavigationCompositeKey_NotFound tests accessing a non-existent item
func TestNavigationCompositeKey_NotFound(t *testing.T) {
	service := setupCompositeKeyTest(t)

	// Access ProductWithCompositeKeys(1)/Descriptions(ProductID=1,LanguageKey='ES') - doesn't exist
	req := httptest.NewRequest(http.MethodGet, "/ProductWithCompositeKeys(1)/Descriptions(ProductID=1,LanguageKey='ES')", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestNavigationCompositeKey_WrongProduct tests accessing a description not related to the parent product
func TestNavigationCompositeKey_WrongProduct(t *testing.T) {
	service := setupCompositeKeyTest(t)

	// Access ProductWithCompositeKeys(1)/Descriptions(ProductID=2,LanguageKey='EN')
	// This description exists but belongs to Product 2, not Product 1
	req := httptest.NewRequest(http.MethodGet, "/ProductWithCompositeKeys(1)/Descriptions(ProductID=2,LanguageKey='EN')", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 (description not related to this product), got %d: %s", w.Code, w.Body.String())
	}
}

// TestNavigationCompositeKey_Ref tests accessing a reference with composite key
func TestNavigationCompositeKey_Ref(t *testing.T) {
	service := setupCompositeKeyTest(t)

	// Access ProductWithCompositeKeys(1)/Descriptions(ProductID=1,LanguageKey='EN')/$ref
	req := httptest.NewRequest(http.MethodGet, "/ProductWithCompositeKeys(1)/Descriptions(ProductID=1,LanguageKey='EN')/$ref", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should have @odata.id
	odataID, ok := response["@odata.id"].(string)
	if !ok {
		t.Fatal("Expected @odata.id in response")
	}

	// The ID should contain the composite key
	expectedID := "ProductDescriptionCompositeKeys(ProductID=1,LanguageKey='EN')"
	if !containsCompositeKeySubstring(odataID, expectedID) {
		t.Errorf("Expected @odata.id to contain '%s', got '%s'", expectedID, odataID)
	}
}

// TestNavigationCompositeKey_AllDescriptions tests accessing all descriptions (no key specified)
func TestNavigationCompositeKey_AllDescriptions(t *testing.T) {
	service := setupCompositeKeyTest(t)

	// Access ProductWithCompositeKeys(1)/Descriptions (all descriptions for product 1)
	req := httptest.NewRequest(http.MethodGet, "/ProductWithCompositeKeys(1)/Descriptions", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	// Product 1 should have 3 descriptions (EN, FR, DE)
	if len(values) != 3 {
		t.Errorf("Expected 3 descriptions, got %d", len(values))
	}
}

// Helper function to check if a string contains a composite key substring
func containsCompositeKeySubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findCompositeKeySubstring(s, substr)))
}

func findCompositeKeySubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
