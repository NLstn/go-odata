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

// globalFieldCache uses sync.Map for lock-free reads under high concurrency
// sync.Map is optimized for cases where entries are written once and read many times
var globalFieldCache sync.Map // map[reflect.Type][]fieldInfo

// getFieldInfos returns cached field information for a struct type
// Uses sync.Map for lock-free reads, eliminating RWMutex contention
func getFieldInfos(t reflect.Type) []fieldInfo {
	// Fast path: lock-free read from sync.Map
	if cached, ok := globalFieldCache.Load(t); ok {
		return cached.([]fieldInfo) //nolint:errcheck // type is guaranteed by our Store calls
	}

	// Slow path: compute field information
	// sync.Map handles concurrent writes safely
	numFields := t.NumField()
	infos := make([]fieldInfo, numFields)

	for i := 0; i < numFields; i++ {
		field := t.Field(i)
		infos[i] = fieldInfo{
			JsonName:   extractJsonFieldName(field),
			IsExported: field.IsExported(),
		}
	}

	// Store and return (LoadOrStore ensures we don't lose concurrent computations)
	actual, _ := globalFieldCache.LoadOrStore(t, infos)
	return actual.([]fieldInfo) //nolint:errcheck // type is guaranteed by our Store calls
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

// globalPropMetaCache uses sync.Map for lock-free reads under high concurrency
// Keys are EntityMetadataProvider, values are map[string]*PropertyMetadata
var globalPropMetaCache sync.Map

// getCachedPropertyMetadataMap returns the entire property metadata map for a metadata provider
// Uses sync.Map for lock-free reads, eliminating RWMutex contention
func getCachedPropertyMetadataMap(metadata EntityMetadataProvider) map[string]*PropertyMetadata {
	// Fast path: lock-free read from sync.Map
	if cached, ok := globalPropMetaCache.Load(metadata); ok {
		return cached.(map[string]*PropertyMetadata) //nolint:errcheck // type is guaranteed by our Store calls
	}

	// Slow path: build cache for this metadata provider
	props := metadata.GetProperties()
	fieldMap := make(map[string]*PropertyMetadata, len(props))
	for i := range props {
		fieldMap[props[i].Name] = &props[i]
	}

	// Store and return (LoadOrStore ensures we don't lose concurrent computations)
	actual, _ := globalPropMetaCache.LoadOrStore(metadata, fieldMap)
	return actual.(map[string]*PropertyMetadata) //nolint:errcheck // type is guaranteed by our Store calls
}
