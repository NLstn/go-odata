package query

import (
	"testing"
)

// TestNullLiteralSQLGeneration tests SQL generation for null literal comparisons
func TestNullLiteralSQLGeneration(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name         string
		filter       string
		expectedSQL  string
		expectedArgs int
	}{
		{
			name:         "eq null",
			filter:       "Description eq null",
			expectedSQL:  "description IS NULL",
			expectedArgs: 0,
		},
		{
			name:         "ne null",
			filter:       "Description ne null",
			expectedSQL:  "description IS NOT NULL",
			expectedArgs: 0,
		},
		{
			name:         "Multiple null checks with AND",
			filter:       "Description eq null and Name eq null",
			expectedSQL:  "(description IS NULL) AND (name IS NULL)",
			expectedArgs: 0,
		},
		{
			name:         "Multiple null checks with OR",
			filter:       "Description eq null or Name eq null",
			expectedSQL:  "(description IS NULL) OR (name IS NULL)",
			expectedArgs: 0,
		},
		{
			name:         "Combine null check with value comparison",
			filter:       "Description eq null and Price gt 100",
			expectedSQL:  "(description IS NULL) AND (price > ?)",
			expectedArgs: 1,
		},
		{
			name:         "NOT with null check",
			filter:       "not (Description eq null)",
			expectedSQL:  "NOT (description IS NULL)",
			expectedArgs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the filter
			filterExpr, err := parseFilter(tt.filter, meta, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			// Build SQL condition
			sql, args := buildFilterCondition("sqlite", filterExpr, meta)

			if sql != tt.expectedSQL {
				t.Errorf("Expected SQL:\n  %s\nGot:\n  %s", tt.expectedSQL, sql)
			}

			if len(args) != tt.expectedArgs {
				t.Errorf("Expected %d args, got %d", tt.expectedArgs, len(args))
			}
		})
	}
}

// TestNullLiteralTokenization verifies that null is tokenized correctly
func TestNullLiteralTokenization(t *testing.T) {
	tokenizer := NewTokenizer("Description eq null")
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	if len(tokens) != 4 { // Identifier, Operator, Null, EOF
		t.Fatalf("Expected 4 tokens, got %d", len(tokens))
	}

	if tokens[2].Type != TokenNull {
		t.Errorf("Expected TokenNull for 'null', got %v", tokens[2].Type)
	}

	if tokens[2].Value != "null" {
		t.Errorf("Expected value 'null', got '%s'", tokens[2].Value)
	}
}

// TestNullLiteralASTParsing verifies that null is parsed into AST correctly
func TestNullLiteralASTParsing(t *testing.T) {
	tokenizer := NewTokenizer("Description eq null")
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	parser := NewASTParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parsing failed: %v", err)
	}

	defer ReleaseASTNode(ast)

	compExpr, ok := ast.(*ComparisonExpr)
	if !ok {
		t.Fatal("Expected ComparisonExpr")
	}

	litExpr, ok := compExpr.Right.(*LiteralExpr)
	if !ok {
		t.Fatal("Expected LiteralExpr on right side")
	}

	if litExpr.Type != "null" {
		t.Errorf("Expected literal type 'null', got '%s'", litExpr.Type)
	}

	if litExpr.Value != nil {
		t.Errorf("Expected literal value nil, got %v", litExpr.Value)
	}
}

// TestNullLiteralFilterConversion verifies that null AST converts to FilterExpression correctly
func TestNullLiteralFilterConversion(t *testing.T) {
	meta := getTestMetadata(t)

	tokenizer := NewTokenizer("Description eq null")
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	parser := NewASTParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parsing failed: %v", err)
	}

	defer ReleaseASTNode(ast)

	filterExpr, err := ASTToFilterExpression(ast, meta)
	if err != nil {
		t.Fatalf("Failed to convert AST to FilterExpression: %v", err)
	}

	if filterExpr.Property != "Description" {
		t.Errorf("Expected property 'Description', got '%s'", filterExpr.Property)
	}

	if filterExpr.Operator != OpEqual {
		t.Errorf("Expected operator OpEqual, got %v", filterExpr.Operator)
	}

	if filterExpr.Value != nil {
		t.Errorf("Expected value nil, got %v", filterExpr.Value)
	}
}
