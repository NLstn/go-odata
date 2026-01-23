package query

import "fmt"

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
		expr := AcquireBinaryExpr()
		expr.Left = left
		expr.Operator = op.Value
		expr.Right = right
		left = expr
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
		expr := AcquireUnaryExpr()
		expr.Operator = op.Value
		expr.Operand = operand
		return expr, nil
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
			expr := AcquireComparisonExpr()
			expr.Left = left
			expr.Operator = op.Value
			expr.Right = right
			return expr, nil
		}

		right, err := p.parseArithmetic()
		if err != nil {
			return nil, err
		}
		expr := AcquireComparisonExpr()
		expr.Left = left
		expr.Operator = op.Value
		expr.Right = right
		return expr, nil
	}

	return left, nil
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

	expr := AcquireCollectionExpr()
	expr.Values = values
	return expr, nil
}
