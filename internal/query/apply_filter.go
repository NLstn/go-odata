package query

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// applyFilter applies filter expressions to the GORM query
func applyFilter(db *gorm.DB, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if filter == nil {
		return db
	}

	if filter.Logical != "" {
		query, args := buildFilterCondition(filter, entityMetadata)
		return db.Where(query, args...)
	}

	query, args := buildFilterCondition(filter, entityMetadata)
	return db.Where(query, args...)
}

// applyHavingFilter applies a HAVING clause filter for post-aggregation filtering
// This is used when a filter transformation comes after a groupby/aggregate in $apply
func applyHavingFilter(db *gorm.DB, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if filter == nil {
		return db
	}

	query, args := buildFilterCondition(filter, entityMetadata)
	return db.Having(query, args...)
}

// buildFilterCondition builds a WHERE condition string and arguments for a filter expression
func buildFilterCondition(filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	if filter.Logical != "" {
		return buildLogicalCondition(filter, entityMetadata)
	}

	query, args := buildComparisonCondition(filter, entityMetadata)

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
	if filter.Operator == OpAny || filter.Operator == OpAll {
		return buildLambdaCondition(filter, entityMetadata)
	}

	if filter.Left != nil && filter.Left.Operator != "" {
		return buildFunctionComparison(filter, entityMetadata)
	}

	var columnName string
	if propertyExists(filter.Property, entityMetadata) {
		columnName = GetColumnName(filter.Property, entityMetadata)
	} else {
		rawName := sanitizeIdentifier(filter.Property)
		if rawName == "" {
			return "", nil
		}
		if rawName == "$it" {
			columnName = rawName
		} else if rawName == "$count" {
			columnName = "`" + rawName + "`"
		} else if len(rawName) > 0 && rawName[0] == '$' {
			columnName = "`" + rawName + "`"
		} else {
			columnName = rawName
		}
	}

	if funcCall, ok := filter.Value.(*FunctionCallExpr); ok {
		rightExpr, err := convertFunctionCallExpr(funcCall, entityMetadata)
		if err == nil && rightExpr != nil {
			rightColumnName := ""
			if rightExpr.Property != "" {
				rightColumnName = GetColumnName(rightExpr.Property, entityMetadata)
			}
			rightSQL, rightArgs := buildFunctionSQL(rightExpr.Operator, rightColumnName, rightExpr.Value)
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
		funcSQL, funcArgs := buildFunctionSQL(OpIsOf, columnName, filter.Value)
		if funcSQL == "" {
			return "", nil
		}
		return fmt.Sprintf("%s = ?", funcSQL), append(funcArgs, true)
	case OpGeoIntersects:
		funcSQL, funcArgs := buildFunctionSQL(OpGeoIntersects, columnName, filter.Value)
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
func buildLambdaCondition(filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	navProp := findNavigationProperty(filter.Property, entityMetadata)
	if navProp == nil {
		return "", nil
	}

	relatedEntityName := navProp.NavigationTarget
	if relatedEntityName == "" {
		relatedEntityName = navProp.JsonName
	}
	relatedTableName := toSnakeCase(pluralize(relatedEntityName))

	parentTableName := toSnakeCase(pluralize(entityMetadata.EntityName))

	foreignKeyColumn := toSnakeCase(entityMetadata.EntityName) + "_id"

	parentPrimaryKey := "id"
	if len(entityMetadata.KeyProperties) > 0 {
		parentPrimaryKey = toSnakeCase(entityMetadata.KeyProperties[0].Name)
	}

	if filter.Value == nil || filter.Left == nil {
		if filter.Operator == OpAny {
			return fmt.Sprintf("EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s)",
				relatedTableName, relatedTableName, foreignKeyColumn, parentTableName, parentPrimaryKey), []interface{}{}
		}
		return fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s) OR EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s)",
			relatedTableName, relatedTableName, foreignKeyColumn, parentTableName, parentPrimaryKey,
			relatedTableName, relatedTableName, foreignKeyColumn, parentTableName, parentPrimaryKey), []interface{}{}
	}

	predicate := filter.Left
	if predicate == nil {
		return "", nil
	}

	predicateSQL, predicateArgs := buildFilterConditionForLambda(predicate, navProp)
	if predicateSQL == "" {
		return "", nil
	}

	var sql string
	if filter.Operator == OpAny {
		sql = fmt.Sprintf("EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s AND %s)",
			relatedTableName, relatedTableName, foreignKeyColumn, parentTableName, parentPrimaryKey, predicateSQL)
	} else {
		sql = fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.%s AND NOT (%s))",
			relatedTableName, relatedTableName, foreignKeyColumn, parentTableName, parentPrimaryKey, predicateSQL)
	}

	return sql, predicateArgs
}

