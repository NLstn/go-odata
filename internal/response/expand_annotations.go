package response

import (
	"fmt"
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

	// Per OData spec §5.1.3, $expand=Nav/$ref returns only entity references.
	if expandOpt.IsRef {
		return toEntityReferences(value, targetMetadata), nil
	}

	updatedValue := value

	// Apply nested $select first if specified
	// NOTE: Only apply if Select has values - this is for nested $select in expand syntax like $expand=Products($select=ID)
	// Top-level select with navigation paths (like $select=Product/Name) is handled differently
	if len(expandOpt.Select) > 0 && targetMetadata != nil {
		selectSet := make(map[string]bool)
		for _, s := range expandOpt.Select {
			selectSet[s] = true
		}

		// Include navigation properties from nested $expand so they are not stripped
		// by the select filter before applyNestedExpandAnnotations can process them
		for _, nestedExpand := range expandOpt.Expand {
			selectSet[nestedExpand.NavigationProperty] = true
		}

		// Convert back to slice
		deduped := make([]string, 0, len(selectSet))
		for s := range selectSet {
			deduped = append(deduped, s)
		}

		updatedValue = applySelectToExpandedValueWithMetadata(updatedValue, deduped, targetMetadata)
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

// toEntityReferences converts an entity or slice of entities to the minimal entity-reference
// representation required by OData spec §5.1.3 when $expand uses the /$ref suffix.
// Each reference is a map containing only @odata.id built from the entity set name and key.
func toEntityReferences(value interface{}, md *metadata.EntityMetadata) interface{} {
	if value == nil || md == nil {
		return nil
	}

	val := reflect.ValueOf(value)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		refs := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			refs[i] = entityToReference(val.Index(i), md)
		}
		return refs
	default:
		return entityToReference(val, md)
	}
}

// entityToReference builds a single {"@odata.id": "EntitySet(key)"} map for one entity.
func entityToReference(val reflect.Value, md *metadata.EntityMetadata) map[string]interface{} {
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	keySegment := buildKeySegmentForRef(val, md)
	if keySegment == "" || md.EntitySetName == "" {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"@odata.id": md.EntitySetName + "(" + keySegment + ")",
	}
}

// buildKeySegmentForRef extracts the key segment string from an entity value using metadata.
func buildKeySegmentForRef(val reflect.Value, md *metadata.EntityMetadata) string {
	if len(md.KeyProperties) == 0 {
		return ""
	}

	if val.Kind() == reflect.Map {
		return buildKeySegmentForRefFromMap(val, md)
	}

	if val.Kind() != reflect.Struct {
		return ""
	}

	if len(md.KeyProperties) == 1 {
		kp := md.KeyProperties[0]
		fv := findFieldByName(val, kp.FieldName, kp.Name)
		if !fv.IsValid() {
			return ""
		}
		return formatRefKeyValue(fv)
	}

	var b strings.Builder
	for i, kp := range md.KeyProperties {
		fv := findFieldByName(val, kp.FieldName, kp.Name)
		if !fv.IsValid() {
			continue
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(kp.JsonName)
		b.WriteByte('=')
		if fv.Kind() == reflect.String {
			b.WriteByte('\'')
			b.WriteString(fv.String())
			b.WriteByte('\'')
		} else {
			b.WriteString(formatRefKeyValue(fv))
		}
	}
	return b.String()
}

func buildKeySegmentForRefFromMap(val reflect.Value, md *metadata.EntityMetadata) string {
	if len(md.KeyProperties) == 1 {
		kp := md.KeyProperties[0]
		v := val.MapIndex(reflect.ValueOf(kp.JsonName))
		if !v.IsValid() {
			v = val.MapIndex(reflect.ValueOf(kp.Name))
		}
		if !v.IsValid() {
			return ""
		}
		elem := v.Elem()
		if elem.Kind() == reflect.String {
			return "'" + elem.String() + "'"
		}
		return fmt.Sprintf("%v", elem.Interface())
	}

	var b strings.Builder
	for i, kp := range md.KeyProperties {
		v := val.MapIndex(reflect.ValueOf(kp.JsonName))
		if !v.IsValid() {
			v = val.MapIndex(reflect.ValueOf(kp.Name))
		}
		if !v.IsValid() {
			continue
		}
		if i > 0 {
			b.WriteByte(',')
		}
		elem := v.Elem()
		b.WriteString(kp.JsonName)
		b.WriteByte('=')
		if elem.Kind() == reflect.String {
			b.WriteByte('\'')
			b.WriteString(elem.String())
			b.WriteByte('\'')
		} else {
			b.WriteString(fmt.Sprintf("%v", elem.Interface()))
		}
	}
	return b.String()
}

// findFieldByName locates a struct field by its Go field name or JSON name.
func findFieldByName(val reflect.Value, fieldName, fallbackName string) reflect.Value {
	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Name == fieldName || f.Name == fallbackName {
			return val.Field(i)
		}
		if tag := f.Tag.Get("json"); tag != "" {
			parts := strings.SplitN(tag, ",", 2)
			if parts[0] == fieldName || parts[0] == fallbackName {
				return val.Field(i)
			}
		}
	}
	return reflect.Value{}
}

func formatRefKeyValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return "'" + v.String() + "'"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
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
		if info.JsonName == "" {
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
					// Apply $top truncation for collection navigation properties.
					// When a nested expand uses $top, ApplyPerParentExpand fetches top+1 items
					// (one extra for nextLink detection). We must trim back to top here because
					// this path (struct → map conversion) does not go through
					// processNavigationPropertyOrderedWithMetadata, which is where the
					// trimming normally happens for top-level expanded collections.
					fv := fieldVal
					if expandOpt.Top != nil && propMeta.NavigationIsArray && fv.Kind() == reflect.Slice {
						fv, _ = TruncateExpandedCollectionToTop(fv, *expandOpt.Top)
					}
					updatedValue, count := ApplyExpandOptionToValue(fv.Interface(), expandOpt, targetMetadata)
					result[jsonName] = updatedValue
					if count != nil {
						result[jsonName+"@odata.count"] = *count
					}
					continue
				}
			}
		}

		result[jsonName] = EncodeEdmBinary(fieldVal.Interface())
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

	keyPropMap := make(map[string]bool)
	for _, keyProp := range entityMetadata.KeyProperties {
		keyPropMap[keyProp.Name] = true
		if keyProp.JsonName != "" {
			keyPropMap[keyProp.JsonName] = true
		}
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
			resultSlice[i] = filterEntityFieldsWithMetadata(itemVal, selectedPropMap, keyPropMap)
		}
		return resultSlice
	}

	// Handle single entity
	return filterEntityFieldsWithMetadata(val, selectedPropMap, keyPropMap)
}

// filterEntityFieldsWithMetadata creates a filtered map of entity fields based on selected properties
func filterEntityFieldsWithMetadata(entityVal reflect.Value, selectedPropMap map[string]bool, keyPropMap map[string]bool) map[string]interface{} {
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

			// Key properties and OData annotations are always included.
			isSelected := selectedPropMap[field.Name] || selectedPropMap[jsonName]
			isKey := keyPropMap[field.Name] || keyPropMap[jsonName]
			if !isSelected && !isKey {
				// Skip non-selected properties, but include OData annotations and potential key fields
				if !strings.HasPrefix(jsonName, "@") {
					continue
				}
			}

			fieldValue := entityVal.Field(j)
			if fieldValue.IsValid() && fieldValue.CanInterface() {
				filteredEntity[jsonName] = EncodeEdmBinary(fieldValue.Interface())
			}
		}
	} else if entityVal.Kind() == reflect.Map {
		// If the value is already a map, just copy selected properties
		entityMapVal := entityVal.Interface()
		if entityMap, ok := entityMapVal.(map[string]interface{}); ok {
			for key, val := range entityMap {
				if selectedPropMap[key] || keyPropMap[key] || strings.HasPrefix(key, "@") {
					filteredEntity[key] = EncodeEdmBinary(val)
				}
			}
		}
	}

	return filteredEntity
}
