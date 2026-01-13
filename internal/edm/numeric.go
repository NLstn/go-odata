package edm

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

func init() {
	RegisterType("Edm.Int32", NewInt32)
	RegisterType("Edm.Int64", NewInt64)
	RegisterType("Edm.Int16", NewInt16)
	RegisterType("Edm.Byte", NewByte)
	RegisterType("Edm.SByte", NewSByte)
	RegisterType("Edm.Double", NewDouble)
	RegisterType("Edm.Single", NewSingle)
}

// Int32 represents an Edm.Int32 value
type Int32 struct {
	value  int32
	isNull bool
	facets Facets
}

// NewInt32 creates a new Edm.Int32 from a value
func NewInt32(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &Int32{isNull: true, facets: facets}, nil
	}

	var int32Value int32
	switch v := value.(type) {
	case int32:
		int32Value = v
	case *int32:
		if v == nil {
			return &Int32{isNull: true, facets: facets}, nil
		}
		int32Value = *v
	case int:
		if v < math.MinInt32 || v > math.MaxInt32 {
			return nil, fmt.Errorf("value %d out of range for Edm.Int32", v)
		}
		int32Value = int32(v)
	case int64:
		if v < math.MinInt32 || v > math.MaxInt32 {
			return nil, fmt.Errorf("value %d out of range for Edm.Int32", v)
		}
		int32Value = int32(v)
	case float64:
		if v < math.MinInt32 || v > math.MaxInt32 {
			return nil, fmt.Errorf("value %f out of range for Edm.Int32", v)
		}
		int32Value = int32(v)
	default:
		return nil, fmt.Errorf("cannot convert %T to Edm.Int32", value)
	}

	return &Int32{value: int32Value, facets: facets}, nil
}

func (i *Int32) TypeName() string { return "Edm.Int32" }
func (i *Int32) IsNull() bool     { return i.isNull }
func (i *Int32) Value() interface{} {
	if i.isNull {
		return nil
	}
	return i.value
}
func (i *Int32) String() string {
	if i.isNull {
		return "null"
	}
	return strconv.Itoa(int(i.value))
}
func (i *Int32) Validate() error               { return nil }
func (i *Int32) SetFacets(facets Facets) error { i.facets = facets; return nil }
func (i *Int32) GetFacets() Facets             { return i.facets }

func (i *Int32) MarshalJSON() ([]byte, error) {
	if i.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(i.value)
}

func (i *Int32) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		i.isNull = true
		return nil
	}
	return json.Unmarshal(data, &i.value)
}

// Int64 represents an Edm.Int64 value
type Int64 struct {
	value  int64
	isNull bool
	facets Facets
}

// NewInt64 creates a new Edm.Int64 from a value
func NewInt64(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &Int64{isNull: true, facets: facets}, nil
	}

	var int64Value int64
	switch v := value.(type) {
	case int64:
		int64Value = v
	case *int64:
		if v == nil {
			return &Int64{isNull: true, facets: facets}, nil
		}
		int64Value = *v
	case int:
		int64Value = int64(v)
	case int32:
		int64Value = int64(v)
	case float64:
		if v < math.MinInt64 || v > math.MaxInt64 {
			return nil, fmt.Errorf("value %f out of range for Edm.Int64", v)
		}
		int64Value = int64(v)
	default:
		return nil, fmt.Errorf("cannot convert %T to Edm.Int64", value)
	}

	return &Int64{value: int64Value, facets: facets}, nil
}

func (i *Int64) TypeName() string { return "Edm.Int64" }
func (i *Int64) IsNull() bool     { return i.isNull }
func (i *Int64) Value() interface{} {
	if i.isNull {
		return nil
	}
	return i.value
}
func (i *Int64) String() string {
	if i.isNull {
		return "null"
	}
	return strconv.FormatInt(i.value, 10)
}
func (i *Int64) Validate() error               { return nil }
func (i *Int64) SetFacets(facets Facets) error { i.facets = facets; return nil }
func (i *Int64) GetFacets() Facets             { return i.facets }

func (i *Int64) MarshalJSON() ([]byte, error) {
	if i.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(i.value)
}

func (i *Int64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		i.isNull = true
		return nil
	}
	return json.Unmarshal(data, &i.value)
}

// Int16 represents an Edm.Int16 value
type Int16 struct {
	value  int16
	isNull bool
	facets Facets
}

