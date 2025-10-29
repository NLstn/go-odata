package query

import (
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// applySelect applies select clause to fetch only specified columns at database level
func applySelect(db *gorm.DB, selectedProperties []string, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if len(selectedProperties) == 0 {
		return db
	}

	columns := make([]string, 0, len(selectedProperties))
	selectedPropMap := make(map[string]bool)

	for _, propName := range selectedProperties {
		propName = strings.TrimSpace(propName)
		for _, prop := range entityMetadata.Properties {
			if (prop.JsonName == propName || prop.Name == propName) && !prop.IsNavigationProp && !prop.IsComplexType && !prop.IsStream {
				columns = append(columns, prop.Name)
				selectedPropMap[prop.Name] = true
				break
			}
		}
	}

	for _, keyProp := range entityMetadata.KeyProperties {
		if !selectedPropMap[keyProp.Name] {
			columns = append(columns, keyProp.Name)
		}
	}

	if len(columns) > 0 {
		db = db.Select(columns)
	}

	return db
}

// ApplySelect converts struct results to map format with only selected properties
// This is called after the query to convert the result to the correct format for OData responses
func ApplySelect(results interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata, expandOptions []ExpandOption) interface{} {
	if len(selectedProperties) == 0 {
		return results
	}

	sliceValue := reflect.ValueOf(results)
	if sliceValue.Kind() != reflect.Slice {
		return results
	}

	filteredResults := make([]map[string]interface{}, sliceValue.Len())

	selectedPropMap := make(map[string]bool)
	navPropSelects := make(map[string][]string)

	for _, propName := range selectedProperties {
		propName = strings.TrimSpace(propName)
		if strings.Contains(propName, "/") {
			parts := strings.SplitN(propName, "/", 2)
			navProp := strings.TrimSpace(parts[0])
			subProp := strings.TrimSpace(parts[1])
			navPropSelects[navProp] = append(navPropSelects[navProp], subProp)
		} else {
			selectedPropMap[propName] = true
		}
	}

	expandedPropMap := make(map[string]*ExpandOption)
	for i := range expandOptions {
		expandedPropMap[expandOptions[i].NavigationProperty] = &expandOptions[i]
	}

	keyPropMap := make(map[string]bool)
	for _, keyProp := range entityMetadata.KeyProperties {
		keyPropMap[keyProp.Name] = true
	}

	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i)
		filteredItem := make(map[string]interface{})

		for _, prop := range entityMetadata.Properties {
			if prop.IsComplexType {
				continue
			}

			isSelected := selectedPropMap[prop.JsonName] || selectedPropMap[prop.Name]
			isKey := keyPropMap[prop.Name]
			isExpanded := prop.IsNavigationProp && (expandedPropMap[prop.Name] != nil || expandedPropMap[prop.JsonName] != nil)
			hasNavSelect := len(navPropSelects[prop.JsonName]) > 0 || len(navPropSelects[prop.Name]) > 0

			if isSelected || isKey || isExpanded || hasNavSelect {
				fieldValue := item.FieldByName(prop.Name)
				if fieldValue.IsValid() && fieldValue.CanInterface() {
					fieldVal := fieldValue.Interface()

					if prop.IsNavigationProp && (isExpanded || hasNavSelect) {
						var expandOpt *ExpandOption
						if expandedPropMap[prop.Name] != nil {
							expandOpt = expandedPropMap[prop.Name]
						} else if expandedPropMap[prop.JsonName] != nil {
							expandOpt = expandedPropMap[prop.JsonName]
						}

						var nestedSelect []string
						if expandOpt != nil && len(expandOpt.Select) > 0 {
							nestedSelect = expandOpt.Select
						} else if len(navPropSelects[prop.JsonName]) > 0 {
							nestedSelect = navPropSelects[prop.JsonName]
						} else if len(navPropSelects[prop.Name]) > 0 {
							nestedSelect = navPropSelects[prop.Name]
						}

						if len(nestedSelect) > 0 && fieldVal != nil {
							fieldVal = applySelectToExpandedEntity(fieldVal, nestedSelect)
						}
					}

					filteredItem[prop.JsonName] = fieldVal
				}
			}
		}

		filteredResults[i] = filteredItem
	}

	return filteredResults
}

// ApplySelectToEntity applies the $select filter to a single entity
func ApplySelectToEntity(entity interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata, expandOptions []ExpandOption) interface{} {
	if len(selectedProperties) == 0 {
		return entity
	}

	selectedPropMap := make(map[string]bool)
	navPropSelects := make(map[string][]string)

	for _, propName := range selectedProperties {
		propName = strings.TrimSpace(propName)
		if strings.Contains(propName, "/") {
			parts := strings.SplitN(propName, "/", 2)
			navProp := strings.TrimSpace(parts[0])
			subProp := strings.TrimSpace(parts[1])
			navPropSelects[navProp] = append(navPropSelects[navProp], subProp)
		} else {
			selectedPropMap[propName] = true
		}
	}

	expandedPropMap := make(map[string]*ExpandOption)
	for i := range expandOptions {
		expandedPropMap[expandOptions[i].NavigationProperty] = &expandOptions[i]
	}

	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	filteredEntity := make(map[string]interface{})

	keyPropMap := make(map[string]bool)
	for _, keyProp := range entityMetadata.KeyProperties {
		keyPropMap[keyProp.Name] = true
	}

	for _, prop := range entityMetadata.Properties {
		if prop.IsComplexType {
			continue
		}

		isSelected := selectedPropMap[prop.JsonName] || selectedPropMap[prop.Name]
		isKey := keyPropMap[prop.Name]
		isExpanded := prop.IsNavigationProp && (expandedPropMap[prop.Name] != nil || expandedPropMap[prop.JsonName] != nil)
		hasNavSelect := len(navPropSelects[prop.JsonName]) > 0 || len(navPropSelects[prop.Name]) > 0

		if isSelected || isKey || isExpanded || hasNavSelect {
			fieldValue := entityValue.FieldByName(prop.Name)
			if fieldValue.IsValid() && fieldValue.CanInterface() {
				fieldVal := fieldValue.Interface()

				if prop.IsNavigationProp && (isExpanded || hasNavSelect) {
					var expandOpt *ExpandOption
					if expandedPropMap[prop.Name] != nil {
						expandOpt = expandedPropMap[prop.Name]
					} else if expandedPropMap[prop.JsonName] != nil {
						expandOpt = expandedPropMap[prop.JsonName]
					}

					var nestedSelect []string
					if expandOpt != nil && len(expandOpt.Select) > 0 {
						nestedSelect = expandOpt.Select
					} else if len(navPropSelects[prop.JsonName]) > 0 {
						nestedSelect = navPropSelects[prop.JsonName]
					} else if len(navPropSelects[prop.Name]) > 0 {
						nestedSelect = navPropSelects[prop.Name]
					}

					if len(nestedSelect) > 0 && fieldVal != nil {
						fieldVal = applySelectToExpandedEntity(fieldVal, nestedSelect)
					}
				}

				filteredEntity[prop.JsonName] = fieldVal
			}
		}
	}

	return filteredEntity
}
