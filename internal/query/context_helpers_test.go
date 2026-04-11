package query

import (
	"reflect"
	"testing"
)

func TestContextPropertiesFromApply_Nil(t *testing.T) {
	result := ContextPropertiesFromApply(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestContextPropertiesFromApply_Empty(t *testing.T) {
	result := ContextPropertiesFromApply([]ApplyTransformation{})
	if result != nil {
		t.Errorf("expected nil for empty slice, got %v", result)
	}
}

func TestContextPropertiesFromApply_FilterOnly(t *testing.T) {
	// filter transformations do not change the output shape
	transformations := []ApplyTransformation{
		{Type: ApplyTypeFilter},
	}
	result := ContextPropertiesFromApply(transformations)
	if result != nil {
		t.Errorf("expected nil for filter-only pipeline, got %v", result)
	}
}

func TestContextPropertiesFromApply_AggregateWithAliases(t *testing.T) {
	transformations := []ApplyTransformation{
		{
			Type: ApplyTypeAggregate,
			Aggregate: &AggregateTransformation{
				Expressions: []AggregateExpression{
					{Property: "Price", Method: AggregationSum, Alias: "TotalPrice"},
					{Property: "$count", Method: AggregationCount, Alias: "Count"},
				},
			},
		},
	}
	result := ContextPropertiesFromApply(transformations)
	expected := []string{"TotalPrice", "Count"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ContextPropertiesFromApply() = %v, want %v", result, expected)
	}
}

func TestContextPropertiesFromApply_GroupByWithNestedAggregate(t *testing.T) {
	transformations := []ApplyTransformation{
		{
			Type: ApplyTypeGroupBy,
			GroupBy: &GroupByTransformation{
				Properties: []string{"CategoryID", "Status"},
				Transform: []ApplyTransformation{
					{
						Type: ApplyTypeAggregate,
						Aggregate: &AggregateTransformation{
							Expressions: []AggregateExpression{
								{Property: "$count", Method: AggregationCount, Alias: "Count"},
							},
						},
					},
				},
			},
		},
	}
	result := ContextPropertiesFromApply(transformations)
	expected := []string{"CategoryID", "Status", "Count"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ContextPropertiesFromApply() = %v, want %v", result, expected)
	}
}

func TestContextPropertiesFromApply_GroupByNoAggregate(t *testing.T) {
	transformations := []ApplyTransformation{
		{
			Type: ApplyTypeGroupBy,
			GroupBy: &GroupByTransformation{
				Properties: []string{"CategoryID"},
			},
		},
	}
	result := ContextPropertiesFromApply(transformations)
	expected := []string{"CategoryID"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ContextPropertiesFromApply() = %v, want %v", result, expected)
	}
}

func TestContextPropertiesFromApply_DeduplicatesProperties(t *testing.T) {
	// Both aggregate steps producing the same alias should be deduplicated
	transformations := []ApplyTransformation{
		{
			Type: ApplyTypeAggregate,
			Aggregate: &AggregateTransformation{
				Expressions: []AggregateExpression{
					{Property: "Price", Method: AggregationSum, Alias: "Total"},
					{Property: "Price", Method: AggregationAvg, Alias: "Total"}, // duplicate
				},
			},
		},
	}
	result := ContextPropertiesFromApply(transformations)
	expected := []string{"Total"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ContextPropertiesFromApply() = %v, want %v", result, expected)
	}
}

func TestContextPropertiesFromApply_ComputeOnly(t *testing.T) {
	// compute transformations alone do not change the output shape (entities still returned)
	transformations := []ApplyTransformation{
		{
			Type: ApplyTypeCompute,
			Compute: &ComputeTransformation{
				Expressions: []ComputeExpression{
					{Alias: "Computed"},
				},
			},
		},
	}
	result := ContextPropertiesFromApply(transformations)
	if result != nil {
		t.Errorf("expected nil for compute-only pipeline, got %v", result)
	}
}

func TestContextPropertiesFromApply_Join(t *testing.T) {
	transformations := []ApplyTransformation{
		{
			Type: ApplyTypeJoin,
			Join: &JoinTransformation{Alias: "Sale"},
		},
	}

	result := ContextPropertiesFromApply(transformations)
	expected := []string{"Sale()"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ContextPropertiesFromApply() = %v, want %v", result, expected)
	}
}
