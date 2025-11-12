package query

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// FTSManager manages SQLite Full-Text Search functionality
type FTSManager struct {
	db           *gorm.DB
	ftsAvailable bool
	ftsVersion   string // "FTS5", "FTS4", "FTS3", or ""
	ftsTables    map[string]bool
}

// NewFTSManager creates a new FTS manager and detects FTS availability
func NewFTSManager(db *gorm.DB) *FTSManager {
	manager := &FTSManager{
		db:        db,
		ftsTables: make(map[string]bool),
	}
	manager.detectFTS()
	return manager
}

// detectFTS checks if SQLite FTS is available and which version
func (m *FTSManager) detectFTS() {
	// Check if we're using SQLite
	if m.db.Dialector.Name() != "sqlite" {
		return
	}

	// Try to detect FTS5 first (preferred)
	if m.isFTSVersionAvailable("fts5") {
		m.ftsAvailable = true
		m.ftsVersion = "FTS5"
		return
	}

	// Fall back to FTS4
	if m.isFTSVersionAvailable("fts4") {
		m.ftsAvailable = true
		m.ftsVersion = "FTS4"
		return
	}

	// Fall back to FTS3
	if m.isFTSVersionAvailable("fts3") {
		m.ftsAvailable = true
		m.ftsVersion = "FTS3"
		return
	}

	m.ftsAvailable = false
	m.ftsVersion = ""
}

// isFTSVersionAvailable checks if a specific FTS version is available
func (m *FTSManager) isFTSVersionAvailable(version string) bool {
	var sqlDB *sql.DB
	var err error

	sqlDB, err = m.db.DB()
	if err != nil {
		return false
	}

	// Try to create a temporary FTS table to test availability
	testTableName := fmt.Sprintf("_test_fts_%s", version)
	_, err = sqlDB.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS %s USING %s(content)", testTableName, version))
	if err != nil {
		return false
	}

	// Clean up test table
	_, _ = sqlDB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", testTableName))

	return true
}

// IsFTSAvailable returns true if FTS is available
func (m *FTSManager) IsFTSAvailable() bool {
	return m.ftsAvailable
}

// GetFTSVersion returns the FTS version available (FTS5, FTS4, FTS3, or empty string)
func (m *FTSManager) GetFTSVersion() string {
	return m.ftsVersion
}

// EnsureFTSTable creates an FTS table for the given entity if it doesn't exist
func (m *FTSManager) EnsureFTSTable(tableName string, entityMetadata *metadata.EntityMetadata) error {
	if !m.ftsAvailable {
		return fmt.Errorf("FTS is not available")
	}

	// Check if FTS table already exists
	ftsTableName := m.getFTSTableName(tableName)
	if m.ftsTables[ftsTableName] {
		return nil
	}

	// Get searchable columns
	searchableCols := m.getSearchableColumns(entityMetadata)
	if len(searchableCols) == 0 {
		// If no searchable columns defined, use all string columns
		searchableCols = m.getAllStringColumns(entityMetadata)
	}

	if len(searchableCols) == 0 {
		return fmt.Errorf("no searchable columns found for entity %s", tableName)
	}

	// Create FTS virtual table
	if err := m.createFTSTable(tableName, ftsTableName, searchableCols, entityMetadata); err != nil {
		return err
	}

	m.ftsTables[ftsTableName] = true
	return nil
}

// getFTSTableName returns the FTS virtual table name for a given table
func (m *FTSManager) getFTSTableName(tableName string) string {
	return fmt.Sprintf("%s_fts", tableName)
}

// getSearchableColumns returns column names marked as searchable
func (m *FTSManager) getSearchableColumns(entityMetadata *metadata.EntityMetadata) []string {
	var cols []string
	for _, prop := range entityMetadata.Properties {
		if prop.IsSearchable && !prop.IsNavigationProp {
			cols = append(cols, toSnakeCase(prop.Name))
		}
	}
	return cols
}

// getAllStringColumns returns all string column names
func (m *FTSManager) getAllStringColumns(entityMetadata *metadata.EntityMetadata) []string {
	var cols []string
	for _, prop := range entityMetadata.Properties {
		if prop.Type.String() == "string" && !prop.IsNavigationProp {
			cols = append(cols, toSnakeCase(prop.Name))
		}
	}
	return cols
}

