package edm

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func TestDecimalType(t *testing.T) {
	t.Run("Create from decimal.Decimal", func(t *testing.T) {
		dec := decimal.NewFromFloat(123.45)
		d, err := NewDecimal(dec, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.TypeName() != "Edm.Decimal" {
			t.Errorf("expected TypeName 'Edm.Decimal', got '%s'", d.TypeName())
		}
		if !d.Value().(decimal.Decimal).Equal(dec) {
			t.Errorf("expected value %v, got %v", dec, d.Value())
		}
	})

	t.Run("Create from string", func(t *testing.T) {
		d, err := NewDecimal("999.9999", Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := decimal.RequireFromString("999.9999")
		if !d.Value().(decimal.Decimal).Equal(expected) {
			t.Errorf("expected value %v, got %v", expected, d.Value())
		}
	})

	t.Run("Create from float64", func(t *testing.T) {
		d, err := NewDecimal(99.99, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := decimal.NewFromFloat(99.99)
		val := d.Value().(decimal.Decimal)
		// Use approximate comparison for float conversions
		if val.Sub(expected).Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
			t.Errorf("expected value ~%v, got %v", expected, val)
		}
	})

	t.Run("Create from int", func(t *testing.T) {
		d, err := NewDecimal(100, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := decimal.NewFromInt(100)
		if !d.Value().(decimal.Decimal).Equal(expected) {
			t.Errorf("expected value %v, got %v", expected, d.Value())
		}
	})

	t.Run("Null value", func(t *testing.T) {
		d, err := NewDecimal(nil, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !d.IsNull() {
			t.Error("expected null decimal")
		}
		if d.Value() != nil {
			t.Errorf("expected nil value, got %v", d.Value())
		}
		if d.String() != "null" {
			t.Errorf("expected 'null', got '%s'", d.String())
		}
	})

	t.Run("OData literal format with M suffix", func(t *testing.T) {
		d, _ := NewDecimal("123.45", Facets{})
		if d.String() != "123.45M" {
			t.Errorf("expected '123.45M', got '%s'", d.String())
		}
	})

	t.Run("Precision facet validation", func(t *testing.T) {
		precision := 5
		// This value has 6 digits, should fail
		_, err := NewDecimal("123456", Facets{Precision: &precision})
		if err == nil {
			t.Error("expected error for value exceeding precision")
		}

		// This value has 5 digits, should pass
		d, err := NewDecimal("12345", Facets{Precision: &precision})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if d == nil {
			t.Error("expected non-nil decimal")
		}
	})

	t.Run("Scale facet validation", func(t *testing.T) {
		scale := 2
		// This value has 3 fractional digits, should fail
		_, err := NewDecimal("123.456", Facets{Scale: &scale})
		if err == nil {
			t.Error("expected error for value exceeding scale")
		}

		// This value has 2 fractional digits, should pass
		d, err := NewDecimal("123.45", Facets{Scale: &scale})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if d == nil {
			t.Error("expected non-nil decimal")
		}
	})

	t.Run("Precision and scale facets together", func(t *testing.T) {
		precision := 18
		scale := 4
		// Valid: 14 integer digits + 4 fractional = 18 total
		d, err := NewDecimal("12345678901234.5678", Facets{Precision: &precision, Scale: &scale})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if d == nil {
			t.Error("expected non-nil decimal")
		}

		// Invalid: Too many total digits
		_, err = NewDecimal("123456789012345.6789", Facets{Precision: &precision, Scale: &scale})
		if err == nil {
			t.Error("expected error for value exceeding precision")
		}

		// Invalid: Too many fractional digits
		_, err = NewDecimal("12345678901234.56789", Facets{Precision: &precision, Scale: &scale})
		if err == nil {
			t.Error("expected error for value exceeding scale")
		}
	})

	t.Run("User's example: Revenue with precision=18,scale=4", func(t *testing.T) {
		precision := 18
		scale := 4
		dec := decimal.NewFromFloat(12345.6789)
		d, err := NewDecimal(dec, Facets{Precision: &precision, Scale: &scale})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		facets := d.GetFacets()
		if facets.Precision == nil || *facets.Precision != 18 {
			t.Errorf("expected precision 18, got %v", facets.Precision)
		}
		if facets.Scale == nil || *facets.Scale != 4 {
			t.Errorf("expected scale 4, got %v", facets.Scale)
		}
	})

	t.Run("JSON marshaling", func(t *testing.T) {
		d, _ := NewDecimal("123.45", Facets{})
		data, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("marshaling error: %v", err)
		}
		// shopspring/decimal marshals as string by default
		expected := `"123.45"`
		if string(data) != expected {
			t.Errorf("expected JSON %s, got '%s'", expected, string(data))
		}
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		var d Decimal
		err := json.Unmarshal([]byte(`"999.99"`), &d)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		expected := decimal.RequireFromString("999.99")
		if !d.Value().(decimal.Decimal).Equal(expected) {
			t.Errorf("expected value %v, got %v", expected, d.Value())
		}
	})

	t.Run("JSON null handling", func(t *testing.T) {
		var d Decimal
		err := json.Unmarshal([]byte(`null`), &d)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if !d.IsNull() {
			t.Error("expected null decimal")
		}
	})

	t.Run("Invalid string format", func(t *testing.T) {
		_, err := NewDecimal("not-a-number", Facets{})
		if err == nil {
			t.Error("expected error for invalid decimal string")
		}
	})

	t.Run("SetFacets with valid facets", func(t *testing.T) {
		d, _ := NewDecimal("123.45", Facets{})
		precision := 5
		scale := 2
		newFacets := Facets{Precision: &precision, Scale: &scale}
		err := d.(*Decimal).SetFacets(newFacets)
		if err != nil {
			t.Errorf("SetFacets() error = %v", err)
		}
		gotFacets := d.(*Decimal).GetFacets()
		if gotFacets.Precision == nil || *gotFacets.Precision != precision {
			t.Errorf("GetFacets() Precision = %v, want %v", gotFacets.Precision, precision)
		}
		if gotFacets.Scale == nil || *gotFacets.Scale != scale {
			t.Errorf("GetFacets() Scale = %v, want %v", gotFacets.Scale, scale)
		}
	})

	t.Run("SetFacets with invalid facets", func(t *testing.T) {
		d, _ := NewDecimal("123.45", Facets{})
		precision := 3 // Too small for 123.45 (needs at least 5 digits)
		newFacets := Facets{Precision: &precision}
		err := d.(*Decimal).SetFacets(newFacets)
		if err == nil {
			t.Error("SetFacets() should error when facets are invalid for current value")
		}
	})

	t.Run("GetDecimalValue", func(t *testing.T) {
		expected := decimal.NewFromFloat(123.45)
		d, _ := NewDecimal(expected, Facets{})
		result := d.(*Decimal).GetDecimalValue()
		if !result.Equal(expected) {
			t.Errorf("GetDecimalValue() = %v, want %v", result, expected)
		}
	})

	t.Run("GetDecimalValue with null", func(t *testing.T) {
		d, _ := NewDecimal(nil, Facets{})
		result := d.(*Decimal).GetDecimalValue()
		if !result.IsZero() {
			t.Errorf("GetDecimalValue() for null decimal should return zero value, got %v", result)
		}
	})
}
