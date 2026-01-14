package query

import (
	"testing"
)

// TestMathFunctions_EndToEnd tests complete flow from OData query to SQL
func TestMathFunctions_EndToEnd(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name        string
		odataFilter string
		expectSQL   string
		expectArgs  int
	}{
		{
			name:        "ceiling filter",
			odataFilter: "ceiling(Price) gt 100",
			expectSQL:   "CASE WHEN price = CAST(price AS INTEGER) THEN price ELSE CAST(price AS INTEGER) + (CASE WHEN price > 0 THEN 1 ELSE 0 END) END > ?",
			expectArgs:  1,
		},
		{
			name:        "floor filter",
			odataFilter: "floor(Price) lt 50",
			expectSQL:   "CASE WHEN price = CAST(price AS INTEGER) THEN price ELSE CAST(price AS INTEGER) - (CASE WHEN price < 0 THEN 1 ELSE 0 END) END < ?",
			expectArgs:  1,
		},
		{
			name:        "round filter",
			odataFilter: "round(Price) eq 100",
			expectSQL:   "ROUND(price) = ?",
			expectArgs:  1,
		},
		{
			name:        "combined math and comparison",
			odataFilter: "ceiling(Price) gt 100 and Name eq 'Laptop'",
			expectSQL:   "(CASE WHEN price = CAST(price AS INTEGER) THEN price ELSE CAST(price AS INTEGER) + (CASE WHEN price > 0 THEN 1 ELSE 0 END) END > ?) AND (name = ?)",
			expectArgs:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the OData filter
			filterExpr, err := parseFilter(tt.odataFilter, meta, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			// Build SQL
			sql, args := buildFilterCondition("sqlite", filterExpr, meta)

			if sql != tt.expectSQL {
				t.Errorf("Expected SQL:\n%s\nGot:\n%s", tt.expectSQL, sql)
			}

			if len(args) != tt.expectArgs {
				t.Errorf("Expected %d args, got %d", tt.expectArgs, len(args))
			}

			t.Logf("✓ OData: %s", tt.odataFilter)
			t.Logf("✓ SQL:   %s", sql)
			t.Logf("✓ Args:  %v", args)
		})
	}
}
