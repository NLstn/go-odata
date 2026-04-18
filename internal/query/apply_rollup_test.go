package query

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ---------------------------------------------------------------------------
// Parser tests
// ---------------------------------------------------------------------------

func TestParseApply_Rollup(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		applyStr  string
		expectErr bool
		validate  func(*testing.T, []ApplyTransformation)
	}{
		{
			name:     "rollup with null (grand total)",
			applyStr: "groupby((rollup(null,Category)),aggregate(Price with sum as Total))",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Fatalf("expected 1 transformation, got %d", len(trans))
				}
				g := trans[0].GroupBy
				if g == nil {
					t.Fatal("GroupBy is nil")
				}
				if g.Rollup == nil {
					t.Fatal("expected Rollup spec, got nil")
				}
				if !g.Rollup.IncludeGrandTotal {
					t.Error("expected IncludeGrandTotal to be true")
				}
				if len(g.Rollup.Properties) != 1 || g.Rollup.Properties[0] != "Category" {
					t.Errorf("expected rollup property 'Category', got %v", g.Rollup.Properties)
				}
				if len(g.Transform) != 1 || g.Transform[0].Type != ApplyTypeAggregate {
					t.Error("expected one nested aggregate transform")
				}
			},
		},
		{
			name:     "rollup without null",
			applyStr: "groupby((rollup(Category)),aggregate(Price with sum as Total))",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				g := trans[0].GroupBy
				if g.Rollup == nil {
					t.Fatal("expected Rollup spec, got nil")
				}
				if g.Rollup.IncludeGrandTotal {
					t.Error("expected IncludeGrandTotal to be false")
				}
				if len(g.Rollup.Properties) != 1 || g.Rollup.Properties[0] != "Category" {
					t.Errorf("expected rollup property 'Category', got %v", g.Rollup.Properties)
				}
			},
		},
		{
			name:     "rollup with multiple properties and null",
			applyStr: "groupby((rollup(null,Category,Name)),aggregate(Price with sum as Total))",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				g := trans[0].GroupBy
				if g.Rollup == nil {
					t.Fatal("expected Rollup spec, got nil")
				}
				if !g.Rollup.IncludeGrandTotal {
					t.Error("expected IncludeGrandTotal to be true")
				}
				if len(g.Rollup.Properties) != 2 {
					t.Fatalf("expected 2 rollup properties, got %d", len(g.Rollup.Properties))
				}
				if g.Rollup.Properties[0] != "Category" || g.Rollup.Properties[1] != "Name" {
					t.Errorf("unexpected rollup properties: %v", g.Rollup.Properties)
				}
			},
		},
		{
			name:     "rollup with regular property and rollup property",
			applyStr: "groupby((Category,rollup(null,Name)),aggregate(Price with sum as Total))",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				g := trans[0].GroupBy
				if g.Rollup == nil {
					t.Fatal("expected Rollup spec, got nil")
				}
				if len(g.Properties) != 1 || g.Properties[0] != "Category" {
					t.Errorf("expected regular property 'Category', got %v", g.Properties)
				}
				if len(g.Rollup.Properties) != 1 || g.Rollup.Properties[0] != "Name" {
					t.Errorf("expected rollup property 'Name', got %v", g.Rollup.Properties)
				}
				if !g.Rollup.IncludeGrandTotal {
					t.Error("expected IncludeGrandTotal to be true")
				}
			},
		},
		{
			name:     "ROLLUP uppercase (case-insensitive)",
			applyStr: "groupby((ROLLUP(null,Category)),aggregate(Price with sum as Total))",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				g := trans[0].GroupBy
				if g.Rollup == nil {
					t.Fatal("expected Rollup spec, got nil")
				}
				if !g.Rollup.IncludeGrandTotal {
					t.Error("expected IncludeGrandTotal to be true")
				}
			},
		},
		{
			name:      "rollup with no properties",
			applyStr:  "groupby((rollup()),aggregate(Price with sum as Total))",
			expectErr: true,
		},
		{
			name:      "rollup with only null",
			applyStr:  "groupby((rollup(null)),aggregate(Price with sum as Total))",
			expectErr: true,
		},
		{
			name:      "rollup with unknown property",
			applyStr:  "groupby((rollup(null,NonExistent)),aggregate(Price with sum as Total))",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans, err := parseApply(tt.applyStr, meta, 0)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseApply(%q) failed: %v", tt.applyStr, err)
			}
			if tt.validate != nil {
				tt.validate(t, trans)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SQL generation tests
// ---------------------------------------------------------------------------

func TestApplyGroupByRollup_SQL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}

	if err := db.AutoMigrate(&ApplyTestEntity{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Seed data: two categories, multiple items each
	entities := []ApplyTestEntity{
		{ID: 1, Name: "A", Category: "X", Price: 10.0, Quantity: 1},
		{ID: 2, Name: "B", Category: "X", Price: 20.0, Quantity: 1},
		{ID: 3, Name: "C", Category: "Y", Price: 30.0, Quantity: 1},
		{ID: 4, Name: "D", Category: "Y", Price: 40.0, Quantity: 1},
	}
	if err := db.Create(&entities).Error; err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	meta := getApplyTestMetadata(t)

	// applyRollupPipeline simulates what the handler does:
	// 1. Apply transformations (stores rollup marker, no GROUP BY at SQL level)
	// 2. Fetch raw rows
	// 3. Apply in-memory rollup
	applyRollupPipeline := func(t *testing.T, applyStr string) []map[string]interface{} {
		t.Helper()
		transformations, err := parseApply(applyStr, meta, 0)
		if err != nil {
			t.Fatalf("parseApply(%q) failed: %v", applyStr, err)
		}

		resultDB := applyTransformations(db.Session(&gorm.Session{}), transformations, meta)

		// Fetch raw rows (no GROUP BY because rollup uses in-memory path)
		var rawResults []map[string]interface{}
		if err := resultDB.Find(&rawResults).Error; err != nil {
			t.Fatalf("query failed: %v", err)
		}

		// Check for rollup marker and apply in-memory aggregation
		rollupGroupBy, ok := GetRollupGroupByFromDB(resultDB)
		if !ok {
			t.Fatalf("expected rollup GroupBy to be stored in GORM context")
		}

		// Simulate the in-memory rollup processing done by the handler
		// We call applyMapGroupBy indirectly by using the rollup execution
		var results []map[string]interface{}
		results, err = applyMapRollup(rawResults, rollupGroupBy)
		if err != nil {
			t.Fatalf("in-memory rollup failed: %v", err)
		}
		return results
	}

	t.Run("rollup with null produces grand total row", func(t *testing.T) {
		applyStr := "groupby((rollup(null,Category)),aggregate(Price with sum as Total))"
		results := applyRollupPipeline(t, applyStr)

		// Expect: row for X (30), row for Y (70), grand total (100)
		if len(results) != 3 {
			t.Fatalf("expected 3 rows (2 categories + grand total), got %d: %v", len(results), results)
		}

		// Find grand total row (Category is nil/null)
		var grandTotalFound bool
		var grandTotal float64
		for _, row := range results {
			if row["Category"] == nil {
				grandTotalFound = true
				grandTotal = toFloat64(row["Total"])
			}
		}
		if !grandTotalFound {
			t.Error("expected a grand total row with Category=nil")
		}
		if grandTotal != 100.0 {
			t.Errorf("expected grand total = 100.0, got %v", grandTotal)
		}
	})

	t.Run("rollup without null excludes grand total row", func(t *testing.T) {
		applyStr := "groupby((rollup(Category)),aggregate(Price with sum as Total))"
		results := applyRollupPipeline(t, applyStr)

		// Expect: row for X (30), row for Y (70) — no grand total
		if len(results) != 2 {
			t.Fatalf("expected 2 rows (2 categories, no grand total), got %d: %v", len(results), results)
		}
		for _, row := range results {
			if row["Category"] == nil {
				t.Error("unexpected grand total row with Category=nil (rollup without null should not include grand total)")
			}
		}
	})

	t.Run("rollup with multiple properties and null", func(t *testing.T) {
		applyStr := "groupby((rollup(null,Category,Name)),aggregate(Price with sum as Total))"
		results := applyRollupPipeline(t, applyStr)

		// Expected levels:
		// (X, A, 10), (X, B, 20), (Y, C, 30), (Y, D, 40) — fine-grain rows
		// (X, null, 30), (Y, null, 70) — category subtotals
		// (null, null, 100) — grand total
		if len(results) != 7 {
			t.Fatalf("expected 7 rows, got %d: %v", len(results), results)
		}
	})

	t.Run("rollup with regular property (always grouped) and rollup property", func(t *testing.T) {
		// groupby((Category, rollup(null, Name)), aggregate(...))
		// Category is always grouped; Name is rolled up.
		// Levels: (X,A), (X,B), (Y,C), (Y,D) — fine-grain
		//         (X, null), (Y, null) — subtotals per category
		//         (X, null) grand-total of X, (Y, null) grand-total of Y — but wait,
		//         because Category is a regular (non-rollup) prop, there's no all-null row.
		applyStr := "groupby((Category,rollup(null,Name)),aggregate(Price with sum as Total))"
		results := applyRollupPipeline(t, applyStr)

		// 4 fine-grain rows + 2 subtotals (one per category)
		// No extra grand total since Category is not in rollup
		if len(results) != 6 {
			t.Fatalf("expected 6 rows, got %d: %v", len(results), results)
		}
	})
}

// applyMapRollup is a thin wrapper used in tests to call the rollup in-memory processing.
// In production this is invoked by the handler via applyMapGroupByRollup.
func applyMapRollup(results []map[string]interface{}, groupBy *GroupByTransformation) ([]map[string]interface{}, error) {
	// Import the handler function by calling the same logic — reproduce the helper here
	// since we're testing the query package directly.
	rollup := groupBy.Rollup
	allProps := append(groupBy.Properties, rollup.Properties...) //nolint:gocritic

	var levels [][]string
	for i := 0; i <= len(rollup.Properties); i++ {
		numRollup := len(rollup.Properties) - i
		level := make([]string, 0, len(groupBy.Properties)+numRollup)
		level = append(level, groupBy.Properties...)
		level = append(level, rollup.Properties[:numRollup]...)
		levels = append(levels, level)
	}
	if !rollup.IncludeGrandTotal {
		levels = levels[:len(levels)-1]
	}

	output := make([]map[string]interface{}, 0)
	for _, levelProps := range levels {
		levelRows, err := groupByPropsInMemory(results, levelProps, allProps, groupBy.Transform)
		if err != nil {
			return nil, err
		}
		output = append(output, levelRows...)
	}
	return output, nil
}

// groupByPropsInMemory is a test helper that mirrors the handler's applyMapGroupByForProps.
func groupByPropsInMemory(results []map[string]interface{}, levelProps []string, allProps []string, transforms []ApplyTransformation) ([]map[string]interface{}, error) {
	groups := make(map[string][]map[string]interface{})
	groupOrder := make([]string, 0)
	keyRowsByKey := make(map[string]map[string]interface{})
	levelPropSet := make(map[string]bool)
	for _, p := range levelProps {
		levelPropSet[strings.ToLower(p)] = true
	}

	getVal := func(row map[string]interface{}, prop string) (interface{}, bool) {
		lc := strings.ToLower(prop)
		for k, v := range row {
			if strings.ToLower(k) == lc {
				return v, true
			}
		}
		return nil, false
	}

	for _, row := range results {
		keyParts := make([]string, 0, len(levelProps))
		keyRow := make(map[string]interface{})
		for _, prop := range levelProps {
			val, _ := getVal(row, prop)
			keyParts = append(keyParts, fmt.Sprintf("%v=%v", prop, val))
			keyRow[prop] = val
		}
		key := strings.Join(keyParts, "|")
		if _, exists := groups[key]; !exists {
			groupOrder = append(groupOrder, key)
			keyRowsByKey[key] = keyRow
		}
		groups[key] = append(groups[key], row)
	}

	output := make([]map[string]interface{}, 0, len(groups))
	for _, key := range groupOrder {
		groupRows := groups[key]
		outRow := make(map[string]interface{})
		for k, v := range keyRowsByKey[key] {
			outRow[k] = v
		}
		for _, prop := range allProps {
			if !levelPropSet[strings.ToLower(prop)] {
				outRow[prop] = nil
			}
		}

		if len(transforms) > 0 {
			for _, tr := range transforms {
				if tr.Type == ApplyTypeAggregate && tr.Aggregate != nil {
					for _, aggExpr := range tr.Aggregate.Expressions {
						var sumTotal float64
						for _, row := range groupRows {
							if val, ok := getVal(row, aggExpr.Property); ok {
								sumTotal += toFloat64(val)
							}
						}
						outRow[aggExpr.Alias] = sumTotal
					}
				}
			}
		}
		output = append(output, outRow)
	}
	return output, nil
}

// ---------------------------------------------------------------------------
// buildRollupGroupByClause tests
// ---------------------------------------------------------------------------

func TestBuildRollupGroupByClause(t *testing.T) {
	tests := []struct {
		name            string
		dialect         string
		regularColumns  []string
		rollupColumns   []string
		wantContains    string
		wantNotContains string
	}{
		{
			name:          "postgres with single rollup column",
			dialect:       "postgres",
			regularColumns: []string{},
			rollupColumns: []string{`"col1"`},
			wantContains:  `ROLLUP("col1")`,
		},
		{
			name:          "postgres with regular and rollup columns",
			dialect:       "postgres",
			regularColumns: []string{`"reg"`},
			rollupColumns: []string{`"col1"`, `"col2"`},
			wantContains:  `ROLLUP("col1", "col2")`,
		},
		{
			name:          "sqlite with single rollup column",
			dialect:       "sqlite",
			regularColumns: []string{},
			rollupColumns: []string{"`col1`"},
			wantContains:  "WITH ROLLUP",
		},
		{
			name:          "mysql with regular and rollup columns",
			dialect:       "mysql",
			regularColumns: []string{"`reg`"},
			rollupColumns: []string{"`col1`", "`col2`"},
			wantContains:  "WITH ROLLUP",
		},
		{
			name:          "sqlserver with rollup columns",
			dialect:       "sqlserver",
			regularColumns: []string{},
			rollupColumns: []string{"[col1]", "[col2]"},
			wantContains:  "ROLLUP([col1], [col2])",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRollupGroupByClause(tt.dialect, tt.regularColumns, tt.rollupColumns)
			if tt.wantContains != "" && !strings.Contains(result, tt.wantContains) {
				t.Errorf("expected clause to contain %q, got: %q", tt.wantContains, result)
			}
			if tt.wantNotContains != "" && strings.Contains(result, tt.wantNotContains) {
				t.Errorf("expected clause NOT to contain %q, got: %q", tt.wantNotContains, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ContextPropertiesFromApply tests for rollup
// ---------------------------------------------------------------------------

func TestContextPropertiesFromApply_Rollup(t *testing.T) {
	trans := []ApplyTransformation{
		{
			Type: ApplyTypeGroupBy,
			GroupBy: &GroupByTransformation{
				Properties: []string{"Category"},
				Rollup: &RollupSpec{
					Properties:        []string{"Name"},
					IncludeGrandTotal: true,
				},
				Transform: []ApplyTransformation{
					{
						Type: ApplyTypeAggregate,
						Aggregate: &AggregateTransformation{
							Expressions: []AggregateExpression{
								{Property: "Price", Method: AggregationSum, Alias: "Total"},
							},
						},
					},
				},
			},
		},
	}

	props := ContextPropertiesFromApply(trans)
	propSet := make(map[string]bool)
	for _, p := range props {
		propSet[p] = true
	}

	for _, expected := range []string{"Category", "Name", "Total"} {
		if !propSet[expected] {
			t.Errorf("expected property %q in context output, got: %v", expected, props)
		}
	}
}
