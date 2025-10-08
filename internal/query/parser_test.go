package query

import (
	"net/url"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

type TestEntity struct {
	ID          int     `json:"ID" odata:"key"`
	Name        string  `json:"Name"`
	Description string  `json:"Description"`
	Price       float64 `json:"Price"`
	Category    string  `json:"Category"`
}

func getTestMetadata(t *testing.T) *metadata.EntityMetadata {
	meta, err := metadata.AnalyzeEntity(TestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	return meta
}

func TestParseSelect(t *testing.T) {
	tests := []struct {
		name      string
		selectStr string
		expected  []string
	}{
		{
			name:      "Single property",
			selectStr: "Name",
			expected:  []string{"Name"},
		},
		{
			name:      "Multiple properties",
			selectStr: "Name,Price,Category",
			expected:  []string{"Name", "Price", "Category"},
		},
		{
			name:      "Properties with spaces",
			selectStr: "Name, Price, Category",
			expected:  []string{"Name", "Price", "Category"},
		},
		{
			name:      "Empty string",
			selectStr: "",
			expected:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSelect(tt.selectStr)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d properties, got %d", len(tt.expected), len(result))
				return
			}
			for i, prop := range result {
				if prop != tt.expected[i] {
					t.Errorf("Expected property %s at index %d, got %s", tt.expected[i], i, prop)
				}
			}
		})
	}
}

func TestParseOrderBy(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name       string
		orderByStr string
		expected   []OrderByItem
		expectErr  bool
	}{
		{
			name:       "Single property ascending",
			orderByStr: "Name asc",
			expected:   []OrderByItem{{Property: "Name", Descending: false}},
			expectErr:  false,
		},
		{
			name:       "Single property descending",
			orderByStr: "Price desc",
			expected:   []OrderByItem{{Property: "Price", Descending: true}},
			expectErr:  false,
		},
		{
			name:       "Single property no direction",
			orderByStr: "Name",
			expected:   []OrderByItem{{Property: "Name", Descending: false}},
			expectErr:  false,
		},
		{
			name:       "Multiple properties",
			orderByStr: "Category asc, Price desc",
			expected: []OrderByItem{
				{Property: "Category", Descending: false},
				{Property: "Price", Descending: true},
			},
			expectErr: false,
		},
		{
			name:       "Invalid property",
			orderByStr: "InvalidProperty asc",
			expected:   nil,
			expectErr:  true,
		},
		{
			name:       "Invalid direction",
			orderByStr: "Name invalid",
			expected:   nil,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseOrderBy(tt.orderByStr, meta)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
				return
			}
			for i, item := range result {
				if item.Property != tt.expected[i].Property {
					t.Errorf("Expected property %s at index %d, got %s", tt.expected[i].Property, i, item.Property)
				}
				if item.Descending != tt.expected[i].Descending {
					t.Errorf("Expected descending=%v at index %d, got %v", tt.expected[i].Descending, i, item.Descending)
				}
			}
		})
	}
}

func TestParseComparisonFilter(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filterStr string
		expectErr bool
		operator  FilterOperator
		property  string
		value     string
	}{
		{
			name:      "Equal operator",
			filterStr: "Name eq 'Laptop'",
			expectErr: false,
			operator:  OpEqual,
			property:  "Name",
			value:     "Laptop",
		},
		{
			name:      "Greater than operator",
			filterStr: "Price gt 100",
			expectErr: false,
			operator:  OpGreaterThan,
			property:  "Price",
			value:     "100",
		},
		{
			name:      "Less than or equal operator",
			filterStr: "Price le 999.99",
			expectErr: false,
			operator:  OpLessThanOrEqual,
			property:  "Price",
			value:     "999.99",
		},
		{
			name:      "Not equal operator",
			filterStr: "Category ne 'Electronics'",
			expectErr: false,
			operator:  OpNotEqual,
			property:  "Category",
			value:     "Electronics",
		},
		{
			name:      "Invalid property",
			filterStr: "InvalidProp eq 'value'",
			expectErr: true,
		},
		{
			name:      "Invalid operator",
			filterStr: "Name invalid 'value'",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseComparisonFilter(tt.filterStr, meta)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result.Operator != tt.operator {
				t.Errorf("Expected operator %s, got %s", tt.operator, result.Operator)
			}
			if result.Property != tt.property {
				t.Errorf("Expected property %s, got %s", tt.property, result.Property)
			}
			if result.Value != tt.value {
				t.Errorf("Expected value %s, got %v", tt.value, result.Value)
			}
		})
	}
}

