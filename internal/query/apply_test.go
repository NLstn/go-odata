package query

import (
	"net/url"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestEntity for testing apply transformations
type ApplyTestEntity struct {
	ID       int     `json:"ID" odata:"key"`
	Name     string  `json:"Name"`
	Category string  `json:"Category"`
	Price    float64 `json:"Price"`
	Quantity int     `json:"Quantity"`
}

func getApplyTestMetadata(t *testing.T) *metadata.EntityMetadata {
	meta, err := metadata.AnalyzeEntity(ApplyTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	return meta
}

func TestParseApply_GroupBy_Simple(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		applyStr  string
		expectErr bool
		validate  func(*testing.T, []ApplyTransformation)
	}{
		{
			name:      "GroupBy single property",
			applyStr:  "groupby((Category))",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(trans))
					return
				}
				if trans[0].Type != ApplyTypeGroupBy {
					t.Errorf("Expected groupby transformation, got %v", trans[0].Type)
				}
				if trans[0].GroupBy == nil {
					t.Error("GroupBy is nil")
					return
				}
				if len(trans[0].GroupBy.Properties) != 1 {
					t.Errorf("Expected 1 property, got %d", len(trans[0].GroupBy.Properties))
					return
				}
				if trans[0].GroupBy.Properties[0] != "Category" {
					t.Errorf("Expected 'Category', got '%s'", trans[0].GroupBy.Properties[0])
				}
			},
		},
		{
			name:      "GroupBy multiple properties",
			applyStr:  "groupby((Category,Name))",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(trans))
					return
				}
				if trans[0].GroupBy == nil {
					t.Error("GroupBy is nil")
					return
				}
				if len(trans[0].GroupBy.Properties) != 2 {
					t.Errorf("Expected 2 properties, got %d", len(trans[0].GroupBy.Properties))
					return
				}
				if trans[0].GroupBy.Properties[0] != "Category" {
					t.Errorf("Expected 'Category', got '%s'", trans[0].GroupBy.Properties[0])
				}
				if trans[0].GroupBy.Properties[1] != "Name" {
					t.Errorf("Expected 'Name', got '%s'", trans[0].GroupBy.Properties[1])
				}
			},
		},
		{
			name:      "Invalid property in groupby",
			applyStr:  "groupby((InvalidProperty))",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseApply(tt.applyStr, meta)
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

func TestParseApply_Aggregate(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		applyStr  string
		expectErr bool
		validate  func(*testing.T, []ApplyTransformation)
	}{
		{
			name:      "Aggregate with sum",
			applyStr:  "aggregate(Price with sum as Total)",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(trans))
					return
				}
				if trans[0].Type != ApplyTypeAggregate {
					t.Errorf("Expected aggregate transformation, got %v", trans[0].Type)
				}
				if trans[0].Aggregate == nil {
					t.Error("Aggregate is nil")
					return
				}
				if len(trans[0].Aggregate.Expressions) != 1 {
					t.Errorf("Expected 1 expression, got %d", len(trans[0].Aggregate.Expressions))
					return
				}
				expr := trans[0].Aggregate.Expressions[0]
				if expr.Property != "Price" {
					t.Errorf("Expected 'Price', got '%s'", expr.Property)
				}
				if expr.Method != AggregationSum {
					t.Errorf("Expected 'sum', got '%s'", expr.Method)
				}
				if expr.Alias != "Total" {
					t.Errorf("Expected 'Total', got '%s'", expr.Alias)
				}
			},
		},
		{
			name:      "Aggregate with average",
			applyStr:  "aggregate(Price with average as AvgPrice)",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(trans))
					return
				}
				expr := trans[0].Aggregate.Expressions[0]
				if expr.Method != AggregationAvg {
					t.Errorf("Expected 'average', got '%s'", expr.Method)
				}
			},
		},
		{
			name:      "Aggregate with count",
			applyStr:  "aggregate($count as Total)",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(trans))
					return
				}
				expr := trans[0].Aggregate.Expressions[0]
				if expr.Property != "$count" {
					t.Errorf("Expected '$count', got '%s'", expr.Property)
				}
				if expr.Method != AggregationCount {
					t.Errorf("Expected 'count', got '%s'", expr.Method)
				}
				if expr.Alias != "Total" {
					t.Errorf("Expected 'Total', got '%s'", expr.Alias)
				}
			},
		},
		{
			name:      "Aggregate with multiple expressions",
			applyStr:  "aggregate(Price with sum as TotalPrice, Quantity with sum as TotalQuantity)",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(trans))
					return
				}
				if len(trans[0].Aggregate.Expressions) != 2 {
					t.Errorf("Expected 2 expressions, got %d", len(trans[0].Aggregate.Expressions))
					return
				}
				expr1 := trans[0].Aggregate.Expressions[0]
				if expr1.Property != "Price" || expr1.Alias != "TotalPrice" {
					t.Errorf("First expression mismatch: %s as %s", expr1.Property, expr1.Alias)
				}
				expr2 := trans[0].Aggregate.Expressions[1]
				if expr2.Property != "Quantity" || expr2.Alias != "TotalQuantity" {
					t.Errorf("Second expression mismatch: %s as %s", expr2.Property, expr2.Alias)
				}
			},
		},
		{
			name:      "Aggregate with min",
			applyStr:  "aggregate(Price with min as MinPrice)",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				expr := trans[0].Aggregate.Expressions[0]
				if expr.Method != AggregationMin {
					t.Errorf("Expected 'min', got '%s'", expr.Method)
				}
			},
		},
		{
			name:      "Aggregate with max",
			applyStr:  "aggregate(Price with max as MaxPrice)",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				expr := trans[0].Aggregate.Expressions[0]
				if expr.Method != AggregationMax {
					t.Errorf("Expected 'max', got '%s'", expr.Method)
				}
			},
		},
		{
			name:      "Invalid property in aggregate",
			applyStr:  "aggregate(InvalidProperty with sum as Total)",
			expectErr: true,
		},
		{
			name:      "Invalid method in aggregate",
			applyStr:  "aggregate(Price with invalid as Total)",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseApply(tt.applyStr, meta)
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

func TestParseApply_GroupByWithAggregate(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		applyStr  string
		expectErr bool
		validate  func(*testing.T, []ApplyTransformation)
	}{
		{
			name:      "GroupBy with aggregate",
			applyStr:  "groupby((Category), aggregate(Price with sum as Total))",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(trans))
					return
				}
				if trans[0].Type != ApplyTypeGroupBy {
					t.Errorf("Expected groupby transformation, got %v", trans[0].Type)
				}
				if trans[0].GroupBy == nil {
					t.Error("GroupBy is nil")
					return
				}
				if len(trans[0].GroupBy.Properties) != 1 {
					t.Errorf("Expected 1 property, got %d", len(trans[0].GroupBy.Properties))
					return
				}
				if trans[0].GroupBy.Properties[0] != "Category" {
					t.Errorf("Expected 'Category', got '%s'", trans[0].GroupBy.Properties[0])
				}
				// Check nested aggregate
				if len(trans[0].GroupBy.Transform) != 1 {
					t.Errorf("Expected 1 nested transformation, got %d", len(trans[0].GroupBy.Transform))
					return
				}
				nestedTrans := trans[0].GroupBy.Transform[0]
				if nestedTrans.Type != ApplyTypeAggregate {
					t.Errorf("Expected aggregate transformation, got %v", nestedTrans.Type)
				}
				if nestedTrans.Aggregate == nil {
					t.Error("Aggregate is nil")
					return
				}
				if len(nestedTrans.Aggregate.Expressions) != 1 {
					t.Errorf("Expected 1 expression, got %d", len(nestedTrans.Aggregate.Expressions))
					return
				}
				expr := nestedTrans.Aggregate.Expressions[0]
				if expr.Property != "Price" {
					t.Errorf("Expected 'Price', got '%s'", expr.Property)
				}
				if expr.Method != AggregationSum {
					t.Errorf("Expected 'sum', got '%s'", expr.Method)
				}
				if expr.Alias != "Total" {
					t.Errorf("Expected 'Total', got '%s'", expr.Alias)
				}
			},
		},
		{
			name:      "GroupBy with multiple aggregates",
			applyStr:  "groupby((Category), aggregate(Price with sum as TotalPrice, Price with average as AvgPrice))",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(trans))
					return
				}
				nestedTrans := trans[0].GroupBy.Transform[0]
				if len(nestedTrans.Aggregate.Expressions) != 2 {
					t.Errorf("Expected 2 expressions, got %d", len(nestedTrans.Aggregate.Expressions))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseApply(tt.applyStr, meta)
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

func TestParseApply_MultipleTransformations(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		applyStr  string
		expectErr bool
		validate  func(*testing.T, []ApplyTransformation)
	}{
		{
			name:      "Filter then GroupBy",
			applyStr:  "filter(Price gt 10)/groupby((Category))",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 2 {
					t.Errorf("Expected 2 transformations, got %d", len(trans))
					return
				}
				if trans[0].Type != ApplyTypeFilter {
					t.Errorf("Expected filter transformation, got %v", trans[0].Type)
				}
				if trans[1].Type != ApplyTypeGroupBy {
					t.Errorf("Expected groupby transformation, got %v", trans[1].Type)
				}
			},
		},
		{
			name:      "Filter then GroupBy then Aggregate",
			applyStr:  "filter(Price gt 10)/groupby((Category), aggregate(Price with sum as Total))",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 2 {
					t.Errorf("Expected 2 transformations, got %d", len(trans))
					return
				}
				if trans[0].Type != ApplyTypeFilter {
					t.Errorf("Expected filter transformation, got %v", trans[0].Type)
				}
				if trans[1].Type != ApplyTypeGroupBy {
					t.Errorf("Expected groupby transformation, got %v", trans[1].Type)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseApply(tt.applyStr, meta)
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

func TestParseApply_Filter(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		applyStr  string
		expectErr bool
		validate  func(*testing.T, []ApplyTransformation)
	}{
		{
			name:      "Simple filter",
			applyStr:  "filter(Price gt 10)",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(trans))
					return
				}
				if trans[0].Type != ApplyTypeFilter {
					t.Errorf("Expected filter transformation, got %v", trans[0].Type)
				}
				if trans[0].Filter == nil {
					t.Error("Filter is nil")
				}
			},
		},
		{
			name:      "Complex filter",
			applyStr:  "filter(Price gt 10 and Category eq 'Electronics')",
			expectErr: false,
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(trans))
					return
				}
				if trans[0].Filter == nil {
					t.Error("Filter is nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseApply(tt.applyStr, meta)
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

func TestParseQueryOptions_Apply(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		query     string
		expectErr bool
		validate  func(*testing.T, *QueryOptions)
	}{
		{
			name:      "Apply with groupby",
			query:     "$apply=groupby((Category))",
			expectErr: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if len(opts.Apply) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(opts.Apply))
				}
			},
		},
		{
			name:      "Apply with aggregate",
			query:     "$apply=aggregate(Price with sum as Total)",
			expectErr: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if len(opts.Apply) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(opts.Apply))
				}
			},
		},
		{
			name:      "Apply with filter",
			query:     "$apply=filter(Price gt 10)",
			expectErr: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if len(opts.Apply) != 1 {
					t.Errorf("Expected 1 transformation, got %d", len(opts.Apply))
				}
			},
		},
		{
			name:      "Apply with multiple transformations",
			query:     "$apply=filter(Price gt 10)/groupby((Category))",
			expectErr: false,
			validate: func(t *testing.T, opts *QueryOptions) {
				if len(opts.Apply) != 2 {
					t.Errorf("Expected 2 transformations, got %d", len(opts.Apply))
				}
			},
		},
		{
			name:      "Invalid apply",
			query:     "$apply=invalid()",
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

func TestSplitApplyTransformations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single transformation",
			input:    "groupby((Category))",
			expected: []string{"groupby((Category))"},
		},
		{
			name:     "Two transformations",
			input:    "filter(Price gt 10)/groupby((Category))",
			expected: []string{"filter(Price gt 10)", "groupby((Category))"},
		},
		{
			name:     "Complex with nested parentheses",
			input:    "filter(Price gt 10)/groupby((Category), aggregate(Price with sum as Total))",
			expected: []string{"filter(Price gt 10)", "groupby((Category), aggregate(Price with sum as Total))"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitApplyTransformations(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parts, got %d", len(tt.expected), len(result))
				return
			}
			for i, part := range result {
				if part != tt.expected[i] {
					t.Errorf("Part %d: expected '%s', got '%s'", i, tt.expected[i], part)
				}
			}
		})
	}
}

func TestParseAggregationMethod(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		expected  AggregationMethod
		expectErr bool
	}{
		{name: "sum", method: "sum", expected: AggregationSum, expectErr: false},
		{name: "average", method: "average", expected: AggregationAvg, expectErr: false},
		{name: "avg", method: "avg", expected: AggregationAvg, expectErr: false},
		{name: "min", method: "min", expected: AggregationMin, expectErr: false},
		{name: "max", method: "max", expected: AggregationMax, expectErr: false},
		{name: "count", method: "count", expected: AggregationCount, expectErr: false},
		{name: "countdistinct", method: "countdistinct", expected: AggregationCountDistinct, expectErr: false},
		{name: "invalid", method: "invalid", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAggregationMethod(tt.method)
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
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
