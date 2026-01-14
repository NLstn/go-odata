package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

type navigationFilterAuthor struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name" gorm:"column:custom_name"`
}

func (navigationFilterAuthor) TableName() string {
	return "nav_authors"
}

type navigationFilterBook struct {
	ID       uint                    `json:"ID" gorm:"primaryKey" odata:"key"`
	AuthorID uint                    `json:"AuthorID"`
	Author   *navigationFilterAuthor `json:"Author,omitempty" gorm:"foreignKey:AuthorID"`
}

func (navigationFilterBook) TableName() string {
	return "nav_books"
}

func buildNavigationFilterMetadata(t *testing.T) (*metadata.EntityMetadata, *metadata.EntityMetadata) {
	t.Helper()

	authorMeta, err := metadata.AnalyzeEntity(&navigationFilterAuthor{})
	if err != nil {
		t.Fatalf("Failed to analyze author entity: %v", err)
	}

	bookMeta, err := metadata.AnalyzeEntity(&navigationFilterBook{})
	if err != nil {
		t.Fatalf("Failed to analyze book entity: %v", err)
	}

	setEntitiesRegistry(authorMeta, bookMeta)

	return authorMeta, bookMeta
}

func TestNavigationFilterUsesTargetColumnMetadata(t *testing.T) {
	_, bookMeta := buildNavigationFilterMetadata(t)

	filterExpr, err := parseFilter("Author/Name eq 'Jane'", bookMeta, nil)
	if err != nil {
		t.Fatalf("Failed to parse navigation filter: %v", err)
	}

	sql, args := buildFilterCondition("postgres", filterExpr, bookMeta)

	expectedSQL := `"nav_authors"."custom_name" = ?`
	if sql != expectedSQL {
		t.Fatalf("expected SQL %q, got %q", expectedSQL, sql)
	}

	if len(args) != 1 || args[0] != "Jane" {
		t.Fatalf("expected args [Jane], got %#v", args)
	}
}