func TestParseFunctionFilter(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filterStr string
		expectErr bool
		operator  FilterOperator
		property  string
		value     string
	}{
		{
			name:      "Contains function",
			filterStr: "contains(Name,'laptop')",
			expectErr: false,
			operator:  OpContains,
			property:  "Name",
			value:     "laptop",
		},
		{
			name:      "StartsWith function",
			filterStr: "startswith(Category,'Elec')",
			expectErr: false,
			operator:  OpStartsWith,
			property:  "Category",
			value:     "Elec",
		},
		{
			name:      "EndsWith function",
			filterStr: "endswith(Description,'technology')",
			expectErr: false,
			operator:  OpEndsWith,
			property:  "Description",
			value:     "technology",
		},
		{
			name:      "Contains with spaces",
			filterStr: "contains(Name, 'test value')",
			expectErr: false,
			operator:  OpContains,
			property:  "Name",
			value:     "test value",
		},
		{
			name:      "Invalid property",
			filterStr: "contains(InvalidProp,'value')",
			expectErr: true,
		},
		{
			name:      "Unsupported function",
			filterStr: "unsupported(Name,'value')",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFunctionFilter(tt.filterStr, meta)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result.Operator != tt.operator {
				t.Errorf("Expected operator %s, got %s", tt.operator, result.Operator)
			}
			if result.Property != tt.property {
				t.Errorf("Expected property %s, got %s", tt.property, result.Property)
			}
			if result.Value != tt.value {
				t.Errorf("Expected value %s, got %v", tt.value, result.Value)
			}
		})
	}
}

func TestParseQueryOptions(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		query     string
		expectErr bool
		validate  func(*testing.T, *QueryOptions)
	}{
		{
			name:      "Filter only",
			query:     "$filter=Price gt 100",
			expectErr: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if opts.Filter == nil {
					t.Error("Expected filter to be set")
				}
			},
		},
		{
			name:      "Select only",
			query:     "$select=Name,Price",
			expectErr: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if len(opts.Select) != 2 {
					t.Errorf("Expected 2 selected properties, got %d", len(opts.Select))
				}
			},
		},
		{
			name:      "OrderBy only",
			query:     "$orderby=Price desc",
			expectErr: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if len(opts.OrderBy) != 1 {
					t.Errorf("Expected 1 orderby item, got %d", len(opts.OrderBy))
				}
			},
		},
		{
			name:      "All options combined",
			query:     "$filter=Price gt 100&$select=Name,Price&$orderby=Name asc",
			expectErr: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if opts.Filter == nil {
					t.Error("Expected filter to be set")
				}
				if len(opts.Select) != 2 {
					t.Errorf("Expected 2 selected properties, got %d", len(opts.Select))
				}
				if len(opts.OrderBy) != 1 {
					t.Errorf("Expected 1 orderby item, got %d", len(opts.OrderBy))
				}
			},
		},
		{
			name:      "Invalid filter",
			query:     "$filter=InvalidProperty eq 'value'",
			expectErr: true,
		},
		{
			name:      "Invalid orderby",
			query:     "$orderby=InvalidProperty desc",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryValues, _ := url.ParseQuery(tt.query)
			result, err := ParseQueryOptions(queryValues, meta)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestSplitFunctionArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		expected []string
	}{
		{
			name:     "Two simple args",
			args:     "Name,'value'",
			expected: []string{"Name", "'value'"},
		},
		{
			name:     "Args with spaces",
			args:     "Name, 'value with spaces'",
			expected: []string{"Name", " 'value with spaces'"},
		},
		{
			name:     "Args with comma in quotes",
			args:     "Name,'value, with, commas'",
			expected: []string{"Name", "'value, with, commas'"},
		},
		{
			name:     "Single arg",
			args:     "Name",
			expected: []string{"Name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitFunctionArgs(tt.args)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d args, got %d", len(tt.expected), len(result))
				return
			}
			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("Expected arg %s at index %d, got %s", tt.expected[i], i, arg)
				}
			}
		})
	}
}
