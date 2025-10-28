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
	IsSingleton   bool               // True if this is a singleton (single instance accessible by name)
	SingletonName string             // Name of the singleton (if IsSingleton is true)
	// Hooks defines which lifecycle hooks are available on this entity
	Hooks struct {
		HasBeforeCreate         bool
		HasAfterCreate          bool
		HasBeforeUpdate         bool
		HasAfterUpdate          bool
		HasBeforeDelete         bool
		HasAfterDelete          bool
		HasBeforeReadCollection bool
		HasAfterReadCollection  bool
		HasBeforeReadEntity     bool
		HasAfterReadEntity      bool
	}
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
	IsComplexType     bool   // True if this property is a complex type (embedded struct)
	// Facets
	MaxLength    int    // Maximum length for string properties
	Precision    int    // Precision for decimal/numeric properties
	Scale        int    // Scale for decimal properties
	DefaultValue string // Default value for the property
	Nullable     *bool  // Explicit nullable override (nil means use default behavior)
	// Referential constraints for navigation properties
	ReferentialConstraints map[string]string // Maps dependent property to principal property
	// Search properties
	IsSearchable     bool    // True if this property should be considered in $search
	SearchFuzziness  int     // Fuzziness level for search (default 1, meaning exact match)
	SearchSimilarity float64 // Similarity score for search (0.0 to 1.0, where 0.95 means 95% similar)
	// Enum properties
	IsEnum       bool   // True if this property is an enum type
	EnumTypeName string // Name of the enum type (for metadata generation)
	IsFlags      bool   // True if this enum supports flag combinations (bitwise operations)
	// Binary properties
	ContentType string // MIME type for binary properties (e.g., "image/svg+xml"), used when serving /$value
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

		property, err := analyzeField(field, metadata)
		if err != nil {
			return nil, fmt.Errorf("error analyzing field %s: %w", field.Name, err)
		}
		metadata.Properties = append(metadata.Properties, property)
	}

	// Validate that we have at least one key property
	if len(metadata.KeyProperties) == 0 {
		return nil, fmt.Errorf("entity %s must have at least one key property (use `odata:\"key\"` tag or name field 'ID')", metadata.EntityName)
	}

	// Validate that no field has both fuzziness and similarity defined
	for _, prop := range metadata.Properties {
		if prop.SearchFuzziness > 0 && prop.SearchSimilarity > 0 {
			return nil, fmt.Errorf("property %s cannot have both fuzziness and similarity defined; use one or the other", prop.Name)
		}
		// Validate similarity range (0.0 to 1.0) - only check if similarity was set (non-zero)
		if prop.SearchSimilarity != 0 && (prop.SearchSimilarity < 0.0 || prop.SearchSimilarity > 1.0) {
			return nil, fmt.Errorf("property %s has invalid similarity value %.2f; must be between 0.0 and 1.0", prop.Name, prop.SearchSimilarity)
		}
	}

	// For backwards compatibility, set KeyProperty to first key if only one key exists
	if len(metadata.KeyProperties) == 1 {
		metadata.KeyProperty = &metadata.KeyProperties[0]
	}

	// Detect available lifecycle hooks
	detectHooks(metadata)

	return metadata, nil
}

// AnalyzeSingleton extracts metadata from a Go struct for OData singleton usage
// Singletons are single instances of an entity type that can be accessed directly by name
func AnalyzeSingleton(entity interface{}, singletonName string) (*EntityMetadata, error) {
	entityType := reflect.TypeOf(entity)

	// Handle pointer types
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	if entityType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("singleton must be a struct, got %s", entityType.Kind())
	}

	metadata := initializeSingletonMetadata(entityType, singletonName)

	// Analyze struct fields
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		property, err := analyzeField(field, metadata)
		if err != nil {
			return nil, fmt.Errorf("error analyzing field %s: %w", field.Name, err)
		}
		metadata.Properties = append(metadata.Properties, property)
	}

	// Singletons don't require key properties for URL addressing,
	// but may still have them for database operations
	// We'll be lenient and allow singletons without keys
	if len(metadata.KeyProperties) == 0 {
		// Auto-detect key if field name is "ID"
		for i := range metadata.Properties {
			if metadata.Properties[i].Name == "ID" {
				metadata.Properties[i].IsKey = true
				metadata.KeyProperties = append(metadata.KeyProperties, metadata.Properties[i])
				break
			}
		}
	}

	// For backwards compatibility, set KeyProperty to first key if only one key exists
	if len(metadata.KeyProperties) == 1 {
		metadata.KeyProperty = &metadata.KeyProperties[0]
	}

	// Detect available lifecycle hooks
	detectHooks(metadata)

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
		IsSingleton:   false,
	}
}

