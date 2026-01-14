package query

import (
	"fmt"
	"reflect"
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

// quoteIdent safely quotes identifiers based on the database dialect.
// Each database uses different quoting conventions for identifiers.
// Embedded quotes are escaped by doubling them per each dialect's standard.
func quoteIdent(dialect string, ident string) string {
	if ident == "" {
		return ident
	}

	switch dialect {
	case "mysql":
		// MySQL/MariaDB uses backticks
		escaped := strings.ReplaceAll(ident, "`", "``")
		return fmt.Sprintf("`%s`", escaped)

	case "postgres", "postgresql":
		// PostgreSQL uses double quotes
		escaped := strings.ReplaceAll(ident, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)

	case "sqlite", "sqlite3":
		// SQLite supports double quotes (SQL standard)
		escaped := strings.ReplaceAll(ident, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)

	case "sqlserver", "mssql":
		// SQL Server uses square brackets (preferred) or double quotes
		escaped := strings.ReplaceAll(ident, "]", "]]")
		return fmt.Sprintf("[%s]", escaped)

	default:
		// Default to double quotes (SQL standard)
		escaped := strings.ReplaceAll(ident, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", escaped)
	}
}

// getQuotedColumnName returns a properly quoted column name for use in SQL queries.
// For navigation property paths (e.g., "Category/Name"), it returns a fully qualified
// and quoted reference like "<TargetTableFromMetadata>"."column_name" to ensure PostgreSQL compatibility.
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

	// For regular properties, use the standard GetColumnName.
	// Note: Regular properties are not quoted here as GORM handles quoting internally
	// when executing queries. Only navigation property paths (above) are explicitly quoted
	// to ensure correct table aliasing for JOIN operations in PostgreSQL.
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
		query, args := buildFilterConditionWithDB(db, dialect, filter, entityMetadata)
		return db.Where(query, args...)
	}

	query, args := buildFilterConditionWithDB(db, dialect, filter, entityMetadata)
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
	// First check for explicit references: tag in GORM as an override
	relatedPrimaryKey := ""
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

	// If no explicit references: tag, use the target entity's actual primary key from metadata
	if relatedPrimaryKey == "" {
		targetMetadata, err := entityMetadata.ResolveNavigationTarget(navPropName)
		if err == nil && targetMetadata != nil && len(targetMetadata.KeyProperties) > 0 {
			// Use the first key property's column name
			// For single keys: This correctly resolves the actual primary key (e.g., "code", "language_key")
			// For composite keys: This uses the first key component, which works when the foreign key
			//   references the first component of the composite key. If a foreign key should reference
			//   a different component, the references: tag MUST be used explicitly.
			relatedPrimaryKey = targetMetadata.KeyProperties[0].ColumnName
		} else {
			// Fallback to "id" only if we can't resolve the target metadata
			relatedPrimaryKey = "id"
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

	query, args := buildFilterConditionWithDB(db, dialect, filter, entityMetadata)
	return db.Having(query, args...)
}

// buildFilterCondition builds a WHERE condition string and arguments for a filter expression.
// This is a convenience wrapper around buildFilterConditionWithDB for callers that don't need
// db context (e.g., for alias resolution). In production, the dialect parameter varies based
// on the database (sqlite, postgres, mysql, etc.), but tests consistently use "sqlite".
//
//nolint:unparam // dialect varies in production based on database, tests use sqlite
func buildFilterCondition(dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	return buildFilterConditionWithDB(nil, dialect, filter, entityMetadata)
}

func buildFilterConditionWithDB(db *gorm.DB, dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	if filter.Logical != "" {
		return buildLogicalConditionWithDB(db, dialect, filter, entityMetadata)
	}

	query, args := buildComparisonConditionWithDB(db, dialect, filter, entityMetadata)

	if filter.IsNot {
		return fmt.Sprintf("NOT (%s)", query), args
	}

	return query, args
}

func buildLogicalConditionWithDB(db *gorm.DB, dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	leftQuery, leftArgs := buildFilterConditionWithDB(db, dialect, filter.Left, entityMetadata)
	rightQuery, rightArgs := buildFilterConditionWithDB(db, dialect, filter.Right, entityMetadata)

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

// resolveColumnName determines the correct column name to use in SQL queries.
// It handles special cases like $it, $count, aggregate aliases, and regular properties.
func resolveColumnName(db *gorm.DB, dialect string, propertyName string, entityMetadata *metadata.EntityMetadata) string {
	// Check if this is a known entity property first
	if propertyExists(propertyName, entityMetadata) {
		return getQuotedColumnName(dialect, propertyName, entityMetadata)
	}

	// Handle special identifiers and aliases
	rawName := sanitizeIdentifier(propertyName)
	if rawName == "" {
		return ""
	}

	// Special case: $it refers to the entity itself
	if rawName == "$it" {
		return rawName
	}

	// Special case: $count requires COUNT(*) expression for certain databases
	if rawName == "$count" {
		// PostgreSQL/MySQL/MariaDB don't support referencing SELECT aliases in HAVING/WHERE
		if dialect == "postgres" || dialect == "mysql" || dialect == "mariadb" {
			if db != nil {
				if expr, ok := getAliasExprFromDB(db, "$count"); ok {
					return expr
				}
			}
			return "COUNT(*)"
		}
		return quoteIdent(dialect, rawName)
	}

	// Handle other special $ identifiers
	if len(rawName) > 0 && rawName[0] == '$' {
		return quoteIdent(dialect, rawName)
	}

	// Try to resolve aggregate/compute aliases for databases that don't support them
	if dialect == "postgres" || dialect == "mysql" || dialect == "mariadb" {
		if db != nil {
			if expr, ok := getAliasExprFromDB(db, rawName); ok {
				return expr
			}
		}
	}

	// Default: quote as identifier
	return quoteIdent(dialect, rawName)
}

// tryBuildRightSideFunctionComparison attempts to build a comparison when the right side is a function call.
// Returns the SQL string, arguments, and a boolean indicating if the comparison was successfully built.
func tryBuildRightSideFunctionComparison(dialect string, leftColumn string, operator FilterOperator, rightValue interface{}, entityMetadata *metadata.EntityMetadata) (string, []interface{}, bool) {
	funcCall, ok := rightValue.(*FunctionCallExpr)
	if !ok {
		return "", nil, false
	}

	rightExpr, err := convertFunctionCallExpr(funcCall, entityMetadata)
	if err != nil || rightExpr == nil {
		return "", nil, false
	}

	// Determine the column name for the right side expression
	rightColumnName := ""
	if rightExpr.Property != "" {
		rightColumnName = getQuotedColumnName(dialect, rightExpr.Property, entityMetadata)
	}

	rightSQL, rightArgs := buildFunctionSQL(dialect, rightExpr.Operator, rightColumnName, rightExpr.Value)
	if rightSQL == "" {
		return "", nil, false
	}

	// Build the comparison SQL based on the operator
	var compSQL string
	switch operator {
	case OpEqual:
		compSQL = fmt.Sprintf("%s = %s", leftColumn, rightSQL)
	case OpNotEqual:
		compSQL = fmt.Sprintf("%s != %s", leftColumn, rightSQL)
	case OpGreaterThan:
		compSQL = fmt.Sprintf("%s > %s", leftColumn, rightSQL)
	case OpGreaterThanOrEqual:
		compSQL = fmt.Sprintf("%s >= %s", leftColumn, rightSQL)
	case OpLessThan:
		compSQL = fmt.Sprintf("%s < %s", leftColumn, rightSQL)
	case OpLessThanOrEqual:
		compSQL = fmt.Sprintf("%s <= %s", leftColumn, rightSQL)
	default:
		return "", nil, false
	}

	return compSQL, rightArgs, true
}

// buildStandardComparison builds the SQL for a standard comparison operation.
// This handles all comparison operators like =, !=, >, <, IN, LIKE, etc.
func buildStandardComparison(dialect string, operator FilterOperator, columnName string, value interface{}, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	// Check if this is a property-to-property comparison
	// (e.g., "Price gt Cost" should generate "price > cost", not "price > 'Cost'")
	if valueStr, ok := value.(string); ok && propertyExists(valueStr, entityMetadata) {
		rightColumnName := getQuotedColumnName(dialect, valueStr, entityMetadata)
		
		switch operator {
		case OpEqual:
			return fmt.Sprintf("%s = %s", columnName, rightColumnName), []interface{}{}
		case OpNotEqual:
			return fmt.Sprintf("%s != %s", columnName, rightColumnName), []interface{}{}
		case OpGreaterThan:
			return fmt.Sprintf("%s > %s", columnName, rightColumnName), []interface{}{}
		case OpGreaterThanOrEqual:
			return fmt.Sprintf("%s >= %s", columnName, rightColumnName), []interface{}{}
		case OpLessThan:
			return fmt.Sprintf("%s < %s", columnName, rightColumnName), []interface{}{}
		case OpLessThanOrEqual:
			return fmt.Sprintf("%s <= %s", columnName, rightColumnName), []interface{}{}
		}
	}
	
	switch operator {
	case OpEqual:
		if value == nil {
			return fmt.Sprintf("%s IS NULL", columnName), []interface{}{}
		}
		return fmt.Sprintf("%s = ?", columnName), []interface{}{value}

	case OpNotEqual:
		if value == nil {
			return fmt.Sprintf("%s IS NOT NULL", columnName), []interface{}{}
		}
		return fmt.Sprintf("%s != ?", columnName), []interface{}{value}

	case OpGreaterThan:
		return fmt.Sprintf("%s > ?", columnName), []interface{}{value}

	case OpGreaterThanOrEqual:
		return fmt.Sprintf("%s >= ?", columnName), []interface{}{value}

	case OpLessThan:
		return fmt.Sprintf("%s < ?", columnName), []interface{}{value}

	case OpLessThanOrEqual:
		return fmt.Sprintf("%s <= ?", columnName), []interface{}{value}

	case OpIn:
		values, ok := value.([]interface{})
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
		return buildLikeComparison(dialect, columnName, value, true, true)

	case OpStartsWith:
		return buildLikeComparison(dialect, columnName, value, false, true)

	case OpEndsWith:
		return buildLikeComparison(dialect, columnName, value, true, false)

	case OpHas:
		return fmt.Sprintf("(%s & ?) = ?", columnName), []interface{}{value, value}

	case OpIsOf:
		typeName, ok := value.(string)
		if !ok {
			// isof() requires a string type name argument
			// Return always-false condition for invalid type
			return "1 = 0", nil
		}

		// Check if this is an entity type check (columnName == "$it").
		// For entity type checks, we need to use the discriminator column via buildEntityTypeFilter,
		// rather than the generic function-based approach used for property type checks.
		if columnName == "$it" {
			return buildEntityTypeFilter(dialect, typeName, entityMetadata)
		}

		// For property type checks, use the standard buildFunctionSQL approach
		funcSQL, funcArgs := buildFunctionSQL(dialect, OpIsOf, columnName, value)
		if funcSQL == "" {
			return "", nil
		}
		// Use integer 1 instead of boolean true for database compatibility (PostgreSQL)
		return fmt.Sprintf("%s = ?", funcSQL), append(funcArgs, 1)

	case OpGeoIntersects:
		funcSQL, funcArgs := buildFunctionSQL(dialect, OpGeoIntersects, columnName, value)
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

// buildComparisonCondition builds a comparison condition
func buildComparisonCondition(dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	return buildComparisonConditionWithDB(nil, dialect, filter, entityMetadata)
}

func buildComparisonConditionWithDB(db *gorm.DB, dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	// Handle lambda operators (any, all)
	if filter.Operator == OpAny || filter.Operator == OpAll {
		return buildLambdaCondition(dialect, filter, entityMetadata)
	}

	// Handle function comparisons (e.g., tolower(Name) eq 'john')
	if filter.Left != nil && filter.Left.Operator != "" {
		return buildFunctionComparison(dialect, filter, entityMetadata)
	}

	// Resolve the column name for the left side of the comparison
	columnName := resolveColumnName(db, dialect, filter.Property, entityMetadata)
	if columnName == "" {
		return "", nil
	}

	// Try to build a comparison with a function on the right side (e.g., Name eq tolower('JOHN'))
	if sql, args, ok := tryBuildRightSideFunctionComparison(dialect, columnName, filter.Operator, filter.Value, entityMetadata); ok {
		return sql, args
	}

	// Build a standard comparison
	return buildStandardComparison(dialect, filter.Operator, columnName, filter.Value, entityMetadata)
}

// buildEntityTypeFilter builds SQL for filtering by entity type using the discriminator column
// For isof('Namespace.EntityType'), this creates a filter on the discriminator column
func buildEntityTypeFilter(dialect string, typeName string, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	if entityMetadata == nil || entityMetadata.TypeDiscriminator == nil {
		// If no discriminator is configured, we can't filter by entity type correctly
		// Return "1 = 0" to match no entities (since we can't verify the type)
		return "1 = 0", nil
	}

	// Extract the simple type name from the qualified name (e.g., "Namespace.SpecialProduct" -> "SpecialProduct")
	simpleTypeName := typeName
	if idx := strings.LastIndex(typeName, "."); idx != -1 {
		simpleTypeName = typeName[idx+1:]
	}

	// Quote the discriminator column name
	quotedColumn := quoteIdent(dialect, entityMetadata.TypeDiscriminator.ColumnName)

	// Return the filter condition
	return fmt.Sprintf("%s = ?", quotedColumn), []interface{}{simpleTypeName}
}

// buildLambdaCondition builds SQL for lambda operators (any/all) using EXISTS subquery
func buildLambdaCondition(dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	navProp := findNavigationProperty(filter.Property, entityMetadata)
	if navProp == nil {
		return "", nil
	}

	navTargetMetadata := getNavigationTargetMetadata(navProp)

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

	predicateSQL, predicateArgs := buildFilterConditionForLambda(dialect, predicate, navTargetMetadata)
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

func getNavigationTargetMetadata(navProp *metadata.PropertyMetadata) *metadata.EntityMetadata {
	if navProp == nil || navProp.Type == nil {
		return nil
	}

	targetType := navProp.Type
	if targetType.Kind() == reflect.Slice {
		targetType = targetType.Elem()
	}
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}
	if targetType.Kind() != reflect.Struct {
		return nil
	}

	targetMeta, err := metadata.AnalyzeEntity(reflect.New(targetType).Interface())
	if err != nil {
		return nil
	}

	return targetMeta
}

// buildFilterConditionForLambda builds a filter condition for lambda subquery predicates
func buildFilterConditionForLambda(dialect string, filter *FilterExpression, navTargetMetadata *metadata.EntityMetadata) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	if filter.Logical != "" {
		return buildLogicalConditionForLambda(dialect, filter, navTargetMetadata)
	}

	if filter.Left != nil && filter.Left.Operator != "" {
		return buildFunctionComparisonForLambda(dialect, filter, navTargetMetadata)
	}

	// Quote the column name for proper database compatibility
	columnName := quoteIdent(dialect, GetColumnName(filter.Property, navTargetMetadata))

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
		return buildLikeComparison(dialect, columnName, filter.Value, true, true)
	case OpStartsWith:
		return buildLikeComparison(dialect, columnName, filter.Value, false, true)
	case OpEndsWith:
		return buildLikeComparison(dialect, columnName, filter.Value, true, false)
	default:
		return "", nil
	}
}

// buildLogicalConditionForLambda builds a logical condition for lambda predicates
func buildLogicalConditionForLambda(dialect string, filter *FilterExpression, navTargetMetadata *metadata.EntityMetadata) (string, []interface{}) {
	leftQuery, leftArgs := buildFilterConditionForLambda(dialect, filter.Left, navTargetMetadata)
	rightQuery, rightArgs := buildFilterConditionForLambda(dialect, filter.Right, navTargetMetadata)

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
func buildFunctionComparisonForLambda(dialect string, filter *FilterExpression, navTargetMetadata *metadata.EntityMetadata) (string, []interface{}) {
	funcExpr := filter.Left
	// Quote the column name for proper database compatibility
	columnName := quoteIdent(dialect, GetColumnName(funcExpr.Property, navTargetMetadata))

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

		compSQL := buildTwoOperandComparisonSQL(filter.Operator, funcSQL, rightFuncSQL)
		if compSQL == "" {
			return "", nil
		}

		allArgs := append(funcArgs, rightFuncArgs...)
		return compSQL, allArgs
	}

	// Check if the right side is a property name (for property-to-property comparison like "Price add 0.01 gt Price")
	if rightPropertyName, ok := filter.Value.(string); ok && propertyExists(rightPropertyName, entityMetadata) {
		// This is a property-to-property comparison
		rightColumnName := getQuotedColumnName(dialect, rightPropertyName, entityMetadata)
		compSQL := buildTwoOperandComparisonSQL(filter.Operator, funcSQL, rightColumnName)
		if compSQL == "" {
			return "", nil
		}
		return compSQL, funcArgs
	}

	compSQL := buildComparisonSQL(filter.Operator, funcSQL)
	if compSQL == "" {
		return "", nil
	}

	// Convert boolean values to integers for isof() function comparisons
	// This is needed because isof() returns 1/0 and PostgreSQL can't compare int with boolean
	rightValue := filter.Value
	if funcExpr.Operator == OpIsOf {
		if boolVal, ok := rightValue.(bool); ok {
			if boolVal {
				rightValue = 1
			} else {
				rightValue = 0
			}
		}
	}

	allArgs := append(funcArgs, rightValue)
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
				// Note: This heuristic (starts with uppercase or contains underscore) may incorrectly
				// treat some literal strings as column references. A proper fix would require passing
				// entityMetadata to buildFunctionSQL and using propertyExists(), which is a larger refactor.
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
			// Note: This heuristic (starts with uppercase or contains underscore) may incorrectly
			// treat some literal strings as column references. A proper fix would require passing
			// entityMetadata to buildFunctionSQL and using propertyExists(), which is a larger refactor.
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
		switch dialect {
		case "postgres":
			return fmt.Sprintf("(EXTRACT(YEAR FROM %s)::INT)", columnName), nil
		case "mysql":
			return fmt.Sprintf("YEAR(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%Y', %s) AS INTEGER)", columnName), nil
		}
	case OpMonth:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("(EXTRACT(MONTH FROM %s)::INT)", columnName), nil
		case "mysql":
			return fmt.Sprintf("MONTH(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%m', %s) AS INTEGER)", columnName), nil
		}
	case OpDay:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("(EXTRACT(DAY FROM %s)::INT)", columnName), nil
		case "mysql":
			return fmt.Sprintf("DAY(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%d', %s) AS INTEGER)", columnName), nil
		}
	case OpHour:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("(EXTRACT(HOUR FROM %s)::INT)", columnName), nil
		case "mysql":
			return fmt.Sprintf("HOUR(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%H', %s) AS INTEGER)", columnName), nil
		}
	case OpMinute:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("(EXTRACT(MINUTE FROM %s)::INT)", columnName), nil
		case "mysql":
			return fmt.Sprintf("MINUTE(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%M', %s) AS INTEGER)", columnName), nil
		}
	case OpSecond:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("(EXTRACT(SECOND FROM %s)::INT)", columnName), nil
		case "mysql":
			return fmt.Sprintf("SECOND(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%S', %s) AS INTEGER)", columnName), nil
		}
	case OpDate:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("(CAST(%s AS DATE))", columnName), nil
		case "mysql":
			return fmt.Sprintf("DATE(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("DATE(%s)", columnName), nil
		}
	case OpTime:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("(CAST(%s AS TIME))", columnName), nil
		case "mysql":
			return fmt.Sprintf("TIME(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("TIME(%s)", columnName), nil
		}
	case OpNow:
		switch dialect {
		case "postgres":
			return "NOW()", nil
		case "mysql":
			return "NOW()", nil
		default: // sqlite
			return "datetime('now')", nil
		}
	case OpCeiling:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("CEILING(%s)", columnName), nil
		case "mysql":
			return fmt.Sprintf("CEILING(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CASE WHEN %s = CAST(%s AS INTEGER) THEN %s ELSE CAST(%s AS INTEGER) + (CASE WHEN %s > 0 THEN 1 ELSE 0 END) END",
				columnName, columnName, columnName, columnName, columnName), nil
		}
	case OpFloor:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("FLOOR(%s)", columnName), nil
		case "mysql":
			return fmt.Sprintf("FLOOR(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CASE WHEN %s = CAST(%s AS INTEGER) THEN %s ELSE CAST(%s AS INTEGER) - (CASE WHEN %s < 0 THEN 1 ELSE 0 END) END",
				columnName, columnName, columnName, columnName, columnName), nil
		}
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

