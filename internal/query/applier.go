package query

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// ApplyQueryOptions applies parsed query options to a GORM database query
func ApplyQueryOptions(db *gorm.DB, options *QueryOptions, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if options == nil {
		return db
	}

	// Apply select at database level to fetch only needed columns
	if len(options.Select) > 0 {
		db = applySelect(db, options.Select, entityMetadata)
	}

	// Apply filter
	if options.Filter != nil {
		db = applyFilter(db, options.Filter, entityMetadata)
	}

	// Apply expand (preload navigation properties)
	if len(options.Expand) > 0 {
		db = applyExpand(db, options.Expand, entityMetadata)
	}

	// Apply orderby
	if len(options.OrderBy) > 0 {
		db = applyOrderBy(db, options.OrderBy, entityMetadata)
	}

	// Apply top and skip (for pagination)
	if options.Skip != nil {
		db = db.Offset(*options.Skip)
	}

	if options.Top != nil {
		db = db.Limit(*options.Top)
	}

	return db
}

// ApplyFilterOnly applies only the filter expression to a GORM database query
// This is useful for getting counts without applying pagination
func ApplyFilterOnly(db *gorm.DB, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if filter == nil {
		return db
	}
	return applyFilter(db, filter, entityMetadata)
}

// ApplyExpandOnly applies only the expand (preload) options to a GORM database query
// This is useful for loading related entities without applying other query options
func ApplyExpandOnly(db *gorm.DB, expand []ExpandOption, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if len(expand) == 0 {
		return db
	}
	return applyExpand(db, expand, entityMetadata)
}

// applySelect applies select clause to fetch only specified columns at database level
func applySelect(db *gorm.DB, selectedProperties []string, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if len(selectedProperties) == 0 {
		return db
	}

	// Build list of columns to select using struct field names
	// GORM will convert them to the correct column names
	columns := make([]string, 0, len(selectedProperties))
	selectedPropMap := make(map[string]bool)

	// Add requested properties
	for _, propName := range selectedProperties {
		propName = strings.TrimSpace(propName)
		// Find the property in metadata
		for _, prop := range entityMetadata.Properties {
			if (prop.JsonName == propName || prop.Name == propName) && !prop.IsNavigationProp {
				// Use struct field name - GORM will handle column name conversion
				columns = append(columns, prop.Name)
				selectedPropMap[prop.Name] = true
				break
			}
		}
	}

	// Always include key properties for OData responses (if not already selected)
	for _, keyProp := range entityMetadata.KeyProperties {
		if !selectedPropMap[keyProp.Name] {
			columns = append(columns, keyProp.Name)
		}
	}

	// Apply select to GORM query
	if len(columns) > 0 {
		db = db.Select(columns)
	}

	return db
}

// applyFilter applies filter expressions to the GORM query
func applyFilter(db *gorm.DB, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if filter == nil {
		return db
	}

	// Handle logical operators (and, or)
	if filter.Logical != "" {
		leftDB := applyFilter(db, filter.Left, entityMetadata)
		rightDB := applyFilter(db, filter.Right, entityMetadata)

		switch filter.Logical {
		case LogicalAnd:
			// For AND, we can chain the conditions
			return leftDB.Where(rightDB)
		case LogicalOr:
			// For OR, we need to build a more complex query
			// This is simplified and may need enhancement for complex cases
			leftQuery, leftArgs := buildFilterCondition(filter.Left, entityMetadata)
			rightQuery, rightArgs := buildFilterCondition(filter.Right, entityMetadata)

			combinedQuery := fmt.Sprintf("(%s) OR (%s)", leftQuery, rightQuery)
			combinedArgs := append(leftArgs, rightArgs...)

			return db.Where(combinedQuery, combinedArgs...)
		}
	}

	// Handle simple comparison operators
	query, args := buildFilterCondition(filter, entityMetadata)
	return db.Where(query, args...)
}

