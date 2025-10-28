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
			result, err := parseOrderBy(tt.orderByStr, meta, nil)
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
		{
			name:      "Invalid query option",
			query:     "$invalidQuery=1234",
			expectErr: true,
		},
		{
			name:      "Multiple invalid query options",
			query:     "$invalidOption=value&$anotherInvalid=test",
			expectErr: true,
		},
		{
			name:      "Valid and invalid query options mixed",
			query:     "$filter=Price gt 100&$invalidQuery=1234",
			expectErr: true,
		},
		{
			name:      "Non-$ prefixed parameter should not cause error",
			query:     "$filter=Price gt 100&customParam=value",
			expectErr: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if opts.Filter == nil {
					t.Error("Expected filter to be set")
				}
			},
		},
		{
			name:      "Delta token",
			query:     "$deltatoken=abc123",
			expectErr: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if opts.DeltaToken == nil || *opts.DeltaToken != "abc123" {
					t.Fatalf("Expected delta token to be parsed")
				}
			},
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

func TestParseSearchOption(t *testing.T) {
	tests := []struct {
		name        string
		searchQuery string
		expectError bool
		expected    string
	}{
		{
			name:        "Valid search query",
			searchQuery: "laptop",
			expectError: false,
			expected:    "laptop",
		},
		{
			name:        "Search query with spaces",
			searchQuery: "  laptop pro  ",
			expectError: false,
			expected:    "laptop pro",
		},
		{
			name:        "Empty search query",
			searchQuery: "",
			expectError: false,
			expected:    "",
		},
		{
			name:        "Search query with only spaces",
			searchQuery: "   ",
			expectError: true,
			expected:    "",
		},
		{
			name:        "Multi-word search",
			searchQuery: "high performance laptop",
			expectError: false,
			expected:    "high performance laptop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams := url.Values{}
			if tt.searchQuery != "" || tt.expectError {
				queryParams.Set("$search", tt.searchQuery)
			}

			options := &QueryOptions{}
			err := parseSearchOption(queryParams, options)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && options.Search != tt.expected {
				t.Errorf("Expected search query '%s', got '%s'", tt.expected, options.Search)
			}
		})
	}
}

func TestParseQueryOptions_WithSearch(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name           string
		queryString    string
		expectError    bool
		expectedSearch string
	}{
		{
			name:           "Valid search parameter",
			queryString:    "$search=laptop",
			expectError:    false,
			expectedSearch: "laptop",
		},
		{
			name:           "Search with filter",
			queryString:    "$search=laptop&$filter=Price gt 500",
			expectError:    false,
			expectedSearch: "laptop",
		},
		{
			name:           "Search with multiple query options",
			queryString:    "$search=gaming&$top=10&$skip=5&$orderby=Price desc",
			expectError:    false,
			expectedSearch: "gaming",
		},
		{
			name:           "Empty search value",
			queryString:    "$search=   ",
			expectError:    true,
			expectedSearch: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams, err := url.ParseQuery(tt.queryString)
			if err != nil {
				t.Fatalf("Failed to parse query string: %v", err)
			}

			options, err := ParseQueryOptions(queryParams, meta)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && options != nil && options.Search != tt.expectedSearch {
				t.Errorf("Expected search query '%s', got '%s'", tt.expectedSearch, options.Search)
			}
		})
	}
}
