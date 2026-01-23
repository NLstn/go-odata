package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestEntityWithBooleans is a test entity with boolean fields
type TestEntityWithBooleans struct {
	ID         int     `json:"ID" odata:"key"`
	Name       string  `json:"Name"`
	Price      float64 `json:"Price"`
	IsActive   bool    `json:"IsActive"`
	IsDeleted  bool    `json:"IsDeleted"`
	IsFeatured bool    `json:"IsFeatured"`
}

func getTestMetadataWithBooleans(t *testing.T) *metadata.EntityMetadata {
	meta, err := metadata.AnalyzeEntity(TestEntityWithBooleans{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	return meta
}

// TestBooleanLiteralSQLGeneration tests SQL generation for boolean literal comparisons
func TestBooleanLiteralSQLGeneration(t *testing.T) {
	meta := getTestMetadataWithBooleans(t)

	tests := []struct {
		name         string
		filter       string
		expectedSQL  string
		expectedArgs int
	}{
		{
			name:         "eq true",
			filter:       "IsActive eq true",
			expectedSQL:  "is_active = ?",
			expectedArgs: 1,
		},
		{
			name:         "eq false",
			filter:       "IsActive eq false",
			expectedSQL:  "is_active = ?",
			expectedArgs: 1,
		},
		{
			name:         "ne true",
			filter:       "IsActive ne true",
			expectedSQL:  "is_active != ?",
			expectedArgs: 1,
		},
		{
			name:         "ne false",
			filter:       "IsActive ne false",
			expectedSQL:  "is_active != ?",
			expectedArgs: 1,
		},
		{
			name:         "Multiple boolean checks with AND",
			filter:       "IsActive eq true and IsDeleted eq false",
			expectedSQL:  "(is_active = ?) AND (is_deleted = ?)",
			expectedArgs: 2,
		},
		{
			name:         "Multiple boolean checks with OR",
			filter:       "IsActive eq true or IsDeleted eq true",
			expectedSQL:  "(is_active = ?) OR (is_deleted = ?)",
			expectedArgs: 2,
		},
		{
			name:         "Combine boolean with value comparison",
			filter:       "IsActive eq true and Price gt 100",
			expectedSQL:  "(is_active = ?) AND (price > ?)",
			expectedArgs: 2,
		},
		{
			name:         "NOT with boolean check",
			filter:       "not (IsActive eq true)",
			expectedSQL:  "NOT (is_active = ?)",
			expectedArgs: 1,
		},
		{
			name:         "Parentheses with boolean literals",
			filter:       "(IsActive eq true and Price gt 50) or IsDeleted eq false",
			expectedSQL:  "((is_active = ?) AND (price > ?)) OR (is_deleted = ?)",
			expectedArgs: 3,
		},
		{
			name:         "Complex boolean expression",
			filter:       "IsActive eq true and (IsDeleted eq false or IsFeatured eq true)",
			expectedSQL:  "(is_active = ?) AND ((is_deleted = ?) OR (is_featured = ?))",
			expectedArgs: 3,
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

// TestBooleanLiteralTokenization verifies that true and false are tokenized correctly
func TestBooleanLiteralTokenization(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
	}{
		{
			name:          "true literal",
			input:         "Name eq true",
			expectedValue: "true",
		},
		{
			name:          "false literal",
			input:         "Name eq false",
			expectedValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			if len(tokens) != 4 { // Identifier, Operator, Boolean, EOF
				t.Fatalf("Expected 4 tokens, got %d", len(tokens))
			}

			if tokens[2].Type != TokenBoolean {
				t.Errorf("Expected TokenBoolean for '%s', got %v", tt.expectedValue, tokens[2].Type)
			}

			if tokens[2].Value != tt.expectedValue {
				t.Errorf("Expected value '%s', got '%s'", tt.expectedValue, tokens[2].Value)
			}
		})
	}
}

// TestBooleanLiteralASTParsing verifies that true and false are parsed into AST correctly
func TestBooleanLiteralASTParsing(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue bool
	}{
		{
			name:          "true literal",
			input:         "Name eq true",
			expectedValue: true,
		},
		{
			name:          "false literal",
			input:         "Name eq false",
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
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

			if litExpr.Type != "boolean" {
				t.Errorf("Expected literal type 'boolean', got '%s'", litExpr.Type)
			}

			boolVal, ok := litExpr.Value.(bool)
			if !ok {
				t.Fatalf("Expected boolean value, got %T", litExpr.Value)
			}

			if boolVal != tt.expectedValue {
				t.Errorf("Expected boolean value %v, got %v", tt.expectedValue, boolVal)
			}
		})
	}
}

// TestBooleanLiteralFilterConversion verifies that boolean AST converts to FilterExpression correctly
func TestBooleanLiteralFilterConversion(t *testing.T) {
	meta := getTestMetadataWithBooleans(t)

	tests := []struct {
		name          string
		input         string
		expectedProp  string
		expectedValue bool
	}{
		{
			name:          "true literal",
			input:         "IsActive eq true",
			expectedProp:  "IsActive",
			expectedValue: true,
		},
		{
			name:          "false literal",
			input:         "IsActive eq false",
			expectedProp:  "IsActive",
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
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

			if filterExpr.Property != tt.expectedProp {
				t.Errorf("Expected property '%s', got '%s'", tt.expectedProp, filterExpr.Property)
			}

			if filterExpr.Operator != OpEqual {
				t.Errorf("Expected operator OpEqual, got %v", filterExpr.Operator)
			}

			boolVal, ok := filterExpr.Value.(bool)
			if !ok {
				t.Fatalf("Expected boolean value, got %T", filterExpr.Value)
			}

			if boolVal != tt.expectedValue {
				t.Errorf("Expected value %v, got %v", tt.expectedValue, boolVal)
			}
		})
	}
}

