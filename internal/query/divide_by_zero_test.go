package query

import (
	"strings"
	"testing"
)

// TestArithmeticDivideByZeroGuard verifies that every division and modulo
// divisor is wrapped in NULLIF so that division by zero yields NULL instead of
// raising a database error on PostgreSQL/SQL Server (SQLite and MySQL already
// return NULL). See nullIfZero.
func TestArithmeticDivideByZeroGuard(t *testing.T) {
	meta := getTestMetadata(t)

	dialects := []string{"sqlite", "postgres", "mysql", "mariadb", "sqlserver"}
	filters := []string{
		"Price div 0 gt 0",
		"Price divby 0 gt 0",
		"Price mod 0 eq 0",
		"ID div 0 eq 0",
		"ID mod 0 eq 0",
	}

	for _, dialect := range dialects {
		for _, filter := range filters {
			t.Run(dialect+"/"+filter, func(t *testing.T) {
				expr, err := parseFilter(filter, meta, nil, 0)
				if err != nil {
					t.Fatalf("parseFilter(%q) error: %v", filter, err)
				}
				sql, _ := buildFilterCondition(dialect, expr, meta)
				if !strings.Contains(sql, "NULLIF(") {
					t.Errorf("dialect %s filter %q: expected NULLIF-guarded divisor, got %q", dialect, filter, sql)
				}
			})
		}
	}
}

// TestArithmeticDivFloatingForDoubleOperand verifies that OData `div` performs
// IEEE 754 floating division when an operand is Edm.Double (Price), and integer
// division when both operands are integers (ID). Without the floating cast a
// database that stores Edm.Double as an exact decimal (e.g. PostgreSQL maps Go
// float64 to numeric) would compute decimal arithmetic that diverges from the
// double-precision semantics the type promises.
func TestArithmeticDivFloatingForDoubleOperand(t *testing.T) {
	meta := getTestMetadata(t)

	// Price is Edm.Double -> floating division (dividend cast to a float type).
	expr, err := parseFilter("Price div 3 eq 0", meta, nil, 0)
	if err != nil {
		t.Fatalf("parseFilter error: %v", err)
	}
	sql, _ := buildFilterCondition("postgres", expr, meta)
	if !strings.Contains(sql, "CAST(") {
		t.Errorf("Price div (Edm.Double): expected floating cast, got %q", sql)
	}

	// ID is an integer -> integer division, no cast.
	expr, err = parseFilter("ID div 3 eq 0", meta, nil, 0)
	if err != nil {
		t.Fatalf("parseFilter error: %v", err)
	}
	sql, _ = buildFilterCondition("postgres", expr, meta)
	if strings.Contains(sql, "CAST(") {
		t.Errorf("ID div (integer): expected integer division without cast, got %q", sql)
	}
}
