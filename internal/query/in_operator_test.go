package query

import (
	"testing"
)

func TestInOperator_Basic(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "in operator with string values",
			filter:    "Category in ('Electronics', 'Books', 'Toys')",
			expectErr: false,
		},
		{
			name:      "in operator with numeric values",
			filter:    "ID in (1, 2, 3, 4, 5)",
			expectErr: false,
		},
		{
			name:      "in operator with single value",
			filter:    "Category in ('Electronics')",
			expectErr: false,
		},
		{
			name:      "in operator with empty collection - should parse",
			filter:    "Category in ()",
			expectErr: false,
		},
		{
			name:      "in operator with float values",
			filter:    "Price in (10.5, 20.0, 30.99)",
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

			filterExpr, err := ASTToFilterExpression(ast, meta)
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

			if filterExpr.Operator != OpIn {
				t.Errorf("Expected operator OpIn, got %v", filterExpr.Operator)
			}

			// Verify the value is a slice
			_, ok := filterExpr.Value.([]interface{})
			if !ok {
				t.Errorf("Expected value to be []interface{}, got %T", filterExpr.Value)
			}
		})
	}
}

func TestInOperator_WithLogicalOperators(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "in operator with AND",
			filter:    "Category in ('Electronics', 'Books') and Price gt 10",
			expectErr: false,
		},
		{
			name:      "in operator with OR",
			filter:    "Category in ('Electronics', 'Books') or Category eq 'Toys'",
			expectErr: false,
		},
		{
			name:      "multiple in operators with AND",
			filter:    "Category in ('Electronics', 'Books') and ID in (1, 2, 3)",
			expectErr: false,
		},
		{
			name:      "in operator with NOT",
			filter:    "not Category in ('Electronics', 'Books')",
			expectErr: false,
		},
		{
			name:      "complex expression with in operator",
			filter:    "Category in ('Electronics', 'Books') and Price gt 10 and ID in (1, 2, 3)",
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

			_, err = ASTToFilterExpression(ast, meta)
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
		})
	}
}

func TestInOperator_SQLGeneration(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name           string
		filter         string
		expectedSQL    string
		expectedArgs   int
		expectErr      bool
	}{
		{
			name:         "in operator with string values",
			filter:       "Category in ('Electronics', 'Books')",
			expectedSQL:  "category IN (?, ?)",
			expectedArgs: 2,
			expectErr:    false,
		},
		{
			name:         "in operator with numeric values",
			filter:       "ID in (1, 2, 3)",
			expectedSQL:  "id IN (?, ?, ?)",
			expectedArgs: 3,
			expectErr:    false,
		},
		{
			name:         "in operator with single value",
			filter:       "Category in ('Electronics')",
			expectedSQL:  "category IN (?)",
			expectedArgs: 1,
			expectErr:    false,
		},
		{
			name:         "in operator with empty collection",
			filter:       "Category in ()",
			expectedSQL:  "1 = 0",
			expectedArgs: 0,
			expectErr:    false,
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

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if err != nil {
				if !tt.expectErr {
					t.Errorf("FilterExpression conversion failed: %v", err)
				}
				return
			}

			sql, args := buildFilterCondition(filterExpr, meta)
			if tt.expectErr {
				if sql != "" {
					t.Error("Expected error but got SQL")
				}
				return
			}

			if sql != tt.expectedSQL {
				t.Errorf("Expected SQL '%s', got '%s'", tt.expectedSQL, sql)
			}

			if len(args) != tt.expectedArgs {
				t.Errorf("Expected %d args, got %d", tt.expectedArgs, len(args))
			}
		})
	}
}

func TestInOperator_ValueValidation(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "valid string values",
			filter:    "Category in ('Electronics', 'Books')",
			expectErr: false,
		},
		{
			name:      "valid numeric values",
			filter:    "ID in (1, 2, 3)",
			expectErr: false,
		},
		{
			name:      "mixed numeric types",
			filter:    "Price in (10, 20.5, 30)",
			expectErr: false,
		},
		{
			name:      "single quoted strings",
			filter:    "Name in ('John', 'Jane', 'Bob')",
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

			filterExpr, err := ASTToFilterExpression(ast, meta)
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

			// Verify value is a slice
			values, ok := filterExpr.Value.([]interface{})
			if !ok {
				t.Errorf("Expected []interface{}, got %T", filterExpr.Value)
				return
			}

			// Verify at least one value
			if len(values) == 0 && tt.filter != "Category in ()" {
				t.Error("Expected at least one value in collection")
			}
		})
	}
}

func TestInOperator_Integration(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "in operator in complex filter",
			filter:    "(Category in ('Electronics', 'Books') and Price gt 10) or (ID in (1, 2) and Name eq 'Test')",
			expectErr: false,
		},
		{
			name:      "in operator with parentheses",
			filter:    "(Category in ('Electronics', 'Books'))",
			expectErr: false,
		},
		{
			name:      "in operator after other operators",
			filter:    "Price gt 10 and Category in ('Electronics', 'Books')",
			expectErr: false,
		},
		{
			name:      "nested parentheses with in",
			filter:    "((Category in ('Electronics', 'Books')) and Price gt 10)",
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

			filterExpr, err := ASTToFilterExpression(ast, meta)
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

			// Also verify SQL generation works
			sql, _ := buildFilterCondition(filterExpr, meta)
			if sql == "" {
				t.Error("Failed to generate SQL for filter expression")
			}
		})
	}
}