// TestBooleanLiteralCaseInsensitive verifies that TRUE and FALSE (uppercase) work
func TestBooleanLiteralCaseInsensitive(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue bool
	}{
		{
			name:          "TRUE uppercase",
			input:         "Name eq TRUE",
			expectedValue: true,
		},
		{
			name:          "FALSE uppercase",
			input:         "Name eq FALSE",
			expectedValue: false,
		},
		{
			name:          "True mixed case",
			input:         "Name eq True",
			expectedValue: true,
		},
		{
			name:          "False mixed case",
			input:         "Name eq False",
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
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

			if litExpr.Type != "boolean" {
				t.Errorf("Expected literal type 'boolean', got '%s'", litExpr.Type)
			}

			boolVal, ok := litExpr.Value.(bool)
			if !ok {
				t.Fatalf("Expected boolean value, got %T", litExpr.Value)
			}

			if boolVal != tt.expectedValue {
				t.Errorf("Expected boolean value %v, got %v", tt.expectedValue, boolVal)
			}
		})
	}
}

// TestBooleanLiteralWithStringFunctions tests boolean literals combined with string functions
func TestBooleanLiteralWithStringFunctions(t *testing.T) {
	meta := getTestMetadataWithBooleans(t)

	tests := []struct {
		name        string
		filter      string
		expectError bool
	}{
		{
			name:        "Boolean with contains function",
			filter:      "contains(Name, 'test') eq true and IsActive eq true",
			expectError: false,
		},
		{
			name:        "Boolean with startswith function",
			filter:      "startswith(Name, 'A') eq true and IsDeleted eq false",
			expectError: false,
		},
		{
			name:        "Boolean with multiple functions",
			filter:      "contains(Name, 'laptop') eq true and IsActive eq true and Price gt 100",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFilter(tt.filter, meta, nil, 0)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestBooleanLiteralEdgeCases tests edge cases for boolean literals
func TestBooleanLiteralEdgeCases(t *testing.T) {
	meta := getTestMetadataWithBooleans(t)

	tests := []struct {
		name        string
		filter      string
		expectError bool
	}{
		{
			name:        "Double negation with boolean",
			filter:      "not (not (IsActive eq true))",
			expectError: false,
		},
		{
			name:        "Nested parentheses with boolean",
			filter:      "((IsActive eq true) and ((IsDeleted eq false)))",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFilter(tt.filter, meta, nil, 0)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
