package odata

import (
	"fmt"
	"math"
	"testing"
	"time"
)

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

func TestApplyQueryOptionsToSlice_NilOptions(t *testing.T) {
	items := []sampleItem{
		{ID: 1, Name: "alpha", Score: 1},
		{ID: 2, Name: "beta", Score: 2},
	}

	result, err := ApplyQueryOptionsToSlice(items, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	// Verify it returns a new slice
	if &items[0] == &result[0] {
		t.Fatal("expected a new slice, got same reference")
	}
}

func TestApplyQueryOptionsToSlice_EmptySlice(t *testing.T) {
	items := []sampleItem{}

	options := &QueryOptions{
		OrderBy: []OrderByItem{
			{Property: "name"},
		},
	}

	result, err := ApplyQueryOptionsToSlice(items, options, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 results, got %d", len(result))
	}
}

func TestApplyQueryOptionsToSlice_SkipExceedsLength(t *testing.T) {
	items := []sampleItem{
		{ID: 1, Name: "alpha", Score: 1},
		{ID: 2, Name: "beta", Score: 2},
	}

	skip := 10
	options := &QueryOptions{
		Skip: &skip,
	}

	result, err := ApplyQueryOptionsToSlice(items, options, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 results, got %d", len(result))
	}
}

func TestApplyQueryOptionsToSlice_TopZero(t *testing.T) {
	items := []sampleItem{
		{ID: 1, Name: "alpha", Score: 1},
		{ID: 2, Name: "beta", Score: 2},
	}

	top := 0
	options := &QueryOptions{
		Top: &top,
	}

	result, err := ApplyQueryOptionsToSlice(items, options, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected 0 results, got %d", len(result))
	}
}

func TestApplyQueryOptionsToSlice_NegativeSkip(t *testing.T) {
	items := []sampleItem{
		{ID: 1, Name: "alpha", Score: 1},
	}

	skip := -1
	options := &QueryOptions{
		Skip: &skip,
	}

	_, err := ApplyQueryOptionsToSlice(items, options, nil)
	if err == nil {
		t.Fatal("expected error for negative skip")
	}
}

func TestApplyQueryOptionsToSlice_NegativeTop(t *testing.T) {
	items := []sampleItem{
		{ID: 1, Name: "alpha", Score: 1},
	}

	top := -1
	options := &QueryOptions{
		Top: &top,
	}

	_, err := ApplyQueryOptionsToSlice(items, options, nil)
	if err == nil {
		t.Fatal("expected error for negative top")
	}
}

func TestApplyQueryOptionsToSlice_MultipleOrderBy(t *testing.T) {
	items := []sampleItem{
		{ID: 1, Name: "alpha", Score: 2},
		{ID: 2, Name: "beta", Score: 1},
		{ID: 3, Name: "alpha", Score: 1},
		{ID: 4, Name: "beta", Score: 2},
	}

	options := &QueryOptions{
		OrderBy: []OrderByItem{
			{Property: "name"},
			{Property: "score", Descending: true},
		},
	}

	result, err := ApplyQueryOptionsToSlice(items, options, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result))
	}

	// Expected order: alpha/2, alpha/1, beta/2, beta/1
	if result[0].ID != 1 || result[1].ID != 3 || result[2].ID != 4 || result[3].ID != 2 {
		t.Fatalf("unexpected ordering: %+v", result)
	}
}

func TestApplyQueryOptionsToSlice_FilterReturnsError(t *testing.T) {
	items := []sampleItem{
		{ID: 1, Name: "alpha", Score: 1},
	}

	options := &QueryOptions{
		Filter: &FilterExpression{
			Property: "name",
			Operator: "eq",
			Value:    "alpha",
		},
	}

	filterFunc := func(item sampleItem, filter *FilterExpression) (bool, error) {
		return false, fmt.Errorf("filter error")
	}

	_, err := ApplyQueryOptionsToSlice(items, options, filterFunc)
	if err == nil {
		t.Fatal("expected error from filter function")
	}
}

type diverseTypes struct {
	BoolVal   bool      `json:"bool"`
	FloatVal  float64   `json:"float"`
	TimeVal   time.Time `json:"time"`
	UintVal   uint      `json:"uint"`
	StringVal string    `json:"string"`
}

