package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type expandTestCategory struct {
	ID   int    `json:"ID" odata:"key"`
	Name string `json:"name"`
}

type expandTestTag struct {
	ID        int    `json:"ID" odata:"key"`
	Name      string `json:"name"`
	ProductID int    `json:"productID"`
}

type expandTestProduct struct {
	ID       int                  `json:"ID" odata:"key"`
	Name     string               `json:"name"`
	Category *expandTestCategory  `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Tags     []expandTestTag      `json:"tags,omitempty" gorm:"foreignKey:ProductID"`
}

func getExpandTestMetadata(t *testing.T) *metadata.EntityMetadata {
	t.Helper()
	meta, err := metadata.AnalyzeEntity(expandTestProduct{})
	if err != nil {
		t.Fatalf("AnalyzeEntity returned error: %v", err)
	}
	return meta
}

func getExpandTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	return db
}

func TestNeedsPerParentExpand(t *testing.T) {
	meta := getExpandTestMetadata(t)

	t.Run("nil nav prop returns false", func(t *testing.T) {
		expandOpt := ExpandOption{}
		result := needsPerParentExpand(expandOpt, nil)
		if result {
			t.Error("expected false for nil nav prop")
		}
	})

	t.Run("non-navigation prop returns false", func(t *testing.T) {
		expandOpt := ExpandOption{}
		navProp := &metadata.PropertyMetadata{
			Name:            "Name",
			IsNavigationProp: false,
		}
		result := needsPerParentExpand(expandOpt, navProp)
		if result {
			t.Error("expected false for non-navigation prop")
		}
	})

	t.Run("single entity navigation returns false", func(t *testing.T) {
		expandOpt := ExpandOption{}
		navProp := meta.FindNavigationProperty("category")
		result := needsPerParentExpand(expandOpt, navProp)
		if result {
			t.Error("expected false for single entity navigation")
		}
	})

	t.Run("array navigation with top returns true", func(t *testing.T) {
		top := 5
		expandOpt := ExpandOption{Top: &top}
		navProp := meta.FindNavigationProperty("tags")
		result := needsPerParentExpand(expandOpt, navProp)
		if !result {
			t.Error("expected true for array navigation with top")
		}
	})

	t.Run("array navigation with skip returns true", func(t *testing.T) {
		skip := 5
		expandOpt := ExpandOption{Skip: &skip}
		navProp := meta.FindNavigationProperty("tags")
		result := needsPerParentExpand(expandOpt, navProp)
		if !result {
			t.Error("expected true for array navigation with skip")
		}
	})

	t.Run("array navigation without top/skip returns false", func(t *testing.T) {
		expandOpt := ExpandOption{}
		navProp := meta.FindNavigationProperty("tags")
		result := needsPerParentExpand(expandOpt, navProp)
		if result {
			t.Error("expected false for array navigation without top/skip")
		}
	})
}

func TestNeedsPreloadCallback(t *testing.T) {
	t.Run("empty expand option returns false", func(t *testing.T) {
		expandOpt := ExpandOption{}
		result := needsPreloadCallback(expandOpt)
		if result {
			t.Error("expected false for empty expand option")
		}
	})

	t.Run("with Select returns true", func(t *testing.T) {
		expandOpt := ExpandOption{Select: []string{"name"}}
		result := needsPreloadCallback(expandOpt)
		if !result {
			t.Error("expected true with Select")
		}
	})

	t.Run("with Filter returns true", func(t *testing.T) {
		expandOpt := ExpandOption{Filter: &FilterExpression{}}
		result := needsPreloadCallback(expandOpt)
		if !result {
			t.Error("expected true with Filter")
		}
	})

	t.Run("with OrderBy returns true", func(t *testing.T) {
		expandOpt := ExpandOption{OrderBy: []OrderByItem{{Property: "name"}}}
		result := needsPreloadCallback(expandOpt)
		if !result {
			t.Error("expected true with OrderBy")
		}
	})

	t.Run("with Top returns true", func(t *testing.T) {
		top := 5
		expandOpt := ExpandOption{Top: &top}
		result := needsPreloadCallback(expandOpt)
		if !result {
			t.Error("expected true with Top")
		}
	})

	t.Run("with Skip returns true", func(t *testing.T) {
		skip := 5
		expandOpt := ExpandOption{Skip: &skip}
		result := needsPreloadCallback(expandOpt)
		if !result {
			t.Error("expected true with Skip")
		}
	})

	t.Run("with nested Expand returns true", func(t *testing.T) {
		expandOpt := ExpandOption{Expand: []ExpandOption{{NavigationProperty: "category"}}}
		result := needsPreloadCallback(expandOpt)
		if !result {
			t.Error("expected true with nested Expand")
		}
	})

	t.Run("with Compute returns true", func(t *testing.T) {
		expandOpt := ExpandOption{Compute: &ComputeTransformation{}}
		result := needsPreloadCallback(expandOpt)
		if !result {
			t.Error("expected true with Compute")
		}
	})

	t.Run("with Count returns true", func(t *testing.T) {
		expandOpt := ExpandOption{Count: true}
		result := needsPreloadCallback(expandOpt)
		if !result {
			t.Error("expected true with Count")
		}
	})

	t.Run("with Levels returns true", func(t *testing.T) {
		levels := 2
		expandOpt := ExpandOption{Levels: &levels}
		result := needsPreloadCallback(expandOpt)
		if !result {
			t.Error("expected true with Levels")
		}
	})
}

func TestApplyExpandCallback(t *testing.T) {
	meta := getExpandTestMetadata(t)
	db := getExpandTestDB(t)

	t.Run("empty expand option returns db unchanged", func(t *testing.T) {
		expandOpt := ExpandOption{}
		result := applyExpandCallback(db, expandOpt, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("with filter", func(t *testing.T) {
		filter := &FilterExpression{
			Property: "name",
			Operator: OpEqual,
			Value:    "Test",
		}
		expandOpt := ExpandOption{Filter: filter}
		result := applyExpandCallback(db, expandOpt, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("with orderBy ascending", func(t *testing.T) {
		expandOpt := ExpandOption{
			OrderBy: []OrderByItem{
				{Property: "name", Descending: false},
			},
		}
		result := applyExpandCallback(db, expandOpt, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("with orderBy descending", func(t *testing.T) {
		expandOpt := ExpandOption{
			OrderBy: []OrderByItem{
				{Property: "name", Descending: true},
			},
		}
		result := applyExpandCallback(db, expandOpt, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("with skip", func(t *testing.T) {
		skip := 5
		expandOpt := ExpandOption{Skip: &skip}
		result := applyExpandCallback(db, expandOpt, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("with top", func(t *testing.T) {
		top := 10
		expandOpt := ExpandOption{Top: &top}
		result := applyExpandCallback(db, expandOpt, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("with skip and top", func(t *testing.T) {
		skip := 5
		top := 10
		expandOpt := ExpandOption{Skip: &skip, Top: &top}
		result := applyExpandCallback(db, expandOpt, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("with nested expand", func(t *testing.T) {
		expandOpt := ExpandOption{
			Expand: []ExpandOption{
				{NavigationProperty: "category"},
			},
		}
		result := applyExpandCallback(db, expandOpt, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})
}

func TestApplyExpandOption(t *testing.T) {
	meta := getExpandTestMetadata(t)
	db := getExpandTestDB(t)

	t.Run("applies expand callback", func(t *testing.T) {
		top := 5
		expandOpt := ExpandOption{Top: &top}
		result := ApplyExpandOption(db, expandOpt, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})
}

func TestQuoteColumnReference(t *testing.T) {
	tests := []struct {
		name     string
		dialect  string
		column   string
		expected string
	}{
		{
			name:     "empty column",
			dialect:  "sqlite",
			column:   "",
			expected: "",
		},
		{
			name:     "simple column sqlite",
			dialect:  "sqlite",
			column:   "name",
			expected: `"name"`,
		},
		{
			name:     "simple column postgres",
			dialect:  "postgres",
			column:   "name",
			expected: `"name"`,
		},
		{
			name:     "qualified column sqlite",
			dialect:  "sqlite",
			column:   "table.column",
			expected: `"table"."column"`,
		},
		{
			name:     "qualified column postgres",
			dialect:  "postgres",
			column:   "schema.table.column",
			expected: `"schema"."table"."column"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteColumnReference(tt.dialect, tt.column)
			if result != tt.expected {
				t.Errorf("quoteColumnReference(%q, %q) = %q, want %q", tt.dialect, tt.column, result, tt.expected)
			}
		})
	}
}

func TestApplyExpand(t *testing.T) {
	meta := getExpandTestMetadata(t)
	db := getExpandTestDB(t)

	t.Run("empty expand options returns db unchanged", func(t *testing.T) {
		result := applyExpand(db, []ExpandOption{}, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("expand non-existent property skips", func(t *testing.T) {
		expandOpts := []ExpandOption{
			{NavigationProperty: "nonExistent"},
		}
		result := applyExpand(db, expandOpts, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("expand without callback", func(t *testing.T) {
		expandOpts := []ExpandOption{
			{NavigationProperty: "category"},
		}
		result := applyExpand(db, expandOpts, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("expand with callback", func(t *testing.T) {
		expandOpts := []ExpandOption{
			{
				NavigationProperty: "category",
				Select:             []string{"name"},
			},
		}
		result := applyExpand(db, expandOpts, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})
}
