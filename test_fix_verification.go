//go:build ignore

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

// VerifyFixProduct represents a test product
type VerifyFixProduct struct {
	ID    uint
	Name  string
	Price float64
}

// VerifyFixProductDescription represents a test product description
type VerifyFixProductDescription struct {
	LanguageKey string
	Description string
	ProductID   uint
	Product     *VerifyFixProduct
}

// VerifyFix runs the verification test
func VerifyFix() bool {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return false
	}

	// Migrate schemas
	if err := db.AutoMigrate(&VerifyFixProduct{}, &VerifyFixProductDescription{}); err != nil {
		fmt.Printf("Failed to migrate: %v\n", err)
		return false
	}

	// Create test data
	productToCreate := &VerifyFixProduct{ID: 1, Name: "TestProduct", Price: 99.99}
	if err := db.Create(productToCreate).Error; err != nil {
		fmt.Printf("Failed to create product: %v\n", err)
		return false
	}

	desc := &VerifyFixProductDescription{
		LanguageKey: "en",
		Description: "Test Description",
		ProductID:   1,
	}
	if err := db.Create(desc).Error; err != nil {
		fmt.Printf("Failed to create description: %v\n", err)
		return false
	}

	// Create OData service
	service, err := NewService(db)
	if err != nil {
		fmt.Printf("NewService() error: %v\n", err)
		return false
	}

	if err := service.RegisterEntity(&VerifyFixProduct{}); err != nil {
		fmt.Printf("Failed to register VerifyFixProduct: %v\n", err)
		return false
	}
	if err := service.RegisterEntity(&VerifyFixProductDescription{}); err != nil {
		fmt.Printf("Failed to register VerifyFixProductDescription: %v\n", err)
		return false
	}

	// TEST: Select multiple navigation paths
	fmt.Println("Testing: $expand=Product&$select=LanguageKey,Product/Name,Product/Price")
	fmt.Println("Expected: Product should include ID (key) plus Name and Price")
	fmt.Println("")

	req := httptest.NewRequest("GET", "/VerifyFixProductDescriptions?$expand=Product&$select=LanguageKey,Product/Name,Product/Price", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	fmt.Printf("Status Code: %d\n", w.Code)

	if w.Code != http.StatusOK {
		fmt.Printf("❌ FAILED: Expected status 200, got %d\n", w.Code)
		fmt.Printf("Body: %s\n", w.Body.String())
		return false
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		fmt.Printf("❌ FAILED to parse response: %v\n", err)
		return false
	}

	// Pretty print response
	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	fmt.Printf("Response:\n%s\n\n", string(jsonBytes))

	value := response["value"].([]interface{})
	if len(value) == 0 {
		fmt.Printf("❌ FAILED: Expected at least one result\n")
		return false
	}

	item := value[0].(map[string]interface{})

	// Check LanguageKey
	if _, ok := item["LanguageKey"]; !ok {
		fmt.Printf("❌ FAILED: Expected LanguageKey property\n")
		return false
	}
	fmt.Printf("✓ LanguageKey present\n")

	// Check Product exists
	product, ok := item["Product"].(map[string]interface{})
	if !ok {
		fmt.Printf("❌ FAILED: Expected Product to be expanded\n")
		return false
	}
	fmt.Printf("✓ Product expanded\n")

	// Check for selected properties
	if _, ok := product["Name"]; !ok {
		fmt.Printf("❌ FAILED: Expected Product.Name property\n")
		return false
	}
	fmt.Printf("✓ Product.Name present\n")

	if _, ok := product["Price"]; !ok {
		fmt.Printf("❌ FAILED: Expected Product.Price property\n")
		return false
	}
	fmt.Printf("✓ Product.Price present\n")

	// THIS IS THE KEY TEST - ID should be present even though not explicitly selected
	if _, ok := product["ID"]; !ok {
		fmt.Printf("❌ FAILED: Expected Product.ID key property (THIS IS THE FIX)\n")
		fmt.Printf("   When selecting specific properties from an expanded entity,\n")
		fmt.Printf("   key properties must be automatically included per OData spec\n")
		return false
	}
	fmt.Printf("✓ Product.ID present (KEY PROPERTY AUTO-INCLUDED - FIX WORKS!)\n")

	fmt.Printf("\n✅ ALL CHECKS PASSED - FIX VERIFIED!\n")
	return true
}

// Dummy test function to make it look like a normal test file
func TestVerifyFix(t *testing.T) {
	if !VerifyFix() {
		t.Fatal("Fix verification failed")
	}
}
