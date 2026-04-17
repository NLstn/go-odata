package query

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/nlstn/go-odata/internal/metadata"
)

// parseApplyOption parses the $apply query parameter
func parseApplyOption(queryParams map[string][]string, entityMetadata *metadata.EntityMetadata, options *QueryOptions, config *ParserConfig, caseInsensitive bool) error {
	applyStr := ""
	if vals, ok := queryParams["$apply"]; ok && len(vals) > 0 {
		applyStr = vals[0]
	}

	if applyStr == "" {
		return nil
	}

	maxInClauseSize := 0
	if config != nil {
		maxInClauseSize = config.MaxInClauseSize
	}

	var transformations []ApplyTransformation
	var err error
	if caseInsensitive {
		transformations, err = parseApply(applyStr, entityMetadata, maxInClauseSize)
	} else {
		transformations, err = parseApplyWithCaseSensitivity(applyStr, entityMetadata, maxInClauseSize, false)
	}
	if err != nil {
		return fmt.Errorf("invalid $apply: %w", err)
	}
	options.Apply = transformations
	return nil
}

// parseApply parses the $apply query parameter value
func parseApply(applyStr string, entityMetadata *metadata.EntityMetadata, maxInClauseSize int) ([]ApplyTransformation, error) {
	return parseApplyWithCaseSensitivity(applyStr, entityMetadata, maxInClauseSize, true)
}

// parseApplyWithCaseSensitivity parses the $apply query parameter value and controls
// whether transformation names are parsed case-insensitively (4.01 behavior) or
// strictly (4.0 behavior).
func parseApplyWithCaseSensitivity(applyStr string, entityMetadata *metadata.EntityMetadata, maxInClauseSize int, caseInsensitive bool) ([]ApplyTransformation, error) {
	applyStr = strings.TrimSpace(applyStr)
	if applyStr == "" {
		return nil, errEmptyApplyString
	}

	// Split by '/' to get individual transformations in sequence
	transformationStrs := splitApplyTransformations(applyStr)
	transformations := make([]ApplyTransformation, 0, len(transformationStrs))

	// Track computed aliases as we parse through transformations
	// This allows subsequent filter/orderby transformations to reference computed properties
	computedAliases := make(map[string]bool)

	for _, transStr := range transformationStrs {
		transStr = strings.TrimSpace(transStr)
		if transStr == "" {
			continue
		}

		transformation, err := parseApplyTransformationWithAliases(transStr, entityMetadata, computedAliases, maxInClauseSize, caseInsensitive)
		if err != nil {
			return nil, err
		}
		transformations = append(transformations, *transformation)

		// Extract aliases from this transformation for use in subsequent transformations
		extractAliasesFromTransformation(transformation, computedAliases)
	}

	if len(transformations) == 0 {
		return nil, errNoValidTransformations
	}

	return transformations, nil
}

