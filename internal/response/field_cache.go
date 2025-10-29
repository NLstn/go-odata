package response

import (
	"reflect"
	"strings"
	"sync"
)

// fieldInfo caches parsed information about a struct field
type fieldInfo struct {
	JsonName   string
	IsExported bool
}

// structFieldCache caches field information for struct types
type structFieldCache struct {
	mu     sync.RWMutex
	fields map[reflect.Type][]fieldInfo
}

var globalFieldCache = &structFieldCache{
	fields: make(map[reflect.Type][]fieldInfo),
}

// getFieldInfos returns cached field information for a struct type
func getFieldInfos(t reflect.Type) []fieldInfo {
	// Fast path: read lock for cache hit
	globalFieldCache.mu.RLock()
	if infos, ok := globalFieldCache.fields[t]; ok {
		globalFieldCache.mu.RUnlock()
		return infos
	}
	globalFieldCache.mu.RUnlock()

	// Slow path: compute and cache
	globalFieldCache.mu.Lock()
	defer globalFieldCache.mu.Unlock()

	// Double-check after acquiring write lock
	if infos, ok := globalFieldCache.fields[t]; ok {
		return infos
	}

	// Compute field information
	numFields := t.NumField()
	infos := make([]fieldInfo, numFields)

	for i := 0; i < numFields; i++ {
		field := t.Field(i)
		infos[i] = fieldInfo{
			JsonName:   extractJsonFieldName(field),
			IsExported: field.IsExported(),
		}
	}

	globalFieldCache.fields[t] = infos
	return infos
}

// extractJsonFieldName extracts the JSON field name from struct tags
// This is an optimized version that avoids string allocations
func extractJsonFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return field.Name
	}

	// Fast path: check for comma (most common case)
	if idx := strings.IndexByte(jsonTag, ','); idx != -1 {
		if idx > 0 {
			return jsonTag[:idx]
		}
		return field.Name
	}

	// No comma found, return the whole tag if non-empty
	if jsonTag != "" {
		return jsonTag
	}

	return field.Name
}

// propertyMetadataCache caches property metadata lookups by field name
type propertyMetadataCache struct {
	mu    sync.RWMutex
	cache map[EntityMetadataProvider]map[string]*PropertyMetadata
}

var globalPropMetaCache = &propertyMetadataCache{
	cache: make(map[EntityMetadataProvider]map[string]*PropertyMetadata),
}

// getCachedPropertyMetadata gets cached property metadata for a field name
func getCachedPropertyMetadata(fieldName string, metadata EntityMetadataProvider) *PropertyMetadata {
	// Fast path: read lock for cache hit
	globalPropMetaCache.mu.RLock()
	if fieldMap, ok := globalPropMetaCache.cache[metadata]; ok {
		propMeta := fieldMap[fieldName]
		globalPropMetaCache.mu.RUnlock()
		return propMeta
	}
	globalPropMetaCache.mu.RUnlock()

	// Slow path: build cache for this metadata provider
	globalPropMetaCache.mu.Lock()
	defer globalPropMetaCache.mu.Unlock()

	// Double-check after acquiring write lock
	if fieldMap, ok := globalPropMetaCache.cache[metadata]; ok {
		return fieldMap[fieldName]
	}

	// Build cache for this metadata provider
	props := metadata.GetProperties()
	fieldMap := make(map[string]*PropertyMetadata, len(props))
	for i := range props {
		fieldMap[props[i].Name] = &props[i]
	}
	globalPropMetaCache.cache[metadata] = fieldMap

	return fieldMap[fieldName]
}
