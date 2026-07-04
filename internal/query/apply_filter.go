package query

import (
	"fmt"
	"math"
	"sort"
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

// quoteTableName safely quotes table names that may include schema prefixes.
// For table names like "schema.table", it splits on dots and quotes
// each part separately to produce "[schema].[table]" for MSSQL
// or "schema"."table" for PostgreSQL.
// For simple table names without schema, it behaves like quoteIdent.
// Supports multi-part names like "database.schema.table".
func quoteTableName(dialect string, tableName string) string {
	if tableName == "" {
		return tableName
	}

	// Split on dots to handle schema.table or database.schema.table notation
	parts := strings.Split(tableName, ".")
	if len(parts) == 1 {
		// No schema, just quote the table name
		return quoteIdent(dialect, tableName)
	}

	// Quote each part separately and rejoin with dots
	quotedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		// Skip empty parts (defensive against malformed input like "schema..table")
		if part != "" {
			quotedParts = append(quotedParts, quoteIdent(dialect, part))
		}
	}

	// Fallback to quoting the entire name if splitting resulted in no valid parts
	if len(quotedParts) == 0 {
		return quoteIdent(dialect, tableName)
	}

	return strings.Join(quotedParts, ".")
}

// getQuotedColumnName returns a properly quoted column name for use in SQL queries.
// For navigation property paths (e.g., "Category/Name"), it returns a fully qualified
// and quoted reference like "<TargetTableFromMetadata>"."column_name" to ensure PostgreSQL compatibility.
// Regular properties are also quoted to prevent issues with reserved keywords like "offset".
func getQuotedColumnName(dialect string, property string, entityMetadata *metadata.EntityMetadata) string {
	if entityMetadata == nil {
		// Quote the column name for PostgreSQL compatibility even without metadata
		return quoteIdent(dialect, toSnakeCase(property))
	}

	// Check if this is a navigation property path
	if _, navSegments, prop, prefix, err := resolveNavigationPropertyPath(property, entityMetadata); err == nil {
		alias := navigationAliasForPath(navSegments)
		columnName := prefix + prop.ColumnName
		if alias != "" {
			// Return fully quoted reference for proper PostgreSQL case handling
			return quoteIdent(dialect, alias) + "." + quoteIdent(dialect, columnName)
		}
	}

	// For regular properties, get the column name and quote it to handle reserved keywords
	// like "offset", "date", "time", etc. that can cause SQL syntax errors when unquoted
	columnName := GetColumnName(property, entityMetadata)
	// Only quote if not already a qualified path (contains a dot)
	if !strings.Contains(columnName, ".") {
		columnName = quoteIdent(dialect, columnName)
	}
	return columnName
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

	var applyJoins func(filter *FilterExpression)
	applyJoins = func(filter *FilterExpression) {
		if filter == nil {
			return
		}

		if filter.Property != "" {
			db = addNavigationJoinsForProperty(db, filter.Property, entityMetadata, joinedNavProps)
		}

		if filter.Left != nil {
			applyJoins(filter.Left)
		}
		if filter.Right != nil {
			applyJoins(filter.Right)
		}
	}

	applyJoins(filter)

	// Store the map in the GORM context so $orderby can re-use it and avoid duplicate JOINs
	db = db.Set("_joined_nav_props", joinedNavProps)

	return db
}

func addNavigationJoinsForProperty(db *gorm.DB, property string, entityMetadata *metadata.EntityMetadata, joinedNavProps map[string]bool) *gorm.DB {
	if entityMetadata == nil || property == "" {
		return db
	}

	targetMetadata, navSegments, _, _, err := resolveNavigationPropertyPath(property, entityMetadata)
	if err != nil || targetMetadata == nil || len(navSegments) == 0 {
		return db
	}

	currentMetadata := entityMetadata
	parentAlias := entityMetadata.TableName

	for i, navSegment := range navSegments {
		navProp := currentMetadata.FindNavigationProperty(navSegment)
		if navProp == nil || navProp.NavigationIsArray {
			return db
		}

		pathKey := strings.Join(navSegments[:i+1], "/")
		if joinedNavProps[pathKey] {
			currentMetadata, err = currentMetadata.ResolveNavigationTarget(navSegment)
			if err != nil || currentMetadata == nil {
				return db
			}
			parentAlias = navigationAliasForPath(navSegments[:i+1])
			continue
		}

		joinAlias := navigationAliasForPath(navSegments[:i+1])
		if joinAlias == "" {
			return db
		}

		db = addNavigationJoin(db, currentMetadata, parentAlias, navProp, joinAlias)
		joinedNavProps[pathKey] = true

		currentMetadata, err = currentMetadata.ResolveNavigationTarget(navSegment)
		if err != nil || currentMetadata == nil {
			return db
		}
		parentAlias = joinAlias
	}

	return db
}

