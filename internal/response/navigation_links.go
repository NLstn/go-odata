package response

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/metadata"
)

func addNavigationLinks(data interface{}, metadata EntityMetadataProvider, expandedProps []string, r *http.Request, entitySetName string, metadataLevel string, fullMetadata *metadata.EntityMetadata) []interface{} {
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
		var entityMap interface{}

		if entity.Kind() == reflect.Map {
			entityMap = processMapEntity(entity, metadata, expandedProps, baseURL, entitySetName, metadataLevel, fullMetadata)
		} else {
			entityMap = processStructEntityOrdered(entity, metadata, expandedProps, baseURL, entitySetName, metadataLevel, fullMetadata)
		}

		if entityMap != nil {
			result[i] = entityMap
		}
	}

	return result
}

func processMapEntity(entity reflect.Value, metadata EntityMetadataProvider, expandedProps []string, baseURL, entitySetName string, metadataLevel string, fullMetadata *metadata.EntityMetadata) map[string]interface{} {
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

		// Add navigation links for unexpanded navigation properties in "full" mode
		for _, prop := range metadata.GetProperties() {
			if !prop.IsNavigationProp {
				continue
			}

			if isPropertyExpanded(prop, expandedProps) {
				continue
			}

			if _, exists := entityMap[prop.JsonName]; !exists {
				keySegment := buildKeySegmentFromMap(entityMap, metadata)
				if keySegment != "" {
					navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, prop.JsonName)
					entityMap[prop.JsonName+"@odata.navigationLink"] = navLink
				}
			}
		}
	}

	return entityMap
}

func processStructEntityOrdered(entity reflect.Value, metadata EntityMetadataProvider, expandedProps []string, baseURL, entitySetName string, metadataLevel string, fullMetadata *metadata.EntityMetadata) *OrderedMap {
	entityType := entity.Type()
	fieldInfos := getFieldInfos(entityType)

	// Pre-calculate capacity: fields + potential metadata annotations (etag, id, type)
	capacity := entity.NumField() + 3
	entityMap := NewOrderedMapWithCapacity(capacity)

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
	}

	// Process entity fields - optimized to reduce reflection calls
	// Cache property metadata lookups per entity type
	propMetaMap := getCachedPropertyMetadataMap(metadata)

	for j := 0; j < entity.NumField(); j++ {
		info := fieldInfos[j]
		if !info.IsExported {
			continue
		}

		field := entityType.Field(j)
		propMeta := propMetaMap[field.Name]

		if propMeta != nil && propMeta.IsNavigationProp {
			// For minimal metadata, skip navigation properties unless they're expanded
			if metadataLevel == "minimal" && !isPropertyExpanded(*propMeta, expandedProps) {
				// Skip unexpanded navigation properties for minimal metadata
				continue
			}
			// Only get fieldValue when we actually need it
			fieldValue := entity.Field(j)
			processNavigationPropertyOrderedWithMetadata(entityMap, entity, propMeta, fieldValue, info.JsonName, expandedProps, baseURL, entitySetName, metadata, metadataLevel, keySegment)
		} else {
			fieldValue := entity.Field(j)
			entityMap.Set(info.JsonName, fieldValue.Interface())
		}
	}

	return entityMap
}

func isPropertyExpanded(prop PropertyMetadata, expandedProps []string) bool {
	for _, expanded := range expandedProps {
		if expanded == prop.Name || expanded == prop.JsonName {
			return true
		}
	}
	return false
}

func processNavigationPropertyOrderedWithMetadata(entityMap *OrderedMap, entity reflect.Value, propMeta *PropertyMetadata, fieldValue reflect.Value, jsonName string, expandedProps []string, baseURL, entitySetName string, metadata EntityMetadataProvider, metadataLevel string, keySegment string) {
	if isPropertyExpanded(*propMeta, expandedProps) {
		entityMap.Set(jsonName, fieldValue.Interface())
	} else if metadataLevel == "full" {
		if keySegment == "" {
			keySegment = BuildKeySegmentFromEntity(entity, metadata)
		}
		if keySegment != "" {
			navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, propMeta.JsonName)
			entityMap.Set(jsonName+"@odata.navigationLink", navLink)
		}
	}
}

// BuildKeySegmentFromEntity builds the key segment for URLs from an entity and metadata.
func BuildKeySegmentFromEntity(entity reflect.Value, metadata EntityMetadataProvider) string {
	return buildKeySegmentFromEntityCached(entity, metadata)
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
			return fmt.Sprintf("%v", keyFieldValue.Interface())
		}
		return ""
	}

	// For composite keys, build the key segment
	var parts []string
	for _, keyProp := range keyProps {
		keyFieldValue := entity.FieldByName(keyProp.Name)
		if keyFieldValue.IsValid() {
			keyValue := keyFieldValue.Interface()
			if keyFieldValue.Kind() == reflect.String {
				parts = append(parts, fmt.Sprintf("%s='%v'", keyProp.JsonName, keyValue))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%v", keyProp.JsonName, keyValue))
			}
		}
	}

	return strings.Join(parts, ",")
}

func buildKeySegmentFromMap(entityMap map[string]interface{}, metadata EntityMetadataProvider) string {
	keyProps := metadata.GetKeyProperties()
	if len(keyProps) == 0 {
		return ""
	}

	if len(keyProps) == 1 {
		if keyValue := entityMap[keyProps[0].JsonName]; keyValue != nil {
			return fmt.Sprintf("%v", keyValue)
		}
		return ""
	}

	var parts []string
	for _, keyProp := range keyProps {
		if keyValue := entityMap[keyProp.JsonName]; keyValue != nil {
			if strVal, ok := keyValue.(string); ok {
				parts = append(parts, fmt.Sprintf("%s='%s'", keyProp.JsonName, strVal))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%v", keyProp.JsonName, keyValue))
			}
		}
	}

	return strings.Join(parts, ",")
}
