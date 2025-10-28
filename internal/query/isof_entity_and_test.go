package query

import (
	"testing"
)

// TestIsOfFunction_EntityTypeWithAnd tests that isof function with entity types
// generates correct SQL when used with logical operators.
// This test validates the fix for the issue where isof('EntityType') without
// explicit comparison (eq true) was generating empty SQL clauses.
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
			name:           "isof entity type with and",
			filter:         "isof('Namespace.SpecialProduct') and Price gt 100",
			expectErr:      false,
			expectedSQL:    "(1 = ?) AND (price > ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "isof entity type eq true with and",
			filter:         "isof('Namespace.SpecialProduct') eq true and Price gt 100",
			expectErr:      false,
			expectedSQL:    "(1 = ?) AND (price > ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "isof entity type with or",
			filter:         "isof('Namespace.SpecialProduct') or Price lt 50",
			expectErr:      false,
			expectedSQL:    "(1 = ?) OR (price < ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "isof entity type standalone",
			filter:         "isof('Namespace.SpecialProduct')",
			expectErr:      false,
			expectedSQL:    "1 = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof entity type negated",
			filter:         "not isof('Namespace.SpecialProduct')",
			expectErr:      false,
			expectedSQL:    "NOT (1 = ?)",
			expectedArgsNo: 1,
		},
		{
			name:           "isof entity type with parentheses and and",
			filter:         "(isof('Namespace.SpecialProduct')) and Price gt 100",
			expectErr:      false,
			expectedSQL:    "(1 = ?) AND (price > ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "isof entity type eq false with and",
			filter:         "isof('Namespace.SpecialProduct') eq false and Price gt 100",
			expectErr:      false,
			expectedSQL:    "(1 = ?) AND (price > ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "multiple isof entity type checks",
			filter:         "isof('Namespace.SpecialProduct') and isof('Namespace.AnotherType')",
			expectErr:      false,
			expectedSQL:    "(1 = ?) AND (1 = ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "isof entity type in complex expression",
			filter:         "(isof('Namespace.SpecialProduct') and Price gt 100) or Category eq 'Electronics'",
			expectErr:      false,
			expectedSQL:    "((1 = ?) AND (price > ?)) OR (category = ?)",
			expectedArgsNo: 3,
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
