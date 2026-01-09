package query

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// aliasExprsKey is the key used to store alias expressions in GORM's context.
// This replaces the previous global variable approach to ensure thread-safety
// when multiple requests are processed concurrently.
const aliasExprsKey = "_odata_alias_exprs"

// applyTransformations applies apply transformations to the GORM query
func applyTransformations(db *gorm.DB, transformations []ApplyTransformation, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if db.Statement != nil && db.Statement.Dest == nil {
		modelInstance := reflect.New(entityMetadata.EntityType).Interface()
		db = db.Model(modelInstance)
	}

	dialect := getDatabaseDialect(db)

	hasGrouping := false
	// Initialize alias expressions map in GORM context for this query
	db = setAliasExprsInDB(db, make(map[string]string))

	for _, transformation := range transformations {
		switch transformation.Type {
		case ApplyTypeGroupBy:
			db = applyGroupBy(db, transformation.GroupBy, entityMetadata)
			hasGrouping = true
		case ApplyTypeAggregate:
			db = applyAggregate(db, transformation.Aggregate, entityMetadata)
			hasGrouping = true
		case ApplyTypeFilter:
			if hasGrouping {
				db = applyHavingFilter(db, transformation.Filter, entityMetadata)
			} else {
				db = applyFilter(db, transformation.Filter, entityMetadata)
			}
		case ApplyTypeCompute:
			db = applyCompute(db, dialect, transformation.Compute, entityMetadata)
		}
	}
	return db
}

// applyGroupBy applies a groupby transformation to the GORM query
func applyGroupBy(db *gorm.DB, groupBy *GroupByTransformation, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if groupBy == nil || len(groupBy.Properties) == 0 {
		return db
	}

	groupByColumns := make([]string, 0, len(groupBy.Properties))
	selectColumns := make([]string, 0, len(groupBy.Properties))

	// Determine dialect and table name for proper identifier quoting
	dialect := getDatabaseDialect(db)
	tableName := entityMetadata.TableName

	for _, propName := range groupBy.Properties {
		prop := findProperty(propName, entityMetadata)
		if prop == nil {
			continue
		}

		columnName := GetColumnName(propName, entityMetadata)
		// Qualify and quote to avoid ambiguity and preserve case-sensitivity
		qualified := quoteIdent(dialect, tableName) + "." + quoteIdent(dialect, columnName)
		groupByColumns = append(groupByColumns, qualified)
		selectColumns = append(selectColumns, fmt.Sprintf("%s as %s", qualified, quoteIdent(dialect, prop.JsonName)))
	}

	if len(groupBy.Transform) > 0 {
		for _, trans := range groupBy.Transform {
			if trans.Type == ApplyTypeAggregate && trans.Aggregate != nil {
				for _, aggExpr := range trans.Aggregate.Expressions {
					aggSQL := buildAggregateSQLWithDB(db, dialect, aggExpr, entityMetadata)
					if aggSQL != "" {
						selectColumns = append(selectColumns, aggSQL)
					}
				}
			}
		}
	} else {
		// Default to COUNT(*) when no aggregate is specified with groupby
		// Quote alias per dialect
		selectColumns = append(selectColumns, fmt.Sprintf("COUNT(*) as %s", quoteIdent(dialect, "$count")))
		// Record $count alias for HAVING clause support in PostgreSQL
		db = setAliasExprInDB(db, "$count", "COUNT(*)")
	}

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

	selectColumns := make([]string, 0, len(aggregate.Expressions))
	dialect := getDatabaseDialect(db)
	for _, aggExpr := range aggregate.Expressions {
		aggSQL := buildAggregateSQLWithDB(db, dialect, aggExpr, entityMetadata)
		if aggSQL != "" {
			selectColumns = append(selectColumns, aggSQL)
		}
	}

	if len(selectColumns) > 0 {
		db = db.Select(strings.Join(selectColumns, ", "))
	}

	return db
}

// setAliasExprsInDB stores the alias expressions map in the GORM db context
func setAliasExprsInDB(db *gorm.DB, exprs map[string]string) *gorm.DB {
	return db.Set(aliasExprsKey, exprs)
}

// getAliasExprsFromDB retrieves the alias expressions map from the GORM db context
func getAliasExprsFromDB(db *gorm.DB) map[string]string {
	if val, ok := db.Get(aliasExprsKey); ok {
		if exprs, ok := val.(map[string]string); ok {
			return exprs
		}
	}
	return nil
}

