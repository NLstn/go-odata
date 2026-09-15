//go:build cgo

package sqlite_test

import (
	"testing"

	odatasqlite "github.com/nlstn/go-odata/sqlite"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestConfigure(t *testing.T) {
	db, err := gorm.Open(gormsqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}

	if err := odatasqlite.Configure(db); err != nil {
		t.Fatalf("Configure() error: %v", err)
	}

	var caseInsensitiveMatch int
	if err := db.Raw("SELECT 'Laptop' LIKE 'laptop'").Scan(&caseInsensitiveMatch).Error; err != nil {
		t.Fatalf("query LIKE behavior: %v", err)
	}
	if caseInsensitiveMatch != 0 {
		t.Fatalf("LIKE remained case-insensitive; got %d, want 0", caseInsensitiveMatch)
	}

	var regexpMatch int
	if err := db.Raw("SELECT 'Laptop' REGEXP '^Lap'").Scan(&regexpMatch).Error; err != nil {
		t.Fatalf("query REGEXP behavior: %v", err)
	}
	if regexpMatch != 1 {
		t.Fatalf("REGEXP result = %d, want 1", regexpMatch)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database pool: %v", err)
	}
	if got := sqlDB.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", got)
	}
}

func TestConfigureRejectsNilDatabase(t *testing.T) {
	if err := odatasqlite.Configure(nil); err == nil {
		t.Fatal("Configure(nil) returned nil error")
	}
}
