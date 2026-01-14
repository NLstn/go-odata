package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestEntity2 represents an entity with two numeric fields for comparison testing
type TestEntity2 struct {
	ID    int     `json:"ID" odata:"key"`
	Price float64 `json:"Price"`
	Cost  float64 `json:"Cost"`
	Name  string  `json:"Name"`
}

func getTestMetadata2(t *testing.T) *metadata.EntityMetadata {
	meta, err := metadata.AnalyzeEntity(TestEntity2{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	return meta
}

// TestPropertyToPropertyComparison tests that property-to-property comparisons
// generate correct SQL (Price > Cost, not Price > 'Cost')
func TestPropertyToPropertyComparison(t *testing.T) {
	meta := getTestMetadata2(t)

	tests := []struct {
		name             string
		filter           string
		expectedSQL      string
		expectIdentifier bool // true if RHS should be treated as identifier, not literal
	}{
		{
			name:             "Property gt Property",
			filter:           "Price gt Cost",
			expectedSQL:      `price > cost`,
			expectIdentifier: true,
		},
		{
			name:             "Property ge Property",
			filter:           "Price ge Cost",
			expectedSQL:      `price >= cost`,
			expectIdentifier: true,
		},
		{
			name:             "Property lt Property",
			filter:           "Price lt Cost",
			expectedSQL:      `price < cost`,
			expectIdentifier: true,
		},
		{
			name:             "Property le Property",
			filter:           "Price le Cost",
			expectedSQL:      `price <= cost`,
			expectIdentifier: true,
		},
		{
			name:             "Property eq Property",
			filter:           "Price eq Cost",
			expectedSQL:      `price = cost`,
			expectIdentifier: true,
		},
		{
			name:             "Property ne Property",
			filter:           "Price ne Cost",
			expectedSQL:      `price != cost`,
			expectIdentifier: true,
		},
		{
			name:             "Property gt literal should use placeholder",
			filter:           "Price gt 100",
			expectedSQL:      `price > ?`,
			expectIdentifier: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, meta, nil)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			// Build SQL for SQLite dialect
			sql, args := buildFilterCondition("sqlite", filterExpr, meta)

			t.Logf("Generated SQL: %s", sql)
			t.Logf("Args: %v", args)

			if tt.expectIdentifier {
				// For property-to-property comparisons, we should have no args
				if len(args) > 0 {
					t.Errorf("Expected no args for property-to-property comparison, got %d args: %v", len(args), args)
				}
				// SQL should contain both column names
				if sql != tt.expectedSQL {
					t.Errorf("Expected SQL: %s, got: %s", tt.expectedSQL, sql)
				}
			} else {
				// For property-to-literal comparisons, we should have args
				if len(args) == 0 {
					t.Errorf("Expected args for property-to-literal comparison, got none")
				}
				if sql != tt.expectedSQL {
					t.Errorf("Expected SQL: %s, got: %s", tt.expectedSQL, sql)
				}
			}
		})
	}
}

// TestPropertyToPropertyComparisonParsing tests that the parser correctly
// identifies property names on the RHS
func TestPropertyToPropertyComparisonParsing(t *testing.T) {
	meta := getTestMetadata2(t)

	filterExpr, err := parseFilter("Price gt Cost", meta, nil)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	// The value should be the string "Cost" (property name)
	if filterExpr.Property != "Price" {
		t.Errorf("Expected property 'Price', got '%s'", filterExpr.Property)
	}

	valueStr, ok := filterExpr.Value.(string)
	if !ok {
		t.Fatalf("Expected value to be a string (property name), got %T", filterExpr.Value)
	}

	if valueStr != "Cost" {
		t.Errorf("Expected value 'Cost', got '%s'", valueStr)
	}

	if filterExpr.Operator != OpGreaterThan {
		t.Errorf("Expected operator gt, got %s", filterExpr.Operator)
	}
}
