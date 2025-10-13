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
	Apply   []ApplyTransformation
	Search  string                 // Search query string
	Compute *ComputeTransformation // Standalone $compute option
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

// ApplyTransformation represents a single apply transformation
type ApplyTransformation struct {
	Type      ApplyTransformationType
	GroupBy   *GroupByTransformation
	Aggregate *AggregateTransformation
	Filter    *FilterExpression
	Compute   *ComputeTransformation
}

// ApplyTransformationType represents the type of apply transformation
type ApplyTransformationType string

const (
	ApplyTypeGroupBy   ApplyTransformationType = "groupby"
	ApplyTypeAggregate ApplyTransformationType = "aggregate"
	ApplyTypeFilter    ApplyTransformationType = "filter"
	ApplyTypeCompute   ApplyTransformationType = "compute"
)

// GroupByTransformation represents a groupby transformation
type GroupByTransformation struct {
	Properties []string
	Transform  []ApplyTransformation // Nested transformations (typically aggregate)
}

// AggregateTransformation represents an aggregate transformation
type AggregateTransformation struct {
	Expressions []AggregateExpression
}

// AggregateExpression represents a single aggregation expression
type AggregateExpression struct {
	Property   string            // Property to aggregate
	Method     AggregationMethod // Aggregation method (sum, avg, min, max, count, etc.)
	Alias      string            // Alias for the result
	Expression *FilterExpression // Optional expression for countdistinct, etc.
}

// AggregationMethod represents aggregation methods
type AggregationMethod string

const (
	AggregationSum           AggregationMethod = "sum"
	AggregationAvg           AggregationMethod = "average"
	AggregationMin           AggregationMethod = "min"
	AggregationMax           AggregationMethod = "max"
	AggregationCount         AggregationMethod = "count"
	AggregationCountDistinct AggregationMethod = "countdistinct"
)

// ComputeTransformation represents a compute transformation
type ComputeTransformation struct {
	Expressions []ComputeExpression
}

// ComputeExpression represents a single compute expression
type ComputeExpression struct {
	Expression *FilterExpression
	Alias      string
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
	OpIn                 FilterOperator = "in"
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
	OpHas                FilterOperator = "has"
	OpAdd                FilterOperator = "add"
	OpSub                FilterOperator = "sub"
	OpMul                FilterOperator = "mul"
	OpDiv                FilterOperator = "div"
	OpMod                FilterOperator = "mod"
	// Math functions
	OpCeiling FilterOperator = "ceiling"
	OpFloor   FilterOperator = "floor"
	OpRound   FilterOperator = "round"
	// Date functions
	OpYear   FilterOperator = "year"
	OpMonth  FilterOperator = "month"
	OpDay    FilterOperator = "day"
	OpHour   FilterOperator = "hour"
	OpMinute FilterOperator = "minute"
	OpSecond FilterOperator = "second"
	OpDate   FilterOperator = "date"
	OpTime   FilterOperator = "time"
	// Lambda operators
	OpAny FilterOperator = "any"
	OpAll FilterOperator = "all"
	// Type conversion functions
	OpCast FilterOperator = "cast"
)

// LogicalOperator represents logical operators for combining filters
type LogicalOperator string

const (
	LogicalAnd LogicalOperator = "and"
	LogicalOr  LogicalOperator = "or"
)

// validQueryOptions is a set of valid OData v4 system query options
var validQueryOptions = map[string]bool{
	"$filter":        true,
	"$select":        true,
	"$expand":        true,
	"$orderby":       true,
	"$top":           true,
	"$skip":          true,
	"$count":         true,
	"$search":        true,
	"$format":        true,
	"$compute":       true,
	"$index":         true,
	"$schemaversion": true,
	"$apply":         true,
	"$deltatoken":    true,
	"$skiptoken":     true,
}

