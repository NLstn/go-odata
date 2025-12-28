package query

import (
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// applySelect applies select clause to fetch only specified columns at database level
func applySelect(db *gorm.DB, selectedProperties []string, entityMetadata *metadata.EntityMetadata, selectSpecified bool) *gorm.DB {
	if len(selectedProperties) == 0 && !selectSpecified {
		return db
	}

	columns := make([]string, 0, len(selectedProperties))
	selectedPropMap := make(map[string]bool)
	tableName := entityMetadata.TableName
	dialect := getDatabaseDialect(db)

	for _, propName := range selectedProperties {
		propName = strings.TrimSpace(propName)
		for _, prop := range entityMetadata.Properties {
			if (prop.JsonName == propName || prop.Name == propName) && !prop.IsNavigationProp && !prop.IsComplexType && !prop.IsStream {
				// Use GetColumnName for proper column name resolution (handles GORM tags and metadata)
				columnName := GetColumnName(prop.Name, entityMetadata)
				// Qualify with quoted table and column names to avoid ambiguous column references when JOINs are present
				// This is necessary for PostgreSQL and also helps with SQLite when multiple tables have the same column names
				// quoteIdent properly quotes identifiers for both PostgreSQL and SQLite
				qualifiedColumn := quoteIdent(dialect, tableName) + "." + quoteIdent(dialect, columnName)
				columns = append(columns, qualifiedColumn)
				selectedPropMap[prop.Name] = true
				break
			}
		}
	}

	for _, keyProp := range entityMetadata.KeyProperties {
		if !selectedPropMap[keyProp.Name] {
			// Use GetColumnName for proper column name resolution (handles GORM tags and metadata)
			columnName := GetColumnName(keyProp.Name, entityMetadata)
			// Qualify with quoted table and column names to avoid ambiguous column references when JOINs are present
			qualifiedColumn := quoteIdent(dialect, tableName) + "." + quoteIdent(dialect, columnName)
			columns = append(columns, qualifiedColumn)
		}
	}

	if len(columns) > 0 {
		db = db.Select(columns)
	}

	return db
}

// ApplySelect converts struct results to map format with only selected properties
// This is called after the query to convert the result to the correct format for OData responses
func ApplySelect(results interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata, expandOptions []ExpandOption, selectSpecified bool) interface{} {
	if !selectSpecified {
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
						nestedSelectSpecified := false
						if expandOpt != nil && expandOpt.SelectSpecified {
							nestedSelect = expandOpt.Select
							nestedSelectSpecified = true
						} else if len(navPropSelects[prop.JsonName]) > 0 {
							nestedSelect = navPropSelects[prop.JsonName]
							nestedSelectSpecified = true
						} else if len(navPropSelects[prop.Name]) > 0 {
							nestedSelect = navPropSelects[prop.Name]
							nestedSelectSpecified = true
						}

						if nestedSelectSpecified && fieldVal != nil {
							fieldVal = applySelectToExpandedEntity(fieldVal, nestedSelect, nestedSelectSpecified)
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
func ApplySelectToEntity(entity interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata, expandOptions []ExpandOption, selectSpecified bool) interface{} {
	if !selectSpecified {
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
					nestedSelectSpecified := false
					if expandOpt != nil && expandOpt.SelectSpecified {
						nestedSelect = expandOpt.Select
						nestedSelectSpecified = true
					} else if len(navPropSelects[prop.JsonName]) > 0 {
						nestedSelect = navPropSelects[prop.JsonName]
						nestedSelectSpecified = true
					} else if len(navPropSelects[prop.Name]) > 0 {
						nestedSelect = navPropSelects[prop.Name]
						nestedSelectSpecified = true
					}

					if nestedSelectSpecified && fieldVal != nil {
						fieldVal = applySelectToExpandedEntity(fieldVal, nestedSelect, nestedSelectSpecified)
					}
				}

				filteredEntity[prop.JsonName] = fieldVal
			}
		}
	}

	return filteredEntity
}

// ApplySelectToMapResults filters map results to only include selected properties
// This is used when $compute is present and results are returned as []map[string]interface{}
// The computedAliases parameter specifies which properties are computed
func ApplySelectToMapResults(results []map[string]interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool, selectSpecified bool) []map[string]interface{} {
	if !selectSpecified {
		return results
	}

	// Build a map of selected properties (including navigation paths)
	selectedPropMap := make(map[string]bool)
	for _, propName := range selectedProperties {
		propName = strings.TrimSpace(propName)
		selectedPropMap[propName] = true
	}

	// Build a map of key properties that must always be included
	keyPropMap := make(map[string]bool)
	for _, keyProp := range entityMetadata.KeyProperties {
		keyPropMap[keyProp.JsonName] = true
	}

	filteredResults := make([]map[string]interface{}, len(results))

	for i, result := range results {
		filteredItem := make(map[string]interface{})

		for key, value := range result {
			isSelected := selectedPropMap[key]
			isKey := keyPropMap[key]

			// Include the property if:
			// 1. It's explicitly selected, OR
			// 2. It's a key property (always included)
			if isSelected || isKey {
				filteredItem[key] = value
			}
		}

		filteredResults[i] = filteredItem
	}

	return filteredResults
}
