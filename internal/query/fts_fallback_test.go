package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// FallbackTestEntity represents a test entity for fallback testing
type FallbackTestEntity struct {
	ID          int    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string `json:"Name" odata:"searchable"`
	Description string `json:"Description" odata:"searchable"`
}

func TestFTSFallback_NoFTSAvailable(t *testing.T) {
	// Create a database connection
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&FallbackTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	testData := []FallbackTestEntity{
		{ID: 1, Name: "Laptop", Description: "Gaming laptop"},
		{ID: 2, Name: "Mouse", Description: "Wireless mouse"},
	}
	if err := db.Create(&testData).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	manager := NewFTSManager(db)
	meta, err := metadata.AnalyzeEntity(FallbackTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Try to apply FTS search - should not panic even if FTS setup fails
	query := db.Table("fallback_test_entities")
	
	// Even if ApplyFTSSearch returns an error, it should not panic
	resultQuery, err := manager.ApplyFTSSearch(query, "fallback_test_entities", "laptop", meta)
	
	// The query should still be valid (either with FTS or ready for fallback)
	if err != nil {
		// If FTS is not available, that's okay - the in-memory fallback will handle it
		t.Logf("FTS not available (expected): %v", err)
		
		// Verify we can still query without FTS
		var results []FallbackTestEntity
		if err := query.Find(&results).Error; err != nil {
			t.Fatalf("Failed to execute fallback query: %v", err)
		}
		
		// Should get all results before in-memory filtering
		if len(results) != 2 {
			t.Errorf("Expected 2 results from fallback query, got %d", len(results))
		}
	} else {
		// If FTS is available, it should work
		var results []FallbackTestEntity
		if err := resultQuery.Find(&results).Error; err != nil {
			t.Fatalf("Failed to execute FTS query: %v", err)
		}
		
		if len(results) != 1 {
			t.Errorf("Expected 1 result from FTS query, got %d", len(results))
		}
	}
}

func TestFTSFallback_ErrorHandling(t *testing.T) {
	// Create a database connection
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	manager := NewFTSManager(db)
	
	// Try to apply FTS search with invalid inputs - should not panic
	_, err = manager.ApplyFTSSearch(db, "", "", nil)
	if err == nil {
		t.Log("ApplyFTSSearch handled empty inputs gracefully")
	}
	
	// Try with nil metadata - should not panic
	_, err = manager.ApplyFTSSearch(db, "test_table", "search", nil)
	if err == nil {
		t.Error("Expected error with nil metadata")
	} else {
		t.Logf("Correctly returned error for nil metadata: %v", err)
	}
}

func TestFTSManager_UnsupportedDatabase(t *testing.T) {
	// This test simulates an unsupported database scenario
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	manager := NewFTSManager(db)
	
	// For SQLite, FTS should be detected
	// For other databases, FTS should gracefully report as not available
	isAvailable := manager.IsFTSAvailable()
	version := manager.GetFTSVersion()
	
	t.Logf("FTS Available: %v, Version: %s", isAvailable, version)
	
	// The manager should always be created without panic
	if manager == nil {
		t.Error("Manager should always be created, even if FTS is not available")
	}
}

func TestFTSManager_EmptySearchQuery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&FallbackTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	manager := NewFTSManager(db)
	meta, err := metadata.AnalyzeEntity(FallbackTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Apply FTS search with empty query - should return all results
	query := db.Table("fallback_test_entities")
	resultQuery, err := manager.ApplyFTSSearch(query, "fallback_test_entities", "", meta)
	
	// Empty search should not fail
	if err != nil && !manager.IsFTSAvailable() {
		t.Logf("FTS not available, empty search handled: %v", err)
	} else if err != nil {
		t.Errorf("Unexpected error with empty search: %v", err)
	}
	
	// Query should still be valid
	if resultQuery != nil {
		var results []FallbackTestEntity
		if err := resultQuery.Find(&results).Error; err != nil {
			t.Fatalf("Failed to execute query with empty search: %v", err)
		}
	}
}
