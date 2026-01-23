package query

import (
	"net/url"
	"testing"
)

// BenchmarkFilterExpressionPool_Simple benchmarks the pool for simple filter parsing
func BenchmarkFilterExpressionPool_Simple(b *testing.B) {
	entityMeta := getTestEntityMetadata()
	params := url.Values{
		"$filter": []string{"Price gt 100"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseQueryOptions(params, entityMeta)
	}
}

// BenchmarkFilterExpressionPool_Complex benchmarks the pool for complex filter parsing
func BenchmarkFilterExpressionPool_Complex(b *testing.B) {
	entityMeta := getTestEntityMetadata()
	params := url.Values{
		"$filter": []string{"contains(Name, 'test') and Price gt 100 and Rating ge 4"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseQueryOptions(params, entityMeta)
	}
}

// BenchmarkFilterExpressionPool_ManyConditions benchmarks the pool with many filter conditions
func BenchmarkFilterExpressionPool_ManyConditions(b *testing.B) {
	entityMeta := getTestEntityMetadata()
	params := url.Values{
		"$filter": []string{"Price gt 100 and Price lt 500 and Rating ge 3 and Rating le 5 and Name ne 'test' and Category eq 'Electronics' and InStock eq true"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseQueryOptions(params, entityMeta)
	}
}

// BenchmarkFilterExpressionPool_Navigation benchmarks the pool with navigation paths
func BenchmarkFilterExpressionPool_Navigation(b *testing.B) {
	entityMeta := getTestEntityMetadataWithNavigationProperties()
	params := url.Values{
		"$filter": []string{"Category/Name eq 'Electronics' and Supplier/Country eq 'USA' and Price gt 100"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseQueryOptions(params, entityMeta)
	}
}

// TestFilterExpressionPoolReset tests that reset properly clears all fields
func TestFilterExpressionPoolReset(t *testing.T) {
	// Create and populate a FilterExpression
	expr := acquireFilterExpression()
	expr.Property = "TestProperty"
	expr.Operator = OpEqual
	expr.Value = "test"
	expr.Logical = LogicalAnd
	expr.IsNot = true
	expr.maxInClauseSize = 100

	// Create child expressions
	left := acquireFilterExpression()
	left.Property = "LeftProp"
	right := acquireFilterExpression()
	right.Property = "RightProp"
	expr.Left = left
	expr.Right = right

	// Reset the expression
	expr.reset()

	// Verify all fields are zeroed
	if expr.Property != "" {
		t.Errorf("Property not reset, got %q", expr.Property)
	}
	if expr.Operator != "" {
		t.Errorf("Operator not reset, got %q", expr.Operator)
	}
	if expr.Value != nil {
		t.Errorf("Value not reset, got %v", expr.Value)
	}
	if expr.Logical != "" {
		t.Errorf("Logical not reset, got %q", expr.Logical)
	}
	if expr.IsNot != false {
		t.Error("IsNot not reset")
	}
	if expr.maxInClauseSize != 0 {
		t.Errorf("maxInClauseSize not reset, got %d", expr.maxInClauseSize)
	}
	if expr.Left != nil {
		t.Error("Left not reset")
	}
	if expr.Right != nil {
		t.Error("Right not reset")
	}
}

// TestReleaseFilterTree tests recursive tree release
func TestReleaseFilterTree(t *testing.T) {
	// Create a filter tree
	left := acquireFilterExpression()
	left.Property = "A"
	left.Operator = OpEqual
	left.Value = 1

	right := acquireFilterExpression()
	right.Property = "B"
	right.Operator = OpEqual
	right.Value = 2

	root := acquireFilterExpression()
	root.Left = left
	root.Right = right
	root.Logical = LogicalAnd

	// Release the tree (should not panic)
	ReleaseFilterTree(root)

	// Test with nil (should not panic)
	ReleaseFilterTree(nil)
}
