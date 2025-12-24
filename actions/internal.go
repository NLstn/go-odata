package actions

import (
	"fmt"
	"reflect"
	"strings"
)

// BoundStructKey is the key used to store bound structs in parameter maps.
// This allows caching of decoded structs to avoid duplicate decoding.
const BoundStructKey = "__go_odata_bound_struct"

// StructFieldBinding represents a mapping between a struct field and a parameter name.
type StructFieldBinding struct {
	Field    reflect.StructField
	Name     string
	Required bool
}

// NewDecodeTarget creates a new reflect.Value that can be used as a decode target
// for the given struct type or pointer to struct type.
func NewDecodeTarget(targetType reflect.Type) (reflect.Value, error) {
	if targetType.Kind() == reflect.Ptr {
		elem := targetType.Elem()
		if elem.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("BindParams target type must be a struct or pointer to struct, got %s", targetType)
		}
		value := reflect.New(elem)
		return value, nil
	}

	if targetType.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("BindParams target type must be a struct or pointer to struct, got %s", targetType)
	}

	value := reflect.New(targetType)
	return value, nil
}

// NormalizeStructType returns the underlying struct type, unwrapping pointer types.
func NormalizeStructType(t reflect.Type) (reflect.Type, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("parameter binding target must be a struct or pointer to struct, got %s", t)
	}
	return t, nil
}

// CollectFieldBindings analyzes a struct type and returns bindings for all fields
// that should be populated from parameters.
func CollectFieldBindings(t reflect.Type) ([]StructFieldBinding, error) {
	structType, err := NormalizeStructType(t)
	if err != nil {
		return nil, err
	}
	return collectFieldBindingsRecursive(structType, nil), nil
}

func collectFieldBindingsRecursive(structType reflect.Type, prefix []int) []StructFieldBinding {
	bindings := make([]StructFieldBinding, 0)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" {
			continue
		}

		index := append([]int{}, prefix...)
		index = append(index, field.Index...)
		field.Index = index

		if field.Anonymous {
			embeddedType := field.Type
			if embeddedType.Kind() == reflect.Ptr {
				embeddedType = embeddedType.Elem()
			}
			if embeddedType.Kind() != reflect.Struct {
				continue
			}
			bindings = append(bindings, collectFieldBindingsRecursive(embeddedType, index)...)
			continue
		}

		name, options := ExtractFieldName(field)
		if name == "-" {
			continue
		}

		required := true
		if field.Type.Kind() == reflect.Ptr {
			required = false
		}
		if _, ok := options["omitempty"]; ok {
			required = false
		}

		bindings = append(bindings, StructFieldBinding{
			Field:    field,
			Name:     name,
			Required: required,
		})
	}
	return bindings
}

// ApplyBindings populates the target struct with values from the params map
// according to the provided bindings.
func ApplyBindings(target reflect.Value, bindings []StructFieldBinding, params map[string]interface{}) error {
	for _, binding := range bindings {
		value, exists := params[binding.Name]
		if !exists {
			continue
		}

		fieldVal, err := ResolveFieldValue(target, binding.Field)
		if err != nil {
			return err
		}

		if err := AssignValue(fieldVal, value); err != nil {
			return fmt.Errorf("failed to bind parameter '%s': %w", binding.Name, err)
		}
	}
	return nil
}

// ResolveFieldValue navigates to the field within the target struct,
// creating intermediate pointer values as needed.
func ResolveFieldValue(target reflect.Value, field reflect.StructField) (reflect.Value, error) {
	if target.Kind() != reflect.Ptr {
		return reflect.Value{}, fmt.Errorf("target must be a pointer to struct, got %s", target.Type())
	}
	if target.IsNil() {
		target.Set(reflect.New(target.Type().Elem()))
	}

	current := target.Elem()
	for i, idx := range field.Index {
		if i == len(field.Index)-1 {
			return current.Field(idx), nil
		}

		next := current.Field(idx)
		switch next.Kind() {
		case reflect.Ptr:
			if next.IsNil() {
				next.Set(reflect.New(next.Type().Elem()))
			}
			current = next.Elem()
		case reflect.Struct:
			current = next
		default:
			return reflect.Value{}, fmt.Errorf("intermediate field %s is not addressable", next.Type())
		}
	}

	return reflect.Value{}, fmt.Errorf("invalid field index for %s", field.Name)
}

