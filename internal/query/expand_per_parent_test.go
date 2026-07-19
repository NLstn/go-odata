package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type perParentAuthor struct {
	ID    uint            `gorm:"primaryKey" odata:"key"`
	Books []perParentBook `gorm:"foreignKey:AuthorID"`
}

type perParentBook struct {
	ID       uint `gorm:"primaryKey" odata:"key"`
	AuthorID uint
	Author   *perParentAuthor `gorm:"foreignKey:AuthorID"`
}

type perParentTag struct {
	ID uint `gorm:"primaryKey" odata:"key"`
}

type perParentProduct struct {
	ID   uint           `gorm:"primaryKey" odata:"key"`
	Tags []perParentTag `gorm:"many2many:per_parent_product_tags"`
}

func TestApplyPerParentExpandLoadsDirectRelationships(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&perParentAuthor{}, &perParentBook{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&perParentAuthor{ID: 1})
	db.Create(&[]perParentBook{{ID: 1, AuthorID: 1}, {ID: 2, AuthorID: 1}})

	authorMeta, err := metadata.AnalyzeEntity(perParentAuthor{})
	if err != nil {
		t.Fatal(err)
	}
	bookMeta, err := metadata.AnalyzeEntity(perParentBook{})
	if err != nil {
		t.Fatal(err)
	}
	registry := map[string]*metadata.EntityMetadata{
		authorMeta.EntitySetName: authorMeta,
		bookMeta.EntitySetName:   bookMeta,
	}
	authorMeta.SetEntitiesRegistry(registry)
	bookMeta.SetEntitiesRegistry(registry)

	var authors []perParentAuthor
	db.Find(&authors)
	if err := ApplyPerParentExpand(db, &authors, []ExpandOption{{NavigationProperty: "Books"}}, authorMeta); err != nil {
		t.Fatal(err)
	}
	if got := len(authors[0].Books); got != 2 {
		t.Fatalf("Books length = %d, want 2", got)
	}

	var books []perParentBook
	db.Find(&books)
	if err := ApplyPerParentExpand(db, &books, []ExpandOption{{NavigationProperty: "Author"}}, bookMeta); err != nil {
		t.Fatal(err)
	}
	if books[0].Author == nil || books[0].Author.ID != 1 {
		t.Fatalf("Author = %#v, want author 1", books[0].Author)
	}
}

func TestApplyPerParentExpandLoadsManyToManyRelationships(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&perParentProduct{}, &perParentTag{}); err != nil {
		t.Fatal(err)
	}
	product := perParentProduct{ID: 1}
	tags := []perParentTag{{ID: 1}, {ID: 2}}
	db.Create(&product)
	db.Create(&tags)
	if err := db.Model(&product).Association("Tags").Append(&tags); err != nil {
		t.Fatal(err)
	}

	productMeta, err := metadata.AnalyzeEntity(perParentProduct{})
	if err != nil {
		t.Fatal(err)
	}
	tagMeta, err := metadata.AnalyzeEntity(perParentTag{})
	if err != nil {
		t.Fatal(err)
	}
	registry := map[string]*metadata.EntityMetadata{
		productMeta.EntitySetName: productMeta,
		tagMeta.EntitySetName:     tagMeta,
	}
	productMeta.SetEntitiesRegistry(registry)
	tagMeta.SetEntitiesRegistry(registry)

	var products []perParentProduct
	db.Find(&products)
	if err := ApplyPerParentExpand(db, &products, []ExpandOption{{NavigationProperty: "Tags"}}, productMeta); err != nil {
		t.Fatal(err)
	}
	if got := len(products[0].Tags); got != 2 {
		t.Fatalf("Tags length = %d, want 2", got)
	}
}
