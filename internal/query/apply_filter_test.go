package query

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
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

	filterExpr, err := parseFilter("Author/Name eq 'Jane'", bookMeta, nil, 0)
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

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	originalStdout := os.Stdout
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	t.Cleanup(func() {
		os.Stdout = originalStdout
	})
	os.Stdout = stdoutWriter

	_, _ = buildLambdaCondition("sqlite", filterExpr, parentMeta, logger)

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

	if len(stdoutBytes) != 0 {
		t.Fatalf("expected no stdout output, got %q", string(stdoutBytes))
	}

	if !strings.Contains(logBuffer.String(), "Foreign key column count does not match key property count for navigation property") {
		t.Fatalf("expected warning to be logged, got %q", logBuffer.String())
	}
}
