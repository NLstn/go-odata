package actions

import (
	"fmt"
	"reflect"
)

// BindParams converts the provided parameter map into the requested struct type.
// It reuses any pre-bound struct stored alongside the map to avoid duplicate decoding
// and validates that required parameters are present.
func BindParams[T any](params map[string]interface{}) (T, error) {
	var zero T

	if params == nil {
		return zero, fmt.Errorf("parameters map cannot be nil")
	}

	if bound, ok := params[BoundStructKey]; ok {
		if converted, ok := tryConvertBound[T](bound); ok {
			return converted, nil
		}
	}

	var target T
	targetVal := reflect.ValueOf(&target).Elem()
	targetType := targetVal.Type()

	bindings, err := CollectFieldBindings(targetType)
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

	storedValue, err := NewDecodeTarget(targetType)
	if err != nil {
		return zero, err
	}

	if err := ApplyBindings(storedValue, bindings, params); err != nil {
		return zero, err
	}

	resultVal := storedValue
	if targetType.Kind() != reflect.Ptr {
		resultVal = storedValue.Elem()
	}

	params[BoundStructKey] = storedValue.Interface()

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
