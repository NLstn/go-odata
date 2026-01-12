package edm

import (
	"fmt"

	"github.com/shopspring/decimal"
)

func init() {
	RegisterType("Edm.Decimal", NewDecimal)
}

// Decimal represents an Edm.Decimal value with arbitrary precision
type Decimal struct {
	value  decimal.Decimal
	isNull bool
	facets Facets
}

// NewDecimal creates a new Edm.Decimal from a value
func NewDecimal(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &Decimal{isNull: true, facets: facets}, nil
	}

	var decValue decimal.Decimal
	switch v := value.(type) {
	case decimal.Decimal:
		decValue = v
	case *decimal.Decimal:
		if v == nil {
			return &Decimal{isNull: true, facets: facets}, nil
		}
		decValue = *v
	case string:
		var err error
		decValue, err = decimal.NewFromString(v)
		if err != nil {
			return nil, fmt.Errorf("cannot parse '%s' as Edm.Decimal: %w", v, err)
		}
	case float64:
		decValue = decimal.NewFromFloat(v)
	case float32:
		decValue = decimal.NewFromFloat32(v)
	case int:
		decValue = decimal.NewFromInt(int64(v))
	case int32:
		decValue = decimal.NewFromInt32(v)
	case int64:
		decValue = decimal.NewFromInt(v)
	default:
		return nil, fmt.Errorf("cannot convert %T to Edm.Decimal", value)
	}

	d := &Decimal{
		value:  decValue,
		isNull: false,
		facets: facets,
	}

	if err := d.Validate(); err != nil {
		return nil, err
	}

	return d, nil
}

// TypeName returns "Edm.Decimal"
func (d *Decimal) TypeName() string {
	return "Edm.Decimal"
}

// IsNull returns true if the value is null
func (d *Decimal) IsNull() bool {
	return d.isNull
}

// Value returns the underlying decimal.Decimal value
func (d *Decimal) Value() interface{} {
	if d.isNull {
		return nil
	}
	return d.value
}

// String returns the OData literal format (with 'M' suffix)
func (d *Decimal) String() string {
	if d.isNull {
		return "null"
	}
	return d.value.String() + "M"
}

// Validate checks if the value meets precision and scale constraints
func (d *Decimal) Validate() error {
	if d.isNull {
		return nil
	}

	// Validate precision and scale facets
	return ValidateDecimalFacets(d.value.String(), d.facets)
}

// SetFacets applies facets to the type
func (d *Decimal) SetFacets(facets Facets) error {
	d.facets = facets
	return d.Validate()
}

// GetFacets returns the current facets
func (d *Decimal) GetFacets() Facets {
	return d.facets
}

// MarshalJSON implements json.Marshaler
func (d *Decimal) MarshalJSON() ([]byte, error) {
	if d.isNull {
		return []byte("null"), nil
	}
	return d.value.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Decimal) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		d.isNull = true
		return nil
	}

	if err := d.value.UnmarshalJSON(data); err != nil {
		return err
	}

	d.isNull = false
	return d.Validate()
}

// GetDecimalValue returns the underlying decimal.Decimal for direct access
func (d *Decimal) GetDecimalValue() decimal.Decimal {
	return d.value
}
