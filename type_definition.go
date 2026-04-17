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
		Name:      name,
		Precision: facets.Precision,
		Scale:     facets.Scale,
		MaxLength: facets.MaxLength,
	})
}
