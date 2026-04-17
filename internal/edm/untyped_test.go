package edm

import (
	"encoding/json"
	"testing"
)

func TestNewUntyped(t *testing.T) {
	t.Run("nil value is null", func(t *testing.T) {
		u, err := NewUntyped(nil, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !u.IsNull() {
			t.Fatal("expected IsNull() == true for nil input")
		}
		if u.Value() != nil {
			t.Fatalf("expected Value() == nil, got %v", u.Value())
		}
		if u.String() != "null" {
			t.Fatalf("expected String() == \"null\", got %q", u.String())
		}
	})

	t.Run("json.RawMessage", func(t *testing.T) {
		raw := json.RawMessage(`{"key":"value"}`)
		u, err := NewUntyped(raw, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.IsNull() {
			t.Fatal("expected IsNull() == false")
		}
		if u.TypeName() != "Edm.Untyped" {
			t.Fatalf("expected TypeName() == \"Edm.Untyped\", got %q", u.TypeName())
		}
		b, err := json.Marshal(u)
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}
		if string(b) != `{"key":"value"}` {
			t.Fatalf("expected JSON %q, got %q", `{"key":"value"}`, string(b))
		}
	})

	t.Run("nil json.RawMessage is null", func(t *testing.T) {
		u, err := NewUntyped(json.RawMessage(nil), Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !u.IsNull() {
			t.Fatal("expected IsNull() == true for nil RawMessage")
		}
	})

	t.Run("invalid json.RawMessage returns error", func(t *testing.T) {
		_, err := NewUntyped(json.RawMessage(`not-valid-json`), Facets{})
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})

	t.Run("interface value (string)", func(t *testing.T) {
		var v interface{} = "hello"
		u, err := NewUntyped(v, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.IsNull() {
			t.Fatal("expected IsNull() == false")
		}
		if u.Value() != "hello" {
			t.Fatalf("expected Value() == \"hello\", got %v", u.Value())
		}
	})

	t.Run("interface value (map)", func(t *testing.T) {
		v := map[string]interface{}{"x": 1}
		u, err := NewUntyped(v, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.IsNull() {
			t.Fatal("expected IsNull() == false")
		}
		b, err := json.Marshal(u)
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}
		if string(b) != `{"x":1}` {
			t.Fatalf("expected JSON %q, got %q", `{"x":1}`, string(b))
		}
	})

	t.Run("Validate always returns nil", func(t *testing.T) {
		u, _ := NewUntyped("anything", Facets{})
		if err := u.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("TypeName is Edm.Untyped", func(t *testing.T) {
		u, _ := NewUntyped(nil, Facets{})
		if u.TypeName() != "Edm.Untyped" {
			t.Fatalf("expected TypeName() == \"Edm.Untyped\", got %q", u.TypeName())
		}
	})

	t.Run("SetFacets and GetFacets round-trip", func(t *testing.T) {
		u, _ := NewUntyped("x", Facets{})
		nullable := true
		f := Facets{Nullable: nullable}
		if err := u.SetFacets(f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := u.GetFacets()
		if got.Nullable != f.Nullable {
			t.Fatalf("expected Nullable=%v, got %v", f.Nullable, got.Nullable)
		}
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		u := &Untyped{}
		if err := json.Unmarshal([]byte(`{"a":1}`), u); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.IsNull() {
			t.Fatal("expected IsNull() == false after unmarshal")
		}
		b, err := json.Marshal(u)
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}
		if string(b) != `{"a":1}` {
			t.Fatalf("expected JSON %q, got %q", `{"a":1}`, string(b))
		}
	})

	t.Run("UnmarshalJSON null", func(t *testing.T) {
		u := &Untyped{}
		if err := json.Unmarshal([]byte(`null`), u); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !u.IsNull() {
			t.Fatal("expected IsNull() == true after unmarshal of null")
		}
	})
}
