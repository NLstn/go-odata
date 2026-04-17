//go:build cgo

package odata

import (
	"context"
	"database/sql"
	"regexp"

	"github.com/mattn/go-sqlite3"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	// Register a custom SQLite driver that provides a REGEXP function, enabling
	// the OData v4.01 matchesPattern() filter function for SQLite databases.
	// The standard sqlite3 driver does not include REGEXP by default.
	sql.Register("sqlite3_odata", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.RegisterFunc("regexp", func(pattern, s string) (bool, error) {
				return regexp.MatchString(pattern, s)
			}, true)
		},
	})
}

// SQLiteOpen returns a GORM dialector for SQLite that includes support for the
// REGEXP operator, which is required for the OData v4.01 matchesPattern() filter
// function. Use this instead of sqlite.Open() when creating your GORM database:
//
//	db, err := gorm.Open(odata.SQLiteOpen("path/to/db.sqlite"), &gorm.Config{})
func SQLiteOpen(dsn string) gorm.Dialector {
	return gormsqlite.Dialector{DriverName: "sqlite3_odata", DSN: dsn}
}

// registerRegexpOnSQLiteConnections attempts to register the REGEXP function on
// the current connections in the SQLite pool. This is a best-effort approach for
// users who opened their database without SQLiteOpen. It covers the common
// single-connection SQLite case (in-memory and most file-based usage).
func registerRegexpOnSQLiteConnections(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}

	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		return
	}
	defer conn.Close() //nolint:errcheck

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
}
