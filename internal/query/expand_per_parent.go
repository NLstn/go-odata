package query

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// ApplyPerParentExpand applies $expand options with $top/$skip for collection navigation properties per parent.
func ApplyPerParentExpand(db *gorm.DB, results interface{}, expandOptions []ExpandOption, entityMetadata *metadata.EntityMetadata) error {
	if db == nil || results == nil || len(expandOptions) == 0 || entityMetadata == nil {
		return nil
	}

	parentValues, err := collectParentValues(results)
	if err != nil {
		return err
	}

	for _, expandOpt := range expandOptions {
		navProp := findNavigationProperty(expandOpt.NavigationProperty, entityMetadata)
		if !needsPerParentExpand(expandOpt, navProp) {
			continue
		}

		targetMetadata, err := entityMetadata.ResolveNavigationTarget(expandOpt.NavigationProperty)
		if err != nil {
			continue
		}

		constraints := resolveParentReferenceProperty(navProp, entityMetadata, targetMetadata)
		if len(constraints) == 0 {
			continue
		}

		parentKeyMap, parentKeys := collectParentKeyValues(parentValues, constraints)
		if len(parentKeys) == 0 {
			continue
		}

		childResults, err := fetchChildrenByParentKeys(db, expandOpt, targetMetadata, constraints, parentKeys)
		if err != nil {
			return err
		}

		childGroups := groupChildrenByParentKey(childResults, constraints)
		sliceType := reflect.SliceOf(targetMetadata.EntityType)

		for key, parents := range parentKeyMap {
			parentChildren := childGroups[key]
			if !parentChildren.IsValid() {
				parentChildren = reflect.MakeSlice(sliceType, 0, 0)
			}

			parentChildren = applyPerParentPagination(parentChildren, expandOpt.Skip, expandOpt.Top)
			childResultsForParent := reflect.New(sliceType)
			childResultsForParent.Elem().Set(parentChildren)

			if len(expandOpt.Expand) > 0 {
				if err := ApplyPerParentExpand(db, childResultsForParent.Interface(), expandOpt.Expand, targetMetadata); err != nil {
					return err
				}
			}

			for _, parentVal := range parents {
				parentStruct := dereferenceValue(parentVal)
				if !parentStruct.IsValid() || parentStruct.Kind() != reflect.Struct {
					continue
				}
				if err := setNavigationValue(parentStruct, navProp, childResultsForParent.Elem()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func collectParentValues(results interface{}) ([]reflect.Value, error) {
	val := reflect.ValueOf(results)
	if !val.IsValid() {
		return nil, fmt.Errorf("invalid results")
	}

	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, fmt.Errorf("nil results")
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		values := make([]reflect.Value, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i)
			if item.Kind() == reflect.Struct && item.CanAddr() {
				values = append(values, item.Addr())
				continue
			}
			values = append(values, item)
		}
		return values, nil
	case reflect.Struct:
		if val.CanAddr() {
			return []reflect.Value{val.Addr()}, nil
		}
		return []reflect.Value{val}, nil
	default:
		return nil, fmt.Errorf("unsupported result kind %s", val.Kind())
	}
}

func dereferenceValue(val reflect.Value) reflect.Value {
	for val.IsValid() && val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return reflect.Value{}
		}
		val = val.Elem()
	}
	return val
}

type parentReferenceConstraint struct {
	dependentProperty string
	dependentColumn   string
	principalProperty string
}

type parentKey struct {
	key    string
	values []interface{}
}

func resolveParentReferenceProperty(navProp *metadata.PropertyMetadata, entityMetadata *metadata.EntityMetadata, targetMetadata *metadata.EntityMetadata) []parentReferenceConstraint {
	if navProp == nil || entityMetadata == nil || targetMetadata == nil {
		return nil
	}

	if len(navProp.ReferentialConstraints) > 0 {
		return expandCompositeConstraints(navProp.ReferentialConstraints, entityMetadata, targetMetadata)
	}

	return fallbackReferenceConstraints(navProp, entityMetadata, targetMetadata)
}

func expandCompositeConstraints(constraints map[string]string, entityMetadata *metadata.EntityMetadata, targetMetadata *metadata.EntityMetadata) []parentReferenceConstraint {
	keys := make([]string, 0, len(constraints))
	for key := range constraints {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	resolved := make([]parentReferenceConstraint, 0)
	for _, dependentKey := range keys {
		principalKey := constraints[dependentKey]
		dependents := splitCSV(dependentKey)
		principals := splitCSV(principalKey)
		count := min(len(dependents), len(principals))
		for i := 0; i < count; i++ {
			dependent := resolvePropertyName(dependents[i], targetMetadata)
			principal := resolvePropertyName(principals[i], entityMetadata)
			resolved = append(resolved, parentReferenceConstraint{
				dependentProperty: dependent,
				dependentColumn:   GetColumnName(dependent, targetMetadata),
				principalProperty: principal,
			})
		}
	}
	return resolved
}

func fallbackReferenceConstraints(navProp *metadata.PropertyMetadata, entityMetadata *metadata.EntityMetadata, targetMetadata *metadata.EntityMetadata) []parentReferenceConstraint {
	if navProp.ForeignKeyColumnName == "" {
		return nil
	}

	principalProps := make([]string, 0, len(entityMetadata.KeyProperties))
	for _, keyProp := range entityMetadata.KeyProperties {
		principalProps = append(principalProps, keyProp.Name)
	}
	if len(principalProps) == 0 && entityMetadata.KeyProperty != nil {
		principalProps = append(principalProps, entityMetadata.KeyProperty.Name)
	}
	if len(principalProps) == 0 {
		return nil
	}

	dependentColumns := splitCSV(navProp.ForeignKeyColumnName)
	count := min(len(principalProps), len(dependentColumns))
	resolved := make([]parentReferenceConstraint, 0, count)
	for i := 0; i < count; i++ {
		column := dependentColumns[i]
		dependentProp := resolvePropertyByColumn(column, targetMetadata)
		resolved = append(resolved, parentReferenceConstraint{
			dependentProperty: dependentProp,
			dependentColumn:   column,
			principalProperty: principalProps[i],
		})
	}
	return resolved
}

// resolvePropertyName resolves a property name or JSON name to the canonical property name.
// Uses EntityMetadata.FindProperty for efficient lookup.
func resolvePropertyName(name string, metadata *metadata.EntityMetadata) string {
	if metadata == nil {
		return name
	}
	trimmed := strings.TrimSpace(name)
	prop := metadata.FindProperty(trimmed)
	if prop != nil {
		return prop.Name
	}
	return trimmed
}

// resolvePropertyByColumn resolves a database column name to the property name.
// Note: This uses linear search as metadata API doesn't provide indexed column lookup.
func resolvePropertyByColumn(column string, metadata *metadata.EntityMetadata) string {
	if metadata == nil {
		return ""
	}
	trimmed := strings.TrimSpace(column)
	for _, prop := range metadata.Properties {
		if prop.ColumnName == trimmed {
			return prop.Name
		}
	}
	return ""
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func collectParentKeyValues(parentValues []reflect.Value, constraints []parentReferenceConstraint) (map[string][]reflect.Value, []parentKey) {
	parentKeyMap := make(map[string][]reflect.Value)
	parentKeys := make([]parentKey, 0)

	for _, parentVal := range parentValues {
		parentStruct := dereferenceValue(parentVal)
		if !parentStruct.IsValid() || parentStruct.Kind() != reflect.Struct {
			continue
		}

		values := make([]interface{}, 0, len(constraints))
		valid := true
		for _, constraint := range constraints {
			if constraint.principalProperty == "" {
				valid = false
				break
			}
			parentKeyValue, ok := getStructFieldValue(parentStruct, constraint.principalProperty)
			if !ok {
				valid = false
				break
			}
			parentKeyValue = normalizeKeyValue(parentKeyValue)
			if parentKeyValue == nil {
				valid = false
				break
			}
			values = append(values, parentKeyValue)
		}

		if !valid {
			continue
		}

		key := buildCompositeKey(values)
		if _, exists := parentKeyMap[key]; !exists {
			parentKeys = append(parentKeys, parentKey{key: key, values: values})
		}
		parentKeyMap[key] = append(parentKeyMap[key], parentVal)
	}

	return parentKeyMap, parentKeys
}

func fetchChildrenByParentKeys(db *gorm.DB, expandOpt ExpandOption, targetMetadata *metadata.EntityMetadata, constraints []parentReferenceConstraint, parentKeys []parentKey) (reflect.Value, error) {
	sliceType := reflect.SliceOf(targetMetadata.EntityType)
	allResults := reflect.MakeSlice(sliceType, 0, 0)
	if len(parentKeys) == 0 {
		return allResults, nil
	}

	batchSize := maxBatchSize(len(constraints))
	expandOpt = stripPagination(expandOpt)

	for start := 0; start < len(parentKeys); start += batchSize {
		end := start + batchSize
		if end > len(parentKeys) {
			end = len(parentKeys)
		}
		batch := parentKeys[start:end]

		childResults := reflect.New(sliceType)
		childDB := db.Session(&gorm.Session{NewDB: true})
		childDB = childDB.Model(reflect.New(targetMetadata.EntityType).Interface())
		childDB = applyParentKeyFilter(childDB, constraints, batch)
		childDB = ApplyExpandOption(childDB, expandOpt, targetMetadata)

		if err := childDB.Find(childResults.Interface()).Error; err != nil {
			return reflect.Value{}, err
		}

		allResults = reflect.AppendSlice(allResults, childResults.Elem())
	}

	return allResults, nil
}

func applyParentKeyFilter(db *gorm.DB, constraints []parentReferenceConstraint, parentKeys []parentKey) *gorm.DB {
	if len(constraints) == 1 {
		column := constraints[0].dependentColumn
		values := make([]interface{}, 0, len(parentKeys))
		for _, key := range parentKeys {
			values = append(values, key.values[0])
		}
		return db.Where(fmt.Sprintf("%s IN ?", column), values)
	}

	conditions := make([]string, 0, len(parentKeys))
	args := make([]interface{}, 0, len(parentKeys)*len(constraints))
	for _, key := range parentKeys {
		parts := make([]string, 0, len(constraints))
		for index, constraint := range constraints {
			parts = append(parts, fmt.Sprintf("%s = ?", constraint.dependentColumn))
			args = append(args, key.values[index])
		}
		conditions = append(conditions, "("+strings.Join(parts, " AND ")+")")
	}

	return db.Where(strings.Join(conditions, " OR "), args...)
}

func groupChildrenByParentKey(children reflect.Value, constraints []parentReferenceConstraint) map[string]reflect.Value {
	grouped := make(map[string]reflect.Value)
	children = dereferenceValue(children)
	if !children.IsValid() || (children.Kind() != reflect.Slice && children.Kind() != reflect.Array) {
		return grouped
	}

	for i := 0; i < children.Len(); i++ {
		child := children.Index(i)
		childStruct := dereferenceValue(child)
		if !childStruct.IsValid() || childStruct.Kind() != reflect.Struct {
			continue
		}

		values := make([]interface{}, 0, len(constraints))
		valid := true
		for _, constraint := range constraints {
			if constraint.dependentProperty == "" {
				valid = false
				break
			}
			childValue, ok := getStructFieldValue(childStruct, constraint.dependentProperty)
			if !ok {
				valid = false
				break
			}
			childValue = normalizeKeyValue(childValue)
			if childValue == nil {
				valid = false
				break
			}
			values = append(values, childValue)
		}
		if !valid {
			continue
		}

		key := buildCompositeKey(values)
		current := grouped[key]
		if !current.IsValid() {
			current = reflect.MakeSlice(children.Type(), 0, 0)
		}
		current = reflect.Append(current, child)
		grouped[key] = current
	}

	return grouped
}

func applyPerParentPagination(values reflect.Value, skip *int, top *int) reflect.Value {
	if !values.IsValid() || values.Kind() != reflect.Slice {
		return values
	}

	start := 0
	if skip != nil && *skip > 0 {
		start = *skip
	}
	if start > values.Len() {
		start = values.Len()
	}

	end := values.Len()
	if top != nil && *top >= 0 {
		if start+*top < end {
			end = start + *top
		}
	}

	return values.Slice(start, end)
}

func stripPagination(expandOpt ExpandOption) ExpandOption {
	expandOpt.Top = nil
	expandOpt.Skip = nil
	return expandOpt
}

func buildCompositeKey(values []interface{}) string {
	var builder strings.Builder
	for _, value := range values {
		normalized := normalizeKeyValue(value)
		builder.WriteString(fmt.Sprintf("%T:%v|", normalized, normalized))
	}
	return builder.String()
}

func normalizeKeyValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}
	rv := reflect.ValueOf(value)
	for rv.IsValid() && rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.IsValid() {
		return rv.Interface()
	}
	return nil
}

func maxBatchSize(constraintCount int) int {
	if constraintCount <= 0 {
		return 1
	}
	// maxArgs is a safety limit to avoid exceeding database parameter limits
	// (e.g., SQLite has a default limit of 999 parameters, so 900 provides a margin).
	const maxArgs = 900
	batch := maxArgs / constraintCount
	if batch < 1 {
		return 1
	}
	return batch
}



func getStructFieldValue(parentStruct reflect.Value, propName string) (interface{}, bool) {
	if !parentStruct.IsValid() || parentStruct.Kind() != reflect.Struct {
		return nil, false
	}

	field := parentStruct.FieldByName(propName)
	if field.IsValid() && field.CanInterface() {
		return field.Interface(), true
	}

	return nil, false
}

func setNavigationValue(parentStruct reflect.Value, navProp *metadata.PropertyMetadata, value reflect.Value) error {
	if navProp == nil {
		return nil
	}

	if parentStruct.IsValid() && parentStruct.Kind() == reflect.Struct && !parentStruct.CanSet() && parentStruct.CanAddr() {
		parentStruct = parentStruct.Addr().Elem()
	}

	fieldName := navProp.FieldName
	if fieldName == "" {
		fieldName = navProp.Name
	}

	field := parentStruct.FieldByName(fieldName)
	if !field.IsValid() || !field.CanSet() {
		return nil
	}

	if field.Kind() != reflect.Slice {
		return nil
	}

	if value.Type().AssignableTo(field.Type()) {
		field.Set(value)
		return nil
	}
	if value.Type().ConvertibleTo(field.Type()) {
		field.Set(value.Convert(field.Type()))
		return nil
	}

	return nil
}
