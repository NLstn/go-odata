package metadata

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"sync"
)

// EnumMember represents a single member of an enum type for metadata generation.
type EnumMember struct {
	Name  string
	Value int64
}

var enumRegistry = struct {
	sync.RWMutex
	data map[reflect.Type][]EnumMember
}{
	data: make(map[reflect.Type][]EnumMember),
}

// RegisterEnumMembers registers enum members for the given enum type.
// The enumType must resolve to an integral type (signed or unsigned) that is compatible with OData enum types.
func RegisterEnumMembers(enumType reflect.Type, members []EnumMember) error {
	if enumType == nil {
		return fmt.Errorf("enum type cannot be nil")
	}

	baseType := resolveEnumBaseType(enumType)
	if baseType == nil {
		return fmt.Errorf("enum type %s must be an integral type", enumType)
	}

	if len(members) == 0 {
		return fmt.Errorf("enum type %s must have at least one member", baseType.Name())
	}

	// Validate members: ensure unique names and deterministic ordering.
	seenNames := make(map[string]struct{})
	seenValues := make(map[int64]struct{})
	normalized := make([]EnumMember, len(members))
	for i, member := range members {
		if member.Name == "" {
			return fmt.Errorf("enum type %s has a member with an empty name", baseType.Name())
		}
		if _, exists := seenNames[member.Name]; exists {
			return fmt.Errorf("enum type %s has duplicate member name %s", baseType.Name(), member.Name)
		}
		seenNames[member.Name] = struct{}{}

		if _, exists := seenValues[member.Value]; exists {
			return fmt.Errorf("enum type %s has duplicate member value %d", baseType.Name(), member.Value)
		}
		seenValues[member.Value] = struct{}{}

		normalized[i] = member
	}

	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Value == normalized[j].Value {
			return normalized[i].Name < normalized[j].Name
		}
		return normalized[i].Value < normalized[j].Value
	})

	enumRegistry.Lock()
	defer enumRegistry.Unlock()
	enumRegistry.data[baseType] = normalized
	return nil
}

// getRegisteredEnumMembers retrieves registered enum members for the given enum type.
func getRegisteredEnumMembers(enumType reflect.Type) ([]EnumMember, bool) {
	enumRegistry.RLock()
	defer enumRegistry.RUnlock()
	members, ok := enumRegistry.data[enumType]
	if !ok {
		return nil, false
	}
	copied := make([]EnumMember, len(members))
	copy(copied, members)
	return copied, true
}

// ResolveEnumMembers resolves enum members for the provided field type.
// It first checks the registry and, if not found, attempts to call an EnumMembers() map[string]<int> method on the enum type.
func ResolveEnumMembers(fieldType reflect.Type) ([]EnumMember, reflect.Type, error) {
	baseType := resolveEnumBaseType(fieldType)
	if baseType == nil {
		return nil, nil, fmt.Errorf("enum field type %s must ultimately resolve to an integral type", fieldType)
	}

	if members, ok := getRegisteredEnumMembers(baseType); ok {
		return members, baseType, nil
	}

	members, err := extractEnumMembersViaMethod(baseType)
	if err != nil {
		return nil, nil, err
	}
	if len(members) == 0 {
		return nil, nil, fmt.Errorf("enum type %s has no registered members", baseType.Name())
	}

	if err := RegisterEnumMembers(baseType, members); err != nil {
		return nil, nil, err
	}

	return members, baseType, nil
}

// resolveEnumBaseType unwraps pointers, slices, and arrays to find the underlying enum type.
func resolveEnumBaseType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}

	for {
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			t = t.Elem()
			continue
		}
		break
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return t
	default:
		return nil
	}
}

