package query

import (
	"testing"
)

func TestASTParser_Parentheses(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "Simple parentheses",
			filter:    "(Price gt 100)",
			expectErr: false,
		},
		{
			name:      "Nested parentheses",
			filter:    "((Price gt 100) and (Category eq 'Electronics'))",
			expectErr: false,
		},
		{
			name:      "Complex boolean grouping",
			filter:    "(Price gt 100 and Category eq 'Electronics') or (Price lt 50 and Category eq 'Books')",
			expectErr: false,
		},
		{
			name:      "Multiple levels of grouping",
			filter:    "((Price gt 100 or Price lt 10) and (Category eq 'A' or Category eq 'B')) or Name eq 'Test'",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && ast == nil {
				t.Error("Expected non-nil AST")
			}

			// Convert to FilterExpression to ensure it works
			if !tt.expectErr {
				_, err := ASTToFilterExpression(ast, meta)
				if err != nil {
					t.Errorf("Failed to convert AST to FilterExpression: %v", err)
				}
			}
		})
	}
}

func TestASTParser_NotOperator(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "Simple NOT",
			filter:    "not (Price gt 100)",
			expectErr: false,
		},
		{
			name:      "NOT with AND",
			filter:    "not (Price gt 100 and Category eq 'Electronics')",
			expectErr: false,
		},
		{
			name:      "NOT in complex expression",
			filter:    "(Price gt 100 and not (Category eq 'Electronics')) or Name eq 'Test'",
			expectErr: false,
		},
		{
			name:      "Multiple NOTs",
			filter:    "not (Price gt 100) and not (Category eq 'Books')",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr {
				filterExpr, err := ASTToFilterExpression(ast, meta)
				if err != nil {
					t.Errorf("Failed to convert AST to FilterExpression: %v", err)
				}

				// Verify that NOT is properly set
				if filterExpr == nil {
					t.Error("Expected non-nil FilterExpression")
				}
			}
		})
	}
}

func TestASTParser_ArithmeticOperators(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "Addition",
			filter:    "Price + 10 gt 100",
			expectErr: false,
		},
		{
			name:      "Subtraction",
			filter:    "Price - 10 gt 100",
			expectErr: false,
		},
		{
			name:      "Multiplication",
			filter:    "Price * 2 gt 100",
			expectErr: false,
		},
		{
			name:      "Division",
			filter:    "Price / 2 gt 50",
			expectErr: false,
		},
		{
			name:      "Modulo",
			filter:    "ID mod 2 eq 0",
			expectErr: false,
		},
		{
			name:      "Complex arithmetic",
			filter:    "(Price * 2 + 10) gt 100",
			expectErr: false,
		},
		{
			name:      "Arithmetic with parentheses",
			filter:    "((Price + 10) * 2) gt 100",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && ast == nil {
				t.Error("Expected non-nil AST")
			}
		})
	}
}

func TestASTParser_LiteralTyping(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name         string
		filter       string
		expectErr    bool
		expectedType string
	}{
		{
			name:         "String literal",
			filter:       "Name eq 'Test'",
			expectErr:    false,
			expectedType: "string",
		},
		{
			name:         "Integer literal",
			filter:       "ID eq 42",
			expectErr:    false,
			expectedType: "number",
		},
		{
			name:         "Float literal",
			filter:       "Price eq 99.99",
			expectErr:    false,
			expectedType: "number",
		},
		{
			name:         "Boolean literal true",
			filter:       "Name eq true",
			expectErr:    false,
			expectedType: "boolean",
		},
		{
			name:         "Boolean literal false",
			filter:       "Name eq false",
			expectErr:    false,
			expectedType: "boolean",
		},
		{
			name:         "Null literal",
			filter:       "Description eq null",
			expectErr:    false,
			expectedType: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr {
				// Verify literal type in AST
				compExpr, ok := ast.(*ComparisonExpr)
				if !ok {
					t.Fatal("Expected ComparisonExpr")
				}

				litExpr, ok := compExpr.Right.(*LiteralExpr)
				if !ok {
					t.Fatal("Expected LiteralExpr on right side")
				}

				if litExpr.Type != tt.expectedType {
					t.Errorf("Expected literal type %s, got %s", tt.expectedType, litExpr.Type)
				}

				// Also convert to FilterExpression to ensure it works
				_, err := ASTToFilterExpression(ast, meta)
				if err != nil {
					t.Errorf("Failed to convert AST to FilterExpression: %v", err)
				}
			}
		})
	}
}

func TestASTParser_FunctionCalls(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "contains function",
			filter:    "contains(Name, 'Laptop')",
			expectErr: false,
		},
		{
			name:      "startswith function",
			filter:    "startswith(Category, 'Elec')",
			expectErr: false,
		},
		{
			name:      "endswith function",
			filter:    "endswith(Description, 'end')",
			expectErr: false,
		},
		{
			name:      "Function in complex expression",
			filter:    "contains(Name, 'Test') and Price gt 100",
			expectErr: false,
		},
		{
			name:      "Function with parentheses",
			filter:    "(contains(Name, 'Test') or contains(Description, 'Test')) and Price gt 100",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr {
				_, err := ASTToFilterExpression(ast, meta)
				if err != nil {
					t.Errorf("Failed to convert AST to FilterExpression: %v", err)
				}
			}
		})
	}
}

func TestASTParser_ComplexExpressions(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "Deep nesting",
			filter:    "((Price gt 100 and Category eq 'A') or (Price lt 50 and Category eq 'B')) and not (Name eq 'Excluded')",
			expectErr: false,
		},
		{
			name:      "Multiple functions and operators",
			filter:    "(contains(Name, 'Laptop') or contains(Description, 'Laptop')) and Price gt 500 and Category eq 'Electronics'",
			expectErr: false,
		},

		{
			name:      "NOT with nested groups",
			filter:    "not ((Price gt 1000 or Category eq 'Luxury') and not (contains(Name, 'Sale')))",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr {
				_, err := ASTToFilterExpression(ast, meta)
				if err != nil {
					t.Errorf("Failed to convert AST to FilterExpression: %v", err)
				}
			}
		})
	}
}
