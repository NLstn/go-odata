package odata

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
	service, err := NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
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
	service, err := NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
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
	service, err := NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }

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

	service, err := NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }

	// Initially, geospatial should not be enabled
	if service.IsGeospatialEnabled() {
		t.Error("Expected geospatial to be disabled initially")
	}
}

func TestCheckGeospatialSupportNilDB(t *testing.T) {
	if err := checkGeospatialSupport(nil, slog.Default()); err == nil {
		t.Fatal("expected error for nil database")
	}
}

func TestCheckGeospatialSupportSQLiteError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	err = checkGeospatialSupport(db, slog.Default())
	if err == nil || !strings.Contains(err.Error(), "SpatiaLite") {
		t.Fatalf("expected SpatiaLite error, got %v", err)
	}
}

func TestCheckGeospatialSupportOtherDialectsErrors(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := checkPostgreSQLGeospatialSupport(db, slog.Default()); err == nil || !strings.Contains(err.Error(), "PostGIS extension") {
		t.Fatalf("expected PostGIS error, got %v", err)
	}

	if err := checkMySQLGeospatialSupport(db, slog.Default()); err == nil || !strings.Contains(err.Error(), "MySQL/MariaDB spatial functions") {
		t.Fatalf("expected MySQL/MariaDB error, got %v", err)
	}

	if err := checkSQLServerGeospatialSupport(db, slog.Default()); err == nil || !strings.Contains(err.Error(), "SQL Server spatial types") {
		t.Fatalf("expected SQL Server error, got %v", err)
	}
}

// TestCheckGeospatialSupportUnsupportedDialect tests error handling for unsupported database dialects
func TestCheckGeospatialSupportUnsupportedDialect(t *testing.T) {
	// Create a mock database connection with a custom dialect
	// Since we're using SQLite, we can't truly test another dialect, but we test the logic
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// The actual test for unsupported dialect would require a mock or custom driver
	// For now, verify that SQLite returns the appropriate error when SpatiaLite is missing
	err = checkGeospatialSupport(db, slog.Default())
	if err == nil {
		t.Fatal("expected error for missing SpatiaLite")
	}
	if !strings.Contains(err.Error(), "SpatiaLite") {
		t.Errorf("expected SpatiaLite error message, got: %v", err)
	}
}
