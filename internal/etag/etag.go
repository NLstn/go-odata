package etag

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
)

// fieldIndexCache caches field indices by type and field name
type fieldIndexCache struct {
	mu       sync.RWMutex
	cache    map[reflect.Type]map[string]int
	maxTypes int
}

var globalFieldIndexCache = &fieldIndexCache{
	cache:    make(map[reflect.Type]map[string]int),
	maxTypes: 1000, // Limit cache to 1000 types to prevent unbounded growth
}

// getFieldIndex returns the cached field index for a type and field name
func getFieldIndex(t reflect.Type, fieldName string) (int, bool) {
	globalFieldIndexCache.mu.RLock()
	if typeCache, ok := globalFieldIndexCache.cache[t]; ok {
		idx, found := typeCache[fieldName]
		globalFieldIndexCache.mu.RUnlock()
		return idx, found
	}
	globalFieldIndexCache.mu.RUnlock()

	globalFieldIndexCache.mu.Lock()
	defer globalFieldIndexCache.mu.Unlock()

	// Double-check after acquiring write lock
	if typeCache, ok := globalFieldIndexCache.cache[t]; ok {
		idx, found := typeCache[fieldName]
		return idx, found
	}

	// Check cache size limit to prevent unbounded growth
	if len(globalFieldIndexCache.cache) >= globalFieldIndexCache.maxTypes {
		// Clear cache when limit is reached (simple eviction strategy)
		globalFieldIndexCache.cache = make(map[reflect.Type]map[string]int)
	}

	// Build cache for this type
	typeCache := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		typeCache[t.Field(i).Name] = i
	}
	globalFieldIndexCache.cache[t] = typeCache

	idx, found := typeCache[fieldName]
	return idx, found
}

// stringBuilderPool is a sync.Pool for reusing strings.Builder instances
var stringBuilderPool = sync.Pool{
	New: func() interface{} {
		sb := &strings.Builder{}
		sb.Grow(68) // Pre-allocate for "W/\"" + 64 hex chars + "\"" = 68 bytes
		return sb
	},
}

