package query

import (
	"fmt"
	"reflect"

	"github.com/nlstn/go-odata/internal/metadata"
)

// conversionContext holds request-scoped state for AST to FilterExpression conversion.
// This replaces the previous global variable approach to ensure thread-safety
// when multiple requests are processed concurrently.
type conversionContext struct {
	computedAliases map[string]bool
	entityMetadata  *metadata.EntityMetadata
	maxInClauseSize int
	cache           *parserCache // Cache for resolved navigation paths
}

// hasComputedAlias checks if an alias is registered as a computed property
func (ctx *conversionContext) hasComputedAlias(alias string) bool {
	if ctx == nil || ctx.computedAliases == nil {
		return false
	}
	return ctx.computedAliases[alias]
}

// propertyExists checks if a property exists, using cache when available
func (ctx *conversionContext) propertyExists(propertyName string) bool {
	if ctx == nil {
		return false
	}
	// If entityMetadata is nil, we can't validate properties
	// This happens for lambda predicates where properties refer to collection element types
	if ctx.entityMetadata == nil {
		return true // Allow properties when metadata is unavailable
	}
	if ctx.cache != nil {
		return ctx.cache.propertyExistsWithCache(propertyName, ctx.entityMetadata)
	}
	return propertyExists(propertyName, ctx.entityMetadata)
}

// ASTToFilterExpressionWithComputed converts an AST to a FilterExpression with computed alias support
func ASTToFilterExpressionWithComputed(node ASTNode, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool, maxInClauseSize int) (*FilterExpression, error) {
	ctx := &conversionContext{
		computedAliases: computedAliases,
		entityMetadata:  entityMetadata,
		maxInClauseSize: maxInClauseSize,
		cache:           newParserCache(),
	}
	return astToFilterExpressionWithContext(node, ctx)
}

// ASTToFilterExpression converts an AST to a FilterExpression
func ASTToFilterExpression(node ASTNode, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	ctx := &conversionContext{
		computedAliases: nil,
		entityMetadata:  entityMetadata,
	}
	return astToFilterExpressionWithContext(node, ctx)
}

// astToFilterExpressionWithContext converts an AST to a FilterExpression using the provided context
func astToFilterExpressionWithContext(node ASTNode, ctx *conversionContext) (*FilterExpression, error) {
	var entityMetadata *metadata.EntityMetadata
	if ctx != nil {
		entityMetadata = ctx.entityMetadata
	}

	switch n := node.(type) {
	case *BinaryExpr:
		return convertBinaryExprWithContext(n, ctx)
	case *UnaryExpr:
		return convertUnaryExprWithContext(n, ctx)
	case *ComparisonExpr:
		return convertComparisonExprWithContext(n, ctx)
	case *FunctionCallExpr:
		return convertFunctionCallExprWithContext(n, entityMetadata, ctx)
	case *LambdaExpr:
		return convertLambdaExprWithContext(n, ctx)
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
		return astToFilterExpressionWithContext(n.Expr, ctx)
	}

	return nil, errUnsupportedASTNodeType
}

