package query

import (
	"testing"
)

func TestCastFunction_Basic(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "cast to string",
			filter:    "cast(Price, 'Edm.String') eq '100'",
			expectErr: false,
		},
		{
			name:      "cast to int32",
			filter:    "cast(Name, 'Edm.Int32') eq 100",
			expectErr: false,
		},
		{
			name:      "cast to decimal",
			filter:    "cast(Price, 'Edm.Decimal') eq 123.45",
			expectErr: false,
		},
		{
			name:      "cast to double",
			filter:    "cast(Price, 'Edm.Double') gt 100.0",
			expectErr: false,
		},
		{
			name:      "cast to int64",
			filter:    "cast(Price, 'Edm.Int64') lt 1000",
			expectErr: false,
		},
		{
			name:      "cast to boolean",
			filter:    "cast(Price, 'Edm.Boolean') eq true",
			expectErr: false,
		},
		{
			name:      "cast with greater than",
			filter:    "cast(Price, 'Edm.Int32') gt 50",
			expectErr: false,
		},
		{
			name:      "cast with less than or equal",
			filter:    "cast(Price, 'Edm.Decimal') le 200.5",
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

func TestCastFunction_ComplexExpressions(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "cast with AND logic",
			filter:    "cast(Price, 'Edm.Int32') gt 100 and Name eq 'Laptop'",
			expectErr: false,
		},
		{
			name:      "cast with OR logic",
			filter:    "cast(Price, 'Edm.String') eq '100' or Category eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "cast with NOT",
			filter:    "not (cast(Price, 'Edm.Int32') eq 100)",
			expectErr: false,
		},
		{
			name:      "multiple casts",
			filter:    "cast(Price, 'Edm.Int32') gt 100 and cast(Name, 'Edm.String') eq 'Test'",
			expectErr: false,
		},
		{
			name:      "cast in complex expression",
			filter:    "(cast(Price, 'Edm.Decimal') gt 100.5 and Name eq 'Laptop') or Category eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "cast with comparison operators",
			filter:    "cast(Price, 'Edm.Int32') ge 50 and cast(Price, 'Edm.Int32') le 150",
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

func TestCastFunction_ErrorCases(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "cast with no arguments",
			filter:    "cast() eq 100",
			expectErr: true,
		},
		{
			name:      "cast with one argument",
			filter:    "cast(Price) eq 100",
			expectErr: true,
		},
		{
			name:      "cast with three arguments",
			filter:    "cast(Price, 'Edm.Int32', 'extra') eq 100",
			expectErr: true,
		},
		{
			name:      "cast with invalid type",
			filter:    "cast(Price, 'InvalidType') eq 100",
			expectErr: true,
		},
		{
			name:      "cast with non-string type",
			filter:    "cast(Price, 123) eq 100",
			expectErr: true,
		},
		{
			name:      "cast with invalid property",
			filter:    "cast(NonExistentProperty, 'Edm.String') eq 'test'",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if tt.expectErr {
					return
				}
				t.Fatalf("Tokenization failed: %v", err)
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if tt.expectErr {
					return
				}
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

func TestCastFunction_AllSupportedTypes(t *testing.T) {
	meta := getTestMetadata(t)

	supportedTypes := []string{
		"Edm.String",
		"Edm.Int32",
		"Edm.Int64",
		"Edm.Decimal",
		"Edm.Double",
		"Edm.Single",
		"Edm.Boolean",
		"Edm.DateTimeOffset",
		"Edm.Date",
		"Edm.TimeOfDay",
		"Edm.Guid",
		"Edm.Binary",
		"Edm.Byte",
		"Edm.SByte",
		"Edm.Int16",
	}

	for _, edmType := range supportedTypes {
		t.Run("cast to "+edmType, func(t *testing.T) {
			filter := "cast(Price, '" + edmType + "') eq 100"

			tokenizer := NewTokenizer(filter)
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
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestCastFunction_SQLGeneration(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name           string
		filter         string
		expectErr      bool
		expectedSQL    string
		expectedArgsNo int
	}{
		{
			name:           "cast to TEXT",
			filter:         "cast(Price, 'Edm.String') eq 'test'",
			expectErr:      false,
			expectedSQL:    "CAST(price AS TEXT) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "cast to INTEGER",
			filter:         "cast(Price, 'Edm.Int32') eq 100",
			expectErr:      false,
			expectedSQL:    "CAST(price AS INTEGER) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "cast to REAL",
			filter:         "cast(Price, 'Edm.Decimal') gt 99.99",
			expectErr:      false,
			expectedSQL:    "CAST(price AS REAL) > ?",
			expectedArgsNo: 1,
		},
		{
			name:           "cast with AND",
			filter:         "cast(Price, 'Edm.Int32') gt 50 and Name eq 'Test'",
			expectErr:      false,
			expectedSQL:    "(CAST(price AS INTEGER) > ?) AND (name = ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "cast to Int64",
			filter:         "cast(Price, 'Edm.Int64') lt 1000",
			expectErr:      false,
			expectedSQL:    "CAST(price AS INTEGER) < ?",
			expectedArgsNo: 1,
		},
		{
			name:           "cast to Double",
			filter:         "cast(Price, 'Edm.Double') ge 50.5",
			expectErr:      false,
			expectedSQL:    "CAST(price AS REAL) >= ?",
			expectedArgsNo: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, meta, nil, 0)
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

			t.Logf("✓ OData: %s", tt.filter)
			t.Logf("✓ SQL:   %s", sql)
			t.Logf("✓ Args:  %v", args)
		})
	}
}

func TestCastFunction_WithOtherFunctions(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "cast with tolower",
			filter:    "tolower(Name) eq 'laptop' and cast(Price, 'Edm.Int32') gt 100",
			expectErr: false,
		},
		{
			name:      "cast with contains",
			filter:    "contains(Name, 'Pro') and cast(Price, 'Edm.Decimal') lt 500.0",
			expectErr: false,
		},
		{
			name:      "cast with ceiling",
			filter:    "ceiling(Price) eq 100 and cast(Name, 'Edm.String') eq 'Test'",
			expectErr: false,
		},
		{
			name:      "cast with year",
			filter:    "year(Name) eq 2023 and cast(Price, 'Edm.Int32') gt 50",
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

func TestCastFunction_EdgeCases(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "cast with parentheses",
			filter:    "(cast(Price, 'Edm.Int32') gt 100)",
			expectErr: false,
		},
		{
			name:      "cast with multiple parentheses",
			filter:    "((cast(Price, 'Edm.String') eq '100'))",
			expectErr: false,
		},
		{
			name:      "cast with null comparison",
			filter:    "cast(Price, 'Edm.Int32') eq null",
			expectErr: false,
		},
		{
			name:      "cast chained with OR",
			filter:    "cast(Price, 'Edm.Int32') eq 100 or cast(Price, 'Edm.Int32') eq 200 or cast(Price, 'Edm.Int32') eq 300",
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
