package query

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// getDatabaseDialect returns the active database dialect name (e.g. "sqlite", "postgres").
func getDatabaseDialect(db *gorm.DB) string {
	if db == nil || db.Dialector == nil {
		return "sqlite"
	}
	dialector := db.Dialector
	return dialector.Name()
}

// quoteIdent safely quotes identifiers in a portable way (double quotes work for sqlite and postgres).
// Embedded double quotes are escaped by doubling them per SQL standard.
func quoteIdent(_ string, ident string) string {
	if ident == "" {
		return ident
	}
	// Escape any embedded double quotes by doubling them
	escaped := strings.ReplaceAll(ident, "\"", "\"\"")
	return fmt.Sprintf("\"%s\"", escaped)
}

// getQuotedColumnName returns a properly quoted column name for use in SQL queries.
// For navigation property paths (e.g., "Category/Name"), it returns a fully qualified
// and quoted reference like "Categories"."name" to ensure PostgreSQL compatibility.
func getQuotedColumnName(dialect string, property string, entityMetadata *metadata.EntityMetadata) string {
	if entityMetadata == nil {
		// Quote the column name for PostgreSQL compatibility even without metadata
		return quoteIdent(dialect, toSnakeCase(property))
	}

	// Check if this is a navigation property path
	if entityMetadata.IsSingleEntityNavigationPath(property) {
		segments := strings.Split(property, "/")
		if len(segments) >= 2 {
			navPropName := strings.TrimSpace(segments[0])
			targetPropertyName := strings.TrimSpace(segments[1])

			navProp := entityMetadata.FindNavigationProperty(navPropName)
			if navProp != nil {
				// Get the related table name from cached metadata
				relatedTableName := navProp.NavigationTargetTableName
				columnName := toSnakeCase(targetPropertyName)
				// Return fully quoted reference for proper PostgreSQL case handling
				return quoteIdent(dialect, relatedTableName) + "." + quoteIdent(dialect, columnName)
			}
		}
	}

	// For regular properties, use the standard GetColumnName
	return GetColumnName(property, entityMetadata)
}

// applyFilter applies filter expressions to the GORM query
func applyFilter(db *gorm.DB, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if filter == nil {
		return db
	}

	// First, add any necessary JOINs for single-entity navigation properties
	db = addNavigationJoins(db, filter, entityMetadata)

	dialect := getDatabaseDialect(db)

	if filter.Logical != "" {
		query, args := buildFilterCondition(dialect, filter, entityMetadata)
		return db.Where(query, args...)
	}

	query, args := buildFilterCondition(dialect, filter, entityMetadata)
	return db.Where(query, args...)
}

// addNavigationJoins adds JOIN clauses for single-entity navigation properties used in filters
// Per OData v4 spec 5.1.1.15, properties of entities related with cardinality 0..1 or 1 can be accessed directly
// Note: If the same navigation property is also in $expand, GORM will handle both the JOIN (for filtering)
// and Preload (for expanding) efficiently without duplicate data loading.
func addNavigationJoins(db *gorm.DB, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if filter == nil || entityMetadata == nil {
		return db
	}

	// Track which navigation properties we've already joined to avoid duplicates
	joinedNavProps := make(map[string]bool)

	// Recursively collect all navigation property paths used in the filter
	collectAndJoinNavigationProperties(filter, entityMetadata, joinedNavProps)

	// Apply the collected joins
	for navPropName := range joinedNavProps {
		db = addNavigationJoin(db, navPropName, entityMetadata)
	}

	return db
}

// collectAndJoinNavigationProperties recursively finds navigation property paths in filter expressions
func collectAndJoinNavigationProperties(filter *FilterExpression, entityMetadata *metadata.EntityMetadata, joinedNavProps map[string]bool) {
	if filter == nil {
		return
	}

	// Check if this filter's property is a navigation property path
	if filter.Property != "" && entityMetadata.IsSingleEntityNavigationPath(filter.Property) {
		// Extract the navigation property name (first segment)
		segments := strings.Split(filter.Property, "/")
		navPropName := strings.TrimSpace(segments[0])
		joinedNavProps[navPropName] = true
	}

	// Recursively process child filters
	if filter.Left != nil {
		collectAndJoinNavigationProperties(filter.Left, entityMetadata, joinedNavProps)
	}
	if filter.Right != nil {
		collectAndJoinNavigationProperties(filter.Right, entityMetadata, joinedNavProps)
	}
}

