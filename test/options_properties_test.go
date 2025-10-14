package odata_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestOptionsProductWithRelations is a test entity with navigation properties
type TestOptionsProductWithRelations struct {
	ID           int                      `json:"id" gorm:"primarykey" odata:"key"`
	Name         string                   `json:"name"`
	CategoryID   int                      `json:"categoryId"`
	Category     *TestOptionsCategory     `json:"category" gorm:"foreignKey:CategoryID" odata:"nav"`
	Descriptions []TestOptionsDescription `json:"descriptions" gorm:"foreignKey:ProductID" odata:"nav"`
}

type TestOptionsCategory struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
}

type TestOptionsDescription struct {
	ID        int    `json:"id" gorm:"primarykey" odata:"key"`
	ProductID int    `json:"productId"`
	Text      string `json:"text"`
}

func setupOptionsPropertiesTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestOptionsProductWithRelations{}, &TestOptionsCategory{}, &TestOptionsDescription{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(TestOptionsProductWithRelations{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}
	if err := service.RegisterEntity(TestOptionsCategory{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}
	if err := service.RegisterEntity(TestOptionsDescription{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestOptionsStructuralProperty tests OPTIONS request on a structural property
func TestOptionsStructuralProperty(t *testing.T) {
	service, db := setupOptionsPropertiesTestService(t)

	// Insert a test product
	product := TestOptionsProductWithRelations{
		ID:   1,
		Name: "Test Product",
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodOptions, "/TestOptionsProductWithRelationses(1)/name", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}

	// OPTIONS should return no body
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty, got %v bytes", w.Body.Len())
	}
}

// TestOptionsStructuralPropertyValue tests OPTIONS request on a structural property with $value
func TestOptionsStructuralPropertyValue(t *testing.T) {
	service, db := setupOptionsPropertiesTestService(t)

	// Insert a test product
	product := TestOptionsProductWithRelations{
		ID:   1,
		Name: "Test Product",
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodOptions, "/TestOptionsProductWithRelationses(1)/name/$value", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}

	// OPTIONS should return no body
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty, got %v bytes", w.Body.Len())
	}
}

// TestOptionsNavigationProperty tests OPTIONS request on a navigation property (single-valued)
func TestOptionsNavigationPropertySingle(t *testing.T) {
	service, db := setupOptionsPropertiesTestService(t)

	// Insert test data
	category := TestOptionsCategory{ID: 1, Name: "Electronics"}
	db.Create(&category)

	product := TestOptionsProductWithRelations{
		ID:         1,
		Name:       "Test Product",
		CategoryID: 1,
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodOptions, "/TestOptionsProductWithRelationses(1)/category", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}

	// OPTIONS should return no body
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty, got %v bytes", w.Body.Len())
	}
}

// TestOptionsNavigationPropertyCollection tests OPTIONS request on a navigation property (collection-valued)
func TestOptionsNavigationPropertyCollection(t *testing.T) {
	service, db := setupOptionsPropertiesTestService(t)

	// Insert test data
	product := TestOptionsProductWithRelations{
		ID:   1,
		Name: "Test Product",
	}
	db.Create(&product)

	description := TestOptionsDescription{
		ID:        1,
		ProductID: 1,
		Text:      "Test description",
	}
	db.Create(&description)

	req := httptest.NewRequest(http.MethodOptions, "/TestOptionsProductWithRelationses(1)/descriptions", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}

	// OPTIONS should return no body
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty, got %v bytes", w.Body.Len())
	}
}

// TestOptionsInvalidProperty tests OPTIONS on a non-existent property
func TestOptionsInvalidProperty(t *testing.T) {
	service, db := setupOptionsPropertiesTestService(t)

	// Insert a test product
	product := TestOptionsProductWithRelations{
		ID:   1,
		Name: "Test Product",
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodOptions, "/TestOptionsProductWithRelationses(1)/nonExistentProperty", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 404 Not Found for non-existent property
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}
