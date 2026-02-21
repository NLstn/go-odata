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

type TestPublisher struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name"`
}

type TestAuthorWithPublisher struct {
	ID          uint                    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string                  `json:"Name"`
	PublisherID uint                    `json:"PublisherID"`
	Publisher   *TestPublisher          `json:"Publisher,omitempty" gorm:"foreignKey:PublisherID"`
	Books       []TestBookWithPublisher `json:"Books" gorm:"foreignKey:AuthorID"`
}

type TestBookWithPublisher struct {
	ID       uint                     `json:"ID" gorm:"primaryKey" odata:"key"`
	Title    string                   `json:"Title"`
	AuthorID uint                     `json:"AuthorID"`
	Author   *TestAuthorWithPublisher `json:"Author,omitempty" gorm:"foreignKey:AuthorID"`
}

func buildAuthorBookMetadata(t *testing.T) (*metadata.EntityMetadata, *metadata.EntityMetadata) {
	t.Helper()

	authorMeta, err := metadata.AnalyzeEntity(&TestAuthor{})
	if err != nil {
		t.Fatalf("Failed to analyze author entity: %v", err)
	}

	bookMeta, err := metadata.AnalyzeEntity(&TestBook{})
	if err != nil {
		t.Fatalf("Failed to analyze book entity: %v", err)
	}

	setEntitiesRegistry(authorMeta, bookMeta)

	return authorMeta, bookMeta
}

func setEntitiesRegistry(metas ...*metadata.EntityMetadata) {
	entities := make(map[string]*metadata.EntityMetadata, len(metas))
	for _, meta := range metas {
		if meta == nil {
			continue
		}
		entities[meta.EntitySetName] = meta
	}
	for _, meta := range metas {
		if meta != nil {
			meta.SetEntitiesRegistry(entities)
		}
	}
}

// TestParseExpandSimple tests parsing a simple $expand
func TestParseExpandSimple(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

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
	authorMeta, _ := buildAuthorBookMetadata(t)

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
	authorMeta, _ := buildAuthorBookMetadata(t)

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

func TestParseExpandWithNegativeNestedTopOrSkip(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	tests := []struct {
		name        string
		expandQuery string
	}{
		{
			name:        "Negative nested top",
			expandQuery: "Books($top=-1)",
		},
		{
			name:        "Negative nested skip",
			expandQuery: "Books($skip=-2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("$expand", tt.expandQuery)

			_, err := ParseQueryOptions(params, authorMeta)
			if err == nil {
				t.Fatalf("Expected error for %s", tt.expandQuery)
			}
		})
	}
}

// TestParseExpandWithNestedSelect tests parsing $expand with nested $select
func TestParseExpandWithNestedSelect(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

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
	authorMeta, _ := buildAuthorBookMetadata(t)

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
	authorMeta, _ := buildAuthorBookMetadata(t)

	params := url.Values{}
	params.Set("$expand", "InvalidProperty")

	_, err := ParseQueryOptions(params, authorMeta)
	if err == nil {
		t.Error("Expected error for invalid navigation property")
	}
}

func TestParseExpandUnbalancedParentheses(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	tests := []struct {
		name        string
		expandQuery string
	}{
		{
			name:        "Extra closing parenthesis",
			expandQuery: "Books)",
		},
		{
			name:        "Missing closing parenthesis",
			expandQuery: "Books($select=Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("$expand", tt.expandQuery)

			_, err := ParseQueryOptions(params, authorMeta)
			if err == nil {
				t.Fatalf("Expected error for %s", tt.expandQuery)
			}
		})
	}
}

// TestParseExpandMultiple tests parsing multiple $expand values
func TestParseExpandMultiple(t *testing.T) {
	// For this test, we need a more complex entity structure
	// Since we only have Author->Books, we'll just test the parsing logic
	authorMeta, _ := buildAuthorBookMetadata(t)

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
	authorMeta, _ := buildAuthorBookMetadata(t)

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
	authorMeta, _ := buildAuthorBookMetadata(t)

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
	authorMeta, _ := buildAuthorBookMetadata(t)

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
	authorMeta, _ := buildAuthorBookMetadata(t)

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
	authorMeta, _ := buildAuthorBookMetadata(t)

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
	authorMeta, _ := buildAuthorBookMetadata(t)

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

	if expand.Filter.Logical != "or" {
		t.Errorf("Expected logical operator 'or', got '%s'", expand.Filter.Logical)
	}

	if expand.Filter.Left == nil || expand.Filter.Right == nil {
		t.Fatal("Expected left and right filter expressions")
	}
}

