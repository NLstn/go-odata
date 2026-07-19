package query

import (
	"strings"
	"testing"
)

// TestNonFiniteComparisonSQLServer verifies that comparisons against the IEEE 754
// special literals INF, -INF and NaN are folded to their constant truth value on
// SQL Server (whose float type cannot bind these values). A finite, non-NULL
// column matches `< INF`, `<= INF`, `> -INF`, `>= -INF` and the corresponding
// `!=` forms, and matches none of the impossible comparisons.
func TestNonFiniteComparisonSQLServer(t *testing.T) {
	meta := getTestMetadata(t)

	// matchesAll asserts the fold selects every finite non-NULL row (IS NOT NULL);
	// matchesNone asserts it selects nothing (1 = 0).
	cases := []struct {
		filter     string
		matchesAll bool
	}{
		{"Price lt INF", true},
		{"Price le INF", true},
		{"Price ne INF", true},
		{"Price gt INF", false},
		{"Price ge INF", false},
		{"Price eq INF", false},

		{"Price gt -INF", true},
		{"Price ge -INF", true},
		{"Price ne -INF", true},
		{"Price lt -INF", false},
		{"Price le -INF", false},
		{"Price eq -INF", false},

		{"Price ne NaN", true},
		{"Price eq NaN", false},
		{"Price lt NaN", false},
		{"Price gt NaN", false},
	}

	for _, tc := range cases {
		t.Run(tc.filter, func(t *testing.T) {
			expr, err := parseFilter(tc.filter, meta, nil, 0)
			if err != nil {
				t.Fatalf("parseFilter(%q) error: %v", tc.filter, err)
			}
			sql, args := buildFilterCondition("sqlserver", expr, meta)
			if len(args) != 0 {
				t.Errorf("%q: expected no bound args (non-finite value must not be bound), got %v", tc.filter, args)
			}
			isAll := strings.Contains(sql, "IS NOT NULL")
			isNone := strings.Contains(sql, "1 = 0")
			if tc.matchesAll && !isAll {
				t.Errorf("%q: expected IS NOT NULL (matches all finite rows), got %q", tc.filter, sql)
			}
			if !tc.matchesAll && !isNone {
				t.Errorf("%q: expected 1 = 0 (matches nothing), got %q", tc.filter, sql)
			}
		})
	}
}
