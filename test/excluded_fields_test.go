package odata_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ProductWithExcludedField entity for testing odata:"-" tag
type ProductWithExcludedField struct {
	ID             int     `json:"id" gorm:"primarykey" odata:"key"`
	Name           string  `json:"name" odata:"required"`
	Price          float64 `json:"price"`
	InternalSecret string  `json:"-" gorm:"-" odata:"-"` // Excluded from OData completely
	// Note: json:"-" is required to exclude from JSON serialization
	// odata:"-" excludes from OData metadata and query operations
}

func setupExcludedFieldsTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ProductWithExcludedField{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(&ProductWithExcludedField{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	if err := db.Create(&ProductWithExcludedField{ID: 1, Name: "Widget", Price: 9.99, InternalSecret: "secret-data"}).Error; err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	return service, db
}

func TestExcludedFields_NotInMetadata(t *testing.T) {
	service, _ := setupExcludedFieldsTestService(t)

	req := httptest.NewRequest("GET", "/$metadata", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Check that InternalSecret is not in the metadata
	metadataXML := w.Body.String()
	if strings.Contains(metadataXML, "InternalSecret") {
		t.Error("InternalSecret should not appear in metadata")
	}
}

func TestExcludedFields_NotInJSON_Response(t *testing.T) {
	service, _ := setupExcludedFieldsTestService(t)

	req := httptest.NewRequest("GET", "/ProductWithExcludedFields(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	t.Logf("Response: %+v", response)

	// Check that InternalSecret is not in the response
	if _, exists := response["InternalSecret"]; exists {
		t.Error("InternalSecret should not appear in JSON response")
	}
	if _, exists := response["internalSecret"]; exists {
		t.Error("internalSecret should not appear in JSON response")
	}

	// Verify other fields are present
	if response["name"] != "Widget" {
		t.Errorf("Expected name to be 'Widget', got %v", response["name"])
	}
}

func TestExcludedFields_CannotFilter(t *testing.T) {
	service, _ := setupExcludedFieldsTestService(t)

	req := httptest.NewRequest("GET", "/ProductWithExcludedFields?$filter=InternalSecret%20eq%20%27secret%27", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return an error because the field doesn't exist in OData
	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestExcludedFields_CannotSelect(t *testing.T) {
	service, _ := setupExcludedFieldsTestService(t)

	req := httptest.NewRequest("GET", "/ProductWithExcludedFields?$select=InternalSecret", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return an error because the field doesn't exist in OData
	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestExcludedFields_CannotOrderBy(t *testing.T) {
	service, _ := setupExcludedFieldsTestService(t)

	req := httptest.NewRequest("GET", "/ProductWithExcludedFields?$orderby=InternalSecret", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return an error because the field doesn't exist in OData
	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}