// addNavigationJoin adds a JOIN clause for a single-entity navigation property
func addNavigationJoin(db *gorm.DB, navPropName string, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	navProp := entityMetadata.FindNavigationProperty(navPropName)
	if navProp == nil || navProp.NavigationIsArray {
		// Only join single-entity navigation properties
		return db
	}

	// Get the related entity's table name from cached metadata
	// This was computed once during entity registration and respects custom TableName() methods
	relatedTableName := navProp.NavigationTargetTableName

	// Get the parent entity's table name from cached metadata
	parentTableName := entityMetadata.TableName

	// Get the foreign key column from cached metadata
	// This was computed once during entity registration and respects GORM foreignKey: tags
	foreignKeyColumn := navProp.ForeignKeyColumnName

	// Determine the primary key column of the related table
	// Default to "id" but should check the related entity's key properties
	relatedPrimaryKey := "id"

	// Check GORM tag for explicit references
	if navProp.GormTag != "" {
		parts := strings.Split(navProp.GormTag, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "references:") {
				refField := strings.TrimPrefix(part, "references:")
				relatedPrimaryKey = toSnakeCase(refField)
				break
			}
		}
	}

	// Build the JOIN clause
	// LEFT JOIN to handle nullable navigation properties (cardinality 0..1)
	// Use GORM's quote mechanism to safely quote identifiers
	dialect := getDatabaseDialect(db)
	quotedRelatedTable := quoteIdent(dialect, relatedTableName)
	quotedParentTable := quoteIdent(dialect, parentTableName)
	quotedForeignKey := quoteIdent(dialect, foreignKeyColumn)
	quotedPrimaryKey := quoteIdent(dialect, relatedPrimaryKey)

	joinClause := fmt.Sprintf("LEFT JOIN %s ON %s.%s = %s.%s",
		quotedRelatedTable,
		quotedParentTable,
		quotedForeignKey,
		quotedRelatedTable,
		quotedPrimaryKey)

	return db.Joins(joinClause)
}

// applyHavingFilter applies a HAVING clause filter for post-aggregation filtering
// This is used when a filter transformation comes after a groupby/aggregate in $apply
func applyHavingFilter(db *gorm.DB, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if filter == nil {
		return db
	}

	dialect := getDatabaseDialect(db)

	query, args := buildFilterCondition(dialect, filter, entityMetadata)
	return db.Having(query, args...)
}

// buildFilterCondition builds a WHERE condition string and arguments for a filter expression
func buildFilterCondition(dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	if filter.Logical != "" {
		return buildLogicalCondition(dialect, filter, entityMetadata)
	}

	query, args := buildComparisonCondition(dialect, filter, entityMetadata)

	if filter.IsNot {
		return fmt.Sprintf("NOT (%s)", query), args
	}

	return query, args
}

