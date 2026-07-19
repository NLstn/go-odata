package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type expandBenchmarkCategory struct {
	ID   uint `gorm:"primaryKey" odata:"key"`
	Name string
}

type expandBenchmarkProduct struct {
	ID         uint `gorm:"primaryKey" odata:"key"`
	Name       string
	CategoryID uint
	Category   *expandBenchmarkCategory `gorm:"foreignKey:CategoryID;references:ID"`
}

func BenchmarkCollectionExpandBelongsTo(b *testing.B) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}
	if err := db.AutoMigrate(&expandBenchmarkCategory{}, &expandBenchmarkProduct{}); err != nil {
		b.Fatal(err)
	}
	categories := make([]expandBenchmarkCategory, 10)
	for i := range categories {
		categories[i] = expandBenchmarkCategory{ID: uint(i + 1), Name: "category"}
	}
	products := make([]expandBenchmarkProduct, 50)
	for i := range products {
		products[i] = expandBenchmarkProduct{ID: uint(i + 1), Name: "product", CategoryID: uint(i%len(categories) + 1)}
	}
	if err := db.Create(&categories).Error; err != nil {
		b.Fatal(err)
	}
	if err := db.Create(&products).Error; err != nil {
		b.Fatal(err)
	}

	productMeta, err := metadata.AnalyzeEntity(expandBenchmarkProduct{})
	if err != nil {
		b.Fatal(err)
	}
	categoryMeta, err := metadata.AnalyzeEntity(expandBenchmarkCategory{})
	if err != nil {
		b.Fatal(err)
	}
	handler := NewEntityHandler(db, productMeta, nil)
	handler.SetEntitiesMetadata(map[string]*metadata.EntityMetadata{
		productMeta.EntitySetName:  productMeta,
		categoryMeta.EntitySetName: categoryMeta,
	})
	req := httptest.NewRequest(http.MethodGet, "/ExpandBenchmarkProducts?$top=50&$expand=Category", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.HandleCollection(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("status = %d: %s", w.Code, w.Body.String())
		}
	}
}
