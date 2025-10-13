package query

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// ShouldUseMapResults returns true if the query options require map results instead of entity results
func ShouldUseMapResults(options *QueryOptions) bool {
	return options != nil && len(options.Apply) > 0
}

// ApplyQueryOptions applies parsed query options to a GORM database query
func ApplyQueryOptions(db *gorm.DB, options *QueryOptions, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if options == nil {
		return db
	}

	// Apply transformations (if present, they take precedence over other query options)
	if len(options.Apply) > 0 {
		return applyTransformations(db, options.Apply, entityMetadata)
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
	case OpIn:
		// Handle IN operator with a collection of values
		values, ok := filter.Value.([]interface{})
		if !ok {
			return "", nil
		}
		if len(values) == 0 {
			// Empty IN clause - return false condition
			return "1 = 0", []interface{}{}
		}
		placeholders := make([]string, len(values))
		for i := range values {
			placeholders[i] = "?"
		}
		return fmt.Sprintf("%s IN (%s)", columnName, strings.Join(placeholders, ", ")), values
	case OpContains:
		return fmt.Sprintf("%s LIKE ?", columnName), []interface{}{"%" + fmt.Sprint(filter.Value) + "%"}
	case OpStartsWith:
		return fmt.Sprintf("%s LIKE ?", columnName), []interface{}{fmt.Sprint(filter.Value) + "%"}
	case OpEndsWith:
		return fmt.Sprintf("%s LIKE ?", columnName), []interface{}{"%" + fmt.Sprint(filter.Value)}
	case OpHas:
		// Bitwise AND for enum flags: (column & value) = value
		return fmt.Sprintf("(%s & ?) = ?", columnName), []interface{}{filter.Value, filter.Value}
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
	case OpHas:
		// Bitwise AND for enum flags: (column & value) = value
		return fmt.Sprintf("(%s & ?)", columnName), []interface{}{value}
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
	// Math functions
	case OpCeiling:
		// SQLite doesn't have native CEIL, so we implement it
		return fmt.Sprintf("CASE WHEN %s = CAST(%s AS INTEGER) THEN %s ELSE CAST(%s AS INTEGER) + (CASE WHEN %s > 0 THEN 1 ELSE 0 END) END",
			columnName, columnName, columnName, columnName, columnName), nil
	case OpFloor:
		// SQLite doesn't have native FLOOR, so we implement it
		return fmt.Sprintf("CASE WHEN %s = CAST(%s AS INTEGER) THEN %s ELSE CAST(%s AS INTEGER) - (CASE WHEN %s < 0 THEN 1 ELSE 0 END) END",
			columnName, columnName, columnName, columnName, columnName), nil
	case OpRound:
		return fmt.Sprintf("ROUND(%s)", columnName), nil
	// Type conversion functions
	case OpCast:
		// value should be the type name (e.g., "Edm.String", "Edm.Int32")
		if typeName, ok := value.(string); ok {
			sqlType := edmTypeToSQLType(typeName)
			return fmt.Sprintf("CAST(%s AS %s)", columnName, sqlType), nil
		}
		return "", nil
	default:
		return "", nil
	}
}