// createFTSTable creates the FTS virtual table
func (m *FTSManager) createFTSTable(tableName, ftsTableName string, searchableCols []string, entityMetadata *metadata.EntityMetadata) error {
	var sqlDB *sql.DB
	var err error

	sqlDB, err = m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Get the key column(s)
	var keyCol string
	if len(entityMetadata.KeyProperties) > 0 {
		keyCol = toSnakeCase(entityMetadata.KeyProperties[0].Name)
	} else {
		return fmt.Errorf("entity has no key properties")
	}

	// Build the column list including the key column
	allCols := []string{keyCol}
	for _, col := range searchableCols {
		if col != keyCol {
			allCols = append(allCols, col)
		}
	}

	// Create standalone FTS virtual table (simpler approach)
	var createSQL string
	createSQL = fmt.Sprintf(
		"CREATE VIRTUAL TABLE IF NOT EXISTS %s USING %s(%s)",
		ftsTableName,
		strings.ToLower(m.ftsVersion),
		strings.Join(allCols, ", "),
	)

	_, err = sqlDB.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("failed to create FTS table: %w", err)
	}

	// Create triggers to keep FTS table in sync
	if err := m.createFTSTriggers(tableName, ftsTableName, allCols, keyCol); err != nil {
		return fmt.Errorf("failed to create FTS triggers: %w", err)
	}

	// Populate existing data into FTS table
	if err := m.populateFTSTable(tableName, ftsTableName, allCols); err != nil {
		return fmt.Errorf("failed to populate FTS table: %w", err)
	}

	return nil
}

// createFTSTriggers creates triggers to keep FTS table in sync with the main table
func (m *FTSManager) createFTSTriggers(tableName, ftsTableName string, cols []string, keyCol string) error {
	var sqlDB *sql.DB
	var err error

	sqlDB, err = m.db.DB()
	if err != nil {
		return err
	}

	colsList := strings.Join(cols, ", ")
	newColsList := "NEW." + strings.Join(cols, ", NEW.")

	// Insert trigger
	insertTrigger := fmt.Sprintf(`
		CREATE TRIGGER IF NOT EXISTS %s_ai AFTER INSERT ON %s BEGIN
			INSERT INTO %s(%s) VALUES (%s);
		END;
	`, ftsTableName, tableName, ftsTableName, colsList, newColsList)

	// Delete trigger - use the key column to match rows
	deleteTrigger := fmt.Sprintf(`
		CREATE TRIGGER IF NOT EXISTS %s_ad AFTER DELETE ON %s BEGIN
			DELETE FROM %s WHERE %s = OLD.%s;
		END;
	`, ftsTableName, tableName, ftsTableName, keyCol, keyCol)

	// Update trigger
	updateTrigger := fmt.Sprintf(`
		CREATE TRIGGER IF NOT EXISTS %s_au AFTER UPDATE ON %s BEGIN
			UPDATE %s SET %s WHERE %s = OLD.%s;
		END;
	`, ftsTableName, tableName, ftsTableName, m.buildUpdateSetClause(cols, "NEW."), keyCol, keyCol)

	triggers := []string{insertTrigger, deleteTrigger, updateTrigger}
	for _, trigger := range triggers {
		if _, err := sqlDB.Exec(trigger); err != nil {
			return fmt.Errorf("failed to create trigger: %w", err)
		}
	}

	return nil
}

// buildUpdateSetClause builds the SET clause for UPDATE statement
func (m *FTSManager) buildUpdateSetClause(cols []string, prefix string) string {
	var parts []string
	for _, col := range cols {
		parts = append(parts, fmt.Sprintf("%s = %s%s", col, prefix, col))
	}
	return strings.Join(parts, ", ")
}

// populateFTSTable populates the FTS table with existing data from the main table
func (m *FTSManager) populateFTSTable(tableName, ftsTableName string, cols []string) error {
	var sqlDB *sql.DB
	var err error

	sqlDB, err = m.db.DB()
	if err != nil {
		return err
	}

	colsList := strings.Join(cols, ", ")
	insertSQL := fmt.Sprintf(
		"INSERT INTO %s(%s) SELECT %s FROM %s",
		ftsTableName, colsList, colsList, tableName,
	)

	_, err = sqlDB.Exec(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to populate FTS table: %w", err)
	}

	return nil
}

// ApplyFTSSearch applies FTS search to a GORM query
func (m *FTSManager) ApplyFTSSearch(db *gorm.DB, tableName string, searchQuery string, entityMetadata *metadata.EntityMetadata) (*gorm.DB, error) {
	if !m.ftsAvailable {
		return db, fmt.Errorf("FTS is not available")
	}

	if searchQuery == "" {
		return db, nil
	}

	// Ensure FTS table exists
	if err := m.EnsureFTSTable(tableName, entityMetadata); err != nil {
		return db, err
	}

	ftsTableName := m.getFTSTableName(tableName)
	keyCol := toSnakeCase(entityMetadata.KeyProperties[0].Name)

	// Escape single quotes in search query
	escapedQuery := strings.ReplaceAll(searchQuery, "'", "''")

	// Apply FTS search using JOIN
	// The FTS table is queried and then joined with the main table
	db = db.Joins(fmt.Sprintf(
		"INNER JOIN %s ON %s.%s = %s.%s",
		ftsTableName, tableName, keyCol, ftsTableName, keyCol,
	))

	// For FTS5, use the MATCH operator
	if m.ftsVersion == "FTS5" {
		db = db.Where(fmt.Sprintf("%s MATCH ?", ftsTableName), escapedQuery)
	} else {
		// For FTS4/FTS3, also use MATCH
		db = db.Where(fmt.Sprintf("%s MATCH ?", ftsTableName), escapedQuery)
	}

	return db, nil
}