func canonicalizeApplyTransformationKeyword(transStr string) string {
	keywords := []string{
		"groupby(",
		"aggregate(",
		"filter(",
		"compute(",
		"orderby(",
		"topcount(",
		"bottomcount(",
		"toppercent(",
		"bottompercent(",
		"topsum(",
		"bottomsum(",
		"search(",
		"concat(",
		"join(",
		"outerjoin(",
		"ancestors(",
		"descendants(",
		"traverse(",
		"top(",
		"skip(",
		"nest(",
		"from(",
	}

	for _, keyword := range keywords {
		if len(transStr) >= len(keyword) && strings.EqualFold(transStr[:len(keyword)], keyword) {
			return keyword + transStr[len(keyword):]
		}
	}

	if strings.EqualFold(transStr, "identity") {
		return "identity"
	}

	return transStr
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

// parseApplyTransformationWithAliases parses a single transformation with computed aliases support
func parseApplyTransformationWithAliases(transStr string, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool, maxInClauseSize int, caseInsensitive bool) (*ApplyTransformation, error) {
	transStr = strings.TrimSpace(transStr)
	if caseInsensitive {
		transStr = canonicalizeApplyTransformationKeyword(transStr)
	}

	if transStr == "identity" {
		return &ApplyTransformation{Type: ApplyTypeIdentity}, nil
	}

	// Determine transformation type
	if strings.HasPrefix(transStr, "groupby(") {
		return parseGroupBy(transStr, entityMetadata, caseInsensitive)
	} else if strings.HasPrefix(transStr, "aggregate(") {
		return parseAggregate(transStr, entityMetadata)
	} else if strings.HasPrefix(transStr, "filter(") {
		return parseFilterTransformation(transStr, entityMetadata, computedAliases, maxInClauseSize)
	} else if strings.HasPrefix(transStr, "compute(") {
		return parseCompute(transStr, entityMetadata, maxInClauseSize)
	} else if strings.HasPrefix(transStr, "orderby(") {
		return parseOrderByTransformation(transStr, entityMetadata, computedAliases)
	} else if strings.HasPrefix(transStr, "top(") {
		return parseTopTransformation(transStr)
	} else if strings.HasPrefix(transStr, "skip(") {
		return parseSkipTransformation(transStr)
	} else if strings.HasPrefix(transStr, "search(") {
		return parseSearchTransformation(transStr)
	} else if strings.HasPrefix(transStr, "concat(") {
		return parseConcatTransformation(transStr, entityMetadata, maxInClauseSize, caseInsensitive)
	} else if strings.HasPrefix(transStr, "join(") {
		return parseJoinTransformation(transStr, entityMetadata, ApplyTypeJoin, caseInsensitive)
	} else if strings.HasPrefix(transStr, "outerjoin(") {
		return parseJoinTransformation(transStr, entityMetadata, ApplyTypeOuterJoin, caseInsensitive)
	} else if strings.HasPrefix(transStr, "topcount(") {
		return parseSetTransformation(transStr, entityMetadata, ApplyTypeTopCount)
	} else if strings.HasPrefix(transStr, "bottomcount(") {
		return parseSetTransformation(transStr, entityMetadata, ApplyTypeBottomCount)
	} else if strings.HasPrefix(transStr, "toppercent(") {
		return parseSetTransformation(transStr, entityMetadata, ApplyTypeTopPercent)
	} else if strings.HasPrefix(transStr, "bottompercent(") {
		return parseSetTransformation(transStr, entityMetadata, ApplyTypeBottomPercent)
	} else if strings.HasPrefix(transStr, "topsum(") {
		return parseSetTransformation(transStr, entityMetadata, ApplyTypeTopSum)
	} else if strings.HasPrefix(transStr, "bottomsum(") {
		return parseSetTransformation(transStr, entityMetadata, ApplyTypeBottomSum)
	} else if strings.HasPrefix(transStr, "ancestors(") {
		return parseHierarchyTransformation(transStr, ApplyTypeAncestors)
	} else if strings.HasPrefix(transStr, "descendants(") {
		return parseHierarchyTransformation(transStr, ApplyTypeDescendants)
	} else if strings.HasPrefix(transStr, "traverse(") {
		return parseHierarchyTransformation(transStr, ApplyTypeTraverse)
	} else if strings.HasPrefix(transStr, "nest(") {
		return parseNestTransformation(transStr, entityMetadata, maxInClauseSize, caseInsensitive)
	} else if strings.HasPrefix(transStr, "from(") {
		return parseFromTransformation(transStr)
	} else if fnName, ok := parseServiceDefinedFunctionTransformation(transStr); ok {
		return nil, fmt.Errorf("service-defined set transformation '%s' is not supported", fnName)
	}

	return nil, fmt.Errorf("unknown transformation: %s", transStr)
}

// extractAliasesFromTransformation extracts computed aliases from a transformation
func extractAliasesFromTransformation(trans *ApplyTransformation, aliases map[string]bool) {
	if trans == nil {
		return
	}

	switch trans.Type {
	case ApplyTypeGroupBy:
		// groupby creates a virtual $count property that can be used in subsequent filters
		if trans.GroupBy != nil {
			aliases["$count"] = true
			// Also extract aliases from nested aggregate transformations
			for _, nestedTrans := range trans.GroupBy.Transform {
				extractAliasesFromTransformation(&nestedTrans, aliases)
			}
		}
	case ApplyTypeAggregate:
		// Extract aliases from aggregate expressions
		if trans.Aggregate != nil {
			for _, expr := range trans.Aggregate.Expressions {
				if expr.Alias != "" {
					aliases[expr.Alias] = true
				}
			}
		}
	case ApplyTypeCompute:
		// Extract aliases from compute expressions
		if trans.Compute != nil {
			for _, expr := range trans.Compute.Expressions {
				if expr.Alias != "" {
					aliases[expr.Alias] = true
				}
			}
		}
	}
}

// parseGroupBy parses a groupby transformation
// Format: groupby((prop1,prop2), aggregate(expr))
// or: groupby((prop1,prop2))
// or: groupby(($all), aggregate(expr))  -- aggregates over the entire input set
func parseGroupBy(transStr string, entityMetadata *metadata.EntityMetadata, caseInsensitive bool) (*ApplyTransformation, error) {
	if !strings.HasPrefix(transStr, "groupby(") {
		return nil, errInvalidGroupByFormat
	}

	// Extract content between groupby( and final )
	content := transStr[8:] // Skip "groupby("
	if !strings.HasSuffix(content, ")") {
		return nil, errMissingClosingParenGroupBy
	}
	content = content[:len(content)-1] // Remove final )
	content = strings.TrimSpace(content)

	// Parse the groupby properties and optional transformations
	// Format: (prop1,prop2), aggregate(...)
	// or: (prop1,prop2)
	// or: ($all), aggregate(...)

	// Find the properties section (first parenthesized section)
	if !strings.HasPrefix(content, "(") {
		return nil, errGroupByPropsNeedParens
	}

	// Find matching closing parenthesis for properties
	propsEndIdx := findMatchingCloseParen(content, 0)
	if propsEndIdx == -1 {
		return nil, errMissingClosingParenGroupByProps
	}

	propsStr := content[1:propsEndIdx] // Extract properties without outer parentheses
	propsStr = strings.TrimSpace(propsStr)

	// Check for $all special keyword: groupby(($all), ...)
	// $all means aggregate across all values without any grouping.
	allValues := strings.EqualFold(propsStr, "$all")

	var groupBy *GroupByTransformation
	if allValues {
		groupBy = &GroupByTransformation{
			AllValues: true,
		}
	} else {
		properties := parseGroupByProperties(propsStr)

		// Validate properties
		for _, prop := range properties {
			if !propertyExists(prop, entityMetadata) {
				return nil, fmt.Errorf("property '%s' does not exist in entity type", prop)
			}
		}

		groupBy = &GroupByTransformation{
			Properties: properties,
		}
	}

	// Check if there are nested transformations after the properties
	remaining := strings.TrimSpace(content[propsEndIdx+1:])
	if remaining != "" {
		// Should start with comma
		if !strings.HasPrefix(remaining, ",") {
			return nil, errExpectedCommaAfterGroupByProps
		}
		remaining = strings.TrimSpace(remaining[1:]) // Skip comma

		// Parse nested transformation sequence.
		// Example: groupby((Category),aggregate(...)/filter(...)/top(5))
		nestedTrans, err := parseApplyWithCaseSensitivity(remaining, entityMetadata, 0, caseInsensitive)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nested transformation sequence: %w", err)
		}
		groupBy.Transform = nestedTrans
	}

	return &ApplyTransformation{
		Type:    ApplyTypeGroupBy,
		GroupBy: groupBy,
	}, nil
}