// NewInt16 creates a new Edm.Int16 from a value
func NewInt16(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &Int16{isNull: true, facets: facets}, nil
	}

	var int16Value int16
	switch v := value.(type) {
	case int16:
		int16Value = v
	case *int16:
		if v == nil {
			return &Int16{isNull: true, facets: facets}, nil
		}
		int16Value = *v
	case int:
		if v < math.MinInt16 || v > math.MaxInt16 {
			return nil, fmt.Errorf("value %d out of range for Edm.Int16", v)
		}
		int16Value = int16(v)
	case int32:
		if v < math.MinInt16 || v > math.MaxInt16 {
			return nil, fmt.Errorf("value %d out of range for Edm.Int16", v)
		}
		int16Value = int16(v)
	case float64:
		if v < math.MinInt16 || v > math.MaxInt16 {
			return nil, fmt.Errorf("value %f out of range for Edm.Int16", v)
		}
		int16Value = int16(v)
	default:
		return nil, fmt.Errorf("cannot convert %T to Edm.Int16", value)
	}

	return &Int16{value: int16Value, facets: facets}, nil
}

func (i *Int16) TypeName() string { return "Edm.Int16" }
func (i *Int16) IsNull() bool     { return i.isNull }
func (i *Int16) Value() interface{} {
	if i.isNull {
		return nil
	}
	return i.value
}
func (i *Int16) String() string {
	if i.isNull {
		return "null"
	}
	return strconv.Itoa(int(i.value))
}
func (i *Int16) Validate() error               { return nil }
func (i *Int16) SetFacets(facets Facets) error { i.facets = facets; return nil }
func (i *Int16) GetFacets() Facets             { return i.facets }

func (i *Int16) MarshalJSON() ([]byte, error) {
	if i.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(i.value)
}

func (i *Int16) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		i.isNull = true
		return nil
	}
	return json.Unmarshal(data, &i.value)
}

// Byte represents an Edm.Byte value (unsigned 8-bit, 0-255)
type Byte struct {
	value  uint8
	isNull bool
	facets Facets
}

// NewByte creates a new Edm.Byte from a value
func NewByte(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &Byte{isNull: true, facets: facets}, nil
	}

	var byteValue uint8
	switch v := value.(type) {
	case uint8:
		byteValue = v
	case *uint8:
		if v == nil {
			return &Byte{isNull: true, facets: facets}, nil
		}
		byteValue = *v
	case int:
		if v < 0 || v > 255 {
			return nil, fmt.Errorf("value %d out of range for Edm.Byte (0-255)", v)
		}
		byteValue = uint8(v)
	case int32:
		if v < 0 || v > 255 {
			return nil, fmt.Errorf("value %d out of range for Edm.Byte (0-255)", v)
		}
		byteValue = uint8(v)
	case float64:
		if v < 0 || v > 255 {
			return nil, fmt.Errorf("value %f out of range for Edm.Byte (0-255)", v)
		}
		byteValue = uint8(v)
	default:
		return nil, fmt.Errorf("cannot convert %T to Edm.Byte", value)
	}

	return &Byte{value: byteValue, facets: facets}, nil
}

func (b *Byte) TypeName() string { return "Edm.Byte" }
func (b *Byte) IsNull() bool     { return b.isNull }
func (b *Byte) Value() interface{} {
	if b.isNull {
		return nil
	}
	return b.value
}
func (b *Byte) String() string {
	if b.isNull {
		return "null"
	}
	return strconv.Itoa(int(b.value))
}
func (b *Byte) Validate() error               { return nil }
func (b *Byte) SetFacets(facets Facets) error { b.facets = facets; return nil }
func (b *Byte) GetFacets() Facets             { return b.facets }

func (b *Byte) MarshalJSON() ([]byte, error) {
	if b.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(b.value)
}

func (b *Byte) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		b.isNull = true
		return nil
	}
	return json.Unmarshal(data, &b.value)
}

// SByte represents an Edm.SByte value (signed 8-bit, -128 to 127)
type SByte struct {
	value  int8
	isNull bool
	facets Facets
}

// NewSByte creates a new Edm.SByte from a value
func NewSByte(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &SByte{isNull: true, facets: facets}, nil
	}

	var sbyteValue int8
	switch v := value.(type) {
	case int8:
		sbyteValue = v
	case *int8:
		if v == nil {
			return &SByte{isNull: true, facets: facets}, nil
		}
		sbyteValue = *v
	case int:
		if v < math.MinInt8 || v > math.MaxInt8 {
			return nil, fmt.Errorf("value %d out of range for Edm.SByte (-128 to 127)", v)
		}
		sbyteValue = int8(v)
	case int32:
		if v < math.MinInt8 || v > math.MaxInt8 {
			return nil, fmt.Errorf("value %d out of range for Edm.SByte (-128 to 127)", v)
		}
		sbyteValue = int8(v)
	case float64:
		if v < math.MinInt8 || v > math.MaxInt8 {
			return nil, fmt.Errorf("value %f out of range for Edm.SByte (-128 to 127)", v)
		}
		sbyteValue = int8(v)
	default:
		return nil, fmt.Errorf("cannot convert %T to Edm.SByte", value)
	}

	return &SByte{value: sbyteValue, facets: facets}, nil
}

