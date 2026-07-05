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

// ensureSQLiteRegexp ensures every SQLite connection this library opens has the
// REGEXP function available and performs ordinal (case-sensitive) string
// comparisons for LIKE. This function is called automatically when creating a
// service with SQLite, so users don't need to do anything special - just use
// sqlite.Open() normally.
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
//
// The same pinned connection is used to enable the "case_sensitive_like"
// pragma. SQLite's LIKE operator is case-insensitive for ASCII characters by
// default, which is at odds with the OData v4.0 URL Conventions spec (Part 2,
// Sec. 5.1.1.7) requirement that contains()/startswith()/endswith() perform
// ordinal (case-sensitive) string comparison. Unlike other dialects, SQLite
// has no per-query syntax (e.g. an explicit COLLATE clause) to force LIKE to
// be case-sensitive - the pragma is the only mechanism, hence it is set here
// alongside the REGEXP registration rather than in the SQL-generation layer.
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
	// below, and the case_sensitive_like pragma, are available/in effect for
	// every query. Without this, new connections created by database/sql
	// would not have the function and any matchesPattern() filter would fail
	// with "no such function: REGEXP", and LIKE-based filters could silently
	// fall back to case-insensitive matching.
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

	// Force ordinal (case-sensitive) matching for LIKE, which backs the
	// contains()/startswith()/endswith() filter functions. See the doc
	// comment above for why this must be a connection-level pragma rather
	// than SQL generated per-query.
	if _, err := conn.ExecContext(ctx, "PRAGMA case_sensitive_like = ON"); err != nil {
		return err
	}

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
