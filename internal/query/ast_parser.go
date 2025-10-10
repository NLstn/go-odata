package query

import (
	"fmt"
	"strconv"

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
		(p.currentToken().Value == "+" || p.currentToken().Value == "-") {
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
		(p.currentToken().Value == "*" || p.currentToken().Value == "/" || p.currentToken().Value == "mod") {
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

	return "", fmt.Errorf("left side of comparison must be a property name or arithmetic expression")
}

// extractPropertyFromArithmeticExpr extracts property from arithmetic expression
func extractPropertyFromArithmeticExpr(binExpr *BinaryExpr, entityMetadata *metadata.EntityMetadata) (string, error) {
	if leftIdent, ok := binExpr.Left.(*IdentifierExpr); ok {
		property := leftIdent.Name
		// Validate property exists
		if entityMetadata != nil && !propertyExists(property, entityMetadata) {
			return "", fmt.Errorf("property '%s' does not exist", property)
		}
		// For modulo operation, we'll create a special representation
		if binExpr.Operator == "mod" {
			if rightLit, ok := binExpr.Right.(*LiteralExpr); ok {
				return fmt.Sprintf("%s_mod_%v", property, rightLit.Value), nil
			}
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
	// Handle function calls like contains(Name, 'text')
	if len(n.Args) != 2 {
		return nil, fmt.Errorf("function %s requires 2 arguments", n.Function)
	}

	var property string
	var value interface{}

	// First argument should be a property name
	if ident, ok := n.Args[0].(*IdentifierExpr); ok {
		property = ident.Name
		if entityMetadata != nil && !propertyExists(property, entityMetadata) {
			return nil, fmt.Errorf("property '%s' does not exist", property)
		}
	} else {
		return nil, fmt.Errorf("first argument of %s must be a property name", n.Function)
	}

	// Second argument should be a literal
	if lit, ok := n.Args[1].(*LiteralExpr); ok {
		value = lit.Value
	} else {
		return nil, fmt.Errorf("second argument of %s must be a literal", n.Function)
	}

	return &FilterExpression{
		Property: property,
		Operator: FilterOperator(n.Function),
		Value:    value,
	}, nil
}
