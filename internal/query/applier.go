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

	// Apply filter
	if options.Filter != nil {
		db = applyFilter(db, options.Filter, entityMetadata)
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

// applyFilter applies filter expressions to the GORM query
func applyFilter(db *gorm.DB, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if filter == nil {
		return db
	}

	// Handle logical operators (and, or)
	if filter.Logical != "" {
		leftDB := applyFilter(db, filter.Left, entityMetadata)
		rightDB := applyFilter(db, filter.Right, entityMetadata)

		if filter.Logical == LogicalAnd {
			// For AND, we can chain the conditions
			return leftDB.Where(rightDB)
		} else if filter.Logical == LogicalOr {
			// For OR, we need to build a more complex query
			// This is simplified and may need enhancement for complex cases
			leftQuery, leftArgs := buildFilterCondition(filter.Left, entityMetadata)
			rightQuery, rightArgs := buildFilterCondition(filter.Right, entityMetadata)

			combinedQuery := fmt.Sprintf("(%s) OR (%s)", leftQuery, rightQuery)
			combinedArgs := append(leftArgs, rightArgs...)

			return db.Where(combinedQuery, combinedArgs...)
		}
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
		leftQuery, leftArgs := buildFilterCondition(filter.Left, entityMetadata)
		rightQuery, rightArgs := buildFilterCondition(filter.Right, entityMetadata)

		if filter.Logical == LogicalAnd {
			query := fmt.Sprintf("(%s) AND (%s)", leftQuery, rightQuery)
			args := append(leftArgs, rightArgs...)
			return query, args
		} else if filter.Logical == LogicalOr {
			query := fmt.Sprintf("(%s) OR (%s)", leftQuery, rightQuery)
			args := append(leftArgs, rightArgs...)
			return query, args
		}
	}

	fieldName := GetPropertyFieldName(filter.Property, entityMetadata)

	switch filter.Operator {
	case OpEqual:
		return fmt.Sprintf("%s = ?", fieldName), []interface{}{filter.Value}
	case OpNotEqual:
		return fmt.Sprintf("%s != ?", fieldName), []interface{}{filter.Value}
	case OpGreaterThan:
		return fmt.Sprintf("%s > ?", fieldName), []interface{}{filter.Value}
	case OpGreaterThanOrEqual:
		return fmt.Sprintf("%s >= ?", fieldName), []interface{}{filter.Value}
	case OpLessThan:
		return fmt.Sprintf("%s < ?", fieldName), []interface{}{filter.Value}
	case OpLessThanOrEqual:
		return fmt.Sprintf("%s <= ?", fieldName), []interface{}{filter.Value}
	case OpContains:
		return fmt.Sprintf("%s LIKE ?", fieldName), []interface{}{"%" + fmt.Sprint(filter.Value) + "%"}
	case OpStartsWith:
		return fmt.Sprintf("%s LIKE ?", fieldName), []interface{}{fmt.Sprint(filter.Value) + "%"}
	case OpEndsWith:
		return fmt.Sprintf("%s LIKE ?", fieldName), []interface{}{"%" + fmt.Sprint(filter.Value)}
	default:
		return "", nil
	}
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

// ApplySelect filters the result to only include selected properties
// This is done after the query by modifying the result slice
func ApplySelect(results interface{}, selectedProperties []string, entityMetadata *metadata.EntityMetadata) interface{} {
	if len(selectedProperties) == 0 {
		return results
	}

	// Get the slice value
	sliceValue := reflect.ValueOf(results)
	if sliceValue.Kind() != reflect.Slice {
		return results
	}

	// Create a new slice of maps to hold the filtered results
	filteredResults := make([]map[string]interface{}, sliceValue.Len())

	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i)
		filteredItem := make(map[string]interface{})

		// Extract only the selected properties
		for _, propName := range selectedProperties {
			propName = strings.TrimSpace(propName)

			// Find the property in metadata
			for _, prop := range entityMetadata.Properties {
				if prop.JsonName == propName || prop.Name == propName {
					// Get the field value
					fieldValue := item.FieldByName(prop.Name)
					if fieldValue.IsValid() && fieldValue.CanInterface() {
						filteredItem[prop.JsonName] = fieldValue.Interface()
					}
					break
				}
			}
		}

		filteredResults[i] = filteredItem
	}

	return filteredResults
}
