package query

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// FTSManager manages Full-Text Search functionality for SQLite and PostgreSQL
type FTSManager struct {
	db           *gorm.DB
	ftsAvailable bool
	ftsVersion   string // "FTS5", "FTS4", "FTS3", "POSTGRES", or ""
	dbDialect    string // "sqlite", "postgres", or other
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

// detectFTS checks if FTS is available and which version/type
func (m *FTSManager) detectFTS() {
	dialector := m.db.Dialector
	m.dbDialect = dialector.Name()

	// Check for PostgreSQL FTS
	if m.dbDialect == "postgres" {
		if m.isPostgresFTSAvailable() {
			m.ftsAvailable = true
			m.ftsVersion = "POSTGRES"
			return
		}
		m.ftsAvailable = false
		m.ftsVersion = ""
		return
	}

	// Check for SQLite FTS
	if m.dbDialect == "sqlite" {
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
	}

	m.ftsAvailable = false
	m.ftsVersion = ""
}

// isFTSVersionAvailable checks if a specific FTS version is available
func (m *FTSManager) isFTSVersionAvailable(version string) bool {
	// Validate version string to prevent SQL injection (defense in depth)
	// Version is called with hardcoded values internally, but validate anyway
	if !isValidSQLIdentifier(version) {
		return false
	}

	// Try to create a temporary FTS table to test availability
	testTableName := fmt.Sprintf("_test_fts_%s", version)
	// Validate the constructed table name as well
	if !isValidSQLIdentifier(testTableName) {
		return false
	}

	err := m.db.Exec(fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS %s USING %s(content)", testTableName, version)).Error
	if err != nil {
		return false
	}

	// Clean up test table - ignore error as cleanup is best-effort
	_ = m.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", testTableName)).Error //nolint:errcheck

	return true
}

// isPostgresFTSAvailable checks if PostgreSQL full-text search is available
func (m *FTSManager) isPostgresFTSAvailable() bool {
	// Test if we can use to_tsvector and to_tsquery functions
	// These are built-in to PostgreSQL and should always be available
	// Use a subquery to make the test more robust
	var result int
	err := m.db.Raw("SELECT 1 FROM (SELECT to_tsvector('english', 'test') @@ to_tsquery('english', 'test') AS matched) AS test WHERE matched").Scan(&result).Error
	// If query executes without error (even if no rows), FTS is available
	if err != nil && err != gorm.ErrRecordNotFound {
		return false
	}

	return true
}

// IsFTSAvailable returns true if FTS is available
func (m *FTSManager) IsFTSAvailable() bool {
	return m.ftsAvailable
}

// GetFTSVersion returns the FTS version available (FTS5, FTS4, FTS3, POSTGRES, or empty string)
func (m *FTSManager) GetFTSVersion() string {
	return m.ftsVersion
}

// ClearFTSCache clears the internal cache of FTS tables
// This is useful after dropping FTS tables (e.g., during database reseeding)
// to ensure the manager will recreate them when needed
func (m *FTSManager) ClearFTSCache() {
	m.ftsTables = make(map[string]bool)
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
			// Use cached column name from metadata
			cols = append(cols, prop.ColumnName)
		}
	}
	return cols
}

// getAllStringColumns returns all string column names
func (m *FTSManager) getAllStringColumns(entityMetadata *metadata.EntityMetadata) []string {
	var cols []string
	for _, prop := range entityMetadata.Properties {
		if prop.Type.String() == "string" && !prop.IsNavigationProp {
			// Use cached column name from metadata
			cols = append(cols, prop.ColumnName)
		}
	}
	return cols
}