// initializeSingletonMetadata creates a new EntityMetadata struct for a singleton
func initializeSingletonMetadata(entityType reflect.Type, singletonName string) *EntityMetadata {
	entityName := entityType.Name()

	return &EntityMetadata{
		EntityType:    entityType,
		EntityName:    entityName,
		EntitySetName: singletonName, // For singletons, we use the singleton name
		SingletonName: singletonName,
		Properties:    make([]PropertyMetadata, 0),
		IsSingleton:   true,
	}
}

// analyzeField analyzes a single struct field and creates a PropertyMetadata
func analyzeField(field reflect.StructField, metadata *EntityMetadata) (PropertyMetadata, error) {
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

	// Auto-detect nullability based on Go type and GORM tags
	// This runs after OData tags so explicit odata:"nullable" takes precedence
	if err := autoDetectNullability(&property); err != nil {
		return PropertyMetadata{}, err
	}

	return property, nil
}

// analyzeNavigationProperty determines if a field is a navigation property or complex type
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

	// If it's a struct type, determine if it's navigation property or complex type
	if fieldType.Kind() == reflect.Struct {
		gormTag := field.Tag.Get("gorm")

		// Check if it's a navigation property (has foreign key, references, or many2many)
		if strings.Contains(gormTag, "foreignKey") || strings.Contains(gormTag, "references") || strings.Contains(gormTag, "many2many") {
			property.IsNavigationProp = true
			property.NavigationTarget = fieldType.Name()
			property.NavigationIsArray = isSlice

			// Extract referential constraints from GORM tags (only for foreignKey/references)
			if strings.Contains(gormTag, "foreignKey") || strings.Contains(gormTag, "references") {
				property.ReferentialConstraints = extractReferentialConstraints(gormTag)
			}
		} else if strings.Contains(gormTag, "embedded") {
			// It's a complex type (embedded struct without foreign keys)
			property.IsComplexType = true
		}
	}
}

// extractReferentialConstraints extracts referential constraints from GORM tags
func extractReferentialConstraints(gormTag string) map[string]string {
	constraints := make(map[string]string)

	// Parse foreignKey and references from gorm tag
	// Format: "foreignKey:UserID;references:ID" or just "foreignKey:UserID" (references defaults to "ID")
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

	// If foreignKey is specified but references is not, GORM defaults to "ID"
	if foreignKey != "" {
		if references == "" {
			references = "ID"
		}
		constraints[foreignKey] = references
	}

	return constraints
}

// analyzeODataTags processes OData-specific tags on a field
func analyzeODataTags(property *PropertyMetadata, field reflect.StructField, metadata *EntityMetadata) {
	if odataTag := field.Tag.Get("odata"); odataTag != "" {
		// Check if similarity is defined in the tag to avoid setting default fuzziness
		hasSimilarity := strings.Contains(odataTag, "similarity=")

		// Parse tag as comma-separated key-value pairs
		parts := strings.Split(odataTag, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			processODataTagPart(property, part, metadata, hasSimilarity)
		}
	}

	// Auto-detect key if no explicit key is set and field name is "ID"
	if len(metadata.KeyProperties) == 0 && field.Name == "ID" {
		property.IsKey = true
		metadata.KeyProperties = append(metadata.KeyProperties, *property)
	}
}