// setAliasExprInDB stores a single alias expression in the GORM db context.
// To avoid race conditions, this function creates a new map copy with the new entry
// rather than modifying the existing map in place.
func setAliasExprInDB(db *gorm.DB, alias, expr string) *gorm.DB {
	current := getAliasExprsFromDB(db)
	newExprs := make(map[string]string)
	for k, v := range current {
		newExprs[k] = v
	}
	newExprs[alias] = expr
	return db.Set(aliasExprsKey, newExprs)
}

// getAliasExprFromDB retrieves an alias expression from the GORM db context
func getAliasExprFromDB(db *gorm.DB, alias string) (string, bool) {
	exprs := getAliasExprsFromDB(db)
	if exprs == nil {
		return "", false
	}
	expr, ok := exprs[alias]
	return expr, ok
}

// resetAliasExprs resets the alias expressions map.
// Note: This function is now a no-op as alias expressions are stored in GORM db context.
// For concurrent safety, use setAliasExprsInDB with an empty map instead.
// This function is kept for backward compatibility with existing tests.
func resetAliasExprs() {
	// No-op: The new implementation uses GORM context which is automatically scoped per-query
}

// buildAggregateSQL builds the SQL for an aggregate expression
// Deprecated: Use buildAggregateSQLWithDB for concurrent safety
func buildAggregateSQL(dialect string, aggExpr AggregateExpression, entityMetadata *metadata.EntityMetadata) string {
	return buildAggregateSQLInternal(nil, dialect, aggExpr, entityMetadata)
}

// buildAggregateSQLWithDB builds the SQL for an aggregate expression with GORM db context
func buildAggregateSQLWithDB(db *gorm.DB, dialect string, aggExpr AggregateExpression, entityMetadata *metadata.EntityMetadata) string {
	return buildAggregateSQLInternal(db, dialect, aggExpr, entityMetadata)
}

// buildAggregateSQLInternal builds the SQL for an aggregate expression with optional db context
func buildAggregateSQLInternal(db *gorm.DB, dialect string, aggExpr AggregateExpression, entityMetadata *metadata.EntityMetadata) string {
	// Helper function to record alias in the db context.
	// Note: State is scoped to a single *gorm.DB session created via Session() or
	// WithContext(), so different HTTP requests do not share this map. However,
	// this does NOT by itself make updates to the underlying map safe across
	// multiple goroutines that share the same db instance within a single request.
	// Callers must avoid concurrent use of the same db for aggregate building, or
	// provide their own synchronization when using goroutines.
	//
	// To avoid concurrent writes to a shared map, always create a new map
	// (copying any existing entries) and store it back via setAliasExprsInDB.
	recordAlias := func(alias, expr string) {
		if db != nil {
			current := getAliasExprsFromDB(db)

			newExprs := make(map[string]string)
			for k, v := range current {
				newExprs[k] = v
			}

			newExprs[alias] = expr
			db = setAliasExprsInDB(db, newExprs)
		}
	}

	if aggExpr.Property == "$count" {
		// Record for HAVING clause support in PostgreSQL
		recordAlias(aggExpr.Alias, "COUNT(*)")
		return fmt.Sprintf("COUNT(*) as %s", quoteIdent(dialect, aggExpr.Alias))
	}

	prop := findProperty(aggExpr.Property, entityMetadata)
	if prop == nil {
		return ""
	}

	columnName := GetColumnName(aggExpr.Property, entityMetadata)
	// Qualify with table name to avoid ambiguity and quote identifiers
	qualified := quoteIdent(dialect, entityMetadata.TableName) + "." + quoteIdent(dialect, columnName)

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
		expr := fmt.Sprintf("COUNT(DISTINCT %s)", qualified)
		// Record for HAVING clause support in PostgreSQL
		recordAlias(aggExpr.Alias, expr)
		return fmt.Sprintf("%s as %s", expr, quoteIdent(dialect, aggExpr.Alias))
	default:
		return ""
	}

	expr := fmt.Sprintf("%s(%s)", sqlFunc, qualified)
	// Record for HAVING clause support in PostgreSQL
	recordAlias(aggExpr.Alias, expr)
	return fmt.Sprintf("%s as %s", expr, quoteIdent(dialect, aggExpr.Alias))
}

