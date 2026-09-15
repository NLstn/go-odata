//go:build cgo

// Package sqlite configures SQLite connections for OData-specific SQL behavior.
package sqlite

import (
	"context"
	"fmt"
	"regexp"

	"github.com/mattn/go-sqlite3"
	"gorm.io/gorm"
)

// Configure prepares a GORM SQLite database for OData string filters.
//
// SQLite registers user-defined functions and pragmas per connection. Configure
// therefore pins the database/sql pool to one connection, enables
// case_sensitive_like for ordinal contains(), startswith(), and endswith()
// comparisons, and registers REGEXP for the OData v4.01 matchesPattern()
// function.
//
// Applications using SQLite should call Configure after opening the database
// and before creating an OData service. Applications using other databases do
// not need this package.
func Configure(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("configure SQLite: database must not be nil")
	}
	if db.Name() != "sqlite" {
		return fmt.Errorf("configure SQLite: unsupported GORM dialector %q", db.Name())
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("configure SQLite: get database pool: %w", err)
	}

	// mattn/go-sqlite3 installs functions and pragmas per connection. Keep one
	// connection alive so every OData query observes the same configuration.
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
		return fmt.Errorf("configure SQLite: get connection: %w", err)
	}
	defer conn.Close() //nolint:errcheck

	if _, err := conn.ExecContext(ctx, "PRAGMA case_sensitive_like = ON"); err != nil {
		return fmt.Errorf("configure SQLite: enable case-sensitive LIKE: %w", err)
	}

	if err := conn.Raw(func(driverConn interface{}) error {
		sqliteConn, ok := driverConn.(*sqlite3.SQLiteConn)
		if !ok {
			return fmt.Errorf("unexpected SQLite driver connection type %T", driverConn)
		}
		return sqliteConn.RegisterFunc("regexp", func(pattern, value string) (bool, error) {
			return regexp.MatchString(pattern, value)
		}, true)
	}); err != nil {
		return fmt.Errorf("configure SQLite: register REGEXP: %w", err)
	}

	return nil
}
