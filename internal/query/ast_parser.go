package query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// ASTParser parses filter expressions into an AST
type ASTParser struct {
	tokens  []*Token
	current int
}

// NewASTParser creates a new AST parser
func NewASTParser(tokens []*Token) *ASTParser {
	return &ASTParser{
		tokens:  tokens,
		current: 0,
	}
}

// currentToken returns the current token
func (p *ASTParser) currentToken() *Token {
	if p.current >= len(p.tokens) {
		return &Token{Type: TokenEOF}
	}
	return p.tokens[p.current]
}

// advance moves to the next token
func (p *ASTParser) advance() *Token {
	token := p.currentToken()
	if p.current < len(p.tokens)-1 {
		p.current++
	}
	return token
}

// expect checks if the current token matches the expected type and advances
func (p *ASTParser) expect(tokenType TokenType) error {
	token := p.currentToken()
	if token.Type != tokenType {
		return fmt.Errorf("expected token type %v, got %v at position %d", tokenType, token.Type, token.Pos)
	}
	p.advance()
	return nil
}

// Parse parses the tokens into an AST
func (p *ASTParser) Parse() (ASTNode, error) {
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	// Verify all tokens were consumed (except EOF)
	if p.currentToken().Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token after expression: %v at position %d",
			p.currentToken().Type, p.currentToken().Pos)
	}

	return node, nil
}

// parseOr handles OR expressions (lowest precedence)
func (p *ASTParser) parseOr() (ASTNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.currentToken().Type == TokenLogical && p.currentToken().Value == "or" {
		op := p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Left:     left,
			Operator: op.Value,
			Right:    right,
		}
	}

	return left, nil
}

// parseAnd handles AND expressions
func (p *ASTParser) parseAnd() (ASTNode, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.currentToken().Type == TokenLogical && p.currentToken().Value == "and" {
		op := p.advance()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Left:     left,
			Operator: op.Value,
			Right:    right,
		}
	}

	return left, nil
}

// parseNot handles NOT expressions
func (p *ASTParser) parseNot() (ASTNode, error) {
	if p.currentToken().Type == TokenNot {
		op := p.advance()
		operand, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{
			Operator: op.Value,
			Operand:  operand,
		}, nil
	}

	return p.parseComparison()
}

// parseComparison handles comparison expressions
func (p *ASTParser) parseComparison() (ASTNode, error) {
	left, err := p.parseArithmetic()
	if err != nil {
		return nil, err
	}

	// Check for comparison operators
	if p.currentToken().Type == TokenOperator {
		op := p.advance()

		// Special handling for 'in' operator - expect a collection
		if op.Value == "in" {
			right, err := p.parseCollection()
			if err != nil {
				return nil, err
			}
			return &ComparisonExpr{
				Left:     left,
				Operator: op.Value,
				Right:    right,
			}, nil
		}

		right, err := p.parseArithmetic()
		if err != nil {
			return nil, err
		}
		return &ComparisonExpr{
			Left:     left,
			Operator: op.Value,
			Right:    right,
		}, nil
	}

	return left, nil
}

// parseArithmetic handles arithmetic expressions
func (p *ASTParser) parseArithmetic() (ASTNode, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for p.currentToken().Type == TokenArithmetic &&
		(p.currentToken().Value == "+" || p.currentToken().Value == "-" ||
			p.currentToken().Value == "add" || p.currentToken().Value == "sub") {
		op := p.advance()
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Left:     left,
			Operator: op.Value,
			Right:    right,
		}
	}

	return left, nil
}

// parseTerm handles multiplication, division, and modulo
func (p *ASTParser) parseTerm() (ASTNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for p.currentToken().Type == TokenArithmetic &&
		(p.currentToken().Value == "*" || p.currentToken().Value == "/" ||
			p.currentToken().Value == "mul" || p.currentToken().Value == "div" ||
			p.currentToken().Value == "mod") {
		op := p.advance()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			Left:     left,
			Operator: op.Value,
			Right:    right,
		}
	}

	return left, nil
}

