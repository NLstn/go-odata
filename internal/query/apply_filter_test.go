package query

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type navigationFilterAuthor struct {
	ID          uint                       `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string                     `json:"Name" gorm:"column:custom_name"`
	PublisherID uint                       `json:"PublisherID"`
	Publisher   *navigationFilterPublisher `json:"Publisher,omitempty" gorm:"foreignKey:PublisherID"`
}

type navigationFilterPublisher struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name" gorm:"column:publisher_name"`
}

func (navigationFilterAuthor) TableName() string {
	return "nav_authors"
}

func (navigationFilterPublisher) TableName() string {
	return "nav_publishers"
}

type navigationFilterBook struct {
	ID       uint                    `json:"ID" gorm:"primaryKey" odata:"key"`
	AuthorID uint                    `json:"AuthorID"`
	Author   *navigationFilterAuthor `json:"Author,omitempty" gorm:"foreignKey:AuthorID"`
}

func (navigationFilterBook) TableName() string {
	return "nav_books"
}

type lambdaMismatchParent struct {
	KeyPartOne uint                `json:"KeyPartOne" gorm:"primaryKey" odata:"key"`
	KeyPartTwo uint                `json:"KeyPartTwo" gorm:"primaryKey" odata:"key"`
	Children   []lambdaMismatchKid `json:"Children" gorm:"foreignKey:ParentKeyPartOne"`
}

type lambdaMismatchKid struct {
	ID               uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	ParentKeyPartOne uint   `json:"ParentKeyPartOne"`
	Name             string `json:"Name"`
}

type countFilterProduct struct {
	ID           uint                     `json:"ID" gorm:"primaryKey" odata:"key"`
	Descriptions []countFilterDescription `json:"Descriptions" gorm:"foreignKey:ProductID"`
}

type countFilterDescription struct {
	ID        uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	ProductID uint   `json:"ProductID"`
	Text      string `json:"Text"`
}

func (countFilterProduct) TableName() string {
	return "count_filter_products"
}

func (countFilterDescription) TableName() string {
	return "count_filter_descriptions"
}

func buildNavigationFilterMetadata(t *testing.T) (*metadata.EntityMetadata, *metadata.EntityMetadata, *metadata.EntityMetadata) {
	t.Helper()

	authorMeta, err := metadata.AnalyzeEntity(&navigationFilterAuthor{})
	if err != nil {
		t.Fatalf("Failed to analyze author entity: %v", err)
	}

	publisherMeta, err := metadata.AnalyzeEntity(&navigationFilterPublisher{})
	if err != nil {
		t.Fatalf("Failed to analyze publisher entity: %v", err)
	}

	bookMeta, err := metadata.AnalyzeEntity(&navigationFilterBook{})
	if err != nil {
		t.Fatalf("Failed to analyze book entity: %v", err)
	}

	setEntitiesRegistry(authorMeta, publisherMeta, bookMeta)

	return authorMeta, publisherMeta, bookMeta
}

