package query

import (
	"fmt"

	"github.com/nlstn/go-odata/internal/metadata"
)

// parseFunctionCall parses a function call like func(arg1, arg2)
func (p *ASTParser) parseFunctionCall(functionName string) (ASTNode, error) {
	p.advance() // consume '('

	var args []ASTNode

	// Parse function arguments
	if p.currentToken().Type != TokenRParen {
		for {
			arg, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if p.currentToken().Type == TokenComma {
				p.advance()
			} else {
				break
			}
		}
	}

	if err := p.expect(TokenRParen); err != nil {
		return nil, err
	}

	return &FunctionCallExpr{
		Function: functionName,
		Args:     args,
	}, nil
}

// convertFunctionCallExpr converts a function call expression to a filter expression
func convertFunctionCallExpr(n *FunctionCallExpr, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	return convertFunctionCallExprWithContext(n, entityMetadata, nil)
}

// convertFunctionCallExprWithContext converts a function call expression to a filter expression using the provided context
func convertFunctionCallExprWithContext(n *FunctionCallExpr, entityMetadata *metadata.EntityMetadata, ctx *conversionContext) (*FilterExpression, error) {
	functionName := n.Function

	// Handle zero-argument functions (now)
	if isZeroArgFunction(functionName) {
		return convertZeroArgFunction(n, functionName)
	}

	// Handle single-argument functions (tolower, toupper, trim, length)
	if isSingleArgFunction(functionName) {
		return convertSingleArgFunctionWithContext(n, functionName, entityMetadata, ctx)
	}

	// Handle concat specially (can have literals as first argument)
	if functionName == "concat" {
		return convertConcatFunctionWithContext(n, entityMetadata, ctx)
	}

	// Handle two-argument functions (contains, startswith, endswith, indexof)
	if isTwoArgFunction(functionName) {
		return convertTwoArgFunctionWithContext(n, functionName, entityMetadata, ctx)
	}

	// Handle arithmetic functions (add, sub, mul, div, mod)
	if isArithmeticFunction(functionName) {
		return convertArithmeticFunctionWithContext(n, functionName, entityMetadata, ctx)
	}

	// Handle substring function (2 or 3 arguments)
	if functionName == "substring" {
		return convertSubstringFunctionWithContext(n, entityMetadata, ctx)
	}

	// Handle cast function (2 arguments)
	if functionName == "cast" {
		return convertCastFunctionWithContext(n, entityMetadata, ctx)
	}

	// Handle isof function (1 or 2 arguments)
	if functionName == "isof" {
		return convertIsOfFunctionWithContext(n, entityMetadata, ctx)
	}

	// Handle geospatial functions
	if isGeospatialFunction(functionName) {
		return convertGeospatialFunctionWithContext(n, functionName, entityMetadata, ctx)
	}

	return nil, fmt.Errorf("unsupported function: %s", functionName)
}

// isZeroArgFunction checks if a function takes zero arguments
func isZeroArgFunction(name string) bool {
	return name == "now"
}

// isSingleArgFunction checks if a function takes a single argument
func isSingleArgFunction(name string) bool {
	return name == "tolower" || name == "toupper" || name == "trim" || name == "length" ||
		name == "year" || name == "month" || name == "day" ||
		name == "hour" || name == "minute" || name == "second" ||
		name == "date" || name == "time" ||
		name == "ceiling" || name == "floor" || name == "round"
}

// isTwoArgFunction checks if a function takes two arguments
func isTwoArgFunction(name string) bool {
	return name == "contains" || name == "startswith" || name == "endswith" ||
		name == "indexof" || name == "has"
}

// isArithmeticFunction checks if a function is an arithmetic function
func isArithmeticFunction(name string) bool {
	return name == "add" || name == "sub" || name == "mul" || name == "div" || name == "mod"
}

// isGeospatialFunction checks if a function is a geospatial function
func isGeospatialFunction(name string) bool {
	return name == "geo.distance" || name == "geo.length" || name == "geo.intersects"
}

