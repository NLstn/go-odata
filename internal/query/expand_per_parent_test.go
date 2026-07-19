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

type perParentCustomer struct {
	ID uint `gorm:"primaryKey" odata:"key"`
}

// perParentOrder's Customer tag names "references:" without "foreignKey:", so
// extractReferentialConstraints leaves ReferentialConstraints empty and
// resolution falls to GormReferenceConstraints (see analyzer.go), resolved
// from GORM's own relationship schema at metadata-analysis time.
type perParentOrder struct {
	ID         uint `gorm:"primaryKey" odata:"key"`
	CustomerID uint
	Customer   *perParentCustomer `gorm:"references:ID"`
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

func TestApplyPerParentExpandUsesGormReferenceConstraintsWithoutForeignKeyTag(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&perParentOrder{}, &perParentCustomer{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&perParentCustomer{ID: 1})
	db.Create(&perParentOrder{ID: 1, CustomerID: 1})

	orderMeta, err := metadata.AnalyzeEntity(perParentOrder{})
	if err != nil {
		t.Fatal(err)
	}
	customerMeta, err := metadata.AnalyzeEntity(perParentCustomer{})
	if err != nil {
		t.Fatal(err)
	}
	registry := map[string]*metadata.EntityMetadata{
		orderMeta.EntitySetName:    orderMeta,
		customerMeta.EntitySetName: customerMeta,
	}
	orderMeta.SetEntitiesRegistry(registry)
	customerMeta.SetEntitiesRegistry(registry)

	customerProp := orderMeta.FindProperty("Customer")
	if customerProp == nil {
		t.Fatal("Customer navigation property not found")
	}
	if len(customerProp.ReferentialConstraints) != 0 {
		t.Fatalf("expected tag-based ReferentialConstraints to stay empty, got %v", customerProp.ReferentialConstraints)
	}
	if len(customerProp.GormReferenceConstraints) == 0 {
		t.Fatal("expected GormReferenceConstraints to be resolved from GORM's schema")
	}

	var orders []perParentOrder
	db.Find(&orders)
	if err := ApplyPerParentExpand(db, &orders, []ExpandOption{{NavigationProperty: "Customer"}}, orderMeta); err != nil {
		t.Fatal(err)
	}
	if orders[0].Customer == nil || orders[0].Customer.ID != 1 {
		t.Fatalf("Customer = %#v, want customer 1", orders[0].Customer)
	}
}
