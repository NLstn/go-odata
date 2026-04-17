//go:build cgo

package odata

import (
	"database/sql"
	"regexp"

	"github.com/mattn/go-sqlite3"
	"gorm.io/gorm"
)

func init() {
	// Register a custom SQLite driver with REGEXP support for the OData v4.01
	// matchesPattern() filter function. This driver is identical to the standard
	// sqlite3 driver except it provides a REGEXP function via ConnectHook.
	sql.Register("sqlite3_regexp", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.RegisterFunc("regexp", func(pattern, s string) (bool, error) {
				return regexp.MatchString(pattern, s)
			}, true)
		},
	})
}

// ensureSQLiteRegexp ensures the REGEXP function is registered on SQLite connections.
// This function is called automatically when creating a service with SQLite,
// so users don't need to do anything special - just use sqlite.Open() normally.
func ensureSQLiteRegexp(db *gorm.DB) error {
	if db.Name() != "sqlite" {
		return nil
	}

	// Get the underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Register REGEXP on a connection from the pool. This will be called
	// on every new connection via the ConnectHook if using the sqlite3_regexp driver.
	// For users with standard sqlite3, attempt to register it here.
	conn, err := sqlDB.Conn(db.Statement.Context)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck

	// Try to register REGEXP; if the driver already has it via ConnectHook, this is a no-op
	//nolint:errcheck
	conn.Raw(func(driverConn interface{}) error {
		sqliteConn, ok := driverConn.(*sqlite3.SQLiteConn)
		if !ok {
			return nil
		}
		return sqliteConn.RegisterFunc("regexp", func(pattern, s string) (bool, error) {
			return regexp.MatchString(pattern, s)
		}, true)
	})

	return nil
}
