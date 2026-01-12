package edm

import (
	"encoding/json"
	"fmt"
)

func init() {
	RegisterType("Edm.String", NewString)
}

// String represents an Edm.String value
type String struct {
	value  string
	isNull bool
	facets Facets
}

// NewString creates a new Edm.String from a value
func NewString(value interface{}, facets Facets) (Type, error) {
	if value == nil {
		return &String{isNull: true, facets: facets}, nil
	}

	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	case *string:
		if v == nil {
			return &String{isNull: true, facets: facets}, nil
		}
		strValue = *v
	default:
		return nil, fmt.Errorf("cannot convert %T to Edm.String", value)
	}

	s := &String{
		value:  strValue,
		isNull: false,
		facets: facets,
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}

	return s, nil
}

// TypeName returns "Edm.String"
func (s *String) TypeName() string {
	return "Edm.String"
}

// IsNull returns true if the value is null
func (s *String) IsNull() bool {
	return s.isNull
}

// Value returns the underlying string value
func (s *String) Value() interface{} {
	if s.isNull {
		return nil
	}
	return s.value
}

// String returns the OData literal format
func (s *String) String() string {
	if s.isNull {
		return "null"
	}
	// OData string literals are single-quoted with escaping
	escaped := s.value
	escaped = replaceAll(escaped, "'", "''")
	return "'" + escaped + "'"
}

// Validate checks if the value meets constraints
func (s *String) Validate() error {
	if s.isNull {
		return nil
	}

	// Validate maxLength facet
	if err := ValidateLengthFacet(len(s.value), s.facets); err != nil {
		return err
	}

	return nil
}

// SetFacets applies facets to the type
func (s *String) SetFacets(facets Facets) error {
	s.facets = facets
	return s.Validate()
}

// GetFacets returns the current facets
func (s *String) GetFacets() Facets {
	return s.facets
}

// MarshalJSON implements json.Marshaler
func (s *String) MarshalJSON() ([]byte, error) {
	if s.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(s.value)
}

// UnmarshalJSON implements json.Unmarshaler
func (s *String) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		s.isNull = true
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	s.value = value
	s.isNull = false
	return s.Validate()
}

// Helper function to replace all occurrences
func replaceAll(s, old, new string) string {
	result := ""
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

// Helper function to find index of substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
