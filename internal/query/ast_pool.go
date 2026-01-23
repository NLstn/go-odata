package query

import "sync"

// AST Node pools for reducing allocations during filter expression parsing.
// Using sync.Pool allows reusing AST nodes across parse operations, which can
// significantly reduce GC pressure in high-throughput scenarios.

var (
	binaryExprPool = sync.Pool{
		New: func() interface{} { return &BinaryExpr{} },
	}
	comparisonExprPool = sync.Pool{
		New: func() interface{} { return &ComparisonExpr{} },
	}
	literalExprPool = sync.Pool{
		New: func() interface{} { return &LiteralExpr{} },
	}
	unaryExprPool = sync.Pool{
		New: func() interface{} { return &UnaryExpr{} },
	}
	identifierExprPool = sync.Pool{
		New: func() interface{} { return &IdentifierExpr{} },
	}
	groupExprPool = sync.Pool{
		New: func() interface{} { return &GroupExpr{} },
	}
	functionCallExprPool = sync.Pool{
		New: func() interface{} { return &FunctionCallExpr{} },
	}
	collectionExprPool = sync.Pool{
		New: func() interface{} { return &CollectionExpr{} },
	}
	lambdaExprPool = sync.Pool{
		New: func() interface{} { return &LambdaExpr{} },
	}
)

// AcquireBinaryExpr gets a BinaryExpr from the pool
func AcquireBinaryExpr() *BinaryExpr {
	v := binaryExprPool.Get()
	if expr, ok := v.(*BinaryExpr); ok {
		return expr
	}
	return &BinaryExpr{}
}

// ReleaseBinaryExpr returns a BinaryExpr to the pool after clearing it
func ReleaseBinaryExpr(e *BinaryExpr) {
	if e == nil {
		return
	}
	e.Left = nil
	e.Operator = ""
	e.Right = nil
	binaryExprPool.Put(e)
}

// AcquireComparisonExpr gets a ComparisonExpr from the pool
func AcquireComparisonExpr() *ComparisonExpr {
	v := comparisonExprPool.Get()
	if expr, ok := v.(*ComparisonExpr); ok {
		return expr
	}
	return &ComparisonExpr{}
}

// ReleaseComparisonExpr returns a ComparisonExpr to the pool after clearing it
func ReleaseComparisonExpr(e *ComparisonExpr) {
	if e == nil {
		return
	}
	e.Left = nil
	e.Operator = ""
	e.Right = nil
	comparisonExprPool.Put(e)
}

// AcquireLiteralExpr gets a LiteralExpr from the pool
func AcquireLiteralExpr() *LiteralExpr {
	v := literalExprPool.Get()
	if expr, ok := v.(*LiteralExpr); ok {
		return expr
	}
	return &LiteralExpr{}
}

// ReleaseLiteralExpr returns a LiteralExpr to the pool after clearing it
func ReleaseLiteralExpr(e *LiteralExpr) {
	if e == nil {
		return
	}
	e.Value = nil
	e.Type = ""
	literalExprPool.Put(e)
}

// AcquireUnaryExpr gets a UnaryExpr from the pool
func AcquireUnaryExpr() *UnaryExpr {
	v := unaryExprPool.Get()
	if expr, ok := v.(*UnaryExpr); ok {
		return expr
	}
	return &UnaryExpr{}
}

// ReleaseUnaryExpr returns a UnaryExpr to the pool after clearing it
func ReleaseUnaryExpr(e *UnaryExpr) {
	if e == nil {
		return
	}
	e.Operator = ""
	e.Operand = nil
	unaryExprPool.Put(e)
}

// AcquireIdentifierExpr gets an IdentifierExpr from the pool
func AcquireIdentifierExpr() *IdentifierExpr {
	v := identifierExprPool.Get()
	if expr, ok := v.(*IdentifierExpr); ok {
		return expr
	}
	return &IdentifierExpr{}
}

