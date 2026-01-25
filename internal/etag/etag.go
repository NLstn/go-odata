package etag

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/nlstn/go-odata/internal/metadata"
)

// hexTable for fast hex encoding without allocation
const hexTable = "0123456789abcdef"

// globalFieldIndexCache uses sync.Map for lock-free reads under high concurrency
// Keys are reflect.Type, values are map[string]int (field name to index)
var globalFieldIndexCache sync.Map

// getFieldIndex returns the cached field index for a type and field name
// Uses sync.Map for lock-free reads, eliminating RWMutex contention
func getFieldIndex(t reflect.Type, fieldName string) (int, bool) {
	// Fast path: lock-free read from sync.Map
	if cached, ok := globalFieldIndexCache.Load(t); ok {
		typeCache := cached.(map[string]int) //nolint:errcheck // type is guaranteed by our Store calls
		idx, found := typeCache[fieldName]
		return idx, found
	}

	// Slow path: build cache for this type
	typeCache := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		typeCache[t.Field(i).Name] = i
	}

	// Store and use result (LoadOrStore ensures we don't lose concurrent computations)
	actual, _ := globalFieldIndexCache.LoadOrStore(t, typeCache)
	actualCache := actual.(map[string]int) //nolint:errcheck // type is guaranteed by our Store calls
	idx, found := actualCache[fieldName]
	return idx, found
}

// etagBufferPool is a sync.Pool for reusing fixed-size byte buffers for ETag generation
// ETag format: W/"<16 hex chars>" = 20 bytes total (xxhash64 produces 8 bytes = 16 hex chars)
var etagBufferPool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate buffer for W/" + 16 hex chars + " = 20 bytes
		buf := make([]byte, 20)
		// Pre-fill the constant parts
		buf[0] = 'W'
		buf[1] = '/'
		buf[2] = '"'
		buf[19] = '"'
		return &buf
	},
}

// encodeHex16 encodes an 8-byte value as 16 hex characters into dst[3:19]
// This is a zero-allocation hex encoding for xxhash64 output
func encodeHex16(dst []byte, v uint64) {
	dst[3] = hexTable[(v>>60)&0xf]
	dst[4] = hexTable[(v>>56)&0xf]
	dst[5] = hexTable[(v>>52)&0xf]
	dst[6] = hexTable[(v>>48)&0xf]
	dst[7] = hexTable[(v>>44)&0xf]
	dst[8] = hexTable[(v>>40)&0xf]
	dst[9] = hexTable[(v>>36)&0xf]
	dst[10] = hexTable[(v>>32)&0xf]
	dst[11] = hexTable[(v>>28)&0xf]
	dst[12] = hexTable[(v>>24)&0xf]
	dst[13] = hexTable[(v>>20)&0xf]
	dst[14] = hexTable[(v>>16)&0xf]
	dst[15] = hexTable[(v>>12)&0xf]
	dst[16] = hexTable[(v>>8)&0xf]
	dst[17] = hexTable[(v>>4)&0xf]
	dst[18] = hexTable[v&0xf]
}

// Generate creates an ETag value for an entity based on its ETag property
// Returns an empty string if no ETag property is defined
// Uses xxhash64 for fast non-cryptographic hashing (ETag doesn't need cryptographic strength)
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

	// Generate xxhash64 of the ETag source (much faster than SHA256)
	hash := xxhash.Sum64String(etagSource)

	// Get pre-allocated buffer from pool
	bufPtr := etagBufferPool.Get().(*[]byte) //nolint:errcheck // sync.Pool.Get() doesn't return error
	buf := *bufPtr

	// Encode hash directly into buffer (zero allocation hex encoding)
	encodeHex16(buf, hash)

	// Create result string (single allocation)
	result := string(buf)

	// Return buffer to pool
	etagBufferPool.Put(bufPtr)

	return result
}

// generateFromMap generates an ETag from a map entity (from $select operations)
// Uses xxhash64 for fast hashing with zero-allocation hex encoding
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
	case int:
		etagSource = strconv.FormatInt(int64(v), 10)
	case int8:
		etagSource = strconv.FormatInt(int64(v), 10)
	case int16:
		etagSource = strconv.FormatInt(int64(v), 10)
	case int32:
		etagSource = strconv.FormatInt(int64(v), 10)
	case int64:
		etagSource = strconv.FormatInt(v, 10)
	case uint:
		etagSource = strconv.FormatUint(uint64(v), 10)
	case uint8:
		etagSource = strconv.FormatUint(uint64(v), 10)
	case uint16:
		etagSource = strconv.FormatUint(uint64(v), 10)
	case uint32:
		etagSource = strconv.FormatUint(uint64(v), 10)
	case uint64:
		etagSource = strconv.FormatUint(v, 10)
	case float32:
		etagSource = strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		etagSource = strconv.FormatFloat(v, 'f', -1, 64)
	case time.Time:
		etagSource = strconv.FormatInt(v.Unix(), 10)
	default:
		etagSource = fmt.Sprintf("%v", v)
	}

	// Generate xxhash64 of the ETag source (much faster than SHA256)
	hash := xxhash.Sum64String(etagSource)

	// Get pre-allocated buffer from pool
	bufPtr := etagBufferPool.Get().(*[]byte) //nolint:errcheck // sync.Pool.Get() doesn't return error
	buf := *bufPtr

	// Encode hash directly into buffer (zero allocation hex encoding)
	encodeHex16(buf, hash)

	// Create result string (single allocation)
	result := string(buf)

	// Return buffer to pool
	etagBufferPool.Put(bufPtr)

	return result
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