// processODataTagPart processes a single OData tag part
func processODataTagPart(property *PropertyMetadata, part string, metadata *EntityMetadata, hasSimilarity bool) {
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
	case part == "searchable":
		property.IsSearchable = true
		// Default fuzziness is 1 (exact match) only if similarity is not going to be set
		if property.SearchFuzziness == 0 && !hasSimilarity {
			property.SearchFuzziness = 1
		}
	case strings.HasPrefix(part, "fuzziness="):
		processIntFacet(part, "fuzziness=", &property.SearchFuzziness)
		// If fuzziness is set, also mark as searchable
		if property.SearchFuzziness > 0 {
			property.IsSearchable = true
		}
	case strings.HasPrefix(part, "similarity="):
		processFloatFacet(part, "similarity=", &property.SearchSimilarity)
		// If similarity is set, also mark as searchable
		if property.SearchSimilarity > 0 {
			property.IsSearchable = true
		}
	case strings.HasPrefix(part, "enum="):
		property.IsEnum = true
		property.EnumTypeName = strings.TrimPrefix(part, "enum=")
	case part == "flags":
		property.IsFlags = true
		// If flags is set without enum, we still mark it as enum
		if !property.IsEnum {
			property.IsEnum = true
		}
	case strings.HasPrefix(part, "contenttype="):
		property.ContentType = strings.TrimPrefix(part, "contenttype=")
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

// processFloatFacet processes a float facet from an OData tag
func processFloatFacet(part, prefix string, target *float64) {
	if val := strings.TrimPrefix(part, prefix); val != "" {
		if parsed, err := parseFloat(val); err == nil {
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

// detectHooks checks if the entity type has any lifecycle hook methods
func detectHooks(metadata *EntityMetadata) {
	entityType := metadata.EntityType

	// Check for both value and pointer receivers
	valueType := entityType
	ptrType := reflect.PointerTo(entityType)

	// Check BeforeCreate
	if hasMethod(valueType, "BeforeCreate") || hasMethod(ptrType, "BeforeCreate") {
		metadata.Hooks.HasBeforeCreate = true
	}

	// Check AfterCreate
	if hasMethod(valueType, "AfterCreate") || hasMethod(ptrType, "AfterCreate") {
		metadata.Hooks.HasAfterCreate = true
	}

	// Check BeforeUpdate
	if hasMethod(valueType, "BeforeUpdate") || hasMethod(ptrType, "BeforeUpdate") {
		metadata.Hooks.HasBeforeUpdate = true
	}

	// Check AfterUpdate
	if hasMethod(valueType, "AfterUpdate") || hasMethod(ptrType, "AfterUpdate") {
		metadata.Hooks.HasAfterUpdate = true
	}

	// Check BeforeDelete
	if hasMethod(valueType, "BeforeDelete") || hasMethod(ptrType, "BeforeDelete") {
		metadata.Hooks.HasBeforeDelete = true
	}

	// Check AfterDelete
	if hasMethod(valueType, "AfterDelete") || hasMethod(ptrType, "AfterDelete") {
		metadata.Hooks.HasAfterDelete = true
	}

	// Check BeforeReadCollection
	if hasMethod(valueType, "BeforeReadCollection") || hasMethod(ptrType, "BeforeReadCollection") {
		metadata.Hooks.HasBeforeReadCollection = true
	}

	// Check AfterReadCollection
	if hasMethod(valueType, "AfterReadCollection") || hasMethod(ptrType, "AfterReadCollection") {
		metadata.Hooks.HasAfterReadCollection = true
	}

	// Check BeforeReadEntity
	if hasMethod(valueType, "BeforeReadEntity") || hasMethod(ptrType, "BeforeReadEntity") {
		metadata.Hooks.HasBeforeReadEntity = true
	}

	// Check AfterReadEntity
	if hasMethod(valueType, "AfterReadEntity") || hasMethod(ptrType, "AfterReadEntity") {
		metadata.Hooks.HasAfterReadEntity = true
	}
}

// hasMethod checks if a type has a method with the given name
func hasMethod(t reflect.Type, methodName string) bool {
	_, found := t.MethodByName(methodName)
	return found
}

// parseInt parses a string to an integer
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// parseFloat parses a string to a float64
func parseFloat(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

// isTypeNullable checks if a Go type can represent null values
func isTypeNullable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice:
		// These types can be nil in Go
		return true
	default:
		// Value types like int, bool, time.Time cannot be nil
		return false
	}
}

// hasGormNotNull checks if a GORM tag contains "not null" constraint
func hasGormNotNull(gormTag string) bool {
	return strings.Contains(gormTag, "not null")
}

// hasGormDefault checks if a GORM tag contains a default value
func hasGormDefault(gormTag string) bool {
	return strings.Contains(gormTag, "default:")
}

// autoDetectNullability automatically sets the Nullable field based on Go type and GORM constraints
// This ensures the OData metadata accurately reflects whether a property can actually be null
func autoDetectNullability(property *PropertyMetadata) error {
	// Skip navigation properties - they have different nullability semantics
	if property.IsNavigationProp {
		return nil
	}

	// If there's an explicit odata:"nullable" or odata:"nullable=false" tag, validate it
	if property.Nullable != nil {
		if *property.Nullable && !isTypeNullable(property.Type) {
			// User explicitly marked as nullable but type doesn't support it
			return fmt.Errorf("property %s is marked as nullable with odata:\"nullable\" tag, but has non-nullable Go type %s (use *%s to make it nullable)",
				property.Name, property.Type, property.Type)
		}
		return nil
	}

	// Auto-detect nullability based on Go type and GORM constraints
	// A property can only be nullable if:
	// 1. The Go type can represent null (pointer, slice, map, interface)
	// 2. GORM doesn't enforce "not null"
	// 3. It's not a key or required field

	canBeNull := isTypeNullable(property.Type)
	hasNotNull := hasGormNotNull(property.GormTag)
	hasDefault := hasGormDefault(property.GormTag)

	// If the type cannot be null in Go, mark as non-nullable
	if !canBeNull {
		nullable := false
		property.Nullable = &nullable
		return nil
	}

	// If GORM enforces "not null", mark as non-nullable
	if hasNotNull {
		nullable := false
		property.Nullable = &nullable
		return nil
	}

	// If it has a default value and is not a pointer type, it's effectively non-nullable
	// (GORM will use the default instead of null)
	if hasDefault && !canBeNull {
		nullable := false
		property.Nullable = &nullable
		return nil
	}

	// For pointer types without "not null", leave nullable as nil
	// The metadata handler will decide based on IsRequired and IsKey
	return nil
}

// FindProperty returns the property metadata matching the provided name or JSON name.
// Returns nil if no property matches.
func (metadata *EntityMetadata) FindProperty(name string) *PropertyMetadata {
	if metadata == nil {
		return nil
	}

	for i := range metadata.Properties {
		prop := &metadata.Properties[i]
		if prop.Name == name || prop.JsonName == name {
			return prop
		}
	}

	return nil
}

// FindNavigationProperty returns the metadata for the requested navigation property.
// Returns nil if the property does not exist or is not a navigation property.
func (metadata *EntityMetadata) FindNavigationProperty(name string) *PropertyMetadata {
	prop := metadata.FindProperty(name)
	if prop != nil && prop.IsNavigationProp {
		return prop
	}
	return nil
}

// FindStructuralProperty returns metadata for structural properties (non-navigation, non-complex types).
// Returns nil if the property does not exist or is not a structural property.
func (metadata *EntityMetadata) FindStructuralProperty(name string) *PropertyMetadata {
	prop := metadata.FindProperty(name)
	if prop != nil && !prop.IsNavigationProp && !prop.IsComplexType {
		return prop
	}
	return nil
}

// FindComplexTypeProperty returns metadata for complex type properties.
// Returns nil if the property does not exist or is not a complex type.
func (metadata *EntityMetadata) FindComplexTypeProperty(name string) *PropertyMetadata {
	prop := metadata.FindProperty(name)
	if prop != nil && prop.IsComplexType {
		return prop
	}
	return nil
}
