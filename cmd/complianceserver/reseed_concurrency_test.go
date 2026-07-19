package main

import (
	"path/filepath"
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestSeedDatabaseConcurrent verifies that concurrent reseeds are serialized and
// never corrupt the schema. The compliance suite runs suites in parallel, each
// issuing a Reseed at setup; without serialization two overlapping drop/migrate/
// seed sequences deadlock and raise foreign-key / duplicate-key errors (observed
// on PostgreSQL). seedMu must make every seedDatabase call run to completion
// before the next begins.
func TestSeedDatabaseConcurrent(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "concurrent.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	const goroutines = 8
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := seedDatabase(db); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent seedDatabase failed: %v", err)
	}

	// The final state must be exactly one clean seed (7 products), proving the
	// reseeds serialized rather than interleaving partial writes.
	var products int64
	if err := db.Table("Products").Count(&products).Error; err != nil {
		t.Fatalf("count products: %v", err)
	}
	if products != 7 {
		t.Errorf("expected 7 products after serialized reseeds, got %d", products)
	}
}
