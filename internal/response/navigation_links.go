package response

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/nlstn/go-odata/internal/etag"
	internalMetadata "github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/query"
)

func addNavigationLinks(data interface{}, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, r *http.Request, entitySetName string, metadataLevel string, fullMetadata *internalMetadata.EntityMetadata) []interface{} {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		return []interface{}{}
	}

	result := make([]interface{}, dataValue.Len())

	baseURL := buildBaseURL(r)

	// Parse annotation filter from odata.include-annotations preference
	var annotationFilter *string
	if pref := preference.ParsePrefer(r); pref.IncludeAnnotations != nil {
		annotationFilter = pref.IncludeAnnotations
	}

	for i := 0; i < dataValue.Len(); i++ {
		entity := dataValue.Index(i)

		if entity.Kind() == reflect.Map {
			entityMap := processMapEntity(entity, metadata, expandOptions, selectedNavProps, baseURL, entitySetName, metadataLevel, fullMetadata, annotationFilter)
			if entityMap != nil {
				result[i] = entityMap
			}
		} else {
			orderedMap := processStructEntityOrdered(entity, metadata, expandOptions, selectedNavProps, baseURL, entitySetName, metadataLevel, fullMetadata, annotationFilter)
			result[i] = orderedMap
		}
	}

	return result
}

func processMapEntity(entity reflect.Value, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, baseURL, entitySetName string, metadataLevel string, fullMetadata *internalMetadata.EntityMetadata, annotationFilter *string) map[string]interface{} {
	entityMap, ok := entity.Interface().(map[string]interface{})
	if !ok {
		return nil
	}

	// Encode Edm.Binary ([]byte) property values as base64url strings per the
	// OData JSON Format spec before any further processing serializes them.
	for k, v := range entityMap {
		entityMap[k] = EncodeEdmBinary(v)
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
				if annotationFilter != nil && !preference.MatchesAnnotationFilter(annotation.QualifiedTerm(), *annotationFilter) {
					continue
				}
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
						if annotationFilter != nil && !preference.MatchesAnnotationFilter(annotation.QualifiedTerm(), *annotationFilter) {
							continue
						}
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

func processStructEntityOrdered(entity reflect.Value, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, baseURL, entitySetName string, metadataLevel string, fullMetadata *internalMetadata.EntityMetadata, annotationFilter *string) *OrderedMap {
	// Pre-calculate capacity: fields + potential metadata annotations (etag, id, type)
	capacity := entity.NumField() + 3
	// Use pooled OrderedMap for better performance
	entityMap := AcquireOrderedMapWithCapacity(capacity)
	processStructEntityOrderedInto(entityMap, entity, metadata, expandOptions, selectedNavProps, baseURL, entitySetName, metadataLevel, fullMetadata, annotationFilter)
	return entityMap
}

// processStructEntityOrderedInto populates entityMap with the ordered representation of entity.
// entityMap must already be acquired by the caller, and may be pre-seeded with entries (such as
// "@odata.context") that need to appear before the metadata/property entries added here.
func processStructEntityOrderedInto(entityMap *OrderedMap, entity reflect.Value, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, baseURL, entitySetName string, metadataLevel string, fullMetadata *internalMetadata.EntityMetadata, annotationFilter *string) {
	fieldInfos := getFieldInfos(entity.Type())

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
				if annotationFilter != nil && !preference.MatchesAnnotationFilter(annotation.QualifiedTerm(), *annotationFilter) {
					continue
				}
				annotationKey := "@" + annotation.QualifiedTerm()
				entityMap.Set(annotationKey, annotation.Value)
			}
		}
	}

	// Process entity fields - optimized to reduce reflection calls
	// Cache property metadata lookups per entity type
	propMetaMap := getCachedPropertyMetadataMap(metadata)

	// Get the dual-keyed (Name + JsonName) property metadata map for annotation/enum lookups.
	// Cached per *EntityMetadata pointer — never rebuilt per entity.
	var fullPropMetaByName map[string]*internalMetadata.PropertyMetadata
	if fullMetadata != nil {
		fullPropMetaByName = getFullPropMetaByName(fullMetadata)
	}

	for j := 0; j < entity.NumField(); j++ {
		info := fieldInfos[j]
		if !info.IsExported {
			continue
		}
		if info.JsonName == "" {
			continue
		}

		propMeta := propMetaMap[info.Name]

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
			// Resolve the full property metadata once per field and reuse it below,
			// instead of re-hashing fullPropMetaByName for the stream check, the
			// annotations lookup, and enum formatting separately.
			var fullProp *internalMetadata.PropertyMetadata
			if fullPropMetaByName != nil {
				fullProp = fullPropMetaByName[info.Name]
			}

			// Skip stream properties — they are emitted as annotations below, not as inline values
			if fullProp != nil && fullProp.IsStream {
				continue
			}
			// Add property-level annotations first (for full metadata)
			if metadataLevel == "full" && fullProp != nil && fullProp.Annotations != nil && fullProp.Annotations.Len() > 0 {
				for _, annotation := range fullProp.Annotations.Get() {
					if annotationFilter != nil && !preference.MatchesAnnotationFilter(annotation.QualifiedTerm(), *annotationFilter) {
						continue
					}
					annotationKey := info.JsonName + "@" + annotation.QualifiedTerm()
					entityMap.Set(annotationKey, annotation.Value)
				}
			}
			// Then add the property value
			fieldValue := entity.Field(j)
			entityMap.Set(info.JsonName, EncodeEdmBinary(enumOrRaw(fieldValue, fullProp)))
		}
	}

	// Add named stream property annotations per OData spec §8.8
	if fullMetadata != nil && metadataLevel != "none" && keySegment != "" {
		for _, streamProp := range fullMetadata.StreamProperties {
			entityURL := baseURL + "/" + entitySetName + "(" + keySegment + ")"
			readLink := entityURL + "/" + streamProp.JsonName + "/$value"
			entityMap.Set(streamProp.JsonName+"@odata.mediaReadLink", readLink)

			if streamProp.StreamContentTypeField != "" {
				ctField := entity.FieldByName(streamProp.StreamContentTypeField)
				if ctField.IsValid() && ctField.Kind() == reflect.String {
					if ct := ctField.String(); ct != "" {
						entityMap.Set(streamProp.JsonName+"@odata.mediaContentType", ct)
					}
				}
			}
		}
	}
}

