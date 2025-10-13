package query

import (
	"testing"
)

// TestEntityForLambda is a test entity for lambda tests
type TestEntityForLambda struct {
	ID         int      `json:"ID" odata:"key"`
	Name       string   `json:"Name"`
	Price      float64  `json:"Price"`
	Tags       []string `json:"Tags"`
	Categories []string `json:"Categories"`
}

func TestLambdaTokenizer(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name:      "any with simple condition",
			input:     "Tags/any(t: t eq 'Electronics')",
			expectErr: false,
		},
		{
			name:      "all with simple condition",
			input:     "Tags/all(t: t eq 'Certified')",
			expectErr: false,
		},
		{
			name:      "parameterless any",
			input:     "Tags/any()",
			expectErr: false,
		},
		{
			name:      "any with complex condition",
			input:     "Orders/any(o: o/Total gt 100 and o/Status eq 'Completed')",
			expectErr: false,
		},
		{
			name:      "nested property path",
			input:     "Orders/any(o: o/Items/any(i: i/Price gt 50))",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("Tokenization failed: %v", err)
				}
				return
			}
			if tt.expectErr {
				t.Error("Expected tokenization to fail but it succeeded")
				return
			}

			// Verify we have tokens
			if len(tokens) == 0 {
				t.Error("Expected tokens but got none")
			}
		})
	}
}

func TestLambdaASTParser_BasicAny(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "any with string comparison",
			filter:    "Tags/any(t: t eq 'Electronics')",
			expectErr: false,
		},
		{
			name:      "any with contains",
			filter:    "Tags/any(t: contains(t, 'tech'))",
			expectErr: false,
		},
		{
			name:      "any with property comparison",
			filter:    "Orders/any(o: o/Total gt 100)",
			expectErr: false,
		},
		{
			name:      "parameterless any",
			filter:    "Tags/any()",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("AST parsing failed: %v", err)
				}
				return
			}

			if tt.expectErr {
				t.Error("Expected parsing to fail but it succeeded")
				return
			}

			// Verify the AST is a lambda expression
			lambdaExpr, ok := ast.(*LambdaExpr)
			if !ok {
				t.Errorf("Expected LambdaExpr but got %T", ast)
				return
			}

			if lambdaExpr.Operator != "any" {
				t.Errorf("Expected operator 'any' but got '%s'", lambdaExpr.Operator)
			}
		})
	}
}

func TestLambdaASTParser_BasicAll(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "all with string comparison",
			filter:    "Tags/all(t: contains(t, 'Certified'))",
			expectErr: false,
		},
		{
			name:      "all with property comparison",
			filter:    "Orders/all(o: o/Status eq 'Completed')",
			expectErr: false,
		},
		{
			name:      "parameterless all",
			filter:    "Tags/all()",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("AST parsing failed: %v", err)
				}
				return
			}

			if tt.expectErr {
				t.Error("Expected parsing to fail but it succeeded")
				return
			}

			// Verify the AST is a lambda expression
			lambdaExpr, ok := ast.(*LambdaExpr)
			if !ok {
				t.Errorf("Expected LambdaExpr but got %T", ast)
				return
			}

			if lambdaExpr.Operator != "all" {
				t.Errorf("Expected operator 'all' but got '%s'", lambdaExpr.Operator)
			}
		})
	}
}

func TestLambdaASTParser_ComplexConditions(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "any with AND condition",
			filter:    "Orders/any(o: o/Total gt 100 and o/Status eq 'Completed')",
			expectErr: false,
		},
		{
			name:      "any with OR condition",
			filter:    "Tags/any(t: t eq 'Electronics' or t eq 'Computers')",
			expectErr: false,
		},
		{
			name:      "any with NOT condition",
			filter:    "Orders/any(o: not (o/Status eq 'Cancelled'))",
			expectErr: false,
		},
		{
			name:      "any with complex nested condition",
			filter:    "Orders/any(o: (o/Total gt 100 and o/Status eq 'Completed') or o/Priority eq 'High')",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("AST parsing failed: %v", err)
				}
				return
			}

			if tt.expectErr {
				t.Error("Expected parsing to fail but it succeeded")
				return
			}

			// Verify the AST is a lambda expression
			_, ok := ast.(*LambdaExpr)
			if !ok {
				t.Errorf("Expected LambdaExpr but got %T", ast)
			}
		})
	}
}

