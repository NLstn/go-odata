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
		// Build the complete condition for logical operators
		query, args := buildFilterCondition(filter, entityMetadata)
		return db.Where(query, args...)
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
		return buildLogicalCondition(filter, entityMetadata)
	}

	// Build comparison condition
	query, args := buildComparisonCondition(filter, entityMetadata)

	// Apply NOT if needed
	if filter.IsNot {
		return fmt.Sprintf("NOT (%s)", query), args
	}

	return query, args
}

// buildLogicalCondition builds a logical condition (AND/OR)
func buildLogicalCondition(filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	leftQuery, leftArgs := buildFilterCondition(filter.Left, entityMetadata)
	rightQuery, rightArgs := buildFilterCondition(filter.Right, entityMetadata)

	var query string
	switch filter.Logical {
	case LogicalAnd:
		query = fmt.Sprintf("(%s) AND (%s)", leftQuery, rightQuery)
	case LogicalOr:
		query = fmt.Sprintf("(%s) OR (%s)", leftQuery, rightQuery)
	default:
		return "", nil
	}

	args := append(leftArgs, rightArgs...)
	if filter.IsNot {
		return fmt.Sprintf("NOT (%s)", query), args
	}
	return query, args
}

// buildComparisonCondition builds a comparison condition
func buildComparisonCondition(filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	// Check if this is a function comparison (function on left side of comparison)
	if filter.Left != nil && filter.Left.Operator != "" {
		return buildFunctionComparison(filter, entityMetadata)
	}

	columnName := GetColumnName(filter.Property, entityMetadata)

	switch filter.Operator {
	case OpEqual:
		return fmt.Sprintf("%s = ?", columnName), []interface{}{filter.Value}
	case OpNotEqual:
		return fmt.Sprintf("%s != ?", columnName), []interface{}{filter.Value}
	case OpGreaterThan:
		return fmt.Sprintf("%s > ?", columnName), []interface{}{filter.Value}
	case OpGreaterThanOrEqual:
		return fmt.Sprintf("%s >= ?", columnName), []interface{}{filter.Value}
	case OpLessThan:
		return fmt.Sprintf("%s < ?", columnName), []interface{}{filter.Value}
	case OpLessThanOrEqual:
		return fmt.Sprintf("%s <= ?", columnName), []interface{}{filter.Value}
	case OpContains:
		return fmt.Sprintf("%s LIKE ?", columnName), []interface{}{"%" + fmt.Sprint(filter.Value) + "%"}
	case OpStartsWith:
		return fmt.Sprintf("%s LIKE ?", columnName), []interface{}{fmt.Sprint(filter.Value) + "%"}
	case OpEndsWith:
		return fmt.Sprintf("%s LIKE ?", columnName), []interface{}{"%" + fmt.Sprint(filter.Value)}
	default:
		return "", nil
	}
}

// buildFunctionComparison builds a comparison with a function on the left side
func buildFunctionComparison(filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	funcExpr := filter.Left
	columnName := GetColumnName(funcExpr.Property, entityMetadata)

	funcSQL, funcArgs := buildFunctionSQL(funcExpr.Operator, columnName, funcExpr.Value)
	if funcSQL == "" {
		return "", nil
	}

	compSQL := buildComparisonSQL(filter.Operator, funcSQL)
	if compSQL == "" {
		return "", nil
	}

	allArgs := append(funcArgs, filter.Value)
	return compSQL, allArgs
}

// buildFunctionSQL generates SQL for a string function
func buildFunctionSQL(op FilterOperator, columnName string, value interface{}) (string, []interface{}) {
	switch op {
	case OpToLower:
		return fmt.Sprintf("LOWER(%s)", columnName), nil
	case OpToUpper:
		return fmt.Sprintf("UPPER(%s)", columnName), nil
	case OpTrim:
		return fmt.Sprintf("TRIM(%s)", columnName), nil
	case OpLength:
		return fmt.Sprintf("LENGTH(%s)", columnName), nil
	case OpIndexOf:
		return fmt.Sprintf("INSTR(%s, ?)", columnName), []interface{}{value}
	case OpConcat:
		return fmt.Sprintf("CONCAT(%s, ?)", columnName), []interface{}{value}
	case OpSubstring:
		return buildSubstringSQL(columnName, value)
	case OpAdd:
		return fmt.Sprintf("(%s + ?)", columnName), []interface{}{value}
	case OpSub:
		return fmt.Sprintf("(%s - ?)", columnName), []interface{}{value}
	case OpMul:
		return fmt.Sprintf("(%s * ?)", columnName), []interface{}{value}
	case OpDiv:
		return fmt.Sprintf("(%s / ?)", columnName), []interface{}{value}
	case OpMod:
		return fmt.Sprintf("(%s %% ?)", columnName), []interface{}{value}
	// Date functions
	case OpYear:
		return fmt.Sprintf("CAST(strftime('%%Y', %s) AS INTEGER)", columnName), nil
	case OpMonth:
		return fmt.Sprintf("CAST(strftime('%%m', %s) AS INTEGER)", columnName), nil
	case OpDay:
		return fmt.Sprintf("CAST(strftime('%%d', %s) AS INTEGER)", columnName), nil
	case OpHour:
		return fmt.Sprintf("CAST(strftime('%%H', %s) AS INTEGER)", columnName), nil
	case OpMinute:
		return fmt.Sprintf("CAST(strftime('%%M', %s) AS INTEGER)", columnName), nil
	case OpSecond:
		return fmt.Sprintf("CAST(strftime('%%S', %s) AS INTEGER)", columnName), nil
	case OpDate:
		return fmt.Sprintf("DATE(%s)", columnName), nil
	case OpTime:
		return fmt.Sprintf("TIME(%s)", columnName), nil
	default:
		return "", nil
	}
}