// WriteODataEntityFromNavigationPath writes a single-entity JSON response for an entity reached via a
// navigation path — for example, addressing a single member of a collection-valued navigation
// property by key (e.g. Categories(1)/Products(2)). Per OData v4.0 Part 2 §4.11, applying a key
// predicate to a collection-valued navigation property addresses a single entity within that
// collection, so the response must be a single-entity object (not a collection wrapped in
// "value": [...]), with @odata.context ending in "/$entity".
//
// contextPath is the URL path segment(s) preceding "/$entity" in the resulting @odata.context (e.g.
// "Categories(1)/Products(2)"). entitySetName, md, and fullMetadata describe the entity actually being
// written, which may belong to a different entity set than the one the request path started from.
func WriteODataEntityFromNavigationPath(w http.ResponseWriter, r *http.Request, contextPath string, entitySetName string, entity interface{}, md EntityMetadataProvider, fullMetadata *internalMetadata.EntityMetadata) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, r, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json and application/atom+xml are supported for data responses.")
	}

	metadataLevel := GetODataMetadataLevel(r)
	baseURL := buildBaseURL(r)

	entityValue := reflect.ValueOf(entity)
	for entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	var annotationFilter *string
	if pref := preference.ParsePrefer(r); pref.IncludeAnnotations != nil {
		annotationFilter = pref.IncludeAnnotations
	}

	capacity := entityValue.NumField() + 4
	entityMap := AcquireOrderedMapWithCapacity(capacity)
	if metadataLevel != "none" {
		entityMap.Set("@odata.context", baseURL+"/$metadata#"+contextPath+"/$entity")
	}
	processStructEntityOrderedInto(entityMap, entityValue, md, nil, nil, baseURL, entitySetName, metadataLevel, fullMetadata, annotationFilter)

	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))

	if r.Method == http.MethodHead {
		jsonBytes, err := entityMap.MarshalJSON()
		entityMap.Release()
		if err != nil {
			return err
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))
		w.WriteHeader(http.StatusOK)
		return nil
	}

	responseBytes, err := entityMap.MarshalJSON()
	entityMap.Release()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(responseBytes)))
	w.WriteHeader(http.StatusOK)
	_, writeErr := w.Write(responseBytes)
	return writeErr
}

func isPropertyExpanded(prop PropertyMetadata, expandOptions []query.ExpandOption) bool {
	return query.FindExpandOption(expandOptions, prop.Name, prop.JsonName) != nil
}

