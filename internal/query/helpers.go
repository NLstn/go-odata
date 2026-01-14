package query

import (
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

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

// propertyExists checks if a property exists in the entity metadata
func propertyExists(propertyName string, entityMetadata *metadata.EntityMetadata) bool {
	if entityMetadata == nil {
		return false
	}

	// Check if this is a single-entity navigation property path
	// Per OData v4 spec 5.1.1.15, single-entity navigation properties can be accessed directly
	if entityMetadata.IsSingleEntityNavigationPath(propertyName) {
		return true
	}

	_, _, err := entityMetadata.ResolvePropertyPath(propertyName)
	return err == nil
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
	// Handle $it - refers to the current instance (OData v4 spec 5.1.1.11.4)
	// Used in isof() function to check the type of the current entity
	if propertyName == "$it" {
		return "$it"
	}

	if entityMetadata == nil {
		return toSnakeCase(propertyName)
	}

	// Check if this is a single-entity navigation property path (e.g., "Team/ClubID")
	// Per OData v4 spec 5.1.1.15, properties of entities with cardinality 0..1 or 1 can be accessed directly
	// Note: IsSingleEntityNavigationPath validates the navigation property but not the target property.
	// If the target property doesn't exist, the database will return an error with the column name.
	if entityMetadata.IsSingleEntityNavigationPath(propertyName) {
		segments := strings.Split(propertyName, "/")
		if len(segments) >= 2 {
			navPropName := strings.TrimSpace(segments[0])
			targetPropertyName := strings.TrimSpace(segments[1])

			navProp := entityMetadata.FindNavigationProperty(navPropName)
			if navProp != nil {
				// Get the related table name from cached metadata
				// This was computed once during entity registration and respects custom TableName() methods
				relatedTableName := navProp.NavigationTargetTableName

				// Return qualified column name: related_table.column_name
				return relatedTableName + "." + toSnakeCase(targetPropertyName)
			}
		}
	}

	prop, prefix, err := entityMetadata.ResolvePropertyPath(propertyName)
	if err != nil || prop == nil {
		// Fallback to the last segment when metadata cannot resolve the path
		if strings.Contains(propertyName, "/") {
			parts := strings.Split(propertyName, "/")
			propertyName = parts[len(parts)-1]
		}
		return toSnakeCase(propertyName)
	}

	// Use cached column name from metadata
	return prefix + prop.ColumnName
}

// findNavigationProperty finds a navigation property in the entity metadata
func findNavigationProperty(propName string, entityMetadata *metadata.EntityMetadata) *metadata.PropertyMetadata {
	if entityMetadata == nil {
		return nil
	}
	return entityMetadata.FindNavigationProperty(propName)
}

// toSnakeCase converts a camelCase or PascalCase string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if the previous character was lowercase or if this is the start of a new word
			// For "ProductID", we want "product_id" not "product_i_d"
			prevRune := rune(s[i-1])
			if prevRune >= 'a' && prevRune <= 'z' {
				result.WriteRune('_')
			} else if i < len(s)-1 {
				// Check if next character is lowercase (e.g., "XMLParser" -> "xml_parser")
				nextRune := rune(s[i+1])
				if nextRune >= 'a' && nextRune <= 'z' {
					result.WriteRune('_')
				}
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// MergeFilterExpressions combines two filter expressions using a logical AND.
func MergeFilterExpressions(left *FilterExpression, right *FilterExpression) *FilterExpression {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	return &FilterExpression{
		Left:    left,
		Right:   right,
		Logical: LogicalAnd,
	}
}

// ParseFilterExpression parses a raw filter string into a filter expression with metadata validation.
func ParseFilterExpression(filterStr string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	return parseFilter(filterStr, entityMetadata, map[string]bool{}, 0)
}
