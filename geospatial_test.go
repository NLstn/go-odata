package odata

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type GeoTestEntity struct {
	ID       int    `json:"ID" odata:"key"`
	Location string `json:"Location"`
	Name     string `json:"Name"`
}

func TestGeospatialNotEnabled(t *testing.T) {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate the test entity
	if err := db.AutoMigrate(&GeoTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create a test entity
	testEntity := GeoTestEntity{ID: 1, Location: "test", Name: "Test Entity"}
	if err := db.Create(&testEntity).Error; err != nil {
		t.Fatalf("Failed to create test entity: %v", err)
	}

	// Create service WITHOUT enabling geospatial
	service := NewService(db)
	if err := service.RegisterEntity(&GeoTestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test with geospatial filter on collection - should return 501 Not Implemented
	params := url.Values{}
	params.Set("$filter", "geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000")
	req := httptest.NewRequest("GET", "/GeoTestEntities?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("Collection query: Expected status 501 Not Implemented, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}
}

func TestGeospatialNotEnabledSingleEntity(t *testing.T) {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate the test entity
	if err := db.AutoMigrate(&GeoTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create a test entity
	testEntity := GeoTestEntity{ID: 1, Location: "test", Name: "Test Entity"}
	if err := db.Create(&testEntity).Error; err != nil {
		t.Fatalf("Failed to create test entity: %v", err)
	}

	// Create service WITHOUT enabling geospatial
	service := NewService(db)
	if err := service.RegisterEntity(&GeoTestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test with geospatial filter on single entity - should return 501 Not Implemented
	params := url.Values{}
	params.Set("$filter", "geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000")
	req := httptest.NewRequest("GET", "/GeoTestEntities(1)?"+params.Encode(), nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("Single entity query: Expected status 501 Not Implemented, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}
}

func TestGeospatialEnabledWithSQLiteNoSpatialite(t *testing.T) {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create service
	service := NewService(db)

	// Try to enable geospatial - should panic because SQLite doesn't have SpatiaLite
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when enabling geospatial without SpatiaLite, but no panic occurred")
		}
	}()

	service.EnableGeospatial()
}

func TestIsGeospatialEnabled(t *testing.T) {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	service := NewService(db)

	// Initially, geospatial should not be enabled
	if service.IsGeospatialEnabled() {
		t.Error("Expected geospatial to be disabled initially")
	}
}
