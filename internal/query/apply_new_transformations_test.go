package query

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestParseApply_GroupByAll tests parsing of groupby(($all)) – the OData "aggregate all
// values without grouping" variant of groupby.
func TestParseApply_GroupByAll(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		applyStr  string
		expectErr bool
		validate  func(*testing.T, []ApplyTransformation)
	}{
		{
			name:     "groupby($all) without nested transform",
			applyStr: "groupby(($all))",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Fatalf("expected 1 transformation, got %d", len(trans))
				}
				if trans[0].Type != ApplyTypeGroupBy {
					t.Fatalf("expected groupby type, got %q", trans[0].Type)
				}
				g := trans[0].GroupBy
				if g == nil {
					t.Fatal("GroupBy is nil")
				}
				if !g.AllValues {
					t.Error("expected AllValues to be true")
				}
				if len(g.Properties) != 0 {
					t.Errorf("expected no properties, got %v", g.Properties)
				}
			},
		},
		{
			name:     "groupby($all) with aggregate",
			applyStr: "groupby(($all),aggregate(Price with sum as Total))",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Fatalf("expected 1 transformation, got %d", len(trans))
				}
				g := trans[0].GroupBy
				if g == nil {
					t.Fatal("GroupBy is nil")
				}
				if !g.AllValues {
					t.Error("expected AllValues to be true")
				}
				if len(g.Transform) != 1 {
					t.Fatalf("expected 1 nested transform, got %d", len(g.Transform))
				}
				if g.Transform[0].Type != ApplyTypeAggregate {
					t.Fatalf("expected aggregate nested transform, got %q", g.Transform[0].Type)
				}
			},
		},
		{
			name:     "groupby($ALL) case-insensitive",
			applyStr: "groupby(($ALL),aggregate(Price with sum as Total))",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				g := trans[0].GroupBy
				if g == nil {
					t.Fatal("GroupBy is nil")
				}
				if !g.AllValues {
					t.Error("expected AllValues to be true for uppercase $ALL")
				}
			},
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

// TestApplyGroupByAll_SQL verifies that groupby(($all)) produces a query
// without a GROUP BY clause but with the aggregate SELECT expressions.
func TestApplyGroupByAll_SQL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}

	if err := db.AutoMigrate(&ApplyTestEntity{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Seed some data
	entities := []ApplyTestEntity{
		{ID: 1, Name: "A", Category: "X", Price: 10.0, Quantity: 2},
		{ID: 2, Name: "B", Category: "X", Price: 20.0, Quantity: 3},
		{ID: 3, Name: "C", Category: "Y", Price: 30.0, Quantity: 1},
	}
	if err := db.Create(&entities).Error; err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	meta := getApplyTestMetadata(t)

	// groupby(($all), aggregate(Price with sum as Total)) should return a single row
	// with the sum of all prices (60.0) regardless of category.
	applyStr := "groupby(($all),aggregate(Price with sum as Total))"
	transformations, err := parseApply(applyStr, meta, 0)
	if err != nil {
		t.Fatalf("parseApply failed: %v", err)
	}

	resultDB := applyTransformations(db.Session(&gorm.Session{}), transformations, meta)

	var results []map[string]interface{}
	if err := resultDB.Find(&results).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result row for $all aggregation, got %d", len(results))
	}

	// Verify the Total field is the sum of all prices
	totalRaw, ok := results[0]["Total"]
	if !ok {
		t.Fatalf("expected 'Total' key in result, got keys: %v", results[0])
	}

	total, err := toFloat64Safe(totalRaw)
	if err != nil {
		t.Fatalf("failed to convert Total to float64: %v (%T)", totalRaw, totalRaw)
	}
	if total != 60.0 {
		t.Errorf("expected Total=60.0, got %v", total)
	}
}

