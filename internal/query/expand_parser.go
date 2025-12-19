package query

import (
	"fmt"
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

		// Parse nested options
		if err := parseNestedExpandOptionsCore(&expand, nestedOptions, entityMetadata, validateMetadata); err != nil {
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
func parseNestedExpandOptionsCore(expand *ExpandOption, optionsStr string, entityMetadata *metadata.EntityMetadata, validateMetadata bool) error {
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
		case "$expand":
			// Parse nested $expand recursively without metadata validation
			// as we don't have easy access to the target entity metadata
			nestedExpand, err := parseExpandWithoutMetadata(value)
			if err != nil {
				return fmt.Errorf("invalid nested $expand: %w", err)
			}
			expand.Expand = nestedExpand
		case "$filter":
			if validateMetadata && entityMetadata != nil {
				// Get the navigation property metadata to find the target entity type
				navProp := findNavigationProperty(expand.NavigationProperty, entityMetadata)
				if navProp == nil {
					return fmt.Errorf("navigation property '%s' not found", expand.NavigationProperty)
				}
			}
			// Parse without metadata validation for both cases
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
