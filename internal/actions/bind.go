package actions

import (
	"fmt"
	"reflect"
)

// ParameterDefinitionsFromStruct derives parameter definitions for the provided struct type.
// The type may be either a struct or a pointer to struct.
func ParameterDefinitionsFromStruct(t reflect.Type) ([]ParameterDefinition, error) {
	bindings, err := CollectFieldBindings(t)
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

func bindStructToParams(params map[string]interface{}, structType reflect.Type) error {
	if structType == nil {
		return nil
	}

	bindings, err := CollectFieldBindings(structType)
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

	storedValue, err := NewDecodeTarget(structType)
	if err != nil {
		return err
	}

	if err := ApplyBindings(storedValue, bindings, params); err != nil {
		return err
	}

	params[BoundStructKey] = storedValue.Interface()
	return nil
}
