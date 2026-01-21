package edm

import (
	"encoding/json"
	"testing"
)

func TestBooleanType(t *testing.T) {
	t.Run("Create from bool true", func(t *testing.T) {
		b, err := NewBoolean(true, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.TypeName() != "Edm.Boolean" {
			t.Errorf("expected TypeName 'Edm.Boolean', got '%s'", b.TypeName())
		}
		if b.Value() != true {
			t.Errorf("expected value true, got %v", b.Value())
		}
		if b.String() != "true" {
			t.Errorf("expected 'true', got '%s'", b.String())
		}
	})

	t.Run("Create from bool false", func(t *testing.T) {
		b, err := NewBoolean(false, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Value() != false {
			t.Errorf("expected value false, got %v", b.Value())
		}
		if b.String() != "false" {
			t.Errorf("expected 'false', got '%s'", b.String())
		}
	})

	t.Run("Create from nil", func(t *testing.T) {
		b, err := NewBoolean(nil, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !b.IsNull() {
			t.Error("expected null boolean")
		}
		if b.Value() != nil {
			t.Errorf("expected nil value, got %v", b.Value())
		}
		if b.String() != "null" {
			t.Errorf("expected 'null', got '%s'", b.String())
		}
	})

	t.Run("Create from bool pointer", func(t *testing.T) {
		val := true
		b, err := NewBoolean(&val, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Value() != true {
			t.Errorf("expected value true, got %v", b.Value())
		}
	})

	t.Run("Create from nil pointer", func(t *testing.T) {
		var val *bool
		b, err := NewBoolean(val, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !b.IsNull() {
			t.Error("expected null boolean")
		}
	})

	t.Run("Invalid type error", func(t *testing.T) {
		_, err := NewBoolean("not-a-bool", Facets{})
		if err == nil {
			t.Error("expected error for invalid type")
		}
	})

	t.Run("Validate", func(t *testing.T) {
		b, _ := NewBoolean(true, Facets{})
		err := b.(*Boolean).Validate()
		if err != nil {
			t.Errorf("Validate() should not error, got %v", err)
		}
	})

	t.Run("SetFacets and GetFacets", func(t *testing.T) {
		b, _ := NewBoolean(true, Facets{})
		newFacets := Facets{Nullable: false}
		err := b.(*Boolean).SetFacets(newFacets)
		if err != nil {
			t.Errorf("SetFacets() error = %v", err)
		}
		gotFacets := b.(*Boolean).GetFacets()
		if gotFacets.Nullable != newFacets.Nullable {
			t.Errorf("GetFacets() = %v, want %v", gotFacets, newFacets)
		}
	})

	t.Run("MarshalJSON true", func(t *testing.T) {
		b, _ := NewBoolean(true, Facets{})
		data, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("MarshalJSON error: %v", err)
		}
		if string(data) != "true" {
			t.Errorf("MarshalJSON() = %v, want 'true'", string(data))
		}
	})

	t.Run("MarshalJSON false", func(t *testing.T) {
		b, _ := NewBoolean(false, Facets{})
		data, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("MarshalJSON error: %v", err)
		}
		if string(data) != "false" {
			t.Errorf("MarshalJSON() = %v, want 'false'", string(data))
		}
	})

	t.Run("MarshalJSON null", func(t *testing.T) {
		b, _ := NewBoolean(nil, Facets{})
		data, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("MarshalJSON error: %v", err)
		}
		if string(data) != "null" {
			t.Errorf("MarshalJSON() = %v, want 'null'", string(data))
		}
	})

	t.Run("UnmarshalJSON true", func(t *testing.T) {
		var b Boolean
		err := json.Unmarshal([]byte("true"), &b)
		if err != nil {
			t.Fatalf("UnmarshalJSON error: %v", err)
		}
		if b.Value() != true {
			t.Errorf("Value() = %v, want true", b.Value())
		}
	})

	t.Run("UnmarshalJSON false", func(t *testing.T) {
		var b Boolean
		err := json.Unmarshal([]byte("false"), &b)
		if err != nil {
			t.Fatalf("UnmarshalJSON error: %v", err)
		}
		if b.Value() != false {
			t.Errorf("Value() = %v, want false", b.Value())
		}
	})

	t.Run("UnmarshalJSON null", func(t *testing.T) {
		var b Boolean
		err := json.Unmarshal([]byte("null"), &b)
		if err != nil {
			t.Fatalf("UnmarshalJSON error: %v", err)
		}
		if !b.IsNull() {
			t.Error("expected null boolean")
		}
	})
}
