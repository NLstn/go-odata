package response

import (
	"reflect"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
)

// ApplyExpandOptionToValue applies nested expand annotations and returns updated value and optional count.
func ApplyExpandOptionToValue(value interface{}, expandOpt *query.ExpandOption, targetMetadata *metadata.EntityMetadata) (interface{}, *int) {
	if expandOpt == nil {
		return value, nil
	}

	updatedValue := applyNestedExpandAnnotations(value, expandOpt.Expand, targetMetadata)
	if !expandOpt.Count {
		return updatedValue, nil
	}

	count := expandedCollectionCount(updatedValue)
	if count == nil {
		return updatedValue, nil
	}

	return updatedValue, count
}

// ApplyExpandAnnotationsToMap applies expand annotations to a map-based entity representation.
func ApplyExpandAnnotationsToMap(entityMap map[string]interface{}, expandOptions []query.ExpandOption, metadata *metadata.EntityMetadata) map[string]interface{} {
	return applyNestedExpandAnnotationsToMap(entityMap, expandOptions, metadata)
}

func applyNestedExpandAnnotations(value interface{}, expandOptions []query.ExpandOption, metadata *metadata.EntityMetadata) interface{} {
	if value == nil || len(expandOptions) == 0 || metadata == nil {
		return value
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			return value
		}
		return applyNestedExpandAnnotations(val.Elem().Interface(), expandOptions, metadata)
	case reflect.Slice, reflect.Array:
		result := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			result[i] = applyNestedExpandAnnotations(val.Index(i).Interface(), expandOptions, metadata)
		}
		return result
	case reflect.Struct:
		return applyNestedExpandAnnotationsToStruct(val, expandOptions, metadata)
	case reflect.Map:
		if val.Type().Key().Kind() != reflect.String {
			return value
		}
		entityMap, ok := value.(map[string]interface{})
		if !ok {
			converted := make(map[string]interface{}, val.Len())
			iter := val.MapRange()
			for iter.Next() {
				converted[iter.Key().String()] = iter.Value().Interface()
			}
			entityMap = converted
		}
		return applyNestedExpandAnnotationsToMap(entityMap, expandOptions, metadata)
	default:
		return value
	}
}

func applyNestedExpandAnnotationsToStruct(entityVal reflect.Value, expandOptions []query.ExpandOption, metadata *metadata.EntityMetadata) map[string]interface{} {
	result := make(map[string]interface{})

	if !entityVal.IsValid() {
		return result
	}

	if entityVal.Kind() == reflect.Ptr {
		if entityVal.IsNil() {
			return result
		}
		entityVal = entityVal.Elem()
	}

	if entityVal.Kind() != reflect.Struct {
		return result
	}

	entityType := entityVal.Type()
	fieldInfos := getFieldInfos(entityType)

	for i := 0; i < entityVal.NumField(); i++ {
		info := fieldInfos[i]
		if !info.IsExported {
			continue
		}

		field := entityType.Field(i)
		fieldVal := entityVal.Field(i)
		jsonName := info.JsonName

		propMeta := metadata.FindProperty(field.Name)
		if propMeta != nil && propMeta.IsNavigationProp {
			expandOpt := query.FindExpandOption(expandOptions, propMeta.Name, propMeta.JsonName)
			if expandOpt != nil {
				targetMetadata, err := metadata.ResolveNavigationTarget(propMeta.Name)
				if err == nil {
					updatedValue, count := ApplyExpandOptionToValue(fieldVal.Interface(), expandOpt, targetMetadata)
					result[jsonName] = updatedValue
					if count != nil {
						result[jsonName+"@odata.count"] = *count
					}
					continue
				}
			}
		}

		result[jsonName] = fieldVal.Interface()
	}

	return result
}

func applyNestedExpandAnnotationsToMap(entityMap map[string]interface{}, expandOptions []query.ExpandOption, metadata *metadata.EntityMetadata) map[string]interface{} {
	for i := range expandOptions {
		opt := expandOptions[i]
		propMeta := metadata.FindNavigationProperty(opt.NavigationProperty)
		if propMeta == nil {
			continue
		}

		value, key := findMapValue(entityMap, propMeta)
		if key == "" {
			continue
		}

		targetMetadata, err := metadata.ResolveNavigationTarget(propMeta.Name)
		if err != nil {
			continue
		}

		updatedValue, count := ApplyExpandOptionToValue(value, &opt, targetMetadata)
		entityMap[key] = updatedValue
		if count != nil {
			entityMap[propMeta.JsonName+"@odata.count"] = *count
		}
	}

	return entityMap
}

func findMapValue(entityMap map[string]interface{}, propMeta *metadata.PropertyMetadata) (interface{}, string) {
	if propMeta == nil {
		return nil, ""
	}
	if val, ok := entityMap[propMeta.JsonName]; ok {
		return val, propMeta.JsonName
	}
	if val, ok := entityMap[propMeta.Name]; ok {
		return val, propMeta.Name
	}
	return nil, ""
}

func expandedCollectionCount(value interface{}) *int {
	if value == nil {
		return nil
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		count := val.Len()
		return &count
	case reflect.Ptr:
		if val.IsNil() {
			count := 0
			return &count
		}
		return expandedCollectionCount(val.Elem().Interface())
	default:
		return nil
	}
}