// createFTSTable creates the FTS virtual table or index
func (m *FTSManager) createFTSTable(tableName, ftsTableName string, searchableCols []string, entityMetadata *metadata.EntityMetadata) error {
	// Get the key column(s)
	// Note: FTS tables use only the first key property for linking back to the main table.
	// This is sufficient for joining and is consistent with the SQLite FTS implementation.
	// For composite keys, the FTS table uses the first key component as the primary key.
	var keyCol string
	if len(entityMetadata.KeyProperties) > 0 {
		// Use cached column name from metadata
		keyCol = entityMetadata.KeyProperties[0].ColumnName
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

	// Choose implementation based on database type
	if m.ftsVersion == "POSTGRES" {
		return m.createPostgresFTSTable(tableName, ftsTableName, searchableCols, keyCol, entityMetadata)
	}

	return m.createSQLiteFTSTable(tableName, ftsTableName, allCols, keyCol)
}

// createSQLiteFTSTable creates SQLite FTS virtual table
func (m *FTSManager) createSQLiteFTSTable(tableName, ftsTableName string, allCols []string, keyCol string) error {
	// Validate identifiers to prevent SQL injection
	if !isValidSQLIdentifier(tableName) || !isValidSQLIdentifier(ftsTableName) || !isValidSQLIdentifier(keyCol) {
		return fmt.Errorf("invalid SQL identifier in table or column name")
	}
	for _, col := range allCols {
		if !isValidSQLIdentifier(col) {
			return fmt.Errorf("invalid SQL identifier in column: %s", col)
		}
	}

	// Create standalone FTS virtual table (simpler approach)
	// Note: identifiers are validated above and come from internal metadata, not user input
	createSQL := fmt.Sprintf(
		"CREATE VIRTUAL TABLE IF NOT EXISTS %s USING %s(%s)",
		ftsTableName,
		strings.ToLower(m.ftsVersion),
		strings.Join(allCols, ", "),
	)

	err := m.db.Exec(createSQL).Error
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

// createPostgresFTSTable creates PostgreSQL FTS table with tsvector column and GIN index
func (m *FTSManager) createPostgresFTSTable(tableName, ftsTableName string, searchableCols []string, keyCol string, entityMetadata *metadata.EntityMetadata) error {
	// Validate identifiers to prevent SQL injection
	if !isValidSQLIdentifier(tableName) || !isValidSQLIdentifier(ftsTableName) || !isValidSQLIdentifier(keyCol) {
		return fmt.Errorf("invalid SQL identifier in table or column name")
	}
	for _, col := range searchableCols {
		if !isValidSQLIdentifier(col) {
			return fmt.Errorf("invalid SQL identifier in searchable column: %s", col)
		}
	}

	// Determine the PostgreSQL type for the primary key column
	keyType := "INTEGER" // Default fallback
	if len(entityMetadata.KeyProperties) > 0 {
		keyType = goTypeToPostgresType(entityMetadata.KeyProperties[0].Type)
	}

	// Build the column list for the FTS table
	// PostgreSQL FTS uses a tsvector column that combines all searchable fields
	// Note: identifiers are validated above and come from internal metadata, not user input
	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			%s %s PRIMARY KEY,
			search_vector tsvector
		)
	`, ftsTableName, keyCol, keyType)

	err := m.db.Exec(createSQL).Error
	if err != nil {
		return fmt.Errorf("failed to create FTS table: %w", err)
	}

	// Create GIN index on tsvector column for fast full-text search
	indexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s_search_idx 
		ON %s USING GIN(search_vector)
	`, ftsTableName, ftsTableName)

	err = m.db.Exec(indexSQL).Error
	if err != nil {
		return fmt.Errorf("failed to create GIN index: %w", err)
	}

	// Create triggers to keep FTS table in sync with main table
	if err := m.createPostgresFTSTriggers(tableName, ftsTableName, searchableCols, keyCol); err != nil {
		return fmt.Errorf("failed to create FTS triggers: %w", err)
	}

	// Populate existing data into FTS table
	if err := m.populatePostgresFTSTable(tableName, ftsTableName, searchableCols, keyCol); err != nil {
		return fmt.Errorf("failed to populate FTS table: %w", err)
	}

	return nil
}

// createFTSTriggers creates triggers to keep FTS table in sync with the main table
func (m *FTSManager) createFTSTriggers(tableName, ftsTableName string, cols []string, keyCol string) error {
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
		if err := m.db.Exec(trigger).Error; err != nil {
			return fmt.Errorf("failed to create trigger: %w", err)
		}
	}

	return nil
}

// buildUpdateSetClause builds the SET clause for UPDATE statement
func (m *FTSManager) buildUpdateSetClause(cols []string, prefix string) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, fmt.Sprintf("%s = %s%s", col, prefix, col))
	}
	return strings.Join(parts, ", ")
}

// populateFTSTable populates the FTS table with existing data from the main table
func (m *FTSManager) populateFTSTable(tableName, ftsTableName string, cols []string) error {
	colsList := strings.Join(cols, ", ")
	insertSQL := fmt.Sprintf(
		"INSERT INTO %s(%s) SELECT %s FROM %s",
		ftsTableName, colsList, colsList, tableName,
	)

	err := m.db.Exec(insertSQL).Error
	if err != nil {
		return fmt.Errorf("failed to populate FTS table: %w", err)
	}

	return nil
}

