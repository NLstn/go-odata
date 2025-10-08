package query

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// QueryOptions represents parsed OData query options
type QueryOptions struct {
	Filter  *FilterExpression
	Select  []string
	OrderBy []OrderByItem
	Top     *int
	Skip    *int
}

// OrderByItem represents a single orderby clause
type OrderByItem struct {
	Property   string
	Descending bool
}

// FilterExpression represents a parsed filter expression
type FilterExpression struct {
	Property string
	Operator FilterOperator
	Value    interface{}
	Left     *FilterExpression
	Right    *FilterExpression
	Logical  LogicalOperator
}

// FilterOperator represents filter comparison operators
type FilterOperator string

const (
	OpEqual              FilterOperator = "eq"
	OpNotEqual           FilterOperator = "ne"
	OpGreaterThan        FilterOperator = "gt"
	OpGreaterThanOrEqual FilterOperator = "ge"
	OpLessThan           FilterOperator = "lt"
	OpLessThanOrEqual    FilterOperator = "le"
	OpContains           FilterOperator = "contains"
	OpStartsWith         FilterOperator = "startswith"
	OpEndsWith           FilterOperator = "endswith"
)

// LogicalOperator represents logical operators for combining filters
type LogicalOperator string

const (
	LogicalAnd LogicalOperator = "and"
	LogicalOr  LogicalOperator = "or"
)

// ParseQueryOptions parses OData query options from the URL
func ParseQueryOptions(queryParams url.Values, entityMetadata *metadata.EntityMetadata) (*QueryOptions, error) {
	options := &QueryOptions{}

	// Parse $filter
	if filterStr := queryParams.Get("$filter"); filterStr != "" {
		filter, err := parseFilter(filterStr, entityMetadata)
		if err != nil {
			return nil, fmt.Errorf("invalid $filter: %w", err)
		}
		options.Filter = filter
	}

	// Parse $select
	if selectStr := queryParams.Get("$select"); selectStr != "" {
		options.Select = parseSelect(selectStr)
	}

	// Parse $orderby
	if orderByStr := queryParams.Get("$orderby"); orderByStr != "" {
		orderBy, err := parseOrderBy(orderByStr, entityMetadata)
		if err != nil {
			return nil, fmt.Errorf("invalid $orderby: %w", err)
		}
		options.OrderBy = orderBy
	}

	return options, nil
}

