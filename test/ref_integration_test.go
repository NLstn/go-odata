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

// Test entities for $ref tests
type RefProduct struct {
	ID             int              `json:"ID" gorm:"primaryKey" odata:"key"`
	Name           string           `json:"Name"`
	RefCategoryID  int              `json:"CategoryID"`
	RefCategory    *RefCategory     `json:"Category,omitempty" gorm:"foreignKey:RefCategoryID"`
	RefDescriptions []RefDescription `json:"Descriptions,omitempty" gorm:"foreignKey:ProductID"`
}

func (RefProduct) TableName() string {
	return "products"
}

type RefCategory struct {
	ID          int          `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string       `json:"Name"`
	RefProducts []RefProduct `json:"Products,omitempty" gorm:"foreignKey:RefCategoryID"`
}

func (RefCategory) TableName() string {
	return "categories"
}

type RefDescription struct {
	ID        int    `json:"ID" gorm:"primaryKey" odata:"key"`
	ProductID int    `json:"ProductID"`
	Language  string `json:"Language"`
	Text      string `json:"Text"`
}

func (RefDescription) TableName() string {
	return "descriptions"
}

func setupRefTest(t *testing.T) *odata.Service {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&RefProduct{}, &RefCategory{}, &RefDescription{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	categories := []RefCategory{
		{ID: 1, Name: "Electronics"},
		{ID: 2, Name: "Books"},
	}
	db.Create(&categories)

	products := []RefProduct{
		{ID: 1, Name: "Laptop", RefCategoryID: 1},
		{ID: 2, Name: "Mouse", RefCategoryID: 1},
		{ID: 3, Name: "Novel", RefCategoryID: 2},
	}
	db.Create(&products)

	descriptions := []RefDescription{
		{ID: 1, ProductID: 1, Language: "en", Text: "A laptop computer"},
		{ID: 2, ProductID: 1, Language: "de", Text: "Ein Laptop-Computer"},
		{ID: 3, ProductID: 2, Language: "en", Text: "A computer mouse"},
	}
	db.Create(&descriptions)

	service := odata.NewService(db)
	_ = service.RegisterEntity(&RefProduct{})
	_ = service.RegisterEntity(&RefCategory{})
	_ = service.RegisterEntity(&RefDescription{})

	return service
}

// TestEntityReference tests requesting a single entity reference
func TestEntityReference(t *testing.T) {
	service := setupRefTest(t)

	req := httptest.NewRequest(http.MethodGet, "/RefProducts(1)/$ref", nil)
	req.Host = "localhost:8080"
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

	// Verify @odata.context
	context, ok := response["@odata.context"].(string)
	if !ok || context == "" {
		t.Error("Expected @odata.context in response")
	}

	// Verify @odata.id
	id, ok := response["@odata.id"].(string)
	if !ok {
		t.Error("Expected @odata.id in response")
	}

	// Check that @odata.id contains the entity reference
	if id != "http://localhost:8080/RefProducts(1)" {
		t.Errorf("Expected @odata.id to be 'http://localhost:8080/RefProducts(1)', got '%s'", id)
	}

	// Ensure no "value" property exists (references don't have value)
	if _, exists := response["value"]; exists {
		t.Error("Entity reference should not contain 'value' property")
	}
}

// TestCollectionReference tests requesting collection entity references
func TestCollectionReference(t *testing.T) {
	service := setupRefTest(t)

	req := httptest.NewRequest(http.MethodGet, "/RefProducts/$ref", nil)
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

	// Verify @odata.context
	context, ok := response["@odata.context"].(string)
	if !ok || context == "" {
		t.Error("Expected @odata.context in response")
	}

	// Verify value array exists
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected 'value' array in response")
	}

	if len(value) != 3 {
		t.Errorf("Expected 3 product references, got %d", len(value))
	}

	// Check first reference structure
	firstRef, ok := value[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected reference to be a map")
	}

	id, ok := firstRef["@odata.id"].(string)
	if !ok || id == "" {
		t.Error("Expected @odata.id in reference")
	}

	// Verify no full entity data is present
	if _, exists := firstRef["Name"]; exists {
		t.Error("Reference should not contain entity properties like 'Name'")
	}
}

// TestNavigationPropertyReference tests requesting references for a navigation property
func TestNavigationPropertyReference(t *testing.T) {
	service := setupRefTest(t)

	req := httptest.NewRequest(http.MethodGet, "/RefProducts(1)/Descriptions/$ref", nil)
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

	// Verify value array exists
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected 'value' array in response")
	}

	if len(value) != 2 {
		t.Errorf("Expected 2 description references for Product 1, got %d", len(value))
	}

	// Check reference structure
	for i, ref := range value {
		refMap, ok := ref.(map[string]interface{})
		if !ok {
			t.Fatalf("Reference %d is not a map", i)
		}

		id, ok := refMap["@odata.id"].(string)
		if !ok || id == "" {
			t.Errorf("Reference %d missing @odata.id", i)
		}
	}
}

// TestSingleNavigationPropertyReference tests requesting reference for a single-valued navigation property
func TestSingleNavigationPropertyReference(t *testing.T) {
	service := setupRefTest(t)

	req := httptest.NewRequest(http.MethodGet, "/RefProducts(1)/Category/$ref", nil)
	req.Host = "localhost:8080"
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

	// Verify @odata.id
	id, ok := response["@odata.id"].(string)
	if !ok || id == "" {
		t.Error("Expected @odata.id in response")
	}

	// Check that it references the correct category
	if id != "http://localhost:8080/RefCategories(1)" {
		t.Errorf("Expected @odata.id to be 'http://localhost:8080/RefCategories(1)', got '%s'", id)
	}

	// Single navigation reference should not have 'value' array
	if _, exists := response["value"]; exists {
		t.Error("Single navigation reference should not contain 'value' array")
	}
}

// TestRefWithFilter tests $ref with $filter query option
func TestRefWithFilter(t *testing.T) {
	service := setupRefTest(t)

	req := httptest.NewRequest(http.MethodGet, "/RefProducts/$ref?$filter=CategoryID%20eq%201", nil)
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

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected 'value' array in response")
	}

	// Should only return products in category 1 (2 products)
	if len(value) != 2 {
		t.Errorf("Expected 2 product references with filter, got %d", len(value))
	}
}

// TestRefWithTop tests $ref with $top query option
func TestRefWithTop(t *testing.T) {
	service := setupRefTest(t)

	req := httptest.NewRequest(http.MethodGet, "/RefProducts/$ref?$top=2", nil)
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

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected 'value' array in response")
	}

	if len(value) != 2 {
		t.Errorf("Expected 2 product references with $top=2, got %d", len(value))
	}
}

// TestRefNotSupportedOnStructuralProperty tests that $ref is not allowed on structural properties
func TestRefNotSupportedOnStructuralProperty(t *testing.T) {
	service := setupRefTest(t)

	req := httptest.NewRequest(http.MethodGet, "/RefProducts(1)/Name/$ref", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}
