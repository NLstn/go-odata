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
	return options != nil && (len(options.Apply) > 0 || options.Compute != nil)
}

// applySelectToExpandedEntity applies select to an expanded navigation property (single entity or collection)
// This is a simplified version that doesn't require full metadata - it works with reflection
func applySelectToExpandedEntity(expandedValue interface{}, selectedProps []string) interface{} {
	if len(selectedProps) == 0 || expandedValue == nil {
		return expandedValue
	}

	// Build a map of selected properties for quick lookup
	selectedPropMap := make(map[string]bool)
	for _, propName := range selectedProps {
		selectedPropMap[strings.TrimSpace(propName)] = true
	}

	val := reflect.ValueOf(expandedValue)

	// Handle pointer
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return expandedValue
		}
		val = val.Elem()
	}

	// Handle slice/array (collection navigation)
	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		resultSlice := make([]map[string]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			itemVal := val.Index(i)
			resultSlice[i] = filterEntityFields(itemVal, selectedPropMap)
		}
		return resultSlice
	}

	// Handle single entity
	if val.Kind() == reflect.Struct {
		return filterEntityFields(val, selectedPropMap)
	}

	return expandedValue
}

// filterEntityFields filters struct fields based on selected properties map
func filterEntityFields(entityVal reflect.Value, selectedPropMap map[string]bool) map[string]interface{} {
	filtered := make(map[string]interface{})
	entityType := entityVal.Type()

	// Always include ID/key fields
	idFields := []string{"ID", "Id", "id"}

	for i := 0; i < entityVal.NumField(); i++ {
		field := entityType.Field(i)
		fieldVal := entityVal.Field(i)

		if !fieldVal.CanInterface() {
			continue
		}

		// Get JSON name from tag if available
		jsonName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				jsonName = parts[0]
			}
		}

		// Check if this field is selected or is an ID field
		isSelected := selectedPropMap[field.Name] || selectedPropMap[jsonName]
		isKeyField := false
		for _, keyName := range idFields {
			if field.Name == keyName {
				isKeyField = true
				break
			}
		}

		// Also check for odata:"key" tag
		if odataTag := field.Tag.Get("odata"); odataTag != "" {
			if strings.Contains(odataTag, "key") {
				isKeyField = true
			}
		}

		if isSelected || isKeyField {
			filtered[jsonName] = fieldVal.Interface()
		}
	}

	return filtered
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

	// Apply standalone compute transformation (before select)
	if options.Compute != nil {
		db = applyCompute(db, options.Compute, entityMetadata)
	}

	// Apply select at database level to fetch only needed columns
	// Skip this if compute is present, as compute handles the select clause
	if len(options.Select) > 0 && options.Compute == nil {
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
	case OpIsOf:
		// isof returns a boolean indicating if the value can be cast to the specified type
		// In SQLite, we use a CASE expression to check if CAST succeeds
		// Returns 1 (true) if cast is valid, 0 (false) otherwise
		if typeName, ok := value.(string); ok {
			// Check if this is an entity type check (no "Edm." prefix)
			// For entity type checks on the current instance ($it), always return true
			if len(typeName) > 0 && (len(typeName) < 4 || typeName[:4] != "Edm.") {
				// Entity type check - for the current instance, always return 1 (true)
				// because all records in the entity set are of that entity type
				if columnName == "$it" {
					return "1", nil
				}
				// For property-level entity type checks, we can't really validate at SQL level
				// so we return 0 (false) as entity types don't apply to properties
				return "0", nil
			}

			// EDM primitive type check
			sqlType := edmTypeToSQLType(typeName)
			// Use a safe type check that doesn't fail on invalid casts
			// We try to cast and compare with the original to see if it's valid
			return fmt.Sprintf("CASE WHEN CAST(%s AS %s) IS NOT NULL THEN 1 ELSE 0 END",
				columnName, sqlType), nil
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
func ApplySelect(results interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata, expandOptions []ExpandOption) interface{} {
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
	// Also track navigation property selects (e.g., "Product/Name")
	navPropSelects := make(map[string][]string) // nav prop name -> list of sub-properties

	for _, propName := range selectedProperties {
		propName = strings.TrimSpace(propName)
		// Check if this is a navigation property path (e.g., "Product/Name")
		if strings.Contains(propName, "/") {
			parts := strings.SplitN(propName, "/", 2)
			navProp := strings.TrimSpace(parts[0])
			subProp := strings.TrimSpace(parts[1])
			if navPropSelects[navProp] == nil {
				navPropSelects[navProp] = []string{}
			}
			navPropSelects[navProp] = append(navPropSelects[navProp], subProp)
		} else {
			selectedPropMap[propName] = true
		}
	}

	// Build a map of expanded properties for quick lookup
	expandedPropMap := make(map[string]*ExpandOption)
	for i := range expandOptions {
		expandedPropMap[expandOptions[i].NavigationProperty] = &expandOptions[i]
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
			// Include if selected OR if it's a key property OR if it's an expanded navigation property
			isSelected := selectedPropMap[prop.JsonName] || selectedPropMap[prop.Name]
			isKey := keyPropMap[prop.Name]
			isExpanded := prop.IsNavigationProp && (expandedPropMap[prop.Name] != nil || expandedPropMap[prop.JsonName] != nil)
			hasNavSelect := len(navPropSelects[prop.JsonName]) > 0 || len(navPropSelects[prop.Name]) > 0

			if isSelected || isKey || isExpanded || hasNavSelect {
				// Get the field value
				fieldValue := item.FieldByName(prop.Name)
				if fieldValue.IsValid() && fieldValue.CanInterface() {
					fieldVal := fieldValue.Interface()

					// If this is an expanded navigation property with a nested select, apply it
					if prop.IsNavigationProp && (isExpanded || hasNavSelect) {
						var expandOpt *ExpandOption
						if expandedPropMap[prop.Name] != nil {
							expandOpt = expandedPropMap[prop.Name]
						} else if expandedPropMap[prop.JsonName] != nil {
							expandOpt = expandedPropMap[prop.JsonName]
						}

						// Get nested select from expand option or from navigation path selects
						var nestedSelect []string
						if expandOpt != nil && len(expandOpt.Select) > 0 {
							nestedSelect = expandOpt.Select
						} else if len(navPropSelects[prop.JsonName]) > 0 {
							nestedSelect = navPropSelects[prop.JsonName]
						} else if len(navPropSelects[prop.Name]) > 0 {
							nestedSelect = navPropSelects[prop.Name]
						}

						// Apply nested select if specified
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

	// Build a map of selected properties for quick lookup
	selectedPropMap := make(map[string]bool)
	// Also track navigation property selects (e.g., "Product/Name")
	navPropSelects := make(map[string][]string) // nav prop name -> list of sub-properties

	for _, propName := range selectedProperties {
		propName = strings.TrimSpace(propName)
		// Check if this is a navigation property path (e.g., "Product/Name")
		if strings.Contains(propName, "/") {
			parts := strings.SplitN(propName, "/", 2)
			navProp := strings.TrimSpace(parts[0])
			subProp := strings.TrimSpace(parts[1])
			if navPropSelects[navProp] == nil {
				navPropSelects[navProp] = []string{}
			}
			navPropSelects[navProp] = append(navPropSelects[navProp], subProp)
		} else {
			selectedPropMap[propName] = true
		}
	}

	// Build a map of expanded properties for quick lookup
	expandedPropMap := make(map[string]*ExpandOption)
	for i := range expandOptions {
		expandedPropMap[expandOptions[i].NavigationProperty] = &expandOptions[i]
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
		// Include if selected OR if it's a key property OR if it's an expanded navigation property
		isSelected := selectedPropMap[prop.JsonName] || selectedPropMap[prop.Name]
		isKey := keyPropMap[prop.Name]
		isExpanded := prop.IsNavigationProp && (expandedPropMap[prop.Name] != nil || expandedPropMap[prop.JsonName] != nil)
		hasNavSelect := len(navPropSelects[prop.JsonName]) > 0 || len(navPropSelects[prop.Name]) > 0

		if isSelected || isKey || isExpanded || hasNavSelect {
			// Get the field value
			fieldValue := entityValue.FieldByName(prop.Name)
			if fieldValue.IsValid() && fieldValue.CanInterface() {
				fieldVal := fieldValue.Interface()

				// If this is an expanded navigation property with a nested select, apply it
				if prop.IsNavigationProp && (isExpanded || hasNavSelect) {
					var expandOpt *ExpandOption
					if expandedPropMap[prop.Name] != nil {
						expandOpt = expandedPropMap[prop.Name]
					} else if expandedPropMap[prop.JsonName] != nil {
						expandOpt = expandedPropMap[prop.JsonName]
					}

					// Get nested select from expand option or from navigation path selects
					var nestedSelect []string
					if expandOpt != nil && len(expandOpt.Select) > 0 {
						nestedSelect = expandOpt.Select
					} else if len(navPropSelects[prop.JsonName]) > 0 {
						nestedSelect = navPropSelects[prop.JsonName]
					} else if len(navPropSelects[prop.Name]) > 0 {
						nestedSelect = navPropSelects[prop.Name]
					}

					// Apply nested select if specified
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
			db = applyCompute(db, transformation.Compute, entityMetadata)
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

// applyCompute applies a compute transformation to the GORM query
func applyCompute(db *gorm.DB, compute *ComputeTransformation, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if compute == nil || len(compute.Expressions) == 0 {
		return db
	}

	// Set the table/model for the query - similar to applyTransformations
	if db.Statement != nil && db.Statement.Dest == nil {
		// Create an instance of the entity type to set the model
		modelInstance := reflect.New(entityMetadata.EntityType).Interface()
		db = db.Model(modelInstance)
	}

	// Build SELECT clause with computed expressions
	// We need to select all original columns plus computed columns
	selectColumns := make([]string, 0)

	// Add all original entity properties
	for _, prop := range entityMetadata.Properties {
		if !prop.IsNavigationProp {
			// Use snake_case column name for SELECT
			columnName := toSnakeCase(prop.Name)
			selectColumns = append(selectColumns, fmt.Sprintf("%s as %s", columnName, prop.JsonName))
		}
	}

	// Add computed expressions
	for _, computeExpr := range compute.Expressions {
		computeSQL := buildComputeSQL(computeExpr, entityMetadata)
		if computeSQL != "" {
			selectColumns = append(selectColumns, computeSQL)
		}
	}

	if len(selectColumns) > 0 {
		db = db.Select(strings.Join(selectColumns, ", "))
	}

	return db
}

// buildComputeSQL builds the SQL for a compute expression
func buildComputeSQL(computeExpr ComputeExpression, entityMetadata *metadata.EntityMetadata) string {
	if computeExpr.Expression == nil {
		return ""
	}

	expr := computeExpr.Expression

	// Check if this is a simple function call (no left/right expressions)
	if expr.Left == nil && expr.Right == nil && expr.Operator != "" && expr.Property != "" {
		// This is a simple function like year(CreatedAt)
		prop := findProperty(expr.Property, entityMetadata)
		if prop == nil {
			return ""
		}

		// Generate SQL for the function using snake_case column name
		columnName := toSnakeCase(prop.Name)
		funcSQL, _ := buildFunctionSQL(expr.Operator, columnName, nil)
		if funcSQL == "" {
			return ""
		}

		return fmt.Sprintf("%s as %s", funcSQL, computeExpr.Alias)
	}

	// For more complex expressions, we would need additional handling
	// For now, we'll handle simple cases
	return ""
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