// parseSelect parses the $select query option
func parseSelect(selectStr string) []string {
	parts := strings.Split(selectStr, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseOrderBy parses the $orderby query option
func parseOrderBy(orderByStr string, entityMetadata *metadata.EntityMetadata) ([]OrderByItem, error) {
	parts := strings.Split(orderByStr, ",")
	result := make([]OrderByItem, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		// Check for "desc" or "asc" suffix
		tokens := strings.Fields(trimmed)
		item := OrderByItem{
			Property:   tokens[0],
			Descending: false,
		}

		if len(tokens) > 1 {
			direction := strings.ToLower(tokens[1])
			if direction == "desc" {
				item.Descending = true
			} else if direction != "asc" {
				return nil, fmt.Errorf("invalid direction '%s', expected 'asc' or 'desc'", tokens[1])
			}
		}

		// Validate property exists
		if !propertyExists(item.Property, entityMetadata) {
			return nil, fmt.Errorf("property '%s' does not exist", item.Property)
		}

		result = append(result, item)
	}

	return result, nil
}

// parseFilter parses a filter expression
// This is a simplified parser that handles basic filter expressions
func parseFilter(filterStr string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	filterStr = strings.TrimSpace(filterStr)

	// Check for logical operators (and, or)
	if idx := findLogicalOperator(filterStr); idx != -1 {
		operator := getLogicalOperatorAt(filterStr, idx)
		left := strings.TrimSpace(filterStr[:idx])
		right := strings.TrimSpace(filterStr[idx+len(operator):])

		leftExpr, err := parseFilter(left, entityMetadata)
		if err != nil {
			return nil, err
		}

		rightExpr, err := parseFilter(right, entityMetadata)
		if err != nil {
			return nil, err
		}

		return &FilterExpression{
			Left:    leftExpr,
			Right:   rightExpr,
			Logical: LogicalOperator(operator),
		}, nil
	}

	// Check for function calls (contains, startswith, endswith)
	if strings.Contains(filterStr, "(") && strings.Contains(filterStr, ")") {
		return parseFunctionFilter(filterStr, entityMetadata)
	}

	// Parse comparison expression (property operator value)
	return parseComparisonFilter(filterStr, entityMetadata)
}

// findLogicalOperator finds the position of a logical operator (and/or) in the filter string
// This is a simple implementation that doesn't handle nested parentheses properly
func findLogicalOperator(filterStr string) int {
	// Look for " and " or " or " (with spaces to avoid matching parts of property names)
	andIdx := strings.Index(strings.ToLower(filterStr), " and ")
	orIdx := strings.Index(strings.ToLower(filterStr), " or ")

	if andIdx == -1 {
		return orIdx
	}
	if orIdx == -1 {
		return andIdx
	}

	// Return the first occurrence
	if andIdx < orIdx {
		return andIdx
	}
	return orIdx
}

// getLogicalOperatorAt returns the logical operator at the given position
func getLogicalOperatorAt(filterStr string, idx int) string {
	lower := strings.ToLower(filterStr)
	if strings.HasPrefix(lower[idx:], " and ") {
		return " and "
	}
	if strings.HasPrefix(lower[idx:], " or ") {
		return " or "
	}
	return ""
}

// parseFunctionFilter parses function-based filters like contains(Name,'laptop')
func parseFunctionFilter(filterStr string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	// Extract function name
	openParen := strings.Index(filterStr, "(")
	if openParen == -1 {
		return nil, fmt.Errorf("invalid function filter: %s", filterStr)
	}

	functionName := strings.ToLower(strings.TrimSpace(filterStr[:openParen]))
	closeParen := strings.LastIndex(filterStr, ")")
	if closeParen == -1 {
		return nil, fmt.Errorf("invalid function filter: missing closing parenthesis")
	}

	args := filterStr[openParen+1 : closeParen]
	parts := splitFunctionArgs(args)

	if len(parts) != 2 {
		return nil, fmt.Errorf("function %s requires 2 arguments", functionName)
	}

	property := strings.TrimSpace(parts[0])
	value := strings.Trim(strings.TrimSpace(parts[1]), "'\"")

	// Validate property exists
	if !propertyExists(property, entityMetadata) {
		return nil, fmt.Errorf("property '%s' does not exist", property)
	}

	var operator FilterOperator
	switch functionName {
	case "contains":
		operator = OpContains
	case "startswith":
		operator = OpStartsWith
	case "endswith":
		operator = OpEndsWith
	default:
		return nil, fmt.Errorf("unsupported function: %s", functionName)
	}

	return &FilterExpression{
		Property: property,
		Operator: operator,
		Value:    value,
	}, nil
}

// splitFunctionArgs splits function arguments by comma
func splitFunctionArgs(args string) []string {
	result := make([]string, 0)
	current := ""
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range args {
		if ch == '\'' || ch == '"' {
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuotes = false
				quoteChar = 0
			}
			current += string(ch)
		} else if ch == ',' && !inQuotes {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

// parseComparisonFilter parses comparison filters like "Price gt 100"
func parseComparisonFilter(filterStr string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	// Split by spaces to find operator
	tokens := strings.Fields(filterStr)
	if len(tokens) < 3 {
		return nil, fmt.Errorf("invalid filter expression: %s", filterStr)
	}

	property := tokens[0]
	operator := strings.ToLower(tokens[1])
	value := strings.Join(tokens[2:], " ")

	// Validate property exists
	if !propertyExists(property, entityMetadata) {
		return nil, fmt.Errorf("property '%s' does not exist", property)
	}

	// Remove quotes from string values
	value = strings.Trim(value, "'\"")

	// Validate operator
	var op FilterOperator
	switch operator {
	case "eq":
		op = OpEqual
	case "ne":
		op = OpNotEqual
	case "gt":
		op = OpGreaterThan
	case "ge":
		op = OpGreaterThanOrEqual
	case "lt":
		op = OpLessThan
	case "le":
		op = OpLessThanOrEqual
	default:
		return nil, fmt.Errorf("unsupported operator: %s", operator)
	}

	return &FilterExpression{
		Property: property,
		Operator: op,
		Value:    value,
	}, nil
}

// propertyExists checks if a property exists in the entity metadata
func propertyExists(propertyName string, entityMetadata *metadata.EntityMetadata) bool {
	for _, prop := range entityMetadata.Properties {
		if prop.JsonName == propertyName || prop.Name == propertyName {
			return true
		}
	}
	return false
}

// GetPropertyFieldName returns the struct field name for a given JSON property name
func GetPropertyFieldName(propertyName string, entityMetadata *metadata.EntityMetadata) string {
	for _, prop := range entityMetadata.Properties {
		if prop.JsonName == propertyName || prop.Name == propertyName {
			return prop.JsonName
		}
	}
	return propertyName
}
