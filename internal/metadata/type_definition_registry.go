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
	UnderlyingType string
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
func inferUnderlyingEdmType(t reflect.Type) (string, error) {
	// Check for well-known named types first
	switch t.String() {
	case "time.Time":
		return "Edm.DateTimeOffset", nil
	case "uuid.UUID", "github.com/google/uuid.UUID":
		return "Edm.Guid", nil
	case "decimal.Decimal", "github.com/shopspring/decimal.Decimal":
		return "Edm.Decimal", nil
	}

	switch t.Kind() {
	case reflect.String:
		return "Edm.String", nil
	case reflect.Bool:
		return "Edm.Boolean", nil
	case reflect.Int8:
		return "Edm.SByte", nil
	case reflect.Int16:
		return "Edm.Int16", nil
	case reflect.Int, reflect.Int32:
		return "Edm.Int32", nil
	case reflect.Int64:
		return "Edm.Int64", nil
	case reflect.Uint8:
		return "Edm.Byte", nil
	case reflect.Uint16:
		return "Edm.Int32", nil
	case reflect.Uint, reflect.Uint32:
		return "Edm.Int64", nil
	case reflect.Uint64:
		return "Edm.Int64", nil
	case reflect.Float32:
		return "Edm.Single", nil
	case reflect.Float64:
		return "Edm.Double", nil
	default:
		if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
			return "Edm.Binary", nil
		}
		return "", fmt.Errorf("unsupported underlying kind %s for TypeDefinition", t.Kind())
	}
}
