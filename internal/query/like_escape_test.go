package query

import "testing"

func TestEscapeLikePattern(t *testing.T) {
	input := "%_\\"
	expected := `\%\_\\`

	result := escapeLikePattern(input)
	if result != expected {
		t.Fatalf("expected escaped pattern %q, got %q", expected, result)
	}
}

func TestGetLikeEscapeClause(t *testing.T) {
	tests := []struct {
		dialect  string
		expected string
	}{
		{"sqlite", "ESCAPE '\\'"},
		{"postgres", "ESCAPE '\\'"},
		{"mysql", "ESCAPE '\\\\'"},
		{"sqlserver", "ESCAPE '\\'"},
	}

	for _, tc := range tests {
		t.Run(tc.dialect, func(t *testing.T) {
			result := getLikeEscapeClause(tc.dialect)
			if result != tc.expected {
				t.Fatalf("dialect %q: expected %q, got %q", tc.dialect, tc.expected, result)
			}
		})
	}
}

func TestBuildFilterCondition_LikeEscapes(t *testing.T) {
	meta := getTestMetadata(t)
		filterExpr, err := parseFilter("contains(Name, '%_')", meta, nil, 0)
	if err != nil {
		t.Fatalf("parseFilter failed: %v", err)
	}

	sql, args := buildFilterCondition("sqlite", filterExpr, meta)
	expectedSQL := "name LIKE ? ESCAPE '\\'"
	if sql != expectedSQL {
		t.Fatalf("expected SQL %q, got %q", expectedSQL, sql)
	}

	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}

	expectedArg := "%\\%\\_%"
	if args[0] != expectedArg {
		t.Fatalf("expected arg %q, got %q", expectedArg, args[0])
	}
}

func TestBuildFilterCondition_LikeEscapes_MySQL(t *testing.T) {
	meta := getTestMetadata(t)
	filterExpr, err := parseFilter("contains(Name, '%_')", meta, nil, 0)
	if err != nil {
		t.Fatalf("parseFilter failed: %v", err)
	}

	sql, args := buildFilterCondition("mysql", filterExpr, meta)
	expectedSQL := "name LIKE ? ESCAPE '\\\\'"
	if sql != expectedSQL {
		t.Fatalf("expected SQL %q, got %q", expectedSQL, sql)
	}

	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}

	expectedArg := "%\\%\\_%"
	if args[0] != expectedArg {
		t.Fatalf("expected arg %q, got %q", expectedArg, args[0])
	}
}

func TestBuildLikeComparison_SQLite(t *testing.T) {
	sql, args := buildLikeComparison("sqlite", "title", "%_", false, true)
	expectedSQL := "title LIKE ? ESCAPE '\\'"
	if sql != expectedSQL {
		t.Fatalf("expected SQL %q, got %q", expectedSQL, sql)
	}

	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}

	expectedArg := "\\%\\_%"
	if args[0] != expectedArg {
		t.Fatalf("expected arg %q, got %q", expectedArg, args[0])
	}
}

func TestBuildLikeComparison_MySQL(t *testing.T) {
	sql, args := buildLikeComparison("mysql", "title", "%_", false, true)
	expectedSQL := "title LIKE ? ESCAPE '\\\\'"
	if sql != expectedSQL {
		t.Fatalf("expected SQL %q, got %q", expectedSQL, sql)
	}

	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}

	expectedArg := "\\%\\_%"
	if args[0] != expectedArg {
		t.Fatalf("expected arg %q, got %q", expectedArg, args[0])
	}
}
