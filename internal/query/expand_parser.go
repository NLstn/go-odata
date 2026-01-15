package query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// parseExpandWithConfig parses the $expand query option with depth tracking
func parseExpandWithConfig(expandStr string, entityMetadata *metadata.EntityMetadata, config *ParserConfig, currentDepth int) ([]ExpandOption, error) {
	// Check depth limit if configured
	if config != nil && config.MaxExpandDepth > 0 && currentDepth >= config.MaxExpandDepth {
		return nil, fmt.Errorf("$expand nesting level (%d) exceeds maximum allowed depth (%d)", currentDepth+1, config.MaxExpandDepth)
	}

	// Split by comma for multiple expands (basic implementation, doesn't handle nested parens)
	parts, err := splitExpandParts(expandStr)
	if err != nil {
		return nil, err
	}
	result := make([]ExpandOption, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		expand, err := parseSingleExpandCoreWithConfig(trimmed, entityMetadata, true, config, currentDepth)
		if err != nil {
			return nil, err
		}
		result = append(result, expand)
	}

	return result, nil
}

// splitExpandParts splits expand string by comma, handling nested parentheses
func splitExpandParts(expandStr string) ([]string, error) {
	result := make([]string, 0)
	var current strings.Builder
	depth := 0
	inString := false

	for i := 0; i < len(expandStr); i++ {
		ch := expandStr[i]
		if ch == '\'' {
			if inString {
				if i+1 < len(expandStr) && expandStr[i+1] == '\'' {
					current.WriteByte(ch)
					current.WriteByte(ch)
					i++
					continue
				}
				inString = false
			} else {
				inString = true
			}
			current.WriteByte(ch)
			continue
		}

		if !inString && ch == '(' {
			depth++
			current.WriteByte(ch)
		} else if !inString && ch == ')' {
			if depth == 0 {
				return nil, fmt.Errorf("invalid $expand syntax: unexpected ')'")
			}
			depth--
			current.WriteByte(ch)
		} else if !inString && ch == ',' && depth == 0 {
			if current.Len() != 0 {
				result = append(result, current.String())
			}
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}

	if inString {
		return nil, fmt.Errorf("invalid $expand syntax: missing closing quote")
	}

	if depth != 0 {
		return nil, fmt.Errorf("invalid $expand syntax: missing ')'")
	}

	if current.Len() != 0 {
		result = append(result, current.String())
	}

	return result, nil
}

// parseSingleExpandCoreWithConfig parses a single expand option with depth tracking
func parseSingleExpandCoreWithConfig(expandStr string, entityMetadata *metadata.EntityMetadata, validateMetadata bool, config *ParserConfig, currentDepth int) (ExpandOption, error) {
	expand := ExpandOption{}

	// Check for nested query options: NavigationProp($select=...,...)
	if idx := strings.Index(expandStr, "("); idx != -1 {
		if !strings.HasSuffix(expandStr, ")") {
			return expand, fmt.Errorf("invalid expand syntax: %s", expandStr)
		}

		expand.NavigationProperty = strings.TrimSpace(expandStr[:idx])
		nestedOptions := expandStr[idx+1 : len(expandStr)-1]

		var targetMetadata *metadata.EntityMetadata
		if validateMetadata && entityMetadata != nil {
			var err error
			targetMetadata, err = entityMetadata.ResolveNavigationTarget(expand.NavigationProperty)
			if err != nil {
				return expand, err
			}
		}

		// Parse nested options
		if err := parseNestedExpandOptionsCoreWithConfig(&expand, nestedOptions, targetMetadata, validateMetadata, config, currentDepth); err != nil {
			return expand, err
		}
	} else {
		expand.NavigationProperty = strings.TrimSpace(expandStr)
	}

	// Validate navigation property exists only if metadata validation is enabled
	if validateMetadata && entityMetadata != nil && !isNavigationProperty(expand.NavigationProperty, entityMetadata) {
		return expand, fmt.Errorf("'%s' is not a valid navigation property", expand.NavigationProperty)
	}

	return expand, nil
}

// parseNestedExpandOptionsCoreWithConfig parses nested query options with depth tracking
func parseNestedExpandOptionsCoreWithConfig(expand *ExpandOption, optionsStr string, targetMetadata *metadata.EntityMetadata, validateMetadata bool, config *ParserConfig, currentDepth int) error {
	// Split by semicolon for different query options
	parts := strings.Split(optionsStr, ";")
	var computedAliases map[string]bool
	var computeValue string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		eqIdx := strings.Index(part, "=")
		if eqIdx == -1 {
			continue
		}

		key := strings.TrimSpace(part[:eqIdx])
		value := strings.TrimSpace(part[eqIdx+1:])
		if strings.EqualFold(key, "$compute") {
			computeValue = value
		}
	}

	if computeValue != "" {
		computedAliases = extractComputeAliasesFromString(computeValue)
	}

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
			if validateMetadata {
				if targetMetadata == nil {
					return fmt.Errorf("navigation target metadata is missing for $select")
				}
				if err := validateExpandSelect(expand.Select, targetMetadata, computedAliases); err != nil {
					return err
				}
			}
		case "$expand":
			if validateMetadata {
				if targetMetadata == nil {
					return fmt.Errorf("navigation target metadata is missing for $expand")
				}
				// Increment depth for nested expand to enforce depth limits
				nestedExpand, err := parseExpandWithConfig(value, targetMetadata, config, currentDepth+1)
				if err != nil {
					return fmt.Errorf("invalid nested $expand: %w", err)
				}
				expand.Expand = nestedExpand
			} else {
				// Parse nested $expand recursively without metadata validation
				// Increment depth for nested expand to enforce depth limits
				nestedExpand, err := parseExpandWithConfig(value, nil, config, currentDepth+1)
				if err != nil {
					return fmt.Errorf("invalid nested $expand: %w", err)
				}
				expand.Expand = nestedExpand
			}
		case "$filter":
			if validateMetadata {
				if targetMetadata == nil {
					return fmt.Errorf("navigation target metadata is missing for $filter")
				}
				maxInClauseSize := 0
				if config != nil {
					maxInClauseSize = config.MaxInClauseSize
				}
				filter, err := parseFilter(value, targetMetadata, computedAliases, maxInClauseSize)
				if err != nil {
					return fmt.Errorf("invalid nested $filter: %w", err)
				}
				expand.Filter = filter
			} else {
				filter, err := parseFilterWithoutMetadata(value)
				if err != nil {
					return fmt.Errorf("invalid nested $filter: %w", err)
				}
				expand.Filter = filter
			}
		case "$orderby":
			if validateMetadata {
				if targetMetadata == nil {
					return fmt.Errorf("navigation target metadata is missing for $orderby")
				}
				orderBy, err := parseOrderBy(value, targetMetadata, computedAliases)
				if err != nil {
					return fmt.Errorf("invalid nested $orderby: %w", err)
				}
				expand.OrderBy = orderBy
			} else {
				// Parse orderby without strict metadata validation
				orderBy, err := parseOrderByWithoutMetadata(value)
				if err != nil {
					return fmt.Errorf("invalid nested $orderby: %w", err)
				}
				expand.OrderBy = orderBy
			}
		case "$top":
			top, err := parseNonNegativeInt(value, "$top")
			if err != nil {
				return fmt.Errorf("invalid nested $top: %w", err)
			}
			expand.Top = &top
		case "$skip":
			skip, err := parseNonNegativeInt(value, "$skip")
			if err != nil {
				return fmt.Errorf("invalid nested $skip: %w", err)
			}
			expand.Skip = &skip
		case "$compute":
			if validateMetadata {
				if targetMetadata == nil {
					return fmt.Errorf("navigation target metadata is missing for $compute")
				}
				maxInClauseSize := 0
				if config != nil {
					maxInClauseSize = config.MaxInClauseSize
				}
				computeTransformation, err := parseCompute("compute("+value+")", targetMetadata, maxInClauseSize)
				if err != nil {
					return fmt.Errorf("invalid nested $compute: %w", err)
				}
				expand.Compute = computeTransformation.Compute
			} else {
				// Parse compute without metadata validation since we don't have the target entity metadata
				compute, err := parseComputeWithoutMetadata(value)
				if err != nil {
					return fmt.Errorf("invalid nested $compute: %w", err)
				}
				expand.Compute = compute
			}
		case "$count":
			// Parse $count option (must be true or false)
			countLower := strings.ToLower(value)
			if countLower == "true" {
				expand.Count = true
			} else if countLower != "false" {
				return fmt.Errorf("invalid nested $count: must be 'true' or 'false'")
			}
		case "$levels":
			// Parse $levels option (positive integer or "max")
			if strings.ToLower(value) == "max" {
				// Use -1 as a sentinel value for "max"
				maxLevels := -1
				expand.Levels = &maxLevels
			} else {
				levels, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid nested $levels: must be a positive integer or 'max'")
				}
				if levels < 1 {
					return fmt.Errorf("invalid nested $levels: must be a positive integer or 'max'")
				}
				expand.Levels = &levels
			}
		}
	}

	return nil
}

func validateExpandSelect(selectedProps []string, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool) error {
	if entityMetadata == nil {
		return fmt.Errorf("entity metadata is nil")
	}

	for _, propName := range selectedProps {
		if computedAliases != nil && computedAliases[propName] {
			continue
		}

		if strings.Contains(propName, "/") {
			parts := strings.SplitN(propName, "/", 2)
			navPropName := strings.TrimSpace(parts[0])
			subPropName := strings.TrimSpace(parts[1])

			if !isNavigationProperty(navPropName, entityMetadata) {
				return fmt.Errorf("property '%s' does not exist in entity type", propName)
			}

			if subPropName != "" {
				targetMetadata, err := entityMetadata.ResolveNavigationTarget(navPropName)
				if err != nil {
					return err
				}
				if !propertyExists(subPropName, targetMetadata) {
					return fmt.Errorf("property '%s' does not exist in entity type", propName)
				}
			}
			continue
		}

		if !propertyExists(propName, entityMetadata) {
			return fmt.Errorf("property '%s' does not exist in entity type", propName)
		}
	}

	return nil
}
