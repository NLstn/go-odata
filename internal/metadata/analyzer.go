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
	TableName     string // Database table name (computed once, respects custom TableName() methods)
	Properties    []PropertyMetadata
	KeyProperties []PropertyMetadata // Support for composite keys
	KeyProperty   *PropertyMetadata  // Deprecated: Use KeyProperties for single or composite keys, kept for backwards compatibility
	ETagProperty  *PropertyMetadata  // Property used for ETag generation (optional)
	IsSingleton   bool               // True if this is a singleton (single instance accessible by name)
	SingletonName string             // Name of the singleton (if IsSingleton is true)
	HasStream     bool               // True if this is a media entity (has a media stream)
	IsVirtual     bool               // True if this is a virtual entity (no database backing store, requires overwrite handlers)
	// ChangeTrackingEnabled indicates whether $deltatoken and change tracking responses are enabled for this entity set
	ChangeTrackingEnabled bool
	StreamProperties      []PropertyMetadata // Named stream properties on this entity
	// DisabledMethods contains HTTP methods that are not allowed for this entity
	DisabledMethods map[string]bool
	// DefaultMaxTop is the default maximum number of results to return if no explicit $top is set
	DefaultMaxTop *int
	// TypeDiscriminator holds information about the type discriminator property for polymorphic entities
	// This is used for isof() filter function and type casting in URLs
	TypeDiscriminator *TypeDiscriminatorInfo
	// Hooks defines which lifecycle hooks are available on this entity
	Hooks struct {
		HasODataBeforeCreate         bool
		HasODataAfterCreate          bool
		HasODataBeforeUpdate         bool
		HasODataAfterUpdate          bool
		HasODataBeforeDelete         bool
		HasODataAfterDelete          bool
		HasODataBeforeReadCollection bool
		HasODataAfterReadCollection  bool
		HasODataBeforeReadEntity     bool
		HasODataAfterReadEntity      bool
	}
	// Annotations holds OData vocabulary annotations for this entity type
	Annotations           *AnnotationCollection
	EntitySetAnnotations  *AnnotationCollection
	SingletonAnnotations  *AnnotationCollection
	entitiesRegistry      map[string]*EntityMetadata
	navigationTargetIndex map[string]*EntityMetadata // Index for fast navigation target lookup by EntityName or EntitySetName
}

// TypeDiscriminatorInfo holds metadata about the type discriminator property
type TypeDiscriminatorInfo struct {
	ColumnName string // Database column name for the discriminator
	JsonName   string // JSON field name for the discriminator
	FieldName  string // Go struct field name
}