// addNavigationJoin adds a JOIN clause for a single-entity navigation property
func addNavigationJoin(db *gorm.DB, parentMetadata *metadata.EntityMetadata, parentAlias string, navProp *metadata.PropertyMetadata, joinAlias string) *gorm.DB {
	// Get the related entity's table name from cached metadata
	// This was computed once during entity registration and respects custom TableName() methods
	relatedTableName := navProp.NavigationTargetTableName

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
		targetMetadata, err := parentMetadata.ResolveNavigationTarget(navProp.Name)
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
	quotedParentTable := quoteIdent(dialect, parentAlias)
	quotedForeignKey := quoteIdent(dialect, foreignKeyColumn)
	quotedPrimaryKey := quoteIdent(dialect, relatedPrimaryKey)
	quotedJoinAlias := quoteIdent(dialect, joinAlias)

	joinClause := fmt.Sprintf("LEFT JOIN %s AS %s ON %s.%s = %s.%s",
		quotedRelatedTable,
		quotedJoinAlias,
		quotedParentTable,
		quotedForeignKey,
		quotedJoinAlias,
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
	if countExpr, ok := buildCollectionCountExpression(dialect, propertyName, entityMetadata); ok {
		return countExpr
	}

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
		// PostgreSQL/MySQL/MariaDB/SQL Server don't reliably support referencing SELECT aliases in HAVING/WHERE
		if dialect == "postgres" || dialect == "mysql" || dialect == "mariadb" || dialect == "sqlserver" || dialect == "mssql" {
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
	if dialect == "postgres" || dialect == "mysql" || dialect == "mariadb" || dialect == "sqlserver" || dialect == "mssql" {
		if db != nil {
			if expr, ok := getAliasExprFromDB(db, rawName); ok {
				return expr
			}
		}
	}

	// Default: quote as identifier
	return quoteIdent(dialect, rawName)
}

func buildCollectionCountExpression(dialect string, propertyName string, entityMetadata *metadata.EntityMetadata) (string, bool) {
	ownerMetadata, navProp, err := resolveCollectionCountPath(propertyName, entityMetadata)
	if err != nil || ownerMetadata == nil || navProp == nil {
		return "", false
	}

	relatedTableName := strings.TrimSpace(navProp.NavigationTargetTableName)
	if relatedTableName == "" {
		return "", false
	}

	foreignKeyColumn := strings.TrimSpace(navProp.ForeignKeyColumnName)
	if foreignKeyColumn == "" {
		return "", false
	}

	parentTableName := strings.TrimSpace(ownerMetadata.TableName)
	if parentTableName == "" {
		return "", false
	}

	quotedRelatedTable := quoteIdent(dialect, relatedTableName)
	quotedParentTable := quoteIdent(dialect, parentTableName)
	foreignKeyColumns := strings.Split(foreignKeyColumn, ",")
	joinConditions := make([]string, 0, len(ownerMetadata.KeyProperties))

	if len(ownerMetadata.KeyProperties) == 0 {
		fkCol := strings.TrimSpace(foreignKeyColumns[0])
		if fkCol == "" {
			return "", false
		}
		joinConditions = append(joinConditions,
			fmt.Sprintf("%s.%s = %s.%s",
				quotedRelatedTable,
				quoteIdent(dialect, fkCol),
				quotedParentTable,
				quoteIdent(dialect, "id")))
	} else {
		for i, keyProp := range ownerMetadata.KeyProperties {
			fkCol := keyProp.ColumnName
			if i < len(foreignKeyColumns) {
				if candidate := strings.TrimSpace(foreignKeyColumns[i]); candidate != "" {
					fkCol = candidate
				}
			}

			joinConditions = append(joinConditions,
				fmt.Sprintf("%s.%s = %s.%s",
					quotedRelatedTable,
					quoteIdent(dialect, fkCol),
					quotedParentTable,
					quoteIdent(dialect, keyProp.ColumnName)))
		}
	}

	if len(joinConditions) == 0 {
		return "", false
	}

	return fmt.Sprintf("(SELECT COUNT(*) FROM %s WHERE %s)", quotedRelatedTable, strings.Join(joinConditions, " AND ")), true
}

// tryBuildRightSideFunctionComparison attempts to build a comparison when the right side is a function call.
// Returns the SQL string, arguments, and a boolean indicating if the comparison was successfully built.
func tryBuildRightSideFunctionComparison(dialect string, leftColumn string, operator FilterOperator, rightValue interface{}, entityMetadata *metadata.EntityMetadata) (string, []interface{}, bool) {
	// Check if right side is a FilterExpression (converted from FunctionCallExpr)
	rightExpr, ok := rightValue.(*FilterExpression)
	if !ok {
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
// Note: IN clause size validation is enforced during AST parsing, not here.
func buildStandardComparison(dialect string, operator FilterOperator, columnName string, value interface{}, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	if (dialect == "sqlserver" || dialect == "mssql") && isNonFiniteNumber(value) {
		// SQL Server drivers reject binding NaN/Inf values as query parameters.
		// Return deterministic non-crashing comparisons for these literals.
		switch operator {
		case OpNotEqual:
			return "1 = 1", []interface{}{}
		case OpEqual, OpGreaterThan, OpGreaterThanOrEqual, OpLessThan, OpLessThanOrEqual:
			return "1 = 0", []interface{}{}
		}
	}

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
		// Note: IN clause size limit is enforced during AST parsing in ast_parser_validation.go
		// No need for redundant validation here
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

	case OpMatchesPattern:
		return buildRegexComparison(dialect, columnName, value)

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

func isNonFiniteNumber(value interface{}) bool {
	switch v := value.(type) {
	case float64:
		return math.IsNaN(v) || math.IsInf(v, 0)
	case float32:
		f := float64(v)
		return math.IsNaN(f) || math.IsInf(f, 0)
	default:
		return false
	}
}

// buildComparisonCondition builds a comparison condition
func buildComparisonCondition(dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	return buildComparisonConditionWithDB(nil, dialect, filter, entityMetadata)
}

// buildComplexTypeNullComparison builds SQL for a complex type eq null or ne null comparison.
// Per OData v4.01 spec, eq null is true when all embedded scalar columns are null,
// and ne null is true when at least one embedded scalar column is non-null.
// Conditions are sorted deterministically by column name.
func buildComplexTypeNullComparison(dialect string, op FilterOperator, complexProp *metadata.PropertyMetadata) (string, []interface{}) {
	if len(complexProp.ComplexTypeFields) == 0 {
		return "", nil
	}

	prefix := complexProp.EmbeddedPrefix
	seen := make(map[string]bool)
	colNames := make([]string, 0, len(complexProp.ComplexTypeFields))

	for _, field := range complexProp.ComplexTypeFields {
		if field.IsNavigationProp || field.IsComplexType {
			continue
		}
		colName := prefix + field.ColumnName
		if seen[colName] {
			continue
		}
		seen[colName] = true
		colNames = append(colNames, colName)
	}

	if len(colNames) == 0 {
		return "", nil
	}

	sort.Strings(colNames)

	conditions := make([]string, len(colNames))
	for i, colName := range colNames {
		quotedCol := quoteIdent(dialect, colName)
		if op == OpEqual {
			conditions[i] = fmt.Sprintf("%s IS NULL", quotedCol)
		} else {
			conditions[i] = fmt.Sprintf("%s IS NOT NULL", quotedCol)
		}
	}

	if op == OpEqual {
		return fmt.Sprintf("(%s)", strings.Join(conditions, " AND ")), []interface{}{}
	}
	return fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")), []interface{}{}
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

	// Handle complex type eq null / ne null comparisons
	if filter.Property != "" && entityMetadata != nil && filter.Value == nil &&
		(filter.Operator == OpEqual || filter.Operator == OpNotEqual) {
		if complexProp := entityMetadata.FindComplexTypeProperty(filter.Property); complexProp != nil {
			return buildComplexTypeNullComparison(dialect, filter.Operator, complexProp)
		}
	}

	// Resolve the column name for the left side of the comparison
	columnName := resolveColumnName(db, dialect, filter.Property, entityMetadata)
	if columnName == "" {
		return "", nil
	}

	// Duration comparison: convert both sides to seconds so that ordering is numeric
	// rather than lexicographic. "P1D" > "PT1H" is false lexicographically but
	// true numerically (86400 > 3600), so we must compare as numbers.
	if filter.ValueType == "duration" {
		if durStr, ok := filter.Value.(string); ok {
			rhsSeconds := parseDurationToSeconds(durStr)
			var colSQL string
			if dialect == "postgres" {
				// PostgreSQL rejects "-P1D" with a SQL error rather than returning NULL.
				// Strip the leading '-' and negate manually so negative durations work.
				colSQL = fmt.Sprintf(
					"CASE WHEN %[1]s IS NULL THEN NULL "+
						"WHEN %[1]s LIKE 'P%%' THEN EXTRACT(EPOCH FROM CAST(%[1]s AS INTERVAL)) "+
						"WHEN %[1]s LIKE '-P%%' THEN -1.0*EXTRACT(EPOCH FROM CAST(SUBSTR(%[1]s,2) AS INTERVAL)) "+
						"ELSE NULL END",
					columnName)
			} else {
				colSQL = iso8601DurationToSecondsSQL(columnName, dialect)
			}
			compSQL := buildComparisonSQL(filter.Operator, colSQL)
			if compSQL != "" {
				return compSQL, []interface{}{rhsSeconds}
			}
			// Operator not supported for duration comparison; skip to avoid falling
			// through to string-based comparison which gives wrong results.
			return "", nil
		}
	}

	// Try to build a comparison with a function on the right side (e.g., Name eq tolower('JOHN'))
	if sql, args, ok := tryBuildRightSideFunctionComparison(dialect, columnName, filter.Operator, filter.Value, entityMetadata); ok {
		return sql, args
	}

	// Build a standard comparison
	sql, args := buildStandardComparison(dialect, filter.Operator, columnName, filter.Value, entityMetadata)

	return sql, args
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

	navTargetMetadata := getNavigationTargetMetadata(entityMetadata, navProp)

	// Use cached table name from metadata (computed once during entity registration)
	relatedTableName := navProp.NavigationTargetTableName

	// Use cached table name from parent entity metadata
	parentTableName := entityMetadata.TableName

	// Use cached foreign key column name from metadata
	// This was computed once during entity registration and respects GORM foreignKey: tags
	// For composite keys, this may contain comma-separated column names
	foreignKeyColumn := navProp.ForeignKeyColumnName

	// Build join conditions for all key properties
	// Support both single keys and composite keys
	var joinConditions []string

	// Parse foreign key column names (may be comma-separated for composite keys)
	foreignKeyColumns := strings.Split(foreignKeyColumn, ",")

	// Get parent key properties
	if len(entityMetadata.KeyProperties) == 0 {
		// Fallback to default "id" if no key properties found
		quotedRelatedTable := quoteIdent(dialect, relatedTableName)
		quotedParentTable := quoteIdent(dialect, parentTableName)
		quotedForeignKey := quoteIdent(dialect, strings.TrimSpace(foreignKeyColumns[0]))
		quotedParentPK := quoteIdent(dialect, "id")
		joinConditions = append(joinConditions,
			fmt.Sprintf("%s.%s = %s.%s", quotedRelatedTable, quotedForeignKey, quotedParentTable, quotedParentPK))
	} else {
		// Match foreign key columns with parent key properties
		// For composite keys, we need to join on all key columns

		// Validate that foreign key column count matches key property count
		if len(foreignKeyColumns) != len(entityMetadata.KeyProperties) {
			// Log warning about potential GORM tag configuration error
			// This may indicate that the foreignKey tag is incomplete or incorrect
			fmt.Printf("Warning: Foreign key column count (%d) does not match key property count (%d) for navigation property '%s'. "+
				"This may indicate a GORM tag configuration error. Foreign keys: %v, Key properties: %v\n",
				len(foreignKeyColumns), len(entityMetadata.KeyProperties), navProp.Name,
				foreignKeyColumns, func() []string {
					names := make([]string, len(entityMetadata.KeyProperties))
					for i, kp := range entityMetadata.KeyProperties {
						names[i] = kp.Name
					}
					return names
				}())
		}

		// Quote table names once outside the loop
		quotedRelatedTable := quoteIdent(dialect, relatedTableName)
		quotedParentTable := quoteIdent(dialect, parentTableName)

		for i, keyProp := range entityMetadata.KeyProperties {
			// Get the corresponding foreign key column
			var fkColumn string
			if i < len(foreignKeyColumns) {
				fkColumn = strings.TrimSpace(foreignKeyColumns[i])
			} else {
				// Fallback: use the key property column name as foreign key column name
				fkColumn = keyProp.ColumnName
			}

			quotedForeignKey := quoteIdent(dialect, fkColumn)
			quotedParentPK := quoteIdent(dialect, keyProp.ColumnName)

			joinConditions = append(joinConditions,
				fmt.Sprintf("%s.%s = %s.%s", quotedRelatedTable, quotedForeignKey, quotedParentTable, quotedParentPK))
		}
	}

	// Combine all join conditions with AND
	joinCondition := strings.Join(joinConditions, " AND ")

	if filter.Value == nil || filter.Left == nil {
		if filter.Operator == OpAny {
			return fmt.Sprintf("EXISTS (SELECT 1 FROM %s WHERE %s)",
				quoteIdent(dialect, relatedTableName), joinCondition), []interface{}{}
		}
		return fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s WHERE %s) OR EXISTS (SELECT 1 FROM %s WHERE %s)",
			quoteIdent(dialect, relatedTableName), joinCondition,
			quoteIdent(dialect, relatedTableName), joinCondition), []interface{}{}
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
		sql = fmt.Sprintf("EXISTS (SELECT 1 FROM %s WHERE %s AND (%s))",
			quoteIdent(dialect, relatedTableName), joinCondition, predicateSQL)
	} else {
		sql = fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s WHERE %s AND NOT (%s))",
			quoteIdent(dialect, relatedTableName), joinCondition, predicateSQL)
	}

	return sql, predicateArgs
}

func getNavigationTargetMetadata(entityMetadata *metadata.EntityMetadata, navProp *metadata.PropertyMetadata) *metadata.EntityMetadata {
	if entityMetadata == nil || navProp == nil {
		return nil
	}

	targetMeta, err := entityMetadata.ResolveNavigationTarget(navProp.Name)
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

	// Handle nested lambda operators (any/all within any/all), e.g.
	// CollectionA/any(a: a/CollectionB/any(b: b/Property eq 'value')). navTargetMetadata
	// is the entity the outer lambda's range variable ranges over, which is exactly the
	// "current" entity metadata that buildLambdaCondition needs to resolve the nested
	// navigation property and recurse into a nested EXISTS subquery.
	if filter.Operator == OpAny || filter.Operator == OpAll {
		return buildLambdaCondition(dialect, filter, navTargetMetadata)
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
	case OpMatchesPattern:
		return buildRegexComparison(dialect, columnName, filter.Value)
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
	var funcSQL string
	var funcArgs []interface{}
	if isArithmeticFilterOperator(funcExpr.Operator) {
		funcSQL, funcArgs = buildComputeExpressionSQLWithArgs(dialect, funcExpr, navTargetMetadata)
		if funcSQL == "" {
			return "", nil
		}
	} else {
		// Quote the column name for proper database compatibility
		columnName := quoteIdent(dialect, GetColumnName(funcExpr.Property, navTargetMetadata))
		funcSQL, funcArgs = buildFunctionSQL(dialect, funcExpr.Operator, columnName, funcExpr.Value)
		if funcSQL == "" {
			return "", nil
		}
	}

	compSQL := buildComparisonSQL(filter.Operator, funcSQL)
	if compSQL == "" {
		return "", nil
	}

	allArgs := append(funcArgs, filter.Value)
	return compSQL, allArgs
}

func isArithmeticFilterOperator(op FilterOperator) bool {
	switch op {
	case OpAdd, OpSub, OpMul, OpDiv, OpDivBy, OpMod:
		return true
	default:
		return false
	}
}

// buildFunctionComparison builds a comparison with a function on the left side
func buildFunctionComparison(dialect string, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	funcExpr := filter.Left
	var funcSQL string
	var funcArgs []interface{}
	if isArithmeticFilterOperator(funcExpr.Operator) {
		funcSQL, funcArgs = buildComputeExpressionSQLWithArgs(dialect, funcExpr, entityMetadata)
		if funcSQL == "" {
			return "", nil
		}
	} else {
		columnName := getQuotedColumnName(dialect, funcExpr.Property, entityMetadata)
		funcSQL, funcArgs = buildFunctionSQL(dialect, funcExpr.Operator, columnName, funcExpr.Value)
		if funcSQL == "" {
			return "", nil
		}
	}

	// Check if right side is a FilterExpression (converted from FunctionCallExpr)
	if rightFuncExpr, ok := filter.Value.(*FilterExpression); ok {
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

// iso8601DurationToSecondsSQL builds a CASE expression that converts an ISO-8601
// duration string column (e.g. "P1D", "PT2H", "P1DT2H30M") into total seconds.
// Edm.Duration values are stored as ISO-8601 strings, so each dialect parses the
// string with its own primitives (INSTR/SUBSTR for SQLite, LOCATE/SUBSTRING for
// MySQL/MariaDB, CHARINDEX/SUBSTRING for SQL Server). The algorithm mirrors the
// SQLite implementation used in OpTotalSeconds.
func iso8601DurationToSecondsSQL(c, dialect string) string {
	var instr func(hay, needle string) string
	var substr3 func(s, start, length string) string
	var substr2 func(s, start string) string
	var castInt func(x string) string
	var castReal func(x string) string
	var stripLeadingChar func(expr string) string

	switch dialect {
	case "mysql", "mariadb":
		instr = func(hay, needle string) string { return "LOCATE(" + needle + "," + hay + ")" }
		substr3 = func(s, start, length string) string { return "SUBSTRING(" + s + "," + start + "," + length + ")" }
		substr2 = func(s, start string) string { return "SUBSTRING(" + s + "," + start + ")" }
		castInt = func(x string) string { return "CAST(" + x + " AS SIGNED)" }
		castReal = func(x string) string { return "CAST(" + x + " AS DECIMAL(38,9))" }
		stripLeadingChar = func(expr string) string { return "SUBSTRING(" + expr + ",2)" }
	case "sqlserver", "mssql":
		instr = func(hay, needle string) string { return "CHARINDEX(" + needle + "," + hay + ")" }
		substr3 = func(s, start, length string) string { return "SUBSTRING(" + s + "," + start + "," + length + ")" }
		// SQL Server's SUBSTRING requires a length; 8000 covers any duration literal.
		substr2 = func(s, start string) string { return "SUBSTRING(" + s + "," + start + ",8000)" }
		castInt = func(x string) string { return "CAST(" + x + " AS INT)" }
		castReal = func(x string) string { return "CAST(" + x + " AS FLOAT)" }
		stripLeadingChar = func(expr string) string { return "SUBSTRING(" + expr + ",2,8000)" }
	default: // sqlite
		instr = func(hay, needle string) string { return "INSTR(" + hay + "," + needle + ")" }
		substr3 = func(s, start, length string) string { return "SUBSTR(" + s + "," + start + "," + length + ")" }
		substr2 = func(s, start string) string { return "SUBSTR(" + s + "," + start + ")" }
		castInt = func(x string) string { return "CAST(" + x + " AS INTEGER)" }
		castReal = func(x string) string { return "CAST(" + x + " AS REAL)" }
		stripLeadingChar = func(expr string) string { return "SUBSTR(" + expr + ",2)" }
	}

	// buildParseExpr returns a SQL arithmetic expression that converts the
	// ISO-8601 duration in col (which must start with 'P') to total seconds.
	buildParseExpr := func(col string) string {
		t := instr(col, "'T'")
		d := instr(col, "'D'")
		h := instr(col, "'H'")
		s := instr(col, "'S'")
		afterT := substr2(col, t+"+1")
		mRel := instr(afterT, "'M'")
		mAbs := t + "+" + mRel

		// Days: digits before the first 'D' when it precedes any 'T'.
		days := castInt("CASE WHEN "+d+">0 AND ("+t+"=0 OR "+d+"<"+t+")"+
			" THEN "+substr3(col, "2", d+"-2")+" ELSE 0 END") + "*86400"
		// Hours: digits between 'T' and 'H'.
		hours := castInt("CASE WHEN "+t+">0 AND "+h+">"+t+
			" THEN "+substr3(col, t+"+1", h+"-"+t+"-1")+" ELSE 0 END") + "*3600"
		// Minutes: digits before 'M' (after 'T'), starting after 'H' if present.
		minStart := "CASE WHEN " + h + ">" + t + " THEN " + h + "+1 ELSE " + t + "+1 END"
		minLen := mAbs + "-1-CASE WHEN " + h + ">" + t + " THEN " + h + " ELSE " + t + " END"
		minutes := castInt("CASE WHEN "+t+">0 AND "+mRel+">0"+
			" THEN "+substr3(col, minStart, minLen)+" ELSE 0 END") + "*60"
		// Seconds: digits before 'S', starting after the last of 'M'/'H'/'T'.
		secStart := "CASE WHEN " + mRel + ">0 THEN " + mAbs + "+1 WHEN " + h + ">" + t + " THEN " + h + "+1 ELSE " + t + "+1 END"
		secLen := s + "-CASE WHEN " + mRel + ">0 THEN " + mAbs + " WHEN " + h + ">" + t + " THEN " + h + " ELSE " + t + " END-1"
		// Use '0' (string) not 0 (int) in the ELSE branch: SQL Server resolves CASE result
		// type by precedence — mixing nvarchar (SUBSTRING) with int (0) picks int, which
		// then fails to convert '1.5' to int for fractional seconds. Both branches as
		// nvarchar lets the outer castReal convert correctly.
		seconds := castReal("CASE WHEN " + t + ">0 AND " + s + ">" + t +
			" THEN " + substr3(col, secStart, secLen) + " ELSE '0' END")

		return days + "+" + hours + "+" + minutes + "+" + seconds
	}

	posExpr := buildParseExpr(c)
	// For negative durations (e.g. "-P1D"), strip the leading '-' so that the
	// P-prefixed remainder can be parsed with the same logic, then negate.
	negExpr := buildParseExpr(stripLeadingChar(c))

	return "CASE WHEN " + c + " IS NULL THEN NULL " +
		"WHEN " + c + " LIKE 'P%' THEN (" + posExpr + ") " +
		"WHEN " + c + " LIKE '-P%' THEN -1.0*(" + negExpr + ") " +
		"ELSE NULL END"
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
		switch dialect {
		case "sqlserver", "mssql":
			return fmt.Sprintf("LEN(%s)", columnName), nil
		default:
			return fmt.Sprintf("LENGTH(%s)", columnName), nil
		}
	case OpIndexOf:
		if dialect == "postgres" {
			return fmt.Sprintf("(POSITION(? IN %s) - 1)", columnName), []interface{}{value}
		}
		if dialect == "sqlserver" || dialect == "mssql" {
			return fmt.Sprintf("CHARINDEX(?, %s) - 1", columnName), []interface{}{value}
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

			// First argument can be a FilterExpression (nested function) or a literal
			if filterExpr, ok := firstArg.(*FilterExpression); ok {
				leftSQL, leftArgs = buildFunctionSQL(dialect, filterExpr.Operator, filterExpr.Property, filterExpr.Value)
			} else {
				leftSQL = "?"
				leftArgs = []interface{}{firstArg}
			}

			var rightSQL string
			var rightArgs []interface{}

			// Second argument can be a FilterExpression (nested function), string (property), or literal
			if filterExpr, ok := secondArg.(*FilterExpression); ok {
				rightSQL, rightArgs = buildFunctionSQL(dialect, filterExpr.Operator, filterExpr.Property, filterExpr.Value)
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

		// Handle FilterExpression (converted from nested function call)
		if filterExpr, ok := value.(*FilterExpression); ok {
			rightColumnName := filterExpr.Property
			rightSQL, rightArgs := buildFunctionSQL(dialect, filterExpr.Operator, rightColumnName, filterExpr.Value)
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
		// For PostgreSQL, cast to prevent smallint overflow in arithmetic
		if dialect == "postgres" {
			return fmt.Sprintf("(CAST(%s AS BIGINT) * ?)", columnName), []interface{}{value}
		}
		return fmt.Sprintf("(%s * ?)", columnName), []interface{}{value}
	case OpDiv:
		return fmt.Sprintf("(%s / ?)", columnName), []interface{}{value}
	case OpDivBy:
		// divby performs decimal (floating-point) division; cast to avoid integer truncation
		switch dialect {
		case "postgres":
			return fmt.Sprintf("(CAST(%s AS FLOAT) / ?)", columnName), []interface{}{value}
		case "mysql", "mariadb":
			return fmt.Sprintf("(CAST(%s AS DOUBLE) / ?)", columnName), []interface{}{value}
		default:
			return fmt.Sprintf("(CAST(%s AS REAL) / ?)", columnName), []interface{}{value}
		}
	case OpMod:
		if dialect == "sqlserver" || dialect == "mssql" {
			return fmt.Sprintf("(CAST(%s AS DECIMAL(38,10)) %% CAST(? AS DECIMAL(38,10)))", columnName), []interface{}{value}
		}
		return fmt.Sprintf("(%s %% ?)", columnName), []interface{}{value}
	case OpYear:
		switch dialect {
		case "postgres":
			// Normalize via TEXT first so this works for both temporal columns and string-backed values.
			return fmt.Sprintf("(EXTRACT(YEAR FROM CAST(NULLIF(CAST(%s AS TEXT), '') AS TIMESTAMP))::INT)", columnName), nil
		case "sqlserver", "mssql":
			return fmt.Sprintf("DATEPART(YEAR, TRY_CONVERT(datetime2, %s))", columnName), nil
		case "mysql", "mariadb":
			// MySQL/MariaDB can handle YEAR() on string dates
			return fmt.Sprintf("YEAR(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%Y', %s) AS INTEGER)", columnName), nil
		}
	case OpMonth:
		switch dialect {
		case "postgres":
			// Normalize via TEXT first so this works for both temporal columns and string-backed values.
			return fmt.Sprintf("(EXTRACT(MONTH FROM CAST(NULLIF(CAST(%s AS TEXT), '') AS TIMESTAMP))::INT)", columnName), nil
		case "sqlserver", "mssql":
			return fmt.Sprintf("DATEPART(MONTH, TRY_CONVERT(datetime2, %s))", columnName), nil
		case "mysql", "mariadb":
			// MySQL/MariaDB can handle MONTH() on string dates
			return fmt.Sprintf("MONTH(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%m', %s) AS INTEGER)", columnName), nil
		}
	case OpDay:
		switch dialect {
		case "postgres":
			// Normalize via TEXT first so this works for both temporal columns and string-backed values.
			return fmt.Sprintf("(EXTRACT(DAY FROM CAST(NULLIF(CAST(%s AS TEXT), '') AS TIMESTAMP))::INT)", columnName), nil
		case "sqlserver", "mssql":
			return fmt.Sprintf("DATEPART(DAY, TRY_CONVERT(datetime2, %s))", columnName), nil
		case "mysql", "mariadb":
			// MySQL/MariaDB can handle DAY() on string dates
			return fmt.Sprintf("DAY(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%d', %s) AS INTEGER)", columnName), nil
		}
	case OpHour:
		switch dialect {
		case "postgres":
			// Normalize via TEXT first so this works for both temporal columns and string-backed values.
			return fmt.Sprintf("(EXTRACT(HOUR FROM CAST(NULLIF(CAST(%s AS TEXT), '') AS TIME))::INT)", columnName), nil
		case "sqlserver", "mssql":
			return fmt.Sprintf("DATEPART(HOUR, TRY_CONVERT(time, %s))", columnName), nil
		case "mysql", "mariadb":
			// MySQL/MariaDB can handle HOUR() on string time columns
			return fmt.Sprintf("HOUR(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%H', %s) AS INTEGER)", columnName), nil
		}
	case OpMinute:
		switch dialect {
		case "postgres":
			// Normalize via TEXT first so this works for both temporal columns and string-backed values.
			return fmt.Sprintf("(EXTRACT(MINUTE FROM CAST(NULLIF(CAST(%s AS TEXT), '') AS TIME))::INT)", columnName), nil
		case "sqlserver", "mssql":
			return fmt.Sprintf("DATEPART(MINUTE, TRY_CONVERT(time, %s))", columnName), nil
		case "mysql", "mariadb":
			// MySQL/MariaDB can handle MINUTE() on string time columns
			return fmt.Sprintf("MINUTE(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%M', %s) AS INTEGER)", columnName), nil
		}
	case OpSecond:
		switch dialect {
		case "postgres":
			// Normalize via TEXT first so this works for both temporal columns and string-backed values.
			return fmt.Sprintf("(EXTRACT(SECOND FROM CAST(NULLIF(CAST(%s AS TEXT), '') AS TIME))::INT)", columnName), nil
		case "sqlserver", "mssql":
			return fmt.Sprintf("DATEPART(SECOND, TRY_CONVERT(time, %s))", columnName), nil
		case "mysql", "mariadb":
			// MySQL/MariaDB can handle SECOND() on string time columns
			return fmt.Sprintf("SECOND(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("CAST(strftime('%%S', %s) AS INTEGER)", columnName), nil
		}
	case OpDate:
		switch dialect {
		case "postgres":
			// Normalize via TEXT first so this works for both temporal columns and string-backed values.
			return fmt.Sprintf("(CAST(NULLIF(CAST(%s AS TEXT), '') AS DATE))", columnName), nil
		case "sqlserver", "mssql":
			return fmt.Sprintf("CAST(TRY_CONVERT(date, %s) AS DATE)", columnName), nil
		case "mysql", "mariadb":
			return fmt.Sprintf("DATE(%s)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("DATE(%s)", columnName), nil
		}
	case OpTime:
		switch dialect {
		case "postgres":
			// Normalize via TEXT first so this works for both temporal columns and string-backed values.
			return fmt.Sprintf("(CAST(NULLIF(CAST(%s AS TEXT), '') AS TIME))", columnName), nil
		case "sqlserver", "mssql":
			return fmt.Sprintf("CAST(TRY_CONVERT(time, %s) AS TIME)", columnName), nil
		case "mysql", "mariadb":
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
		case "sqlserver", "mssql":
			return "SYSDATETIMEOFFSET()", nil
		default: // sqlite
			return "datetime('now')", nil
		}
	case OpFractionalSeconds:
		switch dialect {
		case "postgres":
			// Use a Go variable for the text-normalized cast to avoid duplicating the expression.
			normalizedCast := fmt.Sprintf("CAST(NULLIF(CAST(%s AS TEXT), '') AS TIMESTAMP)", columnName)
			return fmt.Sprintf("(EXTRACT(SECOND FROM %s) - FLOOR(EXTRACT(SECOND FROM %s)))", normalizedCast, normalizedCast), nil
		case "sqlserver", "mssql":
			return fmt.Sprintf("(DATEPART(MICROSECOND, TRY_CONVERT(datetime2, %s)) / 1000000.0)", columnName), nil
		case "mysql", "mariadb":
			return fmt.Sprintf("(MICROSECOND(%s) / 1000000.0)", columnName), nil
		default: // sqlite
			return fmt.Sprintf("(CAST(strftime('%%f', %s) AS REAL) - CAST(strftime('%%S', %s) AS INTEGER))", columnName, columnName), nil
		}
	case OpTotalOffsetMinutes:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("(EXTRACT(TIMEZONE FROM CAST(NULLIF(CAST(%s AS TEXT), '') AS TIMESTAMPTZ)) / 60)::INT", columnName), nil
		case "sqlserver", "mssql":
			return fmt.Sprintf("DATEPART(TZOFFSET, TRY_CONVERT(datetimeoffset, %s))", columnName), nil
		default: // sqlite, mysql - timezone offset not natively stored; return 0
			return "0", nil
		}
	case OpTotalSeconds:
		switch dialect {
		case "postgres":
			return fmt.Sprintf("EXTRACT(EPOCH FROM CAST(NULLIF(CAST(%s AS TEXT), '') AS INTERVAL))", columnName), nil
		case "mysql", "mariadb":
			// Edm.Duration is stored as an ISO-8601 string; parse it rather than
			// treating it as a clock TIME (TIME_TO_SEC cannot read "P1D").
			return iso8601DurationToSecondsSQL(columnName, "mysql"), nil
		case "sqlserver", "mssql":
			return iso8601DurationToSecondsSQL(columnName, "sqlserver"), nil
		default: // sqlite
			return iso8601DurationToSecondsSQL(columnName, dialect), nil
		}
	case OpMinDatetime:
		switch dialect {
		case "postgres":
			return "'0001-01-01 00:00:00+00'::TIMESTAMPTZ", nil
		case "mysql", "mariadb":
			return "'0001-01-01 00:00:00'", nil
		case "sqlserver", "mssql":
			// SQL Server has no datetime() function; use datetime2 which (unlike the
			// legacy datetime type) covers years from 0001.
			return "CAST('0001-01-01T00:00:00' AS datetime2)", nil
		default: // sqlite
			return "datetime('0001-01-01T00:00:00')", nil
		}
	case OpMaxDatetime:
		switch dialect {
		case "postgres":
			return "'9999-12-31 23:59:59.9999999+00'::TIMESTAMPTZ", nil
		case "mysql", "mariadb":
			return "'9999-12-31 23:59:59'", nil
		case "sqlserver", "mssql":
			return "CAST('9999-12-31T23:59:59' AS datetime2)", nil
		default: // sqlite
			return "datetime('9999-12-31T23:59:59')", nil
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
		if dialect == "sqlserver" || dialect == "mssql" {
			return fmt.Sprintf("ROUND(%s, 0)", columnName), nil
		}
		return fmt.Sprintf("ROUND(%s)", columnName), nil
	case OpCast:
		if typeName, ok := value.(string); ok {
			switch typeName {
			case "Edm.Date":
				switch dialect {
				case "postgres", "postgresql":
					return fmt.Sprintf("CAST(%s AS DATE)", columnName), nil
				case "mysql", "mariadb":
					return fmt.Sprintf("DATE(%s)", columnName), nil
				case "sqlserver", "mssql":
					return fmt.Sprintf("CAST(TRY_CONVERT(date, %s) AS DATE)", columnName), nil
				default: // sqlite
					return fmt.Sprintf("date(%s)", columnName), nil
				}
			case "Edm.TimeOfDay":
				switch dialect {
				case "postgres", "postgresql":
					return fmt.Sprintf("CAST(%s AS TIME)", columnName), nil
				case "mysql", "mariadb":
					return fmt.Sprintf("TIME(%s)", columnName), nil
				case "sqlserver", "mssql":
					return fmt.Sprintf("CAST(TRY_CONVERT(time, %s) AS TIME)", columnName), nil
				default: // sqlite
					return fmt.Sprintf("time(%s)", columnName), nil
				}
			}
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
	case "sqlserver", "mssql":
		switch edmType {
		case "Edm.String":
			return "NVARCHAR(MAX)"
		case "Edm.Int32":
			return "INT"
		case "Edm.Int16":
			return "SMALLINT"
		case "Edm.Byte":
			return "TINYINT"
		case "Edm.SByte":
			return "SMALLINT"
		case "Edm.Int64":
			return "BIGINT"
		case "Edm.Decimal":
			return "DECIMAL(38, 18)"
		case "Edm.Double", "Edm.Single":
			return "FLOAT"
		case "Edm.Boolean":
			return "BIT"
		case "Edm.DateTimeOffset":
			return "DATETIMEOFFSET"
		case "Edm.Date":
			return "DATE"
		case "Edm.TimeOfDay":
			return "TIME"
		case "Edm.Guid":
			return "UNIQUEIDENTIFIER"
		case "Edm.Binary":
			return "VARBINARY(MAX)"
		default:
			return "NVARCHAR(MAX)"
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
		if dialect == "sqlserver" || dialect == "mssql" {
			return fmt.Sprintf("SUBSTRING(%s, ? + 1, LEN(%s))", columnName, columnName), []interface{}{args[0]}
		}
		return fmt.Sprintf("SUBSTR(%s, ? + 1, LENGTH(%s))", columnName, columnName), []interface{}{args[0]}
	}
	if len(args) == 2 {
		if dialect == "postgres" {
			return fmt.Sprintf("SUBSTRING(%s FROM ? + 1 FOR ?)", columnName), args
		}
		if dialect == "sqlserver" || dialect == "mssql" {
			return fmt.Sprintf("SUBSTRING(%s, ? + 1, ?)", columnName), args
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