// buildSubstringSQL builds SQL for substring function
func buildSubstringSQL(columnName string, value interface{}) (string, []interface{}) {
	args, ok := value.([]interface{})
	if !ok {
		return "", nil
	}
	if len(args) == 1 {
		return fmt.Sprintf("SUBSTR(%s, ?, LENGTH(%s))", columnName, columnName), []interface{}{args[0]}
	}
	if len(args) == 2 {
		return fmt.Sprintf("SUBSTR(%s, ?, ?)", columnName), args
	}
	return "", nil
}

// buildComparisonSQL builds SQL for comparison operator
func buildComparisonSQL(op FilterOperator, leftSQL string) string {
	switch op {
	case OpEqual:
		return fmt.Sprintf("%s = ?", leftSQL)
	case OpNotEqual:
		return fmt.Sprintf("%s != ?", leftSQL)
	case OpGreaterThan:
		return fmt.Sprintf("%s > ?", leftSQL)
	case OpGreaterThanOrEqual:
		return fmt.Sprintf("%s >= ?", leftSQL)
	case OpLessThan:
		return fmt.Sprintf("%s < ?", leftSQL)
	case OpLessThanOrEqual:
		return fmt.Sprintf("%s <= ?", leftSQL)
	default:
		return ""
	}
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

	// Check if this is a function comparison (function on left side of comparison)
	if filter.Left != nil && filter.Left.Operator != "" {
		return buildSimpleFunctionComparison(filter)
	}

	// Convert property name to snake_case for database column name
	fieldName := toSnakeCase(filter.Property)
	return buildSimpleOperatorCondition(filter.Operator, fieldName, filter.Value)
}

// buildSimpleOperatorCondition builds SQL for a simple operator condition
func buildSimpleOperatorCondition(op FilterOperator, fieldName string, value interface{}) (string, []interface{}) {
	switch op {
	case OpEqual:
		return fmt.Sprintf("%s = ?", fieldName), []interface{}{value}
	case OpNotEqual:
		return fmt.Sprintf("%s != ?", fieldName), []interface{}{value}
	case OpGreaterThan:
		return fmt.Sprintf("%s > ?", fieldName), []interface{}{value}
	case OpGreaterThanOrEqual:
		return fmt.Sprintf("%s >= ?", fieldName), []interface{}{value}
	case OpLessThan:
		return fmt.Sprintf("%s < ?", fieldName), []interface{}{value}
	case OpLessThanOrEqual:
		return fmt.Sprintf("%s <= ?", fieldName), []interface{}{value}
	case OpContains:
		return fmt.Sprintf("%s LIKE ?", fieldName), []interface{}{"%" + fmt.Sprint(value) + "%"}
	case OpStartsWith:
		return fmt.Sprintf("%s LIKE ?", fieldName), []interface{}{fmt.Sprint(value) + "%"}
	case OpEndsWith:
		return fmt.Sprintf("%s LIKE ?", fieldName), []interface{}{"%" + fmt.Sprint(value)}
	default:
		return "", nil
	}
}

// buildSimpleFunctionComparison builds a comparison with a function on the left side (without metadata)
func buildSimpleFunctionComparison(filter *FilterExpression) (string, []interface{}) {
	funcExpr := filter.Left
	fieldName := toSnakeCase(funcExpr.Property)

	funcSQL, funcArgs := buildFunctionSQL(funcExpr.Operator, fieldName, funcExpr.Value)
	if funcSQL == "" {
		return "", nil
	}

	compSQL := buildComparisonSQL(filter.Operator, funcSQL)
	if compSQL == "" {
		return "", nil
	}

	allArgs := append(funcArgs, filter.Value)
	return compSQL, allArgs
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
