package query

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// parseOrderBy parses the $orderby query option
func parseOrderBy(orderByStr string, entityMetadata *metadata.EntityMetadata, computedAliases map[string]bool) ([]OrderByItem, error) {
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

		// Validate property exists (either in entity metadata or as a computed alias)
		if !computedAliases[item.Property] {
			if entityMetadata != nil && entityMetadata.IsSingleEntityNavigationPath(item.Property) {
				segments := strings.Split(item.Property, "/")
				if len(segments) != 2 {
					return nil, fmt.Errorf("property '%s' does not exist", item.Property)
				}

				navPropName := strings.TrimSpace(segments[0])
				targetProperty := strings.TrimSpace(segments[1])

				targetMetadata, err := entityMetadata.ResolveNavigationTarget(navPropName)
				if err != nil {
					return nil, fmt.Errorf("property '%s' does not exist", item.Property)
				}

				if !propertyExists(targetProperty, targetMetadata) {
					return nil, fmt.Errorf("property '%s' does not exist", item.Property)
				}
			} else if !propertyExists(item.Property, entityMetadata) {
				return nil, fmt.Errorf("property '%s' does not exist", item.Property)
			}
		}

		result = append(result, item)
	}

	return result, nil
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
