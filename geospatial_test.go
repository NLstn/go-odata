package odata

import (
	"net/http"
	"net/http/httptest"
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

	// Create service WITHOUT enabling geospatial
	service := NewService(db)
	if err := service.RegisterEntity(&GeoTestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test with geospatial filter - should return 501 Not Implemented
	req := httptest.NewRequest("GET", "/GeoTestEntities?$filter=geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("Expected status 501 Not Implemented, got %d", w.Code)
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
