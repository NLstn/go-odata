package query

import (
	"testing"
)

// Test that all AST node types implement the ASTNode interface
func TestASTNodeInterface(t *testing.T) {
	var _ ASTNode = &BinaryExpr{}
	var _ ASTNode = &UnaryExpr{}
	var _ ASTNode = &ComparisonExpr{}
	var _ ASTNode = &FunctionCallExpr{}
	var _ ASTNode = &IdentifierExpr{}
	var _ ASTNode = &LiteralExpr{}
	var _ ASTNode = &GroupExpr{}
	var _ ASTNode = &CollectionExpr{}
	var _ ASTNode = &LambdaExpr{}
}

func TestBinaryExpr(t *testing.T) {
	expr := &BinaryExpr{
		Left:     &IdentifierExpr{Name: "Price"},
		Operator: "add",
		Right:    &LiteralExpr{Value: 10, Type: "number"},
	}
	expr.astNode() // Call the method to ensure coverage
}

func TestUnaryExpr(t *testing.T) {
	expr := &UnaryExpr{
		Operator: "not",
		Operand:  &IdentifierExpr{Name: "Active"},
	}
	expr.astNode()
}

func TestComparisonExpr(t *testing.T) {
	expr := &ComparisonExpr{
		Left:     &IdentifierExpr{Name: "Price"},
		Operator: "gt",
		Right:    &LiteralExpr{Value: 100, Type: "number"},
	}
	expr.astNode()
}

func TestFunctionCallExpr(t *testing.T) {
	expr := &FunctionCallExpr{
		Function: "contains",
		Args: []ASTNode{
			&IdentifierExpr{Name: "Name"},
			&LiteralExpr{Value: "test", Type: "string"},
		},
	}
	expr.astNode()
}

func TestIdentifierExpr(t *testing.T) {
	expr := &IdentifierExpr{Name: "ProductName"}
	expr.astNode()
}

func TestLiteralExpr(t *testing.T) {
	expr := &LiteralExpr{Value: "test", Type: "string"}
	expr.astNode()
}

func TestGroupExpr(t *testing.T) {
	expr := &GroupExpr{
		Expr: &ComparisonExpr{
			Left:     &IdentifierExpr{Name: "Price"},
			Operator: "gt",
			Right:    &LiteralExpr{Value: 100, Type: "number"},
		},
	}
	expr.astNode()
}

func TestCollectionExpr(t *testing.T) {
	expr := &CollectionExpr{
		Values: []ASTNode{
			&LiteralExpr{Value: 1, Type: "number"},
			&LiteralExpr{Value: 2, Type: "number"},
			&LiteralExpr{Value: 3, Type: "number"},
		},
	}
	expr.astNode()
}

func TestLambdaExpr(t *testing.T) {
	expr := &LambdaExpr{
		Collection:    &IdentifierExpr{Name: "Tags"},
		Operator:      "any",
		RangeVariable: "t",
		Predicate: &ComparisonExpr{
			Left:     &IdentifierExpr{Name: "t/Name"},
			Operator: "eq",
			Right:    &LiteralExpr{Value: "test", Type: "string"},
		},
	}
	expr.astNode()
}
