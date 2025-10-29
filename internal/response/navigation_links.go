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

	if fullMetadata != nil && fullMetadata.ETagProperty != nil && metadataLevel != "none" {
		etagValue := etag.Generate(entityMap, fullMetadata)
		if etagValue != "" {
			entityMap["@odata.etag"] = etagValue
		}
	}

	keySegment := buildKeySegmentFromMap(entityMap, metadata)
	if keySegment != "" {
		entityID := fmt.Sprintf("%s/%s(%s)", baseURL, entitySetName, keySegment)

		switch metadataLevel {
		case "full", "minimal":
			entityMap["@odata.id"] = entityID
		}
	}

	if metadataLevel == "full" {
		entityTypeName := getEntityTypeFromSetName(entitySetName)
		entityMap["@odata.type"] = "#" + metadata.GetNamespace() + "." + entityTypeName
	}

	if metadataLevel == "full" {
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
	entityMap := NewOrderedMap()
	entityType := entity.Type()

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

	switch metadataLevel {
	case "full", "minimal":
		keySegment := BuildKeySegmentFromEntity(entity, metadata)
		if keySegment != "" {
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
	}

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

	fieldInfos := getFieldInfos(entityType)
	for j := 0; j < entity.NumField(); j++ {
		info := fieldInfos[j]
		if !info.IsExported {
			continue
		}

		field := entityType.Field(j)
		fieldValue := entity.Field(j)
		propMeta := getCachedPropertyMetadata(field.Name, metadata)

		if propMeta != nil && propMeta.IsNavigationProp {
			processNavigationPropertyOrderedWithMetadata(entityMap, entity, propMeta, fieldValue, info.JsonName, expandedProps, baseURL, entitySetName, metadata, metadataLevel)
		} else {
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

func processNavigationPropertyOrderedWithMetadata(entityMap *OrderedMap, entity reflect.Value, propMeta *PropertyMetadata, fieldValue reflect.Value, jsonName string, expandedProps []string, baseURL, entitySetName string, metadata EntityMetadataProvider, metadataLevel string) {
	if isPropertyExpanded(*propMeta, expandedProps) {
		entityMap.Set(jsonName, fieldValue.Interface())
	} else if metadataLevel == "full" {
		keySegment := BuildKeySegmentFromEntity(entity, metadata)
		if keySegment != "" {
			navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, propMeta.JsonName)
			entityMap.Set(jsonName+"@odata.navigationLink", navLink)
		}
	}
}

// BuildKeySegmentFromEntity builds the key segment for URLs from an entity and metadata.
func BuildKeySegmentFromEntity(entity reflect.Value, metadata EntityMetadataProvider) string {
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
