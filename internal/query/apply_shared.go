package query

import (
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// applySelectToExpandedEntity applies select to an expanded navigation property (single entity or collection)
// This is a simplified version that doesn't require full metadata - it works with reflection
func applySelectToExpandedEntity(expandedValue interface{}, selectedProps []string) interface{} {
	if len(selectedProps) == 0 || expandedValue == nil {
		return expandedValue
	}

	selectedPropMap := make(map[string]bool)
	for _, propName := range selectedProps {
		selectedPropMap[strings.TrimSpace(propName)] = true
	}

	val := reflect.ValueOf(expandedValue)

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return expandedValue
		}
		val = val.Elem()
	}

	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		resultSlice := make([]map[string]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			itemVal := val.Index(i)
			resultSlice[i] = filterEntityFields(itemVal, selectedPropMap)
		}
		return resultSlice
	}

	if val.Kind() == reflect.Struct {
		return filterEntityFields(val, selectedPropMap)
	}

	return expandedValue
}

// filterEntityFields filters struct fields based on selected properties map
func filterEntityFields(entityVal reflect.Value, selectedPropMap map[string]bool) map[string]interface{} {
	filtered := make(map[string]interface{})
	entityType := entityVal.Type()

	idFields := []string{"ID", "Id", "id"}

	for i := 0; i < entityVal.NumField(); i++ {
		field := entityType.Field(i)
		fieldVal := entityVal.Field(i)

		if !fieldVal.CanInterface() {
			continue
		}

		jsonName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				jsonName = parts[0]
			}
		}

		isSelected := selectedPropMap[field.Name] || selectedPropMap[jsonName]
		isKeyField := false
		for _, keyName := range idFields {
			if field.Name == keyName {
				isKeyField = true
				break
			}
		}

		if odataTag := field.Tag.Get("odata"); odataTag != "" {
			if strings.Contains(odataTag, "key") {
				isKeyField = true
			}
		}

		if isSelected || isKeyField {
			filtered[jsonName] = fieldVal.Interface()
		}
	}

	return filtered
}

// findProperty finds a property by name or JSON name in the entity metadata
func findProperty(propName string, entityMetadata *metadata.EntityMetadata) *metadata.PropertyMetadata {
	if entityMetadata == nil {
		return nil
	}
	return entityMetadata.FindProperty(propName)
}

// sanitizeIdentifier sanitizes a user-provided identifier to prevent SQL injection.
// It ensures that the identifier contains only safe characters (alphanumeric and underscore).
// Returns an empty string if the identifier contains invalid characters.
// Special OData identifiers like "$it" are allowed.
func sanitizeIdentifier(identifier string) string {
	if identifier == "" {
		return ""
	}

	if identifier == "$it" || identifier == "$count" {
		return identifier
	}

	if len(identifier) > 1 && identifier[0] == '$' {
		for i, ch := range identifier[1:] {
			if i == 0 {
				if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && ch != '_' {
					return ""
				}
			} else {
				if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
					return ""
				}
			}
		}
		return identifier
	}

	for i, ch := range identifier {
		if i == 0 {
			if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && ch != '_' {
				return ""
			}
		} else {
			if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
				return ""
			}
		}
	}

	upper := strings.ToUpper(identifier)
	reservedKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE",
		"UNION", "JOIN", "WHERE", "FROM", "INTO", "VALUES", "SET", "EXEC", "EXECUTE",
	}
	for _, keyword := range reservedKeywords {
		if upper == keyword {
			return ""
		}
	}

	return identifier
}
