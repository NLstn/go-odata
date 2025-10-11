package query

import (
	"testing"
)

func TestArithmeticFunctions_Add(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "add simple",
			filter:    "add(Price, 10) gt 100",
			expectErr: false,
		},
		{
			name:      "add with comparison",
			filter:    "add(ID, 5) eq 10",
			expectErr: false,
		},
		{
			name:      "add in complex expression",
			filter:    "add(Price, 50) gt 100 and Category eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "add wrong argument count",
			filter:    "add(Price) eq 100",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestArithmeticFunctions_Sub(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "sub simple",
			filter:    "sub(Price, 10) gt 50",
			expectErr: false,
		},
		{
			name:      "sub with comparison",
			filter:    "sub(ID, 2) eq 8",
			expectErr: false,
		},
		{
			name:      "sub in complex expression",
			filter:    "sub(Price, 25) lt 100 or Category eq 'Books'",
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
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestArithmeticFunctions_Mul(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "mul simple",
			filter:    "mul(Price, 2) gt 100",
			expectErr: false,
		},
		{
			name:      "mul with comparison",
			filter:    "mul(ID, 3) eq 30",
			expectErr: false,
		},
		{
			name:      "mul in complex expression",
			filter:    "mul(Price, 1.5) gt 75 and Name eq 'Laptop'",
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
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestArithmeticFunctions_Div(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "div simple",
			filter:    "div(Price, 2) gt 25",
			expectErr: false,
		},
		{
			name:      "div with comparison",
			filter:    "div(ID, 5) eq 2",
			expectErr: false,
		},
		{
			name:      "div in complex expression",
			filter:    "div(Price, 10) lt 50 and Category eq 'Electronics'",
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
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestArithmeticFunctions_Combined(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "multiple arithmetic functions",
			filter:    "add(Price, 10) gt 100 and sub(ID, 5) lt 10",
			expectErr: false,
		},
		{
			name:      "arithmetic functions with boolean logic",
			filter:    "(mul(Price, 2) gt 100 or div(Price, 2) lt 25) and Category eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "arithmetic functions with NOT",
			filter:    "not (add(ID, 5) eq 10)",
			expectErr: false,
		},
		{
			name:      "mixed arithmetic and string functions",
			filter:    "add(Price, 50) gt 100 and contains(Name, 'Laptop')",
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
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestArithmeticFunctions_SQLGeneration(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name           string
		filter         string
		expectErr      bool
		expectedSQL    string
		expectedArgsNo int
	}{
		{
			name:           "add SQL",
			filter:         "add(Price, 10) gt 100",
			expectErr:      false,
			expectedSQL:    "(price + ?) > ?",
			expectedArgsNo: 2,
		},
		{
			name:           "sub SQL",
			filter:         "sub(Price, 5) lt 50",
			expectErr:      false,
			expectedSQL:    "(price - ?) < ?",
			expectedArgsNo: 2,
		},
		{
			name:           "mul SQL",
			filter:         "mul(Price, 2) eq 100",
			expectErr:      false,
			expectedSQL:    "(price * ?) = ?",
			expectedArgsNo: 2,
		},
		{
			name:           "div SQL",
			filter:         "div(Price, 2) ge 25",
			expectErr:      false,
			expectedSQL:    "(price / ?) >= ?",
			expectedArgsNo: 2,
		},
		{
			name:           "mod SQL with function syntax",
			filter:         "mod(Price, 2) eq 1",
			expectErr:      false,
			expectedSQL:    "(price % ?) = ?",
			expectedArgsNo: 2,
		},
		{
			name:           "mod SQL with infix syntax",
			filter:         "Price mod 2 eq 1",
			expectErr:      false,
			expectedSQL:    "(price % ?) = ?",
			expectedArgsNo: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, meta)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if tt.expectErr {
				return
			}

			sql, args := buildFilterCondition(filterExpr, meta)
			if sql != tt.expectedSQL {
				t.Errorf("Expected SQL: %s, got: %s", tt.expectedSQL, sql)
			}
			if len(args) != tt.expectedArgsNo {
				t.Errorf("Expected %d args, got %d", tt.expectedArgsNo, len(args))
			}
		})
	}
}
