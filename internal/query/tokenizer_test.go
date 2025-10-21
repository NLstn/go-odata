package query

import (
	"testing"
)

func TestTokenizer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:  "Simple comparison",
			input: "Price gt 100",
			expected: []TokenType{
				TokenIdentifier,
				TokenOperator,
				TokenNumber,
				TokenEOF,
			},
		},
		{
			name:  "With parentheses",
			input: "(Price gt 100)",
			expected: []TokenType{
				TokenLParen,
				TokenIdentifier,
				TokenOperator,
				TokenNumber,
				TokenRParen,
				TokenEOF,
			},
		},
		{
			name:  "Logical AND",
			input: "Price gt 100 and Category eq 'Electronics'",
			expected: []TokenType{
				TokenIdentifier,
				TokenOperator,
				TokenNumber,
				TokenLogical,
				TokenIdentifier,
				TokenOperator,
				TokenString,
				TokenEOF,
			},
		},
		{
			name:  "NOT operator",
			input: "not (Price gt 100)",
			expected: []TokenType{
				TokenNot,
				TokenLParen,
				TokenIdentifier,
				TokenOperator,
				TokenNumber,
				TokenRParen,
				TokenEOF,
			},
		},
		{
			name:  "Function call",
			input: "contains(Name,'Laptop')",
			expected: []TokenType{
				TokenIdentifier, // contains is an identifier (function name)
				TokenLParen,
				TokenIdentifier,
				TokenComma,
				TokenString,
				TokenRParen,
				TokenEOF,
			},
		},
		{
			name:  "Arithmetic expression",
			input: "Price + Tax",
			expected: []TokenType{
				TokenIdentifier,
				TokenArithmetic,
				TokenIdentifier,
				TokenEOF,
			},
		},
		{
			name:  "Complex arithmetic",
			input: "(Price * Quantity) + Tax",
			expected: []TokenType{
				TokenLParen,
				TokenIdentifier,
				TokenArithmetic,
				TokenIdentifier,
				TokenRParen,
				TokenArithmetic,
				TokenIdentifier,
				TokenEOF,
			},
		},
		{
			name:  "Boolean literal",
			input: "IsActive eq true",
			expected: []TokenType{
				TokenIdentifier,
				TokenOperator,
				TokenBoolean,
				TokenEOF,
			},
		},
		{
			name:  "Null literal",
			input: "Description eq null",
			expected: []TokenType{
				TokenIdentifier,
				TokenOperator,
				TokenNull,
				TokenEOF,
			},
		},
		{
			name:  "Modulo operator",
			input: "Quantity mod 2 eq 0",
			expected: []TokenType{
				TokenIdentifier,
				TokenArithmetic,
				TokenNumber,
				TokenOperator,
				TokenNumber,
				TokenEOF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
				for i, token := range tokens {
					t.Logf("Token %d: %v (%s)", i, token.Type, token.Value)
				}
				return
			}

			for i, expected := range tt.expected {
				if tokens[i].Type != expected {
					t.Errorf("Token %d: expected type %v, got %v (value: %s)",
						i, expected, tokens[i].Type, tokens[i].Value)
				}
			}
		})
	}
}

func TestTokenizerStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single quotes",
			input:    "'Hello World'",
			expected: "Hello World",
		},
		{
			name:     "Double quotes",
			input:    "\"Hello World\"",
			expected: "Hello World",
		},
		{
			name:     "Escaped quotes (OData spec - doubled quotes)",
			input:    "'It''s working'",
			expected: "It's working",
		},
		{
			name:     "Name with apostrophe",
			input:    "'O''Neil'",
			expected: "O'Neil",
		},
		{
			name:     "Just an escaped quote",
			input:    "''''",
			expected: "'",
		},
		{
			name:     "Multiple escaped quotes",
			input:    "'She said ''Hello'' and ''Goodbye'''",
			expected: "She said 'Hello' and 'Goodbye'",
		},
		{
			name:     "Empty string",
			input:    "''",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			token, err := tokenizer.NextToken()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			if token.Type != TokenString {
				t.Errorf("Expected TokenString, got %v", token.Type)
			}

			if token.Value != tt.expected {
				t.Errorf("Expected value '%s', got '%s'", tt.expected, token.Value)
			}
		})
	}
}

func TestTokenizerNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Integer",
			input:    "42",
			expected: "42",
		},
		{
			name:     "Decimal",
			input:    "3.14",
			expected: "3.14",
		},
		{
			name:     "Negative",
			input:    "-10",
			expected: "-10",
		},
		{
			name:     "Scientific notation",
			input:    "1.5e10",
			expected: "1.5e10",
		},
		{
			name:     "Scientific with negative exponent",
			input:    "2.5e-3",
			expected: "2.5e-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			token, err := tokenizer.NextToken()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			if token.Type != TokenNumber {
				t.Errorf("Expected TokenNumber, got %v", token.Type)
			}

			if token.Value != tt.expected {
				t.Errorf("Expected value '%s', got '%s'", tt.expected, token.Value)
			}
		})
	}
}

func TestTokenizerDateLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Date literal",
			input:    "2024-01-15",
			expected: "2024-01-15",
		},
		{
			name:     "Date in expression",
			input:    "date(CreatedAt) eq 2024-01-15",
			expected: "2024-01-15",
		},
		{
			name:     "Different date",
			input:    "2023-12-31",
			expected: "2023-12-31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			// Find the date token
			var found bool
			for _, token := range tokens {
				if token.Type == TokenDate {
					found = true
					if token.Value != tt.expected {
						t.Errorf("Expected date value '%s', got '%s'", tt.expected, token.Value)
					}
					break
				}
			}

			if !found {
				t.Errorf("Expected to find a TokenDate, but didn't")
			}
		})
	}
}

func TestTokenizerTimeLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Time literal",
			input:    "12:00:00",
			expected: "12:00:00",
		},
		{
			name:     "Time in expression",
			input:    "time(CreatedAt) lt 12:00:00",
			expected: "12:00:00",
		},
		{
			name:     "Time with fractional seconds",
			input:    "14:30:45.123",
			expected: "14:30:45.123",
		},
		{
			name:     "Morning time",
			input:    "08:15:30",
			expected: "08:15:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			// Find the time token
			var found bool
			for _, token := range tokens {
				if token.Type == TokenTime {
					found = true
					if token.Value != tt.expected {
						t.Errorf("Expected time value '%s', got '%s'", tt.expected, token.Value)
					}
					break
				}
			}

			if !found {
				t.Errorf("Expected to find a TokenTime, but didn't")
			}
		})
	}
}

func TestTokenizerQualifiedTypeNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Qualified type name Edm.String",
			input:    "Edm.String",
			expected: "Edm.String",
		},
		{
			name:     "Qualified type name Edm.Decimal",
			input:    "Edm.Decimal",
			expected: "Edm.Decimal",
		},
		{
			name:     "isof with qualified type",
			input:    "isof(Price,Edm.Decimal)",
			expected: "Edm.Decimal",
		},
		{
			name:     "cast with qualified type",
			input:    "cast(Status,Edm.String)",
			expected: "Edm.String",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			// Find the identifier with dot
			var found bool
			for _, token := range tokens {
				if token.Type == TokenIdentifier && token.Value == tt.expected {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected to find identifier '%s', but didn't. Tokens: %+v", tt.expected, tokens)
			}
		})
	}
}

