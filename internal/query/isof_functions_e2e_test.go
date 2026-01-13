package query

import (
	"testing"
)

// TestIsOfFunctions_EndToEnd tests the isof() function with end-to-end SQL generation
func TestIsOfFunctions_EndToEnd(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name           string
		filter         string
		expectErr      bool
		expectedSQL    string
		expectedArgsNo int
	}{
		{
			name:           "isof to string filter",
			filter:         "isof(Price, 'Edm.String') eq true",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS TEXT) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof to integer filter",
			filter:         "isof(Price, 'Edm.Int32') eq true",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS INTEGER) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof to decimal filter",
			filter:         "isof(Price, 'Edm.Decimal') eq false",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS REAL) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof to double filter",
			filter:         "isof(Price, 'Edm.Double') eq true",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS REAL) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof to int64 filter",
			filter:         "isof(Price, 'Edm.Int64') eq true",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS INTEGER) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof to boolean filter",
			filter:         "isof(Price, 'Edm.Boolean') eq false",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS INTEGER) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof with AND operator",
			filter:         "isof(Price, 'Edm.Int32') eq true and Name eq 'Laptop'",
			expectErr:      false,
			expectedSQL:    "(CASE WHEN CAST(price AS INTEGER) IS NOT NULL THEN 1 ELSE 0 END = ?) AND (name = ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "isof with OR operator",
			filter:         "isof(Price, 'Edm.String') eq false or Category eq 'Electronics'",
			expectErr:      false,
			expectedSQL:    "(CASE WHEN CAST(price AS TEXT) IS NOT NULL THEN 1 ELSE 0 END = ?) OR (category = ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "multiple isof calls",
			filter:         "isof(Price, 'Edm.Int32') eq true and isof(Name, 'Edm.String') eq true",
			expectErr:      false,
			expectedSQL:    "(CASE WHEN CAST(price AS INTEGER) IS NOT NULL THEN 1 ELSE 0 END = ?) AND (CASE WHEN CAST(name AS TEXT) IS NOT NULL THEN 1 ELSE 0 END = ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "isof with complex expression",
			filter:         "(isof(Price, 'Edm.Decimal') eq true and Name eq 'Test') or Category eq 'Books'",
			expectErr:      false,
			expectedSQL:    "((CASE WHEN CAST(price AS REAL) IS NOT NULL THEN 1 ELSE 0 END = ?) AND (name = ?)) OR (category = ?)",
			expectedArgsNo: 3,
		},
		{
			name:           "isof to DateTimeOffset",
			filter:         "isof(Name, 'Edm.DateTimeOffset') eq true",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(name AS TEXT) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof to Guid",
			filter:         "isof(ID, 'Edm.Guid') eq true",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(id AS TEXT) IS NOT NULL THEN 1 ELSE 0 END = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof with not equal",
			filter:         "isof(Price, 'Edm.String') ne false",
			expectErr:      false,
			expectedSQL:    "CASE WHEN CAST(price AS TEXT) IS NOT NULL THEN 1 ELSE 0 END != ?",
			expectedArgsNo: 1,
		},
		{
			name:           "isof with greater than",
			filter:         "isof(Price, 'Edm.Int32') eq true and Price gt 100",
			expectErr:      false,
			expectedSQL:    "(CASE WHEN CAST(price AS INTEGER) IS NOT NULL THEN 1 ELSE 0 END = ?) AND (price > ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "isof combined with cast",
			filter:         "isof(Price, 'Edm.Int32') eq true and cast(Price, 'Edm.String') eq '100'",
			expectErr:      false,
			expectedSQL:    "(CASE WHEN CAST(price AS INTEGER) IS NOT NULL THEN 1 ELSE 0 END = ?) AND (CAST(price AS TEXT) = ?)",
			expectedArgsNo: 2,
		},
		{
			name:           "isof with contains function",
			filter:         "contains(Name, 'Pro') and isof(Price, 'Edm.Decimal') eq true",
			expectErr:      false,
			expectedSQL:    "(name LIKE ? ESCAPE '\\') AND (CASE WHEN CAST(price AS REAL) IS NOT NULL THEN 1 ELSE 0 END = ?)",
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
			if sql != tt.expectedSQL {
				t.Errorf("Expected SQL:\n%s\nGot:\n%s", tt.expectedSQL, sql)
			}

			if len(args) != tt.expectedArgsNo {
				t.Errorf("Expected %d args, got %d", tt.expectedArgsNo, len(args))
			}

			t.Logf("✓ OData: %s", tt.filter)
			t.Logf("✓ SQL:   %s", sql)
			t.Logf("✓ Args:  %v", args)
		})
	}
}