func (s *SByte) TypeName() string { return "Edm.SByte" }
func (s *SByte) IsNull() bool     { return s.isNull }
func (s *SByte) Value() interface{} {
	if s.isNull {
		return nil
	}
	return s.value
}
func (s *SByte) String() string {
	if s.isNull {
		return "null"
	}
	return strconv.Itoa(int(s.value))
}
func (s *SByte) Validate() error               { return nil }
func (s *SByte) SetFacets(facets Facets) error { s.facets = facets; return nil }
func (s *SByte) GetFacets() Facets             { return s.facets }

func (s *SByte) MarshalJSON() ([]byte, error) {
	if s.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(s.value)
}

func (s *SByte) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		s.isNull = true
		return nil
	}
	return json.Unmarshal(data, &s.value)
}

// Double represents an Edm.Double value (IEEE 754 double precision)
type Double struct {
	value  float64
	isNull bool
	facets Facets
}

// NewDouble creates a new Edm.Double from a value
func NewDouble(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &Double{isNull: true, facets: facets}, nil
	}

	var float64Value float64
	switch v := value.(type) {
	case float64:
		float64Value = v
	case *float64:
		if v == nil {
			return &Double{isNull: true, facets: facets}, nil
		}
		float64Value = *v
	case float32:
		float64Value = float64(v)
	case int:
		float64Value = float64(v)
	case int32:
		float64Value = float64(v)
	case int64:
		float64Value = float64(v)
	default:
		return nil, fmt.Errorf("cannot convert %T to Edm.Double", value)
	}

	return &Double{value: float64Value, facets: facets}, nil
}

func (d *Double) TypeName() string { return "Edm.Double" }
func (d *Double) IsNull() bool     { return d.isNull }
func (d *Double) Value() interface{} {
	if d.isNull {
		return nil
	}
	return d.value
}
func (d *Double) String() string {
	if d.isNull {
		return "null"
	}
	if math.IsInf(d.value, 1) {
		return "INF"
	}
	if math.IsInf(d.value, -1) {
		return "-INF"
	}
	if math.IsNaN(d.value) {
		return "NaN"
	}
	return strconv.FormatFloat(d.value, 'g', -1, 64)
}
func (d *Double) Validate() error               { return nil }
func (d *Double) SetFacets(facets Facets) error { d.facets = facets; return nil }
func (d *Double) GetFacets() Facets             { return d.facets }

func (d *Double) MarshalJSON() ([]byte, error) {
	if d.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(d.value)
}

func (d *Double) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		d.isNull = true
		return nil
	}
	return json.Unmarshal(data, &d.value)
}

// Single represents an Edm.Single value (IEEE 754 single precision)
type Single struct {
	value  float32
	isNull bool
	facets Facets
}

// NewSingle creates a new Edm.Single from a value
func NewSingle(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &Single{isNull: true, facets: facets}, nil
	}

	var float32Value float32
	switch v := value.(type) {
	case float32:
		float32Value = v
	case *float32:
		if v == nil {
			return &Single{isNull: true, facets: facets}, nil
		}
		float32Value = *v
	case float64:
		if math.Abs(v) > math.MaxFloat32 {
			return nil, fmt.Errorf("value %f out of range for Edm.Single", v)
		}
		float32Value = float32(v)
	case int:
		float32Value = float32(v)
	case int32:
		float32Value = float32(v)
	default:
		return nil, fmt.Errorf("cannot convert %T to Edm.Single", value)
	}

	return &Single{value: float32Value, facets: facets}, nil
}

func (s *Single) TypeName() string { return "Edm.Single" }
func (s *Single) IsNull() bool     { return s.isNull }
func (s *Single) Value() interface{} {
	if s.isNull {
		return nil
	}
	return s.value
}
func (s *Single) String() string {
	if s.isNull {
		return "null"
	}
	f64 := float64(s.value)
	if math.IsInf(f64, 1) {
		return "INF"
	}
	if math.IsInf(f64, -1) {
		return "-INF"
	}
	if math.IsNaN(f64) {
		return "NaN"
	}
	return strconv.FormatFloat(f64, 'g', -1, 32) + "f"
}
func (s *Single) Validate() error               { return nil }
func (s *Single) SetFacets(facets Facets) error { s.facets = facets; return nil }
func (s *Single) GetFacets() Facets             { return s.facets }

func (s *Single) MarshalJSON() ([]byte, error) {
	if s.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(s.value)
}

func (s *Single) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		s.isNull = true
		return nil
	}
	return json.Unmarshal(data, &s.value)
}
