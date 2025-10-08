package query

import (
	"net/url"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// Test entities
type TestAuthor struct {
	ID    uint       `json:"ID" gorm:"primaryKey" odata:"key"`
	Name  string     `json:"Name"`
	Books []TestBook `json:"Books" gorm:"foreignKey:AuthorID"`
}

type TestBook struct {
	ID       uint        `json:"ID" gorm:"primaryKey" odata:"key"`
	Title    string      `json:"Title"`
	AuthorID uint        `json:"AuthorID"`
	Author   *TestAuthor `json:"Author,omitempty" gorm:"foreignKey:AuthorID"`
}

// TestParseExpandSimple tests parsing a simple $expand
func TestParseExpandSimple(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	if options.Expand[0].NavigationProperty != "Books" {
		t.Errorf("Expected navigation property 'Books', got %s", options.Expand[0].NavigationProperty)
	}
}

// TestParseExpandWithNestedTop tests parsing $expand with nested $top
func TestParseExpandWithNestedTop(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books($top=5)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	if options.Expand[0].Top == nil {
		t.Error("Expected $top to be set")
	} else if *options.Expand[0].Top != 5 {
		t.Errorf("Expected $top=5, got %d", *options.Expand[0].Top)
	}
}

// TestParseExpandWithNestedSkip tests parsing $expand with nested $skip
func TestParseExpandWithNestedSkip(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books($skip=2)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	if options.Expand[0].Skip == nil {
		t.Error("Expected $skip to be set")
	} else if *options.Expand[0].Skip != 2 {
		t.Errorf("Expected $skip=2, got %d", *options.Expand[0].Skip)
	}
}

// TestParseExpandWithNestedSelect tests parsing $expand with nested $select
func TestParseExpandWithNestedSelect(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books($select=Title)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	if len(options.Expand[0].Select) != 1 {
		t.Errorf("Expected 1 select property, got %d", len(options.Expand[0].Select))
	} else if options.Expand[0].Select[0] != "Title" {
		t.Errorf("Expected select property 'Title', got %s", options.Expand[0].Select[0])
	}
}

// TestParseExpandWithMultipleNestedOptions tests parsing $expand with multiple nested options
func TestParseExpandWithMultipleNestedOptions(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books($select=Title;$top=3;$skip=1)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	expand := options.Expand[0]

	if len(expand.Select) != 1 || expand.Select[0] != "Title" {
		t.Error("Expected $select=Title")
	}

	if expand.Top == nil || *expand.Top != 3 {
		t.Error("Expected $top=3")
	}

	if expand.Skip == nil || *expand.Skip != 1 {
		t.Error("Expected $skip=1")
	}
}

// TestParseExpandInvalid tests parsing an invalid $expand
func TestParseExpandInvalid(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "InvalidProperty")

	_, err := ParseQueryOptions(params, authorMeta)
	if err == nil {
		t.Error("Expected error for invalid navigation property")
	}
}

// TestParseExpandMultiple tests parsing multiple $expand values
func TestParseExpandMultiple(t *testing.T) {
	// For this test, we need a more complex entity structure
	// Since we only have Author->Books, we'll just test the parsing logic
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Errorf("Expected 1 expand option, got %d", len(options.Expand))
	}
}

// TestParseExpandWithFilterAndOrderBy tests combining $expand with $filter and $orderby
func TestParseExpandWithFilterAndOrderBy(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books")
	params.Set("$filter", "Name eq 'Test'")
	params.Set("$orderby", "Name asc")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Error("Expected 1 expand option")
	}

	if options.Filter == nil {
		t.Error("Expected filter to be set")
	}

	if len(options.OrderBy) != 1 {
		t.Error("Expected 1 orderby clause")
	}
}

// TestParseExpandWithCount tests combining $expand with $count
func TestParseExpandWithCount(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books")
	params.Set("$count", "true")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Error("Expected 1 expand option")
	}

	if !options.Count {
		t.Error("Expected count to be true")
	}
}

