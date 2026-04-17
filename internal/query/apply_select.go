package query

import (
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// selectContainsWildcard returns true if the selected properties list contains the wildcard '*'.
// Per OData v4.01 section 5.1.3, '*' means all declared structural properties.
func selectContainsWildcard(selectedProperties []string) bool {
	for _, p := range selectedProperties {
		if p == "*" {
			return true
		}
	}
	return false
}

// applySelect applies select clause to fetch only specified columns at database level
func applySelect(db *gorm.DB, selectedProperties []string, expandOptions []ExpandOption, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if len(selectedProperties) == 0 {
		return db
	}

	// Wildcard '*' selects all structural properties — no column restriction needed at DB level.
	if selectContainsWildcard(selectedProperties) {
		return db
	}

	columns := make([]string, 0, len(selectedProperties))
	columnSet := make(map[string]bool)
	selectedPropMap := make(map[string]bool)
	tableName := entityMetadata.TableName
	dialect := getDatabaseDialect(db)

	addColumn := func(columnName string) {
		if columnName == "" {
			return
		}
		qualifiedColumn := quoteTableName(dialect, tableName) + "." + quoteIdent(dialect, columnName)
		if columnSet[qualifiedColumn] {
			return
		}
		columnSet[qualifiedColumn] = true
		columns = append(columns, qualifiedColumn)
	}

	for _, propName := range selectedProperties {
		propName = strings.TrimSpace(propName)
		for _, prop := range entityMetadata.Properties {
			if (prop.JsonName == propName || prop.Name == propName) && !prop.IsNavigationProp && !prop.IsComplexType && !prop.IsStream && !prop.IsComputed {
				// Use GetColumnName for proper column name resolution (handles GORM tags and metadata)
				columnName := GetColumnName(prop.Name, entityMetadata)
				addColumn(columnName)
				selectedPropMap[prop.Name] = true
				break
			}
		}
	}

	for _, keyProp := range entityMetadata.KeyProperties {
		if !selectedPropMap[keyProp.Name] {
			// Use GetColumnName for proper column name resolution (handles GORM tags and metadata)
			columnName := GetColumnName(keyProp.Name, entityMetadata)
			addColumn(columnName)
		}
	}

	for _, expandOpt := range expandOptions {
		navPropName := strings.TrimSpace(expandOpt.NavigationProperty)
		if navPropName == "" {
			continue
		}

		for _, prop := range entityMetadata.Properties {
			if !prop.IsNavigationProp || prop.NavigationIsArray {
				continue
			}
			if prop.Name != navPropName && prop.JsonName != navPropName {
				continue
			}

			// Only add FK columns that belong to the parent/current entity (belongs-to relationship).
			// For has-one relationships, the FK is on the child entity and must not be added here,
			// as it would produce invalid SQL referencing a column that does not exist on the parent table.
			for _, fkColumn := range strings.Split(prop.ForeignKeyColumnName, ",") {
				fkColumn = strings.TrimSpace(fkColumn)
				if fkColumn != "" && isForeignKeyOnEntity(fkColumn, entityMetadata) {
					addColumn(fkColumn)
				}
			}
			break
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

	// Wildcard '*' means all structural properties — same as no $select.
	if selectContainsWildcard(selectedProperties) {
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
							var nestedExpand []ExpandOption
							if expandOpt != nil {
								nestedExpand = expandOpt.Expand
							}
							fieldVal = applySelectToExpandedEntity(fieldVal, nestedSelect, nestedExpand)
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

	// Wildcard '*' means all structural properties — same as no $select.
	if selectContainsWildcard(selectedProperties) {
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
						var nestedExpand []ExpandOption
						if expandOpt != nil {
							nestedExpand = expandOpt.Expand
						}
						fieldVal = applySelectToExpandedEntity(fieldVal, nestedSelect, nestedExpand)
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
func ApplySelectToMapResults(results []map[string]interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool) []map[string]interface{} {
	if len(selectedProperties) == 0 {
		return results
	}

	// Wildcard '*' means all structural properties — same as no $select.
	if selectContainsWildcard(selectedProperties) {
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

// isForeignKeyOnEntity checks whether fkColumn is a column that belongs to the given entity.
// This is used to distinguish belongs-to relationships (FK on parent entity) from has-one
// relationships (FK on child entity). Returns true only when the column is found among the
// non-navigation, non-complex properties of the entity.
func isForeignKeyOnEntity(fkColumn string, entityMeta *metadata.EntityMetadata) bool {
	for _, prop := range entityMeta.Properties {
		if prop.IsNavigationProp || prop.IsComplexType {
			continue
		}
		if prop.ColumnName == fkColumn {
			return true
		}
	}
	return false
}