// applyCompute applies a compute transformation to the GORM query
func applyCompute(db *gorm.DB, dialect string, compute *ComputeTransformation, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if compute == nil || len(compute.Expressions) == 0 {
		return db
	}

	if db.Statement != nil && db.Statement.Dest == nil {
		modelInstance := reflect.New(entityMetadata.EntityType).Interface()
		db = db.Model(modelInstance)
	}

	selectColumns := make([]string, 0)
	tableName := entityMetadata.TableName

	streamAuxFields := make(map[string]bool)
	for _, streamProp := range entityMetadata.StreamProperties {
		if streamProp.StreamContentTypeField != "" {
			streamAuxFields[streamProp.StreamContentTypeField] = true
		}
		if streamProp.StreamContentField != "" {
			streamAuxFields[streamProp.StreamContentField] = true
		}
	}

	for _, prop := range entityMetadata.Properties {
		if !prop.IsNavigationProp && !prop.IsComplexType && !prop.IsStream && !streamAuxFields[prop.FieldName] {
			// Qualify and quote identifiers for compatibility across dialects
			qualified := quoteIdent(dialect, tableName) + "." + quoteIdent(dialect, prop.ColumnName)
			selectColumns = append(selectColumns, fmt.Sprintf("%s as %s", qualified, quoteIdent(dialect, prop.JsonName)))
		}
	}

	// Ensure alias expressions map exists in db context
	if getAliasExprsFromDB(db) == nil {
		db = setAliasExprsInDB(db, make(map[string]string))
	}

	for _, computeExpr := range compute.Expressions {
		computeSQL, alias, expr := buildComputeSQLWithDB(dialect, computeExpr, entityMetadata)
		if computeSQL != "" {
			selectColumns = append(selectColumns, computeSQL)
			if alias != "" && expr != "" {
				db = setAliasExprInDB(db, alias, expr)
			}
		}
	}

	if len(selectColumns) > 0 {
		db = db.Select(strings.Join(selectColumns, ", "))
	}

	return db
}

// buildComputeSQLWithDB builds the SQL for a compute expression and returns the alias and expression for registration
func buildComputeSQLWithDB(dialect string, computeExpr ComputeExpression, entityMetadata *metadata.EntityMetadata) (sql, alias, expr string) {
	if computeExpr.Expression == nil {
		return "", "", ""
	}

	expression := computeExpr.Expression

	if expression.Left == nil && expression.Right == nil && expression.Operator != "" && expression.Property != "" {
		prop := findProperty(expression.Property, entityMetadata)
		if prop == nil {
			return "", "", ""
		}

		// Use qualified and quoted column name
		qualified := quoteIdent(dialect, entityMetadata.TableName) + "." + quoteIdent(dialect, prop.ColumnName)
		funcSQL, _ := buildFunctionSQL(dialect, expression.Operator, qualified, nil)
		if funcSQL == "" {
			return "", "", ""
		}

		return fmt.Sprintf("%s as %s", funcSQL, quoteIdent(dialect, computeExpr.Alias)), computeExpr.Alias, funcSQL
	}

	if expression.Left != nil && expression.Right != nil && expression.Logical != "" {
		leftSQL := buildComputeExpressionSQL(dialect, expression.Left, entityMetadata)
		if leftSQL == "" {
			return "", "", ""
		}

		rightSQL := buildComputeExpressionSQL(dialect, expression.Right, entityMetadata)
		if rightSQL == "" {
			return "", "", ""
		}

		var sqlOp string
		switch expression.Logical {
		case "add":
			sqlOp = "+"
		case "sub":
			sqlOp = "-"
		case "mul":
			sqlOp = "*"
		case "div":
			sqlOp = "/"
		case "mod":
			sqlOp = "%"
		default:
			return "", "", ""
		}

		// Build the expression without alias for registration
		exprSQL := fmt.Sprintf("(%s %s %s)", leftSQL, sqlOp, rightSQL)

		return fmt.Sprintf("%s as %s", exprSQL, quoteIdent(dialect, computeExpr.Alias)), computeExpr.Alias, exprSQL
	}

	return "", "", ""
}

