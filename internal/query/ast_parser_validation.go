package query

import (
	"fmt"

	"github.com/nlstn/go-odata/internal/metadata"
)

// computedAliasesContext holds computed property aliases for validation during AST conversion
var computedAliasesContext map[string]bool

// ASTToFilterExpressionWithComputed converts an AST to a FilterExpression with computed alias support
func ASTToFilterExpressionWithComputed(node ASTNode, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool) (*FilterExpression, error) {
	// Store computed aliases in context for use during conversion
	computedAliasesContext = computedAliases
	defer func() {
		// Clear context after conversion
		computedAliasesContext = nil
	}()

	return ASTToFilterExpression(node, entityMetadata)
}

// ASTToFilterExpression converts an AST to a FilterExpression
func ASTToFilterExpression(node ASTNode, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	switch n := node.(type) {
	case *BinaryExpr:
		return convertBinaryExpr(n, entityMetadata)
	case *UnaryExpr:
		return convertUnaryExpr(n, entityMetadata)
	case *ComparisonExpr:
		return convertComparisonExpr(n, entityMetadata)
	case *FunctionCallExpr:
		return convertFunctionCallExpr(n, entityMetadata)
	case *LambdaExpr:
		return convertLambdaExpr(n, entityMetadata)
	case *IdentifierExpr:
		// Standalone identifier (e.g., for boolean properties)
		return &FilterExpression{
			Property: n.Name,
			Operator: OpEqual,
			Value:    true,
		}, nil
	case *LiteralExpr:
		// Standalone literal
		return &FilterExpression{
			Value:    n.Value,
			Operator: OpEqual,
		}, nil
	case *GroupExpr:
		return ASTToFilterExpression(n.Expr, entityMetadata)
	}

	return nil, fmt.Errorf("unsupported AST node type")
}

// convertBinaryExpr converts a binary expression to a filter expression
func convertBinaryExpr(n *BinaryExpr, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	left, err := ASTToFilterExpression(n.Left, entityMetadata)
	if err != nil {
		return nil, err
	}
	right, err := ASTToFilterExpression(n.Right, entityMetadata)
	if err != nil {
		return nil, err
	}

	// Check if this is a logical operator
	if n.Operator == "and" || n.Operator == "or" {
		return &FilterExpression{
			Left:    left,
			Right:   right,
			Logical: LogicalOperator(n.Operator),
		}, nil
	}

	// Arithmetic operators - for now, we'll convert them to a simple expression
	// In a full implementation, this would need more sophisticated handling
	return &FilterExpression{
		Left:    left,
		Right:   right,
		Logical: LogicalOperator(n.Operator), // Store arithmetic operators as logical for now
	}, nil
}

// convertUnaryExpr converts a unary expression to a filter expression
func convertUnaryExpr(n *UnaryExpr, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	if n.Operator == "not" {
		operand, err := ASTToFilterExpression(n.Operand, entityMetadata)
		if err != nil {
			return nil, err
		}
		// Represent NOT as a special filter expression
		operand.IsNot = true
		return operand, nil
	}
	return nil, fmt.Errorf("unsupported unary operator: %s", n.Operator)
}

// convertComparisonExpr converts a comparison expression to a filter expression
func convertComparisonExpr(n *ComparisonExpr, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	// Check if left side is a function call
	if funcCall, ok := n.Left.(*FunctionCallExpr); ok {
		// Handle function calls like tolower(Name) eq 'value'
		funcExpr, err := convertFunctionCallExpr(funcCall, entityMetadata)
		if err != nil {
			return nil, err
		}

		// Get value from right side
		value, err := extractValueFromComparison(n.Right)
		if err != nil {
			return nil, err
		}

		// Store the function operator and the comparison operator together
		// The property holds the column, the operator holds the function,
		// and we store the comparison operator and value separately
		filterExpr := &FilterExpression{
			Property: funcExpr.Property,
			Operator: funcExpr.Operator, // The function (tolower, length, etc.)
			Value:    value,
		}

		// Store comparison info for SQL generation
		// We'll use a special marker in the property name to indicate this is a function comparison
		filterExpr.Property = fmt.Sprintf("_func_%s_%s_%s", funcExpr.Operator, funcExpr.Property, n.Operator)
		filterExpr.Operator = FilterOperator(n.Operator)

		// Store the original function info in Left for SQL generation
		filterExpr.Left = funcExpr

		return filterExpr, nil
	}

	// Check if left side is a binary expression (arithmetic operation)
	if binExpr, ok := n.Left.(*BinaryExpr); ok {
		// Handle arithmetic operations like Price mod 2 eq 1
		arithExpr, err := convertBinaryArithmeticExpr(binExpr, entityMetadata)
		if err != nil {
			return nil, err
		}

		// Get value from right side
		value, err := extractValueFromComparison(n.Right)
		if err != nil {
			return nil, err
		}

		return &FilterExpression{
			Property: arithExpr.Property,
			Operator: FilterOperator(n.Operator),
			Value:    value,
			Left:     arithExpr,
		}, nil
	}

	// Check if left side is a grouped expression
	if groupExpr, ok := n.Left.(*GroupExpr); ok {
		// Unwrap and re-process
		unwrappedComparison := &ComparisonExpr{
			Left:     groupExpr.Expr,
			Operator: n.Operator,
			Right:    n.Right,
		}
		return convertComparisonExpr(unwrappedComparison, entityMetadata)
	}

	// Handle 'in' operator with collection
	if n.Operator == "in" {
		property, err := extractPropertyFromComparison(n.Left, entityMetadata)
		if err != nil {
			return nil, err
		}

		// Right side must be a collection
		collExpr, ok := n.Right.(*CollectionExpr)
		if !ok {
			return nil, fmt.Errorf("'in' operator requires a collection on the right side")
		}

		// Extract values from collection
		values := make([]interface{}, len(collExpr.Values))
		for i, valueNode := range collExpr.Values {
			if lit, ok := valueNode.(*LiteralExpr); ok {
				values[i] = lit.Value
			} else {
				return nil, fmt.Errorf("collection values must be literals")
			}
		}

		return &FilterExpression{
			Property: property,
			Operator: OpIn,
			Value:    values,
		}, nil
	}

	property, err := extractPropertyFromComparison(n.Left, entityMetadata)
	if err != nil {
		return nil, err
	}

	value, err := extractValueFromComparison(n.Right)
	if err != nil {
		return nil, err
	}

	return &FilterExpression{
		Property: property,
		Operator: FilterOperator(n.Operator),
		Value:    value,
	}, nil
}

