package query

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// parseFilter parses a filter expression with metadata validation
func parseFilter(filterStr string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	filterStr = strings.TrimSpace(filterStr)

	// Use the new tokenizer and AST parser
	tokenizer := NewTokenizer(filterStr)
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		// Fall back to old parser if tokenization fails
		return parseFilterLegacy(filterStr, entityMetadata)
	}

	parser := NewASTParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		// Fall back to old parser if AST parsing fails
		return parseFilterLegacy(filterStr, entityMetadata)
	}

	return ASTToFilterExpression(ast, entityMetadata)
}

// parseFilterLegacy is the old recursive-descent parser (kept for backward compatibility)
func parseFilterLegacy(filterStr string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	filterStr = strings.TrimSpace(filterStr)

	// Check for logical operators (and, or)
	if idx := findLogicalOperator(filterStr); idx != -1 {
		operator := getLogicalOperatorAt(filterStr, idx)
		left := strings.TrimSpace(filterStr[:idx])
		right := strings.TrimSpace(filterStr[idx+len(operator):])

		leftExpr, err := parseFilterLegacy(left, entityMetadata)
		if err != nil {
			return nil, err
		}

		rightExpr, err := parseFilterLegacy(right, entityMetadata)
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

// parseFilterWithoutMetadata parses a filter expression without metadata validation
func parseFilterWithoutMetadata(filterStr string) (*FilterExpression, error) {
	filterStr = strings.TrimSpace(filterStr)

	// Try using the new AST parser first
	tokenizer := NewTokenizer(filterStr)
	tokens, err := tokenizer.TokenizeAll()
	if err == nil {
		parser := NewASTParser(tokens)
		ast, err := parser.Parse()
		if err == nil {
			// Convert AST to FilterExpression without metadata validation
			return ASTToFilterExpression(ast, nil)
		}
	}

	// Fall back to legacy parser
	return parseFilterWithoutMetadataLegacy(filterStr)
}

// parseFilterWithoutMetadataLegacy is the old parser without metadata validation
func parseFilterWithoutMetadataLegacy(filterStr string) (*FilterExpression, error) {
	filterStr = strings.TrimSpace(filterStr)

	// Check for logical operators (and, or)
	if idx := findLogicalOperator(filterStr); idx != -1 {
		operator := getLogicalOperatorAt(filterStr, idx)
		left := strings.TrimSpace(filterStr[:idx])
		right := strings.TrimSpace(filterStr[idx+len(operator):])

		leftExpr, err := parseFilterWithoutMetadataLegacy(left)
		if err != nil {
			return nil, err
		}

		rightExpr, err := parseFilterWithoutMetadataLegacy(right)
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
		return parseFunctionFilterWithoutMetadata(filterStr)
	}

	// Parse comparison expression (property operator value)
	return parseComparisonFilterWithoutMetadata(filterStr)
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

// parseFunctionFilterWithoutMetadata parses function-based filters without metadata
func parseFunctionFilterWithoutMetadata(filterStr string) (*FilterExpression, error) {
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

// parseComparisonFilterWithoutMetadata parses comparison filters without metadata
func parseComparisonFilterWithoutMetadata(filterStr string) (*FilterExpression, error) {
	// Split by spaces to find operator
	tokens := strings.Fields(filterStr)
	if len(tokens) < 3 {
		return nil, fmt.Errorf("invalid filter expression: %s", filterStr)
	}

	property := tokens[0]
	operator := strings.ToLower(tokens[1])
	value := strings.Join(tokens[2:], " ")

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