// buildComputeExpressionSQL builds SQL for a sub-expression in a compute
func buildComputeExpressionSQL(dialect string, expr *FilterExpression, entityMetadata *metadata.EntityMetadata) string {
	if expr == nil {
		return ""
	}

	if expr.Property != "" && expr.Left == nil && expr.Right == nil {
		prop := findProperty(expr.Property, entityMetadata)
		if prop == nil {
			return ""
		}
		// Return qualified and quoted column
		return quoteIdent(dialect, entityMetadata.TableName) + "." + quoteIdent(dialect, prop.ColumnName)
	}

	if expr.Value != nil && expr.Property == "" && expr.Left == nil && expr.Right == nil {
		switch v := expr.Value.(type) {
		case bool:
			if v {
				return "1"
			}
			return "0"
		case string:
			return fmt.Sprintf("'%s'", v)
		default:
			return fmt.Sprintf("%v", expr.Value)
		}
	}

	if expr.Property != "" && expr.Operator != "" && expr.Operator != OpEqual && expr.Left == nil && expr.Right == nil {
		prop := findProperty(expr.Property, entityMetadata)
		if prop == nil {
			return ""
		}
		// Use qualified and quoted column name
		qualified := quoteIdent(dialect, entityMetadata.TableName) + "." + quoteIdent(dialect, prop.ColumnName)
		funcSQL, _ := buildFunctionSQL(dialect, expr.Operator, qualified, expr.Value)
		return funcSQL
	}

	if expr.Left != nil && expr.Right != nil && expr.Operator != "" {
		leftSQL := buildComputeExpressionSQL(dialect, expr.Left, entityMetadata)
		rightSQL := buildComputeExpressionSQL(dialect, expr.Right, entityMetadata)
		if leftSQL == "" || rightSQL == "" {
			return ""
		}

		var sqlOp string
		switch expr.Operator {
		case OpAdd:
			sqlOp = "+"
		case OpSub:
			sqlOp = "-"
		case OpMul:
			sqlOp = "*"
		case OpDiv:
			sqlOp = "/"
		case OpMod:
			sqlOp = "%"
		default:
			return ""
		}

		return fmt.Sprintf("(%s %s %s)", leftSQL, sqlOp, rightSQL)
	}

	if expr.Left != nil && expr.Right != nil && expr.Logical != "" {
		leftSQL := buildComputeExpressionSQL(dialect, expr.Left, entityMetadata)
		rightSQL := buildComputeExpressionSQL(dialect, expr.Right, entityMetadata)
		if leftSQL == "" || rightSQL == "" {
			return ""
		}

		var sqlOp string
		switch expr.Logical {
		case "add":
			sqlOp = "+"
		case "sub":
			sqlOp = "-"
		case "mul":
			sqlOp = "*"
		case "div":
			sqlOp = "/"
		case "mod":
			sqlOp = "%"
		default:
			return ""
		}

		return fmt.Sprintf("(%s %s %s)", leftSQL, sqlOp, rightSQL)
	}

	return ""
}

// applyOrderBy applies order by clauses to the GORM query
func applyOrderBy(db *gorm.DB, orderBy []OrderByItem, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	dialect := getDatabaseDialect(db)

	// For PostgreSQL, we need to build all ORDER BY expressions in a single Clauses() call
	// to ensure they're all preserved in the final SQL query
	if dialect == "postgres" {
		var orderExprs []clause.OrderByColumn
		for _, item := range orderBy {
			var columnName string
			if propertyExists(item.Property, entityMetadata) {
				col := GetColumnName(item.Property, entityMetadata)
				columnName = quoteIdent(dialect, col)
			} else {
				sanitizedAlias := sanitizeIdentifier(item.Property)
				if sanitizedAlias == "" {
					continue
				}
				columnName = quoteIdent(dialect, sanitizedAlias)
			}

			// Build the ORDER BY expression with NULL handling
			direction := " ASC NULLS FIRST"
			if item.Descending {
				direction = " DESC NULLS LAST"
			}

			orderExprs = append(orderExprs, clause.OrderByColumn{
				Column: clause.Column{Raw: true, Name: columnName + direction},
			})
		}

		if len(orderExprs) > 0 {
			db = db.Clauses(clause.OrderBy{Columns: orderExprs})
		}
	} else {
		// For other databases, use the simple approach
		for _, item := range orderBy {
			var columnName string
			if propertyExists(item.Property, entityMetadata) {
				columnName = GetColumnName(item.Property, entityMetadata)
			} else {
				sanitizedAlias := sanitizeIdentifier(item.Property)
				if sanitizedAlias == "" {
					continue
				}
				columnName = sanitizedAlias
			}

			db = db.Order(clause.OrderByColumn{
				Column: clause.Column{Name: columnName},
				Desc:   item.Descending,
			})
		}
	}

	return db
}
