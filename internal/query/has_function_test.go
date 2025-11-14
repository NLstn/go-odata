package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

func TestHasFunction(t *testing.T) {
	// Test basic has function parsing
	filterStr := "has(Status, 1)"

	filter, err := parseFilterWithoutMetadata(filterStr)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Expected filter to be non-nil")
	}

	//nolint:staticcheck // SA5011: t.Fatal above ensures filter is not nil
	t.Logf("Parsed filter: Property=%s, Operator=%s, Value=%v", filter.Property, filter.Operator, filter.Value)

	if filter.Operator != OpHas {
		t.Errorf("Expected operator to be OpHas, got %s", filter.Operator)
	}

	if filter.Property != "Status" {
		t.Errorf("Expected property to be 'Status', got '%s'", filter.Property)
	}

	// Value should be the integer 1
	if filter.Value != int64(1) {
		t.Errorf("Expected value to be 1, got %v (type %T)", filter.Value, filter.Value)
	}
}

func TestHasFunctionWithMetadata(t *testing.T) {
	// Create mock metadata
	entityType := &metadata.EntityMetadata{
		EntityName: "Product",
		Properties: []metadata.PropertyMetadata{
			{
				Name:      "Status",
				FieldName: "Status",
				JsonName:  "Status",
				IsEnum:    true,
				IsFlags:   true,
			},
		},
	}

	filterStr := "has(Status, 1)"

	filter, err := parseFilter(filterStr, entityType, nil)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Expected filter to be non-nil")
	}

	t.Logf("Parsed filter: Property=%s, Operator=%s, Value=%v", filter.Property, filter.Operator, filter.Value)

	if filter.Operator != OpHas {
		t.Errorf("Expected operator to be OpHas, got %s", filter.Operator)
	}

	if filter.Property != "Status" {
		t.Errorf("Expected property to be 'Status', got '%s'", filter.Property)
	}
}

func TestHasFunctionSQLGeneration(t *testing.T) {
	filter := &FilterExpression{
		Property: "Status",
		Operator: OpHas,
		Value:    int64(1),
	}

	sql, args := buildSimpleOperatorCondition(filter.Operator, "status", filter.Value)

	t.Logf("Generated SQL: %s", sql)
	t.Logf("Args: %v", args)

	expectedSQL := "(status & ?) = ?"
	if sql != expectedSQL {
		t.Errorf("Expected SQL '%s', got '%s'", expectedSQL, sql)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if args[0] != int64(1) || args[1] != int64(1) {
		t.Errorf("Expected args [1, 1], got %v", args)
	}
}

func TestHasInfixParsing(t *testing.T) {
	// Test basic has infix parsing
	filterStr := "Status has 1"

	filter, err := parseFilterWithoutMetadata(filterStr)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Expected filter to be non-nil")
	}

	t.Logf("Parsed filter: Property=%s, Operator=%s, Value=%v", filter.Property, filter.Operator, filter.Value)

	if filter.Operator != OpHas {
		t.Errorf("Expected operator to be OpHas, got %s", filter.Operator)
	}

	if filter.Property != "Status" {
		t.Errorf("Expected property to be 'Status', got '%s'", filter.Property)
	}

	// Value should be the integer 1
	if filter.Value != int64(1) {
		t.Errorf("Expected value to be 1, got %v (type %T)", filter.Value, filter.Value)
	}
}

func TestHasInfixWithMetadata(t *testing.T) {
	// Create mock metadata
	entityType := &metadata.EntityMetadata{
		EntityName: "Product",
		Properties: []metadata.PropertyMetadata{
			{
				Name:      "Status",
				FieldName: "Status",
				JsonName:  "Status",
				IsEnum:    true,
				IsFlags:   true,
			},
		},
	}

	filterStr := "Status has 1"

	filter, err := parseFilter(filterStr, entityType, nil)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Expected filter to be non-nil")
	}

	t.Logf("Parsed filter: Property=%s, Operator=%s, Value=%v", filter.Property, filter.Operator, filter.Value)

	if filter.Operator != OpHas {
		t.Errorf("Expected operator to be OpHas, got %s", filter.Operator)
	}

	if filter.Property != "Status" {
		t.Errorf("Expected property to be 'Status', got '%s'", filter.Property)
	}
}

func TestHasInfixInComplexExpression(t *testing.T) {
	entityType := &metadata.EntityMetadata{
		EntityName: "Product",
		Properties: []metadata.PropertyMetadata{
			{
				Name:      "Status",
				FieldName: "Status",
				JsonName:  "Status",
				IsEnum:    true,
				IsFlags:   true,
			},
			{
				Name:      "Category",
				FieldName: "Category",
				JsonName:  "Category",
				IsEnum:    false,
				IsFlags:   false,
			},
		},
	}

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "has infix with and",
			filter:    "Status has 1 and Category eq 'Electronics'",
			expectErr: false,
		},
		{
			name:      "has infix with or",
			filter:    "Status has 2 or Status has 4",
			expectErr: false,
		},
		{
			name:      "has infix with parentheses",
			filter:    "(Status has 1) and Category eq 'Books'",
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

			filterExpr, err := ASTToFilterExpression(ast, entityType)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestHasBothSyntaxes(t *testing.T) {
	// Test that both function and infix syntax work and produce the same result
	entityType := &metadata.EntityMetadata{
		EntityName: "Product",
		Properties: []metadata.PropertyMetadata{
			{
				Name:      "Status",
				FieldName: "Status",
				JsonName:  "Status",
				IsEnum:    true,
				IsFlags:   true,
			},
		},
	}

	tests := []struct {
		name         string
		functionForm string
		infixForm    string
	}{
		{
			name:         "basic has",
			functionForm: "has(Status, 1)",
			infixForm:    "Status has 1",
		},
		{
			name:         "has with larger value",
			functionForm: "has(Status, 255)",
			infixForm:    "Status has 255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse function form
			funcFilter, err := parseFilter(tt.functionForm, entityType, nil)
			if err != nil {
				t.Fatalf("Failed to parse function form: %v", err)
			}

			// Parse infix form
			infixFilter, err := parseFilter(tt.infixForm, entityType, nil)
			if err != nil {
				t.Fatalf("Failed to parse infix form: %v", err)
			}

			// Both should have the same operator
			if funcFilter.Operator != infixFilter.Operator {
				t.Errorf("Operators don't match: function=%s, infix=%s", funcFilter.Operator, infixFilter.Operator)
			}

			// Both should be OpHas
			if funcFilter.Operator != OpHas {
				t.Errorf("Expected OpHas, got %s", funcFilter.Operator)
			}

			// Both should have the same property
			if funcFilter.Property != infixFilter.Property {
				t.Errorf("Properties don't match: function=%s, infix=%s", funcFilter.Property, infixFilter.Property)
			}

			// Both should have the same value
			if funcFilter.Value != infixFilter.Value {
				t.Errorf("Values don't match: function=%v, infix=%v", funcFilter.Value, infixFilter.Value)
			}
		})
	}
}