func TestLambdaASTParser_NestedLambdas(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "nested any",
			filter:    "Orders/any(o: o/Items/any(i: i/Price gt 50))",
			expectErr: false,
		},
		{
			name:      "nested all",
			filter:    "Orders/all(o: o/Items/all(i: i/Quantity gt 0))",
			expectErr: false,
		},
		{
			name:      "mixed any and all",
			filter:    "Orders/any(o: o/Items/all(i: i/Status eq 'Valid'))",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("AST parsing failed: %v", err)
				}
				return
			}

			if tt.expectErr {
				t.Error("Expected parsing to fail but it succeeded")
				return
			}

			// Verify the AST is a lambda expression
			_, ok := ast.(*LambdaExpr)
			if !ok {
				t.Errorf("Expected LambdaExpr but got %T", ast)
			}
		})
	}
}

func TestLambdaWithOtherOperators(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "lambda with AND",
			filter:    "Tags/any(t: t eq 'Electronics') and Price gt 100",
			expectErr: false,
		},
		{
			name:      "lambda with OR",
			filter:    "Tags/any(t: t eq 'Sale') or Price lt 50",
			expectErr: false,
		},
		{
			name:      "lambda with NOT",
			filter:    "not (Tags/any(t: t eq 'Discontinued'))",
			expectErr: false,
		},
		{
			name:      "multiple lambdas",
			filter:    "Tags/any(t: t eq 'Electronics') and Categories/any(c: c/Name eq 'Computers')",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("AST parsing failed: %v", err)
				}
				return
			}

			if tt.expectErr {
				t.Error("Expected parsing to fail but it succeeded")
				return
			}

			// Verify we have a valid AST
			if ast == nil {
				t.Error("Expected AST but got nil")
			}
		})
	}
}

func TestLambdaToFilterExpression(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "simple any",
			filter:    "Tags/any(t: t eq 'Electronics')",
			expectErr: false,
		},
		{
			name:      "simple all",
			filter:    "Tags/all(t: contains(t, 'Certified'))",
			expectErr: false,
		},
		{
			name:      "parameterless any",
			filter:    "Tags/any()",
			expectErr: false,
		},
		{
			name:      "any with complex condition",
			filter:    "Orders/any(o: o/Total gt 100 and o/Status eq 'Completed')",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("AST parsing failed: %v", err)
				}
				return
			}

			// Convert to FilterExpression - note we pass nil for metadata
			// because lambda is a new feature and may not have full metadata support yet
			filterExpr, err := ASTToFilterExpression(ast, nil)
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("FilterExpression conversion failed: %v", err)
				return
			}

			// Verify the operator
			if filterExpr.Operator != OpAny && filterExpr.Operator != OpAll {
				t.Errorf("Expected lambda operator (any/all) but got %v", filterExpr.Operator)
			}
		})
	}
}

func TestLambdaEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "missing colon should fail",
			filter:    "Tags/any(t t eq 'test')",
			expectErr: true,
		},
		{
			name:      "missing closing paren should fail",
			filter:    "Tags/any(t: t eq 'test'",
			expectErr: true,
		},
		{
			name:      "empty predicate with colon should fail",
			filter:    "Tags/any(t:)",
			expectErr: true,
		},
		{
			name:      "any without collection should work as identifier",
			filter:    "any eq true",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Errorf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			_, err = parser.Parse()
			if tt.expectErr {
				if err == nil {
					t.Error("Expected parsing to fail but it succeeded")
				}
				return
			}
			if err != nil {
				t.Errorf("Parsing failed unexpectedly: %v", err)
			}
		})
	}
}
