package odata

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/nlstn/go-odata/internal/metadata"
)

// EnumMember describes a single enum member when registering enums programmatically.
type EnumMember = metadata.EnumMember

// RegisterEnumType registers enum metadata for the provided enum type using a map of member names to values.
// The enumValue parameter accepts either a zero value of the enum type or a pointer to the enum type.
// Values must be representable as signed 64-bit integers to comply with the OData specification.
func RegisterEnumType(enumValue interface{}, members map[string]int64) error {
	if enumValue == nil {
		return fmt.Errorf("enumValue cannot be nil")
	}
	if len(members) == 0 {
		return fmt.Errorf("enum members cannot be empty")
	}

	enumType := reflect.TypeOf(enumValue)
	if enumType.Kind() == reflect.Pointer {
		enumType = enumType.Elem()
	}
	if enumType == nil {
		return fmt.Errorf("could not determine enum type")
	}

	converted := make([]metadata.EnumMember, 0, len(members))
	for name, value := range members {
		converted = append(converted, metadata.EnumMember{Name: name, Value: value})
	}

	sort.Slice(converted, func(i, j int) bool {
		if converted[i].Value == converted[j].Value {
			return converted[i].Name < converted[j].Name
		}
		return converted[i].Value < converted[j].Value
	})

	return metadata.RegisterEnumMembers(enumType, converted)
}
