package query

import (
	"testing"
)

func TestMathFunctions_Ceiling(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "ceiling simple",
			filter:    "ceiling(Price) eq 100",
			expectErr: false,
		},
		{
			name:      "ceiling with greater than",
			filter:    "ceiling(Price) gt 50",
			expectErr: false,
		},
		{
			name:      "ceiling in complex expression",
			filter:    "ceiling(Price) lt 200 and Category eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "ceiling with OR logic",
			filter:    "ceiling(Price) eq 100 or Name eq 'Laptop'",
			expectErr: false,
		},
		{
			name:      "ceiling wrong argument count - no args",
			filter:    "ceiling() eq 100",
			expectErr: true,
		},
		{
			name:      "ceiling wrong argument count - two args",
			filter:    "ceiling(Price, 10) eq 100",
			expectErr: true,
		},
		{
			name:      "ceiling with NOT",
			filter:    "not (ceiling(Price) eq 100)",
			expectErr: false,
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

func TestMathFunctions_Floor(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "floor simple",
			filter:    "floor(Price) eq 99",
			expectErr: false,
		},
		{
			name:      "floor with less than",
			filter:    "floor(Price) lt 100",
			expectErr: false,
		},
		{
			name:      "floor in complex expression",
			filter:    "floor(Price) ge 50 or Category eq 'Books'",
			expectErr: false,
		},
		{
			name:      "floor with AND logic",
			filter:    "floor(Price) eq 99 and ID gt 5",
			expectErr: false,
		},
		{
			name:      "floor wrong argument count - no args",
			filter:    "floor() eq 99",
			expectErr: true,
		},
		{
			name:      "floor wrong argument count - two args",
			filter:    "floor(Price, 5) eq 99",
			expectErr: true,
		},
		{
			name:      "floor with NOT",
			filter:    "not (floor(Price) lt 100)",
			expectErr: false,
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

func TestMathFunctions_Round(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "round simple",
			filter:    "round(Price) eq 100",
			expectErr: false,
		},
		{
			name:      "round with not equal",
			filter:    "round(Price) ne 50",
			expectErr: false,
		},
		{
			name:      "round in complex expression",
			filter:    "round(Price) le 150 and Name eq 'Laptop'",
			expectErr: false,
		},
		{
			name:      "round with OR logic",
			filter:    "round(Price) gt 100 or Category eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "round wrong argument count - no args",
			filter:    "round() eq 100",
			expectErr: true,
		},
		{
			name:      "round wrong argument count - two args",
			filter:    "round(Price, 2) eq 100",
			expectErr: true,
		},
		{
			name:      "round with NOT",
			filter:    "not (round(Price) eq 100)",
			expectErr: false,
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

func TestMathFunctions_Combined(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "ceiling and floor",
			filter:    "ceiling(Price) gt 100 and floor(Price) lt 100",
			expectErr: false,
		},
		{
			name:      "all three math functions",
			filter:    "ceiling(Price) eq 100 and floor(Price) lt 100 or round(Price) eq 100",
			expectErr: false,
		},
		{
			name:      "math functions with string functions",
			filter:    "ceiling(Price) eq 100 and contains(Name, 'Laptop')",
			expectErr: false,
		},
		{
			name:      "math functions with arithmetic functions",
			filter:    "ceiling(add(Price, 10)) gt 100",
			expectErr: false,
		},
		{
			name:      "nested with date functions",
			filter:    "round(Price) gt 50 and year(ID) eq 2024",
			expectErr: false,
		},
		{
			name:      "complex boolean logic",
			filter:    "(ceiling(Price) gt 100 or floor(Price) lt 50) and round(Price) ne 75",
			expectErr: false,
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

func TestMathFunctions_SQLGeneration(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name           string
		filter         string
		expectErr      bool
		expectedSQL    string
		expectedArgsNo int
	}{
		{
			name:           "ceiling SQL",
			filter:         "ceiling(Price) eq 100",
			expectErr:      false,
			expectedSQL:    "CASE WHEN price = CAST(price AS INTEGER) THEN price ELSE CAST(price AS INTEGER) + (CASE WHEN price > 0 THEN 1 ELSE 0 END) END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "ceiling SQL with greater than",
			filter:         "ceiling(Price) gt 50",
			expectErr:      false,
			expectedSQL:    "CASE WHEN price = CAST(price AS INTEGER) THEN price ELSE CAST(price AS INTEGER) + (CASE WHEN price > 0 THEN 1 ELSE 0 END) END > ?",
			expectedArgsNo: 1,
		},
		{
			name:           "floor SQL",
			filter:         "floor(Price) eq 99",
			expectErr:      false,
			expectedSQL:    "CASE WHEN price = CAST(price AS INTEGER) THEN price ELSE CAST(price AS INTEGER) - (CASE WHEN price < 0 THEN 1 ELSE 0 END) END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "floor SQL with less than",
			filter:         "floor(Price) lt 100",
			expectErr:      false,
			expectedSQL:    "CASE WHEN price = CAST(price AS INTEGER) THEN price ELSE CAST(price AS INTEGER) - (CASE WHEN price < 0 THEN 1 ELSE 0 END) END < ?",
			expectedArgsNo: 1,
		},
		{
			name:           "round SQL",
			filter:         "round(Price) eq 100",
			expectErr:      false,
			expectedSQL:    "ROUND(price) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "round SQL with not equal",
			filter:         "round(Price) ne 50",
			expectErr:      false,
			expectedSQL:    "ROUND(price) != ?",
			expectedArgsNo: 1,
		},
		{
			name:           "ceiling with less than or equal",
			filter:         "ceiling(Price) le 200",
			expectErr:      false,
			expectedSQL:    "CASE WHEN price = CAST(price AS INTEGER) THEN price ELSE CAST(price AS INTEGER) + (CASE WHEN price > 0 THEN 1 ELSE 0 END) END <= ?",
			expectedArgsNo: 1,
		},
		{
			name:           "floor with greater than or equal",
			filter:         "floor(Price) ge 50",
			expectErr:      false,
			expectedSQL:    "CASE WHEN price = CAST(price AS INTEGER) THEN price ELSE CAST(price AS INTEGER) - (CASE WHEN price < 0 THEN 1 ELSE 0 END) END >= ?",
			expectedArgsNo: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, meta, nil)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if tt.expectErr {
				return
			}

			sql, args := buildFilterCondition("sqlite", filterExpr, meta)
			if sql != tt.expectedSQL {
				t.Errorf("Expected SQL:\n%s\nGot:\n%s", tt.expectedSQL, sql)
			}
			if len(args) != tt.expectedArgsNo {
				t.Errorf("Expected %d args, got %d", tt.expectedArgsNo, len(args))
			}
		})
	}
}

func TestMathFunctions_EdgeCases(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "ceiling on non-numeric property should parse",
			filter:    "ceiling(Name) eq 100",
			expectErr: false, // Parser doesn't validate types, just syntax
		},
		{
			name:      "multiple ceiling in one expression",
			filter:    "ceiling(Price) eq 100 and ceiling(ID) eq 5",
			expectErr: false,
		},
		{
			name:      "floor with ID field",
			filter:    "floor(ID) eq 5",
			expectErr: false,
		},
		{
			name:      "round with parentheses",
			filter:    "(round(Price) gt 50)",
			expectErr: false,
		},
		{
			name:      "math function in nested NOT",
			filter:    "not (not (ceiling(Price) eq 100))",
			expectErr: false,
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