// edmTypeToSQLType converts OData EDM types to database-specific SQL types
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
	case "mysql":
		switch edmType {
		case "Edm.String":
			return "CHAR"
		case "Edm.Int32", "Edm.Int16", "Edm.Byte", "Edm.SByte":
			return "SIGNED"
		case "Edm.Int64":
			return "SIGNED"
		case "Edm.Decimal":
			return "DECIMAL"
		case "Edm.Double", "Edm.Single":
			return "DECIMAL"
		case "Edm.Boolean":
			return "SIGNED"
		case "Edm.DateTimeOffset":
			return "DATETIME"
		case "Edm.Date":
			return "DATE"
		case "Edm.TimeOfDay":
			return "TIME"
		case "Edm.Guid":
			return "CHAR"
		case "Edm.Binary":
			return "BINARY"
		default:
			return "CHAR"
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

// buildTwoOperandComparisonSQL constructs a SQL comparison expression between two operands
// based on the given filter operator. Returns an empty string for unsupported operators.
func buildTwoOperandComparisonSQL(op FilterOperator, leftSQL string, rightSQL string) string {
	switch op {
	case OpEqual:
		return fmt.Sprintf("%s = %s", leftSQL, rightSQL)
	case OpNotEqual:
		return fmt.Sprintf("%s != %s", leftSQL, rightSQL)
	case OpGreaterThan:
		return fmt.Sprintf("%s > %s", leftSQL, rightSQL)
	case OpGreaterThanOrEqual:
		return fmt.Sprintf("%s >= %s", leftSQL, rightSQL)
	case OpLessThan:
		return fmt.Sprintf("%s < %s", leftSQL, rightSQL)
	case OpLessThanOrEqual:
		return fmt.Sprintf("%s <= %s", leftSQL, rightSQL)
	default:
		return ""
	}
}
