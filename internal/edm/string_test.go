package edm

import (
	"encoding/json"
	"testing"
)

func TestStringType(t *testing.T) {
	t.Run("Create from string value", func(t *testing.T) {
		s, err := NewString("hello", Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.TypeName() != "Edm.String" {
			t.Errorf("expected TypeName 'Edm.String', got '%s'", s.TypeName())
		}
		if s.IsNull() {
			t.Error("expected non-null string")
		}
		if s.Value() != "hello" {
			t.Errorf("expected value 'hello', got '%v'", s.Value())
		}
	})

	t.Run("Create from nil", func(t *testing.T) {
		s, err := NewString(nil, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !s.IsNull() {
			t.Error("expected null string")
		}
		if s.Value() != nil {
			t.Errorf("expected nil value, got %v", s.Value())
		}
	})

	t.Run("OData literal format", func(t *testing.T) {
		s, _ := NewString("test", Facets{})
		if s.String() != "'test'" {
			t.Errorf("expected \"'test'\", got '%s'", s.String())
		}
	})

	t.Run("OData literal with single quote", func(t *testing.T) {
		s, _ := NewString("test's", Facets{})
		if s.String() != "'test''s'" {
			t.Errorf("expected \"'test''s'\", got '%s'", s.String())
		}
	})

	t.Run("MaxLength facet validation", func(t *testing.T) {
		maxLen := 5
		_, err := NewString("toolong", Facets{MaxLength: &maxLen})
		if err == nil {
			t.Error("expected error for string exceeding maxLength")
		}
	})

	t.Run("MaxLength facet passes", func(t *testing.T) {
		maxLen := 10
		s, err := NewString("short", Facets{MaxLength: &maxLen})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if s == nil {
			t.Error("expected non-nil string")
		}
	})

	t.Run("JSON marshaling", func(t *testing.T) {
		s, _ := NewString("test", Facets{})
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("marshaling error: %v", err)
		}
		if string(data) != `"test"` {
			t.Errorf("expected JSON '\"test\"', got '%s'", string(data))
		}
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		var s String
		err := json.Unmarshal([]byte(`"hello"`), &s)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if s.Value() != "hello" {
			t.Errorf("expected value 'hello', got '%v'", s.Value())
		}
	})

	t.Run("JSON null handling", func(t *testing.T) {
		var s String
		err := json.Unmarshal([]byte(`null`), &s)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if !s.IsNull() {
			t.Error("expected null string")
		}
	})

	t.Run("SetFacets with valid maxLength", func(t *testing.T) {
		s, _ := NewString("hello", Facets{})
		maxLen := 10
		newFacets := Facets{MaxLength: &maxLen}
		err := s.(*String).SetFacets(newFacets)
		if err != nil {
			t.Errorf("SetFacets() error = %v", err)
		}
		gotFacets := s.(*String).GetFacets()
		if gotFacets.MaxLength == nil || *gotFacets.MaxLength != maxLen {
			t.Errorf("GetFacets() MaxLength = %v, want %v", gotFacets.MaxLength, maxLen)
		}
	})

	t.Run("SetFacets with invalid maxLength", func(t *testing.T) {
		s, _ := NewString("hello", Facets{})
		maxLen := 3 // Too small for "hello"
		newFacets := Facets{MaxLength: &maxLen}
		err := s.(*String).SetFacets(newFacets)
		if err == nil {
			t.Error("SetFacets() should error when maxLength is too small for current value")
		}
	})

	t.Run("GetFacets", func(t *testing.T) {
		maxLen := 50
		s, _ := NewString("test", Facets{MaxLength: &maxLen})
		gotFacets := s.(*String).GetFacets()
		if gotFacets.MaxLength == nil || *gotFacets.MaxLength != maxLen {
			t.Errorf("GetFacets() MaxLength = %v, want %v", gotFacets.MaxLength, maxLen)
		}
	})
}

