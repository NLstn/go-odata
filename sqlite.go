//go:build cgo

package odata

import (
	"context"
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
//
// Because mattn/go-sqlite3 registers user-defined functions per-connection (not
// globally), we must make sure that every connection the OData service uses has
// REGEXP available. The standard gorm sqlite dialector opens via the default
// "sqlite3" driver, which does not expose a ConnectHook. To guarantee REGEXP is
// present on any connection drawn from the pool we pin the pool to a single
// connection (max open = max idle = 1, no idle-timeout) and register REGEXP on
// that single connection. Single-connection SQLite pools are also the common
// recommendation for SQLite because writes are serialized and in-memory
// databases are not shared between connections.
func ensureSQLiteRegexp(db *gorm.DB) error {
	if db.Name() != "sqlite" {
		return nil
	}

	// Get the underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Pin the pool to a single connection so the REGEXP function we register
	// below is available for every query. Without this, new connections
	// created by database/sql would not have the function and any
	// matchesPattern() filter would fail with "no such function: REGEXP".
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)

	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close() //nolint:errcheck

	return conn.Raw(func(driverConn interface{}) error {
		sqliteConn, ok := driverConn.(*sqlite3.SQLiteConn)
		if !ok {
			return nil
		}
		return sqliteConn.RegisterFunc("regexp", func(pattern, s string) (bool, error) {
			return regexp.MatchString(pattern, s)
		}, true)
	})
}
