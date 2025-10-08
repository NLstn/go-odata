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
	KeyProperty   *PropertyMetadata
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

	// Validate that we have a key property
	if metadata.KeyProperty == nil {
		return nil, fmt.Errorf("entity %s must have a key property (use `odata:\"key\"` tag or name field 'ID')", metadata.EntityName)
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
		}
	}
}

// analyzeODataTags processes OData-specific tags on a field
func analyzeODataTags(property *PropertyMetadata, field reflect.StructField, metadata *EntityMetadata) {
	if odataTag := field.Tag.Get("odata"); odataTag != "" {
		if strings.Contains(odataTag, "key") {
			property.IsKey = true
			metadata.KeyProperty = property
		}
		if strings.Contains(odataTag, "required") {
			property.IsRequired = true
		}
	}

	// Auto-detect key if no explicit key is set and field name is "ID"
	if metadata.KeyProperty == nil && field.Name == "ID" {
		property.IsKey = true
		metadata.KeyProperty = property
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
	case strings.HasSuffix(word, "y"):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") || strings.HasSuffix(word, "z") ||
		strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh"):
		return word + "es"
	default:
		return word + "s"
	}
}
