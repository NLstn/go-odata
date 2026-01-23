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
		return &LambdaExpr{
			Collection:    &IdentifierExpr{Name: collectionPath},
			Operator:      operator,
			RangeVariable: "",
			Predicate:     nil,
		}, nil
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

	return &LambdaExpr{
		Collection:    &IdentifierExpr{Name: collectionPath},
		Operator:      operator,
		RangeVariable: rangeVariable,
		Predicate:     predicate,
	}, nil
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
			return &IdentifierExpr{Name: "$it"}
		}
		// Check if this is a property path starting with range variable
		if strings.HasPrefix(n.Name, rangeVariable+"/") {
			// Strip the range variable prefix
			return &IdentifierExpr{Name: strings.TrimPrefix(n.Name, rangeVariable+"/")}
		}
		return n

	case *BinaryExpr:
		return &BinaryExpr{
			Left:     replaceRangeVariableInAST(n.Left, rangeVariable),
			Operator: n.Operator,
			Right:    replaceRangeVariableInAST(n.Right, rangeVariable),
		}

	case *UnaryExpr:
		return &UnaryExpr{
			Operator: n.Operator,
			Operand:  replaceRangeVariableInAST(n.Operand, rangeVariable),
		}

	case *ComparisonExpr:
		return &ComparisonExpr{
			Left:     replaceRangeVariableInAST(n.Left, rangeVariable),
			Operator: n.Operator,
			Right:    replaceRangeVariableInAST(n.Right, rangeVariable),
		}

	case *FunctionCallExpr:
		newArgs := make([]ASTNode, len(n.Args))
		for i, arg := range n.Args {
			newArgs[i] = replaceRangeVariableInAST(arg, rangeVariable)
		}
		return &FunctionCallExpr{
			Function: n.Function,
			Args:     newArgs,
		}

	case *GroupExpr:
		return &GroupExpr{
			Expr: replaceRangeVariableInAST(n.Expr, rangeVariable),
		}

	case *LambdaExpr:
		// Nested lambda - recursively replace
		return &LambdaExpr{
			Collection:    replaceRangeVariableInAST(n.Collection, rangeVariable),
			Operator:      n.Operator,
			RangeVariable: n.RangeVariable,
			Predicate:     replaceRangeVariableInAST(n.Predicate, rangeVariable),
		}
	}

	// For literal expressions and other types, return as is
	return node
}
