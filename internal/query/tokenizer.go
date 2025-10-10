package query

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdentifier
	TokenString
	TokenNumber
	TokenBoolean
	TokenNull
	TokenOperator
	TokenLogical
	TokenNot
	TokenLParen
	TokenRParen
	TokenComma
	TokenArithmetic
)

// Token represents a single token in the filter expression
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Tokenizer tokenizes OData filter expressions
type Tokenizer struct {
	input string
	pos   int
	ch    rune
}

// NewTokenizer creates a new tokenizer
func NewTokenizer(input string) *Tokenizer {
	t := &Tokenizer{
		input: input,
		pos:   0,
	}
	if len(input) > 0 {
		t.ch = rune(input[0])
	}
	return t
}

// advance moves to the next character
func (t *Tokenizer) advance() {
	t.pos++
	if t.pos >= len(t.input) {
		t.ch = 0 // EOF
	} else {
		t.ch = rune(t.input[t.pos])
	}
}

// peek looks ahead without advancing
func (t *Tokenizer) peek() rune {
	if t.pos+1 >= len(t.input) {
		return 0
	}
	return rune(t.input[t.pos+1])
}

// skipWhitespace skips whitespace characters
func (t *Tokenizer) skipWhitespace() {
	for t.ch == ' ' || t.ch == '\t' || t.ch == '\n' || t.ch == '\r' {
		t.advance()
	}
}

// readString reads a quoted string
func (t *Tokenizer) readString() string {
	quote := t.ch
	t.advance() // skip opening quote

	var result strings.Builder
	for t.ch != 0 && t.ch != quote {
		if t.ch == '\\' && t.peek() == quote {
			t.advance()
			result.WriteRune(t.ch)
		} else {
			result.WriteRune(t.ch)
		}
		t.advance()
	}

	if t.ch == quote {
		t.advance() // skip closing quote
	}

	return result.String()
}

// readNumber reads a number
func (t *Tokenizer) readNumber() string {
	var result strings.Builder

	// Handle negative numbers
	if t.ch == '-' {
		result.WriteRune(t.ch)
		t.advance()
	}

	// Read integer part
	for unicode.IsDigit(t.ch) {
		result.WriteRune(t.ch)
		t.advance()
	}

	// Read decimal part
	if t.ch == '.' {
		result.WriteRune(t.ch)
		t.advance()
		for unicode.IsDigit(t.ch) {
			result.WriteRune(t.ch)
			t.advance()
		}
	}

	// Read exponent part
	if t.ch == 'e' || t.ch == 'E' {
		result.WriteRune(t.ch)
		t.advance()
		if t.ch == '+' || t.ch == '-' {
			result.WriteRune(t.ch)
			t.advance()
		}
		for unicode.IsDigit(t.ch) {
			result.WriteRune(t.ch)
			t.advance()
		}
	}

	return result.String()
}

// readIdentifier reads an identifier or keyword
func (t *Tokenizer) readIdentifier() string {
	var result strings.Builder

	for t.ch != 0 && (unicode.IsLetter(t.ch) || unicode.IsDigit(t.ch) || t.ch == '_') {
		result.WriteRune(t.ch)
		t.advance()
	}

	return result.String()
}

// NextToken returns the next token
func (t *Tokenizer) NextToken() (*Token, error) {
	t.skipWhitespace()

	if t.ch == 0 {
		return &Token{Type: TokenEOF, Pos: t.pos}, nil
	}

	pos := t.pos

	// String literals
	if t.ch == '\'' || t.ch == '"' {
		value := t.readString()
		return &Token{Type: TokenString, Value: value, Pos: pos}, nil
	}

	// Numbers
	if unicode.IsDigit(t.ch) || (t.ch == '-' && unicode.IsDigit(t.peek())) {
		value := t.readNumber()
		return &Token{Type: TokenNumber, Value: value, Pos: pos}, nil
	}

	// Parentheses
	if t.ch == '(' {
		t.advance()
		return &Token{Type: TokenLParen, Value: "(", Pos: pos}, nil
	}
	if t.ch == ')' {
		t.advance()
		return &Token{Type: TokenRParen, Value: ")", Pos: pos}, nil
	}

	// Comma
	if t.ch == ',' {
		t.advance()
		return &Token{Type: TokenComma, Value: ",", Pos: pos}, nil
	}

	// Arithmetic operators
	if t.ch == '+' || t.ch == '-' || t.ch == '*' || t.ch == '/' {
		op := string(t.ch)
		t.advance()
		return &Token{Type: TokenArithmetic, Value: op, Pos: pos}, nil
	}

	// Identifiers and keywords
	if unicode.IsLetter(t.ch) || t.ch == '_' {
		value := t.readIdentifier()
		lower := strings.ToLower(value)

		// Check for keywords
		switch lower {
		case "and":
			return &Token{Type: TokenLogical, Value: "and", Pos: pos}, nil
		case "or":
			return &Token{Type: TokenLogical, Value: "or", Pos: pos}, nil
		case "not":
			return &Token{Type: TokenNot, Value: "not", Pos: pos}, nil
		case "true", "false":
			return &Token{Type: TokenBoolean, Value: lower, Pos: pos}, nil
		case "null":
			return &Token{Type: TokenNull, Value: "null", Pos: pos}, nil
		case "eq", "ne", "gt", "ge", "lt", "le":
			return &Token{Type: TokenOperator, Value: lower, Pos: pos}, nil
		case "mod":
			return &Token{Type: TokenArithmetic, Value: "mod", Pos: pos}, nil
		default:
			// Functions like contains, startswith, endswith are identifiers
			// that will be recognized as function calls when followed by '('
			return &Token{Type: TokenIdentifier, Value: value, Pos: pos}, nil
		}
	}

	return nil, fmt.Errorf("unexpected character '%c' at position %d", t.ch, t.pos)
}

// TokenizeAll returns all tokens from the input
func (t *Tokenizer) TokenizeAll() ([]*Token, error) {
	var tokens []*Token

	for {
		token, err := t.NextToken()
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, token)

		if token.Type == TokenEOF {
			break
		}
	}

	return tokens, nil
}
