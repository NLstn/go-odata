package query

import (
	"testing"
)

func TestIsOfFunction_Basic(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "isof with property and string type",
			filter:    "isof(Price, 'Edm.String') eq true",
			expectErr: false,
		},
		{
			name:      "isof with property and int32 type",
			filter:    "isof(Name, 'Edm.Int32') eq false",
			expectErr: false,
		},
		{
			name:      "isof with property and decimal type",
			filter:    "isof(Price, 'Edm.Decimal') eq true",
			expectErr: false,
		},
		{
			name:      "isof with property and double type",
			filter:    "isof(Price, 'Edm.Double') eq true",
			expectErr: false,
		},
		{
			name:      "isof with property and int64 type",
			filter:    "isof(Price, 'Edm.Int64') eq true",
			expectErr: false,
		},
		{
			name:      "isof with property and boolean type",
			filter:    "isof(Price, 'Edm.Boolean') eq false",
			expectErr: false,
		},
		{
			name:      "isof with single argument",
			filter:    "isof('Edm.String') eq true",
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

func TestIsOfFunction_ComplexExpressions(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "isof with AND logic",
			filter:    "isof(Price, 'Edm.Int32') eq true and Name eq 'Laptop'",
			expectErr: false,
		},
		{
			name:      "isof with OR logic",
			filter:    "isof(Price, 'Edm.String') eq false or Category eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "isof with NOT",
			filter:    "not (isof(Price, 'Edm.Int32') eq true)",
			expectErr: false,
		},
		{
			name:      "multiple isof calls",
			filter:    "isof(Price, 'Edm.Int32') eq true and isof(Name, 'Edm.String') eq true",
			expectErr: false,
		},
		{
			name:      "isof in complex expression",
			filter:    "(isof(Price, 'Edm.Decimal') eq true and Name eq 'Laptop') or Category eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "isof with comparison operators",
			filter:    "isof(Price, 'Edm.Int32') eq true and Price gt 100",
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

func TestIsOfFunction_ErrorCases(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "isof with no arguments",
			filter:    "isof() eq true",
			expectErr: true,
		},
		{
			name:      "isof with three arguments",
			filter:    "isof(Price, 'Edm.Int32', 'extra') eq true",
			expectErr: true,
		},
		{
			name:      "isof with invalid type (lowercase)",
			filter:    "isof(Price, 'invalidtype') eq true",
			expectErr: true,
		},
		{
			name:      "isof with non-string type",
			filter:    "isof(Price, 123) eq true",
			expectErr: true,
		},
		{
			name:      "isof with invalid property",
			filter:    "isof(NonExistentProperty, 'Edm.String') eq true",
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

func TestIsOfFunction_AllSupportedTypes(t *testing.T) {
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
		t.Run("isof to "+edmType, func(t *testing.T) {
			filter := "isof(Price, '" + edmType + "') eq true"

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

func TestIsOfFunction_SQLGeneration(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name           string
		filter         string
		expectErr      bool
		expectedSQL    string
		expectedArgsNo int
	}{
		{
			name:           "isof to string type",
			filter:         "isof(Price, 'Edm.String') eq true",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS TEXT) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof to integer type",
			filter:         "isof(Price, 'Edm.Int32') eq true",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS INTEGER) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof to decimal type",
			filter:         "isof(Price, 'Edm.Decimal') eq false",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS REAL) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof with AND",
			filter:         "isof(Price, 'Edm.Int32') eq true and Name eq 'Test'",
			expectErr:      false,
			expectedSQL:    "(CASE WHEN CAST(price AS INTEGER) IS NOT NULL THEN 1 ELSE 0 END = ?) AND (name = ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "isof to Int64",
			filter:         "isof(Price, 'Edm.Int64') eq true",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS INTEGER) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof to Double",
			filter:         "isof(Price, 'Edm.Double') eq true",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS REAL) IS NOT NULL THEN 1 ELSE 0 END = ?",
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

			sql, args := buildFilterCondition(filterExpr, meta)
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

func TestIsOfFunction_WithOtherFunctions(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "isof with tolower",
			filter:    "tolower(Name) eq 'laptop' and isof(Price, 'Edm.Int32') eq true",
			expectErr: false,
		},
		{
			name:      "isof with contains",
			filter:    "contains(Name, 'Pro') and isof(Price, 'Edm.Decimal') eq true",
			expectErr: false,
		},
		{
			name:      "isof with ceiling",
			filter:    "ceiling(Price) eq 100 and isof(Name, 'Edm.String') eq true",
			expectErr: false,
		},
		{
			name:      "isof with year",
			filter:    "year(Name) eq 2023 and isof(Price, 'Edm.Int32') eq true",
			expectErr: false,
		},
		{
			name:      "isof with cast",
			filter:    "isof(Price, 'Edm.Int32') eq true and cast(Price, 'Edm.String') eq '100'",
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

func TestIsOfFunction_EdgeCases(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "isof with parentheses",
			filter:    "(isof(Price, 'Edm.Int32') eq true)",
			expectErr: false,
		},
		{
			name:      "isof with multiple parentheses",
			filter:    "((isof(Price, 'Edm.String') eq false))",
			expectErr: false,
		},
		{
			name:      "isof chained with OR",
			filter:    "isof(Price, 'Edm.Int32') eq true or isof(Price, 'Edm.String') eq true or isof(Price, 'Edm.Decimal') eq true",
			expectErr: false,
		},
		{
			name:      "isof single argument with eq true",
			filter:    "isof('Edm.String') eq true",
			expectErr: false,
		},
		{
			name:      "isof negated result",
			filter:    "isof(Price, 'Edm.Boolean') eq false",
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
