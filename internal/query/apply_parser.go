package query

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// parseApplyOption parses the $apply query parameter
func parseApplyOption(queryParams map[string][]string, entityMetadata *metadata.EntityMetadata, options *QueryOptions) error {
	applyStr := ""
	if vals, ok := queryParams["$apply"]; ok && len(vals) > 0 {
		applyStr = vals[0]
	}

	if applyStr == "" {
		return nil
	}

	transformations, err := parseApply(applyStr, entityMetadata)
	if err != nil {
		return fmt.Errorf("invalid $apply: %w", err)
	}
	options.Apply = transformations
	return nil
}

// parseApply parses the $apply query parameter value
func parseApply(applyStr string, entityMetadata *metadata.EntityMetadata) ([]ApplyTransformation, error) {
	applyStr = strings.TrimSpace(applyStr)
	if applyStr == "" {
		return nil, fmt.Errorf("empty apply string")
	}

	// Split by '/' to get individual transformations in sequence
	transformationStrs := splitApplyTransformations(applyStr)
	transformations := make([]ApplyTransformation, 0, len(transformationStrs))

	for _, transStr := range transformationStrs {
		transStr = strings.TrimSpace(transStr)
		if transStr == "" {
			continue
		}

		transformation, err := parseApplyTransformation(transStr, entityMetadata)
		if err != nil {
			return nil, err
		}
		transformations = append(transformations, *transformation)
	}

	if len(transformations) == 0 {
		return nil, fmt.Errorf("no valid transformations found")
	}

	return transformations, nil
}

// splitApplyTransformations splits the apply string by '/' while respecting parentheses
func splitApplyTransformations(applyStr string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for i := 0; i < len(applyStr); i++ {
		ch := applyStr[i]
		switch ch {
		case '(':
			depth++
			current.WriteByte(ch)
		case ')':
			depth--
			current.WriteByte(ch)
		case '/':
			if depth == 0 {
				// This is a transformation separator
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// parseApplyTransformation parses a single transformation
func parseApplyTransformation(transStr string, entityMetadata *metadata.EntityMetadata) (*ApplyTransformation, error) {
	transStr = strings.TrimSpace(transStr)

	// Determine transformation type
	if strings.HasPrefix(transStr, "groupby(") {
		return parseGroupBy(transStr, entityMetadata)
	} else if strings.HasPrefix(transStr, "aggregate(") {
		return parseAggregate(transStr, entityMetadata)
	} else if strings.HasPrefix(transStr, "filter(") {
		return parseFilterTransformation(transStr, entityMetadata)
	} else if strings.HasPrefix(transStr, "compute(") {
		return parseCompute(transStr, entityMetadata)
	}

	return nil, fmt.Errorf("unknown transformation: %s", transStr)
}

// parseGroupBy parses a groupby transformation
// Format: groupby((prop1,prop2), aggregate(expr))
// or: groupby((prop1,prop2))
func parseGroupBy(transStr string, entityMetadata *metadata.EntityMetadata) (*ApplyTransformation, error) {
	if !strings.HasPrefix(transStr, "groupby(") {
		return nil, fmt.Errorf("invalid groupby format")
	}

	// Extract content between groupby( and final )
	content := transStr[8:] // Skip "groupby("
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in groupby")
	}
	content = content[:len(content)-1] // Remove final )
	content = strings.TrimSpace(content)

	// Parse the groupby properties and optional transformations
	// Format: (prop1,prop2), aggregate(...)
	// or: (prop1,prop2)

	// Find the properties section (first parenthesized section)
	if !strings.HasPrefix(content, "(") {
		return nil, fmt.Errorf("groupby properties must be in parentheses")
	}

	// Find matching closing parenthesis for properties
	propsEndIdx := findMatchingCloseParen(content, 0)
	if propsEndIdx == -1 {
		return nil, fmt.Errorf("missing closing parenthesis for groupby properties")
	}

	propsStr := content[1:propsEndIdx] // Extract properties without outer parentheses
	properties := parseGroupByProperties(propsStr)

	// Validate properties
	for _, prop := range properties {
		if !propertyExists(prop, entityMetadata) {
			return nil, fmt.Errorf("property '%s' does not exist in entity type", prop)
		}
	}

	groupBy := &GroupByTransformation{
		Properties: properties,
	}

	// Check if there are nested transformations after the properties
	remaining := strings.TrimSpace(content[propsEndIdx+1:])
	if remaining != "" {
		// Should start with comma
		if !strings.HasPrefix(remaining, ",") {
			return nil, fmt.Errorf("expected comma after groupby properties")
		}
		remaining = strings.TrimSpace(remaining[1:]) // Skip comma

		// Parse nested transformations
		nestedTrans, err := parseApplyTransformation(remaining, entityMetadata)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nested transformation: %w", err)
		}
		groupBy.Transform = []ApplyTransformation{*nestedTrans}
	}

	return &ApplyTransformation{
		Type:    ApplyTypeGroupBy,
		GroupBy: groupBy,
	}, nil
}

// parseGroupByProperties parses a comma-separated list of properties
func parseGroupByProperties(propsStr string) []string {
	propsStr = strings.TrimSpace(propsStr)
	if propsStr == "" {
		return []string{}
	}

	parts := strings.Split(propsStr, ",")
	properties := make([]string, 0, len(parts))
	for _, part := range parts {
		prop := strings.TrimSpace(part)
		if prop != "" {
			properties = append(properties, prop)
		}
	}
	return properties
}

// parseAggregate parses an aggregate transformation
// Format: aggregate(prop1 with sum as Total, prop2 with average as Avg)
func parseAggregate(transStr string, entityMetadata *metadata.EntityMetadata) (*ApplyTransformation, error) {
	if !strings.HasPrefix(transStr, "aggregate(") {
		return nil, fmt.Errorf("invalid aggregate format")
	}

	content := transStr[10:] // Skip "aggregate("
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in aggregate")
	}
	content = content[:len(content)-1] // Remove final )
	content = strings.TrimSpace(content)

	// Parse individual aggregate expressions
	exprStrs := splitAggregateExpressions(content)
	expressions := make([]AggregateExpression, 0, len(exprStrs))

	for _, exprStr := range exprStrs {
		expr, err := parseAggregateExpression(exprStr, entityMetadata)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, *expr)
	}

	if len(expressions) == 0 {
		return nil, fmt.Errorf("no valid aggregate expressions found")
	}

	return &ApplyTransformation{
		Type: ApplyTypeAggregate,
		Aggregate: &AggregateTransformation{
			Expressions: expressions,
		},
	}, nil
}