// TestApplyGroupByAll_NoTransform_CountAll verifies that groupby(($all)) without
// a nested transform produces a single row with a $count aggregate.
func TestApplyGroupByAll_NoTransform_CountAll(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}

	if err := db.AutoMigrate(&ApplyTestEntity{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	entities := []ApplyTestEntity{
		{ID: 1, Name: "A", Category: "X", Price: 10.0, Quantity: 2},
		{ID: 2, Name: "B", Category: "X", Price: 20.0, Quantity: 3},
		{ID: 3, Name: "C", Category: "Y", Price: 30.0, Quantity: 1},
	}
	if err := db.Create(&entities).Error; err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	meta := getApplyTestMetadata(t)

	applyStr := "groupby(($all))"
	transformations, err := parseApply(applyStr, meta, 0)
	if err != nil {
		t.Fatalf("parseApply failed: %v", err)
	}

	resultDB := applyTransformations(db.Session(&gorm.Session{}), transformations, meta)

	var results []map[string]interface{}
	if err := resultDB.Find(&results).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result row, got %d", len(results))
	}

	countRaw, ok := results[0]["$count"]
	if !ok {
		t.Fatalf("expected '$count' key in result, got keys: %v", results[0])
	}

	count, err := toFloat64Safe(countRaw)
	if err != nil {
		t.Fatalf("failed to convert $count to float64: %v (%T)", countRaw, countRaw)
	}
	if count != 3.0 {
		t.Errorf("expected $count=3, got %v", count)
	}
}

