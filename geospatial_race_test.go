package odata

import (
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestGeospatialConcurrentAccess tests for race conditions when accessing geospatialEnabled
// This test should pass with -race flag if the fix is correct
func TestGeospatialConcurrentAccess(t *testing.T) {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	var wg sync.WaitGroup

	// Start multiple goroutines reading the flag
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = service.IsGeospatialEnabled()
			}
		}()
	}

	// This should not cause a race condition anymore
	// Note: This will panic because SQLite doesn't have SpatiaLite,
	// but that's okay for the test - we're just testing for races
	// The panic is caught and the test continues
	func() {
		defer func() {
			recover() // Ignore the panic from missing SpatiaLite
		}()
		service.EnableGeospatial()
	}()

	wg.Wait()
}

// TestGeospatialHandlerConcurrentAccess tests for race conditions in EntityHandler
func TestGeospatialHandlerConcurrentAccess(t *testing.T) {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Auto-migrate the test entity
	if err := db.AutoMigrate(&GeoTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create service
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	// Register entity
	if err := service.RegisterEntity(&GeoTestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Get the handler
	handler := service.handlers["GeoTestEntities"]

	var wg sync.WaitGroup

	// Start multiple goroutines reading the handler's geospatialEnabled flag
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = handler.IsGeospatialEnabled()
			}
		}()
	}

	// Start multiple goroutines setting the handler's geospatialEnabled flag
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(enabled bool) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				handler.SetGeospatialEnabled(enabled)
			}
		}(i%2 == 0)
	}

	wg.Wait()
}
