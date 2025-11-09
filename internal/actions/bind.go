package actions

import (
	"fmt"
	"reflect"
	"strings"
)

const boundStructKey = "__go_odata_bound_struct"

type structFieldBinding struct {
	Field    reflect.StructField
	Name     string
	Required bool
}

// BindParams converts the provided parameter map into the requested struct type.
// It reuses any pre-bound struct stored alongside the map to avoid duplicate decoding
// and validates that required parameters are present.
func BindParams[T any](params map[string]interface{}) (T, error) {
	var zero T

	if params == nil {
		return zero, fmt.Errorf("parameters map cannot be nil")
	}

	if bound, ok := params[boundStructKey]; ok {
		if converted, ok := tryConvertBound[T](bound); ok {
			return converted, nil
		}
	}

	var target T
	targetVal := reflect.ValueOf(&target).Elem()
	targetType := targetVal.Type()

	bindings, err := collectFieldBindings(targetType)
	if err != nil {
		return zero, err
	}

	for _, binding := range bindings {
		if binding.Required {
			if _, exists := params[binding.Name]; !exists {
				return zero, fmt.Errorf("required parameter '%s' is missing", binding.Name)
			}
		}
	}

	storedValue, err := newDecodeTarget(targetType)
	if err != nil {
		return zero, err
	}

	if err := applyBindings(storedValue, bindings, params); err != nil {
		return zero, err
	}

	resultVal := storedValue
	if targetType.Kind() != reflect.Ptr {
		resultVal = storedValue.Elem()
	}

	if params != nil {
		params[boundStructKey] = storedValue.Interface()
	}

	if !resultVal.Type().AssignableTo(targetVal.Type()) {
		return zero, fmt.Errorf("unable to bind parameters to target type %s", targetType)
	}

	targetVal.Set(resultVal)
	return target, nil
}

func tryConvertBound[T any](bound interface{}) (T, bool) {
	var target T
	boundVal := reflect.ValueOf(bound)
	if !boundVal.IsValid() {
		return target, false
	}

	targetVal := reflect.ValueOf(&target).Elem()

	if boundVal.Type().AssignableTo(targetVal.Type()) {
		targetVal.Set(boundVal)
		return target, true
	}

	if boundVal.Kind() == reflect.Ptr && !boundVal.IsNil() {
		elem := boundVal.Elem()
		if elem.Type().AssignableTo(targetVal.Type()) {
			targetVal.Set(elem)
			return target, true
		}
	}

	return target, false
}

func newDecodeTarget(targetType reflect.Type) (reflect.Value, error) {
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

func normalizeStructType(t reflect.Type) (reflect.Type, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("parameter binding target must be a struct or pointer to struct, got %s", t)
	}
	return t, nil
}

func collectFieldBindings(t reflect.Type) ([]structFieldBinding, error) {
	structType, err := normalizeStructType(t)
	if err != nil {
		return nil, err
	}
	return collectFieldBindingsRecursive(structType, nil), nil
}

func collectFieldBindingsRecursive(structType reflect.Type, prefix []int) []structFieldBinding {
	bindings := make([]structFieldBinding, 0)
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

		name, options := extractFieldName(field)
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

		bindings = append(bindings, structFieldBinding{
			Field:    field,
			Name:     name,
			Required: required,
		})
	}
	return bindings
}

func applyBindings(target reflect.Value, bindings []structFieldBinding, params map[string]interface{}) error {
	for _, binding := range bindings {
		value, exists := params[binding.Name]
		if !exists {
			continue
		}

		fieldVal, err := resolveFieldValue(target, binding.Field)
		if err != nil {
			return err
		}

		if err := assignValue(fieldVal, value); err != nil {
			return fmt.Errorf("failed to bind parameter '%s': %w", binding.Name, err)
		}
	}
	return nil
}

func resolveFieldValue(target reflect.Value, field reflect.StructField) (reflect.Value, error) {
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

func assignValue(fieldVal reflect.Value, value interface{}) error {
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

// ParameterDefinitionsFromStruct derives parameter definitions for the provided struct type.
// The type may be either a struct or a pointer to struct.
func ParameterDefinitionsFromStruct(t reflect.Type) ([]ParameterDefinition, error) {
	bindings, err := collectFieldBindings(t)
	if err != nil {
		return nil, err
	}

	defs := make([]ParameterDefinition, 0, len(bindings))
	for _, binding := range bindings {
		defs = append(defs, ParameterDefinition{
			Name:     binding.Name,
			Type:     binding.Field.Type,
			Required: binding.Required,
		})
	}

	return defs, nil
}

func extractFieldName(field reflect.StructField) (string, map[string]struct{}) {
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

func bindStructToParams(params map[string]interface{}, structType reflect.Type) error {
	if structType == nil {
		return nil
	}

	bindings, err := collectFieldBindings(structType)
	if err != nil {
		return err
	}

	for _, binding := range bindings {
		if binding.Required {
			if _, exists := params[binding.Name]; !exists {
				return fmt.Errorf("required parameter '%s' is missing", binding.Name)
			}
		}
	}

	storedValue, err := newDecodeTarget(structType)
	if err != nil {
		return err
	}

	if err := applyBindings(storedValue, bindings, params); err != nil {
		return err
	}

	params[boundStructKey] = storedValue.Interface()
	return nil
}
