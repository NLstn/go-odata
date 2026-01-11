package odata

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// EnableGeospatial enables geospatial features for the service and validates that
// the underlying database supports spatial operations. This method must be called
// during service startup before handling requests if geospatial features are needed.
//
// If the database does not support the required spatial features, the service will
// panic with a detailed error message indicating what extension, configuration, or
// dependency is missing.
//
// Example:
//
//	service := odata.NewService(db)
//	service.EnableGeospatial() // Validates database support and panics if not available
//	service.RegisterEntity(Product{}, "Products")
func (s *Service) EnableGeospatial() {
	s.logger.Info("Enabling geospatial features")

	// Check if database supports geospatial features
	if err := checkGeospatialSupport(s.db, s.logger); err != nil {
		s.logger.Error("Failed to enable geospatial features", "error", err)
		panic(fmt.Sprintf("geospatial features cannot be enabled: %v", err))
	}

	s.geospatialEnabled = true
	s.logger.Info("Geospatial features enabled successfully")
}

// IsGeospatialEnabled returns whether geospatial features are enabled for this service
func (s *Service) IsGeospatialEnabled() bool {
	return s.geospatialEnabled
}

// checkGeospatialSupport validates that the database supports geospatial operations
// and returns a detailed error message if support is missing
func checkGeospatialSupport(db *gorm.DB, logger *slog.Logger) error {
	if db == nil || db.Dialector == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	dialect := db.Dialector.Name()
	logger.Info("Checking geospatial support", "dialect", dialect)

	switch dialect {
	case "sqlite", "sqlite3":
		return checkSQLiteGeospatialSupport(db, logger)
	case "postgres", "postgresql":
		return checkPostgreSQLGeospatialSupport(db, logger)
	case "mysql":
		return checkMySQLGeospatialSupport(db, logger)
	case "sqlserver", "mssql":
		return checkSQLServerGeospatialSupport(db, logger)
	default:
		return fmt.Errorf("database dialect '%s' is not supported for geospatial features", dialect)
	}
}

// checkSQLiteGeospatialSupport checks if SQLite has SpatiaLite extension loaded
func checkSQLiteGeospatialSupport(db *gorm.DB, logger *slog.Logger) error {
	// Try to load SpatiaLite extension
	var result int
	err := db.Raw("SELECT load_extension('mod_spatialite')").Scan(&result).Error
	if err == nil {
		logger.Info("SpatiaLite extension loaded successfully")
		return nil
	}

	// Try alternative loading methods
	err = db.Exec("SELECT InitSpatialMetaData(1)").Error
	if err == nil {
		logger.Info("SpatiaLite already initialized")
		return nil
	}

	// SpatiaLite is not available
	return fmt.Errorf(`SQLite does not have SpatiaLite extension support enabled.

To enable geospatial support in SQLite, you need to:
1. Install SpatiaLite extension:
   - On Ubuntu/Debian: sudo apt-get install libsqlite3-mod-spatialite
   - On macOS with Homebrew: brew install libspatialite
   - On Windows: Download from https://www.gaia-gis.it/fossil/libspatialite/index

2. Load the extension in your Go application before creating the service:
   import _ "github.com/mattn/go-sqlite3"
   
   db, err := gorm.Open(sqlite.Open("file.db?_loc=auto"), &gorm.Config{})
   db.Exec("SELECT load_extension('mod_spatialite')")

3. Alternatively, compile SQLite with SpatiaLite statically linked

Error details: %v`, err)
}

// checkPostgreSQLGeospatialSupport checks if PostgreSQL has PostGIS extension installed
func checkPostgreSQLGeospatialSupport(db *gorm.DB, logger *slog.Logger) error {
	var exists bool
	err := db.Raw("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'postgis')").Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("failed to check for PostGIS extension: %w\n\nTo check manually, run: SELECT * FROM pg_extension WHERE extname = 'postgis';", err)
	}

	if !exists {
		return fmt.Errorf(`PostgreSQL does not have PostGIS extension installed.

To enable geospatial support in PostgreSQL, you need to:
1. Install PostGIS:
   - On Ubuntu/Debian: sudo apt-get install postgresql-{version}-postgis-3
   - On macOS with Homebrew: brew install postgis
   - On Windows: Install via PostgreSQL installer or EDB packages

2. Enable the extension in your database:
   psql -d your_database -c "CREATE EXTENSION postgis;"

3. Verify installation:
   psql -d your_database -c "SELECT PostGIS_version();"

For more information, visit: https://postgis.net/install/`)
	}

	// Verify PostGIS functions are available
	var version string
	err = db.Raw("SELECT PostGIS_version()").Scan(&version).Error
	if err != nil {
		return fmt.Errorf("PostGIS extension is installed but not functioning correctly: %w", err)
	}

	logger.Info("PostGIS extension available", "version", version)
	return nil
}

// checkMySQLGeospatialSupport checks if MySQL/MariaDB has spatial function support
func checkMySQLGeospatialSupport(db *gorm.DB, logger *slog.Logger) error {
	// MySQL 5.7+ and MariaDB 10.2+ have built-in spatial support
	// Test if ST_Distance function is available
	var result interface{}
	err := db.Raw("SELECT ST_Distance(POINT(0, 0), POINT(1, 1))").Scan(&result).Error
	if err != nil {
		return fmt.Errorf(`MySQL/MariaDB spatial functions are not available.

To enable geospatial support in MySQL/MariaDB:
1. Ensure you are using:
   - MySQL 5.7 or later (spatial functions are built-in)
   - MariaDB 10.2 or later (spatial functions are built-in)

2. Verify your version:
   mysql -e "SELECT VERSION();"

3. Test spatial functions:
   mysql -e "SELECT ST_Distance(POINT(0, 0), POINT(1, 1));"

4. If using an older version, upgrade to a supported version

Error details: %v`, err)
	}

	logger.Info("MySQL/MariaDB spatial functions available")
	return nil
}

// checkSQLServerGeospatialSupport checks if SQL Server has spatial type support
func checkSQLServerGeospatialSupport(db *gorm.DB, logger *slog.Logger) error {
	// SQL Server 2008+ has built-in spatial support with geography and geometry types
	// Test if spatial types are available
	var result int
	err := db.Raw("SELECT geography::Point(0, 0, 4326).STDistance(geography::Point(1, 1, 4326))").Scan(&result).Error
	if err != nil {
		return fmt.Errorf(`SQL Server spatial types are not available.

To enable geospatial support in SQL Server:
1. Ensure you are using SQL Server 2008 or later (spatial types are built-in)

2. Verify spatial types are available:
   SELECT geography::Point(0, 0, 4326).STDistance(geography::Point(1, 1, 4326));

3. Ensure your database compatibility level is set to 100 or higher:
   SELECT compatibility_level FROM sys.databases WHERE name = 'your_database';
   ALTER DATABASE your_database SET COMPATIBILITY_LEVEL = 100;

4. If using SQL Server Express or an older version, upgrade to a supported version

For more information, visit: https://docs.microsoft.com/en-us/sql/relational-databases/spatial/spatial-data-sql-server

Error details: %v`, err)
	}

	logger.Info("SQL Server spatial types available")
	return nil
}
