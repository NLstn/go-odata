package response

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/etag"
	internalMetadata "github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
)

func addNavigationLinks(data interface{}, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, r *http.Request, entitySetName string, metadataLevel string, fullMetadata *internalMetadata.EntityMetadata) []interface{} {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		return []interface{}{}
	}

	result := make([]interface{}, dataValue.Len())

	// Fast path for "none" metadata level - no processing needed
	if metadataLevel == "none" {
		for i := 0; i < dataValue.Len(); i++ {
			result[i] = dataValue.Index(i).Interface()
		}
		return result
	}

	baseURL := buildBaseURL(r)

	for i := 0; i < dataValue.Len(); i++ {
		entity := dataValue.Index(i)

		if entity.Kind() == reflect.Map {
			entityMap := processMapEntity(entity, metadata, expandOptions, selectedNavProps, baseURL, entitySetName, metadataLevel, fullMetadata)
			if entityMap != nil {
				result[i] = entityMap
			}
		} else {
			orderedMap := processStructEntityOrdered(entity, metadata, expandOptions, selectedNavProps, baseURL, entitySetName, metadataLevel, fullMetadata)
			result[i] = orderedMap
		}
	}

	return result
}

func processMapEntity(entity reflect.Value, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, baseURL, entitySetName string, metadataLevel string, fullMetadata *internalMetadata.EntityMetadata) map[string]interface{} {
	entityMap, ok := entity.Interface().(map[string]interface{})
	if !ok {
		return nil
	}

	// Add ETag if present and metadata level is not "none"
	if fullMetadata != nil && fullMetadata.ETagProperty != nil && metadataLevel != "none" {
		etagValue := etag.Generate(entityMap, fullMetadata)
		if etagValue != "" {
			entityMap["@odata.etag"] = etagValue
		}
	}

	// Add @odata.id for "full" and "minimal" metadata levels
	if metadataLevel == "full" || metadataLevel == "minimal" {
		keySegment := buildKeySegmentFromMap(entityMap, metadata)
		if keySegment != "" {
			entityID := fmt.Sprintf("%s/%s(%s)", baseURL, entitySetName, keySegment)
			entityMap["@odata.id"] = entityID
		}
	}

	// Add @odata.type for "full" metadata level
	if metadataLevel == "full" {
		entityTypeName := getEntityTypeFromSetName(entitySetName)
		entityMap["@odata.type"] = "#" + metadata.GetNamespace() + "." + entityTypeName

		// Add entity-level vocabulary annotations for full metadata
		if fullMetadata != nil && fullMetadata.Annotations != nil && fullMetadata.Annotations.Len() > 0 {
			for _, annotation := range fullMetadata.Annotations.Get() {
				annotationKey := "@" + annotation.QualifiedTerm()
				entityMap[annotationKey] = annotation.Value
			}
		}

		// Add property-level vocabulary annotations for full metadata
		// Only add annotations for properties that are present in the response
		if fullMetadata != nil {
			for _, prop := range fullMetadata.Properties {
				// Check if property exists in the entity map (respects $select filtering)
				if _, exists := entityMap[prop.JsonName]; !exists {
					continue
				}
				if prop.Annotations != nil && prop.Annotations.Len() > 0 {
					for _, annotation := range prop.Annotations.Get() {
						annotationKey := prop.JsonName + "@" + annotation.QualifiedTerm()
						entityMap[annotationKey] = annotation.Value
					}
				}
			}
		}

		// Add navigation links for unexpanded navigation properties in "full" mode
		for _, prop := range metadata.GetProperties() {
			if !prop.IsNavigationProp {
				continue
			}

			if isPropertyExpanded(prop, expandOptions) {
				continue
			}

			keySegment := buildKeySegmentFromMap(entityMap, metadata)
			if keySegment != "" {
				navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, prop.JsonName)
				entityMap[prop.JsonName+"@odata.navigationLink"] = navLink
			}
		}
		return entityMap
	}

	if metadataLevel == "minimal" {
		for _, prop := range metadata.GetProperties() {
			if !prop.IsNavigationProp {
				continue
			}

			if isPropertyExpanded(prop, expandOptions) || !isPropertySelectedForLinks(prop, selectedNavProps) {
				continue
			}

			keySegment := buildKeySegmentFromMap(entityMap, metadata)
			if keySegment != "" {
				navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, prop.JsonName)
				entityMap[prop.JsonName+"@odata.navigationLink"] = navLink
			}
		}
	}

	if fullMetadata != nil {
		applyNestedExpandAnnotationsToMap(entityMap, expandOptions, fullMetadata)
	}

	return entityMap
}