// AssignValue assigns a parameter value to a struct field, performing type
// conversion as needed.
func AssignValue(fieldVal reflect.Value, value interface{}) error {
	if !fieldVal.CanSet() {
		return fmt.Errorf("field of type %s cannot be set", fieldVal.Type())
	}

	if value == nil {
		fieldVal.Set(reflect.Zero(fieldVal.Type()))
		return nil
	}

	val := reflect.ValueOf(value)
	if !val.IsValid() {
		fieldVal.Set(reflect.Zero(fieldVal.Type()))
		return nil
	}

	if val.Type().AssignableTo(fieldVal.Type()) {
		fieldVal.Set(val)
		return nil
	}

	if val.Type().ConvertibleTo(fieldVal.Type()) {
		fieldVal.Set(val.Convert(fieldVal.Type()))
		return nil
	}

	if fieldVal.Kind() == reflect.Ptr {
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				fieldVal.Set(reflect.Zero(fieldVal.Type()))
				return nil
			}
			if val.Type().AssignableTo(fieldVal.Type()) {
				fieldVal.Set(val)
				return nil
			}
			if val.Elem().Type().AssignableTo(fieldVal.Type().Elem()) {
				elemType := fieldVal.Type().Elem()
				newVal := reflect.New(elemType)
				newVal.Elem().Set(val.Elem())
				fieldVal.Set(newVal)
				return nil
			}
			if val.Elem().Type().ConvertibleTo(fieldVal.Type().Elem()) {
				elemType := fieldVal.Type().Elem()
				newVal := reflect.New(elemType)
				newVal.Elem().Set(val.Elem().Convert(elemType))
				fieldVal.Set(newVal)
				return nil
			}
		}

		elemType := fieldVal.Type().Elem()
		if val.Type().AssignableTo(elemType) {
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(elemType))
			}
			fieldVal.Elem().Set(val)
			return nil
		}
		if val.Type().ConvertibleTo(elemType) {
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(elemType))
			}
			fieldVal.Elem().Set(val.Convert(elemType))
			return nil
		}
	}

	if val.Kind() == reflect.Ptr && !val.IsNil() {
		if val.Elem().Type().AssignableTo(fieldVal.Type()) {
			fieldVal.Set(val.Elem())
			return nil
		}
		if val.Elem().Type().ConvertibleTo(fieldVal.Type()) {
			fieldVal.Set(val.Elem().Convert(fieldVal.Type()))
			return nil
		}
	}

	return fmt.Errorf("cannot assign value of type %s to field of type %s", val.Type(), fieldVal.Type())
}

// ExtractFieldName determines the parameter name for a struct field by checking
// mapstructure and json tags.
func ExtractFieldName(field reflect.StructField) (string, map[string]struct{}) {
	tags := []string{field.Tag.Get("mapstructure"), field.Tag.Get("json")}
	var merged map[string]struct{}
	for _, tag := range tags {
		if tag == "" {
			continue
		}

		name, options := parseTag(tag)
		merged = mergeOptions(merged, options)

		if name == "" {
			continue
		}
		if name == "-" {
			return "-", ensureOptionsMap(merged)
		}
		return name, ensureOptionsMap(merged)
	}

	return field.Name, ensureOptionsMap(merged)
}

func parseTag(tag string) (string, map[string]struct{}) {
	options := map[string]struct{}{}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	for _, opt := range parts[1:] {
		opt = strings.TrimSpace(opt)
		if opt == "" {
			continue
		}
		options[opt] = struct{}{}
	}
	return name, options
}

func mergeOptions(dst, src map[string]struct{}) map[string]struct{} {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]struct{}, len(src))
	}
	for opt := range src {
		dst[opt] = struct{}{}
	}
	return dst
}

func ensureOptionsMap(options map[string]struct{}) map[string]struct{} {
	if options == nil {
		return map[string]struct{}{}
	}
	return options
}
