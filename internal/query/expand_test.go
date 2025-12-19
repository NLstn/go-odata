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

	if expand.Filter.Logical != "or" {
		t.Errorf("Expected logical operator 'or', got '%s'", expand.Filter.Logical)
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

// TestParseExpandWithComplexFilter tests parsing $expand with complex nested filters
func TestParseExpandWithComplexFilter(t *testing.T) {
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

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
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

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
	authorMeta, _ := metadata.AnalyzeEntity(&TestAuthor{})

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
