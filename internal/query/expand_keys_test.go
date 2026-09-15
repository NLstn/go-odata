package query

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

type namedExpandKey int
type formattedExpandKey int

func (k formattedExpandKey) Format(s fmt.State, _ rune) {
	_, _ = fmt.Fprintf(s, "custom(%d)", int(k))
}

func legacyCompositeKey(values []interface{}) string {
	var builder strings.Builder
	for _, value := range values {
		normalized := normalizeKeyValue(value)
		_, _ = fmt.Fprintf(&builder, "%T:%v|", normalized, normalized)
	}
	return builder.String()
}

func TestBuildCompositeKeyEquivalence(t *testing.T) {
	named := namedExpandKey(42)
	ptr := &named
	cases := []interface{}{
		nil, (*int)(nil), (**int)(nil), 0, -1, int(math.MinInt), int(math.MaxInt),
		int8(-128), int16(-32768), int32(math.MinInt32), int64(math.MinInt64),
		uint(0), uint(math.MaxUint), uint8(255), uint16(65535), uint32(math.MaxUint32),
		uint64(math.MaxUint64), uintptr(math.MaxUint), true, false,
		"", "a:int:42|", "'\"\\\n\u2028", named, &named, &ptr, formattedExpandKey(7),
		float32(1.25), 1.25, math.NaN(), math.Inf(1), math.Copysign(0, -1),
		[]byte{0, 1, 255}, struct{ ID int }{3},
	}
	for _, value := range cases {
		for _, values := range [][]interface{}{{value}, {value, "x|", int64(9)}} {
			if got, want := buildCompositeKey(values), legacyCompositeKey(values); got != want {
				t.Errorf("%#v: got %q, want %q", values, got, want)
			}
		}
	}
	if got := buildCompositeKey(nil); got != "" {
		t.Fatalf("empty key = %q", got)
	}
	seen := make(map[string]bool)
	for _, value := range []interface{}{int(1), int64(1), uint(1), "1", namedExpandKey(1)} {
		key := buildCompositeKey([]interface{}{value})
		if seen[key] {
			t.Fatalf("lost type discrimination for %T", value)
		}
		seen[key] = true
	}
}

type expandKeyRow struct {
	Key   interface{}
	Other *int
	Order int
}

func TestExpandKeyGroupingOwnershipAndOrder(t *testing.T) {
	one, two := 1, 2
	rows := []*expandKeyRow{
		{int(1), &one, 0}, {int(2), &two, 1}, {int(1), &one, 2},
		{int64(1), &one, 3}, {nil, &one, 4}, nil,
		{int(9), nil, 6}, {namedExpandKey(1), &one, 7}, {&one, &two, 8},
		{(*int)(nil), &one, 9},
	}
	constraints := []parentReferenceConstraint{
		{principalProperty: "Key", dependentProperty: "Key"},
		{principalProperty: "Other", dependentProperty: "Other"},
	}
	parents, err := collectParentValues(rows)
	if err != nil {
		t.Fatal(err)
	}
	for _, cs := range [][]parentReferenceConstraint{constraints[:1], constraints} {
		parentMap, keys := collectParentKeyValues(parents, cs)
		childMap := groupChildrenByParentKey(reflect.ValueOf(rows), cs)
		want := make(map[string][]*expandKeyRow)
		var wantKeys []parentKey
		for _, row := range rows {
			if row == nil || normalizeKeyValue(row.Key) == nil || (len(cs) == 2 && row.Other == nil) {
				continue
			}
			values := []interface{}{normalizeKeyValue(row.Key)}
			if len(cs) == 2 {
				values = append(values, *row.Other)
			}
			key := legacyCompositeKey(values)
			if _, exists := want[key]; !exists {
				wantKeys = append(wantKeys, parentKey{key: key, values: values})
			}
			want[key] = append(want[key], row)
		}
		if !reflect.DeepEqual(keys, wantKeys) {
			t.Fatalf("retained keys changed or alias scratch: got %#v, want %#v", keys, wantKeys)
		}
		if len(parentMap) != len(want) || len(childMap) != len(want) {
			t.Fatalf("wrong group counts: parents=%d children=%d want=%d", len(parentMap), len(childMap), len(want))
		}
		for key, expected := range want {
			if !reflect.DeepEqual(childMap[key].Interface(), expected) {
				t.Errorf("children for %q lost values or order", key)
			}
			if len(parentMap[key]) != len(expected) {
				t.Fatalf("wrong parent count for %q", key)
			}
			for i, parent := range parentMap[key] {
				if parent.Interface() != expected[i] {
					t.Errorf("parent order changed for %q", key)
				}
			}
		}
	}
}

func TestGroupChildrenPreservesNamedSliceAndCopiesStructs(t *testing.T) {
	type row struct {
		Key   namedExpandKey
		Order int
	}
	type rows []row
	input := make(rows, 256)
	for i := range input {
		input[i] = row{Key: namedExpandKey(i % 2), Order: i}
	}
	cs := []parentReferenceConstraint{{dependentProperty: "Key"}}
	groups := groupChildrenByParentKey(reflect.ValueOf(&input), cs)
	for k := 0; k < 2; k++ {
		key := legacyCompositeKey([]interface{}{namedExpandKey(k)})
		group, ok := groups[key].Interface().(rows)
		if !ok || len(group) != 128 {
			t.Fatalf("group %q changed slice type or size", key)
		}
		for i, child := range group {
			if child != input[2*i+k] {
				t.Fatalf("group %q index %d changed order/value", key, i)
			}
		}
		group[0].Order = -1
		if input[k].Order != k {
			t.Fatal("group unexpectedly aliases input structs")
		}
	}
}

func TestExpandKeysInvalidFields(t *testing.T) {
	rows := []*expandKeyRow{{Key: 1}, nil, {Key: 2}}
	parents, err := collectParentValues(rows)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"", "Missing"} {
		cs := []parentReferenceConstraint{
			{principalProperty: "Key", dependentProperty: "Key"},
			{principalProperty: field, dependentProperty: field},
		}
		parentMap, keys := collectParentKeyValues(parents, cs)
		if len(parentMap) != 0 || len(keys) != 0 || len(groupChildrenByParentKey(reflect.ValueOf(rows), cs)) != 0 {
			t.Fatalf("invalid field %q produced groups", field)
		}
	}
}

func BenchmarkExpandKeyGrouping(b *testing.B) {
	type row struct {
		ID       int
		ParentID int
		Payload  [128]byte
	}
	for _, size := range []int{50, 10000} {
		rows := make([]row, size)
		for i := range rows {
			rows[i].ID = i
			rows[i].ParentID = i % 100
		}
		children := reflect.ValueOf(rows)
		parents, err := collectParentValues(rows)
		if err != nil {
			b.Fatal(err)
		}
		cs := []parentReferenceConstraint{{principalProperty: "ParentID", dependentProperty: "ParentID"}}
		b.Run(fmt.Sprintf("Children%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				groupChildrenByParentKey(children, cs)
			}
		})
		b.Run(fmt.Sprintf("Parents%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				collectParentKeyValues(parents, cs)
			}
		})
	}
}
