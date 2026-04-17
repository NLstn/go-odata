//go:build !cgo

package odata

import "gorm.io/gorm"

// SQLiteOpen returns a GORM dialector for SQLite. In non-CGo builds, this is
// equivalent to sqlite.Open() and does not include REGEXP support. To use
// matchesPattern() with SQLite, a CGo-enabled build is required.
func SQLiteOpen(dsn string) gorm.Dialector {
	panic("SQLiteOpen requires CGo; rebuild with CGo enabled")
}

// registerRegexpOnSQLiteConnections is a no-op in non-CGo builds.
func registerRegexpOnSQLiteConnections(_ *gorm.DB) {}
