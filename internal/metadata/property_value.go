package metadata

import "reflect"

// ReadPropertyValue reads a property from a struct or string-keyed map.
// Map values use the OData JSON property name as their canonical key.
func ReadPropertyValue(entity interface{}, property PropertyMetadata) (interface{}, bool) {
	if entity == nil {
		return nil, false
	}

	value := reflect.ValueOf(entity)
	for value.IsValid() && (value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return nil, false
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		for _, name := range propertyFieldNames(property) {
			field := value.FieldByName(name)
			if field.IsValid() && field.CanInterface() {
				return field.Interface(), true
			}
		}
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return nil, false
		}
		for _, name := range propertyMapKeys(property) {
			key := reflect.ValueOf(name).Convert(value.Type().Key())
			mapValue := value.MapIndex(key)
			if mapValue.IsValid() {
				return mapValue.Interface(), true
			}
		}
	}

	return nil, false
}

func propertyFieldNames(property PropertyMetadata) []string {
	return uniquePropertyNames(property.FieldName, property.Name)
}

func propertyMapKeys(property PropertyMetadata) []string {
	return uniquePropertyNames(property.JsonName, property.Name, property.FieldName)
}

func uniquePropertyNames(names ...string) []string {
	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}