// extractPropertyFromFunctionArgWithContext extracts property from function argument using the provided context
func extractPropertyFromFunctionArgWithContext(arg ASTNode, functionName string, entityMetadata *metadata.EntityMetadata, ctx *conversionContext) (string, error) {
	if ident, ok := arg.(*IdentifierExpr); ok {
		property := ident.Name
		// Validate property exists (either in entity metadata or as a computed alias)
		hasComputedAlias := ctx != nil && ctx.hasComputedAlias(property)
		if entityMetadata != nil && !propertyExists(property, entityMetadata) && !hasComputedAlias {
			return "", fmt.Errorf("property '%s' does not exist", property)
		}
		return property, nil
	}

	if funcCall, ok := arg.(*FunctionCallExpr); ok {
		innerExpr, err := convertFunctionCallExprWithContext(funcCall, entityMetadata, ctx)
		if err != nil {
			return "", err
		}
		return innerExpr.Property, nil
	}

	return "", fmt.Errorf("first argument of %s must be a property name or function call", functionName)
}

// convertZeroArgFunction converts zero-argument functions like now()
func convertZeroArgFunction(n *FunctionCallExpr, functionName string) (*FilterExpression, error) {
	if len(n.Args) != 0 {
		return nil, fmt.Errorf("function %s requires 0 arguments", functionName)
	}

	return &FilterExpression{
		Property: "", // Zero-arg functions don't operate on a property
		Operator: FilterOperator(functionName),
		Value:    nil,
	}, nil
}

// convertSingleArgFunctionWithContext converts single-argument functions using the provided context
func convertSingleArgFunctionWithContext(n *FunctionCallExpr, functionName string, entityMetadata *metadata.EntityMetadata, ctx *conversionContext) (*FilterExpression, error) {
	if len(n.Args) != 1 {
		return nil, fmt.Errorf("function %s requires 1 argument", functionName)
	}

	property, err := extractPropertyFromFunctionArgWithContext(n.Args[0], functionName, entityMetadata, ctx)
	if err != nil {
		return nil, err
	}

	return &FilterExpression{
		Property: property,
		Operator: FilterOperator(functionName),
		Value:    nil,
	}, nil
}

// convertTwoArgFunctionWithContext converts two-argument functions using the provided context
func convertTwoArgFunctionWithContext(n *FunctionCallExpr, functionName string, entityMetadata *metadata.EntityMetadata, ctx *conversionContext) (*FilterExpression, error) {
	if len(n.Args) != 2 {
		return nil, fmt.Errorf("function %s requires 2 arguments", functionName)
	}

	property, err := extractPropertyFromFunctionArgWithContext(n.Args[0], functionName, entityMetadata, ctx)
	if err != nil {
		return nil, err
	}

	// Second argument can be a literal, property, or function call
	var value interface{}
	if lit, ok := n.Args[1].(*LiteralExpr); ok {
		value = lit.Value
	} else if ident, ok := n.Args[1].(*IdentifierExpr); ok {
		// For concat, the second argument can be a property reference
		// We'll store it as a string and handle it in SQL generation
		value = ident.Name
	} else if funcCall, ok := n.Args[1].(*FunctionCallExpr); ok {
		// For concat, the second argument can be another function call
		// We'll store the function call for later processing
		value = funcCall
	} else {
		return nil, fmt.Errorf("second argument of %s must be a literal, property, or function", functionName)
	}

	return &FilterExpression{
		Property: property,
		Operator: FilterOperator(functionName),
		Value:    value,
	}, nil
}