// createPostgresFTSTriggers creates PostgreSQL triggers to keep FTS table in sync
func (m *FTSManager) createPostgresFTSTriggers(tableName, ftsTableName string, searchableCols []string, keyCol string) error {
	// Build tsvector expression combining all searchable columns
	tsvectorExpr := m.buildPostgresTSVectorExpr(searchableCols)

	// Create a trigger function for INSERT and UPDATE
	functionSQL := fmt.Sprintf(`
		CREATE OR REPLACE FUNCTION %s_sync() RETURNS TRIGGER AS $$
		BEGIN
			IF TG_OP = 'INSERT' THEN
				INSERT INTO %s (%s, search_vector)
				VALUES (NEW.%s, %s);
				RETURN NEW;
			ELSIF TG_OP = 'UPDATE' THEN
				UPDATE %s SET search_vector = %s WHERE %s = NEW.%s;
				RETURN NEW;
			ELSIF TG_OP = 'DELETE' THEN
				DELETE FROM %s WHERE %s = OLD.%s;
				RETURN OLD;
			END IF;
			RETURN NULL;
		END;
		$$ LANGUAGE plpgsql;
	`, ftsTableName, ftsTableName, keyCol, keyCol, tsvectorExpr,
		ftsTableName, tsvectorExpr, keyCol, keyCol,
		ftsTableName, keyCol, keyCol)

	err := m.db.Exec(functionSQL).Error
	if err != nil {
		return fmt.Errorf("failed to create trigger function: %w", err)
	}

	// Drop existing triggers if they exist (to avoid duplicates)
	dropTriggers := []string{
		fmt.Sprintf("DROP TRIGGER IF EXISTS %s_insert_trigger ON %s", ftsTableName, tableName),
		fmt.Sprintf("DROP TRIGGER IF EXISTS %s_update_trigger ON %s", ftsTableName, tableName),
		fmt.Sprintf("DROP TRIGGER IF EXISTS %s_delete_trigger ON %s", ftsTableName, tableName),
	}

	for _, dropSQL := range dropTriggers {
		// Ignore errors for non-existent triggers - this is expected on first run
		_ = m.db.Exec(dropSQL).Error //nolint:errcheck
	}

	// Create triggers for INSERT, UPDATE, DELETE
	insertTrigger := fmt.Sprintf(`
		CREATE TRIGGER %s_insert_trigger
		AFTER INSERT ON %s
		FOR EACH ROW
		EXECUTE FUNCTION %s_sync()
	`, ftsTableName, tableName, ftsTableName)

	updateTrigger := fmt.Sprintf(`
		CREATE TRIGGER %s_update_trigger
		AFTER UPDATE ON %s
		FOR EACH ROW
		EXECUTE FUNCTION %s_sync()
	`, ftsTableName, tableName, ftsTableName)

	deleteTrigger := fmt.Sprintf(`
		CREATE TRIGGER %s_delete_trigger
		AFTER DELETE ON %s
		FOR EACH ROW
		EXECUTE FUNCTION %s_sync()
	`, ftsTableName, tableName, ftsTableName)

	triggers := []string{insertTrigger, updateTrigger, deleteTrigger}
	for _, trigger := range triggers {
		if err := m.db.Exec(trigger).Error; err != nil {
			return fmt.Errorf("failed to create trigger: %w", err)
		}
	}

	return nil
}

// buildPostgresTSVectorExpr builds a PostgreSQL tsvector expression from column names
func (m *FTSManager) buildPostgresTSVectorExpr(cols []string) string {
	// Validate identifiers for defense in depth
	for _, col := range cols {
		if !isValidSQLIdentifier(col) {
			// Return empty string if validation fails - caller will handle the error
			return ""
		}
	}

	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		// Use coalesce to handle NULL values
		parts = append(parts, fmt.Sprintf("to_tsvector('english', coalesce(NEW.%s, ''))", col))
	}
	// PostgreSQL's || operator for tsvectors correctly merges them without needing a space literal
	return strings.Join(parts, " || ")
}