// Generate creates an ETag value for an entity based on its ETag property
// Returns an empty string if no ETag property is defined
func Generate(entity interface{}, meta *metadata.EntityMetadata) string {
	if meta.ETagProperty == nil {
		return ""
	}

	// Get the entity value
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Handle map entities (from $select operations)
	if entityValue.Kind() == reflect.Map {
		return generateFromMap(entity, meta)
	}

	// Use cached field index instead of FieldByName
	idx, found := getFieldIndex(entityValue.Type(), meta.ETagProperty.FieldName)
	if !found {
		return ""
	}
	
	fieldValue := entityValue.Field(idx)
	if !fieldValue.IsValid() {
		return ""
	}

	// Convert the field value to a string for hashing
	var etagSource string
	switch fieldValue.Kind() {
	case reflect.String:
		etagSource = fieldValue.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		etagSource = strconv.FormatInt(fieldValue.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		etagSource = strconv.FormatUint(fieldValue.Uint(), 10)
	case reflect.Struct:
		// Handle time.Time specially
		if t, ok := fieldValue.Interface().(time.Time); ok {
			etagSource = strconv.FormatInt(t.Unix(), 10)
		} else {
			etagSource = fmt.Sprintf("%v", fieldValue.Interface())
		}
	default:
		etagSource = fmt.Sprintf("%v", fieldValue.Interface())
	}

	// Generate SHA256 hash of the ETag source
	hash := sha256.Sum256([]byte(etagSource))
	
	// Use strings.Builder from pool for efficient string concatenation
	sb := stringBuilderPool.Get().(*strings.Builder) //nolint:errcheck // sync.Pool.Get() doesn't return error
	sb.Reset()
	sb.Grow(68) // Pre-allocate for "W/\"" + 64 hex chars + "\"" = 68 bytes
	defer func() {
		// Only return builders to pool if they're not too large (< 1KB)
		// This prevents unbounded memory growth from large strings
		if sb.Cap() < 1024 {
			stringBuilderPool.Put(sb)
		}
	}()
	
	sb.WriteString("W/\"")
	sb.WriteString(hex.EncodeToString(hash[:]))
	sb.WriteString("\"")
	
	return sb.String()
}

// generateFromMap generates an ETag from a map entity (from $select operations)
func generateFromMap(entity interface{}, meta *metadata.EntityMetadata) string {
	entityMap, ok := entity.(map[string]interface{})
	if !ok {
		return ""
	}

	// Try to get the ETag field value using JsonName first, then FieldName
	var fieldValue interface{}
	var found bool

	// Try JsonName
	if meta.ETagProperty.JsonName != "" {
		fieldValue, found = entityMap[meta.ETagProperty.JsonName]
	}

	// If not found, try FieldName
	if !found && meta.ETagProperty.FieldName != "" {
		fieldValue, found = entityMap[meta.ETagProperty.FieldName]
	}

	if !found || fieldValue == nil {
		return ""
	}

	// Convert the field value to a string for hashing
	var etagSource string
	switch v := fieldValue.(type) {
	case string:
		etagSource = v
	case int, int8, int16, int32, int64:
		etagSource = fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		etagSource = fmt.Sprintf("%d", v)
	case float32, float64:
		etagSource = fmt.Sprintf("%v", v)
	case time.Time:
		etagSource = strconv.FormatInt(v.Unix(), 10)
	default:
		etagSource = fmt.Sprintf("%v", v)
	}

	// Generate SHA256 hash of the ETag source
	hash := sha256.Sum256([]byte(etagSource))
	hashStr := hex.EncodeToString(hash[:])

	// Return as quoted ETag (weak ETag format: W/"hash")
	return fmt.Sprintf("W/\"%s\"", hashStr)
}

// Parse extracts the ETag value from a quoted ETag string
// Handles both strong ("value") and weak (W/"value") ETags
func Parse(etagHeader string) string {
	if etagHeader == "" {
		return ""
	}

	// Remove W/ prefix if present (weak ETag)
	if len(etagHeader) > 2 && etagHeader[:2] == "W/" {
		etagHeader = etagHeader[2:]
	}

	// Remove quotes
	if len(etagHeader) >= 2 && etagHeader[0] == '"' && etagHeader[len(etagHeader)-1] == '"' {
		return etagHeader[1 : len(etagHeader)-1]
	}

	return etagHeader
}

// Match checks if the provided If-Match header value matches the current ETag
// Returns true if they match or if ifMatch is "*" (match any)
func Match(ifMatch string, currentETag string) bool {
	if ifMatch == "" {
		return true // No If-Match header means no precondition
	}

	// "*" matches any ETag (entity must exist)
	if ifMatch == "*" {
		return currentETag != ""
	}

	// Parse both ETags and compare
	parsedIfMatch := Parse(ifMatch)
	parsedCurrent := Parse(currentETag)

	return parsedIfMatch == parsedCurrent
}

// NoneMatch checks if the provided If-None-Match header value does NOT match the current ETag
// Returns true if they don't match or if ifNoneMatch is empty (no condition)
// Returns false if they match (meaning resource hasn't changed - should return 304)
// The "*" wildcard means "match if entity exists" for If-None-Match
func NoneMatch(ifNoneMatch string, currentETag string) bool {
	if ifNoneMatch == "" {
		return true // No If-None-Match header means no condition, proceed normally
	}

	// "*" matches any existing entity, so none-match is false if entity exists
	if ifNoneMatch == "*" {
		return currentETag == ""
	}

	// Parse both ETags and compare
	parsedIfNoneMatch := Parse(ifNoneMatch)
	parsedCurrent := Parse(currentETag)

	// Return true if they DON'T match (proceed with normal response)
	// Return false if they DO match (should return 304)
	return parsedIfNoneMatch != parsedCurrent
}
