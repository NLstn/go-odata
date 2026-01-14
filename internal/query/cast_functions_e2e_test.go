package query

import (
	"testing"
)

// TestCastFunctions_EndToEnd tests complete flow from OData query to SQL
func TestCastFunctions_EndToEnd(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name        string
		odataFilter string
		expectSQL   string
		expectArgs  int
	}{
		{
			name:        "cast to string filter",
			odataFilter: "cast(Price, 'Edm.String') eq '100'",
			expectSQL:   "CAST(price AS TEXT) = ?",
			expectArgs:  1,
		},
		{
			name:        "cast to integer filter",
			odataFilter: "cast(Price, 'Edm.Int32') gt 100",
			expectSQL:   "CAST(price AS INTEGER) > ?",
			expectArgs:  1,
		},
		{
			name:        "cast to decimal filter",
			odataFilter: "cast(Price, 'Edm.Decimal') lt 99.99",
			expectSQL:   "CAST(price AS REAL) < ?",
			expectArgs:  1,
		},
		{
			name:        "cast to double filter",
			odataFilter: "cast(Price, 'Edm.Double') ge 50.5",
			expectSQL:   "CAST(price AS REAL) >= ?",
			expectArgs:  1,
		},
		{
			name:        "cast to int64 filter",
			odataFilter: "cast(Price, 'Edm.Int64') le 1000",
			expectSQL:   "CAST(price AS INTEGER) <= ?",
			expectArgs:  1,
		},
		{
			name:        "cast to boolean filter",
			odataFilter: "cast(Price, 'Edm.Boolean') eq true",
			expectSQL:   "CAST(price AS INTEGER) = ?",
			expectArgs:  1,
		},
		{
			name:        "cast with AND operator",
			odataFilter: "cast(Price, 'Edm.Int32') gt 100 and Name eq 'Laptop'",
			expectSQL:   "(CAST(price AS INTEGER) > ?) AND (name = ?)",
			expectArgs:  2,
		},
		{
			name:        "cast with OR operator",
			odataFilter: "cast(Price, 'Edm.String') eq '100' or Category eq 'Electronics'",
			expectSQL:   "(CAST(price AS TEXT) = ?) OR (category = ?)",
			expectArgs:  2,
		},
		{
			name:        "multiple casts",
			odataFilter: "cast(Price, 'Edm.Int32') gt 50 and cast(Price, 'Edm.Int32') lt 150",
			expectSQL:   "(CAST(price AS INTEGER) > ?) AND (CAST(price AS INTEGER) < ?)",
			expectArgs:  2,
		},
		{
			name:        "cast with complex expression",
			odataFilter: "(cast(Price, 'Edm.Decimal') gt 100.0 and Name eq 'Test') or Category eq 'Books'",
			expectSQL:   "((CAST(price AS REAL) > ?) AND (name = ?)) OR (category = ?)",
			expectArgs:  3,
		},
		{
			name:        "cast to DateTimeOffset",
			odataFilter: "cast(Name, 'Edm.DateTimeOffset') eq '2023-01-01'",
			expectSQL:   "CAST(name AS TEXT) = ?",
			expectArgs:  1,
		},
		{
			name:        "cast to Guid",
			odataFilter: "cast(ID, 'Edm.Guid') eq 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'",
			expectSQL:   "CAST(id AS TEXT) = ?",
			expectArgs:  1,
		},
		{
			name:        "cast with not equal",
			odataFilter: "cast(Price, 'Edm.Int32') ne 0",
			expectSQL:   "CAST(price AS INTEGER) != ?",
			expectArgs:  1,
		},
		{
			name:        "cast combined with string function",
			odataFilter: "tolower(Name) eq 'laptop' and cast(Price, 'Edm.Int32') gt 500",
			expectSQL:   "(LOWER(name) = ?) AND (CAST(price AS INTEGER) > ?)",
			expectArgs:  2,
		},
		{
			name:        "cast combined with math function",
			odataFilter: "ceiling(Price) gt 100 and cast(Name, 'Edm.String') eq 'Product'",
			expectSQL:   "(CASE WHEN price = CAST(price AS INTEGER) THEN price ELSE CAST(price AS INTEGER) + (CASE WHEN price > 0 THEN 1 ELSE 0 END) END > ?) AND (CAST(name AS TEXT) = ?)",
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

// TestCastFunctions_Integration tests cast function with different data types
func TestCastFunctions_Integration(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name        string
		odataFilter string
		expectSQL   string
		expectArgs  int
	}{
		{
			name:        "cast Int16",
			odataFilter: "cast(Price, 'Edm.Int16') eq 100",
			expectSQL:   "CAST(price AS INTEGER) = ?",
			expectArgs:  1,
		},
		{
			name:        "cast Byte",
			odataFilter: "cast(Price, 'Edm.Byte') gt 50",
			expectSQL:   "CAST(price AS INTEGER) > ?",
			expectArgs:  1,
		},
		{
			name:        "cast SByte",
			odataFilter: "cast(Price, 'Edm.SByte') lt 127",
			expectSQL:   "CAST(price AS INTEGER) < ?",
			expectArgs:  1,
		},
		{
			name:        "cast Single (float)",
			odataFilter: "cast(Price, 'Edm.Single') ge 99.9",
			expectSQL:   "CAST(price AS REAL) >= ?",
			expectArgs:  1,
		},
		{
			name:        "cast Date",
			odataFilter: "cast(Name, 'Edm.Date') eq '2023-01-01'",
			expectSQL:   "CAST(name AS TEXT) = ?",
			expectArgs:  1,
		},
		{
			name:        "cast TimeOfDay",
			odataFilter: "cast(Name, 'Edm.TimeOfDay') eq '12:00:00'",
			expectSQL:   "CAST(name AS TEXT) = ?",
			expectArgs:  1,
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