// convertConcatFunctionWithContext converts concat function using the provided context
func convertConcatFunctionWithContext(n *FunctionCallExpr, entityMetadata *metadata.EntityMetadata, ctx *conversionContext) (*FilterExpression, error) {
	if len(n.Args) != 2 {
		return nil, fmt.Errorf("function concat requires 2 arguments")
	}

	// First argument can be a literal, property, or function call
	var firstArg interface{}
	var property string

	if lit, ok := n.Args[0].(*LiteralExpr); ok {
		// First argument is a literal
		firstArg = lit.Value
		property = "" // Empty property indicates literal-based concat
	} else if ident, ok := n.Args[0].(*IdentifierExpr); ok {
		// First argument is a property
		property = ident.Name
		// Validate property exists (either in entity metadata or as a computed alias)
		hasComputedAlias := ctx != nil && ctx.hasComputedAlias(property)
		if entityMetadata != nil && !propertyExists(property, entityMetadata) && !hasComputedAlias {
			return nil, fmt.Errorf("property '%s' does not exist", property)
		}
		firstArg = nil // Property is stored in Property field
	} else if funcCall, ok := n.Args[0].(*FunctionCallExpr); ok {
		// First argument is a function call
		innerExpr, err := convertFunctionCallExprWithContext(funcCall, entityMetadata, ctx)
		if err != nil {
			return nil, err
		}
		property = innerExpr.Property
		firstArg = funcCall // Store function call for later processing
	} else {
		return nil, fmt.Errorf("first argument of concat must be a literal, property, or function")
	}

	// Second argument can be literal, property, or function call
	var secondArg interface{}
	if lit, ok := n.Args[1].(*LiteralExpr); ok {
		secondArg = lit.Value
	} else if ident, ok := n.Args[1].(*IdentifierExpr); ok {
		secondArg = ident.Name
	} else if funcCall, ok := n.Args[1].(*FunctionCallExpr); ok {
		secondArg = funcCall
	} else {
		return nil, fmt.Errorf("second argument of concat must be a literal, property, or function")
	}

	// Store both arguments in Value as a slice for special handling
	var value interface{}
	if firstArg != nil {
		// First argument is a literal or function, store both arguments
		value = []interface{}{firstArg, secondArg}
	} else {
		// First argument is a property (stored in Property field), store only second argument
		value = secondArg
	}

	return &FilterExpression{
		Property: property,
		Operator: OpConcat,
		Value:    value,
	}, nil
}

// convertSubstringFunctionWithContext converts substring function using the provided context
func convertSubstringFunctionWithContext(n *FunctionCallExpr, entityMetadata *metadata.EntityMetadata, ctx *conversionContext) (*FilterExpression, error) {
	if len(n.Args) < 2 || len(n.Args) > 3 {
		return nil, fmt.Errorf("function substring requires 2 or 3 arguments")
	}

	property, err := extractPropertyFromFunctionArgWithContext(n.Args[0], "substring", entityMetadata, ctx)
	if err != nil {
		return nil, err
	}

	args := make([]interface{}, 0, len(n.Args)-1)
	for i := 1; i < len(n.Args); i++ {
		lit, ok := n.Args[i].(*LiteralExpr)
		if !ok {
			return nil, fmt.Errorf("substring arguments must be literals")
		}

		// Validate start index (argument 2, index 1)
		if i == 1 {
			switch v := lit.Value.(type) {
			case int:
				if v < 0 {
					return nil, fmt.Errorf("substring start parameter must be non-negative")
				}
			case int64:
				if v < 0 {
					return nil, fmt.Errorf("substring start parameter must be non-negative")
				}
			case float64:
				if v < 0 {
					return nil, fmt.Errorf("substring start parameter must be non-negative")
				}
			}
		}

		// Validate length parameter if present (argument 3, index 2)
		if i == 2 {
			// Length should also be non-negative
			switch v := lit.Value.(type) {
			case int:
				if v < 0 {
					return nil, fmt.Errorf("substring length parameter must be non-negative")
				}
			case int64:
				if v < 0 {
					return nil, fmt.Errorf("substring length parameter must be non-negative")
				}
			case float64:
				if v < 0 {
					return nil, fmt.Errorf("substring length parameter must be non-negative")
				}
			}
		}

		args = append(args, lit.Value)
	}

	return &FilterExpression{
		Property: property,
		Operator: OpSubstring,
		Value:    args,
	}, nil
}

// convertArithmeticFunctionWithContext converts arithmetic functions using the provided context
func convertArithmeticFunctionWithContext(n *FunctionCallExpr, functionName string, entityMetadata *metadata.EntityMetadata, ctx *conversionContext) (*FilterExpression, error) {
	if len(n.Args) != 2 {
		return nil, fmt.Errorf("function %s requires 2 arguments", functionName)
	}

	// First argument can be a property or another arithmetic expression
	var property string

	// Check if first argument is a property
	if ident, ok := n.Args[0].(*IdentifierExpr); ok {
		property = ident.Name
		// Validate property exists (either in entity metadata or as a computed alias)
		hasComputedAlias := ctx != nil && ctx.hasComputedAlias(property)
		if entityMetadata != nil && !propertyExists(property, entityMetadata) && !hasComputedAlias {
			return nil, fmt.Errorf("property '%s' does not exist", property)
		}
	} else {
		// For complex expressions, use a placeholder
		property = "_arithmetic_"
	}

	// Second argument should be a literal or identifier
	var value interface{}
	if lit, ok := n.Args[1].(*LiteralExpr); ok {
		value = lit.Value
	} else if ident, ok := n.Args[1].(*IdentifierExpr); ok {
		// Allow property references as second argument
		value = ident.Name
	} else {
		return nil, fmt.Errorf("second argument of %s must be a literal or property", functionName)
	}

	return &FilterExpression{
		Property: property,
		Operator: FilterOperator(functionName),
		Value:    value,
	}, nil
}

