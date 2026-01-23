package query

import (
	"sync"
)

// filterExpressionPool is a sync.Pool for FilterExpression structs.
// Using a pool reduces GC pressure by reusing FilterExpression objects
// instead of allocating new ones for every filter parsing operation.
var filterExpressionPool = sync.Pool{
	New: func() interface{} {
		return &FilterExpression{}
	},
}

// acquireFilterExpression gets a FilterExpression from the pool.
// The returned expression may contain previous values - callers should
// set all required fields. Fields are reset when expressions are returned
// to the pool via releaseFilterExpression.
func acquireFilterExpression() *FilterExpression {
	//nolint:errcheck // Type assertion is guaranteed by pool's New function
	return filterExpressionPool.Get().(*FilterExpression)
}

// releaseFilterExpression returns a FilterExpression to the pool.
// The expression is reset before being returned.
// NOTE: Only call this when you are certain the FilterExpression
// is no longer referenced anywhere.
func releaseFilterExpression(expr *FilterExpression) {
	if expr == nil {
		return
	}
	expr.reset()
	filterExpressionPool.Put(expr)
}

// reset clears all fields of a FilterExpression to their zero values.
// This is called before returning an expression to the pool.
func (f *FilterExpression) reset() {
	f.Property = ""
	f.Operator = ""
	f.Value = nil
	f.Left = nil
	f.Right = nil
	f.Logical = ""
	f.IsNot = false
	f.maxInClauseSize = 0
}

// ReleaseFilterTree recursively releases a FilterExpression and all its children to the pool.
// This is provided for advanced use cases where callers want to explicitly return
// FilterExpressions to the pool after they are done with query processing.
// This is optional - expressions will be garbage collected normally if not released.
//
// This should only be called when the entire filter tree is no longer needed.
// WARNING: After calling this, the FilterExpression and all children are invalid.
func ReleaseFilterTree(expr *FilterExpression) {
	if expr == nil {
		return
	}
	// Recursively release children first
	ReleaseFilterTree(expr.Left)
	ReleaseFilterTree(expr.Right)
	// Release this expression
	releaseFilterExpression(expr)
}
