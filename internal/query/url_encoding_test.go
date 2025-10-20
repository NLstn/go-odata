package query

import (
	"testing"
)

// TestURLEncodingEscapedQuotes tests handling of URL-encoded escaped quotes
// This is related to OData v4 compliance test 11.2.14
func TestURLEncodingEscapedQuotes(t *testing.T) {
	tests := []struct {
		name          string
		filter        string
		expectedError bool
	}{
		{
			name:          "Escaped single quote in contains function",
			filter:        "contains(Name,'''')",
			expectedError: false,
		},
		{
			name:          "String with apostrophe",
			filter:        "Name eq 'O''Neil'",
			expectedError: false,
		},
		{
			name:          "Empty string",
			filter:        "Name eq ''",
			expectedError: false,
		},
		{
			name:          "Multiple escaped quotes",
			filter:        "contains(Description,'It''s a ''test''')",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing filter: %s", tt.filter)

			// Tokenize
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectedError {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			t.Logf("Tokens:")
			for i, tok := range tokens {
				t.Logf("  %d: Type=%v, Value='%s', Pos=%d", i, tok.Type, tok.Value, tok.Pos)
			}

			// Parse
			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectedError {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			if tt.expectedError {
				t.Fatal("Expected error but got none")
			}

			t.Logf("AST parsed successfully: %T", ast)
		})
	}
}