// parseOrderByTransformation parses an orderby transformation.
// Format: orderby(prop1 desc,prop2 asc)
func parseOrderByTransformation(transStr string, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool) (*ApplyTransformation, error) {
	if !strings.HasPrefix(transStr, "orderby(") {
		return nil, fmt.Errorf("invalid orderby format")
	}

	content := transStr[8:] // Skip "orderby("
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in orderby")
	}
	content = strings.TrimSpace(content[:len(content)-1])

	orderBy, err := parseOrderBy(content, entityMetadata, computedAliases)
	if err != nil {
		return nil, err
	}

	return &ApplyTransformation{
		Type:    ApplyTypeOrderBy,
		OrderBy: orderBy,
	}, nil
}

// parseTopTransformation parses a top transformation.
// Format: top(5)
func parseTopTransformation(transStr string) (*ApplyTransformation, error) {
	if !strings.HasPrefix(transStr, "top(") {
		return nil, fmt.Errorf("invalid top format")
	}

	content := transStr[4:] // Skip "top("
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in top")
	}
	content = strings.TrimSpace(content[:len(content)-1])

	top, err := parseNonNegativeInt(content, "top")
	if err != nil {
		return nil, err
	}

	return &ApplyTransformation{Type: ApplyTypeTop, Top: &top}, nil
}