// convertCastFunctionWithContext converts cast function using the provided context
func convertCastFunctionWithContext(n *FunctionCallExpr, entityMetadata *metadata.EntityMetadata, ctx *conversionContext) (*FilterExpression, error) {
	if len(n.Args) != 2 {
		return nil, fmt.Errorf("function cast requires 2 arguments")
	}

	// First argument can be a property or another function call
	property, err := extractPropertyFromFunctionArgWithContext(n.Args[0], "cast", entityMetadata, ctx)
	if err != nil {
		return nil, err
	}

	// Second argument should be a type name (either as an identifier or string literal)
	var typeName string

	// Try as identifier first (OData v4 spec: unquoted type names)
	if ident, ok := n.Args[1].(*IdentifierExpr); ok {
		typeName = ident.Name
	} else if lit, ok := n.Args[1].(*LiteralExpr); ok {
		// Also accept string literals for backwards compatibility
		typeNameVal, ok := lit.Value.(string)
		if !ok {
			return nil, fmt.Errorf("second argument of cast must be a type name")
		}
		typeName = typeNameVal
	} else {
		return nil, fmt.Errorf("second argument of cast must be a type name")
	}

	// Validate the type name (basic validation)
	validTypes := map[string]bool{
		"Edm.String":         true,
		"Edm.Int32":          true,
		"Edm.Int64":          true,
		"Edm.Decimal":        true,
		"Edm.Double":         true,
		"Edm.Single":         true,
		"Edm.Boolean":        true,
		"Edm.DateTimeOffset": true,
		"Edm.Date":           true,
		"Edm.TimeOfDay":      true,
		"Edm.Guid":           true,
		"Edm.Binary":         true,
		"Edm.Byte":           true,
		"Edm.SByte":          true,
		"Edm.Int16":          true,
	}

	if !validTypes[typeName] {
		return nil, fmt.Errorf("unsupported cast type: %s", typeName)
	}

	return &FilterExpression{
		Property: property,
		Operator: OpCast,
		Value:    typeName,
	}, nil
}

