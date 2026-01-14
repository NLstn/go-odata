package query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

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

		expand, err := parseSingleExpandCore(trimmed, entityMetadata, true)
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

// parseSingleExpandCore parses a single expand option with optional metadata validation
func parseSingleExpandCore(expandStr string, entityMetadata *metadata.EntityMetadata, validateMetadata bool) (ExpandOption, error) {
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
		if err := parseNestedExpandOptionsCore(&expand, nestedOptions, targetMetadata, validateMetadata); err != nil {
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

// parseNestedExpandOptionsCore parses nested query options with optional metadata validation
func parseNestedExpandOptionsCore(expand *ExpandOption, optionsStr string, targetMetadata *metadata.EntityMetadata, validateMetadata bool) error {
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
				nestedExpand, err := parseExpand(value, targetMetadata)
				if err != nil {
					return fmt.Errorf("invalid nested $expand: %w", err)
				}
				expand.Expand = nestedExpand
			} else {
				// Parse nested $expand recursively without metadata validation
				nestedExpand, err := parseExpandWithoutMetadata(value)
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
				filter, err := parseFilter(value, targetMetadata, computedAliases)
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
		case "$compute":
			if validateMetadata {
				if targetMetadata == nil {
					return fmt.Errorf("navigation target metadata is missing for $compute")
				}
				computeTransformation, err := parseCompute("compute("+value+")", targetMetadata)
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

	// TODO: Implement support for $count and $levels in nested expand
	// For now, return an error if these unsupported options are used
	if expand.Count {
		return fmt.Errorf("nested $count in $expand is not yet implemented")
	}
	if expand.Levels != nil {
		return fmt.Errorf("nested $levels in $expand is not yet implemented")
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

// parseExpandWithoutMetadata parses expand without entity metadata validation
// This is used for nested expand levels where we don't have easy access to target entity metadata
func parseExpandWithoutMetadata(expandStr string) ([]ExpandOption, error) {
	// Split by comma for multiple expands
	parts := splitExpandParts(expandStr)
	result := make([]ExpandOption, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		expand, err := parseSingleExpandCore(trimmed, nil, false)
		if err != nil {
			return nil, err
		}
		result = append(result, expand)
	}

	return result, nil
}
