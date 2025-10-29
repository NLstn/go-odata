package query

import (
	"fmt"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// applyExpand applies expand (preload) options to the GORM query
func applyExpand(db *gorm.DB, expand []ExpandOption, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	for _, expandOpt := range expand {
		navProp := findNavigationProperty(expandOpt.NavigationProperty, entityMetadata)
		if navProp == nil {
			continue
		}

		if expandOpt.Select != nil || expandOpt.Filter != nil || expandOpt.OrderBy != nil ||
			expandOpt.Top != nil || expandOpt.Skip != nil {
			db = db.Preload(navProp.Name, func(db *gorm.DB) *gorm.DB {
				if expandOpt.Filter != nil {
					db = applyFilterForExpand(db, expandOpt.Filter)
				}

				if len(expandOpt.OrderBy) > 0 {
					for _, item := range expandOpt.OrderBy {
						direction := "ASC"
						if item.Descending {
							direction = "DESC"
						}
						columnName := toSnakeCase(item.Property)
						db = db.Order(fmt.Sprintf("%s %s", columnName, direction))
					}
				}

				if expandOpt.Skip != nil {
					db = db.Offset(*expandOpt.Skip)
				}
				if expandOpt.Top != nil {
					db = db.Limit(*expandOpt.Top)
				}

				return db
			})
		} else {
			db = db.Preload(navProp.Name)
		}
	}
	return db
}

// applyFilterForExpand applies filter to expanded navigation property without metadata context
func applyFilterForExpand(db *gorm.DB, filter *FilterExpression) *gorm.DB {
	if filter == nil {
		return db
	}

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

	if filter.Left != nil && filter.Left.Operator != "" {
		return buildSimpleFunctionComparison(filter)
	}

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