// splitAggregateExpressions splits aggregate expressions by comma, respecting parentheses
func splitAggregateExpressions(content string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for i := 0; i < len(content); i++ {
		ch := content[i]
		switch ch {
		case '(':
			depth++
			current.WriteByte(ch)
		case ')':
			depth--
			current.WriteByte(ch)
		case ',':
			if depth == 0 {
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// parseAggregateExpression parses a single aggregate expression
// Formats:
//   - prop with sum as Total
//   - prop with average as Avg
//   - $count as Total
func parseAggregateExpression(exprStr string, entityMetadata *metadata.EntityMetadata) (*AggregateExpression, error) {
	exprStr = strings.TrimSpace(exprStr)

	// Check for $count as alias format
	if strings.HasPrefix(exprStr, "$count") {
		// Format: $count as alias
		parts := strings.Fields(exprStr)
		if len(parts) != 3 || parts[1] != "as" {
			return nil, fmt.Errorf("invalid $count format, expected '$count as alias'")
		}
		return &AggregateExpression{
			Property: "$count",
			Method:   AggregationCount,
			Alias:    parts[2],
		}, nil
	}

	// Standard format: property with method as alias
	// Split by " with " to get property and method/alias
	withIdx := strings.Index(exprStr, " with ")
	if withIdx == -1 {
		return nil, fmt.Errorf("invalid aggregate expression format, expected 'property with method as alias'")
	}

	property := strings.TrimSpace(exprStr[:withIdx])
	remainder := strings.TrimSpace(exprStr[withIdx+6:]) // Skip " with "

	// Split remainder by " as " to get method and alias
	asIdx := strings.Index(remainder, " as ")
	if asIdx == -1 {
		return nil, fmt.Errorf("invalid aggregate expression format, missing 'as alias'")
	}

	methodStr := strings.TrimSpace(remainder[:asIdx])
	alias := strings.TrimSpace(remainder[asIdx+4:]) // Skip " as "

	// Validate property exists
	if !propertyExists(property, entityMetadata) {
		return nil, fmt.Errorf("property '%s' does not exist in entity type", property)
	}

	// Parse aggregation method
	method, err := parseAggregationMethod(methodStr)
	if err != nil {
		return nil, err
	}

	return &AggregateExpression{
		Property: property,
		Method:   method,
		Alias:    alias,
	}, nil
}

// parseAggregationMethod parses an aggregation method string
func parseAggregationMethod(methodStr string) (AggregationMethod, error) {
	switch strings.ToLower(methodStr) {
	case "sum":
		return AggregationSum, nil
	case "average", "avg":
		return AggregationAvg, nil
	case "min":
		return AggregationMin, nil
	case "max":
		return AggregationMax, nil
	case "count":
		return AggregationCount, nil
	case "countdistinct":
		return AggregationCountDistinct, nil
	default:
		return "", fmt.Errorf("unknown aggregation method: %s", methodStr)
	}
}

// parseFilterTransformation parses a filter transformation within apply
// Format: filter(expression)
func parseFilterTransformation(transStr string, entityMetadata *metadata.EntityMetadata) (*ApplyTransformation, error) {
	if !strings.HasPrefix(transStr, "filter(") {
		return nil, fmt.Errorf("invalid filter format")
	}

	content := transStr[7:] // Skip "filter("
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in filter")
	}
	content = content[:len(content)-1] // Remove final )
	content = strings.TrimSpace(content)

	// Parse the filter expression
	filter, err := parseFilter(content, entityMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse filter expression: %w", err)
	}

	return &ApplyTransformation{
		Type:   ApplyTypeFilter,
		Filter: filter,
	}, nil
}

// parseCompute parses a compute transformation
// Format: compute(expression as alias, ...)
func parseCompute(transStr string, entityMetadata *metadata.EntityMetadata) (*ApplyTransformation, error) {
	if !strings.HasPrefix(transStr, "compute(") {
		return nil, fmt.Errorf("invalid compute format")
	}

	content := transStr[8:] // Skip "compute("
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in compute")
	}
	content = content[:len(content)-1] // Remove final )
	content = strings.TrimSpace(content)

	// Parse individual compute expressions
	exprStrs := splitComputeExpressions(content)
	expressions := make([]ComputeExpression, 0, len(exprStrs))

	for _, exprStr := range exprStrs {
		expr, err := parseComputeExpression(exprStr, entityMetadata)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, *expr)
	}

	if len(expressions) == 0 {
		return nil, fmt.Errorf("no valid compute expressions found")
	}

	return &ApplyTransformation{
		Type: ApplyTypeCompute,
		Compute: &ComputeTransformation{
			Expressions: expressions,
		},
	}, nil
}

// splitComputeExpressions splits compute expressions by comma, respecting parentheses
func splitComputeExpressions(content string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for i := 0; i < len(content); i++ {
		ch := content[i]
		switch ch {
		case '(':
			depth++
			current.WriteByte(ch)
		case ')':
			depth--
			current.WriteByte(ch)
		case ',':
			if depth == 0 {
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// parseComputeExpression parses a single compute expression
// Format: expression as alias
func parseComputeExpression(exprStr string, entityMetadata *metadata.EntityMetadata) (*ComputeExpression, error) {
	exprStr = strings.TrimSpace(exprStr)

	// Split by " as " to get expression and alias
	asIdx := strings.Index(exprStr, " as ")
	if asIdx == -1 {
		return nil, fmt.Errorf("invalid compute expression format, expected 'expression as alias'")
	}

	expressionStr := strings.TrimSpace(exprStr[:asIdx])
	alias := strings.TrimSpace(exprStr[asIdx+4:]) // Skip " as "

	// Parse the expression as a filter expression
	// For simplicity, we'll support basic expressions for now
	expression, err := parseFilter(expressionStr, entityMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compute expression: %w", err)
	}

	return &ComputeExpression{
		Expression: expression,
		Alias:      alias,
	}, nil
}

// findMatchingCloseParen finds the index of the closing parenthesis that matches the opening one at startIdx
func findMatchingCloseParen(s string, startIdx int) int {
	if startIdx >= len(s) || s[startIdx] != '(' {
		return -1
	}

	depth := 0
	for i := startIdx; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1 // No matching closing parenthesis found
}
