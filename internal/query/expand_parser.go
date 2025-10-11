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