// edmTypeToSQLType converts OData EDM types to SQLite types
func edmTypeToSQLType(edmType string) string {
	switch edmType {
	case "Edm.String":
		return "TEXT"
	case "Edm.Int32", "Edm.Int16", "Edm.Byte", "Edm.SByte":
		return "INTEGER"
	case "Edm.Int64":
		return "INTEGER"
	case "Edm.Decimal", "Edm.Double", "Edm.Single":
		return "REAL"
	case "Edm.Boolean":
		return "INTEGER" // SQLite uses 0/1 for boolean
	case "Edm.DateTimeOffset", "Edm.Date", "Edm.TimeOfDay":
		return "TEXT" // SQLite stores dates as text
	case "Edm.Guid":
		return "TEXT"
	case "Edm.Binary":
		return "BLOB"
	default:
		return "TEXT" // Default to TEXT for unknown types
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
	case OpHas:
		// Bitwise AND for enum flags: (column & value) = value
		return fmt.Sprintf("(%s & ?) = ?", fieldName), []interface{}{value, value}
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

	// Always include key properties (OData requirement)
	keyPropMap := make(map[string]bool)
	for _, keyProp := range entityMetadata.KeyProperties {
		keyPropMap[keyProp.Name] = true
	}

	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i)
		filteredItem := make(map[string]interface{})

		// Extract selected properties and key properties
		for _, prop := range entityMetadata.Properties {
			// Include if selected OR if it's a key property
			isSelected := selectedPropMap[prop.JsonName] || selectedPropMap[prop.Name]
			isKey := keyPropMap[prop.Name]

			if isSelected || isKey {
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

// ApplySelectToEntity applies the $select filter to a single entity
func ApplySelectToEntity(entity interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata) interface{} {
	if len(selectedProperties) == 0 {
		return entity
	}

	// Build a map of selected properties for quick lookup
	selectedPropMap := make(map[string]bool)
	for _, propName := range selectedProperties {
		selectedPropMap[strings.TrimSpace(propName)] = true
	}

	// Get the entity value
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Create a map to hold the filtered result
	filteredEntity := make(map[string]interface{})

	// Always include key properties (OData requirement)
	keyPropMap := make(map[string]bool)
	for _, keyProp := range entityMetadata.KeyProperties {
		keyPropMap[keyProp.Name] = true
	}

	// Extract selected properties and key properties
	for _, prop := range entityMetadata.Properties {
		// Include if selected OR if it's a key property
		isSelected := selectedPropMap[prop.JsonName] || selectedPropMap[prop.Name]
		isKey := keyPropMap[prop.Name]

		if isSelected || isKey {
			// Get the field value
			fieldValue := entityValue.FieldByName(prop.Name)
			if fieldValue.IsValid() && fieldValue.CanInterface() {
				filteredEntity[prop.JsonName] = fieldValue.Interface()
			}
		}
	}

	return filteredEntity
}

// applyTransformations applies apply transformations to the GORM query
func applyTransformations(db *gorm.DB, transformations []ApplyTransformation, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	// Set the table/model for the query - use the statement destination if available
	// Otherwise, we'll need the caller to set this
	if db.Statement != nil && db.Statement.Dest == nil {
		// Create an instance of the entity type to set the model
		modelInstance := reflect.New(entityMetadata.EntityType).Interface()
		db = db.Model(modelInstance)
	}

	for _, transformation := range transformations {
		switch transformation.Type {
		case ApplyTypeGroupBy:
			db = applyGroupBy(db, transformation.GroupBy, entityMetadata)
		case ApplyTypeAggregate:
			db = applyAggregate(db, transformation.Aggregate, entityMetadata)
		case ApplyTypeFilter:
			db = applyFilter(db, transformation.Filter, entityMetadata)
		case ApplyTypeCompute:
			// Compute transformations would require more complex handling
			// For now, we'll skip them
			continue
		}
	}
	return db
}

// applyGroupBy applies a groupby transformation to the GORM query
func applyGroupBy(db *gorm.DB, groupBy *GroupByTransformation, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if groupBy == nil || len(groupBy.Properties) == 0 {
		return db
	}

	// Build the GROUP BY clause
	groupByColumns := make([]string, 0, len(groupBy.Properties))
	selectColumns := make([]string, 0, len(groupBy.Properties))

	for _, propName := range groupBy.Properties {
		// Find the property metadata
		prop := findProperty(propName, entityMetadata)
		if prop == nil {
			continue
		}

		// Use the struct field name for GROUP BY (GORM will convert to column name)
		groupByColumns = append(groupByColumns, prop.Name)
		// Use "column as fieldname" to ensure the result has the correct key
		selectColumns = append(selectColumns, fmt.Sprintf("%s as %s", prop.Name, prop.JsonName))
	}

	// Apply nested transformations (typically aggregate)
	if len(groupBy.Transform) > 0 {
		for _, trans := range groupBy.Transform {
			if trans.Type == ApplyTypeAggregate && trans.Aggregate != nil {
				// Build aggregate expressions for SELECT
				for _, aggExpr := range trans.Aggregate.Expressions {
					aggSQL := buildAggregateSQL(aggExpr, entityMetadata)
					if aggSQL != "" {
						selectColumns = append(selectColumns, aggSQL)
					}
				}
			}
		}
	}

	// Apply GROUP BY and SELECT
	if len(groupByColumns) > 0 {
		db = db.Group(strings.Join(groupByColumns, ", "))
	}

	if len(selectColumns) > 0 {
		db = db.Select(strings.Join(selectColumns, ", "))
	}

	return db
}

// applyAggregate applies an aggregate transformation to the GORM query
func applyAggregate(db *gorm.DB, aggregate *AggregateTransformation, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if aggregate == nil || len(aggregate.Expressions) == 0 {
		return db
	}

	// Build aggregate expressions for SELECT
	selectColumns := make([]string, 0, len(aggregate.Expressions))
	for _, aggExpr := range aggregate.Expressions {
		aggSQL := buildAggregateSQL(aggExpr, entityMetadata)
		if aggSQL != "" {
			selectColumns = append(selectColumns, aggSQL)
		}
	}

	if len(selectColumns) > 0 {
		db = db.Select(strings.Join(selectColumns, ", "))
	}

	return db
}

// buildAggregateSQL builds the SQL for an aggregate expression
func buildAggregateSQL(aggExpr AggregateExpression, entityMetadata *metadata.EntityMetadata) string {
	// Handle $count special case
	if aggExpr.Property == "$count" {
		return fmt.Sprintf("COUNT(*) as %s", aggExpr.Alias)
	}

	// Find the property metadata
	prop := findProperty(aggExpr.Property, entityMetadata)
	if prop == nil {
		return ""
	}

	// Build the aggregate SQL based on method
	var sqlFunc string
	switch aggExpr.Method {
	case AggregationSum:
		sqlFunc = "SUM"
	case AggregationAvg:
		sqlFunc = "AVG"
	case AggregationMin:
		sqlFunc = "MIN"
	case AggregationMax:
		sqlFunc = "MAX"
	case AggregationCount:
		sqlFunc = "COUNT"
	case AggregationCountDistinct:
		return fmt.Sprintf("COUNT(DISTINCT %s) as %s", prop.Name, aggExpr.Alias)
	default:
		return ""
	}

	return fmt.Sprintf("%s(%s) as %s", sqlFunc, prop.Name, aggExpr.Alias)
}

// findProperty finds a property by name or JSON name in the entity metadata
func findProperty(propName string, entityMetadata *metadata.EntityMetadata) *metadata.PropertyMetadata {
	for _, prop := range entityMetadata.Properties {
		if prop.Name == propName || prop.JsonName == propName {
			return &prop
		}
	}
	return nil
}