// parsePrimary handles primary expressions (literals, identifiers, function calls, grouped expressions)
func (p *ASTParser) parsePrimary() (ASTNode, error) {
	token := p.currentToken()

	// Grouped expression
	if token.Type == TokenLParen {
		return p.parseGroupedExpression()
	}

	// Literals
	if node := p.parseLiteral(token); node != nil {
		return node, nil
	}

	// Identifier or function call
	if token.Type == TokenIdentifier {
		return p.parseIdentifierOrFunctionCall(token)
	}

	return nil, fmt.Errorf("unexpected token %v at position %d", token.Type, token.Pos)
}

// parseGroupedExpression parses a grouped expression like (expr)
func (p *ASTParser) parseGroupedExpression() (ASTNode, error) {
	p.advance() // consume '('
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if err := p.expect(TokenRParen); err != nil {
		return nil, err
	}
	return &GroupExpr{Expr: expr}, nil
}

// parseLiteral parses literal values (string, number, boolean, null)
func (p *ASTParser) parseLiteral(token *Token) ASTNode {
	switch token.Type {
	case TokenString:
		p.advance()
		return &LiteralExpr{Value: token.Value, Type: "string"}
	case TokenNumber:
		p.advance()
		return p.parseNumberLiteral(token.Value)
	case TokenBoolean:
		p.advance()
		boolVal := token.Value == "true"
		return &LiteralExpr{Value: boolVal, Type: "boolean"}
	case TokenNull:
		p.advance()
		return &LiteralExpr{Value: nil, Type: "null"}
	default:
		return nil
	}
}

// parseNumberLiteral parses a number literal (integer or float)
func (p *ASTParser) parseNumberLiteral(value string) ASTNode {
	// Try to parse as integer first, then as float
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return &LiteralExpr{Value: intVal, Type: "number"}
	}
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return &LiteralExpr{Value: floatVal, Type: "number"}
	}
	return &LiteralExpr{Value: value, Type: "number"}
}

// parseIdentifierOrFunctionCall parses an identifier or function call
func (p *ASTParser) parseIdentifierOrFunctionCall(token *Token) (ASTNode, error) {
	p.advance()

	// Check for property path with slashes (e.g., Orders/Items or Tags/any)
	// Use "/" as path separator when followed by an identifier (not in arithmetic context)
	if p.currentToken().Type == TokenArithmetic && p.currentToken().Value == "/" {
		// Peek ahead to see if this is a property path or arithmetic division
		nextPos := p.current + 1
		if nextPos < len(p.tokens) && p.tokens[nextPos].Type == TokenIdentifier {
			return p.parsePropertyPath(token.Value)
		}
	}

	// Check if this is a function call
	if p.currentToken().Type == TokenLParen {
		return p.parseFunctionCall(token.Value)
	}

	return &IdentifierExpr{Name: token.Value}, nil
}

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

// parsePropertyPath parses a property path with slashes (e.g., Orders/Items/any)
func (p *ASTParser) parsePropertyPath(initialProp string) (ASTNode, error) {
	path := initialProp

	// Build the property path
	for p.currentToken().Type == TokenArithmetic && p.currentToken().Value == "/" {
		p.advance() // consume '/'

		if p.currentToken().Type != TokenIdentifier {
			return nil, fmt.Errorf("expected identifier after '/' in property path at position %d", p.currentToken().Pos)
		}

		nextProp := p.currentToken().Value
		p.advance()

		// Check if this is a lambda operator (any/all)
		lowerProp := strings.ToLower(nextProp)
		if lowerProp == "any" || lowerProp == "all" {
			if p.currentToken().Type == TokenLParen {
				return p.parseLambdaExpression(path, lowerProp)
			}
			// If not followed by '(', treat as regular property
		}

		// Continue building path
		path = path + "/" + nextProp
	}

	// If it ends with a function call, parse it
	if p.currentToken().Type == TokenLParen {
		// Check if the last part of the path is a lambda operator
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			lastPart := strings.ToLower(parts[len(parts)-1])
			if lastPart == "any" || lastPart == "all" {
				// Remove the lambda operator from path and parse lambda
				collectionPath := strings.Join(parts[:len(parts)-1], "/")
				return p.parseLambdaExpression(collectionPath, lastPart)
			}
		}
		// Otherwise treat as regular function call
		return p.parseFunctionCall(path)
	}

	return &IdentifierExpr{Name: path}, nil
}

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

