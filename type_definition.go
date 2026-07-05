package odata

import (
	"fmt"
	"reflect"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TypeDefinitionFacets holds optional facets for an OData TypeDefinition.
// Facets constrain the values of the underlying primitive type.
type TypeDefinitionFacets struct {
	// Precision is the numeric precision facet, valid for Edm.Decimal.
	Precision int
	// Scale is the numeric scale facet, valid for Edm.Decimal.
	Scale int
	// MaxLength is the maximum length facet, valid for Edm.String and Edm.Binary.
	MaxLength int
	// UnderlyingType optionally overrides the EDM primitive type this TypeDefinition
	// aliases (e.g. "Edm.Date"). When empty, it's inferred from the Go type's kind
	// (e.g. a string field infers Edm.String). Set this to declare a string-backed
	// field as Edm.Date, Edm.TimeOfDay, or Edm.Duration, none of which have a
	// distinct Go representation in this library - without it, $filter cannot
	// distinguish such a property from a genuine Edm.String and won't type-check
	// date/time functions or literals applied to it.
	UnderlyingType string
}

// RegisterTypeDefinition registers a Go named type as an OData TypeDefinition.
// The goValue parameter should be a zero value or pointer to the named type.
// An optional name can be provided via the name parameter; when empty the Go type name is used.
// An optional set of facets can be provided to constrain the underlying type.
//
// Example:
//
//	type Weight float64
//	odata.RegisterTypeDefinition(Weight(0), "", odata.TypeDefinitionFacets{})
//
//	type Description string
//	odata.RegisterTypeDefinition(Description(""), "Description", odata.TypeDefinitionFacets{MaxLength: 255})
func RegisterTypeDefinition(goValue interface{}, name string, facets TypeDefinitionFacets) error {
	if goValue == nil {
		return fmt.Errorf("goValue cannot be nil")
	}

	goType := reflect.TypeOf(goValue)
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	return metadata.RegisterTypeDefinition(goType, metadata.TypeDefinitionInfo{
		Name:           name,
		UnderlyingType: facets.UnderlyingType,
		Precision:      facets.Precision,
		Scale:          facets.Scale,
		MaxLength:      facets.MaxLength,
	})
}
