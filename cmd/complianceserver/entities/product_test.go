package entities

import (
	"testing"
)

func TestNumericScanersAcceptWiderSQLServerValues(t *testing.T) {
	t.Run("rating", func(t *testing.T) {
		var v RatingValue
		if err := v.Scan(int64(246)); err != nil {
			t.Fatalf("Scan(int64) returned error: %v", err)
		}
		if v != RatingValue(246) {
			t.Fatalf("Scan(int64) = %d, want 246", v)
		}
	})

	t.Run("temperature", func(t *testing.T) {
		var v TemperatureValue
		if err := v.Scan(int64(-10)); err != nil {
			t.Fatalf("Scan(int64) returned error: %v", err)
		}
		if v != TemperatureValue(-10) {
			t.Fatalf("Scan(int64) = %d, want -10", v)
		}
	})

	t.Run("quantity", func(t *testing.T) {
		var v QuantityValue
		if err := v.Scan(int64(32767)); err != nil {
			t.Fatalf("Scan(int64) returned error: %v", err)
		}
		if v != QuantityValue(32767) {
			t.Fatalf("Scan(int64) = %d, want 32767", v)
		}
	})
}

func TestNumericValuesRoundTripToDriverValue(t *testing.T) {
	t.Run("rating", func(t *testing.T) {
		v := RatingValue(200)
		got, err := v.Value()
		if err != nil {
			t.Fatalf("Value() returned error: %v", err)
		}
		if got != int64(200) {
			t.Fatalf("Value() = %v, want 200", got)
		}
	})

	t.Run("temperature", func(t *testing.T) {
		v := TemperatureValue(-10)
		got, err := v.Value()
		if err != nil {
			t.Fatalf("Value() returned error: %v", err)
		}
		if got != int64(-10) {
			t.Fatalf("Value() = %v, want -10", got)
		}
	})

	t.Run("quantity", func(t *testing.T) {
		v := QuantityValue(1200)
		got, err := v.Value()
		if err != nil {
			t.Fatalf("Value() returned error: %v", err)
		}
		if got != int64(1200) {
			t.Fatalf("Value() = %v, want 1200", got)
		}
	})
}