// buildFilterConditionForLambda builds a filter condition for lambda subquery predicates
func buildFilterConditionForLambda(filter *FilterExpression, navProp *metadata.PropertyMetadata) (string, []interface{}) {
	if filter == nil {
		return "", nil
	}

	if filter.Logical != "" {
		return buildLogicalConditionForLambda(filter, navProp)
	}

	if filter.Left != nil && filter.Left.Operator != "" {
		return buildFunctionComparisonForLambda(filter, navProp)
	}

	columnName := toSnakeCase(filter.Property)

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
func buildLogicalConditionForLambda(filter *FilterExpression, navProp *metadata.PropertyMetadata) (string, []interface{}) {
	leftQuery, leftArgs := buildFilterConditionForLambda(filter.Left, navProp)
	rightQuery, rightArgs := buildFilterConditionForLambda(filter.Right, navProp)

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
func buildFunctionComparisonForLambda(filter *FilterExpression, _ *metadata.PropertyMetadata) (string, []interface{}) {
	funcExpr := filter.Left
	columnName := toSnakeCase(funcExpr.Property)

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

// buildFunctionComparison builds a comparison with a function on the left side
func buildFunctionComparison(filter *FilterExpression, entityMetadata *metadata.EntityMetadata) (string, []interface{}) {
	funcExpr := filter.Left
	columnName := GetColumnName(funcExpr.Property, entityMetadata)

	funcSQL, funcArgs := buildFunctionSQL(funcExpr.Operator, columnName, funcExpr.Value)
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

		rightColumnName := GetColumnName(rightFuncExpr.Property, entityMetadata)
		rightFuncSQL, rightFuncArgs := buildFunctionSQL(rightFuncExpr.Operator, rightColumnName, rightFuncExpr.Value)
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

	compSQL := buildComparisonSQL(filter.Operator, funcSQL)
	if compSQL == "" {
		return "", nil
	}

	allArgs := append(funcArgs, filter.Value)
	return compSQL, allArgs
}

// buildFunctionSQL generates SQL for a function expression
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
		return fmt.Sprintf("INSTR(%s, ?) - 1", columnName), []interface{}{value}
	case OpConcat:
		if args, ok := value.([]interface{}); ok && len(args) == 2 {
			firstArg := args[0]
			secondArg := args[1]

			var leftSQL string
			var leftArgs []interface{}

			if funcCall, ok := firstArg.(*FunctionCallExpr); ok {
				leftExpr, err := convertFunctionCallExpr(funcCall, nil)
				if err == nil {
					leftSQL, leftArgs = buildFunctionSQL(leftExpr.Operator, leftExpr.Property, leftExpr.Value)
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
					rightSQL, rightArgs = buildFunctionSQL(rightExpr.Operator, rightExpr.Property, rightExpr.Value)
				}
				if rightSQL == "" {
					rightSQL = "?"
					rightArgs = []interface{}{secondArg}
				}
			} else if strVal, ok := secondArg.(string); ok {
				if len(strVal) > 0 && (strVal[0] >= 'A' && strVal[0] <= 'Z' || strings.Contains(strVal, "_")) {
					rightSQL = toSnakeCase(strVal)
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
			return fmt.Sprintf("CONCAT(%s, %s)", leftSQL, rightSQL), allArgs
		}

		if funcCall, ok := value.(*FunctionCallExpr); ok {
			rightExpr, err := convertFunctionCallExpr(funcCall, nil)
			if err != nil {
				return fmt.Sprintf("CONCAT(%s, ?)", columnName), []interface{}{value}
			}
			rightColumnName := rightExpr.Property
			rightSQL, rightArgs := buildFunctionSQL(rightExpr.Operator, rightColumnName, rightExpr.Value)
			if rightSQL == "" {
				return fmt.Sprintf("CONCAT(%s, ?)", columnName), []interface{}{value}
			}
			return fmt.Sprintf("CONCAT(%s, %s)", columnName, rightSQL), rightArgs
		}
		if strVal, ok := value.(string); ok {
			if len(strVal) > 0 && (strVal[0] >= 'A' && strVal[0] <= 'Z' || strings.Contains(strVal, "_")) {
				columnName2 := toSnakeCase(strVal)
				return fmt.Sprintf("CONCAT(%s, %s)", columnName, columnName2), nil
			}
		}
		return fmt.Sprintf("CONCAT(%s, ?)", columnName), []interface{}{value}
	case OpHas:
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
	case OpNow:
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
			sqlType := edmTypeToSQLType(typeName)
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

			sqlType := edmTypeToSQLType(typeName)
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

// buildSubstringSQL builds SQL for substring function
func buildSubstringSQL(columnName string, value interface{}) (string, []interface{}) {
	args, ok := value.([]interface{})
	if !ok {
		return "", nil
	}
	if len(args) == 1 {
		return fmt.Sprintf("SUBSTR(%s, ? + 1, LENGTH(%s))", columnName, columnName), []interface{}{args[0]}
	}
	if len(args) == 2 {
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
