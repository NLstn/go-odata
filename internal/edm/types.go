package edm

import (
	"fmt"
	"reflect"
	"sync"
)

// Type represents an EDM primitive type with value and metadata
type Type interface {
	// TypeName returns the EDM type name (e.g., "Edm.String")
	TypeName() string

	// IsNull indicates if the value is null
	IsNull() bool

	// Value returns the underlying Go value
	Value() interface{}

	// String converts to OData literal format
	String() string

	// Validate checks if the value meets type constraints
	Validate() error

	// SetFacets applies facets to the type
	SetFacets(facets Facets) error

	// GetFacets returns the current facets
	GetFacets() Facets
}

// Parser is a function that parses a value into an EDM type
type Parser func(value interface{}, facets Facets) (Type, error)

// typeRegistry maintains registered EDM types
// Uses sync.Map for concurrent-safe access during package initialization
var typeRegistry sync.Map

// RegisterType registers a parser for an EDM type name
func RegisterType(typeName string, parser Parser) {
	typeRegistry.Store(typeName, parser)
}

// IsValidType checks if a type name is registered
func IsValidType(typeName string) bool {
	_, ok := typeRegistry.Load(typeName)
	return ok
}

// ParseType parses a value into the specified EDM type
func ParseType(typeName string, value interface{}, facets Facets) (Type, error) {
	val, ok := typeRegistry.Load(typeName)
	if !ok {
		return nil, fmt.Errorf("unknown EDM type: %s", typeName)
	}
	parser, ok := val.(Parser)
	if !ok {
		return nil, fmt.Errorf("invalid parser type for EDM type: %s", typeName)
	}
	return parser(value, facets)
}

// FromGoType infers the EDM type from a Go type
func FromGoType(goType reflect.Type) (string, error) {
	if goType == nil {
		return "", fmt.Errorf("nil type")
	}

	// Handle pointer types
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	// Check for specific known types
	if goType.PkgPath() == "time" && goType.Name() == "Time" {
		return "Edm.DateTimeOffset", nil
	}

	if goType.PkgPath() == "github.com/shopspring/decimal" && goType.Name() == "Decimal" {
		return "Edm.Decimal", nil
	}

	if goType.PkgPath() == "github.com/google/uuid" && goType.Name() == "UUID" {
		return "Edm.Guid", nil
	}

	// Handle byte slices
	if goType.Kind() == reflect.Slice && goType.Elem().Kind() == reflect.Uint8 {
		return "Edm.Binary", nil
	}

	if goType.Kind() == reflect.Array && goType.Elem().Kind() == reflect.Uint8 {
		return "Edm.Binary", nil
	}

	// Map basic Go types to EDM types
	switch goType.Kind() {
	case reflect.String:
		return "Edm.String", nil
	case reflect.Int, reflect.Int32:
		return "Edm.Int32", nil
	case reflect.Int64:
		return "Edm.Int64", nil
	case reflect.Int16:
		return "Edm.Int16", nil
	case reflect.Int8:
		return "Edm.SByte", nil
	case reflect.Uint, reflect.Uint32:
		return "Edm.Int64", nil
	case reflect.Uint64:
		return "Edm.Int64", nil
	case reflect.Uint16:
		return "Edm.Int32", nil
	case reflect.Uint8:
		return "Edm.Byte", nil
	case reflect.Float32:
		return "Edm.Single", nil
	case reflect.Float64:
		return "Edm.Double", nil
	case reflect.Bool:
		return "Edm.Boolean", nil
	default:
		return "", fmt.Errorf("unsupported Go type: %s", goType.String())
	}
}

// FromGoValue infers the EDM type from a Go value and parses it
func FromGoValue(value interface{}) (Type, error) {
	if value == nil {
		return nil, fmt.Errorf("cannot infer type from nil value")
	}

	goType := reflect.TypeOf(value)
	typeName, err := FromGoType(goType)
	if err != nil {
		return nil, err
	}

	return ParseType(typeName, value, Facets{})
}

// FromStructField creates an EDM type from a struct field with tag support
func FromStructField(field reflect.StructField, value interface{}) (Type, error) {
	tag := field.Tag.Get("odata")
	if tag == "" {
		// No tag, infer from field type
		typeName, err := FromGoType(field.Type)
		if err != nil {
			return nil, err
		}
		return ParseType(typeName, value, Facets{})
	}

	// Parse type and facets from tag
	typeName, facets, err := ParseTypeFromTag(tag)
	if err != nil {
		return nil, fmt.Errorf("failed to parse odata tag: %w", err)
	}

	// If no explicit type in tag, infer from field type
	if typeName == "" {
		typeName, err = FromGoType(field.Type)
		if err != nil {
			return nil, err
		}
	}

	return ParseType(typeName, value, facets)
}