// buildLogicalCondition builds a logical condition (AND/OR)
func buildLogicalCondition(dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	leftQuery, leftArgs := buildFilterCondition(dialect, filter.Left, entityMetadata)
	rightQuery, rightArgs := buildFilterCondition(dialect, filter.Right, entityMetadata)

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
func buildComparisonCondition(dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	if filter.Operator == OpAny || filter.Operator == OpAll {
		return buildLambdaCondition(dialect, filter, entityMetadata)
	}

	if filter.Left != nil && filter.Left.Operator != "" {
		return buildFunctionComparison(dialect, filter, entityMetadata)
	}

	var columnName string
	if propertyExists(filter.Property, entityMetadata) {
		// Use getQuotedColumnName for proper PostgreSQL handling of navigation paths
		columnName = getQuotedColumnName(dialect, filter.Property, entityMetadata)
	} else {
		rawName := sanitizeIdentifier(filter.Property)
		if rawName == "" {
			return "", nil
		}
		if rawName == "$it" {
			columnName = rawName
		} else if rawName == "$count" {
			columnName = quoteIdent(dialect, rawName)
		} else if len(rawName) > 0 && rawName[0] == '$' {
			columnName = quoteIdent(dialect, rawName)
		} else {
			// Quote computed aliases and unknown identifiers for PostgreSQL compatibility
			// This ensures aliases from $apply (like "Total" from groupby/aggregate) work correctly
			columnName = quoteIdent(dialect, rawName)
		}
	}

	if funcCall, ok := filter.Value.(*FunctionCallExpr); ok {
		rightExpr, err := convertFunctionCallExpr(funcCall, entityMetadata)
		if err == nil && rightExpr != nil {
			rightColumnName := ""
			if rightExpr.Property != "" {
				rightColumnName = getQuotedColumnName(dialect, rightExpr.Property, entityMetadata)
			}
			rightSQL, rightArgs := buildFunctionSQL(dialect, rightExpr.Operator, rightColumnName, rightExpr.Value)
			if rightSQL != "" {
				var compSQL string
				switch filter.Operator {
				case OpEqual:
					compSQL = fmt.Sprintf("%s = %s", columnName, rightSQL)
				case OpNotEqual:
					compSQL = fmt.Sprintf("%s != %s", columnName, rightSQL)
				case OpGreaterThan:
					compSQL = fmt.Sprintf("%s > %s", columnName, rightSQL)
				case OpGreaterThanOrEqual:
					compSQL = fmt.Sprintf("%s >= %s", columnName, rightSQL)
				case OpLessThan:
					compSQL = fmt.Sprintf("%s < %s", columnName, rightSQL)
				case OpLessThanOrEqual:
					compSQL = fmt.Sprintf("%s <= %s", columnName, rightSQL)
				default:
					return "", nil
				}
				return compSQL, rightArgs
			}
		}
	}

	switch filter.Operator {
	case OpEqual:
		if filter.Value == nil {
			return fmt.Sprintf("%s IS NULL", columnName), []interface{}{}
		}
		return fmt.Sprintf("%s = ?", columnName), []interface{}{filter.Value}
	case OpNotEqual:
		if filter.Value == nil {
			return fmt.Sprintf("%s IS NOT NULL", columnName), []interface{}{}
		}
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
		values, ok := filter.Value.([]interface{})
		if !ok {
			return "", nil
		}
		if len(values) == 0 {
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
		return fmt.Sprintf("(%s & ?) = ?", columnName), []interface{}{filter.Value, filter.Value}
	case OpIsOf:
		funcSQL, funcArgs := buildFunctionSQL(dialect, OpIsOf, columnName, filter.Value)
		if funcSQL == "" {
			return "", nil
		}
		return fmt.Sprintf("%s = ?", funcSQL), append(funcArgs, true)
	case OpGeoIntersects:
		funcSQL, funcArgs := buildFunctionSQL(dialect, OpGeoIntersects, columnName, filter.Value)
		if funcSQL == "" {
			return "", nil
		}
		return funcSQL, funcArgs
	case OpCast:
		return "", nil
	default:
		return "", nil
	}
}

// buildLambdaCondition builds SQL for lambda operators (any/all) using EXISTS subquery
func buildLambdaCondition(dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	navProp := findNavigationProperty(filter.Property, entityMetadata)
	if navProp == nil {
		return "", nil
	}

	// Use cached table name from metadata (computed once during entity registration)
	relatedTableName := navProp.NavigationTargetTableName

	// Use cached table name from parent entity metadata
	parentTableName := entityMetadata.TableName

	// Use cached foreign key column name from metadata
	// This was computed once during entity registration and respects GORM foreignKey: tags
	foreignKeyColumn := navProp.ForeignKeyColumnName

	parentPrimaryKey := "id"
	if len(entityMetadata.KeyProperties) > 0 {
		parentPrimaryKey = toSnakeCase(entityMetadata.KeyProperties[0].Name)
	}

	// Quote identifiers for database compatibility (especially PostgreSQL)
	quotedRelatedTable := quoteIdent(dialect, relatedTableName)
	quotedParentTable := quoteIdent(dialect, parentTableName)
	quotedForeignKey := quoteIdent(dialect, foreignKeyColumn)
	quotedParentPK := quoteIdent(dialect, parentPrimaryKey)

	if filter.Value == nil || filter.Left == nil {
		if filter.Operator == OpAny {
			return fmt.Sprintf("EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s)",
				quotedRelatedTable, quotedRelatedTable, quotedForeignKey, quotedParentTable, quotedParentPK), []interface{}{}
		}
		return fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s) OR EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s)",
			quotedRelatedTable, quotedRelatedTable, quotedForeignKey, quotedParentTable, quotedParentPK,
			quotedRelatedTable, quotedRelatedTable, quotedForeignKey, quotedParentTable, quotedParentPK), []interface{}{}
	}

	predicate := filter.Left
	if predicate == nil {
		return "", nil
	}

	predicateSQL, predicateArgs := buildFilterConditionForLambda(dialect, predicate, navProp)
	if predicateSQL == "" {
		return "", nil
	}

	var sql string
	if filter.Operator == OpAny {
		sql = fmt.Sprintf("EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s AND %s)",
			quotedRelatedTable, quotedRelatedTable, quotedForeignKey, quotedParentTable, quotedParentPK, predicateSQL)
	} else {
		sql = fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s AND NOT (%s))",
			quotedRelatedTable, quotedRelatedTable, quotedForeignKey, quotedParentTable, quotedParentPK, predicateSQL)
	}

	return sql, predicateArgs
}