// buildFilterCondition builds a WHERE condition string and arguments for a filter expression
func buildFilterCondition(filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	// Handle logical operators recursively
	if filter.Logical != "" {
		leftQuery, leftArgs := buildFilterCondition(filter.Left, entityMetadata)
		rightQuery, rightArgs := buildFilterCondition(filter.Right, entityMetadata)

		switch filter.Logical {
		case LogicalAnd:
			query := fmt.Sprintf("(%s) AND (%s)", leftQuery, rightQuery)
			args := append(leftArgs, rightArgs...)
			if filter.IsNot {
				return fmt.Sprintf("NOT (%s)", query), args
			}
			return query, args
		case LogicalOr:
			query := fmt.Sprintf("(%s) OR (%s)", leftQuery, rightQuery)
			args := append(leftArgs, rightArgs...)
			if filter.IsNot {
				return fmt.Sprintf("NOT (%s)", query), args
			}
			return query, args
		}
	}

	// Get the database column name (snake_case)
	columnName := GetColumnName(filter.Property, entityMetadata)

	var query string
	var args []interface{}

	switch filter.Operator {
	case OpEqual:
		query = fmt.Sprintf("%s = ?", columnName)
		args = []interface{}{filter.Value}
	case OpNotEqual:
		query = fmt.Sprintf("%s != ?", columnName)
		args = []interface{}{filter.Value}
	case OpGreaterThan:
		query = fmt.Sprintf("%s > ?", columnName)
		args = []interface{}{filter.Value}
	case OpGreaterThanOrEqual:
		query = fmt.Sprintf("%s >= ?", columnName)
		args = []interface{}{filter.Value}
	case OpLessThan:
		query = fmt.Sprintf("%s < ?", columnName)
		args = []interface{}{filter.Value}
	case OpLessThanOrEqual:
		query = fmt.Sprintf("%s <= ?", columnName)
		args = []interface{}{filter.Value}
	case OpContains:
		query = fmt.Sprintf("%s LIKE ?", columnName)
		args = []interface{}{"%" + fmt.Sprint(filter.Value) + "%"}
	case OpStartsWith:
		query = fmt.Sprintf("%s LIKE ?", columnName)
		args = []interface{}{fmt.Sprint(filter.Value) + "%"}
	case OpEndsWith:
		query = fmt.Sprintf("%s LIKE ?", columnName)
		args = []interface{}{"%" + fmt.Sprint(filter.Value)}
	default:
		return "", nil
	}

	// Apply NOT if needed
	if filter.IsNot {
		return fmt.Sprintf("NOT (%s)", query), args
	}
	
	return query, args
}

// applyExpand applies expand (preload) options to the GORM query
func applyExpand(db *gorm.DB, expand []ExpandOption, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	for _, expandOpt := range expand {
		// Find the navigation property
		navProp := findNavigationProperty(expandOpt.NavigationProperty, entityMetadata)
		if navProp == nil {
			continue // Skip invalid navigation property
		}

		// Build preload with conditions if nested options exist
		if expandOpt.Select != nil || expandOpt.Filter != nil || expandOpt.OrderBy != nil ||
			expandOpt.Top != nil || expandOpt.Skip != nil {
			// Preload with conditions
			db = db.Preload(navProp.Name, func(db *gorm.DB) *gorm.DB {
				// Apply nested filter
				if expandOpt.Filter != nil {
					// Note: This would require navigation target metadata
					// For now, just add basic support
					db = applyFilterForExpand(db, expandOpt.Filter)
				}

				// Apply nested orderby
				if len(expandOpt.OrderBy) > 0 {
					for _, item := range expandOpt.OrderBy {
						direction := "ASC"
						if item.Descending {
							direction = "DESC"
						}
						// Convert property name to snake_case for database column name
						columnName := toSnakeCase(item.Property)
						db = db.Order(fmt.Sprintf("%s %s", columnName, direction))
					}
				}

				// Apply nested pagination
				if expandOpt.Skip != nil {
					db = db.Offset(*expandOpt.Skip)
				}
				if expandOpt.Top != nil {
					db = db.Limit(*expandOpt.Top)
				}

				return db
			})
		} else {
			// Simple preload without conditions
			db = db.Preload(navProp.Name)
		}
	}
	return db
}

// applyFilterForExpand applies filter to expanded navigation property
// This is a simplified version that doesn't have full metadata context
func applyFilterForExpand(db *gorm.DB, filter *FilterExpression) *gorm.DB {
	if filter == nil {
		return db
	}

	// Handle logical operators
	if filter.Logical != "" {
		leftDB := applyFilterForExpand(db, filter.Left)
		rightDB := applyFilterForExpand(db, filter.Right)

		switch filter.Logical {
		case LogicalAnd:
			return leftDB.Where(rightDB)
		case LogicalOr:
			leftQuery, leftArgs := buildSimpleFilterCondition(filter.Left)
			rightQuery, rightArgs := buildSimpleFilterCondition(filter.Right)
			combinedQuery := fmt.Sprintf("(%s) OR (%s)", leftQuery, rightQuery)
			combinedArgs := append(leftArgs, rightArgs...)
			return db.Where(combinedQuery, combinedArgs...)
		}
	}

	// Handle simple comparison
	query, args := buildSimpleFilterCondition(filter)
	return db.Where(query, args...)
}

