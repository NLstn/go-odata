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
