package query

import (
	"fmt"
	"strconv"
	"strings"
)

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

// parseLiteral parses literal values (string, number, boolean, null, date, time)
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
	case TokenDate:
		p.advance()
		return &LiteralExpr{Value: token.Value, Type: "date"}
	case TokenTime:
		p.advance()
		return &LiteralExpr{Value: token.Value, Type: "time"}
	case TokenDateTime:
		p.advance()
		return &LiteralExpr{Value: token.Value, Type: "datetime"}
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

	// Check for geospatial literals: geography'...' or geometry'...'
	lowerIdent := strings.ToLower(token.Value)
	if (lowerIdent == "geography" || lowerIdent == "geometry") && p.currentToken().Type == TokenString {
		geoType := lowerIdent
		geoValue := p.currentToken().Value
		p.advance()
		return &LiteralExpr{Value: geoValue, Type: geoType}, nil
	}

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
