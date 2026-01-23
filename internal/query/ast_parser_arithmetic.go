package query

import (
	"fmt"
	"math"
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
		expr := AcquireBinaryExpr()
		expr.Left = left
		expr.Operator = op.Value
		expr.Right = right
		left = expr
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
		expr := AcquireBinaryExpr()
		expr.Left = left
		expr.Operator = op.Value
		expr.Right = right
		left = expr
	}

	return left, nil
}

// parsePrimary handles primary expressions (literals, identifiers, function calls, grouped expressions)
func (p *ASTParser) parsePrimary() (ASTNode, error) {
	token := p.currentToken()

	// Handle -INF and -NaN (OData v4 spec special float literals)
	if token.Type == TokenArithmetic && token.Value == "-" {
		// Peek at next token to see if it's INF or NaN
		nextPos := p.current + 1
		if nextPos < len(p.tokens) {
			nextToken := p.tokens[nextPos]
			if nextToken.Type == TokenNumber {
				lowerValue := strings.ToLower(nextToken.Value)
				if lowerValue == "inf" || lowerValue == "nan" {
					// Consume the minus sign
					p.advance()
					// Consume INF/NaN and return negative value
					p.advance()
					if lowerValue == "inf" {
						expr := AcquireLiteralExpr()
						expr.Value = math.Inf(-1)
						expr.Type = "number"
						return expr, nil // -INF
					} else {
						// Note: -NaN is technically the same as NaN, but we handle it for completeness
						expr := AcquireLiteralExpr()
						expr.Value = math.NaN()
						expr.Type = "number"
						return expr, nil // NaN
					}
				}
			}
		}
	}

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
	groupExpr := AcquireGroupExpr()
	groupExpr.Expr = expr
	return groupExpr, nil
}

// parseLiteral parses literal values (string, number, boolean, null, date, time, GUID)
func (p *ASTParser) parseLiteral(token *Token) ASTNode {
	switch token.Type {
	case TokenString:
		p.advance()
		expr := AcquireLiteralExpr()
		expr.Value = token.Value
		expr.Type = "string"
		return expr
	case TokenNumber:
		p.advance()
		return p.parseNumberLiteral(token.Value)
	case TokenBoolean:
		p.advance()
		boolVal := token.Value == "true"
		expr := AcquireLiteralExpr()
		expr.Value = boolVal
		expr.Type = "boolean"
		return expr
	case TokenNull:
		p.advance()
		expr := AcquireLiteralExpr()
		expr.Value = nil
		expr.Type = "null"
		return expr
	case TokenDate:
		p.advance()
		expr := AcquireLiteralExpr()
		expr.Value = token.Value
		expr.Type = "date"
		return expr
	case TokenTime:
		p.advance()
		expr := AcquireLiteralExpr()
		expr.Value = token.Value
		expr.Type = "time"
		return expr
	case TokenDateTime:
		p.advance()
		expr := AcquireLiteralExpr()
		expr.Value = token.Value
		expr.Type = "datetime"
		return expr
	case TokenGUID:
		p.advance()
		expr := AcquireLiteralExpr()
		expr.Value = token.Value
		expr.Type = "guid"
		return expr
	default:
		return nil
	}
}

// parseNumberLiteral parses a number literal (integer or float)
func (p *ASTParser) parseNumberLiteral(value string) ASTNode {
	// Check for special floating-point literals (OData v4 spec)
	lowerValue := strings.ToLower(value)
	switch lowerValue {
	case "inf":
		expr := AcquireLiteralExpr()
		expr.Value = math.Inf(1)
		expr.Type = "number"
		return expr
	case "-inf":
		expr := AcquireLiteralExpr()
		expr.Value = math.Inf(-1)
		expr.Type = "number"
		return expr
	case "nan":
		expr := AcquireLiteralExpr()
		expr.Value = math.NaN()
		expr.Type = "number"
		return expr
	}

	// Try to parse as integer first, then as float
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		expr := AcquireLiteralExpr()
		expr.Value = intVal
		expr.Type = "number"
		return expr
	}
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		expr := AcquireLiteralExpr()
		expr.Value = floatVal
		expr.Type = "number"
		return expr
	}
	expr := AcquireLiteralExpr()
	expr.Value = value
	expr.Type = "number"
	return expr
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
		expr := AcquireLiteralExpr()
		expr.Value = geoValue
		expr.Type = geoType
		return expr, nil
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

	identExpr := AcquireIdentifierExpr()
	identExpr.Name = token.Value
	return identExpr, nil
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

	identExpr := AcquireIdentifierExpr()
	identExpr.Name = path
	return identExpr, nil
}
