package query

import (
	"os"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresFTSTestEntity represents a test entity for PostgreSQL FTS testing
type PostgresFTSTestEntity struct {
	ID          int    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string `json:"Name" odata:"searchable"`
	Description string `json:"Description" odata:"searchable"`
	Category    string `json:"Category"`
}

// getPostgresDB creates a test database connection for PostgreSQL
// Returns nil if PostgreSQL is not available (e.g., in CI without postgres)
func getPostgresDB(t *testing.T) *gorm.DB {
	// Try to get DSN from environment variable
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		// Default test DSN
		dsn = "postgresql://postgres:postgres@localhost:5432/odata_test?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skip("PostgreSQL not available, skipping test:", err)
		return nil
	}

	return db
}

func TestPostgresFTSManager_DetectFTS(t *testing.T) {
	db := getPostgresDB(t)
	if db == nil {
		return
	}

	manager := NewFTSManager(db)

	if !manager.IsFTSAvailable() {
		t.Error("Expected PostgreSQL FTS to be available")
	}

	version := manager.GetFTSVersion()
	if version != "POSTGRES" {
		t.Errorf("Expected FTS version 'POSTGRES', got '%s'", version)
	}

	t.Logf("Detected FTS version: %s", version)
}

func TestPostgresFTSManager_EnsureFTSTable(t *testing.T) {
	db := getPostgresDB(t)
	if db == nil {
		return
	}

	// Clean up before test
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities CASCADE")

	// Auto-migrate test entity
	if err := db.AutoMigrate(&PostgresFTSTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	manager := NewFTSManager(db)
	if !manager.IsFTSAvailable() {
		t.Skip("PostgreSQL FTS not available, skipping test")
	}

	meta, err := metadata.AnalyzeEntity(PostgresFTSTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Ensure FTS table is created
	err = manager.EnsureFTSTable("postgres_fts_test_entities", meta)
	if err != nil {
		t.Errorf("Failed to ensure FTS table: %v", err)
	}

	// Verify FTS table was created
	var count int64
	result := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'postgres_fts_test_entities_fts'").Scan(&count)
	if result.Error != nil {
		t.Fatalf("Failed to check FTS table: %v", result.Error)
	}

	if count != 1 {
		t.Error("Expected FTS table to be created")
	}

	// Verify search_vector column exists
	result = db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'postgres_fts_test_entities_fts' AND column_name = 'search_vector'").Scan(&count)
	if result.Error != nil {
		t.Fatalf("Failed to check search_vector column: %v", result.Error)
	}

	if count != 1 {
		t.Error("Expected search_vector column to exist")
	}

	// Clean up after test
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities CASCADE")
}

func TestPostgresFTSManager_ApplyFTSSearch(t *testing.T) {
	db := getPostgresDB(t)
	if db == nil {
		return
	}

	// Clean up before test
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities CASCADE")

	// Auto-migrate test entity
	if err := db.AutoMigrate(&PostgresFTSTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	testData := []PostgresFTSTestEntity{
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
		t.Skip("PostgreSQL FTS not available, skipping test")
	}

	meta, err := metadata.AnalyzeEntity(PostgresFTSTestEntity{})
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
			name:          "Search for 'keyboard'",
			searchQuery:   "keyboard",
			expectedCount: 1,
			description:   "Should find keyboard",
		},
		{
			name:          "Search for empty string",
			searchQuery:   "",
			expectedCount: 4,
			description:   "Should return all results",
		},
		{
			name:          "Search for multiple words",
			searchQuery:   "mechanical keyboard",
			expectedCount: 1,
			description:   "Should find mechanical keyboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := db.Table("postgres_fts_test_entities")

			if tt.searchQuery != "" {
				query, err = manager.ApplyFTSSearch(query, "postgres_fts_test_entities", tt.searchQuery, meta)
				if err != nil {
					t.Fatalf("Failed to apply FTS search: %v", err)
				}
			}

			var results []PostgresFTSTestEntity
			if err := query.Find(&results).Error; err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("%s: Expected %d results, got %d", tt.description, tt.expectedCount, len(results))
			}
		})
	}

	// Clean up after test
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities CASCADE")
}

