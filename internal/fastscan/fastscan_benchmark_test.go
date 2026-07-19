package fastscan

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// BenchProduct mirrors the shape of a typical OData entity: mixed scalar
// kinds, nullable pointer columns, a named enum type, and timestamps.
type BenchProduct struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Description *string
	Price       float64
	CategoryID  *uint
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const (
	benchRows = 5000
	benchPage = 100
)

func setupBenchDB(b *testing.B) *gorm.DB {
	b.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		b.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&BenchProduct{}); err != nil {
		b.Fatalf("migrate: %v", err)
	}
	now := time.Now().UTC()
	products := make([]BenchProduct, 0, benchRows)
	for i := 0; i < benchRows; i++ {
		desc := fmt.Sprintf("Description for product %d with some padding text", i)
		cat := uint(i%10 + 1)
		products = append(products, BenchProduct{
			Name:        fmt.Sprintf("Product %d", i),
			Description: &desc,
			Price:       float64(i) * 1.5,
			CategoryID:  &cat,
			Status:      Status(i % 16),
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	if err := db.CreateInBatches(products, 500).Error; err != nil {
		b.Fatalf("seed: %v", err)
	}
	return db
}

func benchQuery(db *gorm.DB) *gorm.DB {
	return db.Where("price > ?", 10.0).Order("id").Limit(benchPage)
}

func BenchmarkGormFind(b *testing.B) {
	db := setupBenchDB(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := make([]BenchProduct, 0, benchPage)
		if err := benchQuery(db).Find(&results).Error; err != nil {
			b.Fatal(err)
		}
		if len(results) != benchPage {
			b.Fatalf("got %d rows", len(results))
		}
	}
}

func BenchmarkFastscanFind(b *testing.B) {
	db := setupBenchDB(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := make([]BenchProduct, 0, benchPage)
		if err := Find(benchQuery(db), &results); err != nil {
			b.Fatal(err)
		}
		if len(results) != benchPage {
			b.Fatalf("got %d rows", len(results))
		}
	}
}

// firstQuery mirrors a single-entity read by primary key: WHERE id = ? with the
// LIMIT 1 / ORDER BY primary key that First adds itself.
func firstQuery(db *gorm.DB) *gorm.DB {
	return db.Where("id = ?", benchRows/2)
}

func BenchmarkGormFirst(b *testing.B) {
	db := setupBenchDB(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var product BenchProduct
		if err := firstQuery(db).First(&product).Error; err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFastscanFirst(b *testing.B) {
	db := setupBenchDB(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var product BenchProduct
		if err := First(firstQuery(db), &product); err != nil {
			b.Fatal(err)
		}
	}
}

// benchApplyQuery mirrors the $apply groupby/aggregate shape that reaches map
// scanning: one grouping key plus one aggregate, producing one row per group.
// With benchRows products spread over benchGroups categories the result is
// benchGroups rows — the "usually few rows" case issue #836 flags.
const benchGroups = 10

func benchApplyQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&BenchProduct{}).
		Select("category_id, AVG(price) AS avg_price").
		Group("category_id").
		Order("category_id")
}

func BenchmarkGormMapScan(b *testing.B) {
	db := setupBenchDB(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var results []map[string]interface{}
		if err := benchApplyQuery(db).Find(&results).Error; err != nil {
			b.Fatal(err)
		}
		if len(results) != benchGroups {
			b.Fatalf("got %d rows", len(results))
		}
	}
}

func BenchmarkFastscanMapScan(b *testing.B) {
	db := setupBenchDB(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var results []map[string]interface{}
		if err := FindMap(benchApplyQuery(db), &results); err != nil {
			b.Fatal(err)
		}
		if len(results) != benchGroups {
			b.Fatalf("got %d rows", len(results))
		}
	}
}

// benchComputeQuery mirrors the $compute shape: every entity column plus a
// computed expression, returning one map per row (a full page), which is the
// scan-bound many-row case for map results.
func benchComputeQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&BenchProduct{}).
		Select("*, price * 1.1 AS price_with_tax").
		Where("price > ?", 10.0).
		Order("id").
		Limit(benchPage)
}

func BenchmarkGormMapScanCompute(b *testing.B) {
	db := setupBenchDB(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var results []map[string]interface{}
		if err := benchComputeQuery(db).Find(&results).Error; err != nil {
			b.Fatal(err)
		}
		if len(results) != benchPage {
			b.Fatalf("got %d rows", len(results))
		}
	}
}

func BenchmarkFastscanMapScanCompute(b *testing.B) {
	db := setupBenchDB(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var results []map[string]interface{}
		if err := FindMap(benchComputeQuery(db), &results); err != nil {
			b.Fatal(err)
		}
		if len(results) != benchPage {
			b.Fatalf("got %d rows", len(results))
		}
	}
}
