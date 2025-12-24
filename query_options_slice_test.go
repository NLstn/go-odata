package odata

import "testing"

type sampleItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type taggedItem struct {
	ProductName string `json:"name"`
	Rank        int    `json:"rank"`
}

func TestApplyQueryOptionsToSlice_AppliesFilterOrderBySkipTop(t *testing.T) {
	items := []sampleItem{
		{ID: 1, Name: "beta", Score: 2},
		{ID: 2, Name: "alpha", Score: 1},
		{ID: 3, Name: "beta", Score: 3},
		{ID: 4, Name: "beta", Score: 0},
	}

	top := 2
	skip := 0
	options := &QueryOptions{
		Filter: &FilterExpression{
			Property: "name",
			Operator: "eq",
			Value:    "beta",
		},
		OrderBy: []OrderByItem{
			{Property: "score", Descending: true},
		},
		Top:  &top,
		Skip: &skip,
	}

	filterFunc := func(item sampleItem, filter *FilterExpression) (bool, error) {
		if filter.Operator != "eq" || filter.Property != "name" {
			return false, nil
		}
		return item.Name == filter.Value, nil
	}

	result, err := ApplyQueryOptionsToSlice(items, options, filterFunc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0].ID != 3 || result[1].ID != 1 {
		t.Fatalf("unexpected ordering: %+v", result)
	}
}

func TestApplyQueryOptionsToSlice_FilterRequiresEvaluator(t *testing.T) {
	options := &QueryOptions{
		Filter: &FilterExpression{
			Property: "name",
			Operator: "eq",
			Value:    "beta",
		},
	}

	_, err := ApplyQueryOptionsToSlice([]sampleItem{{ID: 1, Name: "beta"}}, options, nil)
	if err == nil {
		t.Fatal("expected error when filter evaluator is nil")
	}
}

func TestApplyQueryOptionsToSlice_OrderByUsesJSONTag(t *testing.T) {
	items := []taggedItem{
		{ProductName: "beta", Rank: 2},
		{ProductName: "alpha", Rank: 1},
	}

	options := &QueryOptions{
		OrderBy: []OrderByItem{
			{Property: "name"},
		},
	}

	result, err := ApplyQueryOptionsToSlice(items, options, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result[0].ProductName != "alpha" {
		t.Fatalf("unexpected ordering: %+v", result)
	}
}

func TestApplyQueryOptionsToSlice_OrderByUnknownProperty(t *testing.T) {
	items := []sampleItem{
		{ID: 1, Name: "beta", Score: 2},
		{ID: 2, Name: "alpha", Score: 1},
	}

	options := &QueryOptions{
		OrderBy: []OrderByItem{
			{Property: "missing"},
		},
	}

	_, err := ApplyQueryOptionsToSlice(items, options, nil)
	if err == nil {
		t.Fatal("expected error for unknown order by property")
	}
}
