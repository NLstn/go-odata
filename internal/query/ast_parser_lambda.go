package query

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// parseLambdaExpression parses a lambda expression like any(x: x/Price gt 100)
func (p *ASTParser) parseLambdaExpression(collectionPath, operator string) (ASTNode, error) {
	p.advance() // consume '('

	var rangeVariable string
	var predicate ASTNode

	// Check if this is parameterless any/all (e.g., Tags/any())
	if p.currentToken().Type == TokenRParen {
		p.advance() // consume ')'
		collIdent := AcquireIdentifierExpr()
		collIdent.Name = collectionPath
		lambdaExpr := AcquireLambdaExpr()
		lambdaExpr.Collection = collIdent
		lambdaExpr.Operator = operator
		lambdaExpr.RangeVariable = ""
		lambdaExpr.Predicate = nil
		return lambdaExpr, nil
	}

	// Parse range variable (e.g., "t" in "t: ...")
	if p.currentToken().Type == TokenIdentifier {
		rangeVariable = p.currentToken().Value
		p.advance()

		// Expect colon
		if err := p.expect(TokenColon); err != nil {
			return nil, fmt.Errorf("expected ':' after lambda range variable: %w", err)
		}

		// Parse the predicate
		var err error
		predicate, err = p.parseOr()
		if err != nil {
			return nil, fmt.Errorf("failed to parse lambda predicate: %w", err)
		}
	}

	if err := p.expect(TokenRParen); err != nil {
		return nil, err
	}

	collIdent := AcquireIdentifierExpr()
	collIdent.Name = collectionPath
	lambdaExpr := AcquireLambdaExpr()
	lambdaExpr.Collection = collIdent
	lambdaExpr.Operator = operator
	lambdaExpr.RangeVariable = rangeVariable
	lambdaExpr.Predicate = predicate
	return lambdaExpr, nil
}

// convertLambdaExprWithContext converts a lambda expression (any/all) to a filter expression using the provided context
func convertLambdaExprWithContext(n *LambdaExpr, ctx *conversionContext) (*FilterExpression, error) {
	// Extract collection property path
	collectionPath := ""
	if collIdent, ok := n.Collection.(*IdentifierExpr); ok {
		collectionPath = collIdent.Name
	} else {
		return nil, errLambdaCollMustBePropPath
	}

	// Create the lambda filter expression from the pool
	lambdaFilter := acquireFilterExpression()
	lambdaFilter.Property = collectionPath
	lambdaFilter.Operator = FilterOperator(n.Operator)

	// If there's a predicate, convert it
	if n.Predicate != nil {
		// For now, we'll store the range variable and predicate info
		// The predicate needs special handling because it refers to the range variable
		// Pass the context for computed alias validation within lambda predicates
		var entityMeta *metadata.EntityMetadata
		if ctx != nil {
			entityMeta = ctx.entityMetadata
		}
		predicate, err := convertLambdaPredicateWithRangeVariable(n.Predicate, n.RangeVariable, entityMeta)
		if err != nil {
			return nil, fmt.Errorf("failed to convert lambda predicate: %w", err)
		}

		// Store the predicate as the Left field
		lambdaFilter.Left = predicate
		// Store the range variable in the Value field for SQL generation
		lambdaFilter.Value = map[string]interface{}{
			"rangeVariable": n.RangeVariable,
			"predicate":     predicate,
		}
	} else {
		// Parameterless any/all - just checks if collection is non-empty
		lambdaFilter.Value = nil
	}

	return lambdaFilter, nil
}

// convertLambdaPredicateWithRangeVariable converts a lambda predicate, replacing range variable references
func convertLambdaPredicateWithRangeVariable(predicate ASTNode, rangeVariable string, _ *metadata.EntityMetadata) (*FilterExpression, error) {
	// Replace range variable references with property paths relative to the collection
	predicateWithReplacedVars := replaceRangeVariableInAST(predicate, rangeVariable)

	// Convert the modified AST to FilterExpression
	// Note: We pass nil for entityMetadata here because the properties in the predicate
	// refer to the collection element type, not the parent entity
	return ASTToFilterExpression(predicateWithReplacedVars, nil)
}

// replaceRangeVariableInAST replaces range variable references in the AST
func replaceRangeVariableInAST(node ASTNode, rangeVariable string) ASTNode {
	switch n := node.(type) {
	case *IdentifierExpr:
		// If the identifier matches the range variable, keep it as is
		// Otherwise, if it starts with rangeVariable/, strip the prefix
		if n.Name == rangeVariable {
			// This is a direct reference to the collection element
			// We'll represent this as a special marker
			expr := AcquireIdentifierExpr()
			expr.Name = "$it"
			return expr
		}
		// Check if this is a property path starting with range variable
		if strings.HasPrefix(n.Name, rangeVariable+"/") {
			// Strip the range variable prefix
			expr := AcquireIdentifierExpr()
			expr.Name = strings.TrimPrefix(n.Name, rangeVariable+"/")
			return expr
		}
		return n

	case *BinaryExpr:
		expr := AcquireBinaryExpr()
		expr.Left = replaceRangeVariableInAST(n.Left, rangeVariable)
		expr.Operator = n.Operator
		expr.Right = replaceRangeVariableInAST(n.Right, rangeVariable)
		return expr

	case *UnaryExpr:
		expr := AcquireUnaryExpr()
		expr.Operator = n.Operator
		expr.Operand = replaceRangeVariableInAST(n.Operand, rangeVariable)
		return expr

	case *ComparisonExpr:
		expr := AcquireComparisonExpr()
		expr.Left = replaceRangeVariableInAST(n.Left, rangeVariable)
		expr.Operator = n.Operator
		expr.Right = replaceRangeVariableInAST(n.Right, rangeVariable)
		return expr

	case *FunctionCallExpr:
		newArgs := make([]ASTNode, len(n.Args))
		for i, arg := range n.Args {
			newArgs[i] = replaceRangeVariableInAST(arg, rangeVariable)
		}
		expr := AcquireFunctionCallExpr()
		expr.Function = n.Function
		expr.Args = newArgs
		return expr

	case *GroupExpr:
		expr := AcquireGroupExpr()
		expr.Expr = replaceRangeVariableInAST(n.Expr, rangeVariable)
		return expr

	case *LambdaExpr:
		// Nested lambda - recursively replace
		expr := AcquireLambdaExpr()
		expr.Collection = replaceRangeVariableInAST(n.Collection, rangeVariable)
		expr.Operator = n.Operator
		expr.RangeVariable = n.RangeVariable
		expr.Predicate = replaceRangeVariableInAST(n.Predicate, rangeVariable)
		return expr
	}

	// For literal expressions and other types, return as is
	return node
}
