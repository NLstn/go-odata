//go:build !cgo

// Package sqlite configures SQLite connections for OData-specific SQL behavior.
package sqlite

import (
	"fmt"

	"gorm.io/gorm"
)

// Configure reports that the mattn/go-sqlite3 integration requires CGO.
func Configure(_ *gorm.DB) error {
	return fmt.Errorf("configure SQLite: CGO is required")
}
