package edm

import (
	"encoding/json"
	"math"
	"testing"
)

func TestInt32Type(t *testing.T) {
	t.Run("Create from int32", func(t *testing.T) {
		i, err := NewInt32(int32(42), Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i.TypeName() != "Edm.Int32" {
			t.Errorf("expected TypeName 'Edm.Int32', got '%s'", i.TypeName())
		}
		if i.Value() != int32(42) {
			t.Errorf("expected value 42, got %v", i.Value())
		}
		if i.String() != "42" {
			t.Errorf("expected '42', got '%s'", i.String())
		}
	})

	t.Run("Create from int", func(t *testing.T) {
		i, err := NewInt32(100, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i.Value() != int32(100) {
			t.Errorf("expected value 100, got %v", i.Value())
		}
	})

	t.Run("Out of range error", func(t *testing.T) {
		_, err := NewInt32(int64(math.MaxInt32)+1, Facets{})
		if err == nil {
			t.Error("expected error for out of range value")
		}
	})

	t.Run("Null value", func(t *testing.T) {
		i, err := NewInt32(nil, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !i.IsNull() {
			t.Error("expected null int32")
		}
		if i.String() != "null" {
			t.Errorf("expected 'null', got '%s'", i.String())
		}
	})

	t.Run("JSON marshaling", func(t *testing.T) {
		i, _ := NewInt32(42, Facets{})
		data, err := json.Marshal(i)
		if err != nil {
			t.Fatalf("marshaling error: %v", err)
		}
		if string(data) != `42` {
			t.Errorf("expected JSON '42', got '%s'", string(data))
		}
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		var i Int32
		err := json.Unmarshal([]byte(`123`), &i)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if i.Value() != int32(123) {
			t.Errorf("expected value 123, got %v", i.Value())
		}
	})
}

func TestInt64Type(t *testing.T) {
	t.Run("Create from int64", func(t *testing.T) {
		i, err := NewInt64(int64(9223372036854775807), Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i.TypeName() != "Edm.Int64" {
			t.Errorf("expected TypeName 'Edm.Int64', got '%s'", i.TypeName())
		}
		if i.Value() != int64(9223372036854775807) {
			t.Errorf("unexpected value: %v", i.Value())
		}
	})

	t.Run("Create from int32", func(t *testing.T) {
		i, err := NewInt64(int32(100), Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i.Value() != int64(100) {
			t.Errorf("expected value 100, got %v", i.Value())
		}
	})

	t.Run("JSON marshaling", func(t *testing.T) {
		i, _ := NewInt64(9223372036854775807, Facets{})
		data, err := json.Marshal(i)
		if err != nil {
			t.Fatalf("marshaling error: %v", err)
		}
		if string(data) != `9223372036854775807` {
			t.Errorf("expected JSON '9223372036854775807', got '%s'", string(data))
		}
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		var i Int64
		err := json.Unmarshal([]byte(`123456789`), &i)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if i.Value() != int64(123456789) {
			t.Errorf("expected value 123456789, got %v", i.Value())
		}
	})

	t.Run("JSON unmarshaling null", func(t *testing.T) {
		var i Int64
		err := json.Unmarshal([]byte(`null`), &i)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if !i.IsNull() {
			t.Error("expected null int64")
		}
	})
}

func TestInt16Type(t *testing.T) {
	t.Run("Create from int16", func(t *testing.T) {
		i, err := NewInt16(int16(1000), Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i.TypeName() != "Edm.Int16" {
			t.Errorf("expected TypeName 'Edm.Int16', got '%s'", i.TypeName())
		}
		if i.Value() != int16(1000) {
			t.Errorf("expected value 1000, got %v", i.Value())
		}
	})

	t.Run("Boundary values", func(t *testing.T) {
		min, err := NewInt16(int16(-32768), Facets{})
		if err != nil {
			t.Fatalf("unexpected error for min: %v", err)
		}
		if min.Value() != int16(-32768) {
			t.Errorf("expected min value -32768, got %v", min.Value())
		}

		max, err := NewInt16(int16(32767), Facets{})
		if err != nil {
			t.Fatalf("unexpected error for max: %v", err)
		}
		if max.Value() != int16(32767) {
			t.Errorf("expected max value 32767, got %v", max.Value())
		}
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		var i Int16
		err := json.Unmarshal([]byte(`1000`), &i)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if i.Value() != int16(1000) {
			t.Errorf("expected value 1000, got %v", i.Value())
		}
	})

	t.Run("JSON unmarshaling null", func(t *testing.T) {
		var i Int16
		err := json.Unmarshal([]byte(`null`), &i)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if !i.IsNull() {
			t.Error("expected null int16")
		}
	})

	t.Run("Out of range error", func(t *testing.T) {
		_, err := NewInt16(int32(40000), Facets{})
		if err == nil {
			t.Error("expected error for out of range value")
		}
	})
}

func TestByteType(t *testing.T) {
	t.Run("Create from uint8", func(t *testing.T) {
		b, err := NewByte(uint8(200), Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.TypeName() != "Edm.Byte" {
			t.Errorf("expected TypeName 'Edm.Byte', got '%s'", b.TypeName())
		}
		if b.Value() != uint8(200) {
			t.Errorf("expected value 200, got %v", b.Value())
		}
	})

	t.Run("Boundary values", func(t *testing.T) {
		min, err := NewByte(uint8(0), Facets{})
		if err != nil {
			t.Fatalf("unexpected error for min: %v", err)
		}
		if min.Value() != uint8(0) {
			t.Errorf("expected min value 0, got %v", min.Value())
		}

		max, err := NewByte(uint8(255), Facets{})
		if err != nil {
			t.Fatalf("unexpected error for max: %v", err)
		}
		if max.Value() != uint8(255) {
			t.Errorf("expected max value 255, got %v", max.Value())
		}
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		var b Byte
		err := json.Unmarshal([]byte(`200`), &b)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if b.Value() != uint8(200) {
			t.Errorf("expected value 200, got %v", b.Value())
		}
	})

	t.Run("JSON unmarshaling null", func(t *testing.T) {
		var b Byte
		err := json.Unmarshal([]byte(`null`), &b)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if !b.IsNull() {
			t.Error("expected null byte")
		}
	})

	t.Run("Out of range error", func(t *testing.T) {
		_, err := NewByte(int(300), Facets{})
		if err == nil {
			t.Error("expected error for out of range value")
		}

		_, err = NewByte(int(-1), Facets{})
		if err == nil {
			t.Error("expected error for negative value")
		}
	})
}

func TestSByteType(t *testing.T) {
	t.Run("Create from int8", func(t *testing.T) {
		s, err := NewSByte(int8(-50), Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.TypeName() != "Edm.SByte" {
			t.Errorf("expected TypeName 'Edm.SByte', got '%s'", s.TypeName())
		}
		if s.Value() != int8(-50) {
			t.Errorf("expected value -50, got %v", s.Value())
		}
	})

	t.Run("Boundary values", func(t *testing.T) {
		min, err := NewSByte(int8(-128), Facets{})
		if err != nil {
			t.Fatalf("unexpected error for min: %v", err)
		}
		if min.Value() != int8(-128) {
			t.Errorf("expected min value -128, got %v", min.Value())
		}

		max, err := NewSByte(int8(127), Facets{})
		if err != nil {
			t.Fatalf("unexpected error for max: %v", err)
		}
		if max.Value() != int8(127) {
			t.Errorf("expected max value 127, got %v", max.Value())
		}
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		var s SByte
		err := json.Unmarshal([]byte(`-50`), &s)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if s.Value() != int8(-50) {
			t.Errorf("expected value -50, got %v", s.Value())
		}
	})

	t.Run("JSON unmarshaling null", func(t *testing.T) {
		var s SByte
		err := json.Unmarshal([]byte(`null`), &s)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if !s.IsNull() {
			t.Error("expected null sbyte")
		}
	})

	t.Run("Out of range error", func(t *testing.T) {
		_, err := NewSByte(int(200), Facets{})
		if err == nil {
			t.Error("expected error for out of range value")
		}
	})
}

func TestDoubleType(t *testing.T) {
	t.Run("Create from float64", func(t *testing.T) {
		d, err := NewDouble(3.14159, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.TypeName() != "Edm.Double" {
			t.Errorf("expected TypeName 'Edm.Double', got '%s'", d.TypeName())
		}
		if d.Value() != 3.14159 {
			t.Errorf("expected value 3.14159, got %v", d.Value())
		}
	})

	t.Run("Create from int", func(t *testing.T) {
		d, err := NewDouble(42, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.Value() != float64(42) {
			t.Errorf("expected value 42.0, got %v", d.Value())
		}
	})

	t.Run("Special values", func(t *testing.T) {
		inf, _ := NewDouble(math.Inf(1), Facets{})
		if inf.String() != "INF" {
			t.Errorf("expected 'INF', got '%s'", inf.String())
		}

		negInf, _ := NewDouble(math.Inf(-1), Facets{})
		if negInf.String() != "-INF" {
			t.Errorf("expected '-INF', got '%s'", negInf.String())
		}

		nan, _ := NewDouble(math.NaN(), Facets{})
		if nan.String() != "NaN" {
			t.Errorf("expected 'NaN', got '%s'", nan.String())
		}
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		var d Double
		err := json.Unmarshal([]byte(`3.14159`), &d)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if d.Value() != 3.14159 {
			t.Errorf("expected value 3.14159, got %v", d.Value())
		}
	})

	t.Run("JSON unmarshaling null", func(t *testing.T) {
		var d Double
		err := json.Unmarshal([]byte(`null`), &d)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if !d.IsNull() {
			t.Error("expected null double")
		}
	})
}

func TestSingleType(t *testing.T) {
	t.Run("Create from float32", func(t *testing.T) {
		s, err := NewSingle(float32(3.14), Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.TypeName() != "Edm.Single" {
			t.Errorf("expected TypeName 'Edm.Single', got '%s'", s.TypeName())
		}
		// Use approximate comparison for float32
		val := s.Value().(float32)
		if val < 3.13 || val > 3.15 {
			t.Errorf("expected value ~3.14, got %v", val)
		}
	})

	t.Run("OData literal format with f suffix", func(t *testing.T) {
		s, _ := NewSingle(float32(2.5), Facets{})
		str := s.String()
		if str != "2.5f" {
			t.Errorf("expected '2.5f', got '%s'", str)
		}
	})

	t.Run("Out of range error", func(t *testing.T) {
		_, err := NewSingle(float64(math.MaxFloat64), Facets{})
		if err == nil {
			t.Error("expected error for out of range value")
		}
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		var s Single
		err := json.Unmarshal([]byte(`2.5`), &s)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		val := s.Value().(float32)
		if val < 2.49 || val > 2.51 {
			t.Errorf("expected value ~2.5, got %v", val)
		}
	})

	t.Run("JSON unmarshaling null", func(t *testing.T) {
		var s Single
		err := json.Unmarshal([]byte(`null`), &s)
		if err != nil {
			t.Fatalf("unmarshaling error: %v", err)
		}
		if !s.IsNull() {
			t.Error("expected null single")
		}
	})
}