// buildSimpleFilterCondition builds a filter condition without metadata
func buildSimpleFilterCondition(filter *FilterExpression) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	if filter.Logical != "" {
		leftQuery, leftArgs := buildSimpleFilterCondition(filter.Left)
		rightQuery, rightArgs := buildSimpleFilterCondition(filter.Right)

		switch filter.Logical {
		case LogicalAnd:
			query := fmt.Sprintf("(%s) AND (%s)", leftQuery, rightQuery)
			args := append(leftArgs, rightArgs...)
			return query, args
		case LogicalOr:
			query := fmt.Sprintf("(%s) OR (%s)", leftQuery, rightQuery)
			args := append(leftArgs, rightArgs...)
			return query, args
		}
	}

	// Convert property name to snake_case for database column name
	fieldName := toSnakeCase(filter.Property)

	switch filter.Operator {
	case OpEqual:
		return fmt.Sprintf("%s = ?", fieldName), []interface{}{filter.Value}
	case OpNotEqual:
		return fmt.Sprintf("%s != ?", fieldName), []interface{}{filter.Value}
	case OpGreaterThan:
		return fmt.Sprintf("%s > ?", fieldName), []interface{}{filter.Value}
	case OpGreaterThanOrEqual:
		return fmt.Sprintf("%s >= ?", fieldName), []interface{}{filter.Value}
	case OpLessThan:
		return fmt.Sprintf("%s < ?", fieldName), []interface{}{filter.Value}
	case OpLessThanOrEqual:
		return fmt.Sprintf("%s <= ?", fieldName), []interface{}{filter.Value}
	case OpContains:
		return fmt.Sprintf("%s LIKE ?", fieldName), []interface{}{"%" + fmt.Sprint(filter.Value) + "%"}
	case OpStartsWith:
		return fmt.Sprintf("%s LIKE ?", fieldName), []interface{}{fmt.Sprint(filter.Value) + "%"}
	case OpEndsWith:
		return fmt.Sprintf("%s LIKE ?", fieldName), []interface{}{"%" + fmt.Sprint(filter.Value)}
	default:
		return "", nil
	}
}

// findNavigationProperty finds a navigation property by name
func findNavigationProperty(propName string, entityMetadata *metadata.EntityMetadata) *metadata.PropertyMetadata {
	for _, prop := range entityMetadata.Properties {
		if (prop.JsonName == propName || prop.Name == propName) && prop.IsNavigationProp {
			return &prop
		}
	}
	return nil
}

// applyOrderBy applies order by clauses to the GORM query
func applyOrderBy(db *gorm.DB, orderBy []OrderByItem, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	for _, item := range orderBy {
		if !propertyExists(item.Property, entityMetadata) {
			continue // Skip unrecognized properties in $orderby
		}
		fieldName := GetPropertyFieldName(item.Property, entityMetadata)
		direction := "ASC"
		if item.Descending {
			direction = "DESC"
		}
		db = db.Order(fmt.Sprintf("%s %s", fieldName, direction))
	}
	return db
}

// ApplySelect converts struct results to map format with only selected properties
// This is called after the query to convert the result to the correct format for OData responses
// The database query already fetched only the needed columns via applySelect
func ApplySelect(results interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata) interface{} {
	if len(selectedProperties) == 0 {
		return results
	}

	// Get the slice value
	sliceValue := reflect.ValueOf(results)
	if sliceValue.Kind() != reflect.Slice {
		return results
	}

	// Create a new slice of maps to hold the results
	filteredResults := make([]map[string]interface{}, sliceValue.Len())

	// Build a map of selected properties for quick lookup
	selectedPropMap := make(map[string]bool)
	for _, propName := range selectedProperties {
		selectedPropMap[strings.TrimSpace(propName)] = true
	}

	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i)
		filteredItem := make(map[string]interface{})

		// Extract only the selected properties (database already filtered columns)
		for _, prop := range entityMetadata.Properties {
			// Check if this property was selected
			if selectedPropMap[prop.JsonName] || selectedPropMap[prop.Name] {
				// Get the field value
				fieldValue := item.FieldByName(prop.Name)
				if fieldValue.IsValid() && fieldValue.CanInterface() {
					filteredItem[prop.JsonName] = fieldValue.Interface()
				}
			}
		}

		filteredResults[i] = filteredItem
	}

	return filteredResults
}