// TestParseExpandWithAllNestedOptions tests parsing $expand with all nested options
func TestParseExpandWithAllNestedOptions(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

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

func TestParseExpandWithInvalidNestedOptions(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	tests := []struct {
		name        string
		expandQuery string
	}{
		{
			name:        "Invalid nested select",
			expandQuery: "Books($select=DoesNotExist)",
		},
		{
			name:        "Invalid nested filter",
			expandQuery: "Books($filter=DoesNotExist eq 1)",
		},
		{
			name:        "Invalid nested orderby",
			expandQuery: "Books($orderby=DoesNotExist desc)",
		},
		{
			name:        "Invalid nested compute",
			expandQuery: "Books($compute=DoesNotExist eq 1 as Total)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("$expand", tt.expandQuery)

			_, err := ParseQueryOptions(params, authorMeta)
			if err == nil {
				t.Fatalf("Expected error for %s", tt.expandQuery)
			}
		})
	}
}

// TestSplitExpandParts tests the expand parts splitting logic
func TestSplitExpandParts(t *testing.T) {
	tests := []struct {
		input     string
		expected  []string
		expectErr bool
		errMsg    string
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
		{
			input:    "Books($filter=contains(Title,'a,b')),Author",
			expected: []string{"Books($filter=contains(Title,'a,b'))", "Author"},
		},
		{
			input:    "Books($filter=contains(Title,'O''Reilly, Inc.')),Author",
			expected: []string{"Books($filter=contains(Title,'O''Reilly, Inc.'))", "Author"},
		},
		{
			input:     "Books)",
			expectErr: true,
		},
		{
			input:     "Books($top=5",
			expectErr: true,
		},
		{
			input:     "Books($filter=contains(Title,'unclosed))",
			expectErr: true,
			errMsg:    "invalid $expand syntax: missing closing quote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := splitExpandParts(tt.input)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Expected error %v, got %v", tt.expectErr, err)
			}
			if tt.expectErr {
				if tt.errMsg != "" && err != nil && err.Error() != tt.errMsg {
					t.Fatalf("Expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
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

func TestParseExpandWithFilterContainingComma(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	params := url.Values{}
	params.Set("$expand", "Books($filter=contains(Title,'a,b'))")

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

	if expand.Filter.Operator != OpContains {
		t.Errorf("Expected operator 'contains', got '%s'", expand.Filter.Operator)
	}

	if expand.Filter.Value != "a,b" {
		t.Errorf("Expected value 'a,b', got '%v'", expand.Filter.Value)
	}
}

func TestParseNestedExpandWithFilterContainingComma(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	params := url.Values{}
	params.Set("$expand", "Books($expand=Author($filter=contains(Name,'a,b')))")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	firstLevel := options.Expand[0]
	if len(firstLevel.Expand) != 1 {
		t.Fatalf("Expected 1 nested expand option, got %d", len(firstLevel.Expand))
	}

	nestedExpand := firstLevel.Expand[0]
	if nestedExpand.Filter == nil {
		t.Fatal("Expected nested $filter to be set")
	}

	if nestedExpand.Filter.Property != "Name" {
		t.Errorf("Expected property 'Name', got '%s'", nestedExpand.Filter.Property)
	}

	if nestedExpand.Filter.Operator != OpContains {
		t.Errorf("Expected operator 'contains', got '%s'", nestedExpand.Filter.Operator)
	}

	if nestedExpand.Filter.Value != "a,b" {
		t.Errorf("Expected value 'a,b', got '%v'", nestedExpand.Filter.Value)
	}
}

func TestParseNestedExpandWithInnerSemicolonOptions(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	params := url.Values{}
	params.Set("$expand", "Books($expand=Author($select=Name;$filter=contains(Name,'A;B'));$top=2)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	booksExpand := options.Expand[0]
	if booksExpand.Top == nil || *booksExpand.Top != 2 {
		t.Fatalf("Expected outer $top=2")
	}

	if len(booksExpand.Expand) != 1 {
		t.Fatalf("Expected 1 nested expand option, got %d", len(booksExpand.Expand))
	}

	authorExpand := booksExpand.Expand[0]
	if len(authorExpand.Select) != 1 || authorExpand.Select[0] != "Name" {
		t.Fatalf("Expected nested $select=Name")
	}

	if authorExpand.Filter == nil {
		t.Fatalf("Expected nested $filter to be parsed")
	}

	if authorExpand.Filter.Value != "A;B" {
		t.Fatalf("Expected nested filter value A;B, got %v", authorExpand.Filter.Value)
	}
}

func TestParseExpandWithSemicolonInQuotedLiteral(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	params := url.Values{}
	params.Set("$expand", "Books($filter=contains(Title,'A;B');$select=Title)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	expand := options.Expand[0]
	if expand.Filter == nil {
		t.Fatalf("Expected filter to be parsed")
	}

	if expand.Filter.Value != "A;B" {
		t.Fatalf("Expected filter value A;B, got %v", expand.Filter.Value)
	}

	if len(expand.Select) != 1 || expand.Select[0] != "Title" {
		t.Fatalf("Expected $select=Title")
	}
}

func TestParseExpandWithMixedNestedOptionsAndInnerExpand(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	params := url.Values{}
	params.Set("$expand", "Books($select=Title;$filter=contains(Title,'A;B');$orderby=Title desc;$expand=Author($select=Name;$filter=contains(Name,'X;Y')))")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	booksExpand := options.Expand[0]
	if len(booksExpand.Select) != 1 || booksExpand.Select[0] != "Title" {
		t.Fatalf("Expected books $select=Title")
	}

	if booksExpand.Filter == nil || booksExpand.Filter.Value != "A;B" {
		t.Fatalf("Expected books filter with value A;B")
	}

	if len(booksExpand.OrderBy) != 1 || booksExpand.OrderBy[0].Property != "Title" || !booksExpand.OrderBy[0].Descending {
		t.Fatalf("Expected books $orderby=Title desc")
	}

	if len(booksExpand.Expand) != 1 {
		t.Fatalf("Expected 1 nested author expand, got %d", len(booksExpand.Expand))
	}

	authorExpand := booksExpand.Expand[0]
	if len(authorExpand.Select) != 1 || authorExpand.Select[0] != "Name" {
		t.Fatalf("Expected author $select=Name")
	}

	if authorExpand.Filter == nil || authorExpand.Filter.Value != "X;Y" {
		t.Fatalf("Expected author filter with value X;Y")
	}
}

func TestSplitExpandOptionsParts(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []string
		expectErr bool
	}{
		{
			name:     "simple",
			input:    "$select=Title;$top=1",
			expected: []string{"$select=Title", "$top=1"},
		},
		{
			name:     "nested expand with inner semicolons",
			input:    "$select=Title;$expand=Author($select=Name;$filter=contains(Name,'A;B'));$top=1",
			expected: []string{"$select=Title", "$expand=Author($select=Name;$filter=contains(Name,'A;B'))", "$top=1"},
		},
		{
			name:     "quoted semicolon and escaped quote",
			input:    "$filter=contains(Title,'O''Reilly; Inc.');$orderby=Title",
			expected: []string{"$filter=contains(Title,'O''Reilly; Inc.')", "$orderby=Title"},
		},
		{
			name:      "unclosed quote",
			input:     "$filter=contains(Title,'bad;$select=Title",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, err := splitExpandOptionsParts(tt.input)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Expected error=%v, got err=%v", tt.expectErr, err)
			}

			if tt.expectErr {
				return
			}

			if len(parts) != len(tt.expected) {
				t.Fatalf("Expected %d parts, got %d", len(tt.expected), len(parts))
			}

			for i := range tt.expected {
				if parts[i] != tt.expected[i] {
					t.Fatalf("Part %d expected %q, got %q", i, tt.expected[i], parts[i])
				}
			}
		})
	}
}

// TestParseExpandWithComplexFilter tests parsing $expand with complex nested filters
func TestParseExpandWithComplexFilter(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	tests := []struct {
		name        string
		expandQuery string
		expectErr   bool
		description string
	}{
		{
			name:        "Filter with parentheses",
			expandQuery: "Books($filter=(Title eq 'Book1' or Title eq 'Book2'))",
			expectErr:   false,
			description: "Should support parentheses in nested filters",
		},
		{
			name:        "Filter with NOT operator",
			expandQuery: "Books($filter=not (Title eq 'Excluded'))",
			expectErr:   false,
			description: "Should support NOT operator in nested filters",
		},
		{
			name:        "Complex nested filter",
			expandQuery: "Books($filter=(Title eq 'Book1' and ID gt 10) or (Title eq 'Book2' and ID lt 5))",
			expectErr:   false,
			description: "Should support complex boolean combinations",
		},
		{
			name:        "Filter with function and boolean logic",
			expandQuery: "Books($filter=contains(Title,'Test') and not (ID eq 999))",
			expectErr:   false,
			description: "Should support functions with NOT and boolean logic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("$expand", tt.expandQuery)

			options, err := ParseQueryOptions(params, authorMeta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
				return
			}

			if !tt.expectErr {
				if len(options.Expand) != 1 {
					t.Errorf("Expected 1 expand option, got %d", len(options.Expand))
					return
				}

				if options.Expand[0].Filter == nil {
					t.Error("Expected filter to be set in expand option")
				}
			}
		})
	}
}

// TestParseExpandWithMultipleLevels tests multi-level expand (future support)
func TestParseExpandWithMultipleLevels(t *testing.T) {
	// Define entities with multi-level relationships
	type Publisher struct {
		ID      uint         `json:"ID" gorm:"primaryKey" odata:"key"`
		Name    string       `json:"Name"`
		Authors []TestAuthor `json:"Authors" gorm:"foreignKey:PublisherID"`
	}

	type TestAuthorWithPublisher struct {
		ID          uint       `json:"ID" gorm:"primaryKey" odata:"key"`
		Name        string     `json:"Name"`
		PublisherID uint       `json:"PublisherID"`
		Publisher   *Publisher `json:"Publisher,omitempty" gorm:"foreignKey:PublisherID"`
		Books       []TestBook `json:"Books" gorm:"foreignKey:AuthorID"`
	}

	authorMeta, err := metadata.AnalyzeEntity(&TestAuthorWithPublisher{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	publisherMeta, err := metadata.AnalyzeEntity(&Publisher{})
	if err != nil {
		t.Fatalf("Failed to analyze publisher entity: %v", err)
	}

	bookMeta, err := metadata.AnalyzeEntity(&TestBook{})
	if err != nil {
		t.Fatalf("Failed to analyze book entity: %v", err)
	}

	setEntitiesRegistry(authorMeta, publisherMeta, bookMeta)

	tests := []struct {
		name        string
		expandQuery string
		expectErr   bool
		description string
	}{
		{
			name:        "Simple two-level expand",
			expandQuery: "Books,Publisher",
			expectErr:   false,
			description: "Should support expanding multiple navigation properties",
		},
		{
			name:        "Expand with nested options on multiple properties",
			expandQuery: "Books($top=5),Publisher($select=Name)",
			expectErr:   false,
			description: "Should support different nested options on different properties",
		},
		{
			name:        "Complex expand with filters on multiple levels",
			expandQuery: "Books($filter=Title eq 'Test';$top=10),Publisher($filter=Name ne 'Excluded')",
			expectErr:   false,
			description: "Should support filters on multiple expanded properties",
		},
		{
			name:        "Nested expand - Books with nested Author expand",
			expandQuery: "Books($expand=Author)",
			expectErr:   false,
			description: "Should support nested $expand",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("$expand", tt.expandQuery)

			options, err := ParseQueryOptions(params, authorMeta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
				return
			}

			if !tt.expectErr && len(options.Expand) == 0 {
				t.Error("Expected at least one expand option")
			}
		})
	}
}

// TestParseNestedExpand tests parsing of nested $expand syntax
func TestParseNestedExpand(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	params := url.Values{}
	params.Set("$expand", "Books($expand=Author)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	// Check the first level expand
	firstLevel := options.Expand[0]
	if firstLevel.NavigationProperty != "Books" {
		t.Errorf("Expected navigation property 'Books', got '%s'", firstLevel.NavigationProperty)
	}

	// Check the nested expand
	if len(firstLevel.Expand) != 1 {
		t.Fatalf("Expected 1 nested expand option, got %d", len(firstLevel.Expand))
	}

	nestedExpand := firstLevel.Expand[0]
	if nestedExpand.NavigationProperty != "Author" {
		t.Errorf("Expected nested navigation property 'Author', got '%s'", nestedExpand.NavigationProperty)
	}
}

// TestParseNestedExpandWithOptions tests nested expand with additional query options
func TestParseNestedExpandWithOptions(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	params := url.Values{}
	params.Set("$expand", "Books($expand=Author($select=Name);$top=5)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	firstLevel := options.Expand[0]
	if firstLevel.NavigationProperty != "Books" {
		t.Errorf("Expected navigation property 'Books', got '%s'", firstLevel.NavigationProperty)
	}

	// Check $top on Books
	if firstLevel.Top == nil || *firstLevel.Top != 5 {
		t.Error("Expected $top=5 on Books expand")
	}

	// Check nested expand
	if len(firstLevel.Expand) != 1 {
		t.Fatalf("Expected 1 nested expand option, got %d", len(firstLevel.Expand))
	}

	nestedExpand := firstLevel.Expand[0]
	if nestedExpand.NavigationProperty != "Author" {
		t.Errorf("Expected nested navigation property 'Author', got '%s'", nestedExpand.NavigationProperty)
	}

	// Check $select on nested Author
	if len(nestedExpand.Select) != 1 || nestedExpand.Select[0] != "Name" {
		t.Error("Expected $select=Name on nested Author expand")
	}
}

func TestParseNestedExpandDepthLimit(t *testing.T) {
	authorMeta, err := metadata.AnalyzeEntity(&TestAuthorWithPublisher{})
	if err != nil {
		t.Fatalf("Failed to analyze author entity: %v", err)
	}

	bookMeta, err := metadata.AnalyzeEntity(&TestBookWithPublisher{})
	if err != nil {
		t.Fatalf("Failed to analyze book entity: %v", err)
	}

	publisherMeta, err := metadata.AnalyzeEntity(&TestPublisher{})
	if err != nil {
		t.Fatalf("Failed to analyze publisher entity: %v", err)
	}

	setEntitiesRegistry(authorMeta, bookMeta, publisherMeta)

	params := url.Values{}
	params.Set("$expand", "Books($expand=Author($expand=Publisher))")

	_, err = ParseQueryOptionsWithConfig(params, authorMeta, &ParserConfig{MaxExpandDepth: 1})
	if err == nil {
		t.Fatal("Expected expand depth error, got nil")
	}

	expectedErr := "invalid $expand: invalid nested $expand: $expand nesting level (2) exceeds maximum allowed depth (1)"
	if err.Error() != expectedErr {
		t.Fatalf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

// TestParseMultiLevelNestedExpand tests deeply nested expand syntax
func TestParseMultiLevelNestedExpand(t *testing.T) {
	// Define entities with multi-level relationships
	type Club struct {
		ID   string `json:"ID" gorm:"primaryKey" odata:"key"`
		Name string `json:"Name"`
	}

	type Member struct {
		ID     string `json:"ID" gorm:"primaryKey" odata:"key"`
		UserID string `json:"UserID"`
		ClubID string `json:"ClubID"`
		Role   string `json:"Role"`
		Club   *Club  `json:"Club,omitempty" gorm:"foreignKey:ClubID" odata:"nav"`
	}

	type User struct {
		ID      string   `json:"ID" gorm:"primaryKey" odata:"key"`
		Name    string   `json:"Name"`
		Members []Member `gorm:"foreignKey:UserID" json:"Members,omitempty" odata:"nav"`
	}

	userMeta, err := metadata.AnalyzeEntity(&User{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	memberMeta, err := metadata.AnalyzeEntity(&Member{})
	if err != nil {
		t.Fatalf("Failed to analyze member entity: %v", err)
	}

	clubMeta, err := metadata.AnalyzeEntity(&Club{})
	if err != nil {
		t.Fatalf("Failed to analyze club entity: %v", err)
	}

	setEntitiesRegistry(userMeta, memberMeta, clubMeta)

	params := url.Values{}
	params.Set("$expand", "Members($expand=Club)")

	options, err := ParseQueryOptions(params, userMeta)
	if err != nil {
		t.Fatalf("Failed to parse query options: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	// Check first level (Members)
	firstLevel := options.Expand[0]
	if firstLevel.NavigationProperty != "Members" {
		t.Errorf("Expected navigation property 'Members', got '%s'", firstLevel.NavigationProperty)
	}

	// Check nested expand (Club)
	if len(firstLevel.Expand) != 1 {
		t.Fatalf("Expected 1 nested expand option, got %d", len(firstLevel.Expand))
	}

	nestedExpand := firstLevel.Expand[0]
	if nestedExpand.NavigationProperty != "Club" {
		t.Errorf("Expected nested navigation property 'Club', got '%s'", nestedExpand.NavigationProperty)
	}
}

// TestComplexFilterCombinations tests complex filter expressions at the top level
func TestComplexFilterCombinations(t *testing.T) {
	type TestProduct struct {
		ID          int     `json:"ID" odata:"key"`
		Name        string  `json:"Name"`
		Price       float64 `json:"Price"`
		Category    string  `json:"Category"`
		IsAvailable bool    `json:"IsAvailable"`
		Quantity    int     `json:"Quantity"`
	}

	productMeta, err := metadata.AnalyzeEntity(&TestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "Nested boolean groups with parentheses",
			filter:    "(Price gt 100 and Category eq 'Electronics') or (Price lt 50 and Category eq 'Books')",
			expectErr: false,
		},
		{
			name:      "Multiple levels of grouping",
			filter:    "((Price gt 100 or Price lt 10) and (Category eq 'A' or Category eq 'B')) or Name eq 'Test'",
			expectErr: false,
		},
		{
			name:      "NOT with complex expressions",
			filter:    "not ((Price gt 1000 or Category eq 'Luxury') and IsAvailable eq true)",
			expectErr: false,
		},
		{
			name:      "Multiple NOT operators",
			filter:    "not (Price gt 100) and not (Category eq 'Books') and IsAvailable eq true",
			expectErr: false,
		},
		{
			name:      "Mixed functions and boolean logic",
			filter:    "(contains(Name,'Laptop') or contains(Name,'Computer')) and Price gt 500 and Category eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "Deep nesting with NOT",
			filter:    "not (not (Price gt 100 and Category eq 'Electronics'))",
			expectErr: false,
		},
		{
			name:      "Boolean literal",
			filter:    "IsAvailable eq true",
			expectErr: false,
		},
		{
			name:      "Arithmetic modulo (limited support)",
			filter:    "Quantity mod 2 eq 0",
			expectErr: false, // Basic arithmetic supported
		},
		{
			name:      "Complex multi-condition filter",
			filter:    "((Price gt 100 and not (Category eq 'Books')) or contains(Name,'Special')) and IsAvailable eq true",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("$filter", tt.filter)

			options, err := ParseQueryOptions(params, productMeta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v (filter: %s)", tt.expectErr, err, tt.filter)
				return
			}

			if !tt.expectErr && options.Filter == nil {
				t.Error("Expected filter to be set")
			}
		})
	}
}

// TestParseOrderByWithMultipleProperties tests ordering by multiple properties
func TestParseOrderByWithMultipleProperties(t *testing.T) {
	type TestProduct struct {
		ID       int     `json:"ID" odata:"key"`
		Name     string  `json:"Name"`
		Price    float64 `json:"Price"`
		Category string  `json:"Category"`
	}

	productMeta, err := metadata.AnalyzeEntity(&TestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	tests := []struct {
		name          string
		orderBy       string
		expectedCount int
		expectErr     bool
	}{
		{
			name:          "Multiple properties with mixed directions",
			orderBy:       "Category asc,Price desc,Name asc",
			expectedCount: 3,
			expectErr:     false,
		},
		{
			name:          "Multiple properties without explicit direction",
			orderBy:       "Category,Price,Name",
			expectedCount: 3,
			expectErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("$orderby", tt.orderBy)

			options, err := ParseQueryOptions(params, productMeta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
				return
			}

			if !tt.expectErr {
				if len(options.OrderBy) != tt.expectedCount {
					t.Errorf("Expected %d orderby items, got %d", tt.expectedCount, len(options.OrderBy))
				}
			}
		})
	}
}

// TestParseExpandWithNestedCount tests parsing $expand with nested $count
func TestParseExpandWithNestedCount(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	tests := []struct {
		name        string
		expandQuery string
		expectCount bool
		expectErr   bool
	}{
		{
			name:        "Count true",
			expandQuery: "Books($count=true)",
			expectCount: true,
			expectErr:   false,
		},
		{
			name:        "Count false - allowed",
			expandQuery: "Books($count=false)",
			expectCount: false,
			expectErr:   false, // count=false is allowed (no-op)
		},
		{
			name:        "Invalid count value",
			expandQuery: "Books($count=invalid)",
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("$expand", tt.expandQuery)

			options, err := ParseQueryOptions(params, authorMeta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
				return
			}

			if !tt.expectErr {
				if len(options.Expand) != 1 {
					t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
				}

				if options.Expand[0].Count != tt.expectCount {
					t.Errorf("Expected Count=%v, got %v", tt.expectCount, options.Expand[0].Count)
				}
			}
		})
	}
}

// TestParseExpandWithNestedLevels tests parsing $expand with nested $levels
func TestParseExpandWithNestedLevels(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	tests := []struct {
		name        string
		expandQuery string
		expectErr   bool
		description string
	}{
		{
			name:        "Levels with integer value",
			expandQuery: "Books($levels=2)",
			expectErr:   false,
			description: "Should accept numeric levels",
		},
		{
			name:        "Levels with max",
			expandQuery: "Books($levels=max)",
			expectErr:   false,
			description: "Should accept 'max'",
		},
		{
			name:        "Invalid levels - zero",
			expandQuery: "Books($levels=0)",
			expectErr:   true,
			description: "Should reject zero",
		},
		{
			name:        "Invalid levels - negative",
			expandQuery: "Books($levels=-5)",
			expectErr:   true,
			description: "Should reject negative numbers",
		},
		{
			name:        "Invalid levels - text",
			expandQuery: "Books($levels=invalid)",
			expectErr:   true,
			description: "Should reject non-numeric non-max values",
		},
		{
			name:        "Invalid levels - partial numeric",
			expandQuery: "Books($levels=5abc)",
			expectErr:   true,
			description: "Should reject values with trailing non-numeric characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := url.Values{}
			params.Set("$expand", tt.expandQuery)

			options, err := ParseQueryOptions(params, authorMeta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v (%s)", tt.expectErr, err, tt.description)
				return
			}

			if !tt.expectErr {
				if len(options.Expand) != 1 {
					t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
				}

				expand := options.Expand[0]
				if expand.Levels != nil {
					t.Errorf("Expected Levels to be nil after normalization, got %v", *expand.Levels)
				}
			}
		})
	}
}

// TestParseExpandWithCountAndLevels tests parsing $expand with both $count and $levels
func TestParseExpandWithCountAndLevels(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	params := url.Values{}
	params.Set("$expand", "Books($count=true;$levels=3)")

	options, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}
	if !options.Expand[0].Count {
		t.Fatal("Expected Count to be true")
	}
	if options.Expand[0].Levels != nil {
		t.Fatalf("Expected Levels to be nil after normalization, got %v", options.Expand[0].Levels)
	}
}

func TestParseExpandWithRecursiveLevels(t *testing.T) {
	type Node struct {
		ID       uint   `json:"ID" gorm:"primaryKey" odata:"key"`
		Name     string `json:"Name"`
		Children []Node `json:"Children,omitempty" gorm:"foreignKey:ParentID"`
		ParentID *uint  `json:"ParentID"`
	}

	nodeMeta, err := metadata.AnalyzeEntity(&Node{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	setEntitiesRegistry(nodeMeta)

	params := url.Values{}
	params.Set("$expand", "Children($levels=2)")

	options, err := ParseQueryOptions(params, nodeMeta)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(options.Expand))
	}

	if options.Expand[0].Levels != nil {
		t.Fatalf("Expected Levels to be nil after normalization, got %v", options.Expand[0].Levels)
	}

	if len(options.Expand[0].Expand) == 0 {
		t.Fatal("Expected recursive expand to be generated for $levels")
	}
}

// TestParseExpandWithAllNestedOptionsIncludingCountAndLevels tests all nested options together
func TestParseExpandWithAllNestedOptionsIncludingCountAndLevels(t *testing.T) {
	authorMeta, _ := buildAuthorBookMetadata(t)

	params := url.Values{}
	params.Set("$expand", "Books($filter=Title ne 'Archived';$select=Title;$orderby=Title;$top=5;$skip=2;$count=true;$levels=2)")

	_, err := ParseQueryOptions(params, authorMeta)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