func processStructEntityOrdered(entity reflect.Value, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, baseURL, entitySetName string, metadataLevel string, fullMetadata *internalMetadata.EntityMetadata) *OrderedMap {
	entityType := entity.Type()
	fieldInfos := getFieldInfos(entityType)

	// Pre-calculate capacity: fields + potential metadata annotations (etag, id, type)
	capacity := entity.NumField() + 3
	// Use pooled OrderedMap for better performance
	entityMap := AcquireOrderedMapWithCapacity(capacity)

	// Pre-compute key segment and entity ID if needed (reuse across annotations)
	var keySegment string
	needsKeySegment := (metadataLevel == "full" || metadataLevel == "minimal")

	if needsKeySegment {
		keySegment = buildKeySegmentFromEntityCached(entity, metadata)
	}

	// Add ETag if present and metadata level is not "none"
	if fullMetadata != nil && fullMetadata.ETagProperty != nil && metadataLevel != "none" {
		var entityInterface interface{}
		if entity.Kind() == reflect.Ptr {
			entityInterface = entity.Interface()
		} else {
			if entity.CanAddr() {
				entityInterface = entity.Addr().Interface()
			} else {
				entityInterface = entity.Interface()
			}
		}
		etagValue := etag.Generate(entityInterface, fullMetadata)
		if etagValue != "" {
			entityMap.Set("@odata.etag", etagValue)
		}
	}

	// Add @odata.id for "full" and "minimal" metadata levels
	if needsKeySegment && keySegment != "" {
		var entityID strings.Builder
		entityID.Grow(len(baseURL) + len(entitySetName) + len(keySegment) + 3)
		entityID.WriteString(baseURL)
		entityID.WriteByte('/')
		entityID.WriteString(entitySetName)
		entityID.WriteByte('(')
		entityID.WriteString(keySegment)
		entityID.WriteByte(')')
		entityMap.Set("@odata.id", entityID.String())
	}

	// Add @odata.type for "full" metadata level
	if metadataLevel == "full" {
		entityTypeName := getEntityTypeFromSetName(entitySetName)
		namespace := metadata.GetNamespace()
		var typeStr strings.Builder
		typeStr.Grow(1 + len(namespace) + 1 + len(entityTypeName))
		typeStr.WriteByte('#')
		typeStr.WriteString(namespace)
		typeStr.WriteByte('.')
		typeStr.WriteString(entityTypeName)
		entityMap.Set("@odata.type", typeStr.String())

		// Add entity-level vocabulary annotations for full metadata
		if fullMetadata != nil && fullMetadata.Annotations != nil && fullMetadata.Annotations.Len() > 0 {
			for _, annotation := range fullMetadata.Annotations.Get() {
				annotationKey := "@" + annotation.QualifiedTerm()
				entityMap.Set(annotationKey, annotation.Value)
			}
		}
	}

	// Process entity fields - optimized to reduce reflection calls
	// Cache property metadata lookups per entity type
	propMetaMap := getCachedPropertyMetadataMap(metadata)

	// Pre-build a map for full property metadata by field name (for annotation lookup)
	// Only build this map if we need it (full metadata level)
	var fullPropMetaByName map[string]*internalMetadata.PropertyMetadata
	if metadataLevel == "full" && fullMetadata != nil {
		// Allocate capacity for Name and JsonName entries to avoid map reallocations
		fullPropMetaByName = make(map[string]*internalMetadata.PropertyMetadata, len(fullMetadata.Properties)*2)
		for i := range fullMetadata.Properties {
			prop := &fullMetadata.Properties[i]
			fullPropMetaByName[prop.Name] = prop
			if prop.JsonName != "" && prop.JsonName != prop.Name {
				fullPropMetaByName[prop.JsonName] = prop
			}
		}
	}

	for j := 0; j < entity.NumField(); j++ {
		info := fieldInfos[j]
		if !info.IsExported {
			continue
		}

		field := entityType.Field(j)
		propMeta := propMetaMap[field.Name]

		if propMeta != nil && propMeta.IsNavigationProp {
			// For minimal metadata, skip navigation properties unless they're expanded
			if metadataLevel == "minimal" && !isPropertyExpanded(*propMeta, expandOptions) && !isPropertySelectedForLinks(*propMeta, selectedNavProps) {
				// Skip unexpanded navigation properties for minimal metadata
				continue
			}
			// Only get fieldValue when we actually need it
			fieldValue := entity.Field(j)
			expandOpt := query.FindExpandOption(expandOptions, propMeta.Name, propMeta.JsonName)
			processNavigationPropertyOrderedWithMetadata(entityMap, entity, propMeta, fieldValue, info.JsonName, expandOpt, selectedNavProps, baseURL, entitySetName, metadata, metadataLevel, keySegment, fullMetadata)
		} else {
			// Add property-level annotations first (for full metadata)
			if metadataLevel == "full" && fullPropMetaByName != nil {
				// O(1) lookup using pre-built map
				if fullProp := fullPropMetaByName[field.Name]; fullProp != nil {
					if fullProp.Annotations != nil && fullProp.Annotations.Len() > 0 {
						for _, annotation := range fullProp.Annotations.Get() {
							annotationKey := info.JsonName + "@" + annotation.QualifiedTerm()
							entityMap.Set(annotationKey, annotation.Value)
						}
					}
				}
			}
			// Then add the property value
			fieldValue := entity.Field(j)
			entityMap.Set(info.JsonName, fieldValue.Interface())
		}
	}

	return entityMap
}

