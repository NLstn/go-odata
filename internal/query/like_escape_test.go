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

func TestBuildFilterCondition_LikeEscapes(t *testing.T) {
	meta := getTestMetadata(t)
	filterExpr, err := parseFilter("contains(Name, '%_')", meta, nil)
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

func TestBuildSimpleFilterCondition_LikeEscapes(t *testing.T) {
	filter := &FilterExpression{
		Operator: OpStartsWith,
		Property: "Title",
		Value:    "%_",
	}

	sql, args := buildSimpleFilterCondition("sqlite", filter)
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
