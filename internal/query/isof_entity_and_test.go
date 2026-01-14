package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestIsOfFunction_EntityTypeWithAnd tests that isof function with entity types
// generates correct SQL when used with logical operators.
// This test validates that when no discriminator column is configured,
// isof('EntityType') returns 1 = 0 (matches no entities).
func TestIsOfFunction_EntityTypeWithAnd(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name           string
		filter         string
		expectErr      bool
		expectedSQL    string
		expectedArgsNo int
	}{
		{
			name:           "isof entity type with and (no discriminator)",
			filter:         "isof('Namespace.SpecialProduct') and Price gt 100",
			expectErr:      false,
			expectedSQL:    "(1 = 0) AND (price > ?)",
			expectedArgsNo: 1,
		},
		{
			name:           "isof entity type eq true with and (no discriminator)",
			filter:         "isof('Namespace.SpecialProduct') eq true and Price gt 100",
			expectErr:      false,
			expectedSQL:    "(1 = ?) AND (price > ?)", // Goes through different code path
			expectedArgsNo: 2,
		},
		{
			name:           "isof entity type with or (no discriminator)",
			filter:         "isof('Namespace.SpecialProduct') or Price lt 50",
			expectErr:      false,
			expectedSQL:    "(1 = 0) OR (price < ?)",
			expectedArgsNo: 1,
		},
		{
			name:           "isof entity type standalone (no discriminator)",
			filter:         "isof('Namespace.SpecialProduct')",
			expectErr:      false,
			expectedSQL:    "1 = 0",
			expectedArgsNo: 0,
		},
		{
			name:           "isof entity type negated (no discriminator)",
			filter:         "not isof('Namespace.SpecialProduct')",
			expectErr:      false,
			expectedSQL:    "NOT (1 = 0)",
			expectedArgsNo: 0,
		},
		{
			name:           "isof entity type with parentheses and and (no discriminator)",
			filter:         "(isof('Namespace.SpecialProduct')) and Price gt 100",
			expectErr:      false,
			expectedSQL:    "(1 = 0) AND (price > ?)",
			expectedArgsNo: 1,
		},
		{
			name:           "isof entity type eq false with and (no discriminator)",
			filter:         "isof('Namespace.SpecialProduct') eq false and Price gt 100",
			expectErr:      false,
			expectedSQL:    "(1 = ?) AND (price > ?)", // Goes through different code path
			expectedArgsNo: 2,
		},
		{
			name:           "multiple isof entity type checks (no discriminator)",
			filter:         "isof('Namespace.SpecialProduct') and isof('Namespace.AnotherType')",
			expectErr:      false,
			expectedSQL:    "(1 = 0) AND (1 = 0)",
			expectedArgsNo: 0,
		},
		{
			name:           "isof entity type in complex expression (no discriminator)",
			filter:         "(isof('Namespace.SpecialProduct') and Price gt 100) or Category eq 'Electronics'",
			expectErr:      false,
			expectedSQL:    "((1 = 0) AND (price > ?)) OR (category = ?)",
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

			sql, args := buildFilterCondition("sqlite", filterExpr, meta)
			t.Logf("✓ OData: %s", tt.filter)
			t.Logf("✓ SQL:   %s", sql)
			t.Logf("✓ Args:  %v", args)

			if sql != tt.expectedSQL {
				t.Errorf("Expected SQL:\n%s\nGot:\n%s", tt.expectedSQL, sql)
			}

			if len(args) != tt.expectedArgsNo {
				t.Errorf("Expected %d args, got %d", tt.expectedArgsNo, len(args))
			}
		})
	}
}

// PolymorphicEntity is a test entity with a type discriminator property
type PolymorphicEntity struct {
	ID          string  `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name"`
	Price       float64 `json:"Price"`
	ProductType string  `json:"ProductType"` // Type discriminator
}

// TestIsOfFunction_EntityTypeWithDiscriminator tests that isof function with entity types
// generates correct SQL when a discriminator column is configured.
func TestIsOfFunction_EntityTypeWithDiscriminator(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(PolymorphicEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Verify discriminator is detected
	if meta.TypeDiscriminator == nil {
		t.Fatal("Expected TypeDiscriminator to be set")
	}
	if meta.TypeDiscriminator.ColumnName != "product_type" {
		t.Errorf("Expected discriminator column 'product_type', got '%s'", meta.TypeDiscriminator.ColumnName)
	}

	tests := []struct {
		name           string
		filter         string
		expectErr      bool
		expectedSQL    string
		expectedArgsNo int
	}{
		{
			name:           "isof entity type with discriminator",
			filter:         "isof('Namespace.SpecialProduct')",
			expectErr:      false,
			expectedSQL:    `"product_type" = ?`,
			expectedArgsNo: 1,
		},
		{
			name:           "isof entity type with and (discriminator)",
			filter:         "isof('Namespace.SpecialProduct') and Price gt 100",
			expectErr:      false,
			expectedSQL:    `("product_type" = ?) AND (price > ?)`,
			expectedArgsNo: 2,
		},
		{
			name:           "isof entity type with or (discriminator)",
			filter:         "isof('Namespace.SpecialProduct') or Price lt 50",
			expectErr:      false,
			expectedSQL:    `("product_type" = ?) OR (price < ?)`,
			expectedArgsNo: 2,
		},
		{
			name:           "isof entity type negated (discriminator)",
			filter:         "not isof('Namespace.SpecialProduct')",
			expectErr:      false,
			expectedSQL:    `NOT ("product_type" = ?)`,
			expectedArgsNo: 1,
		},
		{
			name:           "isof simple type name extracts from qualified",
			filter:         "isof('ComplianceService.SpecialProduct')",
			expectErr:      false,
			expectedSQL:    `"product_type" = ?`,
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
			t.Logf("✓ OData: %s", tt.filter)
			t.Logf("✓ SQL:   %s", sql)
			t.Logf("✓ Args:  %v", args)

			if sql != tt.expectedSQL {
				t.Errorf("Expected SQL:\n%s\nGot:\n%s", tt.expectedSQL, sql)
			}

			if len(args) != tt.expectedArgsNo {
				t.Errorf("Expected %d args, got %d", tt.expectedArgsNo, len(args))
			}

			// Verify the type name is extracted correctly
			if len(args) > 0 {
				if typeName, ok := args[0].(string); ok {
					if typeName != "SpecialProduct" {
						t.Errorf("Expected type name 'SpecialProduct', got '%s'", typeName)
					}
				}
			}
		})
	}
}