// parseSkipTransformation parses a skip transformation.
// Format: skip(5)
func parseSkipTransformation(transStr string) (*ApplyTransformation, error) {
	if !strings.HasPrefix(transStr, "skip(") {
		return nil, fmt.Errorf("invalid skip format")
	}

	content := transStr[5:] // Skip "skip("
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in skip")
	}
	content = strings.TrimSpace(content[:len(content)-1])

	skip, err := parseNonNegativeInt(content, "skip")
	if err != nil {
		return nil, err
	}

	return &ApplyTransformation{Type: ApplyTypeSkip, Skip: &skip}, nil
}

// parseSearchTransformation parses a search transformation.
// Format: search(term-expression)
func parseSearchTransformation(transStr string) (*ApplyTransformation, error) {
	content := transStr[len("search("):]
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in search")
	}
	query := strings.TrimSpace(content[:len(content)-1])
	if query == "" {
		return nil, errInvalidSearch
	}
	return &ApplyTransformation{Type: ApplyTypeSearch, Search: &query}, nil
}

// parseConcatTransformation parses concat(seq1,seq2,...).
func parseConcatTransformation(transStr string, entityMetadata *metadata.EntityMetadata, maxInClauseSize int, caseInsensitive bool) (*ApplyTransformation, error) {
	content := transStr[len("concat("):]
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in concat")
	}
	content = strings.TrimSpace(content[:len(content)-1])
	if content == "" {
		return nil, fmt.Errorf("concat requires at least one transformation sequence")
	}

	parts := splitAggregateExpressions(content)
	if len(parts) == 0 {
		return nil, fmt.Errorf("concat requires at least one transformation sequence")
	}

	sequences := make([][]ApplyTransformation, 0, len(parts))
	for _, part := range parts {
		seq, err := parseApplyWithCaseSensitivity(strings.TrimSpace(part), entityMetadata, maxInClauseSize, caseInsensitive)
		if err != nil {
			return nil, fmt.Errorf("failed to parse concat sequence: %w", err)
		}
		sequences = append(sequences, seq)
	}

	return &ApplyTransformation{
		Type:   ApplyTypeConcat,
		Concat: &ConcatTransformation{Sequences: sequences},
	}, nil
}

