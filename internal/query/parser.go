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
	OpToLower            FilterOperator = "tolower"
	OpToUpper            FilterOperator = "toupper"
	OpTrim               FilterOperator = "trim"
	OpLength             FilterOperator = "length"
	OpIndexOf            FilterOperator = "indexof"
	OpSubstring          FilterOperator = "substring"
	OpConcat             FilterOperator = "concat"
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
