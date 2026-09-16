package query

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/url"
	"strings"
	"testing"
)

func TestApplyAliasScopeDoesNotMutateModel(t *testing.T) {
	meta := getApplyTestMetadata(t)
	before := len(meta.Properties)
	for _, expr := range []string{"compute(Price mul 2 as Twice)/aggregate(Twice with sum as Total)", "compute(Price mul 2 as Twice)/concat(aggregate(Twice with sum as Total),aggregate(Twice with max as Total))"} {
		if _, err := parseApply(expr, meta, 0); err != nil {
			t.Fatal(err)
		}
	}
	if len(meta.Properties) != before || meta.FindProperty("Twice") != nil {
		t.Fatal("shared model mutated")
	}
	for _, expr := range []string{"concat(identity)", "compute(Price as Price)", "compute(Price as X,Price as X)", "filter(Twice gt 0)/compute(Price mul 2 as Twice)"} {
		if _, err := parseApply(expr, meta, 0); err == nil {
			t.Errorf("accepted %s", expr)
		}
	}
}
func TestApplySQLPreservesPageBeforeAggregation(t *testing.T) {
	meta := getApplyTestMetadata(t)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	opts, err := ParseQueryOptions(url.Values{"$apply": {"orderby(Price asc)/top(1)/aggregate(Price with sum as Total)"}}, meta)
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]interface{}
	result := ApplyQueryOptions(db, opts, meta, nil).Find(&rows)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	sql := result.Statement.SQL.String()
	if !strings.Contains(sql, "FROM (SELECT") || !strings.Contains(sql, "LIMIT 1)") {
		t.Fatalf("missing limited input relation: %s", sql)
	}
}
