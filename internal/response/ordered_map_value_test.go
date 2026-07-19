package response

import (
	"encoding/json"
	"testing"
	"time"
)

// valueJSON returns the JSON encoding of a single value using the same settings
// (HTML escaping disabled) that marshalTo uses, so we can assert byte-for-byte
// that the fast paths match encoding/json.
func valueJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal(%T) failed: %v", v, err)
	}
	return string(b)
}

// marshalOne marshals an OrderedMap holding a single key and returns the encoded
// value portion (everything between `{"k":` and the closing `}`).
func marshalOne(t *testing.T, v interface{}) string {
	t.Helper()
	om := NewOrderedMap()
	om.Set("k", v)
	b, err := om.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	s := string(b)
	const prefix = `{"k":`
	if len(s) < len(prefix)+1 || s[:len(prefix)] != prefix || s[len(s)-1] != '}' {
		t.Fatalf("unexpected envelope: %s", s)
	}
	return s[len(prefix) : len(s)-1]
}

func TestMarshalTo_TimeMatchesStdlib(t *testing.T) {
	cases := []time.Time{
		time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC),
		time.Date(2024, 6, 1, 8, 0, 0, 0, time.FixedZone("CEST", 2*60*60)),
		time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC),
		{}, // zero time
	}
	for _, tc := range cases {
		got := marshalOne(t, tc)
		want := valueJSON(t, tc)
		if got != want {
			t.Errorf("time %v: got %s, want %s", tc, got, want)
		}
	}
}

func TestMarshalTo_PointerScalarsMatchStdlib(t *testing.T) {
	s := "hello \"world\"\n"
	i := 42
	i64 := int64(-9007199254740991)
	u := uint(7)
	f := 3.14159
	b := true
	tm := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)

	values := []interface{}{
		&s, &i, &i64, &u, &f, &b, &tm,
		(*string)(nil), (*int)(nil), (*int64)(nil), (*uint)(nil),
		(*float64)(nil), (*bool)(nil), (*time.Time)(nil),
	}
	for _, v := range values {
		got := marshalOne(t, v)
		want := valueJSON(t, v)
		if got != want {
			t.Errorf("value %#v: got %s, want %s", v, got, want)
		}
	}
}

func TestMarshalTo_StringEscapingMatchesStdlib(t *testing.T) {
	cases := []string{
		"plain",
		"with \"quotes\"",
		"tab\tand\nnewline",
		"unicode: café — 日本語",
		"backslash \\ end",
	}
	for _, tc := range cases {
		got := marshalOne(t, tc)
		want := valueJSON(t, tc)
		if got != want {
			t.Errorf("string %q: got %s, want %s", tc, got, want)
		}
	}
}

// realisticProductEntity builds an OrderedMap mirroring the perfserver Product
// entity: mixed scalars, a time.Time, nullable pointer fields, an enum rendered
// as a string, and a nested *OrderedMap for an embedded complex type.
func realisticProductEntity(id int) *OrderedMap {
	desc := "A longer description for the product that might be a bit verbose"
	catID := uint(id%100 + 1)
	created := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	dims := AcquireOrderedMapWithCapacity(3)
	dims.Set("Length", 12.5)
	dims.Set("Width", 8.0)
	dims.Set("Height", 3.25)

	om := AcquireOrderedMapWithCapacity(12)
	om.Set("@odata.id", "http://localhost/Products(1)")
	om.Set("ID", uint(id))
	om.Set("Name", "Product Name Here")
	om.Set("Description", &desc)
	om.Set("Price", 99.99)
	om.Set("CategoryID", &catID)
	om.Set("Status", "InStock, OnSale")
	om.Set("Version", 1)
	om.Set("CreatedAt", created)
	om.Set("Dimensions", dims)
	return om
}

// BenchmarkMarshalRealisticEntity exercises the value types that dominate the
// #841 baseline profile (time.Time, nullable pointers, enum-as-string).
func BenchmarkMarshalRealisticEntity(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		om := realisticProductEntity(i)
		buf, err := om.marshalToPooledBuffer()
		if err != nil {
			b.Fatal(err)
		}
		releasePooledBuffer(buf)
		om.Release()
	}
}

// BenchmarkMarshalRealisticCollection500 mirrors the heaviest baseline scenario:
// serializing a 500-entity collection into one buffer.
func BenchmarkMarshalRealisticCollection500(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		value := make([]interface{}, 500)
		for j := range value {
			value[j] = realisticProductEntity(j)
		}
		env := AcquireOrderedMapWithCapacity(2)
		env.Set("@odata.context", "http://localhost/$metadata#Products")
		env.Set("value", value)

		buf, err := env.marshalToPooledBuffer()
		if err != nil {
			b.Fatal(err)
		}
		releasePooledBuffer(buf)

		for _, v := range value {
			v.(*OrderedMap).Release() //nolint:errcheck // known type
		}
		env.Release()
	}
}
