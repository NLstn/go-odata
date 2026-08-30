package metadata

import (
	"fmt"
	"reflect"
	"sync"
)

// TypeDefinitionInfo holds metadata for an OData TypeDefinition element.
// TypeDefinitions are custom named types aliasing a primitive OData type.
// See OData v4.0 spec, Section 10.4.
type TypeDefinitionInfo struct {
	// Name is the OData type name for the TypeDefinition.
	// If empty, the Go type name is used.
	Name string
	// UnderlyingType is the EDM primitive type (e.g., "Edm.Decimal", "Edm.String").
	// Derived automatically from the Go type if not provided.
	UnderlyingType PrimitiveType
	// Precision is the numeric precision facet (only for Edm.Decimal).
	Precision int
	// Scale is the numeric scale facet (only for Edm.Decimal).
	Scale int
	// MaxLength is the max length facet (only for Edm.String and Edm.Binary).
	MaxLength int
}

var typeDefinitionRegistry = struct {
	sync.RWMutex
	data map[reflect.Type]*TypeDefinitionInfo
}{
	data: make(map[reflect.Type]*TypeDefinitionInfo),
}

// RegisterTypeDefinition registers a Go named type as an OData TypeDefinition.
// The goType must be a named type whose underlying kind maps to an EDM primitive.
// If info.Name is empty, the Go type name is used.
// If info.UnderlyingType is empty, it is inferred from the Go type's kind.
func RegisterTypeDefinition(goType reflect.Type, info TypeDefinitionInfo) error {
	if goType == nil {
		return fmt.Errorf("goType cannot be nil")
	}

	// Dereference pointers
	for goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	if goType.Name() == "" {
		return fmt.Errorf("goType must be a named type, got anonymous type %s", goType)
	}

	// Infer name from Go type if not provided
	if info.Name == "" {
		info.Name = goType.Name()
	}

	// Infer underlying type from Go kind if not provided
	if info.UnderlyingType == "" {
		underlying, err := inferUnderlyingEdmType(goType)
		if err != nil {
			return fmt.Errorf("cannot infer underlying EDM type for %s: %w", goType.Name(), err)
		}
		info.UnderlyingType = underlying
	} else if _, err := ParsePrimitiveType(string(info.UnderlyingType)); err != nil {
		return fmt.Errorf("invalid underlying EDM type for %s: %w", goType.Name(), err)
	}

	typeDefinitionRegistry.Lock()
	defer typeDefinitionRegistry.Unlock()
	infoCopy := info
	typeDefinitionRegistry.data[goType] = &infoCopy
	return nil
}

// GetTypeDefinition returns the registered TypeDefinitionInfo for the given Go type, if any.
func GetTypeDefinition(goType reflect.Type) (*TypeDefinitionInfo, bool) {
	if goType == nil {
		return nil, false
	}
	for goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}
	typeDefinitionRegistry.RLock()
	defer typeDefinitionRegistry.RUnlock()
	info, ok := typeDefinitionRegistry.data[goType]
	if !ok {
		return nil, false
	}
	// Return a copy to avoid mutation
	copy := *info
	return &copy, true
}

// inferUnderlyingEdmType returns the EDM primitive type name for the given Go type.
func inferUnderlyingEdmType(t reflect.Type) (PrimitiveType, error) {
	return PrimitiveTypeFromGoType(t)
}