// PropertyMetadata holds metadata information about an entity property
type PropertyMetadata struct {
	Name                      string
	Type                      reflect.Type
	FieldName                 string
	ColumnName                string // Database column name (computed once, respects GORM column: tags)
	IsKey                     bool
	KeyGenerator              string
	DatabaseGenerated         bool
	IsRequired                bool
	JsonName                  string
	GormTag                   string
	IsNavigationProp          bool
	NavigationTarget          string // Entity type name for navigation properties
	NavigationTargetTableName string // Database table name for navigation target (computed once)
	ForeignKeyColumnName      string // Foreign key column name for navigation properties (computed once)
	NavigationIsArray         bool   // True for collection navigation properties
	IsETag                    bool   // True if this property should be used for ETag generation
	IsComplexType             bool   // True if this property is a complex type (embedded struct)
	EmbeddedPrefix            string
	ComplexTypeFields         map[string]*PropertyMetadata
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
	IsEnum             bool         // True if this property is an enum type
	EnumTypeName       string       // Name of the enum type (for metadata generation)
	IsFlags            bool         // True if this enum supports flag combinations (bitwise operations)
	EnumMembers        []EnumMember // Enum members for metadata generation
	EnumUnderlyingType string       // Underlying EDM type (e.g., Edm.Int32)
	EnumType           reflect.Type // Underlying Go enum type
	// Binary properties
	ContentType string // MIME type for binary properties (e.g., "image/svg+xml"), used when serving /$value
	// Stream properties
	IsStream               bool   // True if this is a stream property (Edm.Stream type)
	StreamContentTypeField string // Name of the field containing the content type for this stream
	StreamContentField     string // Name of the field containing the binary content for this stream
	// Auto properties
	IsAuto bool // True if this property is automatically set server-side (clients cannot provide/modify it)
	// Computed properties
	IsComputed bool // True if this property is computed server-side and has no database column
	// Annotations
	Annotations *AnnotationCollection // OData vocabulary annotations for this property
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

		// Skip fields explicitly excluded from OData
		if field.Tag.Get("odata") == "-" {
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

	// Detect if this is a media entity (has HasStream() method)
	detectMediaEntity(metadata, entity)

	// Detect stream properties
	detectStreamProperties(metadata)

	// Detect type discriminator property for polymorphic entities
	detectTypeDiscriminator(metadata)

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

		// Skip fields explicitly excluded from OData
		if field.Tag.Get("odata") == "-" {
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

// AnalyzeVirtualEntity extracts metadata from a Go struct for OData virtual entity usage.
// Virtual entities have no database backing store and require overwrite handlers for all operations.
func AnalyzeVirtualEntity(entity interface{}) (*EntityMetadata, error) {
	entityType := reflect.TypeOf(entity)

	// Handle pointer types
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	if entityType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("entity must be a struct, got %s", entityType.Kind())
	}

	metadata := initializeVirtualEntityMetadata(entityType)

	// Analyze struct fields
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip fields explicitly excluded from OData
		if field.Tag.Get("odata") == "-" {
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

	// Detect if this is a media entity (has HasStream() method)
	detectMediaEntity(metadata, entity)

	// Detect stream properties
	detectStreamProperties(metadata)

	// Detect available lifecycle hooks
	detectHooks(metadata)

	return metadata, nil
}

// initializeMetadata creates a new EntityMetadata struct with basic information
func initializeMetadata(entityType reflect.Type) *EntityMetadata {
	entityName := entityType.Name()
	entitySetName := getEntitySetName(entityType)
	tableName := getTableNameFromReflectType(entityType)

	return &EntityMetadata{
		EntityType:    entityType,
		EntityName:    entityName,
		EntitySetName: entitySetName,
		TableName:     tableName,
		Properties:    make([]PropertyMetadata, 0),
		IsSingleton:   false,
	}
}

// initializeSingletonMetadata creates a new EntityMetadata struct for a singleton
func initializeSingletonMetadata(entityType reflect.Type, singletonName string) *EntityMetadata {
	entityName := entityType.Name()
	tableName := getTableNameFromReflectType(entityType)

	return &EntityMetadata{
		EntityType:    entityType,
		EntityName:    entityName,
		EntitySetName: singletonName, // For singletons, we use the singleton name
		SingletonName: singletonName,
		TableName:     tableName,
		Properties:    make([]PropertyMetadata, 0),
		IsSingleton:   true,
	}
}

// initializeVirtualEntityMetadata creates a new EntityMetadata struct for a virtual entity
func initializeVirtualEntityMetadata(entityType reflect.Type) *EntityMetadata {
	entityName := entityType.Name()
	entitySetName := getEntitySetName(entityType)
	tableName := getTableNameFromReflectType(entityType)

	return &EntityMetadata{
		EntityType:    entityType,
		EntityName:    entityName,
		EntitySetName: entitySetName,
		TableName:     tableName,
		Properties:    make([]PropertyMetadata, 0),
		IsVirtual:     true,
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

	// Compute and cache the column name (respects GORM column: tags)
	property.ColumnName = getColumnNameFromProperty(&property)

	// Check for OData tags
	if err := analyzeODataTags(&property, field, metadata); err != nil {
		return PropertyMetadata{}, err
	}

	// Auto-detect nullability based on Go type and GORM tags
	// This runs after OData tags so explicit odata:"nullable" takes precedence
	if err := autoDetectNullability(&property); err != nil {
		return PropertyMetadata{}, err
	}

	if property.KeyGenerator != "" {
		property.DatabaseGenerated = false
	} else if property.IsKey {
		property.DatabaseGenerated = isDatabaseGeneratedKey(property)
	}

	if property.IsKey {
		upsertKeyProperty(metadata, property)
	}

	if property.IsEnum {
		enumMembers, enumType, err := ResolveEnumMembers(field.Type)
		if err != nil {
			return PropertyMetadata{}, fmt.Errorf("error resolving enum members for field %s: %w", field.Name, err)
		}
		property.EnumMembers = append([]EnumMember(nil), enumMembers...)
		property.EnumType = enumType

		if property.EnumTypeName == "" {
			property.EnumTypeName = enumType.Name()
		}

		underlyingType, err := DetermineEnumUnderlyingType(enumType)
		if err != nil {
			return PropertyMetadata{}, fmt.Errorf("unsupported enum type for field %s: %w", field.Name, err)
		}
		property.EnumUnderlyingType = underlyingType
	}

	// Auto-detect annotations from property flags
	autoDetectPropertyAnnotations(&property)

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
			// Use fieldType (already dereferenced) to get the table name for the target entity
			property.NavigationTargetTableName = getTableNameFromReflectType(fieldType)
			property.NavigationIsArray = isSlice

			// Compute and cache the foreign key column name (respects GORM foreignKey: tags)
			property.ForeignKeyColumnName = getForeignKeyColumnName(property)

			// Extract referential constraints from GORM tags (only for foreignKey/references)
			if strings.Contains(gormTag, "foreignKey") || strings.Contains(gormTag, "references") {
				property.ReferentialConstraints = extractReferentialConstraints(gormTag)
			}
		} else if strings.Contains(gormTag, "embedded") {
			// It's a complex type (embedded struct without foreign keys)
			property.IsComplexType = true
			property.EmbeddedPrefix = extractEmbeddedPrefix(gormTag)
			analyzeComplexTypeFields(property, fieldType)
		}
	}
}

// analyzeComplexTypeFields inspects the fields of an embedded complex type and captures their metadata.
func analyzeComplexTypeFields(property *PropertyMetadata, fieldType reflect.Type) {
	if property == nil {
		return
	}

	structType := dereferenceType(fieldType)
	if structType.Kind() != reflect.Struct {
		return
	}

	if property.ComplexTypeFields == nil {
		property.ComplexTypeFields = make(map[string]*PropertyMetadata)
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		if !field.IsExported() {
			continue
		}

		nestedProp := PropertyMetadata{
			Name:      field.Name,
			Type:      field.Type,
			FieldName: field.Name,
			JsonName:  getJsonName(field),
			GormTag:   field.Tag.Get("gorm"),
		}

		analyzeNavigationProperty(&nestedProp, field)

		// Compute and cache the column name for complex type fields
		nestedProp.ColumnName = getColumnNameFromProperty(&nestedProp)

		if nestedProp.IsComplexType {
			nestedProp.EmbeddedPrefix = extractEmbeddedPrefix(nestedProp.GormTag)
			analyzeComplexTypeFields(&nestedProp, field.Type)
		}

		propCopy := nestedProp
		propPtr := &propCopy

		property.ComplexTypeFields[propCopy.Name] = propPtr

		jsonKey := strings.TrimSpace(propCopy.JsonName)
		if jsonKey != "" && jsonKey != "-" {
			property.ComplexTypeFields[jsonKey] = propPtr
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

// extractEmbeddedPrefix extracts the embedded prefix from a GORM tag (embeddedPrefix:<value>).
func extractEmbeddedPrefix(gormTag string) string {
	parts := strings.Split(gormTag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "embeddedPrefix:") {
			return strings.TrimPrefix(part, "embeddedPrefix:")
		}
	}
	return ""
}

// analyzeODataTags processes OData-specific tags on a field
func analyzeODataTags(property *PropertyMetadata, field reflect.StructField, metadata *EntityMetadata) error {
	var sawKey bool

	if odataTag := field.Tag.Get("odata"); odataTag != "" {
		// Check if similarity is defined in the tag to avoid setting default fuzziness
		hasSimilarity := strings.Contains(odataTag, "similarity=")

		// Parse tag as comma-separated key-value pairs
		parts := strings.Split(odataTag, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "key" {
				sawKey = true
			}
			if err := processODataTagPart(property, part, metadata, hasSimilarity); err != nil {
				return err
			}
		}
	}

	// Auto-detect key if no explicit key is set and field name is "ID"
	if len(metadata.KeyProperties) == 0 && field.Name == "ID" {
		property.IsKey = true
		sawKey = true
	}

	if property.KeyGenerator != "" {
		property.KeyGenerator = strings.ToLower(property.KeyGenerator)
		if !KnownKeyGeneratorName(property.KeyGenerator) {
			return fmt.Errorf("unknown key generator '%s' for field %s", property.KeyGenerator, property.Name)
		}
	}

	if property.KeyGenerator != "" && !property.IsKey {
		return fmt.Errorf("key generator configured for non-key field %s", property.Name)
	}

	if sawKey {
		property.IsKey = true
	}

	return nil
}

// processODataTagPart processes a single OData tag part
func processODataTagPart(property *PropertyMetadata, part string, metadata *EntityMetadata, hasSimilarity bool) error {
	switch {
	case part == "key":
		property.IsKey = true
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
	case strings.HasPrefix(part, "generate="):
		property.KeyGenerator = strings.TrimSpace(strings.TrimPrefix(part, "generate="))
	case part == "auto":
		property.IsAuto = true
	case part == "computed":
		property.IsComputed = true
	case strings.HasPrefix(part, "annotation:"):
		// Handle annotation tags: annotation:Core.Computed or annotation:Org.OData.Core.V1.Description=Some description
		annotationValue := strings.TrimPrefix(part, "annotation:")
		annotation, err := ParseAnnotationTag(annotationValue)
		if err != nil {
			return fmt.Errorf("invalid annotation tag %q: %w", annotationValue, err)
		}
		if property.Annotations == nil {
			property.Annotations = NewAnnotationCollection()
		}
		property.Annotations.Add(annotation)
	}

	return nil
}

// autoDetectPropertyAnnotations automatically adds standard OData annotations based on property flags.
// This enables automatic annotation generation for common patterns like computed properties,
// immutable keys, ETag properties, and auto-generated fields.
func autoDetectPropertyAnnotations(property *PropertyMetadata) {
	if property == nil {
		return
	}

	// Database-generated keys are computed (auto-increment)
	if property.IsKey && property.DatabaseGenerated {
		ensureCoreComputedAnnotation(property)
	}

	// Auto properties are computed server-side
	if property.IsAuto {
		ensureCoreComputedAnnotation(property)
	}

	// Computed properties are server-side computed values
	if property.IsComputed {
		ensureCoreComputedAnnotation(property)
	}

	// ETag properties use optimistic concurrency (we add a marker annotation)
	// Note: The actual OptimisticConcurrency annotation is typically on the EntityType
	if property.IsETag {
		ensureCoreComputedAnnotation(property)
	}
}

// ensureCoreComputedAnnotation adds the Core.Computed annotation to a property if it doesn't already exist
func ensureCoreComputedAnnotation(property *PropertyMetadata) {
	if property == nil {
		return
	}
	if property.Annotations == nil {
		property.Annotations = NewAnnotationCollection()
	}
	if !property.Annotations.Has(CoreComputed) {
		property.Annotations.AddTerm(CoreComputed, true)
	}
}

func upsertKeyProperty(metadata *EntityMetadata, property PropertyMetadata) {
	if metadata == nil || !property.IsKey {
		return
	}

	for i := range metadata.KeyProperties {
		if metadata.KeyProperties[i].Name == property.Name {
			metadata.KeyProperties[i] = property
			return
		}
	}

	metadata.KeyProperties = append(metadata.KeyProperties, property)
}

func isDatabaseGeneratedKey(property PropertyMetadata) bool {
	if !property.IsKey {
		return false
	}

	gormTag := strings.ToLower(property.GormTag)
	if strings.Contains(gormTag, "autoincrement:false") {
		return false
	}

	keyType := property.Type
	for keyType.Kind() == reflect.Ptr {
		keyType = keyType.Elem()
	}

	switch keyType.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strings.Contains(gormTag, "autoincrement")
	default:
		return false
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

// getEntitySetName determines the entity set name for an entity type.
// It first checks if the entity implements an EntitySetName() method,
// similar to how GORM's TableName() works. If not, it falls back to
// pluralizing the entity name.
func getEntitySetName(entityType reflect.Type) string {
	// Check for both value and pointer receivers
	valueType := entityType
	ptrType := reflect.PointerTo(entityType)

	// Try value receiver first
	if name := tryGetEntitySetName(valueType, entityType); name != "" {
		return name
	}

	// Try pointer receiver
	if name := tryGetEntitySetName(ptrType, entityType); name != "" {
		return name
	}

	// Fall back to pluralization
	return pluralize(entityType.Name())
}

// tryGetEntitySetName attempts to call the EntitySetName method on the given type.
// Returns empty string if the method doesn't exist or has wrong signature.
func tryGetEntitySetName(checkType reflect.Type, entityType reflect.Type) string {
	method, found := checkType.MethodByName("EntitySetName")
	if !found {
		return ""
	}

	// Verify the method signature: func() string
	methodType := method.Type
	if methodType.NumIn() != 1 || methodType.NumOut() != 1 || methodType.Out(0).Kind() != reflect.String {
		return ""
	}

	// Create appropriate zero value and call the method
	var zeroVal reflect.Value
	if checkType.Kind() == reflect.Ptr {
		zeroVal = reflect.New(entityType)
	} else {
		zeroVal = reflect.New(entityType).Elem()
	}

	result := zeroVal.MethodByName("EntitySetName").Call(nil)
	if len(result) > 0 {
		return result[0].String()
	}

	return ""
}

// detectMediaEntity checks if the entity implements HasStream() method
func detectMediaEntity(metadata *EntityMetadata, entity interface{}) {
	entityType := metadata.EntityType

	// Check for both value and pointer receivers
	valueType := entityType
	ptrType := reflect.PointerTo(entityType)

	// Check if HasStream method exists
	if hasMethod(valueType, "HasStream") || hasMethod(ptrType, "HasStream") {
		// Call the method to get the actual value
		val := reflect.ValueOf(entity)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Try to call HasStream() to get the value
		method := val.MethodByName("HasStream")
		if !method.IsValid() {
			// Try on pointer
			ptrVal := val.Addr()
			method = ptrVal.MethodByName("HasStream")
		}

		if method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() == 1 {
			result := method.Call(nil)
			if len(result) > 0 && result[0].Kind() == reflect.Bool {
				metadata.HasStream = result[0].Bool()
			}
		}
	}
}

// detectStreamProperties finds properties tagged with odata:"stream"
func detectStreamProperties(metadata *EntityMetadata) {
	for i := range metadata.Properties {
		prop := &metadata.Properties[i]

		// Look up the field in the entity type to get its tags
		field, found := metadata.EntityType.FieldByName(prop.FieldName)
		if !found {
			continue
		}

		odataTag := field.Tag.Get("odata")
		if strings.Contains(odataTag, "stream") {
			prop.IsStream = true

			// Look for associated content type and content fields
			// Convention: for a stream property "Photo", look for "PhotoContentType" and "PhotoContent"
			for j := range metadata.Properties {
				otherProp := &metadata.Properties[j]

				// Check for content type field
				if otherProp.FieldName == prop.FieldName+"ContentType" {
					prop.StreamContentTypeField = otherProp.FieldName
				}

				// Check for content field
				if otherProp.FieldName == prop.FieldName+"Content" {
					prop.StreamContentField = otherProp.FieldName
				}
			}

			// Add to stream properties list
			metadata.StreamProperties = append(metadata.StreamProperties, *prop)
		}
	}
}

// detectTypeDiscriminator finds a type discriminator property in the entity metadata.
// The discriminator is used for polymorphic entities to distinguish between base and derived types.
// Common discriminator property names include: <EntityName>Type, ProductType, Type, EntityType
func detectTypeDiscriminator(metadata *EntityMetadata) {
	if metadata == nil {
		return
	}

	// Candidate property names to check for discriminator
	// Priority: EntityName + "Type" (e.g., ProductType for Product entity), then common names
	candidates := []string{
		metadata.EntityName + "Type", // e.g., ProductType for Product entity
		"ProductType",                // Common discriminator name for products
		"Type",
		"EntityType",
		"Discriminator",
	}

	for _, candidate := range candidates {
		for i := range metadata.Properties {
			prop := &metadata.Properties[i]

			// Discriminator must be a string type
			if prop.Type.Kind() != reflect.String {
				continue
			}

			// Check if the property name matches any candidate (case-insensitive)
			if strings.EqualFold(prop.FieldName, candidate) ||
				strings.EqualFold(prop.Name, candidate) ||
				strings.EqualFold(prop.JsonName, candidate) {
				metadata.TypeDiscriminator = &TypeDiscriminatorInfo{
					ColumnName: prop.ColumnName,
					JsonName:   prop.JsonName,
					FieldName:  prop.FieldName,
				}
				return
			}
		}
	}
}

// detectHooks checks if the entity type has any lifecycle hook methods
func detectHooks(metadata *EntityMetadata) {
	entityType := metadata.EntityType

	// Check for both value and pointer receivers
	valueType := entityType
	ptrType := reflect.PointerTo(entityType)

	// Check ODataBeforeCreate
	if hasMethod(valueType, "ODataBeforeCreate") || hasMethod(ptrType, "ODataBeforeCreate") {
		metadata.Hooks.HasODataBeforeCreate = true
	}

	// Check ODataAfterCreate
	if hasMethod(valueType, "ODataAfterCreate") || hasMethod(ptrType, "ODataAfterCreate") {
		metadata.Hooks.HasODataAfterCreate = true
	}

	// Check ODataBeforeUpdate
	if hasMethod(valueType, "ODataBeforeUpdate") || hasMethod(ptrType, "ODataBeforeUpdate") {
		metadata.Hooks.HasODataBeforeUpdate = true
	}

	// Check ODataAfterUpdate
	if hasMethod(valueType, "ODataAfterUpdate") || hasMethod(ptrType, "ODataAfterUpdate") {
		metadata.Hooks.HasODataAfterUpdate = true
	}

	// Check ODataBeforeDelete
	if hasMethod(valueType, "ODataBeforeDelete") || hasMethod(ptrType, "ODataBeforeDelete") {
		metadata.Hooks.HasODataBeforeDelete = true
	}

	// Check ODataAfterDelete
	if hasMethod(valueType, "ODataAfterDelete") || hasMethod(ptrType, "ODataAfterDelete") {
		metadata.Hooks.HasODataAfterDelete = true
	}

	// Check ODataBeforeReadCollection
	if hasMethod(valueType, "ODataBeforeReadCollection") || hasMethod(ptrType, "ODataBeforeReadCollection") {
		metadata.Hooks.HasODataBeforeReadCollection = true
	}

	// Check ODataAfterReadCollection
	if hasMethod(valueType, "ODataAfterReadCollection") || hasMethod(ptrType, "ODataAfterReadCollection") {
		metadata.Hooks.HasODataAfterReadCollection = true
	}

	// Check ODataBeforeReadEntity
	if hasMethod(valueType, "ODataBeforeReadEntity") || hasMethod(ptrType, "ODataBeforeReadEntity") {
		metadata.Hooks.HasODataBeforeReadEntity = true
	}

	// Check ODataAfterReadEntity
	if hasMethod(valueType, "ODataAfterReadEntity") || hasMethod(ptrType, "ODataAfterReadEntity") {
		metadata.Hooks.HasODataAfterReadEntity = true
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

// ResolvePropertyPath resolves a property path (e.g., "ShippingAddress/City") to the corresponding metadata and embedded prefix.
// It returns the metadata for the final segment and the concatenated embedded prefix used for database columns.
func (metadata *EntityMetadata) ResolvePropertyPath(path string) (*PropertyMetadata, string, error) {
	if metadata == nil {
		return nil, "", fmt.Errorf("entity metadata is nil")
	}

	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return nil, "", fmt.Errorf("property path cannot be empty")
	}

	segments := strings.Split(trimmedPath, "/")
	var currentProp *PropertyMetadata
	var prefixBuilder strings.Builder

	for i, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return nil, "", fmt.Errorf("property path '%s' contains an empty segment", trimmedPath)
		}

		if i == 0 {
			currentProp = metadata.FindProperty(segment)
		} else {
			if currentProp == nil || !currentProp.IsComplexType {
				return nil, "", fmt.Errorf("property '%s' is not a complex type in path '%s'", segments[i-1], trimmedPath)
			}
			prefixBuilder.WriteString(currentProp.EmbeddedPrefix)
			currentProp = currentProp.FindComplexField(segment)
		}

		if currentProp == nil {
			return nil, "", fmt.Errorf("property '%s' not found in path '%s'", segment, trimmedPath)
		}
	}

	return currentProp, prefixBuilder.String(), nil
}

// FindComplexField returns a nested property within a complex type by either struct field name or JSON name.
func (property *PropertyMetadata) FindComplexField(name string) *PropertyMetadata {
	if property == nil || !property.IsComplexType || property.ComplexTypeFields == nil {
		return nil
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil
	}

	if nested, ok := property.ComplexTypeFields[trimmedName]; ok {
		return nested
	}

	return nil
}

// ResolveSingleEntityNavigationPath resolves a navigation path that traverses only single-entity navigation properties.
// It returns the target entity metadata, the navigation segments traversed, and the remaining property path.
// For example, "Team/Club/Name" returns the Club entity metadata, ["Team", "Club"], and "Name".
func (metadata *EntityMetadata) ResolveSingleEntityNavigationPath(path string) (*EntityMetadata, []string, string, error) {
	if metadata == nil {
		return nil, nil, "", fmt.Errorf("entity metadata is nil")
	}

	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" || !strings.Contains(trimmedPath, "/") {
		return nil, nil, "", fmt.Errorf("navigation path '%s' is not a multi-segment path", path)
	}

	segments := strings.Split(trimmedPath, "/")
	currentMetadata := metadata
	rootRegistry := metadata.entitiesRegistry
	navigationSegments := make([]string, 0, len(segments)-1)

	for i, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return nil, nil, "", fmt.Errorf("navigation path '%s' contains an empty segment", trimmedPath)
		}

		navProp := currentMetadata.FindNavigationProperty(segment)
		if navProp == nil {
			if len(navigationSegments) == 0 {
				return nil, nil, "", fmt.Errorf("navigation path '%s' does not start with a navigation property", trimmedPath)
			}
			remaining := strings.Join(segments[i:], "/")
			return currentMetadata, navigationSegments, remaining, nil
		}

		if navProp.NavigationIsArray {
			return nil, nil, "", fmt.Errorf("navigation property '%s' is a collection", segment)
		}

		navigationSegments = append(navigationSegments, segment)
		targetMetadata, err := currentMetadata.ResolveNavigationTarget(segment)
		if err != nil {
			return nil, nil, "", err
		}
		if targetMetadata != nil && targetMetadata.navigationTargetIndex == nil && rootRegistry != nil {
			targetMetadata.SetEntitiesRegistry(rootRegistry)
		}
		currentMetadata = targetMetadata
	}

	return nil, nil, "", fmt.Errorf("navigation path '%s' cannot end with a navigation property", trimmedPath)
}

// IsSingleEntityNavigationPath checks if a property path represents a single-entity navigation property path.
// For example, "Team/ClubID" where Team is a single-entity (not collection) navigation property.
// Returns true if the path traverses only single-entity navigation properties and ends on a property.
// Per OData v4 spec 5.1.1.15, properties of entities related with cardinality 0..1 or 1 can be accessed directly.
func (metadata *EntityMetadata) IsSingleEntityNavigationPath(path string) bool {
	if metadata == nil {
		return false
	}

	targetMetadata, _, remainingPath, err := metadata.ResolveSingleEntityNavigationPath(path)
	if err != nil || targetMetadata == nil || remainingPath == "" {
		return false
	}

	prop, _, err := targetMetadata.ResolvePropertyPath(remainingPath)
	if err != nil || prop == nil {
		return false
	}

	return !prop.IsNavigationProp
}

// SetEntitiesRegistry provides access to the registered entity metadata for navigation resolution.
func (metadata *EntityMetadata) SetEntitiesRegistry(entities map[string]*EntityMetadata) {
	if metadata == nil {
		return
	}
	metadata.entitiesRegistry = entities

	// Build navigation target index for O(1) lookup
	metadata.navigationTargetIndex = make(map[string]*EntityMetadata)
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		// Index by both EntityName and EntitySetName for flexible lookups
		if entity.EntityName != "" {
			metadata.navigationTargetIndex[entity.EntityName] = entity
		}
		if entity.EntitySetName != "" {
			metadata.navigationTargetIndex[entity.EntitySetName] = entity
		}
	}
}