// buildFilterConditionForLambda builds a filter condition for lambda subquery predicates
func buildFilterConditionForLambda(dialect string, filter *FilterExpression, navProp *metadata.PropertyMetadata) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	if filter.Logical != "" {
		return buildLogicalConditionForLambda(dialect, filter, navProp)
	}

	if filter.Left != nil && filter.Left.Operator != "" {
		return buildFunctionComparisonForLambda(dialect, filter, navProp)
	}

	// Quote the column name for proper database compatibility
	columnName := quoteIdent(dialect, toSnakeCase(filter.Property))

	switch filter.Operator {
	case OpEqual:
		if filter.Value == nil {
			return fmt.Sprintf("%s IS NULL", columnName), []interface{}{}
		}
		return fmt.Sprintf("%s = ?", columnName), []interface{}{filter.Value}
	case OpNotEqual:
		if filter.Value == nil {
			return fmt.Sprintf("%s IS NOT NULL", columnName), []interface{}{}
		}
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

// buildLogicalConditionForLambda builds a logical condition for lambda predicates
func buildLogicalConditionForLambda(dialect string, filter *FilterExpression, navProp *metadata.PropertyMetadata) (string, []interface{}) {
	leftQuery, leftArgs := buildFilterConditionForLambda(dialect, filter.Left, navProp)
	rightQuery, rightArgs := buildFilterConditionForLambda(dialect, filter.Right, navProp)

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
	return query, args
}

// buildFunctionComparisonForLambda builds a function comparison for lambda predicates
func buildFunctionComparisonForLambda(dialect string, filter *FilterExpression, _ *metadata.PropertyMetadata) (string, []interface{}) {
	funcExpr := filter.Left
	// Quote the column name for proper database compatibility
	columnName := quoteIdent(dialect, toSnakeCase(funcExpr.Property))

	funcSQL, funcArgs := buildFunctionSQL(dialect, funcExpr.Operator, columnName, funcExpr.Value)
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

// buildFunctionComparison builds a comparison with a function on the left side
func buildFunctionComparison(dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	funcExpr := filter.Left
	columnName := getQuotedColumnName(dialect, funcExpr.Property, entityMetadata)

	funcSQL, funcArgs := buildFunctionSQL(dialect, funcExpr.Operator, columnName, funcExpr.Value)
	if funcSQL == "" {
		return "", nil
	}

	if rightFunc, ok := filter.Value.(*FunctionCallExpr); ok {
		rightFuncExpr, err := convertFunctionCallExpr(rightFunc, entityMetadata)
		if err != nil {
			compSQL := buildComparisonSQL(filter.Operator, funcSQL)
			if compSQL == "" {
				return "", nil
			}
			allArgs := append(funcArgs, filter.Value)
			return compSQL, allArgs
		}

		rightColumnName := getQuotedColumnName(dialect, rightFuncExpr.Property, entityMetadata)
		rightFuncSQL, rightFuncArgs := buildFunctionSQL(dialect, rightFuncExpr.Operator, rightColumnName, rightFuncExpr.Value)
		if rightFuncSQL == "" {
			return "", nil
		}

		var compSQL string
		switch filter.Operator {
		case OpEqual:
			compSQL = fmt.Sprintf("%s = %s", funcSQL, rightFuncSQL)
		case OpNotEqual:
			compSQL = fmt.Sprintf("%s != %s", funcSQL, rightFuncSQL)
		case OpGreaterThan:
			compSQL = fmt.Sprintf("%s > %s", funcSQL, rightFuncSQL)
		case OpGreaterThanOrEqual:
			compSQL = fmt.Sprintf("%s >= %s", funcSQL, rightFuncSQL)
		case OpLessThan:
			compSQL = fmt.Sprintf("%s < %s", funcSQL, rightFuncSQL)
		case OpLessThanOrEqual:
			compSQL = fmt.Sprintf("%s <= %s", funcSQL, rightFuncSQL)
		default:
			return "", nil
		}

		allArgs := append(funcArgs, rightFuncArgs...)
		return compSQL, allArgs
	}

	// Check if the right side is a property name (for property-to-property comparison like "Price add 0.01 gt Price")
	if rightPropertyName, ok := filter.Value.(string); ok && propertyExists(rightPropertyName, entityMetadata) {
		// This is a property-to-property comparison
		rightColumnName := getQuotedColumnName(dialect, rightPropertyName, entityMetadata)
		var compSQL string
		switch filter.Operator {
		case OpEqual:
			compSQL = fmt.Sprintf("%s = %s", funcSQL, rightColumnName)
		case OpNotEqual:
			compSQL = fmt.Sprintf("%s != %s", funcSQL, rightColumnName)
		case OpGreaterThan:
			compSQL = fmt.Sprintf("%s > %s", funcSQL, rightColumnName)
		case OpGreaterThanOrEqual:
			compSQL = fmt.Sprintf("%s >= %s", funcSQL, rightColumnName)
		case OpLessThan:
			compSQL = fmt.Sprintf("%s < %s", funcSQL, rightColumnName)
		case OpLessThanOrEqual:
			compSQL = fmt.Sprintf("%s <= %s", funcSQL, rightColumnName)
		default:
			return "", nil
		}
		return compSQL, funcArgs
	}

	compSQL := buildComparisonSQL(filter.Operator, funcSQL)
	if compSQL == "" {
		return "", nil
	}

	allArgs := append(funcArgs, filter.Value)
	return compSQL, allArgs
}

// buildFunctionSQL generates SQL for a function expression
func buildFunctionSQL(dialect string, op FilterOperator, columnName string, value interface{}) (string, []interface{}) {
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
		if dialect == "postgres" {
			return fmt.Sprintf("(POSITION(? IN %s) - 1)", columnName), []interface{}{value}
		}
		return fmt.Sprintf("INSTR(%s, ?) - 1", columnName), []interface{}{value}
	case OpConcat:
		// PostgreSQL uses || operator, SQLite supports both CONCAT() and ||
		// For maximum compatibility, use || for PostgreSQL and CONCAT() for SQLite
		if args, ok := value.([]interface{}); ok && len(args) == 2 {
			firstArg := args[0]
			secondArg := args[1]

			var leftSQL string
			var leftArgs []interface{}

			if funcCall, ok := firstArg.(*FunctionCallExpr); ok {
				leftExpr, err := convertFunctionCallExpr(funcCall, nil)
				if err == nil {
					leftSQL, leftArgs = buildFunctionSQL(dialect, leftExpr.Operator, leftExpr.Property, leftExpr.Value)
				}
				if leftSQL == "" {
					leftSQL = "?"
					leftArgs = []interface{}{firstArg}
				}
			} else {
				leftSQL = "?"
				leftArgs = []interface{}{firstArg}
			}

			var rightSQL string
			var rightArgs []interface{}

			if funcCall, ok := secondArg.(*FunctionCallExpr); ok {
				rightExpr, err := convertFunctionCallExpr(funcCall, nil)
				if err == nil {
					rightSQL, rightArgs = buildFunctionSQL(dialect, rightExpr.Operator, rightExpr.Property, rightExpr.Value)
				}
				if rightSQL == "" {
					rightSQL = "?"
					rightArgs = []interface{}{secondArg}
				}
			} else if strVal, ok := secondArg.(string); ok {
				if len(strVal) > 0 && (strVal[0] >= 'A' && strVal[0] <= 'Z' || strings.Contains(strVal, "_")) {
					// Quote the column name for PostgreSQL compatibility
					rightSQL = quoteIdent(dialect, toSnakeCase(strVal))
					rightArgs = nil
				} else {
					rightSQL = "?"
					rightArgs = []interface{}{strVal}
				}
			} else {
				rightSQL = "?"
				rightArgs = []interface{}{secondArg}
			}

			allArgs := append(leftArgs, rightArgs...)
			if dialect == "postgres" {
				return fmt.Sprintf("(%s || %s)", leftSQL, rightSQL), allArgs
			}
			return fmt.Sprintf("CONCAT(%s, %s)", leftSQL, rightSQL), allArgs
		}

		if funcCall, ok := value.(*FunctionCallExpr); ok {
			rightExpr, err := convertFunctionCallExpr(funcCall, nil)
			if err != nil {
				if dialect == "postgres" {
					return fmt.Sprintf("(%s || ?)", columnName), []interface{}{value}
				}
				return fmt.Sprintf("CONCAT(%s, ?)", columnName), []interface{}{value}
			}
			rightColumnName := rightExpr.Property
			rightSQL, rightArgs := buildFunctionSQL(dialect, rightExpr.Operator, rightColumnName, rightExpr.Value)
			if rightSQL == "" {
				if dialect == "postgres" {
					return fmt.Sprintf("(%s || ?)", columnName), []interface{}{value}
				}
				return fmt.Sprintf("CONCAT(%s, ?)", columnName), []interface{}{value}
			}
			if dialect == "postgres" {
				return fmt.Sprintf("(%s || %s)", columnName, rightSQL), rightArgs
			}
			return fmt.Sprintf("CONCAT(%s, %s)", columnName, rightSQL), rightArgs
		}
		if strVal, ok := value.(string); ok {
			if len(strVal) > 0 && (strVal[0] >= 'A' && strVal[0] <= 'Z' || strings.Contains(strVal, "_")) {
				// Quote the column name for PostgreSQL compatibility
				columnName2 := quoteIdent(dialect, toSnakeCase(strVal))
				if dialect == "postgres" {
					return fmt.Sprintf("(%s || %s)", columnName, columnName2), nil
				}
				return fmt.Sprintf("CONCAT(%s, %s)", columnName, columnName2), nil
			}
		}
		if dialect == "postgres" {
			return fmt.Sprintf("(%s || ?)", columnName), []interface{}{value}
		}
		return fmt.Sprintf("CONCAT(%s, ?)", columnName), []interface{}{value}
	case OpHas:
		return fmt.Sprintf("(%s & ?)", columnName), []interface{}{value}
	case OpSubstring:
		return buildSubstringSQL(dialect, columnName, value)
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
	case OpYear:
		if dialect == "postgres" {
			return fmt.Sprintf("(EXTRACT(YEAR FROM %s)::INT)", columnName), nil
		}
		return fmt.Sprintf("CAST(strftime('%%Y', %s) AS INTEGER)", columnName), nil
	case OpMonth:
		if dialect == "postgres" {
			return fmt.Sprintf("(EXTRACT(MONTH FROM %s)::INT)", columnName), nil
		}
		return fmt.Sprintf("CAST(strftime('%%m', %s) AS INTEGER)", columnName), nil
	case OpDay:
		if dialect == "postgres" {
			return fmt.Sprintf("(EXTRACT(DAY FROM %s)::INT)", columnName), nil
		}
		return fmt.Sprintf("CAST(strftime('%%d', %s) AS INTEGER)", columnName), nil
	case OpHour:
		if dialect == "postgres" {
			return fmt.Sprintf("(EXTRACT(HOUR FROM %s)::INT)", columnName), nil
		}
		return fmt.Sprintf("CAST(strftime('%%H', %s) AS INTEGER)", columnName), nil
	case OpMinute:
		if dialect == "postgres" {
			return fmt.Sprintf("(EXTRACT(MINUTE FROM %s)::INT)", columnName), nil
		}
		return fmt.Sprintf("CAST(strftime('%%M', %s) AS INTEGER)", columnName), nil
	case OpSecond:
		if dialect == "postgres" {
			return fmt.Sprintf("(EXTRACT(SECOND FROM %s)::INT)", columnName), nil
		}
		return fmt.Sprintf("CAST(strftime('%%S', %s) AS INTEGER)", columnName), nil
	case OpDate:
		if dialect == "postgres" {
			return fmt.Sprintf("(CAST(%s AS DATE))", columnName), nil
		}
		return fmt.Sprintf("DATE(%s)", columnName), nil
	case OpTime:
		if dialect == "postgres" {
			return fmt.Sprintf("(CAST(%s AS TIME))", columnName), nil
		}
		return fmt.Sprintf("TIME(%s)", columnName), nil
	case OpNow:
		if dialect == "postgres" {
			return "NOW()", nil
		}
		return "datetime('now')", nil
	case OpCeiling:
		return fmt.Sprintf("CASE WHEN %s = CAST(%s AS INTEGER) THEN %s ELSE CAST(%s AS INTEGER) + (CASE WHEN %s > 0 THEN 1 ELSE 0 END) END",
			columnName, columnName, columnName, columnName, columnName), nil
	case OpFloor:
		return fmt.Sprintf("CASE WHEN %s = CAST(%s AS INTEGER) THEN %s ELSE CAST(%s AS INTEGER) - (CASE WHEN %s < 0 THEN 1 ELSE 0 END) END",
			columnName, columnName, columnName, columnName, columnName), nil
	case OpRound:
		return fmt.Sprintf("ROUND(%s)", columnName), nil
	case OpCast:
		if typeName, ok := value.(string); ok {
			sqlType := edmTypeToSQLType(dialect, typeName)
			return fmt.Sprintf("CAST(%s AS %s)", columnName, sqlType), nil
		}
		return "", nil
	case OpIsOf:
		if typeName, ok := value.(string); ok {
			if len(typeName) > 0 && (len(typeName) < 4 || typeName[:4] != "Edm.") {
				if columnName == "$it" {
					return "1", nil
				}
				return "0", nil
			}

			sqlType := edmTypeToSQLType(dialect, typeName)
			return fmt.Sprintf("CASE WHEN CAST(%s AS %s) IS NOT NULL THEN 1 ELSE 0 END",
				columnName, sqlType), nil
		}
		return "", nil
	case OpGeoDistance:
		if geoStr, ok := value.(string); ok {
			return fmt.Sprintf("ST_Distance(%s, ST_GeomFromText(?, 4326))", columnName), []interface{}{geoStr}
		}
		return "", nil
	case OpGeoLength:
		return fmt.Sprintf("ST_Length(%s)", columnName), nil
	case OpGeoIntersects:
		if geoStr, ok := value.(string); ok {
			return fmt.Sprintf("ST_Intersects(%s, ST_GeomFromText(?, 4326))", columnName), []interface{}{geoStr}
		}
		return "", nil
	default:
		return "", nil
	}
}

// edmTypeToSQLType converts OData EDM types to SQLite types
func edmTypeToSQLType(dialect string, edmType string) string {
	switch dialect {
	case "postgres":
		switch edmType {
		case "Edm.String":
			return "TEXT"
		case "Edm.Int32", "Edm.Int16", "Edm.Byte", "Edm.SByte":
			return "INTEGER"
		case "Edm.Int64":
			return "BIGINT"
		case "Edm.Decimal":
			return "NUMERIC"
		case "Edm.Double", "Edm.Single":
			return "DOUBLE PRECISION"
		case "Edm.Boolean":
			return "BOOLEAN"
		case "Edm.DateTimeOffset":
			return "TIMESTAMP WITH TIME ZONE"
		case "Edm.Date":
			return "DATE"
		case "Edm.TimeOfDay":
			return "TIME"
		case "Edm.Guid":
			return "UUID"
		case "Edm.Binary":
			return "BYTEA"
		default:
			return "TEXT"
		}
	default: // sqlite and fallback
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
			return "INTEGER"
		case "Edm.DateTimeOffset", "Edm.Date", "Edm.TimeOfDay":
			return "TEXT"
		case "Edm.Guid":
			return "TEXT"
		case "Edm.Binary":
			return "BLOB"
		default:
			return "TEXT"
		}
	}
}

// buildSubstringSQL builds SQL for substring function
func buildSubstringSQL(dialect string, columnName string, value interface{}) (string, []interface{}) {
	args, ok := value.([]interface{})
	if !ok {
		return "", nil
	}
	if len(args) == 1 {
		if dialect == "postgres" {
			return fmt.Sprintf("SUBSTRING(%s FROM ? + 1 FOR LENGTH(%s))", columnName, columnName), []interface{}{args[0]}
		}
		return fmt.Sprintf("SUBSTR(%s, ? + 1, LENGTH(%s))", columnName, columnName), []interface{}{args[0]}
	}
	if len(args) == 2 {
		if dialect == "postgres" {
			return fmt.Sprintf("SUBSTRING(%s FROM ? + 1 FOR ?)", columnName), args
		}
		return fmt.Sprintf("SUBSTR(%s, ? + 1, ?)", columnName), args
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
