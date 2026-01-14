package query

import (
	"net/url"
	"testing"
)

func TestParameterAliases(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name        string
		queryString string
		expectError bool
		validate    func(t *testing.T, opts *QueryOptions)
	}{
		{
			name:        "filter with parameter alias",
			queryString: "$filter=Price gt @p&@p=10",
			expectError: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if opts.Filter == nil {
					t.Error("Expected filter to be parsed")
				}
			},
		},
		{
			name:        "top with parameter alias",
			queryString: "$top=@t&@t=5",
			expectError: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if opts.Top == nil {
					t.Error("Expected top to be parsed")
				} else if *opts.Top != 5 {
					t.Errorf("Expected top=5, got %d", *opts.Top)
				}
			},
		},
		{
			name:        "skip with parameter alias",
			queryString: "$skip=@s&@s=10",
			expectError: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if opts.Skip == nil {
					t.Error("Expected skip to be parsed")
				} else if *opts.Skip != 10 {
					t.Errorf("Expected skip=10, got %d", *opts.Skip)
				}
			},
		},
		{
			name:        "multiple parameter aliases",
			queryString: "$filter=Price gt @min and Price lt @max&@min=10&@max=100",
			expectError: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if opts.Filter == nil {
					t.Error("Expected filter to be parsed")
				}
			},
		},
		{
			name:        "parameter alias with string value",
			queryString: "$filter=Name eq @name&@name='test'",
			expectError: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if opts.Filter == nil {
					t.Error("Expected filter to be parsed")
				}
			},
		},
		{
			name:        "undefined parameter alias",
			queryString: "$filter=Price gt @p",
			expectError: true,
		},
		{
			name:        "empty parameter alias name",
			queryString: "$filter=Price gt 10&@=5",
			expectError: true,
		},
		{
			name:        "orderby with parameter alias",
			queryString: "$orderby=@prop&@prop=Price",
			expectError: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if len(opts.OrderBy) != 1 {
					t.Error("Expected one orderby item")
				} else if opts.OrderBy[0].Property != "Price" {
					t.Errorf("Expected orderby property=Price, got %s", opts.OrderBy[0].Property)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams, err := url.ParseQuery(tt.queryString)
			if err != nil {
				t.Fatalf("Failed to parse query string: %v", err)
			}

			opts, err := ParseQueryOptions(queryParams, meta)
			if tt.expectError {
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
				tt.validate(t, opts)
			}
		})
	}
}

func TestExtractParameterAliases(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		expected    map[string]string
		expectError bool
	}{
		{
			name:        "single parameter alias",
			queryString: "@p=10",
			expected:    map[string]string{"p": "10"},
			expectError: false,
		},
		{
			name:        "multiple parameter aliases",
			queryString: "@p=10&@name=test&@id=5",
			expected:    map[string]string{"p": "10", "name": "test", "id": "5"},
			expectError: false,
		},
		{
			name:        "no parameter aliases",
			queryString: "$filter=Price gt 10",
			expected:    map[string]string{},
			expectError: false,
		},
		{
			name:        "empty alias name",
			queryString: "@=10",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams, err := url.ParseQuery(tt.queryString)
			if err != nil {
				t.Fatalf("Failed to parse query string: %v", err)
			}

			aliases, err := extractParameterAliases(queryParams)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(aliases) != len(tt.expected) {
				t.Errorf("Expected %d aliases, got %d", len(tt.expected), len(aliases))
				return
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := aliases[key]; !ok {
					t.Errorf("Expected alias @%s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected alias @%s=%s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestResolveAliasesInString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		aliases     map[string]string
		expected    string
		expectError bool
	}{
		{
			name:        "single alias",
			input:       "Price gt @p",
			aliases:     map[string]string{"p": "10"},
			expected:    "Price gt 10",
			expectError: false,
		},
		{
			name:        "multiple aliases",
			input:       "@prop gt @min and @prop lt @max",
			aliases:     map[string]string{"prop": "Price", "min": "10", "max": "100"},
			expected:    "Price gt 10 and Price lt 100",
			expectError: false,
		},
		{
			name:        "no aliases",
			input:       "Price gt 10",
			aliases:     map[string]string{},
			expected:    "Price gt 10",
			expectError: false,
		},
		{
			name:        "undefined alias",
			input:       "Price gt @p",
			aliases:     map[string]string{},
			expected:    "",
			expectError: true,
		},
		{
			name:        "alias with underscore",
			input:       "@my_prop eq @my_value",
			aliases:     map[string]string{"my_prop": "Name", "my_value": "'test'"},
			expected:    "Name eq 'test'",
			expectError: false,
		},
		{
			name:        "alias at end of string",
			input:       "Price eq @p",
			aliases:     map[string]string{"p": "10"},
			expected:    "Price eq 10",
			expectError: false,
		},
		{
			name:        "alias in single-quoted string literal",
			input:       "Name eq '@p'",
			aliases:     map[string]string{"p": "'x'"},
			expected:    "Name eq '@p'",
			expectError: false,
		},
		{
			name:        "alias in string literal function call",
			input:       "contains(Name,'@p')",
			aliases:     map[string]string{"p": "'x'"},
			expected:    "contains(Name,'@p')",
			expectError: false,
		},
		{
			name:        "alias outside string literal",
			input:       "Name eq @p",
			aliases:     map[string]string{"p": "'x'"},
			expected:    "Name eq 'x'",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveAliasesInString(tt.input, tt.aliases)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