func TestPostgresFTSManager_CaseInsensitive(t *testing.T) {
	db := getPostgresDB(t)
	if db == nil {
		return
	}

	// Clean up before test
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities CASCADE")

	// Auto-migrate test entity
	if err := db.AutoMigrate(&PostgresFTSTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data with mixed case
	testData := []PostgresFTSTestEntity{
		{ID: 1, Name: "LAPTOP PRO", Description: "High-performance LAPTOP", Category: "Electronics"},
		{ID: 2, Name: "desktop computer", Description: "powerful DESKTOP", Category: "Electronics"},
	}
	if err := db.Create(&testData).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	manager := NewFTSManager(db)
	if !manager.IsFTSAvailable() {
		t.Skip("PostgreSQL FTS not available, skipping test")
	}

	meta, err := metadata.AnalyzeEntity(PostgresFTSTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Test case-insensitive search
	query := db.Table("postgres_fts_test_entities")
	query, err = manager.ApplyFTSSearch(query, "postgres_fts_test_entities", "laptop", meta)
	if err != nil {
		t.Fatalf("Failed to apply FTS search: %v", err)
	}

	var results []PostgresFTSTestEntity
	if err := query.Find(&results).Error; err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	// Should find the LAPTOP entry with lowercase search
	if len(results) != 1 {
		t.Errorf("Expected 1 result for case-insensitive search, got %d", len(results))
	}

	if len(results) > 0 && results[0].ID != 1 {
		t.Errorf("Expected to find entry with ID 1, got %d", results[0].ID)
	}

	// Clean up after test
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities CASCADE")
}

func TestPostgresFTSManager_TriggersSync(t *testing.T) {
	db := getPostgresDB(t)
	if db == nil {
		return
	}

	// Clean up before test
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities CASCADE")

	// Auto-migrate test entity
	if err := db.AutoMigrate(&PostgresFTSTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	manager := NewFTSManager(db)
	if !manager.IsFTSAvailable() {
		t.Skip("PostgreSQL FTS not available, skipping test")
	}

	meta, err := metadata.AnalyzeEntity(PostgresFTSTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Ensure FTS table is created
	err = manager.EnsureFTSTable("postgres_fts_test_entities", meta)
	if err != nil {
		t.Fatalf("Failed to ensure FTS table: %v", err)
	}

	// Insert a new record
	newEntity := PostgresFTSTestEntity{ID: 1, Name: "Test Product", Description: "Test description", Category: "Test"}
	if err := db.Create(&newEntity).Error; err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Verify FTS table was updated via trigger
	var count int64
	result := db.Raw("SELECT COUNT(*) FROM postgres_fts_test_entities_fts WHERE id = 1").Scan(&count)
	if result.Error != nil {
		t.Fatalf("Failed to check FTS table: %v", result.Error)
	}

	if count != 1 {
		t.Error("Expected FTS table to be updated via INSERT trigger")
	}

	// Update the record
	db.Model(&newEntity).Update("Description", "Updated description")

	// Search for the updated text
	query := db.Table("postgres_fts_test_entities")
	query, err = manager.ApplyFTSSearch(query, "postgres_fts_test_entities", "updated", meta)
	if err != nil {
		t.Fatalf("Failed to apply FTS search: %v", err)
	}

	var results []PostgresFTSTestEntity
	if err := query.Find(&results).Error; err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result after UPDATE trigger, got %d", len(results))
	}

	// Delete the record
	db.Delete(&newEntity)

	// Verify FTS table entry was deleted via trigger
	result = db.Raw("SELECT COUNT(*) FROM postgres_fts_test_entities_fts WHERE id = 1").Scan(&count)
	if result.Error != nil {
		t.Fatalf("Failed to check FTS table: %v", result.Error)
	}

	if count != 0 {
		t.Error("Expected FTS table entry to be deleted via DELETE trigger")
	}

	// Clean up after test
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities_fts CASCADE")
	db.Exec("DROP TABLE IF EXISTS postgres_fts_test_entities CASCADE")
}