// populatePostgresFTSTable populates the PostgreSQL FTS table with existing data
func (m *FTSManager) populatePostgresFTSTable(tableName, ftsTableName string, searchableCols []string, keyCol string) error {
	// Validate identifiers for defense in depth
	if !isValidSQLIdentifier(tableName) || !isValidSQLIdentifier(ftsTableName) || !isValidSQLIdentifier(keyCol) {
		return fmt.Errorf("invalid SQL identifier in table or column name")
	}
	for _, col := range searchableCols {
		if !isValidSQLIdentifier(col) {
			return fmt.Errorf("invalid SQL identifier in searchable column: %s", col)
		}
	}

	// Build tsvector expression for existing data
	tsvectorParts := make([]string, 0, len(searchableCols))
	for _, col := range searchableCols {
		tsvectorParts = append(tsvectorParts, fmt.Sprintf("to_tsvector('english', coalesce(%s, ''))", col))
	}
	// PostgreSQL's || operator for tsvectors correctly merges them without needing a space literal
	tsvectorExpr := strings.Join(tsvectorParts, " || ")

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s, search_vector) SELECT %s, %s FROM %s",
		ftsTableName, keyCol, keyCol, tsvectorExpr, tableName,
	)

	err := m.db.Exec(insertSQL).Error
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

	// Validate inputs
	if entityMetadata == nil {
		return db, fmt.Errorf("entity metadata is required")
	}

	if tableName == "" {
		return db, fmt.Errorf("table name is required")
	}

	if len(entityMetadata.KeyProperties) == 0 {
		return db, fmt.Errorf("entity has no key properties")
	}

	// Ensure FTS table exists
	if err := m.EnsureFTSTable(tableName, entityMetadata); err != nil {
		return db, err
	}

	ftsTableName := m.getFTSTableName(tableName)
	// Use cached column name from metadata
	keyCol := entityMetadata.KeyProperties[0].ColumnName

	// Validate identifiers to prevent SQL injection
	if !isValidSQLIdentifier(tableName) || !isValidSQLIdentifier(ftsTableName) || !isValidSQLIdentifier(keyCol) {
		return db, fmt.Errorf("invalid SQL identifier in table or column name")
	}

	// Apply FTS search using JOIN
	// The FTS table is queried and then joined with the main table
	// Note: table and column names are validated above and come from internal metadata,
	// not user input, so string formatting is safe here
	db = db.Joins(fmt.Sprintf(
		"INNER JOIN %s ON %s.%s = %s.%s",
		ftsTableName, tableName, keyCol, ftsTableName, keyCol,
	))

	// Apply search condition based on database type
	// Search query is passed as a parameterized value, not interpolated
	switch m.ftsVersion {
	case "POSTGRES":
		// PostgreSQL uses @@ operator with plainto_tsquery for simple text search
		// plainto_tsquery handles normalization and removes special characters
		// No manual escaping needed - parameterized query handles it safely
		db = db.Where(fmt.Sprintf("%s.search_vector @@ plainto_tsquery('english', ?)", ftsTableName), searchQuery)
	case "FTS5":
		// For FTS5, use the MATCH operator
		db = db.Where(fmt.Sprintf("%s MATCH ?", ftsTableName), searchQuery)
	default:
		// For FTS4/FTS3, also use MATCH
		db = db.Where(fmt.Sprintf("%s MATCH ?", ftsTableName), searchQuery)
	}

	return db, nil
}

// isValidSQLIdentifier validates that a string is a safe SQL identifier
// It checks that the identifier only contains alphanumeric characters and underscores
func isValidSQLIdentifier(identifier string) bool {
	if identifier == "" {
		return false
	}
	for _, ch := range identifier {
		// Apply De Morgan's law for simpler logic
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
			return false
		}
	}
	return true
}

// goTypeToPostgresType maps Go types to PostgreSQL column types
func goTypeToPostgresType(t reflect.Type) string {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int32:
		return "INTEGER"
	case reflect.Int8:
		return "SMALLINT"
	case reflect.Int16:
		return "SMALLINT"
	case reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint32:
		return "BIGINT"
	case reflect.Uint8:
		return "SMALLINT"
	case reflect.Uint16:
		return "INTEGER"
	case reflect.Uint64:
		return "BIGINT"
	case reflect.String:
		return "TEXT"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		return "DOUBLE PRECISION"
	case reflect.Bool:
		return "BOOLEAN"
	default:
		// Check for UUID types by name
		typeName := t.String()
		if strings.Contains(typeName, "UUID") || strings.Contains(typeName, "uuid") {
			return "UUID"
		}
		// Default to TEXT for unknown types
		return "TEXT"
	}
}
