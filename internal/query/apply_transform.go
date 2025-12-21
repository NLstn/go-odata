package query

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// applyTransformations applies apply transformations to the GORM query
func applyTransformations(db *gorm.DB, transformations []ApplyTransformation, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if db.Statement != nil && db.Statement.Dest == nil {
		modelInstance := reflect.New(entityMetadata.EntityType).Interface()
		db = db.Model(modelInstance)
	}

	dialect := getDatabaseDialect(db)

	hasGrouping := false

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

	for _, propName := range groupBy.Properties {
		prop := findProperty(propName, entityMetadata)
		if prop == nil {
			continue
		}

		columnName := GetColumnName(propName, entityMetadata)
		groupByColumns = append(groupByColumns, columnName)
		selectColumns = append(selectColumns, fmt.Sprintf("%s as %s", columnName, prop.JsonName))
	}

	if len(groupBy.Transform) > 0 {
		for _, trans := range groupBy.Transform {
			if trans.Type == ApplyTypeAggregate && trans.Aggregate != nil {
				for _, aggExpr := range trans.Aggregate.Expressions {
					aggSQL := buildAggregateSQL(aggExpr, entityMetadata)
					if aggSQL != "" {
						selectColumns = append(selectColumns, aggSQL)
					}
				}
			}
		}
	} else {
		selectColumns = append(selectColumns, "COUNT(*) as \"$count\"")
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
	if aggExpr.Property == "$count" {
		return fmt.Sprintf("COUNT(*) as \"%s\"", aggExpr.Alias)
	}

	prop := findProperty(aggExpr.Property, entityMetadata)
	if prop == nil {
		return ""
	}

	columnName := GetColumnName(aggExpr.Property, entityMetadata)

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
		return fmt.Sprintf("COUNT(DISTINCT %s) as \"%s\"", columnName, aggExpr.Alias)
	default:
		return ""
	}

	return fmt.Sprintf("%s(%s) as \"%s\"", sqlFunc, columnName, aggExpr.Alias)
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
			columnName := toSnakeCase(prop.Name)
			selectColumns = append(selectColumns, fmt.Sprintf("%s as %s", columnName, prop.JsonName))
		}
	}

	for _, computeExpr := range compute.Expressions {
		computeSQL := buildComputeSQL(dialect, computeExpr, entityMetadata)
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
func buildComputeSQL(dialect string, computeExpr ComputeExpression, entityMetadata *metadata.EntityMetadata) string {
	if computeExpr.Expression == nil {
		return ""
	}

	expr := computeExpr.Expression

	if expr.Left == nil && expr.Right == nil && expr.Operator != "" && expr.Property != "" {
		prop := findProperty(expr.Property, entityMetadata)
		if prop == nil {
			return ""
		}

		columnName := toSnakeCase(prop.Name)
		funcSQL, _ := buildFunctionSQL(dialect, expr.Operator, columnName, nil)
		if funcSQL == "" {
			return ""
		}

		return fmt.Sprintf("%s as %s", funcSQL, computeExpr.Alias)
	}

	if expr.Left != nil && expr.Right != nil && expr.Logical != "" {
		leftSQL := buildComputeExpressionSQL(dialect, expr.Left, entityMetadata)
		if leftSQL == "" {
			return ""
		}

		rightSQL := buildComputeExpressionSQL(dialect, expr.Right, entityMetadata)
		if rightSQL == "" {
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

		return fmt.Sprintf("(%s %s %s) as %s", leftSQL, sqlOp, rightSQL, computeExpr.Alias)
	}

	return ""
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
		return toSnakeCase(prop.Name)
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
		columnName := toSnakeCase(prop.Name)
		funcSQL, _ := buildFunctionSQL(dialect, expr.Operator, columnName, expr.Value)
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
	for _, item := range orderBy {
		if propertyExists(item.Property, entityMetadata) {
			columnName := GetColumnName(item.Property, entityMetadata)
			db = db.Order(clause.OrderByColumn{
				Column: clause.Column{Name: columnName},
				Desc:   item.Descending,
			})
		} else {
			sanitizedAlias := sanitizeIdentifier(item.Property)
			if sanitizedAlias == "" {
				continue
			}
			db = db.Order(clause.OrderByColumn{
				Column: clause.Column{Name: sanitizedAlias},
				Desc:   item.Descending,
			})
		}
	}
	return db
}