func TestNavigationFilterUsesTargetColumnMetadata(t *testing.T) {
	_, _, bookMeta := buildNavigationFilterMetadata(t)

	filterExpr, err := parseFilter("Author/Name eq 'Jane'", bookMeta, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse navigation filter: %v", err)
	}

	sql, args := buildFilterCondition("postgres", filterExpr, bookMeta)

	expectedSQL := `"nav_author"."custom_name" = ?`
	if sql != expectedSQL {
		t.Fatalf("expected SQL %q, got %q", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != "Jane" {
		t.Fatalf("expected args [Jane], got %#v", args)
	}
}

func TestNavigationFilterUsesMultiHopNavigationPath(t *testing.T) {
	_, _, bookMeta := buildNavigationFilterMetadata(t)

	filterExpr, err := parseFilter("Author/Publisher/Name eq 'Acme'", bookMeta, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse navigation filter: %v", err)
	}

	sql, args := buildFilterCondition("postgres", filterExpr, bookMeta)

	expectedSQL := `"nav_author_publisher"."publisher_name" = ?`
	if sql != expectedSQL {
		t.Fatalf("expected SQL %q, got %q", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != "Acme" {
		t.Fatalf("expected args [Acme], got %#v", args)
	}
}

// TestFilterNestedArithmeticGroupedOnLeft is a regression test for a bug where
// "(Price div 3) mul 3 eq 15.5" silently dropped the parenthesized "Price div 3"
// sub-expression: the outer "mul" node's Left operand was a *GroupExpr, which
// convertBinaryArithmeticExprWithContext didn't unwrap, so it fell into the
// "no property" placeholder branch and the filter degenerated into comparing the
// bare literal 3 against 15.5 (i.e., it matched no rows regardless of Price).
func TestFilterNestedArithmeticGroupedOnLeft(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	rows := []TestEntity{
		{ID: 1, Name: "Coffee Mug", Price: 15.5}, // (15.5 div 3) mul 3 == 15.5
		{ID: 2, Name: "Notebook", Price: 10.0},   // (10 div 3) mul 3 == 9.0 != 10.0
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("Failed to seed rows: %v", err)
	}

	meta := getTestMetadata(t)

	filterExpr, err := parseFilter("(Price div 3) mul 3 eq 15.5", meta, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filterExpr.Left == nil || filterExpr.Left.Left == nil {
		t.Fatalf("expected nested arithmetic expression to be preserved, got %+v", filterExpr.Left)
	}

	var matched []TestEntity
	query := applyFilter(db.Model(&TestEntity{}), filterExpr, meta)
	if err := query.Find(&matched).Error; err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if len(matched) != 1 || matched[0].ID != 1 {
		t.Fatalf("expected exactly [Coffee Mug] (ID 1) to match, got %+v", matched)
	}
}

func TestBuildLambdaConditionLogsForeignKeyMismatch(t *testing.T) {
	parentMeta, err := metadata.AnalyzeEntity(&lambdaMismatchParent{})
	if err != nil {
		t.Fatalf("Failed to analyze parent entity: %v", err)
	}

	childMeta, err := metadata.AnalyzeEntity(&lambdaMismatchKid{})
	if err != nil {
		t.Fatalf("Failed to analyze child entity: %v", err)
	}

	setEntitiesRegistry(parentMeta, childMeta)

	filterExpr, err := parseFilter("Children/any(c:c/Name eq 'Jane')", parentMeta, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse lambda filter: %v", err)
	}

	originalStdout := os.Stdout
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})
	os.Stdout = stdoutWriter

	_, _ = buildLambdaCondition("sqlite", filterExpr, parentMeta, "")

	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("Failed to close stdout writer: %v", err)
	}
	stdoutBytes, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatalf("Failed to read stdout: %v", err)
	}
	if err := stdoutReader.Close(); err != nil {
		t.Fatalf("Failed to close stdout reader: %v", err)
	}

	if len(stdoutBytes) == 0 {
		t.Fatalf("expected warning output to stdout, got none")
	}

	stdoutOutput := string(stdoutBytes)
	if !strings.Contains(stdoutOutput, "Foreign key column count") {
		t.Fatalf("expected warning about foreign key mismatch in stdout, got %q", stdoutOutput)
	}
}

func TestCollectionCountFilterBuildsCorrelatedSubquery(t *testing.T) {
	productMeta, err := metadata.AnalyzeEntity(&countFilterProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze product entity: %v", err)
	}

	descriptionMeta, err := metadata.AnalyzeEntity(&countFilterDescription{})
	if err != nil {
		t.Fatalf("Failed to analyze description entity: %v", err)
	}

	setEntitiesRegistry(productMeta, descriptionMeta)

	filterExpr, err := parseFilter("Descriptions/$count gt 1", productMeta, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse collection count filter: %v", err)
	}

	sql, args := buildFilterCondition("postgres", filterExpr, productMeta)
	expectedSQL := "(SELECT COUNT(*) FROM \"count_filter_descriptions\" WHERE \"count_filter_descriptions\".\"product_id\" = \"count_filter_products\".\"id\") > ?"
	if sql != expectedSQL {
		t.Fatalf("expected SQL %q, got %q", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != int64(1) {
		t.Fatalf("expected args [1], got %#v", args)
	}
}
