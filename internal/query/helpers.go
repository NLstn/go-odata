package query

import (
	"reflect"
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
				// Get the related table name using the entity type
				// This respects custom TableName() methods
				relatedTableName := getTableNameFromReflectType(navProp.Type)
				
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

	if column := extractColumnFromGormTag(prop.GormTag); column != "" {
		return prefix + column
	}

	return prefix + toSnakeCase(prop.Name)
}

func extractColumnFromGormTag(gormTag string) string {
	if gormTag == "" {
		return ""
	}
	parts := strings.Split(gormTag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "column:") {
			return strings.TrimPrefix(part, "column:")
		}
	}
	return ""
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

// pluralize creates a simple pluralized form of a word
// This follows the same rules as GORM's default table naming
func pluralize(word string) string {
	if word == "" {
		return word
	}

	// Simple pluralization rules
	switch {
	case strings.HasSuffix(word, "y") && len(word) > 1 && !isVowel(rune(word[len(word)-2])):
		// Only change y to ies if preceded by a consonant (e.g., "Category" -> "Categories")
		// If preceded by a vowel, just add s (e.g., "Key" -> "Keys")
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") || strings.HasSuffix(word, "z") ||
		strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh"):
		return word + "es"
	default:
		return word + "s"
	}
}

// isVowel checks if a rune is a vowel
func isVowel(r rune) bool {
	lower := strings.ToLower(string(r))
	return lower == "a" || lower == "e" || lower == "i" || lower == "o" || lower == "u"
}

// getTableNameFromReflectType returns the table name for a given entity type
// This respects custom TableName() methods on the entity without requiring a db connection
func getTableNameFromReflectType(entityType reflect.Type) string {
	// Handle pointer types
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	
	// Create a zero value instance and check if it implements TableName()
	instance := reflect.New(entityType).Interface()
	
	// Check if the entity implements the TableName() method
	if tabler, ok := instance.(interface{ TableName() string }); ok {
		return tabler.TableName()
	}
	
	// Fallback to default GORM naming (snake_case pluralization)
	return toSnakeCase(pluralize(entityType.Name()))
}
