package odata_test

import (
	"testing"

	odata "github.com/nlstn/go-odata"
)

// TestExportedApplyTypes tests that Apply transformation types are properly exported
func TestExportedApplyTypes(t *testing.T) {
	// Test that we can create and use ApplyTransformation
	var transformation odata.ApplyTransformation
	transformation.Type = odata.ApplyTypeGroupBy

	// Test that we can use GroupByTransformation
	var groupBy odata.GroupByTransformation
	groupBy.Properties = []string{"Category"}

	// Test that we can use AggregateTransformation
	var aggregate odata.AggregateTransformation
	aggregate.Expressions = []odata.AggregateExpression{
		{
			Property: "Price",
			Method:   odata.AggregationSum,
			Alias:    "TotalPrice",
		},
	}

	// Test that we can use ComputeTransformation
	var compute odata.ComputeTransformation
	compute.Expressions = []odata.ComputeExpression{
		{
			Alias: "FullName",
		},
	}

	// Test ParserConfig
	var config odata.ParserConfig
	config.MaxInClauseSize = 100
	config.MaxExpandDepth = 5

	// Verify constants are accessible
	if odata.ApplyTypeGroupBy != "groupby" {
		t.Errorf("Expected ApplyTypeGroupBy to be 'groupby', got %v", odata.ApplyTypeGroupBy)
	}
	if odata.ApplyTypeAggregate != "aggregate" {
		t.Errorf("Expected ApplyTypeAggregate to be 'aggregate', got %v", odata.ApplyTypeAggregate)
	}
	if odata.ApplyTypeFilter != "filter" {
		t.Errorf("Expected ApplyTypeFilter to be 'filter', got %v", odata.ApplyTypeFilter)
	}
	if odata.ApplyTypeCompute != "compute" {
		t.Errorf("Expected ApplyTypeCompute to be 'compute', got %v", odata.ApplyTypeCompute)
	}

	// Verify aggregation method constants
	if odata.AggregationSum != "sum" {
		t.Errorf("Expected AggregationSum to be 'sum', got %v", odata.AggregationSum)
	}
	if odata.AggregationAvg != "average" {
		t.Errorf("Expected AggregationAvg to be 'average', got %v", odata.AggregationAvg)
	}
	if odata.AggregationMin != "min" {
		t.Errorf("Expected AggregationMin to be 'min', got %v", odata.AggregationMin)
	}
	if odata.AggregationMax != "max" {
		t.Errorf("Expected AggregationMax to be 'max', got %v", odata.AggregationMax)
	}
	if odata.AggregationCount != "count" {
		t.Errorf("Expected AggregationCount to be 'count', got %v", odata.AggregationCount)
	}
	if odata.AggregationCountDistinct != "countdistinct" {
		t.Errorf("Expected AggregationCountDistinct to be 'countdistinct', got %v", odata.AggregationCountDistinct)
	}
}

// TestExportedFilterOperators tests that filter operator constants are properly exported
func TestExportedFilterOperators(t *testing.T) {
	// Test comparison operators
	if odata.OpEqual != "eq" {
		t.Errorf("Expected OpEqual to be 'eq', got %v", odata.OpEqual)
	}
	if odata.OpNotEqual != "ne" {
		t.Errorf("Expected OpNotEqual to be 'ne', got %v", odata.OpNotEqual)
	}
	if odata.OpGreaterThan != "gt" {
		t.Errorf("Expected OpGreaterThan to be 'gt', got %v", odata.OpGreaterThan)
	}
	if odata.OpGreaterThanOrEqual != "ge" {
		t.Errorf("Expected OpGreaterThanOrEqual to be 'ge', got %v", odata.OpGreaterThanOrEqual)
	}
	if odata.OpLessThan != "lt" {
		t.Errorf("Expected OpLessThan to be 'lt', got %v", odata.OpLessThan)
	}
	if odata.OpLessThanOrEqual != "le" {
		t.Errorf("Expected OpLessThanOrEqual to be 'le', got %v", odata.OpLessThanOrEqual)
	}
	if odata.OpIn != "in" {
		t.Errorf("Expected OpIn to be 'in', got %v", odata.OpIn)
	}

	// Test string function operators
	if odata.OpContains != "contains" {
		t.Errorf("Expected OpContains to be 'contains', got %v", odata.OpContains)
	}
	if odata.OpStartsWith != "startswith" {
		t.Errorf("Expected OpStartsWith to be 'startswith', got %v", odata.OpStartsWith)
	}
	if odata.OpEndsWith != "endswith" {
		t.Errorf("Expected OpEndsWith to be 'endswith', got %v", odata.OpEndsWith)
	}

	// Test arithmetic operators
	if odata.OpAdd != "add" {
		t.Errorf("Expected OpAdd to be 'add', got %v", odata.OpAdd)
	}
	if odata.OpSub != "sub" {
		t.Errorf("Expected OpSub to be 'sub', got %v", odata.OpSub)
	}
	if odata.OpMul != "mul" {
		t.Errorf("Expected OpMul to be 'mul', got %v", odata.OpMul)
	}
	if odata.OpDiv != "div" {
		t.Errorf("Expected OpDiv to be 'div', got %v", odata.OpDiv)
	}
	if odata.OpMod != "mod" {
		t.Errorf("Expected OpMod to be 'mod', got %v", odata.OpMod)
	}

	// Test date function operators
	if odata.OpYear != "year" {
		t.Errorf("Expected OpYear to be 'year', got %v", odata.OpYear)
	}
	if odata.OpMonth != "month" {
		t.Errorf("Expected OpMonth to be 'month', got %v", odata.OpMonth)
	}
	if odata.OpDay != "day" {
		t.Errorf("Expected OpDay to be 'day', got %v", odata.OpDay)
	}

	// Test lambda operators
	if odata.OpAny != "any" {
		t.Errorf("Expected OpAny to be 'any', got %v", odata.OpAny)
	}
	if odata.OpAll != "all" {
		t.Errorf("Expected OpAll to be 'all', got %v", odata.OpAll)
	}
}

