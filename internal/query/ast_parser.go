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

// peek returns the next token without advancing
func (p *ASTParser) peek() *Token {
	if p.current+1 >= len(p.tokens) {
		return &Token{Type: TokenEOF}
	}
	return p.tokens[p.current+1]
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
func (p *ASTParser) expect(tokenType TokenType) (*Token, error) {
	token := p.currentToken()
	if token.Type != tokenType {
		return nil, fmt.Errorf("expected token type %v, got %v at position %d", tokenType, token.Type, token.Pos)
	}
	p.advance()
	return token, nil
}

// Parse parses the tokens into an AST
func (p *ASTParser) Parse() (ASTNode, error) {
	return p.parseOr()
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
		p.advance() // consume '('
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return &GroupExpr{Expr: expr}, nil
	}

	// Literals
	if token.Type == TokenString {
		p.advance()
		return &LiteralExpr{Value: token.Value, Type: "string"}, nil
	}

	if token.Type == TokenNumber {
		p.advance()
		// Try to parse as integer first, then as float
		if intVal, err := strconv.ParseInt(token.Value, 10, 64); err == nil {
			return &LiteralExpr{Value: intVal, Type: "number"}, nil
		}
		if floatVal, err := strconv.ParseFloat(token.Value, 64); err == nil {
			return &LiteralExpr{Value: floatVal, Type: "number"}, nil
		}
		return &LiteralExpr{Value: token.Value, Type: "number"}, nil
	}

	if token.Type == TokenBoolean {
		p.advance()
		boolVal := token.Value == "true"
		return &LiteralExpr{Value: boolVal, Type: "boolean"}, nil
	}

	if token.Type == TokenNull {
		p.advance()
		return &LiteralExpr{Value: nil, Type: "null"}, nil
	}

	// Identifier or function call
	if token.Type == TokenIdentifier {
		p.advance()
		
		// Check if this is a function call
		if p.currentToken().Type == TokenLParen {
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
			
			if _, err := p.expect(TokenRParen); err != nil {
				return nil, err
			}
			
			return &FunctionCallExpr{
				Function: token.Value,
				Args:     args,
			}, nil
		}
		
		return &IdentifierExpr{Name: token.Value}, nil
	}

	return nil, fmt.Errorf("unexpected token %v at position %d", token.Type, token.Pos)
}

// ASTToFilterExpression converts an AST to a FilterExpression
func ASTToFilterExpression(node ASTNode, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	switch n := node.(type) {
	case *BinaryExpr:
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

	case *UnaryExpr:
		if n.Operator == "not" {
			operand, err := ASTToFilterExpression(n.Operand, entityMetadata)
			if err != nil {
				return nil, err
			}
			// Represent NOT as a special filter expression
			operand.IsNot = true
			return operand, nil
		}

	case *ComparisonExpr:
		// Extract property and value
		var property string
		var value interface{}

		// Left side should be an identifier (property name)
		if ident, ok := n.Left.(*IdentifierExpr); ok {
			property = ident.Name
			// Validate property exists
			if entityMetadata != nil && !propertyExists(property, entityMetadata) {
				return nil, fmt.Errorf("property '%s' does not exist", property)
			}
		} else {
			return nil, fmt.Errorf("left side of comparison must be a property name")
		}

		// Right side should be a literal
		if lit, ok := n.Right.(*LiteralExpr); ok {
			value = lit.Value
		} else if ident, ok := n.Right.(*IdentifierExpr); ok {
			// Allow identifiers as values (for property-to-property comparisons in the future)
			value = ident.Name
		} else {
			return nil, fmt.Errorf("right side of comparison must be a literal or property")
		}

		return &FilterExpression{
			Property: property,
			Operator: FilterOperator(n.Operator),
			Value:    value,
		}, nil

	case *FunctionCallExpr:
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
