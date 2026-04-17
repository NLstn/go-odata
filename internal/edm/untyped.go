package edm

import (
	"encoding/json"
	"fmt"
)

func init() {
	RegisterType("Edm.Untyped", NewUntyped)
}

// Untyped represents an Edm.Untyped value, which can hold arbitrary JSON.
type Untyped struct {
	value  interface{}
	isNull bool
	facets Facets
}

// NewUntyped creates a new Edm.Untyped from a value.
// Accepted Go types: nil, json.RawMessage, interface{}, map[string]interface{},
// []interface{}, or any JSON-serialisable value.
func NewUntyped(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &Untyped{isNull: true, facets: facets}, nil
	}

	switch v := value.(type) {
	case json.RawMessage:
		if v == nil {
			return &Untyped{isNull: true, facets: facets}, nil
		}
		// Validate that it's valid JSON
		if !json.Valid(v) {
			return nil, fmt.Errorf("Edm.Untyped: value is not valid JSON")
		}
		return &Untyped{value: v, facets: facets}, nil
	default:
		return &Untyped{value: value, facets: facets}, nil
	}
}

// TypeName returns "Edm.Untyped"
func (u *Untyped) TypeName() string {
	return "Edm.Untyped"
}

// IsNull returns true if the value is null
func (u *Untyped) IsNull() bool {
	return u.isNull
}

// Value returns the underlying value
func (u *Untyped) Value() interface{} {
	if u.isNull {
		return nil
	}
	return u.value
}

// String returns the OData literal representation.
// For Edm.Untyped the value is passed through as JSON.
func (u *Untyped) String() string {
	if u.isNull {
		return "null"
	}
	b, err := json.Marshal(u.value)
	if err != nil {
		return "null"
	}
	return string(b)
}

// Validate always succeeds – Edm.Untyped has no type constraints.
func (u *Untyped) Validate() error {
	return nil
}

// SetFacets applies facets to the type
func (u *Untyped) SetFacets(facets Facets) error {
	u.facets = facets
	return nil
}

// GetFacets returns the current facets
func (u *Untyped) GetFacets() Facets {
	return u.facets
}

// MarshalJSON implements json.Marshaler
func (u *Untyped) MarshalJSON() ([]byte, error) {
	if u.isNull {
		return []byte("null"), nil
	}
	// If the stored value is already a raw JSON message, return it directly
	if raw, ok := u.value.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(u.value)
}

// UnmarshalJSON implements json.Unmarshaler
func (u *Untyped) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		u.isNull = true
		u.value = nil
		return nil
	}
	// Store as raw JSON to preserve the original representation
	raw := make(json.RawMessage, len(data))
	copy(raw, data)
	u.value = raw
	u.isNull = false
	return nil
}