// TestParseExpandWithTopAndSkip tests combining $expand with $top and $skip on main entity
func TestParseExpandWithTopAndSkip(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books")
	params.Set("$top", "10")
	params.Set("$skip", "5")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Error("Expected 1 expand option")
	}

	if options.Top == nil || *options.Top != 10 {
		t.Error("Expected $top=10")
	}

	if options.Skip == nil || *options.Skip != 5 {
		t.Error("Expected $skip=5")
	}
}

// TestParseExpandWithNestedFilter tests parsing $expand with nested $filter
func TestParseExpandWithNestedFilter(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books($filter=Title eq 'Test Book')")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	expand := options.Expand[0]
	if expand.Filter == nil {
		t.Fatal("Expected $filter to be set")
	}

	if expand.Filter.Property != "Title" {
		t.Errorf("Expected property 'Title', got '%s'", expand.Filter.Property)
	}

	if expand.Filter.Operator != OpEqual {
		t.Errorf("Expected operator 'eq', got '%s'", expand.Filter.Operator)
	}

	if expand.Filter.Value != "Test Book" {
		t.Errorf("Expected value 'Test Book', got '%v'", expand.Filter.Value)
	}
}

// TestParseExpandWithNestedOrderBy tests parsing $expand with nested $orderby
func TestParseExpandWithNestedOrderBy(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books($orderby=Title desc)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	expand := options.Expand[0]
	if len(expand.OrderBy) != 1 {
		t.Fatalf("Expected 1 orderby item, got %d", len(expand.OrderBy))
	}

	orderBy := expand.OrderBy[0]
	if orderBy.Property != "Title" {
		t.Errorf("Expected property 'Title', got '%s'", orderBy.Property)
	}

	if !orderBy.Descending {
		t.Error("Expected descending order")
	}
}

// TestParseExpandWithMultipleNestedFilters tests parsing $expand with complex nested $filter
func TestParseExpandWithMultipleNestedFilters(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books($filter=Title eq 'Book A' or Title eq 'Book B')")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	expand := options.Expand[0]
	if expand.Filter == nil {
		t.Fatal("Expected $filter to be set")
	}

	if expand.Filter.Logical != " or " {
		t.Errorf("Expected logical operator ' or ', got '%s'", expand.Filter.Logical)
	}

	if expand.Filter.Left == nil || expand.Filter.Right == nil {
		t.Fatal("Expected left and right filter expressions")
	}
}

// TestParseExpandWithAllNestedOptions tests parsing $expand with all nested options
func TestParseExpandWithAllNestedOptions(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

	params := url.Values{}
	params.Set("$expand", "Books($filter=Title ne 'Archived';$select=Title;$orderby=Title;$top=5;$skip=2)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	expand := options.Expand[0]

	// Check filter
	if expand.Filter == nil {
		t.Error("Expected $filter to be set")
	} else {
		if expand.Filter.Property != "Title" {
			t.Errorf("Expected filter property 'Title', got '%s'", expand.Filter.Property)
		}
		if expand.Filter.Operator != OpNotEqual {
			t.Errorf("Expected filter operator 'ne', got '%s'", expand.Filter.Operator)
		}
	}

	// Check select
	if len(expand.Select) != 1 || expand.Select[0] != "Title" {
		t.Error("Expected $select=Title")
	}

	// Check orderby
	if len(expand.OrderBy) != 1 {
		t.Error("Expected 1 orderby item")
	} else if expand.OrderBy[0].Property != "Title" {
		t.Errorf("Expected orderby property 'Title', got '%s'", expand.OrderBy[0].Property)
	}

	// Check top
	if expand.Top == nil || *expand.Top != 5 {
		t.Error("Expected $top=5")
	}

	// Check skip
	if expand.Skip == nil || *expand.Skip != 2 {
		t.Error("Expected $skip=2")
	}
}

// TestSplitExpandParts tests the expand parts splitting logic
func TestSplitExpandParts(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "Books",
			expected: []string{"Books"},
		},
		{
			input:    "Books($top=5)",
			expected: []string{"Books($top=5)"},
		},
		{
			input:    "Books,Author",
			expected: []string{"Books", "Author"},
		},
		{
			input:    "Books($top=5),Author($select=Name)",
			expected: []string{"Books($top=5)", "Author($select=Name)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitExpandParts(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parts, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Part %d: expected '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}