// extractPropertyFromComparison extracts property name from the left side of a comparison
func extractPropertyFromComparison(node ASTNode, entityMetadata *metadata.EntityMetadata) (string, error) {
	if ident, ok := node.(*IdentifierExpr); ok {
		property := ident.Name
		// Validate property exists (either in entity metadata or as a computed alias)
		if entityMetadata != nil && !propertyExists(property, entityMetadata) && !computedAliasesContext[property] {
			return "", fmt.Errorf("property '%s' does not exist", property)
		}
		return property, nil
	}

	if binExpr, ok := node.(*BinaryExpr); ok {
		return extractPropertyFromArithmeticExpr(binExpr, entityMetadata)
	}

	if groupExpr, ok := node.(*GroupExpr); ok {
		// Unwrap grouped expression and try again
		return extractPropertyFromComparison(groupExpr.Expr, entityMetadata)
	}

	return "", fmt.Errorf("left side of comparison must be a property name or arithmetic expression")
}

// convertBinaryArithmeticExpr converts a binary arithmetic expression to a filter expression
func convertBinaryArithmeticExpr(binExpr *BinaryExpr, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	// Map operator to FilterOperator
	var op FilterOperator
	switch binExpr.Operator {
	case "add", "+":
		op = OpAdd
	case "sub", "-":
		op = OpSub
	case "mul", "*":
		op = OpMul
	case "div", "/":
		op = OpDiv
	case "mod":
		op = OpMod
	default:
		return nil, fmt.Errorf("unsupported arithmetic operator: %s", binExpr.Operator)
	}

	// Extract property from left side
	var property string
	if leftIdent, ok := binExpr.Left.(*IdentifierExpr); ok {
		property = leftIdent.Name
		// Validate property exists (either in entity metadata or as a computed alias)
		if entityMetadata != nil && !propertyExists(property, entityMetadata) && !computedAliasesContext[property] {
			return nil, fmt.Errorf("property '%s' does not exist", property)
		}
	} else if leftBinExpr, ok := binExpr.Left.(*BinaryExpr); ok {
		// Nested arithmetic expression - recursively convert it
		leftFilterExpr, err := convertBinaryArithmeticExpr(leftBinExpr, entityMetadata)
		if err != nil {
			return nil, err
		}
		// For nested expressions, we'll use a placeholder property
		// The actual SQL generation will need to handle this recursively
		property = leftFilterExpr.Property
	} else {
		// For other complex cases, use a placeholder
		property = "_arithmetic_"
	}

	// Extract value from right side
	var value interface{}
	if rightLit, ok := binExpr.Right.(*LiteralExpr); ok {
		value = rightLit.Value
	} else if rightIdent, ok := binExpr.Right.(*IdentifierExpr); ok {
		value = rightIdent.Name
	} else if rightBinExpr, ok := binExpr.Right.(*BinaryExpr); ok {
		// Nested arithmetic expression on right side - recursively convert it
		// This handles cases like "Price add (10 mul 2)"
		rightFilterExpr, err := convertBinaryArithmeticExpr(rightBinExpr, entityMetadata)
		if err != nil {
			return nil, err
		}
		// For nested expressions, store the converted expression
		// The actual SQL generation will need to handle this
		value = rightFilterExpr
	} else {
		return nil, fmt.Errorf("right side of arithmetic expression must be a literal, property, or arithmetic expression")
	}

	return &FilterExpression{
		Property: property,
		Operator: op,
		Value:    value,
	}, nil
}

// extractPropertyFromArithmeticExpr extracts property from arithmetic expression
func extractPropertyFromArithmeticExpr(binExpr *BinaryExpr, entityMetadata *metadata.EntityMetadata) (string, error) {
	if leftIdent, ok := binExpr.Left.(*IdentifierExpr); ok {
		property := leftIdent.Name
		// Validate property exists (either in entity metadata or as a computed alias)
		if entityMetadata != nil && !propertyExists(property, entityMetadata) && !computedAliasesContext[property] {
			return "", fmt.Errorf("property '%s' does not exist", property)
		}
		return property, nil
	}
	// Complex arithmetic expression - for now, just use a placeholder
	return "arithmetic_expr", nil
}

// extractValueFromComparison extracts value from the right side of a comparison
func extractValueFromComparison(node ASTNode) (interface{}, error) {
	if lit, ok := node.(*LiteralExpr); ok {
		return lit.Value, nil
	}
	if ident, ok := node.(*IdentifierExpr); ok {
		// Allow identifiers as values (for property-to-property comparisons)
		return ident.Name, nil
	}
	// Support function calls on the right side (e.g., tolower(Name) eq tolower(Name))
	if funcCall, ok := node.(*FunctionCallExpr); ok {
		// Return a special marker that this is a function call
		// The actual function will be processed during SQL generation
		return funcCall, nil
	}
	return nil, fmt.Errorf("right side of comparison must be a literal, property, or function")
}