func processNavigationPropertyOrderedWithMetadata(entityMap *OrderedMap, entity reflect.Value, propMeta *PropertyMetadata, fieldValue reflect.Value, jsonName string, expandOpt *query.ExpandOption, selectedNavProps []string, baseURL, entitySetName string, metadata EntityMetadataProvider, metadataLevel string, keySegment string, fullMetadata *internalMetadata.EntityMetadata) {
	if expandOpt != nil {
		// Detect truncation for collection navigation properties: when $top is set and
		// the collection has top+1 items, the server fetched one extra to detect "has more".
		truncated := false
		if expandOpt.Top != nil && propMeta.NavigationIsArray {
			fieldValue, truncated = TruncateExpandedCollectionToTop(fieldValue, *expandOpt.Top)
		}

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

		// Emit @odata.nextLink when the expanded collection was truncated (OData v4 §11.2.5.7).
		if truncated && keySegment != "" {
			entityMap.Set(jsonName+"@odata.nextLink", BuildExpandedCollectionNextLink(baseURL, entitySetName, keySegment, jsonName, expandOpt))
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

// TruncateExpandedCollectionToTop truncates the collection value to top items.
// It returns the (possibly truncated) value and a boolean indicating whether truncation occurred.
// The caller passes top+1 items to signal "has more"; this function slices to top.
func TruncateExpandedCollectionToTop(v reflect.Value, top int) (reflect.Value, bool) {
	val := v
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return v, false
		}
		val = val.Elem()
	}
	if (val.Kind() == reflect.Slice || val.Kind() == reflect.Array) && val.Len() > top {
		return val.Slice(0, top), true
	}
	return v, false
}

// BuildExpandedCollectionNextLink constructs the @odata.nextLink URL for a truncated expanded
// collection. The URL points to the navigation property resource with $skip set to the next page.
func BuildExpandedCollectionNextLink(baseURL, entitySetName, keySegment, jsonName string, expandOpt *query.ExpandOption) string {
	skip := 0
	if expandOpt.Skip != nil {
		skip = *expandOpt.Skip
	}
	nextSkip := skip + *expandOpt.Top
	return fmt.Sprintf("%s/%s(%s)/%s?$skip=%d", baseURL, entitySetName, keySegment, jsonName, nextSkip)
}

// enumOrRaw returns the OData string representation for enum fields, or the raw value otherwise.
// prop is the field's already-resolved full property metadata (nil if unavailable), so callers
// that already looked it up don't pay for a second map lookup here.
func enumOrRaw(v reflect.Value, prop *internalMetadata.PropertyMetadata) interface{} {
	if prop != nil && prop.IsEnum && len(prop.EnumMembers) > 0 {
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return internalMetadata.EnumValueToString(v.Int(), prop.EnumMembers, prop.IsFlags)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return internalMetadata.EnumValueToString(int64(v.Uint()), prop.EnumMembers, prop.IsFlags)
		}
	}
	return v.Interface()
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

// fullPropMetaByNameCache caches the dual-keyed property metadata map per *EntityMetadata.
// Built once on first access; EntityMetadata is immutable after registration.
var fullPropMetaByNameCache sync.Map // map[*internalMetadata.EntityMetadata]map[string]*internalMetadata.PropertyMetadata

// getFullPropMetaByName returns a map resolving both Name and JsonName to *PropertyMetadata.
// The result is cached per *EntityMetadata pointer and never rebuilt.
func getFullPropMetaByName(fullMetadata *internalMetadata.EntityMetadata) map[string]*internalMetadata.PropertyMetadata {
	if cached, ok := fullPropMetaByNameCache.Load(fullMetadata); ok {
		return cached.(map[string]*internalMetadata.PropertyMetadata) //nolint:errcheck
	}
	m := make(map[string]*internalMetadata.PropertyMetadata, len(fullMetadata.Properties)*2)
	for i := range fullMetadata.Properties {
		prop := &fullMetadata.Properties[i]
		m[prop.Name] = prop
		if prop.JsonName != "" && prop.JsonName != prop.Name {
			m[prop.JsonName] = prop
		}
	}
	actual, _ := fullPropMetaByNameCache.LoadOrStore(fullMetadata, m)
	return actual.(map[string]*internalMetadata.PropertyMetadata) //nolint:errcheck
}

// buildKeySegmentFromEntityCached is an internal helper for building key segments
func buildKeySegmentFromEntityCached(entity reflect.Value, metadata EntityMetadataProvider) string {
	keyProps := metadata.GetKeyProperties()
	if len(keyProps) == 0 {
		return ""
	}

	entityType := entity.Type()

	if len(keyProps) == 1 {
		idx := getFieldIndexCached(entityType, keyProps[0].Name)
		if idx != nil {
			return formatKeyValue(entity.FieldByIndex(idx))
		}
		return ""
	}

	// For composite keys, build the key segment
	var builder strings.Builder
	// Estimate capacity: name=value, separated by commas
	builder.Grow(len(keyProps) * 20)

	for i, keyProp := range keyProps {
		idx := getFieldIndexCached(entityType, keyProp.Name)
		if idx == nil {
			continue
		}
		keyFieldValue := entity.FieldByIndex(idx)
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
