package query

import (
	"testing"
)

func TestStringFunctions_ToLower(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "tolower simple",
			filter:    "tolower(Name) eq 'laptop'",
			expectErr: false,
		},
		{
			name:      "tolower with comparison",
			filter:    "tolower(Category) eq 'electronics'",
			expectErr: false,
		},
		{
			name:      "tolower in complex expression",
			filter:    "tolower(Name) eq 'test' and Price gt 100",
			expectErr: false,
		},
		{
			name:      "tolower wrong argument count",
			filter:    "tolower(Name, 'extra') eq 'test'",
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

func TestStringFunctions_ToUpper(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "toupper simple",
			filter:    "toupper(Name) eq 'LAPTOP'",
			expectErr: false,
		},
		{
			name:      "toupper with comparison",
			filter:    "toupper(Category) eq 'ELECTRONICS'",
			expectErr: false,
		},
		{
			name:      "toupper in complex expression",
			filter:    "toupper(Name) eq 'TEST' or Price lt 50",
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

func TestStringFunctions_Trim(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "trim simple",
			filter:    "trim(Name) eq 'Laptop'",
			expectErr: false,
		},
		{
			name:      "trim with not equal",
			filter:    "trim(Description) ne 'Empty'",
			expectErr: false,
		},
		{
			name:      "trim in boolean expression",
			filter:    "trim(Name) eq 'Test' and Category eq 'Books'",
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

func TestStringFunctions_Length(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "length equals",
			filter:    "length(Name) eq 10",
			expectErr: false,
		},
		{
			name:      "length greater than",
			filter:    "length(Description) gt 50",
			expectErr: false,
		},
		{
			name:      "length less than",
			filter:    "length(Category) lt 20",
			expectErr: false,
		},
		{
			name:      "length in complex expression",
			filter:    "length(Name) gt 5 and Price lt 100",
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

func TestStringFunctions_IndexOf(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "indexof simple",
			filter:    "indexof(Name, 'Lap') eq 1",
			expectErr: false,
		},
		{
			name:      "indexof greater than",
			filter:    "indexof(Description, 'test') gt 0",
			expectErr: false,
		},
		{
			name:      "indexof not found",
			filter:    "indexof(Category, 'xyz') eq -1",
			expectErr: false,
		},
		{
			name:      "indexof in complex expression",
			filter:    "indexof(Name, 'Pro') gt 0 and Price gt 500",
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

func TestStringFunctions_Substring(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "substring with start only",
			filter:    "substring(Name, 1) eq 'aptop'",
			expectErr: false,
		},
		{
			name:      "substring with start and length",
			filter:    "substring(Name, 1, 3) eq 'apt'",
			expectErr: false,
		},
		{
			name:      "substring with comparison",
			filter:    "substring(Category, 0, 5) eq 'Elect'",
			expectErr: false,
		},
		{
			name:      "substring in complex expression",
			filter:    "substring(Name, 0, 3) eq 'Lap' and Price gt 100",
			expectErr: false,
		},
		{
			name:      "substring wrong argument count",
			filter:    "substring(Name) eq 'test'",
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

func TestStringFunctions_Concat(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "concat simple",
			filter:    "concat(Name, ' Pro') eq 'Laptop Pro'",
			expectErr: false,
		},
		{
			name:      "concat with comparison",
			filter:    "concat(Category, 's') eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "concat in complex expression",
			filter:    "concat(Name, ' Test') eq 'Item Test' or Price gt 100",
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

func TestStringFunctions_ConcatWithLiterals(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "concat with two empty strings",
			filter:    "concat('','') eq ''",
			expectErr: false,
		},
		{
			name:      "concat with first arg literal",
			filter:    "concat('prefix_', Name) eq 'prefix_Laptop'",
			expectErr: false,
		},
		{
			name:      "concat with both literal strings",
			filter:    "concat('Hello', ' World') eq 'Hello World'",
			expectErr: false,
		},
		{
			name:      "concat with literal and property",
			filter:    "concat('Item: ', Name) ne ''",
			expectErr: false,
		},
		{
			name:      "concat with empty literal first",
			filter:    "concat('', Name) eq Name",
			expectErr: false,
		},
		{
			name:      "concat in complex expression with literals",
			filter:    "concat('test', 'value') eq 'testvalue' or Price gt 100",
			expectErr: false,
		},
		{
			name:      "nested concat with literals",
			filter:    "concat(concat('a', 'b'), 'c') eq 'abc'",
			expectErr: false,
		},
		{
			name:      "concat literal with function result",
			filter:    "concat('PREFIX_', tolower(Name)) ne ''",
			expectErr: false,
		},
		{
			name:      "concat with special characters in literals",
			filter:    "concat('Hello', '!@#$%') eq 'Hello!@#$%'",
			expectErr: false,
		},
		{
			name:      "concat with unicode in literals",
			filter:    "concat('café', ' au lait') eq 'café au lait'",
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

func TestStringFunctions_Combined(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "multiple string functions",
			filter:    "tolower(Name) eq 'laptop' and length(Description) gt 10",
			expectErr: false,
		},
		{
			name:      "string functions with boolean logic",
			filter:    "(toupper(Category) eq 'ELECTRONICS' or toupper(Category) eq 'BOOKS') and Price gt 50",
			expectErr: false,
		},
		{
			name:      "string functions with NOT",
			filter:    "not (trim(Name) eq 'Test')",
			expectErr: false,
		},
		{
			name:      "mixed string and comparison functions",
			filter:    "contains(Name, 'Laptop') and length(Name) gt 5 and tolower(Category) eq 'electronics'",
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

func TestStringFunctions_SQLGeneration(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name           string
		filter         string
		expectErr      bool
		expectedSQL    string
		expectedArgsNo int
	}{
		{
			name:           "tolower SQL",
			filter:         "tolower(Name) eq 'laptop'",
			expectErr:      false,
			expectedSQL:    "LOWER(name) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "toupper SQL",
			filter:         "toupper(Category) eq 'ELECTRONICS'",
			expectErr:      false,
			expectedSQL:    "UPPER(category) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "trim SQL",
			filter:         "trim(Name) eq 'Test'",
			expectErr:      false,
			expectedSQL:    "TRIM(name) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "length SQL",
			filter:         "length(Name) gt 10",
			expectErr:      false,
			expectedSQL:    "LENGTH(name) > ?",
			expectedArgsNo: 1,
		},
		{
			name:           "indexof SQL",
			filter:         "indexof(Name, 'Lap') eq 1",
			expectErr:      false,
			expectedSQL:    "INSTR(name, ?) - 1 = ?",
			expectedArgsNo: 2,
		},
		{
			name:           "substring SQL with 2 args",
			filter:         "substring(Name, 1) eq 'aptop'",
			expectErr:      false,
			expectedSQL:    "SUBSTR(name, ? + 1, LENGTH(name)) = ?",
			expectedArgsNo: 2,
		},
		{
			name:           "substring SQL with 3 args",
			filter:         "substring(Name, 1, 3) eq 'apt'",
			expectErr:      false,
			expectedSQL:    "SUBSTR(name, ? + 1, ?) = ?",
			expectedArgsNo: 3,
		},
		{
			name:           "concat SQL",
			filter:         "concat(Name, ' Pro') eq 'Laptop Pro'",
			expectErr:      false,
			expectedSQL:    "CONCAT(name, ?) = ?",
			expectedArgsNo: 2,
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
				t.Errorf("Expected SQL: %s, got: %s", tt.expectedSQL, sql)
			}
			if len(args) != tt.expectedArgsNo {
				t.Errorf("Expected %d args, got %d", tt.expectedArgsNo, len(args))
			}
		})
	}
}
