package metadata

import (
	"fmt"
	"reflect"
	"strings"
)

// EntityMetadata holds metadata information about an OData entity
type EntityMetadata struct {
	EntityType    reflect.Type
	EntityName    string
	EntitySetName string
	Properties    []PropertyMetadata
	KeyProperties []PropertyMetadata // Support for composite keys
	KeyProperty   *PropertyMetadata  // Deprecated: Use KeyProperties for single or composite keys, kept for backwards compatibility
	ETagProperty  *PropertyMetadata  // Property used for ETag generation (optional)
}

// PropertyMetadata holds metadata information about an entity property
type PropertyMetadata struct {
	Name              string
	Type              reflect.Type
	FieldName         string
	IsKey             bool
	IsRequired        bool
	JsonName          string
	GormTag           string
	IsNavigationProp  bool
	NavigationTarget  string // Entity type name for navigation properties
	NavigationIsArray bool   // True for collection navigation properties
	IsETag            bool   // True if this property should be used for ETag generation
	// Facets
	MaxLength    int    // Maximum length for string properties
	Precision    int    // Precision for decimal/numeric properties
	Scale        int    // Scale for decimal properties
	DefaultValue string // Default value for the property
	Nullable     *bool  // Explicit nullable override (nil means use default behavior)
	// Referential constraints for navigation properties
	ReferentialConstraints map[string]string // Maps dependent property to principal property
}

// AnalyzeEntity extracts metadata from a Go struct for OData usage
func AnalyzeEntity(entity interface{}) (*EntityMetadata, error) {
	entityType := reflect.TypeOf(entity)

	// Handle pointer types
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	if entityType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("entity must be a struct, got %s", entityType.Kind())
	}

	metadata := initializeMetadata(entityType)

	// Analyze struct fields
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		property := analyzeField(field, metadata)
		metadata.Properties = append(metadata.Properties, property)
	}

	// Validate that we have at least one key property
	if len(metadata.KeyProperties) == 0 {
		return nil, fmt.Errorf("entity %s must have at least one key property (use `odata:\"key\"` tag or name field 'ID')", metadata.EntityName)
	}

	// For backwards compatibility, set KeyProperty to first key if only one key exists
	if len(metadata.KeyProperties) == 1 {
		metadata.KeyProperty = &metadata.KeyProperties[0]
	}

	return metadata, nil
}

// initializeMetadata creates a new EntityMetadata struct with basic information
func initializeMetadata(entityType reflect.Type) *EntityMetadata {
	entityName := entityType.Name()
	entitySetName := pluralize(entityName)

	return &EntityMetadata{
		EntityType:    entityType,
		EntityName:    entityName,
		EntitySetName: entitySetName,
		Properties:    make([]PropertyMetadata, 0),
	}
}

// analyzeField analyzes a single struct field and creates a PropertyMetadata
func analyzeField(field reflect.StructField, metadata *EntityMetadata) PropertyMetadata {
	property := PropertyMetadata{
		Name:      field.Name,
		Type:      field.Type,
		FieldName: field.Name,
		JsonName:  getJsonName(field),
		GormTag:   field.Tag.Get("gorm"),
	}

	// Check if this is a navigation property
	analyzeNavigationProperty(&property, field)

	// Check for OData tags
	analyzeODataTags(&property, field, metadata)

	return property
}

// analyzeNavigationProperty determines if a field is a navigation property
func analyzeNavigationProperty(property *PropertyMetadata, field reflect.StructField) {
	fieldType := field.Type
	isSlice := fieldType.Kind() == reflect.Slice
	if isSlice {
		fieldType = fieldType.Elem()
	}

	// Check if it's a pointer type
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// If it's a struct type and has gorm foreign key tag, it's a navigation property
	if fieldType.Kind() == reflect.Struct {
		gormTag := field.Tag.Get("gorm")
		if strings.Contains(gormTag, "foreignKey") || strings.Contains(gormTag, "references") {
			property.IsNavigationProp = true
			property.NavigationTarget = fieldType.Name()
			property.NavigationIsArray = isSlice

			// Extract referential constraints from GORM tags
			property.ReferentialConstraints = extractReferentialConstraints(gormTag)
		}
	}
}

// extractReferentialConstraints extracts referential constraints from GORM tags
func extractReferentialConstraints(gormTag string) map[string]string {
	constraints := make(map[string]string)

	// Parse foreignKey and references from gorm tag
	// Format: "foreignKey:UserID;references:ID"
	var foreignKey, references string

	parts := strings.Split(gormTag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "foreignKey:") {
			foreignKey = strings.TrimPrefix(part, "foreignKey:")
		} else if strings.HasPrefix(part, "references:") {
			references = strings.TrimPrefix(part, "references:")
		}
	}

	if foreignKey != "" && references != "" {
		constraints[foreignKey] = references
	}

	return constraints
}

// analyzeODataTags processes OData-specific tags on a field
func analyzeODataTags(property *PropertyMetadata, field reflect.StructField, metadata *EntityMetadata) {
	if odataTag := field.Tag.Get("odata"); odataTag != "" {
		// Parse tag as comma-separated key-value pairs
		parts := strings.Split(odataTag, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			processODataTagPart(property, part, metadata)
		}
	}

	// Auto-detect key if no explicit key is set and field name is "ID"
	if len(metadata.KeyProperties) == 0 && field.Name == "ID" {
		property.IsKey = true
		metadata.KeyProperties = append(metadata.KeyProperties, *property)
	}
}

// processODataTagPart processes a single OData tag part
func processODataTagPart(property *PropertyMetadata, part string, metadata *EntityMetadata) {
	switch {
	case part == "key":
		property.IsKey = true
		metadata.KeyProperties = append(metadata.KeyProperties, *property)
	case part == "etag":
		property.IsETag = true
		metadata.ETagProperty = property
	case part == "required":
		property.IsRequired = true
	case strings.HasPrefix(part, "maxlength="):
		processIntFacet(part, "maxlength=", &property.MaxLength)
	case strings.HasPrefix(part, "precision="):
		processIntFacet(part, "precision=", &property.Precision)
	case strings.HasPrefix(part, "scale="):
		processIntFacet(part, "scale=", &property.Scale)
	case strings.HasPrefix(part, "default="):
		property.DefaultValue = strings.TrimPrefix(part, "default=")
	case part == "nullable":
		nullable := true
		property.Nullable = &nullable
	case part == "nullable=false":
		nullable := false
		property.Nullable = &nullable
	}
}

// processIntFacet processes an integer facet from an OData tag
func processIntFacet(part, prefix string, target *int) {
	if val := strings.TrimPrefix(part, prefix); val != "" {
		if parsed, err := parseInt(val); err == nil {
			*target = parsed
		}
	}
}

// getJsonName extracts the JSON field name from struct tags
func getJsonName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return field.Name
	}

	// Handle json:",omitempty" or json:"fieldname,omitempty"
	parts := strings.Split(jsonTag, ",")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return field.Name
}

// pluralize creates a simple pluralized form of the entity name
// This is a basic implementation - could be enhanced with proper pluralization library
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
	switch r {
	case 'a', 'e', 'i', 'o', 'u', 'A', 'E', 'I', 'O', 'U':
		return true
	default:
		return false
	}
}

// parseInt parses a string to an integer
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
