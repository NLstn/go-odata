package query

import "fmt"

// buildSimpleOperatorCondition builds SQL for a simple operator condition.
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