func TestApplyQueryOptionsToSlice_DifferentDataTypes(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	tests := []struct {
		name     string
		property string
		items    []diverseTypes
		expected []diverseTypes
	}{
		{
			name:     "bool ordering",
			property: "bool",
			items: []diverseTypes{
				{BoolVal: true},
				{BoolVal: false},
			},
			expected: []diverseTypes{
				{BoolVal: false},
				{BoolVal: true},
			},
		},
		{
			name:     "float ordering",
			property: "float",
			items: []diverseTypes{
				{FloatVal: 3.14},
				{FloatVal: 1.41},
			},
			expected: []diverseTypes{
				{FloatVal: 1.41},
				{FloatVal: 3.14},
			},
		},
		{
			name:     "time ordering",
			property: "time",
			items: []diverseTypes{
				{TimeVal: later},
				{TimeVal: now},
			},
			expected: []diverseTypes{
				{TimeVal: now},
				{TimeVal: later},
			},
		},
		{
			name:     "uint ordering",
			property: "uint",
			items: []diverseTypes{
				{UintVal: 100},
				{UintVal: 50},
			},
			expected: []diverseTypes{
				{UintVal: 50},
				{UintVal: 100},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &QueryOptions{
				OrderBy: []OrderByItem{
					{Property: tt.property},
				},
			}

			result, err := ApplyQueryOptionsToSlice(tt.items, options, nil)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(result))
			}

			// Compare based on property type
			for i := range result {
				switch tt.property {
				case "bool":
					if result[i].BoolVal != tt.expected[i].BoolVal {
						t.Fatalf("unexpected ordering at index %d: got %v, want %v", i, result[i].BoolVal, tt.expected[i].BoolVal)
					}
				case "float":
					if result[i].FloatVal != tt.expected[i].FloatVal {
						t.Fatalf("unexpected ordering at index %d: got %v, want %v", i, result[i].FloatVal, tt.expected[i].FloatVal)
					}
				case "time":
					if !result[i].TimeVal.Equal(tt.expected[i].TimeVal) {
						t.Fatalf("unexpected ordering at index %d: got %v, want %v", i, result[i].TimeVal, tt.expected[i].TimeVal)
					}
				case "uint":
					if result[i].UintVal != tt.expected[i].UintVal {
						t.Fatalf("unexpected ordering at index %d: got %v, want %v", i, result[i].UintVal, tt.expected[i].UintVal)
					}
				}
			}
		})
	}
}

func TestApplyQueryOptionsToSlice_NaNHandling(t *testing.T) {
	items := []diverseTypes{
		{FloatVal: 1.0},
		{FloatVal: math.NaN()},
		{FloatVal: 2.0},
		{FloatVal: math.NaN()},
	}

	options := &QueryOptions{
		OrderBy: []OrderByItem{
			{Property: "float"},
		},
	}

	result, err := ApplyQueryOptionsToSlice(items, options, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result))
	}

	// NaN values should be sorted first (treated as less than all values)
	if !math.IsNaN(result[0].FloatVal) || !math.IsNaN(result[1].FloatVal) {
		t.Fatalf("expected NaN values first, got: %+v", result)
	}

	if result[2].FloatVal != 1.0 || result[3].FloatVal != 2.0 {
		t.Fatalf("unexpected ordering of non-NaN values: %+v", result)
	}
}

type nilableFields struct {
	ID      int     `json:"id"`
	StrPtr  *string `json:"str_ptr"`
	IntPtr  *int    `json:"int_ptr"`
	TimePtr *time.Time
}

func TestApplyQueryOptionsToSlice_NilValueHandling(t *testing.T) {
	str1 := "alpha"
	str2 := "beta"
	int1 := 1
	int2 := 2

	items := []nilableFields{
		{ID: 1, StrPtr: &str2, IntPtr: &int2},
		{ID: 2, StrPtr: nil, IntPtr: nil},
		{ID: 3, StrPtr: &str1, IntPtr: &int1},
		{ID: 4, StrPtr: nil, IntPtr: &int2},
	}

	options := &QueryOptions{
		OrderBy: []OrderByItem{
			{Property: "str_ptr"},
		},
	}

	result, err := ApplyQueryOptionsToSlice(items, options, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result))
	}

	// Nil values should be sorted first
	if result[0].StrPtr != nil || result[1].StrPtr != nil {
		t.Fatalf("expected nil values first, got: %+v", result)
	}

	// Non-nil values should be sorted alphabetically
	if result[2].ID != 3 || result[3].ID != 1 {
		t.Fatalf("unexpected ordering: %+v", result)
	}
}
