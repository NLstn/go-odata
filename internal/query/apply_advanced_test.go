package query

import (
	"testing"
)

// TestParseApply_FilterAfterGroupBy tests that filters after groupby/aggregate
// properly track computed aliases
func TestParseApply_FilterAfterGroupBy(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		applyStr  string
		expectErr bool
		validate  func(*testing.T, []ApplyTransformation)
	}{
		{
			name:      "Filter on aggregate alias after groupby",
			applyStr:  "groupby((Category),aggregate(Price with sum as Total))/filter(Total gt 100)",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 2 {
					t.Errorf("Expected 2 transformations, got %d", len(trans))
					return
				}
				if trans[0].Type != ApplyTypeGroupBy {
					t.Errorf("Expected first transformation to be groupby, got %v", trans[0].Type)
				}
				if trans[1].Type != ApplyTypeFilter {
					t.Errorf("Expected second transformation to be filter, got %v", trans[1].Type)
				}
				// Verify the filter references the computed alias
				if trans[1].Filter == nil {
					t.Error("Filter is nil")
					return
				}
				if trans[1].Filter.Property != "Total" {
					t.Errorf("Expected filter property to be 'Total', got '%s'", trans[1].Filter.Property)
				}
			},
		},
		{
			name:      "Filter on $count after groupby",
			applyStr:  "filter(Price gt 10)/groupby((Category))/filter($count gt 1)",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 3 {
					t.Errorf("Expected 3 transformations, got %d", len(trans))
					return
				}
				if trans[2].Type != ApplyTypeFilter {
					t.Errorf("Expected third transformation to be filter, got %v", trans[2].Type)
				}
				// Verify the filter references $count
				if trans[2].Filter == nil {
					t.Error("Filter is nil")
					return
				}
				if trans[2].Filter.Property != "$count" {
					t.Errorf("Expected filter property to be '$count', got '%s'", trans[2].Filter.Property)
				}
			},
		},
		{
			name:      "Complex pipeline with multiple filters",
			applyStr:  "filter(Price gt 10)/groupby((Category),aggregate(Price with average as AvgPrice))/filter(AvgPrice gt 50)",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 3 {
					t.Errorf("Expected 3 transformations, got %d", len(trans))
					return
				}
				// First filter should be on Price
				if trans[0].Filter.Property != "Price" {
					t.Errorf("Expected first filter on 'Price', got '%s'", trans[0].Filter.Property)
				}
				// Last filter should be on computed alias AvgPrice
				if trans[2].Filter.Property != "AvgPrice" {
					t.Errorf("Expected last filter on 'AvgPrice', got '%s'", trans[2].Filter.Property)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseApply(tt.applyStr, meta, 0)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestExtractAliasesFromTransformation tests that aliases are properly extracted
// from various transformation types
func TestExtractAliasesFromTransformation(t *testing.T) {
	tests := []struct {
		name               string
		transformation     ApplyTransformation
		expectedAliases    []string
		shouldContainCount bool
	}{
		{
			name: "GroupBy creates $count",
			transformation: ApplyTransformation{
				Type: ApplyTypeGroupBy,
				GroupBy: &GroupByTransformation{
					Properties: []string{"Category"},
				},
			},
			expectedAliases:    []string{},
			shouldContainCount: true,
		},
		{
			name: "Aggregate creates aliases",
			transformation: ApplyTransformation{
				Type: ApplyTypeAggregate,
				Aggregate: &AggregateTransformation{
					Expressions: []AggregateExpression{
						{Property: "Price", Method: AggregationSum, Alias: "TotalPrice"},
						{Property: "Price", Method: AggregationAvg, Alias: "AvgPrice"},
					},
				},
			},
			expectedAliases:    []string{"TotalPrice", "AvgPrice"},
			shouldContainCount: false,
		},
		{
			name: "Compute creates aliases",
			transformation: ApplyTransformation{
				Type: ApplyTypeCompute,
				Compute: &ComputeTransformation{
					Expressions: []ComputeExpression{
						{Alias: "DiscountedPrice"},
						{Alias: "TaxAmount"},
					},
				},
			},
			expectedAliases:    []string{"DiscountedPrice", "TaxAmount"},
			shouldContainCount: false,
		},
		{
			name: "GroupBy with nested aggregate",
			transformation: ApplyTransformation{
				Type: ApplyTypeGroupBy,
				GroupBy: &GroupByTransformation{
					Properties: []string{"Category"},
					Transform: []ApplyTransformation{
						{
							Type: ApplyTypeAggregate,
							Aggregate: &AggregateTransformation{
								Expressions: []AggregateExpression{
									{Property: "Price", Method: AggregationSum, Alias: "Total"},
								},
							},
						},
					},
				},
			},
			expectedAliases:    []string{"Total"},
			shouldContainCount: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliases := make(map[string]bool)
			extractAliasesFromTransformation(&tt.transformation, aliases)

			// Check for expected aliases
			for _, expectedAlias := range tt.expectedAliases {
				if !aliases[expectedAlias] {
					t.Errorf("Expected alias '%s' not found in extracted aliases", expectedAlias)
				}
			}

			// Check for $count
			if tt.shouldContainCount {
				if !aliases["$count"] {
					t.Error("Expected $count to be in aliases for groupby transformation")
				}
			}

			// Verify we didn't get unexpected aliases
			expectedCount := len(tt.expectedAliases)
			if tt.shouldContainCount {
				expectedCount++
			}
			if len(aliases) != expectedCount {
				t.Errorf("Expected %d aliases, got %d: %v", expectedCount, len(aliases), aliases)
			}
		})
	}
}

// TestBuildAggregateSQL_ColumnNames tests that aggregate SQL uses correct column names
func TestBuildAggregateSQL_ColumnNames(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name     string
		expr     AggregateExpression
		expected string
	}{
		{
			name: "CountDistinct uses snake_case column",
			expr: AggregateExpression{
				Property: "Category",
				Method:   AggregationCountDistinct,
				Alias:    "UniqueCategories",
			},
			expected: "COUNT(DISTINCT \"apply_test_entities\".\"category\") as \"UniqueCategories\"",
		},
		{
			name: "Sum uses snake_case column",
			expr: AggregateExpression{
				Property: "Price",
				Method:   AggregationSum,
				Alias:    "TotalPrice",
			},
			expected: "SUM(\"apply_test_entities\".\"price\") as \"TotalPrice\"",
		},
		{
			name: "Average uses snake_case column",
			expr: AggregateExpression{
				Property: "Quantity",
				Method:   AggregationAvg,
				Alias:    "AvgQty",
			},
			expected: "AVG(\"apply_test_entities\".\"quantity\") as \"AvgQty\"",
		},
		{
			name: "Min uses snake_case column",
			expr: AggregateExpression{
				Property: "Price",
				Method:   AggregationMin,
				Alias:    "MinPrice",
			},
			expected: "MIN(\"apply_test_entities\".\"price\") as \"MinPrice\"",
		},
		{
			name: "Max uses snake_case column",
			expr: AggregateExpression{
				Property: "Price",
				Method:   AggregationMax,
				Alias:    "MaxPrice",
			},
			expected: "MAX(\"apply_test_entities\".\"price\") as \"MaxPrice\"",
		},
		{
			name: "$count special case",
			expr: AggregateExpression{
				Property: "$count",
				Method:   AggregationCount,
				Alias:    "Total",
			},
			expected: "COUNT(*) as \"Total\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAggregateSQL("sqlite", tt.expr, meta)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTokenizer_DollarSign tests that the tokenizer properly handles $ in identifiers
func TestTokenizer_DollarSign(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Expected token values
	}{
		{
			name:     "$count property",
			input:    "$count gt 1",
			expected: []string{"$count", "gt", "1"},
		},
		{
			name:     "$count in complex expression",
			input:    "$count gt 5 and Category eq 'Electronics'",
			expected: []string{"$count", "gt", "5", "and", "Category", "eq", "Electronics"},
		},
		{
			name:     "Regular property without $",
			input:    "Total gt 100",
			expected: []string{"Total", "gt", "100"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			// Filter out EOF token
			var values []string
			for _, token := range tokens {
				if token.Type != TokenEOF {
					values = append(values, token.Value)
				}
			}

			// Check we got the expected number of tokens
			if len(values) != len(tt.expected) {
				t.Fatalf("Expected %d tokens, got %d: %v", len(tt.expected), len(values), values)
			}

			// Check each token value
			for i, expectedValue := range tt.expected {
				if values[i] != expectedValue {
					t.Errorf("Token %d: expected '%s', got '%s'", i, expectedValue, values[i])
				}
			}
		})
	}
}