func isPropertyExpanded(prop PropertyMetadata, expandOptions []query.ExpandOption) bool {
	return query.FindExpandOption(expandOptions, prop.Name, prop.JsonName) != nil
}

func processNavigationPropertyOrderedWithMetadata(entityMap *OrderedMap, entity reflect.Value, propMeta *PropertyMetadata, fieldValue reflect.Value, jsonName string, expandOpt *query.ExpandOption, selectedNavProps []string, baseURL, entitySetName string, metadata EntityMetadataProvider, metadataLevel string, keySegment string, fullMetadata *internalMetadata.EntityMetadata) {
	if expandOpt != nil {
		updatedValue := fieldValue.Interface()
		if fullMetadata != nil {
			if targetMetadata, err := fullMetadata.ResolveNavigationTarget(propMeta.Name); err == nil {
				var count *int
				updatedValue, count = ApplyExpandOptionToValue(updatedValue, expandOpt, targetMetadata)
				if count != nil {
					entityMap.Set(jsonName+"@odata.count", *count)
				}
			}
		}
		entityMap.Set(jsonName, updatedValue)
		return
	}

	if metadataLevel == "full" || (metadataLevel == "minimal" && isPropertySelectedForLinks(*propMeta, selectedNavProps)) {
		if keySegment == "" {
			keySegment = BuildKeySegmentFromEntity(entity, metadata)
		}
		if keySegment != "" {
			navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, propMeta.JsonName)
			entityMap.Set(jsonName+"@odata.navigationLink", navLink)
		}
	}
}

func isPropertySelectedForLinks(prop PropertyMetadata, selectedNavProps []string) bool {
	for _, selected := range selectedNavProps {
		if selected == prop.Name || selected == prop.JsonName {
			return true
		}
	}
	return false
}

// BuildKeySegmentFromEntity builds the key segment for URLs from an entity and metadata.
func BuildKeySegmentFromEntity(entity reflect.Value, metadata EntityMetadataProvider) string {
	return buildKeySegmentFromEntityCached(entity, metadata)
}

// formatKeyValue converts a reflect.Value to its string representation without using fmt.Sprintf
// This is significantly faster than fmt.Sprintf("%v", ...) for common types
func formatKeyValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	default:
		// Fallback for complex types (UUIDs, etc.)
		return fmt.Sprintf("%v", v.Interface())
	}
}

// formatInterfaceValue converts an interface{} to its string representation without using fmt.Sprintf
// This is significantly faster than fmt.Sprintf("%v", ...) for common types
func formatInterfaceValue(v interface{}) string {
	switch val := v.(type) {
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	default:
		// Fallback for complex types (UUIDs, etc.)
		return fmt.Sprintf("%v", v)
	}
}

// buildKeySegmentFromEntityCached is an internal helper for building key segments
func buildKeySegmentFromEntityCached(entity reflect.Value, metadata EntityMetadataProvider) string {
	keyProps := metadata.GetKeyProperties()
	if len(keyProps) == 0 {
		return ""
	}

	if len(keyProps) == 1 {
		keyFieldValue := entity.FieldByName(keyProps[0].Name)
		if keyFieldValue.IsValid() {
			return formatKeyValue(keyFieldValue)
		}
		return ""
	}

	// For composite keys, build the key segment
	var builder strings.Builder
	// Estimate capacity: name=value, separated by commas
	builder.Grow(len(keyProps) * 20)

	for i, keyProp := range keyProps {
		keyFieldValue := entity.FieldByName(keyProp.Name)
		if keyFieldValue.IsValid() {
			if i > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(keyProp.JsonName)
			if keyFieldValue.Kind() == reflect.String {
				builder.WriteString("='")
				builder.WriteString(keyFieldValue.String())
				builder.WriteByte('\'')
			} else {
				builder.WriteByte('=')
				builder.WriteString(formatKeyValue(keyFieldValue))
			}
		}
	}

	return builder.String()
}

func buildKeySegmentFromMap(entityMap map[string]interface{}, metadata EntityMetadataProvider) string {
	keyProps := metadata.GetKeyProperties()
	if len(keyProps) == 0 {
		return ""
	}

	if len(keyProps) == 1 {
		if keyValue := entityMap[keyProps[0].JsonName]; keyValue != nil {
			return formatInterfaceValue(keyValue)
		}
		return ""
	}

	var builder strings.Builder
	// Estimate capacity: name=value, separated by commas
	builder.Grow(len(keyProps) * 20)

	first := true
	for _, keyProp := range keyProps {
		if keyValue := entityMap[keyProp.JsonName]; keyValue != nil {
			if !first {
				builder.WriteByte(',')
			}
			first = false
			builder.WriteString(keyProp.JsonName)
			if strVal, ok := keyValue.(string); ok {
				builder.WriteString("='")
				builder.WriteString(strVal)
				builder.WriteByte('\'')
			} else {
				builder.WriteByte('=')
				builder.WriteString(formatInterfaceValue(keyValue))
			}
		}
	}

	return builder.String()
}
