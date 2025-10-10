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
	Expand  []ExpandOption
	OrderBy []OrderByItem
	Top     *int
	Skip    *int
	Count   bool
}

// ExpandOption represents a single $expand clause
type ExpandOption struct {
	NavigationProperty string
	Select             []string          // Nested $select
	Filter             *FilterExpression // Nested $filter
	OrderBy            []OrderByItem     // Nested $orderby
	Top                *int              // Nested $top
	Skip               *int              // Nested $skip
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
	IsNot    bool // Indicates if this is a NOT expression
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

	// Parse each query option
	if err := parseFilterOption(queryParams, entityMetadata, options); err != nil {
		return nil, err
	}

	parseSelectOption(queryParams, options)

	if err := parseExpandOption(queryParams, entityMetadata, options); err != nil {
		return nil, err
	}

	if err := parseOrderByOption(queryParams, entityMetadata, options); err != nil {
		return nil, err
	}

	if err := parseTopOption(queryParams, options); err != nil {
		return nil, err
	}

	if err := parseSkipOption(queryParams, options); err != nil {
		return nil, err
	}

	if err := parseCountOption(queryParams, options); err != nil {
		return nil, err
	}

	return options, nil
}

// parseFilterOption parses the $filter query parameter
func parseFilterOption(queryParams url.Values, entityMetadata *metadata.EntityMetadata, options *QueryOptions) error {
	if filterStr := queryParams.Get("$filter"); filterStr != "" {
		filter, err := parseFilter(filterStr, entityMetadata)
		if err != nil {
			return fmt.Errorf("invalid $filter: %w", err)
		}
		options.Filter = filter
	}
	return nil
}

// parseSelectOption parses the $select query parameter
func parseSelectOption(queryParams url.Values, options *QueryOptions) {
	if selectStr := queryParams.Get("$select"); selectStr != "" {
		options.Select = parseSelect(selectStr)
	}
}

// parseExpandOption parses the $expand query parameter
func parseExpandOption(queryParams url.Values, entityMetadata *metadata.EntityMetadata, options *QueryOptions) error {
	if expandStr := queryParams.Get("$expand"); expandStr != "" {
		expand, err := parseExpand(expandStr, entityMetadata)
		if err != nil {
			return fmt.Errorf("invalid $expand: %w", err)
		}
		options.Expand = expand
	}
	return nil
}

// parseOrderByOption parses the $orderby query parameter
func parseOrderByOption(queryParams url.Values, entityMetadata *metadata.EntityMetadata, options *QueryOptions) error {
	if orderByStr := queryParams.Get("$orderby"); orderByStr != "" {
		orderBy, err := parseOrderBy(orderByStr, entityMetadata)
		if err != nil {
			return fmt.Errorf("invalid $orderby: %w", err)
		}
		options.OrderBy = orderBy
	}
	return nil
}

// parseTopOption parses the $top query parameter
func parseTopOption(queryParams url.Values, options *QueryOptions) error {
	if topStr := queryParams.Get("$top"); topStr != "" {
		top, err := parseNonNegativeInt(topStr, "$top")
		if err != nil {
			return err
		}
		options.Top = &top
	}
	return nil
}

// parseSkipOption parses the $skip query parameter
func parseSkipOption(queryParams url.Values, options *QueryOptions) error {
	if skipStr := queryParams.Get("$skip"); skipStr != "" {
		skip, err := parseNonNegativeInt(skipStr, "$skip")
		if err != nil {
			return err
		}
		options.Skip = &skip
	}
	return nil
}

// parseCountOption parses the $count query parameter
func parseCountOption(queryParams url.Values, options *QueryOptions) error {
	if countStr := queryParams.Get("$count"); countStr != "" {
		countLower := strings.ToLower(countStr)
		if countLower == "true" {
			options.Count = true
		} else if countLower != "false" {
			return fmt.Errorf("invalid $count: must be 'true' or 'false'")
		}
	}
	return nil
}

