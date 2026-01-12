package edm

import (
	"encoding/json"
	"fmt"
)

func init() {
	RegisterType("Edm.Boolean", NewBoolean)
}

// Boolean represents an Edm.Boolean value
type Boolean struct {
	value  bool
	isNull bool
	facets Facets
}

// NewBoolean creates a new Edm.Boolean from a value
func NewBoolean(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &Boolean{isNull: true, facets: facets}, nil
	}

	var boolValue bool
	switch v := value.(type) {
	case bool:
		boolValue = v
	case *bool:
		if v == nil {
			return &Boolean{isNull: true, facets: facets}, nil
		}
		boolValue = *v
	default:
		return nil, fmt.Errorf("cannot convert %T to Edm.Boolean", value)
	}

	return &Boolean{value: boolValue, facets: facets}, nil
}

// TypeName returns "Edm.Boolean"
func (b *Boolean) TypeName() string {
	return "Edm.Boolean"
}

// IsNull returns true if the value is null
func (b *Boolean) IsNull() bool {
	return b.isNull
}

// Value returns the underlying bool value
func (b *Boolean) Value() interface{} {
	if b.isNull {
		return nil
	}
	return b.value
}

// String returns the OData literal format
func (b *Boolean) String() string {
	if b.isNull {
		return "null"
	}
	if b.value {
		return "true"
	}
	return "false"
}

// Validate checks if the value meets constraints
func (b *Boolean) Validate() error {
	return nil
}

// SetFacets applies facets to the type
func (b *Boolean) SetFacets(facets Facets) error {
	b.facets = facets
	return nil
}

// GetFacets returns the current facets
func (b *Boolean) GetFacets() Facets {
	return b.facets
}

// MarshalJSON implements json.Marshaler
func (b *Boolean) MarshalJSON() ([]byte, error) {
	if b.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(b.value)
}

// UnmarshalJSON implements json.Unmarshaler
func (b *Boolean) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		b.isNull = true
		return nil
	}

	return json.Unmarshal(data, &b.value)
}