// convertIsOfFunctionWithContext converts isof function using the provided context
func convertIsOfFunctionWithContext(n *FunctionCallExpr, entityMetadata *metadata.EntityMetadata, ctx *conversionContext) (*FilterExpression, error) {
	// isof can have 1 or 2 arguments
	if len(n.Args) < 1 || len(n.Args) > 2 {
		return nil, fmt.Errorf("function isof requires 1 or 2 arguments")
	}

	var property string
	var typeName string

	if len(n.Args) == 1 {
		// Single argument form: isof(TypeName) or isof('TypeName')
		// This checks the type of the current instance (implicit property)

		// Try as identifier first (OData v4 spec: unquoted type names)
		if ident, ok := n.Args[0].(*IdentifierExpr); ok {
			typeName = ident.Name
		} else if lit, ok := n.Args[0].(*LiteralExpr); ok {
			// Also accept string literals for backwards compatibility
			typeNameVal, ok := lit.Value.(string)
			if !ok {
				return nil, fmt.Errorf("argument of isof must be a type name")
			}
			typeName = typeNameVal
		} else {
			return nil, fmt.Errorf("argument of isof must be a type name")
		}

		property = "$it" // Special marker for current instance
	} else {
		// Two argument form: isof(property, TypeName) or isof(property, 'TypeName')
		var err error
		property, err = extractPropertyFromFunctionArgWithContext(n.Args[0], "isof", entityMetadata, ctx)
		if err != nil {
			return nil, err
		}

		// Second argument should be a type name (either as an identifier or string literal)
		// Try as identifier first (OData v4 spec: unquoted type names)
		if ident, ok := n.Args[1].(*IdentifierExpr); ok {
			typeName = ident.Name
		} else if lit, ok := n.Args[1].(*LiteralExpr); ok {
			// Also accept string literals for backwards compatibility
			typeNameVal, ok := lit.Value.(string)
			if !ok {
				return nil, fmt.Errorf("second argument of isof must be a type name")
			}
			typeName = typeNameVal
		} else {
			return nil, fmt.Errorf("second argument of isof must be a type name")
		}
	}

	// Validate the type name (basic validation)
	validTypes := map[string]bool{
		"Edm.String":         true,
		"Edm.Int32":          true,
		"Edm.Int64":          true,
		"Edm.Decimal":        true,
		"Edm.Double":         true,
		"Edm.Single":         true,
		"Edm.Boolean":        true,
		"Edm.DateTimeOffset": true,
		"Edm.Date":           true,
		"Edm.TimeOfDay":      true,
		"Edm.Guid":           true,
		"Edm.Binary":         true,
		"Edm.Byte":           true,
		"Edm.SByte":          true,
		"Edm.Int16":          true,
	}

	// Check if it's a valid EDM type
	isEdmType := validTypes[typeName]

	// If it's not an EDM type, check if it might be an entity type
	// Entity types don't have the "Edm." prefix and should start with uppercase
	// For entity type checks, we'll accept any non-Edm type that looks like a valid identifier
	isEntityType := false
	if !isEdmType && len(typeName) > 0 {
		// Check if it starts with "Edm."
		hasEdmPrefix := len(typeName) >= 4 && typeName[:4] == "Edm."
		// Entity types should not have the Edm. prefix and should start with uppercase
		if !hasEdmPrefix && len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z' {
			isEntityType = true
		}
	}

	if !isEdmType && !isEntityType {
		return nil, fmt.Errorf("unsupported isof type: %s", typeName)
	}

	return &FilterExpression{
		Property: property,
		Operator: OpIsOf,
		Value:    typeName,
	}, nil
}

// convertGeospatialFunctionWithContext converts geospatial functions using the provided context
func convertGeospatialFunctionWithContext(n *FunctionCallExpr, functionName string, entityMetadata *metadata.EntityMetadata, ctx *conversionContext) (*FilterExpression, error) {
	switch functionName {
	case "geo.distance":
		// geo.distance(point1, point2) - requires 2 arguments
		if len(n.Args) != 2 {
			return nil, fmt.Errorf("function geo.distance requires 2 arguments")
		}

		// First argument should be a property (the location field)
		property, err := extractPropertyFromFunctionArgWithContext(n.Args[0], "geo.distance", entityMetadata, ctx)
		if err != nil {
			return nil, err
		}

		// Second argument should be a geography/geometry literal
		var geoValue interface{}
		if lit, ok := n.Args[1].(*LiteralExpr); ok {
			geoValue = lit.Value
		} else {
			return nil, fmt.Errorf("second argument of geo.distance must be a geography or geometry literal")
		}

		return &FilterExpression{
			Property: property,
			Operator: OpGeoDistance,
			Value:    geoValue,
		}, nil

	case "geo.length":
		// geo.length(linestring) - requires 1 argument
		if len(n.Args) != 1 {
			return nil, fmt.Errorf("function geo.length requires 1 argument")
		}

		property, err := extractPropertyFromFunctionArgWithContext(n.Args[0], "geo.length", entityMetadata, ctx)
		if err != nil {
			return nil, err
		}

		return &FilterExpression{
			Property: property,
			Operator: OpGeoLength,
			Value:    nil,
		}, nil

	case "geo.intersects":
		// geo.intersects(geo1, geo2) - requires 2 arguments
		if len(n.Args) != 2 {
			return nil, fmt.Errorf("function geo.intersects requires 2 arguments")
		}

		// First argument should be a property
		property, err := extractPropertyFromFunctionArgWithContext(n.Args[0], "geo.intersects", entityMetadata, ctx)
		if err != nil {
			return nil, err
		}

		// Second argument should be a geography/geometry literal
		var geoValue interface{}
		if lit, ok := n.Args[1].(*LiteralExpr); ok {
			geoValue = lit.Value
		} else {
			return nil, fmt.Errorf("second argument of geo.intersects must be a geography or geometry literal")
		}

		return &FilterExpression{
			Property: property,
			Operator: OpGeoIntersects,
			Value:    geoValue,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported geospatial function: %s", functionName)
	}
}