// AddEntityToRegistry adds a single entity to the navigation target index.
// This is more efficient than rebuilding the entire index when adding one entity at a time.
func (metadata *EntityMetadata) AddEntityToRegistry(entity *EntityMetadata) {
	if metadata == nil || entity == nil {
		return
	}

	// Initialize the index if it doesn't exist
	if metadata.navigationTargetIndex == nil {
		metadata.navigationTargetIndex = make(map[string]*EntityMetadata)
	}

	// Add the entity to the index
	if entity.EntityName != "" {
		metadata.navigationTargetIndex[entity.EntityName] = entity
	}
	if entity.EntitySetName != "" {
		metadata.navigationTargetIndex[entity.EntitySetName] = entity
	}
}

// ResolveNavigationTarget returns the target entity metadata for a navigation property.
func (metadata *EntityMetadata) ResolveNavigationTarget(name string) (*EntityMetadata, error) {
	if metadata == nil {
		return nil, fmt.Errorf("entity metadata is nil")
	}

	navProp := metadata.FindNavigationProperty(name)
	if navProp == nil {
		return nil, fmt.Errorf("navigation property '%s' not found", name)
	}

	if metadata.navigationTargetIndex == nil {
		return nil, fmt.Errorf("entity metadata registry is not configured")
	}

	// Use O(1) map lookup instead of O(n) iteration
	entity, ok := metadata.navigationTargetIndex[navProp.NavigationTarget]
	if !ok {
		return nil, fmt.Errorf("navigation target '%s' not registered", navProp.NavigationTarget)
	}

	return entity, nil
}

