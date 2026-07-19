package response

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// refJSON encodes v exactly as marshalTo's fallback would: encoding/json with
// HTML escaping disabled and the trailing newline trimmed.
func refJSON(t *testing.T, v interface{}) string {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		t.Fatalf("reference encode failed: %v", err)
	}
	return strings.TrimRight(buf.String(), "\n")
}

// loweredJSON runs v through writeComplexValue and returns the bytes it wrote,
// matching how marshalTo serializes a complex-type value.
func loweredJSON(t *testing.T, v interface{}) (string, bool) {
	t.Helper()
	var buf bytes.Buffer
	var enc *json.Encoder
	handled, err := writeComplexValue(&buf, reflect.ValueOf(v), &enc)
	if err != nil {
		t.Fatalf("writeComplexValue failed: %v", err)
	}
	if !handled {
		return "", false
	}
	return buf.String(), true
}

// notLowered reports whether writeComplexValue declined to handle v (i.e. it
// falls back to encoding/json).
func notLowered(v reflect.Value) bool {
	var buf bytes.Buffer
	var enc *json.Encoder
	handled, _ := writeComplexValue(&buf, v, &enc)
	return handled
}

type addr struct {
	Street  string `json:"Street"`
	City    string `json:"City"`
	Country string `json:"Country"`
}

type dims struct {
	Length float64 `json:"Length"`
	Width  float64 `json:"Width"`
	Unit   string  `json:"Unit"`
}

func TestWriteComplexValue_MatchesStdlib(t *testing.T) {
	a := addr{Street: "1 Main St", City: "Springfield", Country: "USA"}
	d := dims{Length: 12.5, Width: 8, Unit: "cm"}

	cases := []interface{}{
		a,  // struct value
		&a, // pointer to struct
		d,  // floats incl. integer-valued
		&d,
	}
	for _, v := range cases {
		got, ok := loweredJSON(t, v)
		if !ok {
			t.Fatalf("expected %T to be lowerable", v)
		}
		if want := refJSON(t, v); got != want {
			t.Errorf("%T: got %s, want %s", v, got, want)
		}
	}
}

func TestWriteComplexValue_NilPointerIsNull(t *testing.T) {
	got, ok := loweredJSON(t, (*addr)(nil))
	if !ok {
		t.Fatal("nil pointer should be handled")
	}
	if got != "null" {
		t.Errorf("nil pointer: got %s, want null", got)
	}
}

func TestWriteComplexValue_TagsAndVisibility(t *testing.T) {
	type tagged struct {
		Kept       string `json:"kept"`
		Renamed    int    `json:"renamed_name"`
		Skipped    string `json:"-"`
		unexported string //nolint:unused // present to verify it is skipped
		NoTag      bool
	}
	v := tagged{Kept: "a", Renamed: 5, Skipped: "hidden", unexported: "x", NoTag: true}
	got, ok := loweredJSON(t, v)
	if !ok {
		t.Fatal("expected lowerable")
	}
	if want := refJSON(t, v); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestWriteComplexValue_OmitEmpty(t *testing.T) {
	type withOmit struct {
		Name  string  `json:"name,omitempty"`
		Count int     `json:"count,omitempty"`
		Rate  float64 `json:"rate,omitempty"`
		Ptr   *string `json:"ptr,omitempty"`
		Flag  bool    `json:"flag,omitempty"`
		Tags  []int   `json:"tags,omitempty"`
	}
	s := "set"
	cases := []withOmit{
		{}, // everything empty -> all omitted
		{Name: "x", Count: 1, Rate: 2.5, Ptr: &s, Flag: true, Tags: []int{1}}, // all present
		{Count: 0, Name: "only"}, // mixed
	}
	for _, v := range cases {
		got, ok := loweredJSON(t, v)
		if !ok {
			t.Fatalf("expected lowerable: %+v", v)
		}
		if want := refJSON(t, v); got != want {
			t.Errorf("%+v: got %s, want %s", v, got, want)
		}
	}
}

func TestWriteComplexValue_NestedAndTime(t *testing.T) {
	type outer struct {
		Label   string    `json:"Label"`
		Addr    addr      `json:"Addr"`
		AddrPtr *addr     `json:"AddrPtr"`
		When    time.Time `json:"When"`
	}
	v := outer{
		Label:   "n",
		Addr:    addr{Street: "s", City: "c", Country: "d"},
		AddrPtr: &addr{Street: "s2", City: "c2", Country: "d2"},
		When:    time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC),
	}
	got, ok := loweredJSON(t, v)
	if !ok {
		t.Fatal("expected lowerable")
	}
	if want := refJSON(t, v); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// marshalerStruct customizes its own JSON and therefore must NOT be lowered.
type marshalerStruct struct {
	X int
}

func (marshalerStruct) MarshalJSON() ([]byte, error) { return []byte(`"custom"`), nil }

func TestWriteComplexValue_CustomMarshalerNotLowered(t *testing.T) {
	if notLowered(reflect.ValueOf(marshalerStruct{X: 1})) {
		t.Error("types with custom MarshalJSON must not be lowered")
	}
}

func TestWriteComplexValue_AnonymousEmbeddingNotLowered(t *testing.T) {
	type base struct {
		A string `json:"A"`
	}
	type withEmbed struct {
		base
		B string `json:"B"`
	}
	if notLowered(reflect.ValueOf(withEmbed{})) {
		t.Error("structs with anonymous embedded fields must not be lowered")
	}
}

func TestWriteComplexValue_NonStructNotLowered(t *testing.T) {
	for _, v := range []interface{}{"s", 42, 3.14, true} {
		if notLowered(reflect.ValueOf(v)) {
			t.Errorf("%T should not be lowerable", v)
		}
	}
}

// benchAddr mirrors a typical embedded complex type (flat scalar struct).
type benchAddr struct {
	Street     string `json:"Street"`
	City       string `json:"City"`
	State      string `json:"State"`
	PostalCode string `json:"PostalCode"`
	Country    string `json:"Country"`
}

// BenchmarkComplexValue_DirectWrite measures the plan-based direct buffer write.
func BenchmarkComplexValue_DirectWrite(b *testing.B) {
	v := reflect.ValueOf(&benchAddr{Street: "123 Main St", City: "Springfield", State: "IL", PostalCode: "62704", Country: "USA"})
	var buf bytes.Buffer
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		var enc *json.Encoder
		if _, err := writeComplexValue(&buf, v, &enc); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComplexValue_Fallback measures the previous reflection path
// (encoding/json.Encoder) for the same value.
func BenchmarkComplexValue_Fallback(b *testing.B) {
	val := &benchAddr{Street: "123 Main St", City: "Springfield", State: "IL", PostalCode: "62704", Country: "USA"}
	var buf bytes.Buffer
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		var enc *json.Encoder
		if err := encodeFallback(&buf, &enc, val); err != nil {
			b.Fatal(err)
		}
	}
}
