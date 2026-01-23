package query

import "fmt"

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
		expr := AcquireBinaryExpr()
		expr.Left = left
		expr.Operator = op.Value
		expr.Right = right
		left = expr
	}

	return left, nil
}
