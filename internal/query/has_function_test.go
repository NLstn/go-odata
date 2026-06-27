package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHasFunction(t *testing.T) {
	// Test basic has function parsing
	filterStr := "has(Status, 1)"

	filter, err := parseFilter(filterStr, nil, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Expected filter to be non-nil")
	}

	//nolint:staticcheck // SA5011: t.Fatal above ensures filter is not nil
	t.Logf("Parsed filter: Property=%s, Operator=%s, Value=%v", filter.Property, filter.Operator, filter.Value)

	//nolint:staticcheck // SA5011: t.Fatal above ensures filter is not nil
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

	filter, err := parseFilter(filterStr, entityType, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Expected filter to be non-nil")
	}

	//nolint:staticcheck // SA5011: t.Fatal above ensures filter is not nil
	t.Logf("Parsed filter: Property=%s, Operator=%s, Value=%v", filter.Property, filter.Operator, filter.Value)

	//nolint:staticcheck // SA5011: t.Fatal above ensures filter is not nil
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

// TestHasEnumValueLiteral verifies that the OData enum value literal syntax
// (Namespace.TypeName'MemberName') works correctly with the has operator.
func TestHasEnumValueLiteral(t *testing.T) {
	entityMeta := &metadata.EntityMetadata{
		EntityName: "Product",
		Properties: []metadata.PropertyMetadata{
			{
				Name:      "Status",
				FieldName: "Status",
				JsonName:  "Status",
				IsEnum:    true,
				IsFlags:   true,
				EnumMembers: []metadata.EnumMember{
					{Name: "None", Value: 0},
					{Name: "InStock", Value: 1},
					{Name: "OnSale", Value: 2},
					{Name: "Discontinued", Value: 4},
					{Name: "Featured", Value: 8},
				},
			},
		},
	}

	tests := []struct {
		name          string
		filter        string
		wantValue     int64
		wantErr       bool
	}{
		{
			name:      "qualified enum literal with has infix",
			filter:    "Status has MyService.ProductStatus'Featured'",
			wantValue: 8,
		},
		{
			name:      "qualified enum literal — case-insensitive member name",
			filter:    "Status has MyService.ProductStatus'featured'",
			wantValue: 8,
		},
		{
			name:      "qualified enum literal — InStock",
			filter:    "Status has MyService.ProductStatus'InStock'",
			wantValue: 1,
		},
		{
			name:    "unknown member name returns error",
			filter:  "Status has MyService.ProductStatus'Unknown'",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parseFilter(tt.filter, entityMeta, nil, 0)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none (filter=%v)", filter)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if filter.Operator != OpHas {
				t.Errorf("expected OpHas, got %s", filter.Operator)
			}
			if filter.Value != tt.wantValue {
				t.Errorf("expected value %d, got %v", tt.wantValue, filter.Value)
			}
		})
	}
}

// TestHasEnumValueLiteralSQL verifies end-to-end that enum value literals produce
// the correct bitwise SQL and match the right rows in SQLite.
func TestHasEnumValueLiteralSQL(t *testing.T) {
	type Product struct {
		ID     int    `gorm:"primaryKey"`
		Name   string `gorm:"column:name"`
		Status int    `gorm:"column:status"`
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Status values: InStock=1, OnSale=2, Discontinued=4, Featured=8
	products := []Product{
		{ID: 1, Name: "Normal", Status: 1},        // InStock
		{ID: 2, Name: "Sale", Status: 3},           // InStock|OnSale
		{ID: 3, Name: "Featured", Status: 9},       // InStock|Featured
		{ID: 4, Name: "FeaturedSale", Status: 11},  // InStock|OnSale|Featured
		{ID: 5, Name: "Discontinued", Status: 4},   // Discontinued
	}
	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	meta := &metadata.EntityMetadata{
		EntityName: "Product",
		Properties: []metadata.PropertyMetadata{
			{Name: "ID", FieldName: "ID", JsonName: "ID", ColumnName: "id"},
			{Name: "Name", FieldName: "Name", JsonName: "Name", ColumnName: "name"},
			{
				Name:       "Status",
				FieldName:  "Status",
				JsonName:   "Status",
				ColumnName: "status",
				IsEnum:     true,
				IsFlags:    true,
				EnumMembers: []metadata.EnumMember{
					{Name: "InStock", Value: 1},
					{Name: "OnSale", Value: 2},
					{Name: "Discontinued", Value: 4},
					{Name: "Featured", Value: 8},
				},
			},
		},
	}

	filterExpr, err := parseFilter("Status has Svc.ProductStatus'Featured'", meta, nil, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var count int64
	if err := db.Model(&Product{}).Scopes(func(d *gorm.DB) *gorm.DB {
		return ApplyFilterOnly(d, filterExpr, meta, nil)
	}).Count(&count).Error; err != nil {
		t.Fatalf("query: %v", err)
	}

	// Products 3 and 4 have the Featured bit set
	if count != 2 {
		t.Errorf("expected 2 products with Featured flag, got %d", count)
	}
}

func TestHasInfixParsing(t *testing.T) {
	// Test basic has infix parsing
	filterStr := "Status has 1"

	filter, err := ParseFilterWithoutMetadata(filterStr)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Expected filter to be non-nil")
	}

	//nolint:staticcheck // SA5011: t.Fatal above ensures filter is not nil
	t.Logf("Parsed filter: Property=%s, Operator=%s, Value=%v", filter.Property, filter.Operator, filter.Value)

	//nolint:staticcheck // SA5011: t.Fatal above ensures filter is not nil
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

	filter, err := parseFilter(filterStr, entityType, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Expected filter to be non-nil")
	}

	//nolint:staticcheck // SA5011: t.Fatal above ensures filter is not nil
	t.Logf("Parsed filter: Property=%s, Operator=%s, Value=%v", filter.Property, filter.Operator, filter.Value)

	//nolint:staticcheck // SA5011: t.Fatal above ensures filter is not nil
	if filter.Operator != OpHas {
		t.Errorf("Expected operator to be OpHas, got %s", filter.Operator)
	}

	//nolint:staticcheck // SA5011: t.Fatal above ensures filter is not nil
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

			defer ReleaseASTNode(ast)

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
			funcFilter, err := parseFilter(tt.functionForm, entityType, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse function form: %v", err)
			}

			// Parse infix form
			infixFilter, err := parseFilter(tt.infixForm, entityType, nil, 0)
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