// extractEnumMembersViaMethod attempts to call EnumMembers() on the enum type to obtain members.
func extractEnumMembersViaMethod(enumType reflect.Type) ([]EnumMember, error) {
	pointerValue := reflect.New(enumType)
	method := pointerValue.MethodByName("EnumMembers")
	if !method.IsValid() {
		value := pointerValue.Elem()
		if value.CanAddr() {
			method = value.Addr().MethodByName("EnumMembers")
		}
		if !method.IsValid() {
			method = value.MethodByName("EnumMembers")
		}
	}

	if !method.IsValid() {
		return nil, nil
	}

	if method.Type().NumIn() != 0 || method.Type().NumOut() != 1 {
		return nil, fmt.Errorf("EnumMembers method on type %s must have signature EnumMembers() map[string]<integer>", enumType.Name())
	}

	resultType := method.Type().Out(0)
	if resultType.Kind() != reflect.Map || resultType.Key().Kind() != reflect.String {
		return nil, fmt.Errorf("EnumMembers method on type %s must return map[string]<integer>", enumType.Name())
	}

	values := method.Call(nil)
	if len(values) != 1 {
		return nil, fmt.Errorf("EnumMembers method on type %s returned unexpected results", enumType.Name())
	}

	mapValue := values[0]
	if mapValue.IsNil() {
		return nil, fmt.Errorf("EnumMembers method on type %s returned nil", enumType.Name())
	}

	iter := mapValue.MapRange()
	members := make([]EnumMember, 0, mapValue.Len())
	seen := make(map[string]struct{})
	valueKind := resultType.Elem().Kind()
	if !isSupportedEnumValueKind(valueKind) {
		return nil, fmt.Errorf("EnumMembers method on type %s must return map[string]<integer>", enumType.Name())
	}

	for iter.Next() {
		name := iter.Key().String()
		if name == "" {
			return nil, fmt.Errorf("enum type %s has a member with an empty name", enumType.Name())
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("enum type %s has duplicate member name %s", enumType.Name(), name)
		}
		seen[name] = struct{}{}

		value := iter.Value()
		memberValue, err := convertEnumValueToInt64(value)
		if err != nil {
			return nil, fmt.Errorf("enum type %s has invalid member %s: %w", enumType.Name(), name, err)
		}
		members = append(members, EnumMember{Name: name, Value: memberValue})
	}

	sort.Slice(members, func(i, j int) bool {
		if members[i].Value == members[j].Value {
			return members[i].Name < members[j].Name
		}
		return members[i].Value < members[j].Value
	})

	return members, nil
}

func isSupportedEnumValueKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func convertEnumValueToInt64(value reflect.Value) (int64, error) {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		unsigned := value.Uint()
		if unsigned > math.MaxInt64 {
			return 0, fmt.Errorf("value %d exceeds maximum supported enum value", unsigned)
		}
		return int64(unsigned), nil
	default:
		return 0, fmt.Errorf("unsupported enum value kind %s", value.Kind())
	}
}

// DetermineEnumUnderlyingType returns the corresponding Edm type for the enum.
func DetermineEnumUnderlyingType(enumType reflect.Type) (string, error) {
	enumType = resolveEnumBaseType(enumType)
	if enumType == nil {
		return "", fmt.Errorf("enum type must be an integer")
	}

	switch enumType.Kind() {
	case reflect.Int8:
		return "Edm.SByte", nil
	case reflect.Uint8:
		return "Edm.Byte", nil
	case reflect.Int16:
		return "Edm.Int16", nil
	case reflect.Uint16:
		return "Edm.Int32", nil
	case reflect.Int32:
		return "Edm.Int32", nil
	case reflect.Uint32:
		return "Edm.Int64", nil
	case reflect.Int64:
		return "Edm.Int64", nil
	case reflect.Uint64:
		return "", fmt.Errorf("Edm.Int64 is the maximum supported underlying type; uint64 is not supported")
	case reflect.Int:
		if strconv.IntSize == 32 {
			return "Edm.Int32", nil
		}
		return "Edm.Int64", nil
	case reflect.Uint:
		if strconv.IntSize == 32 {
			return "Edm.Int64", nil
		}
		return "Edm.Int64", nil
	default:
		return "Edm.Int32", nil
	}
}
