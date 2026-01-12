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
}

func TestBooleanType(t *testing.T) {
	t.Run("Create from true", func(t *testing.T) {
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

	t.Run("Create from false", func(t *testing.T) {
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
		if b.String() != "null" {
			t.Errorf("expected 'null', got '%s'", b.String())
		}
	})

	t.Run("JSON marshaling", func(t *testing.T) {
		b, _ := NewBoolean(true, Facets{})
		data, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("marshaling error: %v", err)
		}
		if string(data) != `true` {
			t.Errorf("expected JSON 'true', got '%s'", string(data))
		}
	})

	t.Run("JSON unmarshaling valid true", func(t *testing.T) {
		var b Boolean
		err := json.Unmarshal([]byte(`true`), &b)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Value() != true {
			t.Errorf("expected true, got %v", b.Value())
		}
	})

	t.Run("JSON unmarshaling valid false", func(t *testing.T) {
		var b Boolean
		err := json.Unmarshal([]byte(`false`), &b)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Value() != false {
			t.Errorf("expected false, got %v", b.Value())
		}
	})

	t.Run("JSON unmarshaling null", func(t *testing.T) {
		var b Boolean
		err := json.Unmarshal([]byte(`null`), &b)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !b.IsNull() {
			t.Error("expected null boolean")
		}
	})

	t.Run("JSON unmarshaling invalid", func(t *testing.T) {
		var b Boolean
		err := json.Unmarshal([]byte(`"not a boolean"`), &b)
		if err == nil {
			t.Error("expected error for invalid input")
		}
	})
}