func parseJoinTransformation(transStr string, entityMetadata *metadata.EntityMetadata, t ApplyTransformationType, caseInsensitive bool) (*ApplyTransformation, error) {
	keyword := string(t) + "("
	content := transStr[len(keyword):]
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in %s", t)
	}
	content = strings.TrimSpace(content[:len(content)-1])
	if content == "" {
		return nil, fmt.Errorf("%s requires a collection navigation property and alias", t)
	}

	parts := splitAggregateExpressions(content)
	if len(parts) == 0 {
		return nil, fmt.Errorf("%s requires a collection navigation property and alias", t)
	}
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid %s format, expected '%s(Property as Alias[,transform-sequence])'", t, t)
	}

	binding := strings.TrimSpace(parts[0])
	separatorIndex := strings.Index(strings.ToLower(binding), " as ")
	if caseInsensitive {
		separatorIndex = strings.Index(strings.ToLower(binding), " as ")
	}
	if separatorIndex < 0 {
		return nil, fmt.Errorf("invalid %s format, expected '%s(Property as Alias)'", t, t)
	}

	property := strings.TrimSpace(binding[:separatorIndex])
	alias := strings.TrimSpace(binding[separatorIndex+4:])
	if property == "" || alias == "" {
		return nil, fmt.Errorf("invalid %s format, expected '%s(Property as Alias)'", t, t)
	}
	if entityMetadata == nil {
		return nil, fmt.Errorf("entity metadata is required for %s", t)
	}

	navProp := entityMetadata.FindNavigationProperty(property)
	if navProp == nil {
		return nil, fmt.Errorf("navigation property '%s' does not exist in entity type", property)
	}
	if !navProp.NavigationIsArray {
		return nil, fmt.Errorf("navigation property '%s' is not a collection", property)
	}

	var nestedTransform []ApplyTransformation
	if len(parts) == 2 {
		nestedStr := strings.TrimSpace(parts[1])
		if nestedStr == "" {
			return nil, fmt.Errorf("invalid %s format, expected nested transformation sequence", t)
		}
		parsed, err := parseApplyWithCaseSensitivity(nestedStr, entityMetadata, 0, caseInsensitive)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nested transformation sequence: %w", err)
		}
		nestedTransform = parsed
	}

	return &ApplyTransformation{
		Type: t,
		Join: &JoinTransformation{
			Property:  navProp.Name,
			Alias:     alias,
			Transform: nestedTransform,
		},
	}, nil
}

func parseHierarchyTransformation(transStr string, t ApplyTransformationType) (*ApplyTransformation, error) {
	keyword := string(t) + "("
	content := transStr[len(keyword):]
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in %s", t)
	}
	content = strings.TrimSpace(content[:len(content)-1])
	if content == "" {
		return nil, fmt.Errorf("%s requires fully-specified hierarchy parameters and must not be invoked without arguments", t)
	}
	// Hierarchy traversal semantics are not yet implemented
	return nil, fmt.Errorf("%s requires fully-specified hierarchy parameters; hierarchy traversal is not yet supported", t)
}

