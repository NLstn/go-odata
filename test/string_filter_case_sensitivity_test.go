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

// CaseSensitivityProduct is a minimal entity used to verify that the
// contains()/startswith()/endswith() OData filter functions perform ordinal
// (case-sensitive) string comparison, per the OData v4.0 URL Conventions spec
// (Part 2, Sec. 5.1.1.7). Regression test for
// https://github.com/NLstn/go-odata/issues/790.
type CaseSensitivityProduct struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name" gorm:"not null"`
}

func setupCaseSensitivityTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&CaseSensitivityProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	products := []CaseSensitivityProduct{
		{ID: 1, Name: "Laptop"},
		{ID: 2, Name: "Premium Laptop Pro"},
	}
	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to seed products: %v", err)
	}

	return db
}

func newCaseSensitivityService(t *testing.T, db *gorm.DB) *odata.Service {
	t.Helper()

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&CaseSensitivityProduct{}); err != nil {
		t.Fatalf("Failed to register CaseSensitivityProduct entity: %v", err)
	}
	return service
}

func queryCaseSensitivityProducts(t *testing.T, service *odata.Service, filter string) []interface{} {
	t.Helper()

	req := httptest.NewRequest("GET", "/CaseSensitivityProducts?$filter="+url.QueryEscape(filter), nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatalf("Response does not contain value array: %s", w.Body.String())
	}

	return value
}

// TestFilterContainsIsCaseSensitive verifies that contains() only matches
// substrings with the exact case supplied by the client. A lowercase
// 'laptop' must not match the differently-cased "Laptop" or
// "Premium Laptop Pro" substrings.
func TestFilterContainsIsCaseSensitive(t *testing.T) {
	db := setupCaseSensitivityTestDB(t)
	service := newCaseSensitivityService(t, db)

	t.Run("differently-cased substring does not match", func(t *testing.T) {
		results := queryCaseSensitivityProducts(t, service, "contains(Name,'laptop')")
		if len(results) != 0 {
			t.Errorf("Expected 0 results for contains(Name,'laptop'), got %d: %v", len(results), results)
		}
	})

	t.Run("exact-case substring still matches", func(t *testing.T) {
		results := queryCaseSensitivityProducts(t, service, "contains(Name,'Laptop')")
		if len(results) != 2 {
			t.Errorf("Expected 2 results for contains(Name,'Laptop'), got %d: %v", len(results), results)
		}
	})
}

// TestFilterStartsWithIsCaseSensitive verifies that startswith() performs
// ordinal (case-sensitive) matching.
func TestFilterStartsWithIsCaseSensitive(t *testing.T) {
	db := setupCaseSensitivityTestDB(t)
	service := newCaseSensitivityService(t, db)

	t.Run("differently-cased prefix does not match", func(t *testing.T) {
		results := queryCaseSensitivityProducts(t, service, "startswith(Name,'laptop')")
		if len(results) != 0 {
			t.Errorf("Expected 0 results for startswith(Name,'laptop'), got %d: %v", len(results), results)
		}
	})

	t.Run("exact-case prefix still matches", func(t *testing.T) {
		results := queryCaseSensitivityProducts(t, service, "startswith(Name,'Laptop')")
		if len(results) != 1 {
			t.Errorf("Expected 1 result for startswith(Name,'Laptop'), got %d: %v", len(results), results)
		}
	})
}

// TestFilterEndsWithIsCaseSensitive verifies that endswith() performs
// ordinal (case-sensitive) matching.
func TestFilterEndsWithIsCaseSensitive(t *testing.T) {
	db := setupCaseSensitivityTestDB(t)
	service := newCaseSensitivityService(t, db)

	t.Run("differently-cased suffix does not match", func(t *testing.T) {
		results := queryCaseSensitivityProducts(t, service, "endswith(Name,'pro')")
		if len(results) != 0 {
			t.Errorf("Expected 0 results for endswith(Name,'pro'), got %d: %v", len(results), results)
		}
	})

	t.Run("exact-case suffix still matches", func(t *testing.T) {
		results := queryCaseSensitivityProducts(t, service, "endswith(Name,'Pro')")
		if len(results) != 1 {
			t.Errorf("Expected 1 result for endswith(Name,'Pro'), got %d: %v", len(results), results)
		}
	})
}
