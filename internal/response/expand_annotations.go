package response

import (
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
)

// ApplyExpandOptionToValue applies nested expand annotations and returns updated value and optional count.
func ApplyExpandOptionToValue(value interface{}, expandOpt *query.ExpandOption, targetMetadata *metadata.EntityMetadata) (interface{}, *int) {
	if expandOpt == nil {
		return value, nil
	}

	updatedValue := value

	// Apply nested $select first if specified
	if len(expandOpt.Select) > 0 && targetMetadata != nil {
		updatedValue = applySelectToExpandedValueWithMetadata(updatedValue, expandOpt.Select, targetMetadata)
	}

	// Apply nested $expand
	updatedValue = applyNestedExpandAnnotations(updatedValue, expandOpt.Expand, targetMetadata)

	// Handle $count on the expanded collection
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

// applySelectToExpandedValueWithMetadata applies $select to the expanded value using metadata
func applySelectToExpandedValueWithMetadata(value interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata) interface{} {
	if value == nil || len(selectedProperties) == 0 || entityMetadata == nil {
		return value
	}

	// Build selected properties map
	selectedPropMap := make(map[string]bool)
	for _, propName := range selectedProperties {
		selectedPropMap[propName] = true
	}

	val := reflect.ValueOf(value)

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return value
		}
		val = val.Elem()
	}

	// Handle collections
	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		resultSlice := make([]map[string]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			itemVal := val.Index(i)
			resultSlice[i] = filterEntityFieldsWithMetadata(itemVal, selectedPropMap)
		}
		return resultSlice
	}

	// Handle single entity
	return filterEntityFieldsWithMetadata(val, selectedPropMap)
}

// filterEntityFieldsWithMetadata creates a filtered map of entity fields based on selected properties
func filterEntityFieldsWithMetadata(entityVal reflect.Value, selectedPropMap map[string]bool) map[string]interface{} {
	if entityVal.Kind() == reflect.Ptr {
		if entityVal.IsNil() {
			return nil
		}
		entityVal = entityVal.Elem()
	}

	filteredEntity := make(map[string]interface{})

	// Add OData metadata annotations (always included)
	selectedPropMap["@odata.id"] = true
	selectedPropMap["@odata.type"] = true
	selectedPropMap["@odata.etag"] = true

	if entityVal.Kind() == reflect.Struct {
		for j := 0; j < entityVal.NumField(); j++ {
			field := entityVal.Type().Field(j)

			// Get the JsonName or use the field name
			jsonName := field.Name
			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				// Extract the field name from json tag (format: "fieldname,options...")
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" {
					jsonName = parts[0]
				}
			}

			// Check if selected
			if !selectedPropMap[field.Name] && !selectedPropMap[jsonName] {
				// Skip non-selected properties, but include OData annotations
				if !strings.HasPrefix(jsonName, "@") {
					continue
				}
			}

			fieldValue := entityVal.Field(j)
			if fieldValue.IsValid() && fieldValue.CanInterface() {
				filteredEntity[jsonName] = fieldValue.Interface()
			}
		}
	} else if entityVal.Kind() == reflect.Map {
		// If the value is already a map, just copy selected properties
		entityMapVal := entityVal.Interface()
		if entityMap, ok := entityMapVal.(map[string]interface{}); ok {
			for key, val := range entityMap {
				if selectedPropMap[key] || strings.HasPrefix(key, "@") {
					filteredEntity[key] = val
				}
			}
		}
	}

	return filteredEntity
}