func parseServiceDefinedFunctionTransformation(transStr string) (string, bool) {
	open := strings.IndexByte(transStr, '(')
	if open <= 0 || !strings.HasSuffix(transStr, ")") {
		return "", false
	}

	name := strings.TrimSpace(transStr[:open])
	if name == "" || !strings.Contains(name, ".") {
		return "", false
	}

	args := strings.TrimSpace(transStr[open+1 : len(transStr)-1])
	if args != "" {
		return "", false
	}

	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return "", false
	}
	for _, part := range parts {
		if !isIdentifier(part) {
			return "", false
		}
	}

	return name, true
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// parseSetTransformation parses topcount/bottomcount/toppercent/bottompercent/topsum/bottomsum.
func parseSetTransformation(transStr string, entityMetadata *metadata.EntityMetadata, t ApplyTransformationType) (*ApplyTransformation, error) {
	open := strings.IndexByte(transStr, '(')
	if open < 0 || !strings.HasSuffix(transStr, ")") {
		return nil, fmt.Errorf("invalid %s format", t)
	}

	content := strings.TrimSpace(transStr[open+1 : len(transStr)-1])
	parts := splitAggregateExpressions(content)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid %s format, expected '%s(value,measure)'", t, t)
	}

	valueStr := strings.TrimSpace(parts[0])
	measure := strings.TrimSpace(parts[1])
	if !propertyExists(measure, entityMetadata) {
		return nil, fmt.Errorf("property '%s' does not exist in entity type", measure)
	}

	set := &SetTransformation{Measure: measure}

	switch t {
	case ApplyTypeTopCount, ApplyTypeBottomCount:
		count, err := parsePositiveInt(valueStr, string(t))
		if err != nil {
			return nil, err
		}
		set.Count = &count
		set.Parameter = float64(count)
	case ApplyTypeTopPercent, ApplyTypeBottomPercent, ApplyTypeTopSum, ApplyTypeBottomSum:
		v, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid %s value: %s", t, valueStr)
		}
		if (t == ApplyTypeTopPercent || t == ApplyTypeBottomPercent) && (v <= 0 || v > 100) {
			return nil, fmt.Errorf("%s value must be greater than 0 and less than or equal to 100", t)
		}
		set.Parameter = v
	default:
		return nil, fmt.Errorf("unsupported set transformation type: %s", t)
	}

	return &ApplyTransformation{Type: t, Set: set}, nil
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
		return nil, errInvalidAggregateFormat
	}

	content := transStr[10:] // Skip "aggregate("
	if !strings.HasSuffix(content, ")") {
		return nil, errMissingClosingParenAggregate
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
		return nil, errNoValidAggregateExpressions
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
			return nil, errInvalidCountFormat
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
		return nil, errInvalidAggregateExprFormat
	}

	property := strings.TrimSpace(exprStr[:withIdx])
	remainder := strings.TrimSpace(exprStr[withIdx+6:]) // Skip " with "

	// Split remainder by " as " to get method and alias
	asIdx := strings.Index(remainder, " as ")
	if asIdx == -1 {
		return nil, errInvalidAggregateExprMissingAs
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
func parseFilterTransformation(transStr string, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool, maxInClauseSize int) (*ApplyTransformation, error) {
	if !strings.HasPrefix(transStr, "filter(") {
		return nil, errInvalidFilterFormat
	}

	content := transStr[7:] // Skip "filter("
	if !strings.HasSuffix(content, ")") {
		return nil, errMissingClosingParenFilter
	}
	content = content[:len(content)-1] // Remove final )
	content = strings.TrimSpace(content)

	// Parse the filter expression with computed aliases support and maxInClauseSize
	filter, err := parseFilter(content, entityMetadata, computedAliases, maxInClauseSize)
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
func parseCompute(transStr string, entityMetadata *metadata.EntityMetadata, maxInClauseSize int) (*ApplyTransformation, error) {
	if !strings.HasPrefix(transStr, "compute(") {
		return nil, errInvalidComputeFormat
	}

	content := transStr[8:] // Skip "compute("
	if !strings.HasSuffix(content, ")") {
		return nil, errMissingClosingParenCompute
	}
	content = content[:len(content)-1] // Remove final )
	content = strings.TrimSpace(content)

	// Parse individual compute expressions
	exprStrs := splitComputeExpressions(content)
	expressions := make([]ComputeExpression, 0, len(exprStrs))

	for _, exprStr := range exprStrs {
		expr, err := parseComputeExpression(exprStr, entityMetadata, maxInClauseSize)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, *expr)
	}

	if len(expressions) == 0 {
		return nil, errNoValidComputeExpressions
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
func parseComputeExpression(exprStr string, entityMetadata *metadata.EntityMetadata, maxInClauseSize int) (*ComputeExpression, error) {
	exprStr = strings.TrimSpace(exprStr)

	// Split by " as " to get expression and alias
	asIdx := strings.Index(exprStr, " as ")
	if asIdx == -1 {
		return nil, errInvalidComputeExprFormat
	}

	expressionStr := strings.TrimSpace(exprStr[:asIdx])
	alias := strings.TrimSpace(exprStr[asIdx+4:]) // Skip " as "

	// Parse the expression as a filter expression with maxInClauseSize enforcement
	expression, err := parseFilter(expressionStr, entityMetadata, nil, maxInClauseSize)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compute expression: %w", err)
	}

	return &ComputeExpression{
		Expression: expression,
		Alias:      alias,
	}, nil
}

// parseComputeWithoutMetadata parses $compute expressions without entity metadata validation
// This is used for nested $compute within $expand where we don't have access to the target entity metadata
func parseComputeWithoutMetadata(computeStr string) (*ComputeTransformation, error) {
	computeStr = strings.TrimSpace(computeStr)
	if computeStr == "" {
		return nil, errEmptyComputeExpression
	}

	// Parse individual compute expressions
	exprStrs := splitComputeExpressions(computeStr)
	expressions := make([]ComputeExpression, 0, len(exprStrs))

	for _, exprStr := range exprStrs {
		expr, err := parseComputeExpressionWithoutMetadata(exprStr)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, *expr)
	}

	if len(expressions) == 0 {
		return nil, errNoValidComputeExpressions
	}

	return &ComputeTransformation{
		Expressions: expressions,
	}, nil
}

// parseComputeExpressionWithoutMetadata parses a single compute expression without metadata validation
func parseComputeExpressionWithoutMetadata(exprStr string) (*ComputeExpression, error) {
	exprStr = strings.TrimSpace(exprStr)

	// Split by " as " (case-insensitive) to get expression and alias
	lowerExprStr := strings.ToLower(exprStr)
	asIdx := strings.Index(lowerExprStr, " as ")
	if asIdx == -1 {
		return nil, errInvalidComputeExprFormat
	}

	expressionStr := strings.TrimSpace(exprStr[:asIdx])
	alias := strings.TrimSpace(exprStr[asIdx+4:]) // Skip " as "

	// Parse the expression without metadata validation
	expression, err := ParseFilterWithoutMetadata(expressionStr)
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

// parseNestTransformation parses a nest($apply=...) or nest($apply=..., alias) transformation.
// Format: nest($apply=transformationSequence)
// or:     nest($apply=transformationSequence, alias)
//
// The nest transformation nests the result of an inner $apply transformation as a
// sub-collection property on each entity in the current input set.
func parseNestTransformation(transStr string, entityMetadata *metadata.EntityMetadata, maxInClauseSize int, caseInsensitive bool) (*ApplyTransformation, error) {
	content := transStr[len("nest("):]
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in nest")
	}
	content = strings.TrimSpace(content[:len(content)-1])
	if content == "" {
		return nil, fmt.Errorf("nest requires a $apply= argument")
	}

	// Parse: $apply=transformationSequence[, alias]
	// Split at the top-level comma to separate $apply= value from optional alias
	applyPrefix := "$apply="
	if !strings.HasPrefix(strings.ToLower(content), applyPrefix) {
		return nil, fmt.Errorf("nest argument must start with $apply=")
	}
	applyContent := content[len(applyPrefix):]

	// Check for an optional alias after the transformation sequence
	alias := ""
	// Split top-level: everything before the last top-level comma is the $apply, after is alias
	commaIdx := -1
	depth := 0
	for i := 0; i < len(applyContent); i++ {
		switch applyContent[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				commaIdx = i
			}
		}
	}
	if commaIdx >= 0 {
		alias = strings.TrimSpace(applyContent[commaIdx+1:])
		applyContent = strings.TrimSpace(applyContent[:commaIdx])
	}

	innerTransforms, err := parseApplyWithCaseSensitivity(applyContent, entityMetadata, maxInClauseSize, caseInsensitive)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nest inner transformation: %w", err)
	}

	return &ApplyTransformation{
		Type: ApplyTypeNest,
		Nest: &NestTransformation{
			Apply: innerTransforms,
			Alias: alias,
		},
	}, nil
}

// parseFromTransformation parses a from(NavigationPath) transformation.
// Format: from(NavigationPath)
//
// The from transformation changes the current input collection to the related collection
// identified by NavigationPath before applying the subsequent transformation sequence.
// At the SQL-builder layer this is a pass-through; higher-level handlers may act on
// the FromTransformation.Path to rebase the query onto the related collection.
func parseFromTransformation(transStr string) (*ApplyTransformation, error) {
	content := transStr[len("from("):]
	if !strings.HasSuffix(content, ")") {
		return nil, fmt.Errorf("missing closing parenthesis in from")
	}
	content = strings.TrimSpace(content[:len(content)-1])
	if content == "" {
		return nil, fmt.Errorf("from requires a navigation path argument")
	}

	// Split "NavigationPath/transformationSequence" at the first slash
	// that is not inside parentheses.
	// The navigation path is the part inside the from() parens.
	// However, per the spec, the from() call itself only contains the path;
	// subsequent transformations are chained via '/' at the outer level.
	// Here the parser receives just the content of from(...), which is the path.
	path := content

	return &ApplyTransformation{
		Type: ApplyTypeFrom,
		From: &FromTransformation{
			Path: path,
		},
	}, nil
}