// ReleaseIdentifierExpr returns an IdentifierExpr to the pool after clearing it
func ReleaseIdentifierExpr(e *IdentifierExpr) {
	if e == nil {
		return
	}
	e.Name = ""
	identifierExprPool.Put(e)
}

// AcquireGroupExpr gets a GroupExpr from the pool
func AcquireGroupExpr() *GroupExpr {
	v := groupExprPool.Get()
	if expr, ok := v.(*GroupExpr); ok {
		return expr
	}
	return &GroupExpr{}
}

// ReleaseGroupExpr returns a GroupExpr to the pool after clearing it
func ReleaseGroupExpr(e *GroupExpr) {
	if e == nil {
		return
	}
	e.Expr = nil
	groupExprPool.Put(e)
}

// AcquireFunctionCallExpr gets a FunctionCallExpr from the pool
func AcquireFunctionCallExpr() *FunctionCallExpr {
	v := functionCallExprPool.Get()
	if expr, ok := v.(*FunctionCallExpr); ok {
		return expr
	}
	return &FunctionCallExpr{}
}

// ReleaseFunctionCallExpr returns a FunctionCallExpr to the pool after clearing it
func ReleaseFunctionCallExpr(e *FunctionCallExpr) {
	if e == nil {
		return
	}
	e.Function = ""
	e.Args = nil
	functionCallExprPool.Put(e)
}

// AcquireCollectionExpr gets a CollectionExpr from the pool
func AcquireCollectionExpr() *CollectionExpr {
	v := collectionExprPool.Get()
	if expr, ok := v.(*CollectionExpr); ok {
		return expr
	}
	return &CollectionExpr{}
}

// ReleaseCollectionExpr returns a CollectionExpr to the pool after clearing it
func ReleaseCollectionExpr(e *CollectionExpr) {
	if e == nil {
		return
	}
	e.Values = nil
	collectionExprPool.Put(e)
}

// AcquireLambdaExpr gets a LambdaExpr from the pool
func AcquireLambdaExpr() *LambdaExpr {
	v := lambdaExprPool.Get()
	if expr, ok := v.(*LambdaExpr); ok {
		return expr
	}
	return &LambdaExpr{}
}

// ReleaseLambdaExpr returns a LambdaExpr to the pool after clearing it
func ReleaseLambdaExpr(e *LambdaExpr) {
	if e == nil {
		return
	}
	e.Collection = nil
	e.Operator = ""
	e.RangeVariable = ""
	e.Predicate = nil
	lambdaExprPool.Put(e)
}

// ReleaseASTNode releases an AST node back to its appropriate pool.
// This is a convenience function that determines the node type and calls the appropriate release function.
func ReleaseASTNode(node ASTNode) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *BinaryExpr:
		// Recursively release children first
		ReleaseASTNode(n.Left)
		ReleaseASTNode(n.Right)
		ReleaseBinaryExpr(n)
	case *ComparisonExpr:
		ReleaseASTNode(n.Left)
		ReleaseASTNode(n.Right)
		ReleaseComparisonExpr(n)
	case *LiteralExpr:
		ReleaseLiteralExpr(n)
	case *UnaryExpr:
		ReleaseASTNode(n.Operand)
		ReleaseUnaryExpr(n)
	case *IdentifierExpr:
		ReleaseIdentifierExpr(n)
	case *GroupExpr:
		ReleaseASTNode(n.Expr)
		ReleaseGroupExpr(n)
	case *FunctionCallExpr:
		for _, arg := range n.Args {
			ReleaseASTNode(arg)
		}
		ReleaseFunctionCallExpr(n)
	case *CollectionExpr:
		for _, v := range n.Values {
			ReleaseASTNode(v)
		}
		ReleaseCollectionExpr(n)
	case *LambdaExpr:
		ReleaseASTNode(n.Collection)
		ReleaseASTNode(n.Predicate)
		ReleaseLambdaExpr(n)
	}
}
