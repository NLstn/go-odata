package query

import (
	"log/slog"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type applierBenchmarkProduct struct {
	ID    uint
	Price float64
}

func BenchmarkApplyQueryOptions(b *testing.B) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}

	meta := getTestEntityMetadata()
	opts := &QueryOptions{
		Filter: &FilterExpression{Property: "Price", Operator: OpGreaterThan, Value: 100},
		Top:    intPtr(100),
	}

	b.Run("current", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = ApplyQueryOptions(db.Model(&applierBenchmarkProduct{}), opts, meta, slog.Default())
		}
	})

	b.Run("legacy_logger_session_clone", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			legacyDB := db.Model(&applierBenchmarkProduct{}).Set("_odata_logger", slog.Default())
			_ = ApplyQueryOptions(legacyDB, opts, meta, slog.Default())
		}
	})
}

func intPtr(value int) *int {
	return &value
}
