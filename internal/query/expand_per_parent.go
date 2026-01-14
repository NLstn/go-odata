package query

import (
	"fmt"
	"reflect"
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

		if navProp.ForeignKeyColumnName == "" || strings.Contains(navProp.ForeignKeyColumnName, ",") {
			continue
		}

		targetMetadata, err := entityMetadata.ResolveNavigationTarget(expandOpt.NavigationProperty)
		if err != nil {
			continue
		}

		parentPropName := resolveParentReferenceProperty(navProp, entityMetadata)
		if parentPropName == "" {
			continue
		}

		for _, parentVal := range parentValues {
			parentStruct := dereferenceValue(parentVal)
			if !parentStruct.IsValid() || parentStruct.Kind() != reflect.Struct {
				continue
			}

			parentKeyValue, ok := getStructFieldValue(parentStruct, parentPropName)
			if !ok {
				continue
			}

			childResults := reflect.New(reflect.SliceOf(targetMetadata.EntityType))
			childDB := db.Session(&gorm.Session{NewDB: true})
			childDB = childDB.Model(reflect.New(targetMetadata.EntityType).Interface())
			childDB = childDB.Where(fmt.Sprintf("%s = ?", navProp.ForeignKeyColumnName), parentKeyValue)
			childDB = ApplyExpandOption(childDB, expandOpt, targetMetadata)

			if err := childDB.Find(childResults.Interface()).Error; err != nil {
				return err
			}

			if len(expandOpt.Expand) > 0 {
				if err := ApplyPerParentExpand(db, childResults.Interface(), expandOpt.Expand, targetMetadata); err != nil {
					return err
				}
			}

			if err := setNavigationValue(parentStruct, navProp, childResults.Elem()); err != nil {
				return err
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

func resolveParentReferenceProperty(navProp *metadata.PropertyMetadata, entityMetadata *metadata.EntityMetadata) string {
	if navProp == nil || entityMetadata == nil {
		return ""
	}

	for _, principal := range navProp.ReferentialConstraints {
		if principal != "" {
			return principal
		}
	}

	if len(entityMetadata.KeyProperties) > 0 {
		return entityMetadata.KeyProperties[0].Name
	}

	if entityMetadata.KeyProperty != nil {
		return entityMetadata.KeyProperty.Name
	}

	return ""
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