// validateQueryOptions validates that all query parameters starting with $ are valid OData query options
func validateQueryOptions(queryParams url.Values) error {
	for key := range queryParams {
		// Only validate parameters that start with $
		if strings.HasPrefix(key, "$") {
			if !validQueryOptions[key] {
				return fmt.Errorf("unknown query option: '%s'", key)
			}
		}
	}
	return nil
}

// ParseQueryOptions parses OData query options from the URL
func ParseQueryOptions(queryParams url.Values, entityMetadata *metadata.EntityMetadata) (*QueryOptions, error) {
	options := &QueryOptions{}

	// Validate that all query parameters starting with $ are valid OData query options
	if err := validateQueryOptions(queryParams); err != nil {
		return nil, err
	}

	// Parse each query option
	if err := parseFilterOption(queryParams, entityMetadata, options); err != nil {
		return nil, err
	}

	if err := parseSelectOption(queryParams, entityMetadata, options); err != nil {
		return nil, err
	}

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

	if err := parseApplyOption(queryParams, entityMetadata, options); err != nil {
		return nil, err
	}

	if err := parseSearchOption(queryParams, options); err != nil {
		return nil, err
	}

	if err := parseComputeOption(queryParams, entityMetadata, options); err != nil {
		return nil, err
	}

	// Post-process: merge navigation property selections into expand options
	// This handles cases like $select=Product/Name with $expand=Product
	mergeNavigationSelects(options)

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
func parseSelectOption(queryParams url.Values, entityMetadata *metadata.EntityMetadata, options *QueryOptions) error {
	if selectStr := queryParams.Get("$select"); selectStr != "" {
		selectedProps := parseSelect(selectStr)

		// If $compute or $apply with compute is present, we need to extract computed property aliases
		// to avoid validation errors for properties that will be computed
		computedAliases := make(map[string]bool)

		// Check for standalone $compute parameter
		if computeStr := queryParams.Get("$compute"); computeStr != "" {
			aliases := extractComputeAliasesFromString(computeStr)
			for alias := range aliases {
				computedAliases[alias] = true
			}
		}

		// Check for compute within $apply
		if applyStr := queryParams.Get("$apply"); applyStr != "" {
			aliases := extractComputedAliases(applyStr)
			for alias := range aliases {
				computedAliases[alias] = true
			}
		}

		// Validate that all selected properties exist (either as entity properties or computed properties)
		for _, propName := range selectedProps {
			// Handle navigation property paths (e.g., "Product/Name")
			if strings.Contains(propName, "/") {
				parts := strings.SplitN(propName, "/", 2)
				navPropName := strings.TrimSpace(parts[0])
				subPropName := strings.TrimSpace(parts[1])

				// Validate navigation property exists
				if !isNavigationProperty(navPropName, entityMetadata) {
					return fmt.Errorf("property '%s' does not exist in entity type", propName)
				}

				// Note: We can't easily validate the sub-property without loading the target entity metadata
				// The validation of sub-properties will be handled when the expand is processed
				// For now, we just validate that the navigation property itself exists
				_ = subPropName // Used to track sub-property for later processing
			} else {
				// Regular property validation (also check computed aliases)
				if !propertyExists(propName, entityMetadata) && !computedAliases[propName] {
					return fmt.Errorf("property '%s' does not exist in entity type", propName)
				}
			}
		}
		options.Select = selectedProps
	}
	return nil
}

// extractComputeAliasesFromString extracts aliases from a $compute string
func extractComputeAliasesFromString(computeStr string) map[string]bool {
	aliases := make(map[string]bool)

	// Split by comma and extract aliases
	expressions := splitComputeExpressions(computeStr)
	for _, expr := range expressions {
		// Each expression has format: "expression as alias"
		asIdx := strings.Index(expr, " as ")
		if asIdx != -1 {
			alias := strings.TrimSpace(expr[asIdx+4:])
			aliases[alias] = true
		}
	}

	return aliases
}

// extractComputedAliases extracts aliases from $compute expressions in $apply
func extractComputedAliases(applyStr string) map[string]bool {
	aliases := make(map[string]bool)

	// Look for compute(...) in the apply string
	// Format: compute(expression as alias, ...)
	computeStart := strings.Index(applyStr, "compute(")
	if computeStart == -1 {
		return aliases
	}

	// Find the matching closing parenthesis
	depth := 0
	start := computeStart + 8 // Skip "compute("
	for i := start; i < len(applyStr); i++ {
		if applyStr[i] == '(' {
			depth++
		} else if applyStr[i] == ')' {
			if depth == 0 {
				// Extract the content between compute( and )
				content := applyStr[start:i]
				// Split by comma and extract aliases
				expressions := splitComputeExpressions(content)
				for _, expr := range expressions {
					// Each expression has format: "expression as alias"
					asIdx := strings.Index(expr, " as ")
					if asIdx != -1 {
						alias := strings.TrimSpace(expr[asIdx+4:])
						aliases[alias] = true
					}
				}
				break
			}
			depth--
		}
	}

	return aliases
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

// parseSearchOption parses the $search query parameter
func parseSearchOption(queryParams url.Values, options *QueryOptions) error {
	if searchStr := queryParams.Get("$search"); searchStr != "" {
		searchStr = strings.TrimSpace(searchStr)
		if searchStr == "" {
			return fmt.Errorf("invalid $search: search query cannot be empty")
		}
		options.Search = searchStr
	}
	return nil
}

// parseComputeOption parses the $compute query parameter
func parseComputeOption(queryParams url.Values, entityMetadata *metadata.EntityMetadata, options *QueryOptions) error {
	if computeStr := queryParams.Get("$compute"); computeStr != "" {
		// Parse the compute transformation using the existing parseCompute function from apply_parser.go
		// We need to wrap it in compute(...) format
		computeTransformation, err := parseCompute("compute("+computeStr+")", entityMetadata)
		if err != nil {
			return fmt.Errorf("invalid $compute: %w", err)
		}

		if computeTransformation == nil || computeTransformation.Compute == nil {
			return fmt.Errorf("invalid $compute: failed to parse compute transformation")
		}

		options.Compute = computeTransformation.Compute
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

// mergeNavigationSelects processes $select with navigation paths and merges them into expand options
// For example: $select=Product/Name with $expand=Product should result in Product being expanded with $select=Name
func mergeNavigationSelects(options *QueryOptions) {
	if len(options.Select) == 0 {
		return
	}

	// Track navigation property selections
	navSelects := make(map[string][]string) // nav prop name -> list of sub-properties

	// Process select properties to extract navigation paths
	for _, propName := range options.Select {
		if strings.Contains(propName, "/") {
			parts := strings.SplitN(propName, "/", 2)
			navProp := strings.TrimSpace(parts[0])
			subProp := strings.TrimSpace(parts[1])
			if navSelects[navProp] == nil {
				navSelects[navProp] = []string{}
			}
			navSelects[navProp] = append(navSelects[navProp], subProp)
		}
	}

	// If there are navigation selects, update or add expand options
	if len(navSelects) > 0 {
		// Build a map of existing expand options
		expandMap := make(map[string]*ExpandOption)
		for i := range options.Expand {
			expandMap[options.Expand[i].NavigationProperty] = &options.Expand[i]
		}

		// For each navigation property in select, update or create expand option
		for navProp, subProps := range navSelects {
			if expandOpt, exists := expandMap[navProp]; exists {
				// Merge with existing expand: combine select properties
				if expandOpt.Select == nil {
					expandOpt.Select = subProps
				} else {
					// Merge the lists, avoiding duplicates
					selectMap := make(map[string]bool)
					for _, s := range expandOpt.Select {
						selectMap[s] = true
					}
					for _, s := range subProps {
						if !selectMap[s] {
							expandOpt.Select = append(expandOpt.Select, s)
						}
					}
				}
			} else {
				// Create a new expand option with nested select
				newExpand := ExpandOption{
					NavigationProperty: navProp,
					Select:             subProps,
				}
				options.Expand = append(options.Expand, newExpand)
			}
		}
	}
}
