package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// FTSTestEntity represents a test entity for FTS testing
type FTSTestEntity struct {
	ID          int    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string `json:"Name" odata:"searchable"`
	Description string `json:"Description" odata:"searchable"`
	Category    string `json:"Category"`
}

func TestFTSManager_DetectFTS(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	manager := NewFTSManager(db)

	if !manager.IsFTSAvailable() {
		t.Error("Expected FTS to be available in SQLite")
	}

	version := manager.GetFTSVersion()
	if version == "" {
		t.Error("Expected FTS version to be detected")
	}

	t.Logf("Detected FTS version: %s", version)
}

func TestFTSManager_EnsureFTSTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate test entity
	if err := db.AutoMigrate(&FTSTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	manager := NewFTSManager(db)
	if !manager.IsFTSAvailable() {
		t.Skip("FTS not available, skipping test")
	}

	meta, err := metadata.AnalyzeEntity(FTSTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Ensure FTS table is created
	err = manager.EnsureFTSTable("fts_test_entities", meta)
	if err != nil {
		t.Errorf("Failed to ensure FTS table: %v", err)
	}

	// Verify FTS table was created
	var count int64
	sqlDB, _ := db.DB()
	row := sqlDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='fts_test_entities_fts'")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to check FTS table: %v", err)
	}

	if count != 1 {
		t.Error("Expected FTS table to be created")
	}
}

func TestFTSManager_ApplyFTSSearch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate test entity
	if err := db.AutoMigrate(&FTSTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	testData := []FTSTestEntity{
		{ID: 1, Name: "Laptop Pro", Description: "High-performance laptop", Category: "Electronics"},
		{ID: 2, Name: "Desktop Computer", Description: "Powerful desktop for gaming", Category: "Electronics"},
		{ID: 3, Name: "Wireless Mouse", Description: "Ergonomic wireless mouse", Category: "Accessories"},
		{ID: 4, Name: "Keyboard", Description: "Mechanical keyboard with RGB", Category: "Accessories"},
	}
	if err := db.Create(&testData).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	manager := NewFTSManager(db)
	if !manager.IsFTSAvailable() {
		t.Skip("FTS not available, skipping test")
	}

	meta, err := metadata.AnalyzeEntity(FTSTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	tests := []struct {
		name          string
		searchQuery   string
		expectedCount int
		description   string
	}{
		{
			name:          "Search for 'laptop'",
			searchQuery:   "laptop",
			expectedCount: 1,
			description:   "Should find laptop",
		},
		{
			name:          "Search for 'gaming'",
			searchQuery:   "gaming",
			expectedCount: 1,
			description:   "Should find desktop with gaming in description",
		},
		{
			name:          "Search for 'wireless'",
			searchQuery:   "wireless",
			expectedCount: 1,
			description:   "Should find wireless mouse",
		},
		{
			name:          "Search for empty string",
			searchQuery:   "",
			expectedCount: 4,
			description:   "Should return all results",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := db.Table("fts_test_entities")

			if tt.searchQuery != "" {
				query, err = manager.ApplyFTSSearch(query, "fts_test_entities", tt.searchQuery, meta)
				if err != nil {
					t.Fatalf("Failed to apply FTS search: %v", err)
				}
			}

			var results []FTSTestEntity
			if err := query.Find(&results).Error; err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("%s: Expected %d results, got %d", tt.description, tt.expectedCount, len(results))
			}
		})
	}
}

func TestFTSManager_NonSQLiteDialector(t *testing.T) {
	// This test verifies that FTS manager handles non-SQLite databases gracefully
	// We can't easily test with a different database here, so we'll just verify
	// that the manager initializes without panic

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	manager := NewFTSManager(db)

	// Should not panic
	if manager == nil {
		t.Error("Expected manager to be created")
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"PascalCase", "pascal_case"},
		{"camelCase", "camel_case"},
		{"snake_case", "snake_case"},
		{"ID", "id"},                  // Consecutive uppercase letters stay together
		{"HTTPServer", "http_server"}, // Consecutive uppercase followed by lowercase
		{"Name", "name"},
		{"Description", "description"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