// parseNonNegativeInt parses a string as a non-negative integer
func parseNonNegativeInt(str, paramName string) (int, error) {
	var value int
	if _, err := fmt.Sscanf(str, "%d", &value); err != nil {
		return 0, fmt.Errorf("invalid %s: must be a non-negative integer", paramName)
	}
	if value < 0 {
		return 0, fmt.Errorf("invalid %s: must be a non-negative integer", paramName)
	}
	return value, nil
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

// parseExpand parses the $expand query option
func parseExpand(expandStr string, entityMetadata *metadata.EntityMetadata) ([]ExpandOption, error) {
	// Split by comma for multiple expands (basic implementation, doesn't handle nested parens)
	parts := splitExpandParts(expandStr)
	result := make([]ExpandOption, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		expand, err := parseSingleExpand(trimmed, entityMetadata)
		if err != nil {
			return nil, err
		}
		result = append(result, expand)
	}

	return result, nil
}

// splitExpandParts splits expand string by comma, handling nested parentheses
func splitExpandParts(expandStr string) []string {
	result := make([]string, 0)
	current := ""
	depth := 0

	for _, ch := range expandStr {
		if ch == '(' {
			depth++
			current += string(ch)
		} else if ch == ')' {
			depth--
			current += string(ch)
		} else if ch == ',' && depth == 0 {
			if current != "" {
				result = append(result, current)
			}
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

// parseSingleExpand parses a single expand option with potential nested query options
func parseSingleExpand(expandStr string, entityMetadata *metadata.EntityMetadata) (ExpandOption, error) {
	expand := ExpandOption{}

	// Check for nested query options: NavigationProp($select=...,...)
	if idx := strings.Index(expandStr, "("); idx != -1 {
		if !strings.HasSuffix(expandStr, ")") {
			return expand, fmt.Errorf("invalid expand syntax: %s", expandStr)
		}

		expand.NavigationProperty = strings.TrimSpace(expandStr[:idx])
		nestedOptions := expandStr[idx+1 : len(expandStr)-1]

		// Parse nested options (simplified - doesn't handle complex nested cases)
		if err := parseNestedExpandOptions(&expand, nestedOptions, entityMetadata); err != nil {
			return expand, err
		}
	} else {
		expand.NavigationProperty = strings.TrimSpace(expandStr)
	}

	// Validate navigation property exists
	if !isNavigationProperty(expand.NavigationProperty, entityMetadata) {
		return expand, fmt.Errorf("'%s' is not a valid navigation property", expand.NavigationProperty)
	}

	return expand, nil
}

// parseNestedExpandOptions parses nested query options within an expand
func parseNestedExpandOptions(expand *ExpandOption, optionsStr string, entityMetadata *metadata.EntityMetadata) error {
	// Split by semicolon for different query options
	parts := strings.Split(optionsStr, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Find the equals sign
		eqIdx := strings.Index(part, "=")
		if eqIdx == -1 {
			continue
		}

		key := strings.TrimSpace(part[:eqIdx])
		value := strings.TrimSpace(part[eqIdx+1:])

		switch strings.ToLower(key) {
		case "$select":
			expand.Select = parseSelect(value)
		case "$filter":
			// Get the navigation property metadata to find the target entity type
			navProp := findNavigationProperty(expand.NavigationProperty, entityMetadata)
			if navProp == nil {
				return fmt.Errorf("navigation property '%s' not found", expand.NavigationProperty)
			}

			// Create a temporary entity metadata for the target type
			// We need to find the target entity type in the metadata
			// For now, we'll parse without metadata validation
			filter, err := parseFilterWithoutMetadata(value)
			if err != nil {
				return fmt.Errorf("invalid nested $filter: %w", err)
			}
			expand.Filter = filter
		case "$orderby":
			// Parse orderby without strict metadata validation
			orderBy, err := parseOrderByWithoutMetadata(value)
			if err != nil {
				return fmt.Errorf("invalid nested $orderby: %w", err)
			}
			expand.OrderBy = orderBy
		case "$top":
			var top int
			if _, err := fmt.Sscanf(value, "%d", &top); err != nil {
				return fmt.Errorf("invalid nested $top: %w", err)
			}
			expand.Top = &top
		case "$skip":
			var skip int
			if _, err := fmt.Sscanf(value, "%d", &skip); err != nil {
				return fmt.Errorf("invalid nested $skip: %w", err)
			}
			expand.Skip = &skip
		}
	}

	return nil
}

// parseFilterWithoutMetadata parses a filter expression without validating property names
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

// parseOrderByWithoutMetadata parses orderby without metadata validation
func parseOrderByWithoutMetadata(orderByStr string) ([]OrderByItem, error) {
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

		result = append(result, item)
	}

	return result, nil
}

// isNavigationProperty checks if a property is a navigation property
func isNavigationProperty(propName string, entityMetadata *metadata.EntityMetadata) bool {
	for _, prop := range entityMetadata.Properties {
		if (prop.JsonName == propName || prop.Name == propName) && prop.IsNavigationProp {
			return true
		}
	}
	return false
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

// parseFilter parses a filter expression using the new AST-based parser
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
// This returns the actual Go struct field name, not the JSON name
func GetPropertyFieldName(propertyName string, entityMetadata *metadata.EntityMetadata) string {
	for _, prop := range entityMetadata.Properties {
		if prop.JsonName == propertyName || prop.Name == propertyName {
			return prop.Name // Return the struct field name
		}
	}
	return propertyName
}

// GetColumnName returns the database column name (snake_case) for a property
func GetColumnName(propertyName string, entityMetadata *metadata.EntityMetadata) string {
	for _, prop := range entityMetadata.Properties {
		if prop.JsonName == propertyName || prop.Name == propertyName {
			// Check if there's a GORM tag specifying the column name
			if prop.GormTag != "" {
				// Parse GORM tag for column name
				parts := strings.Split(prop.GormTag, ";")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if strings.HasPrefix(part, "column:") {
						return strings.TrimPrefix(part, "column:")
					}
				}
			}
			// Use the struct field name and convert to snake_case
			return toSnakeCase(prop.Name)
		}
	}
	// Fallback: convert property name to snake_case
	return toSnakeCase(propertyName)
}

// toSnakeCase converts a camelCase or PascalCase string to snake_case
func toSnakeCase(s string) string {
	result := make([]rune, 0, len(s)+5)
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