// TestExportedLogicalOperators tests that logical operator constants are properly exported
func TestExportedLogicalOperators(t *testing.T) {
	if odata.LogicalAnd != "and" {
		t.Errorf("Expected LogicalAnd to be 'and', got %v", odata.LogicalAnd)
	}
	if odata.LogicalOr != "or" {
		t.Errorf("Expected LogicalOr to be 'or', got %v", odata.LogicalOr)
	}
}

// TestParseFilter tests the exported ParseFilter function
func TestParseFilter(t *testing.T) {
	tests := []struct {
		name        string
		filterStr   string
		expectError bool
	}{
		{
			name:        "Simple equality filter",
			filterStr:   "Name eq 'John'",
			expectError: false,
		},
		{
			name:        "Simple comparison filter",
			filterStr:   "Age gt 18",
			expectError: false,
		},
		{
			name:        "Logical AND filter",
			filterStr:   "Name eq 'John' and Age gt 18",
			expectError: false,
		},
		{
			name:        "String function filter",
			filterStr:   "contains(Name, 'test')",
			expectError: false,
		},
		{
			name:        "Invalid filter syntax",
			filterStr:   "Name eq",
			expectError: true,
		},
		{
			name:        "Empty filter",
			filterStr:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := odata.ParseFilter(tt.filterStr)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for filter '%s', but got none", tt.filterStr)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for filter '%s': %v", tt.filterStr, err)
				}
				if filter == nil {
					t.Errorf("Expected non-nil filter for '%s', but got nil", tt.filterStr)
				}
			}
		})
	}
}

// TestParseFilterUsage demonstrates how to use ParseFilter with operator constants
func TestParseFilterUsage(t *testing.T) {
	// Parse a filter
	filter, err := odata.ParseFilter("Name eq 'John'")
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	// Verify the parsed filter structure
	if filter.Property != "Name" {
		t.Errorf("Expected property 'Name', got '%s'", filter.Property)
	}

	if filter.Operator != odata.OpEqual {
		t.Errorf("Expected operator 'eq', got '%s'", filter.Operator)
	}

	if filter.Value != "'John'" && filter.Value != "John" {
		t.Errorf("Expected value 'John' or ''John'', got '%v'", filter.Value)
	}
}

// TestQueryOptionsWithApply demonstrates using QueryOptions with Apply transformations
func TestQueryOptionsWithApply(t *testing.T) {
	// Create a QueryOptions with Apply transformations
	options := &odata.QueryOptions{
		Apply: []odata.ApplyTransformation{
			{
				Type: odata.ApplyTypeGroupBy,
				GroupBy: &odata.GroupByTransformation{
					Properties: []string{"Category"},
					Transform: []odata.ApplyTransformation{
						{
							Type: odata.ApplyTypeAggregate,
							Aggregate: &odata.AggregateTransformation{
								Expressions: []odata.AggregateExpression{
									{
										Property: "Price",
										Method:   odata.AggregationSum,
										Alias:    "TotalPrice",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Verify the structure
	if len(options.Apply) != 1 {
		t.Fatalf("Expected 1 apply transformation, got %d", len(options.Apply))
	}

	if options.Apply[0].Type != odata.ApplyTypeGroupBy {
		t.Errorf("Expected groupby transformation, got %v", options.Apply[0].Type)
	}

	if options.Apply[0].GroupBy == nil {
		t.Fatal("Expected GroupBy to be non-nil")
	}

	if len(options.Apply[0].GroupBy.Properties) != 1 {
		t.Fatalf("Expected 1 groupby property, got %d", len(options.Apply[0].GroupBy.Properties))
	}

	if options.Apply[0].GroupBy.Properties[0] != "Category" {
		t.Errorf("Expected groupby property 'Category', got '%s'", options.Apply[0].GroupBy.Properties[0])
	}

	if len(options.Apply[0].GroupBy.Transform) != 1 {
		t.Fatalf("Expected 1 nested transformation, got %d", len(options.Apply[0].GroupBy.Transform))
	}

	if options.Apply[0].GroupBy.Transform[0].Type != odata.ApplyTypeAggregate {
		t.Errorf("Expected aggregate transformation, got %v", options.Apply[0].GroupBy.Transform[0].Type)
	}

	if options.Apply[0].GroupBy.Transform[0].Aggregate == nil {
		t.Fatal("Expected Aggregate to be non-nil")
	}

	if len(options.Apply[0].GroupBy.Transform[0].Aggregate.Expressions) != 1 {
		t.Fatalf("Expected 1 aggregate expression, got %d", len(options.Apply[0].GroupBy.Transform[0].Aggregate.Expressions))
	}

	expr := options.Apply[0].GroupBy.Transform[0].Aggregate.Expressions[0]
	if expr.Property != "Price" {
		t.Errorf("Expected aggregate property 'Price', got '%s'", expr.Property)
	}
	if expr.Method != odata.AggregationSum {
		t.Errorf("Expected aggregation method 'sum', got '%s'", expr.Method)
	}
	if expr.Alias != "TotalPrice" {
		t.Errorf("Expected alias 'TotalPrice', got '%s'", expr.Alias)
	}
}