// convertBinaryExprWithContext converts a binary expression to a filter expression
func convertBinaryExprWithContext(n *BinaryExpr, ctx *conversionContext) (*FilterExpression, error) {
	left, err := astToFilterExpressionWithContext(n.Left, ctx)
	if err != nil {
		return nil, err
	}
	right, err := astToFilterExpressionWithContext(n.Right, ctx)
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

// convertUnaryExprWithContext converts a unary expression to a filter expression
func convertUnaryExprWithContext(n *UnaryExpr, ctx *conversionContext) (*FilterExpression, error) {
	if n.Operator == "not" {
		operand, err := astToFilterExpressionWithContext(n.Operand, ctx)
		if err != nil {
			return nil, err
		}
		// Represent NOT as a special filter expression
		operand.IsNot = true
		return operand, nil
	}
	return nil, fmt.Errorf("unsupported unary operator: %s", n.Operator)
}

// convertComparisonExprWithContext converts a comparison expression to a filter expression
func convertComparisonExprWithContext(n *ComparisonExpr, ctx *conversionContext) (*FilterExpression, error) {
	var entityMetadata *metadata.EntityMetadata
	if ctx != nil {
		entityMetadata = ctx.entityMetadata
	}

	// Check if left side is a function call
	if funcCall, ok := n.Left.(*FunctionCallExpr); ok {
		// Handle function calls like tolower(Name) eq 'value'
		funcExpr, err := convertFunctionCallExprWithContext(funcCall, entityMetadata, ctx)
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
		arithExpr, err := convertBinaryArithmeticExprWithContext(binExpr, ctx)
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
		return convertComparisonExprWithContext(unwrappedComparison, ctx)
	}

	// Handle 'in' operator with collection
	if n.Operator == "in" {
		property, err := extractPropertyFromComparisonWithContext(n.Left, ctx)
		if err != nil {
			return nil, err
		}

		// Right side must be a collection
		collExpr, ok := n.Right.(*CollectionExpr)
		if !ok {
			return nil, errInOperatorRequiresCollection
		}

		// Extract values from collection
		values := make([]interface{}, len(collExpr.Values))
		for i, valueNode := range collExpr.Values {
			if lit, ok := valueNode.(*LiteralExpr); ok {
				values[i] = lit.Value
			} else {
				return nil, errCollectionValuesMustBeLiterals
			}
		}

		// Validate IN clause size if configured
		if ctx != nil && ctx.maxInClauseSize > 0 && len(values) > ctx.maxInClauseSize {
			return nil, fmt.Errorf("IN clause size (%d) exceeds maximum allowed (%d)", len(values), ctx.maxInClauseSize)
		}

		return &FilterExpression{
			Property: property,
			Operator: OpIn,
			Value:    values,
		}, nil
	}

	property, err := extractPropertyFromComparisonWithContext(n.Left, ctx)
	if err != nil {
		return nil, err
	}

	value, err := extractValueFromComparison(n.Right)
	if err != nil {
		return nil, err
	}

	// Validate numeric value against property type (OData v4 spec compliance)
	if entityMetadata != nil {
		if err := validateValueAgainstPropertyType(property, value, entityMetadata); err != nil {
			return nil, err
		}
	}

	return &FilterExpression{
		Property: property,
		Operator: FilterOperator(n.Operator),
		Value:    value,
	}, nil
}

// validateValueAgainstPropertyType validates that a filter value is appropriate for the target property type.
// Returns an error if the value would overflow or is incompatible with the property type.
func validateValueAgainstPropertyType(property string, value interface{}, entityMetadata *metadata.EntityMetadata) error {
	// Only validate numeric values
	floatVal, ok := value.(float64)
	if !ok {
		return nil
	}

	// Get property metadata
	prop := entityMetadata.FindProperty(property)
	if prop == nil {
		// Property might be a navigation path or computed property - skip validation
		return nil
	}

	// Get the property's Go type (handling pointers)
	propType := prop.Type
	if propType.Kind() == reflect.Ptr {
		propType = propType.Elem()
	}

	// Check for Int64 overflow
	// Per OData v4 spec, numeric literals that overflow the target integer type should be rejected
	if propType.Kind() == reflect.Int64 || propType.Kind() == reflect.Uint64 {
		// Use 2^63 as the threshold instead of 2^63-1 to avoid float64 precision issues
		// Max int64 is 2^63-1 (9223372036854775807), which rounds to 9.223372036854776e+18 in float64
		// 2^63 (9223372036854775808) is exactly representable in float64 as 9.223372036854776e+18
		// So we check if the value is >= 2^63 to catch overflow
		const maxInt64Plus1 = 9223372036854775808.0 // 2^63
		const minInt64 = -9223372036854775808.0     // -2^63

		// Check if the value is out of Int64 range (including the exact boundary)
		// Values >= 2^63 or < -2^63 overflow Int64
		if floatVal >= maxInt64Plus1 || floatVal < minInt64 {
			return errNumericLiteralOutOfRange
		}
	}

	return nil
}

// extractPropertyFromComparisonWithContext extracts property name from the left side of a comparison
func extractPropertyFromComparisonWithContext(node ASTNode, ctx *conversionContext) (string, error) {
	if ident, ok := node.(*IdentifierExpr); ok {
		property := ident.Name
		// Validate property exists (either in entity metadata or as a computed alias)
		hasComputedAlias := ctx != nil && ctx.hasComputedAlias(property)
		if ctx != nil && !ctx.propertyExists(property) && !hasComputedAlias {
			return "", fmt.Errorf("property '%s' does not exist", property)
		}
		return property, nil
	}

	if binExpr, ok := node.(*BinaryExpr); ok {
		return extractPropertyFromArithmeticExprWithContext(binExpr, ctx)
	}

	if groupExpr, ok := node.(*GroupExpr); ok {
		// Unwrap grouped expression and try again
		return extractPropertyFromComparisonWithContext(groupExpr.Expr, ctx)
	}

	return "", errLeftSideOfCompMustBeProp
}

// convertBinaryArithmeticExprWithContext converts a binary arithmetic expression to a filter expression
func convertBinaryArithmeticExprWithContext(binExpr *BinaryExpr, ctx *conversionContext) (*FilterExpression, error) {
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
		hasComputedAlias := ctx != nil && ctx.hasComputedAlias(property)
		if ctx != nil && !ctx.propertyExists(property) && !hasComputedAlias {
			return nil, fmt.Errorf("property '%s' does not exist", property)
		}
	} else if leftBinExpr, ok := binExpr.Left.(*BinaryExpr); ok {
		// Nested arithmetic expression - recursively convert it
		leftFilterExpr, err := convertBinaryArithmeticExprWithContext(leftBinExpr, ctx)
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
		rightFilterExpr, err := convertBinaryArithmeticExprWithContext(rightBinExpr, ctx)
		if err != nil {
			return nil, err
		}
		// For nested expressions, store the converted expression
		// The actual SQL generation will need to handle this
		value = rightFilterExpr
	} else {
		return nil, errRightSideOfArithMustBeLitPropArith
	}

	return &FilterExpression{
		Property: property,
		Operator: op,
		Value:    value,
	}, nil
}

// extractPropertyFromArithmeticExprWithContext extracts property from arithmetic expression
func extractPropertyFromArithmeticExprWithContext(binExpr *BinaryExpr, ctx *conversionContext) (string, error) {
	if leftIdent, ok := binExpr.Left.(*IdentifierExpr); ok {
		property := leftIdent.Name
		// Validate property exists (either in entity metadata or as a computed alias)
		hasComputedAlias := ctx != nil && ctx.hasComputedAlias(property)
		if ctx != nil && !ctx.propertyExists(property) && !hasComputedAlias {
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
	return nil, errRightSideOfCompMustBeLitPropFunc
}
