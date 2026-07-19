package fastscan

import (
	"reflect"
	"testing"

	"gorm.io/gorm"
)

// findMapBoth runs the same query through fastscan.FindMap and GORM's Find into
// a []map[string]interface{} and requires identical results, so FindMap can
// never diverge from the maps the handlers previously received.
func findMapBoth(t *testing.T, build func() *gorm.DB) []map[string]interface{} {
	t.Helper()
	var fast []map[string]interface{}
	if err := FindMap(build(), &fast); err != nil {
		t.Fatalf("fastscan.FindMap: %v", err)
	}
	var slow []map[string]interface{}
	if err := build().Find(&slow).Error; err != nil {
		t.Fatalf("gorm Find: %v", err)
	}
	if !reflect.DeepEqual(fast, slow) {
		t.Fatalf("FindMap result differs from GORM:\nfast: %#v\ngorm: %#v", fast, slow)
	}
	return fast
}

// TestFindMapMatchesGormEntityColumns covers a plain projection of entity
// columns, exercising every scan mode plus NULLs, pointer fields, named types,
// []byte, time.Time, and a driver.Valuer field — all of which GORM's scanIntoMap
// normalizes and FindMap must reproduce byte-for-byte.
func TestFindMapMatchesGormEntityColumns(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	result := findMapBoth(t, func() *gorm.DB {
		return db.Model(&Widget{}).Order("id")
	})
	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
}

// TestFindMapCompute mirrors the $compute shape: all entity columns plus a
// computed alias that is not a schema field, so it scans through the
// new(interface{}) branch rather than a typed field pointer.
func TestFindMapCompute(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	result := findMapBoth(t, func() *gorm.DB {
		return db.Model(&Widget{}).
			Select("*, price * 2 AS double_price").
			Order("id")
	})
	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
	if _, ok := result[0]["double_price"]; !ok {
		t.Fatalf("computed alias missing: %#v", result[0])
	}
}

// TestFindMapApplyGroupBy mirrors the $apply groupby/aggregate shape: a grouping
// key that maps to a schema field (Status) plus an aggregate alias that does not.
func TestFindMapApplyGroupBy(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	findMapBoth(t, func() *gorm.DB {
		return db.Model(&Widget{}).
			Select("status, COUNT(*) AS cnt, AVG(price) AS avg_price").
			Group("status").
			Order("status")
	})
}

// TestFindMapEmptyResult confirms an empty result behaves exactly like GORM's
// map scan — which leaves the destination untouched (nil) — so downstream
// serialization sees the same value it did before.
func TestFindMapEmptyResult(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	result := findMapBoth(t, func() *gorm.DB {
		return db.Model(&Widget{}).Where("1 = 0")
	})
	if len(result) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(result))
	}
}

// TestFindMapNoModel covers the schema-less path: a raw table query with no
// model set, where columns are typed from the driver's ScanType instead of
// schema fields. FindMap must still match GORM.
func TestFindMapNoModel(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	findMapBoth(t, func() *gorm.DB {
		return db.Table("widgets").Select("id, name, price").Order("id")
	})
}

// TestFindMapReusesBackingArray confirms FindMap appends into a caller-provided
// slice the way GORM's map scan does.
func TestFindMapReusesBackingArray(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	results := make([]map[string]interface{}, 0, 8)
	if err := FindMap(db.Model(&Widget{}).Order("id"), &results); err != nil {
		t.Fatalf("FindMap: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(results))
	}
}