// parseCollection parses a collection expression like (value1, value2, value3)
func (p *ASTParser) parseCollection() (ASTNode, error) {
	if err := p.expect(TokenLParen); err != nil {
		return nil, fmt.Errorf("expected '(' after 'in' operator: %w", err)
	}

	var values []ASTNode

	// Parse collection values
	if p.currentToken().Type != TokenRParen {
		for {
			// Parse a primary value (literal or identifier)
			value, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			values = append(values, value)

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

	return &CollectionExpr{Values: values}, nil
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

		// Store comparison info for SQL generation
		filterExpr := &FilterExpression{
			Property: fmt.Sprintf("_func_%s_%s_%s", arithExpr.Operator, arithExpr.Property, n.Operator),
			Operator: FilterOperator(n.Operator),
			Value:    value,
		}

		// Store the original arithmetic info in Left for SQL generation
		filterExpr.Left = arithExpr

		return filterExpr, nil
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
		// Validate property exists
		if entityMetadata != nil && !propertyExists(property, entityMetadata) {
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
		// Validate property exists
		if entityMetadata != nil && !propertyExists(property, entityMetadata) {
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
		// Validate property exists
		if entityMetadata != nil && !propertyExists(property, entityMetadata) {
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
		// Allow identifiers as values (for property-to-property comparisons in the future)
		return ident.Name, nil
	}
	return nil, fmt.Errorf("right side of comparison must be a literal or property")
}

// convertFunctionCallExpr converts a function call expression to a filter expression
func convertFunctionCallExpr(n *FunctionCallExpr, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	functionName := n.Function

	// Handle single-argument functions (tolower, toupper, trim, length)
	if isSingleArgFunction(functionName) {
		return convertSingleArgFunction(n, functionName, entityMetadata)
	}

	// Handle two-argument functions (contains, startswith, endswith, indexof, concat)
	if isTwoArgFunction(functionName) {
		return convertTwoArgFunction(n, functionName, entityMetadata)
	}

	// Handle arithmetic functions (add, sub, mul, div, mod)
	if isArithmeticFunction(functionName) {
		return convertArithmeticFunction(n, functionName, entityMetadata)
	}

	// Handle substring function (2 or 3 arguments)
	if functionName == "substring" {
		return convertSubstringFunction(n, entityMetadata)
	}

	// Handle cast function (2 arguments)
	if functionName == "cast" {
		return convertCastFunction(n, entityMetadata)
	}

	return nil, fmt.Errorf("unsupported function: %s", functionName)
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
		name == "indexof" || name == "concat" || name == "has"
}

// isArithmeticFunction checks if a function is an arithmetic function
func isArithmeticFunction(name string) bool {
	return name == "add" || name == "sub" || name == "mul" || name == "div" || name == "mod"
}

// extractPropertyFromFunctionArg extracts property from function argument
func extractPropertyFromFunctionArg(arg ASTNode, functionName string, entityMetadata *metadata.EntityMetadata) (string, error) {
	if ident, ok := arg.(*IdentifierExpr); ok {
		property := ident.Name
		if entityMetadata != nil && !propertyExists(property, entityMetadata) {
			return "", fmt.Errorf("property '%s' does not exist", property)
		}
		return property, nil
	}

	if funcCall, ok := arg.(*FunctionCallExpr); ok {
		innerExpr, err := convertFunctionCallExpr(funcCall, entityMetadata)
		if err != nil {
			return "", err
		}
		return innerExpr.Property, nil
	}

	return "", fmt.Errorf("first argument of %s must be a property name or function call", functionName)
}

// convertSingleArgFunction converts single-argument functions
func convertSingleArgFunction(n *FunctionCallExpr, functionName string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	if len(n.Args) != 1 {
		return nil, fmt.Errorf("function %s requires 1 argument", functionName)
	}

	property, err := extractPropertyFromFunctionArg(n.Args[0], functionName, entityMetadata)
	if err != nil {
		return nil, err
	}

	return &FilterExpression{
		Property: property,
		Operator: FilterOperator(functionName),
		Value:    nil,
	}, nil
}

// convertTwoArgFunction converts two-argument functions
func convertTwoArgFunction(n *FunctionCallExpr, functionName string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	if len(n.Args) != 2 {
		return nil, fmt.Errorf("function %s requires 2 arguments", functionName)
	}

	property, err := extractPropertyFromFunctionArg(n.Args[0], functionName, entityMetadata)
	if err != nil {
		return nil, err
	}

	// Second argument should be a literal
	lit, ok := n.Args[1].(*LiteralExpr)
	if !ok {
		return nil, fmt.Errorf("second argument of %s must be a literal", functionName)
	}

	return &FilterExpression{
		Property: property,
		Operator: FilterOperator(functionName),
		Value:    lit.Value,
	}, nil
}

// convertSubstringFunction converts substring function
func convertSubstringFunction(n *FunctionCallExpr, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	if len(n.Args) < 2 || len(n.Args) > 3 {
		return nil, fmt.Errorf("function substring requires 2 or 3 arguments")
	}

	property, err := extractPropertyFromFunctionArg(n.Args[0], "substring", entityMetadata)
	if err != nil {
		return nil, err
	}

	// Collect numeric arguments
	args := []interface{}{}
	for i := 1; i < len(n.Args); i++ {
		lit, ok := n.Args[i].(*LiteralExpr)
		if !ok {
			return nil, fmt.Errorf("argument %d of substring must be a number", i+1)
		}
		args = append(args, lit.Value)
	}

	return &FilterExpression{
		Property: property,
		Operator: OpSubstring,
		Value:    args,
	}, nil
}

// convertArithmeticFunction converts arithmetic functions (add, sub, mul, div, mod)
func convertArithmeticFunction(n *FunctionCallExpr, functionName string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	if len(n.Args) != 2 {
		return nil, fmt.Errorf("function %s requires 2 arguments", functionName)
	}

	// First argument can be a property or another arithmetic expression
	var property string

	// Check if first argument is a property
	if ident, ok := n.Args[0].(*IdentifierExpr); ok {
		property = ident.Name
		if entityMetadata != nil && !propertyExists(property, entityMetadata) {
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

// convertCastFunction converts cast function
// Format: cast(property, 'TypeName')
func convertCastFunction(n *FunctionCallExpr, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	if len(n.Args) != 2 {
		return nil, fmt.Errorf("function cast requires 2 arguments")
	}

	// First argument can be a property or another function call
	property, err := extractPropertyFromFunctionArg(n.Args[0], "cast", entityMetadata)
	if err != nil {
		return nil, err
	}

	// Second argument should be a string literal representing the target type
	lit, ok := n.Args[1].(*LiteralExpr)
	if !ok {
		return nil, fmt.Errorf("second argument of cast must be a type name string")
	}

	typeName, ok := lit.Value.(string)
	if !ok {
		return nil, fmt.Errorf("second argument of cast must be a string")
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

// convertLambdaExpr converts a lambda expression (any/all) to a filter expression
func convertLambdaExpr(n *LambdaExpr, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	// Extract collection property path
	collectionPath := ""
	if collIdent, ok := n.Collection.(*IdentifierExpr); ok {
		collectionPath = collIdent.Name
	} else {
		return nil, fmt.Errorf("lambda collection must be a property path")
	}

	// Create the lambda filter expression
	lambdaFilter := &FilterExpression{
		Property: collectionPath,
		Operator: FilterOperator(n.Operator),
	}

	// If there's a predicate, convert it
	if n.Predicate != nil {
		// For now, we'll store the range variable and predicate info
		// The predicate needs special handling because it refers to the range variable
		predicate, err := convertLambdaPredicateWithRangeVariable(n.Predicate, n.RangeVariable, entityMetadata)
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
