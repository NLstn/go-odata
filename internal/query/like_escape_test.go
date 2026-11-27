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
	if !sqlEquivalent(expectedSQL, sql) {
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
	expectedSQL := "BINARY name LIKE ? ESCAPE '\\\\'"
	if !sqlEquivalent(expectedSQL, sql) {
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
	expectedSQL := "BINARY title LIKE ? ESCAPE '\\\\'"
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

func TestBuildLikeComparison_MariaDB(t *testing.T) {
	sql, args := buildLikeComparison("mariadb", "title", "%_", false, true)
	expectedSQL := "BINARY title LIKE ? ESCAPE '\\'"
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

func TestBuildLikeComparison_SQLServer(t *testing.T) {
	sql, args := buildLikeComparison("sqlserver", "title", "%_", false, true)
	expectedSQL := "title COLLATE Latin1_General_BIN2 LIKE ? ESCAPE '\\'"
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

func TestBuildLikeComparison_Postgres(t *testing.T) {
	sql, args := buildLikeComparison("postgres", "title", "%_", false, true)
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

func TestBuildRegexComparison_SQLServerEREQuantifier(t *testing.T) {
	sql, args := buildRegexComparison("sqlserver", "name", "e+")
	if sql != "name LIKE ? ESCAPE '\\'" {
		t.Fatalf("unexpected SQL: %q", sql)
	}
	if len(args) != 1 || args[0] != "%e%" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildRegexComparison_SQLServerEREAlternation(t *testing.T) {
	sql, args := buildRegexComparison("sqlserver", "name", "Laptop|Coffee")
	expectedSQL := "(name LIKE ? ESCAPE '\\' OR name LIKE ? ESCAPE '\\')"
	if sql != expectedSQL {
		t.Fatalf("expected SQL %q, got %q", expectedSQL, sql)
	}
	if len(args) != 2 || args[0] != "%Laptop%" || args[1] != "%Coffee%" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestBuildFlaggedRegexComparison_CaseSensitivity(t *testing.T) {
	tests := []struct {
		dialect string
		flags   string
		wantSQL string
		wantArg string
	}{
		{"mysql", "", "name REGEXP ?", "(?-i:^Laptop$)"},
		{"mysql", "m", "name REGEXP ?", "(?m-i:^Laptop$)"},
		{"mysql", "i", "name REGEXP ?", "(?i:^Laptop$)"},
		{"mariadb", "", "name REGEXP ?", "(?-i:^Laptop$)"},
		{"mssql", "", "name COLLATE Latin1_General_BIN2 LIKE ? ESCAPE '\\'", "Laptop"},
		{"sqlserver", "i", "LOWER(name) COLLATE Latin1_General_BIN2 LIKE ? ESCAPE '\\'", "laptop"},
	}
	for _, tt := range tests {
		t.Run(tt.dialect+"/"+tt.flags, func(t *testing.T) {
			sql, args := buildRegexComparison(tt.dialect, "name", regexMatch{pattern: "^Laptop$", flags: tt.flags, valid: true})
			if sql != tt.wantSQL || len(args) != 1 || args[0] != tt.wantArg {
				t.Fatalf("got %q with args %v, want %q with arg %q", sql, args, tt.wantSQL, tt.wantArg)
			}
		})
	}
}