// TestParseApply_Nest tests parsing of the nest($apply=...) transformation.
func TestParseApply_Nest(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		applyStr  string
		expectErr bool
		validate  func(*testing.T, []ApplyTransformation)
	}{
		{
			name:     "basic nest with aggregate",
			applyStr: "nest($apply=aggregate(Price with sum as Total))",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Fatalf("expected 1 transformation, got %d", len(trans))
				}
				if trans[0].Type != ApplyTypeNest {
					t.Fatalf("expected nest type, got %q", trans[0].Type)
				}
				n := trans[0].Nest
				if n == nil {
					t.Fatal("Nest is nil")
				}
				if len(n.Apply) != 1 {
					t.Fatalf("expected 1 inner transformation, got %d", len(n.Apply))
				}
				if n.Apply[0].Type != ApplyTypeAggregate {
					t.Fatalf("expected inner aggregate, got %q", n.Apply[0].Type)
				}
				if n.Alias != "" {
					t.Errorf("expected no alias, got %q", n.Alias)
				}
			},
		},
		{
			name:     "nest with groupby inner transformation",
			applyStr: "nest($apply=groupby((Category),aggregate(Price with sum as Total)))",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Fatalf("expected 1 transformation, got %d", len(trans))
				}
				n := trans[0].Nest
				if n == nil {
					t.Fatal("Nest is nil")
				}
				if len(n.Apply) != 1 {
					t.Fatalf("expected 1 inner transformation, got %d", len(n.Apply))
				}
				if n.Apply[0].Type != ApplyTypeGroupBy {
					t.Fatalf("expected inner groupby, got %q", n.Apply[0].Type)
				}
			},
		},
		{
			name:     "nest with alias",
			applyStr: "nest($apply=aggregate(Price with sum as Total),CategoryTotals)",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Fatalf("expected 1 transformation, got %d", len(trans))
				}
				n := trans[0].Nest
				if n == nil {
					t.Fatal("Nest is nil")
				}
				if n.Alias != "CategoryTotals" {
					t.Errorf("expected alias 'CategoryTotals', got %q", n.Alias)
				}
			},
		},
		{
			name:      "nest missing $apply= prefix returns error",
			applyStr:  "nest(aggregate(Price with sum as Total))",
			expectErr: true,
		},
		{
			name:      "nest empty returns error",
			applyStr:  "nest()",
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

// TestParseApply_From tests parsing of the from(NavigationPath) transformation.
func TestParseApply_From(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name      string
		applyStr  string
		expectErr bool
		validate  func(*testing.T, []ApplyTransformation)
	}{
		{
			name:     "from with navigation property",
			applyStr: "from(Lines)",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Fatalf("expected 1 transformation, got %d", len(trans))
				}
				if trans[0].Type != ApplyTypeFrom {
					t.Fatalf("expected from type, got %q", trans[0].Type)
				}
				f := trans[0].From
				if f == nil {
					t.Fatal("From is nil")
				}
				if f.Path != "Lines" {
					t.Errorf("expected path 'Lines', got %q", f.Path)
				}
			},
		},
		{
			name:     "from with navigation path",
			applyStr: "from(Lines/SubItems)",
			validate: func(t *testing.T, trans []ApplyTransformation) {
				if len(trans) != 1 {
					t.Fatalf("expected 1 transformation, got %d", len(trans))
				}
				f := trans[0].From
				if f == nil {
					t.Fatal("From is nil")
				}
				if f.Path != "Lines/SubItems" {
					t.Errorf("expected path 'Lines/SubItems', got %q", f.Path)
				}
			},
		},
		{
			name:      "from empty returns error",
			applyStr:  "from()",
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

// TestApplyNest_Passthrough verifies that the nest transformation leaves the
// query set unchanged (pass-through behaviour) at the SQL-builder layer.
func TestApplyNest_Passthrough(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}

	if err := db.AutoMigrate(&ApplyTestEntity{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	entities := []ApplyTestEntity{
		{ID: 1, Name: "A", Category: "X", Price: 10.0, Quantity: 2},
		{ID: 2, Name: "B", Category: "Y", Price: 20.0, Quantity: 3},
	}
	if err := db.Create(&entities).Error; err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	meta := getApplyTestMetadata(t)

	// nest() is a pass-through at the SQL-builder level; the full set is returned.
	applyStr := "nest($apply=aggregate(Price with sum as Total))"
	transformations, err := parseApply(applyStr, meta, 0)
	if err != nil {
		t.Fatalf("parseApply failed: %v", err)
	}

	resultDB := applyTransformations(db.Session(&gorm.Session{}), transformations, meta)
	var results []ApplyTestEntity
	if err := resultDB.Find(&results).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}

	// Pass-through: all entities should be returned unchanged
	if len(results) != 2 {
		t.Errorf("expected 2 entities (pass-through), got %d", len(results))
	}
}

// TestApplyFrom_Passthrough verifies that the from transformation leaves the
// query set unchanged (pass-through behaviour) at the SQL-builder layer.
func TestApplyFrom_Passthrough(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}

	if err := db.AutoMigrate(&ApplyTestEntity{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	entities := []ApplyTestEntity{
		{ID: 1, Name: "A", Category: "X", Price: 10.0, Quantity: 2},
		{ID: 2, Name: "B", Category: "Y", Price: 20.0, Quantity: 3},
	}
	if err := db.Create(&entities).Error; err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	meta := getApplyTestMetadata(t)

	applyStr := "from(Lines)"
	transformations, err := parseApply(applyStr, meta, 0)
	if err != nil {
		t.Fatalf("parseApply failed: %v", err)
	}

	resultDB := applyTransformations(db.Session(&gorm.Session{}), transformations, meta)
	var results []ApplyTestEntity
	if err := resultDB.Find(&results).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}

	// Pass-through: all entities should be returned unchanged
	if len(results) != 2 {
		t.Errorf("expected 2 entities (pass-through), got %d", len(results))
	}
}

// TestParseApply_CaseInsensitive_NewKeywords verifies that nest and from are
// recognized case-insensitively when caseInsensitive=true.
func TestParseApply_CaseInsensitive_NewKeywords(t *testing.T) {
	meta := getApplyTestMetadata(t)

	tests := []struct {
		name         string
		applyStr     string
		expectedType ApplyTransformationType
	}{
		{
			name:         "NEST uppercase",
			applyStr:     "NEST($apply=aggregate(Price with sum as Total))",
			expectedType: ApplyTypeNest,
		},
		{
			name:         "FROM uppercase",
			applyStr:     "FROM(Lines)",
			expectedType: ApplyTypeFrom,
		},
		{
			name:         "Nest mixed case",
			applyStr:     "Nest($apply=aggregate(Price with sum as Total))",
			expectedType: ApplyTypeNest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans, err := parseApply(tt.applyStr, meta, 0)
			if err != nil {
				t.Fatalf("parseApply(%q) failed: %v", tt.applyStr, err)
			}
			if len(trans) != 1 {
				t.Fatalf("expected 1 transformation, got %d", len(trans))
			}
			if trans[0].Type != tt.expectedType {
				t.Fatalf("expected type %q, got %q", tt.expectedType, trans[0].Type)
			}
		})
	}
}

// toFloat64Safe converts various numeric types returned by SQLite/GORM to float64.
func toFloat64Safe(v interface{}) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case int32:
		return float64(x), nil
	default:
		return 0, nil
	}
}
