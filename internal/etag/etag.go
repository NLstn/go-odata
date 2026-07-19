package etag

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/nlstn/go-odata/internal/metadata"
)

// hexTable for fast hex encoding without allocation
const hexTable = "0123456789abcdef"

// timeType caches the reflect.Type for time.Time to avoid recomputing it on every call.
var timeType = reflect.TypeOf(time.Time{})

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

// appendHex16 appends v as 16 lowercase hex characters to dst.
func appendHex16(dst []byte, v uint64) []byte {
	return append(dst,
		hexTable[(v>>60)&0xf], hexTable[(v>>56)&0xf], hexTable[(v>>52)&0xf], hexTable[(v>>48)&0xf],
		hexTable[(v>>44)&0xf], hexTable[(v>>40)&0xf], hexTable[(v>>36)&0xf], hexTable[(v>>32)&0xf],
		hexTable[(v>>28)&0xf], hexTable[(v>>24)&0xf], hexTable[(v>>20)&0xf], hexTable[(v>>16)&0xf],
		hexTable[(v>>12)&0xf], hexTable[(v>>8)&0xf], hexTable[(v>>4)&0xf], hexTable[v&0xf],
	)
}

// Generate creates an ETag value for an entity based on its ETag property
// Returns an empty string if no ETag property is defined
// Uses xxhash64 for fast non-cryptographic hashing (ETag doesn't need cryptographic strength)
func Generate(entity interface{}, meta *metadata.EntityMetadata) string {
	hash, ok := etagHash(entity, meta)
	if !ok {
		return ""
	}

	// Get pre-allocated buffer from pool (pre-filled with W/"..." framing)
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

// AppendJSON appends the JSON-encoded ETag string (e.g. `"W/\"0123456789abcdef\""`)
// to dst and returns the extended slice with true. It returns (dst, false),
// writing nothing, when the entity has no resolvable ETag — matching Generate
// returning "". Emitting the pre-escaped JSON form directly lets response
// serializers write the ETag straight into their output buffer, avoiding both
// Generate's per-entity result string allocation and a second JSON-escaping pass
// (the ETag value always contains quotes, so it would otherwise route through the
// reflection encoder).
func AppendJSON(dst []byte, entity interface{}, meta *metadata.EntityMetadata) ([]byte, bool) {
	hash, ok := etagHash(entity, meta)
	if !ok {
		return dst, false
	}
	// JSON encoding of the ETag value W/"<hex>": the inner double quotes are
	// backslash-escaped, matching encoding/json's output for that string.
	dst = append(dst, '"', 'W', '/', '\\', '"')
	dst = appendHex16(dst, hash)
	dst = append(dst, '\\', '"', '"')
	return dst, true
}

// etagHash resolves the entity's ETag property value and returns its xxhash64,
// or (0, false) when no ETag property is defined or the field cannot be resolved.
// It is the shared core of Generate and AppendJSON.
func etagHash(entity interface{}, meta *metadata.EntityMetadata) (uint64, bool) {
	if meta.ETagProperty == nil {
		return 0, false
	}

	// Get the entity value
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Handle map entities (from $select operations)
	if entityValue.Kind() == reflect.Map {
		return etagHashFromMap(entity, meta)
	}

	// Use cached field index instead of FieldByName
	idx, found := getFieldIndex(entityValue.Type(), meta.ETagProperty.FieldName)
	if !found {
		return 0, false
	}

	fieldValue := entityValue.Field(idx)
	if !fieldValue.IsValid() {
		return 0, false
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
		if fieldValue.Type() == timeType {
			// Boxing a 24-byte time.Time value into an interface{} via fieldValue.Interface()
			// heap-allocates. Take its address instead (fields of a dereferenced pointer are
			// addressable) and box the pointer, which fits directly in the interface with no
			// extra allocation.
			if fieldValue.CanAddr() {
				t := fieldValue.Addr().Interface().(*time.Time) //nolint:errcheck // type guaranteed by the timeType check above
				etagSource = strconv.FormatInt(t.Unix(), 10)
			} else {
				t := fieldValue.Interface().(time.Time) //nolint:errcheck // type guaranteed by the timeType check above
				etagSource = strconv.FormatInt(t.Unix(), 10)
			}
		} else {
			etagSource = fmt.Sprintf("%v", fieldValue.Interface())
		}
	default:
		etagSource = fmt.Sprintf("%v", fieldValue.Interface())
	}

	return xxhash.Sum64String(etagSource), true
}

// etagHashFromMap resolves the ETag property from a map entity (from $select
// operations) and returns its xxhash64, or (0, false) when it cannot be resolved.
func etagHashFromMap(entity interface{}, meta *metadata.EntityMetadata) (uint64, bool) {
	entityMap, ok := entity.(map[string]interface{})
	if !ok {
		return 0, false
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
		return 0, false
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

	return xxhash.Sum64String(etagSource), true
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

// splitETags splits a comma-separated ETag list header value into individual ETag tokens.
// Leading and trailing whitespace is trimmed from each token.
func splitETags(header string) []string {
	parts := strings.Split(header, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// Match checks if the provided If-Match header value matches the current ETag.
// Returns true if ifMatch is empty (no precondition), if ifMatch is "*" and the
// entity exists, or if any ETag in the comma-separated list matches the current ETag.
func Match(ifMatch string, currentETag string) bool {
	if ifMatch == "" {
		return true // No If-Match header means no precondition
	}

	// "*" matches any ETag (entity must exist)
	if strings.TrimSpace(ifMatch) == "*" {
		return currentETag != ""
	}

	parsedCurrent := Parse(currentETag)
	for _, tag := range splitETags(ifMatch) {
		if Parse(tag) == parsedCurrent {
			return true
		}
	}
	return false
}

// NoneMatch checks if the provided If-None-Match header value does NOT match the current ETag.
// Returns true if they don't match or if ifNoneMatch is empty (no condition).
// Returns false if ifNoneMatch is "*" and the entity exists, or if any ETag in the
// comma-separated list matches the current ETag (meaning resource hasn't changed - 304).
func NoneMatch(ifNoneMatch string, currentETag string) bool {
	if ifNoneMatch == "" {
		return true // No If-None-Match header means no condition, proceed normally
	}

	// "*" matches any existing entity, so none-match is false if entity exists
	if strings.TrimSpace(ifNoneMatch) == "*" {
		return currentETag == ""
	}

	parsedCurrent := Parse(currentETag)
	for _, tag := range splitETags(ifNoneMatch) {
		if Parse(tag) == parsedCurrent {
			return false // A tag matched — resource hasn't changed
		}
	}
	return true
}