// dereferenceType unwraps pointer types to obtain the underlying type.
func dereferenceType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// getTableNameFromReflectType returns the table name for a given entity type
// This respects custom TableName() methods on the entity by using reflection
// to create a zero-value instance and checking if it implements the TableName() interface
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

// getColumnNameFromProperty computes the database column name for a property
// This respects GORM column: tags and falls back to snake_case conversion
func getColumnNameFromProperty(prop *PropertyMetadata) string {
	if prop == nil {
		return ""
	}

	// Check GORM tag for explicit column name
	if prop.GormTag != "" {
		parts := strings.Split(prop.GormTag, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "column:") {
				return strings.TrimPrefix(part, "column:")
			}
		}
	}

	// Fall back to snake_case conversion
	return toSnakeCase(prop.Name)
}

// getForeignKeyColumnName computes the foreign key column name for a navigation property
// This respects GORM foreignKey: tags and falls back to <navigation_property_name>_id convention
func getForeignKeyColumnName(prop *PropertyMetadata) string {
	if prop == nil || !prop.IsNavigationProp {
		return ""
	}

	// Check GORM tag for explicit foreignKey
	if prop.GormTag != "" {
		parts := strings.Split(prop.GormTag, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "foreignKey:") {
				fkField := strings.TrimPrefix(part, "foreignKey:")
				// For composite keys, handle comma-separated field names
				fkFields := strings.Split(fkField, ",")
				snakeCaseFields := make([]string, len(fkFields))
				for i, field := range fkFields {
					snakeCaseFields[i] = toSnakeCase(strings.TrimSpace(field))
				}
				return strings.Join(snakeCaseFields, ",")
			}
		}
	}

	// Fall back to convention: <navigation_property_name>_id
	return toSnakeCase(prop.Name) + "_id"
}

// toSnakeCase converts a camelCase or PascalCase string to snake_case
// NOTE: This is duplicated in internal/query/helpers.go and internal/handlers/helpers.go
// A future refactor could extract this to a shared utility package
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
