package response

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"
)

// bufferPool is a sync.Pool for reusing bytes.Buffer instances
var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := new(bytes.Buffer)
		buf.Grow(16 * 1024) // Pre-allocate 16KB; collection responses grow into the same buffer via marshalTo
		return buf
	},
}

// orderedMapPool is a sync.Pool for reusing OrderedMap instances
// This significantly reduces allocation overhead for collection responses
var orderedMapPool = sync.Pool{
	New: func() interface{} {
		return &OrderedMap{
			keys:   make([]string, 0, 16),
			values: make(map[string]interface{}, 16),
		}
	},
}

// OrderedMap maintains insertion order of keys
type OrderedMap struct {
	keys   []string
	values map[string]interface{}
}

// NewOrderedMap creates a new OrderedMap
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		keys:   make([]string, 0, 16), // Pre-allocate for typical entity size (increased from 8)
		values: make(map[string]interface{}, 16),
	}
}

// NewOrderedMapWithCapacity creates a new OrderedMap with pre-allocated capacity
func NewOrderedMapWithCapacity(capacity int) *OrderedMap {
	return &OrderedMap{
		keys:   make([]string, 0, capacity),
		values: make(map[string]interface{}, capacity),
	}
}

// AcquireOrderedMap gets an OrderedMap from the pool
// The returned map is reset and ready for use
func AcquireOrderedMap() *OrderedMap {
	om := orderedMapPool.Get().(*OrderedMap) //nolint:errcheck // sync.Pool.Get() doesn't return error
	return om
}

// AcquireOrderedMapWithCapacity gets an OrderedMap from the pool and ensures capacity
// The returned map is reset and ready for use
func AcquireOrderedMapWithCapacity(capacity int) *OrderedMap {
	om := orderedMapPool.Get().(*OrderedMap) //nolint:errcheck // sync.Pool.Get() doesn't return error
	// Ensure we have enough capacity for keys slice
	if cap(om.keys) < capacity {
		om.keys = make([]string, 0, capacity)
	}
	// Recreate map only if significantly undersized to avoid thrashing
	// Check current map capacity vs needed - maps can't be resized, only recreated
	if capacity > 16 && len(om.values) == 0 {
		om.values = make(map[string]interface{}, capacity)
	}
	return om
}

// Release returns the OrderedMap to the pool for reuse
// After calling Release, the OrderedMap must not be used
func (om *OrderedMap) Release() {
	if om == nil {
		return
	}
	om.Reset()
	// Only return to pool if not too large (prevents memory bloat)
	if cap(om.keys) <= 128 {
		orderedMapPool.Put(om)
	}
}

// Reset clears the OrderedMap for reuse
func (om *OrderedMap) Reset() {
	// Clear the keys slice (reuse underlying array)
	om.keys = om.keys[:0]
	// Clear the map (delete all entries, reuse map structure)
	for k := range om.values {
		delete(om.values, k)
	}
}

// Set adds or updates a key-value pair in the ordered map
func (om *OrderedMap) Set(key string, value interface{}) {
	if _, exists := om.values[key]; !exists {
		om.keys = append(om.keys, key)
	}
	om.values[key] = value
}

// Delete removes a key-value pair from the ordered map
func (om *OrderedMap) Delete(key string) {
	if _, exists := om.values[key]; exists {
		delete(om.values, key)
		// Remove from keys slice
		for i, k := range om.keys {
			if k == key {
				om.keys = append(om.keys[:i], om.keys[i+1:]...)
				break
			}
		}
	}
}

// InsertAfter inserts a key-value pair after the specified key
// If afterKey is empty or not found, appends to the end
func (om *OrderedMap) InsertAfter(afterKey, key string, value interface{}) {
	// If key already exists, delete it first
	om.Delete(key)

	// Find the position of afterKey
	position := -1
	for i, k := range om.keys {
		if k == afterKey {
			position = i
			break
		}
	}

	if position == -1 {
		// afterKey not found, append to end
		om.keys = append(om.keys, key)
	} else {
		// Insert after the found position
		om.keys = append(om.keys[:position+1], append([]string{key}, om.keys[position+1:]...)...)
	}

	om.values[key] = value
}

// ToMap returns the underlying map (loses ordering)
func (om *OrderedMap) ToMap() map[string]interface{} {
	return om.values
}

// MarshalJSON implements json.Marshaler to maintain field order.
func (om *OrderedMap) MarshalJSON() ([]byte, error) {
	buf, err := om.marshalToPooledBuffer()
	if err != nil {
		return nil, err
	}
	defer releasePooledBuffer(buf)

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// marshalToPooledBuffer serializes the ordered map into a buffer obtained from bufferPool.
// Callers that only need to write the bytes out (e.g. to an http.ResponseWriter) can use
// buf.Bytes() directly and avoid the copy-out allocation that MarshalJSON performs solely
// to satisfy the json.Marshaler interface contract of returning an owned []byte. The
// returned buffer must be released via releasePooledBuffer once its bytes are no longer
// needed.
func (om *OrderedMap) marshalToPooledBuffer() (*bytes.Buffer, error) {
	buf := bufferPool.Get().(*bytes.Buffer) //nolint:errcheck // sync.Pool.Get() doesn't return error
	buf.Reset()

	if len(om.keys) == 0 {
		buf.WriteString("{}")
		return buf, nil
	}

	buf.Grow(len(om.keys) * 100)

	if err := om.marshalTo(buf); err != nil {
		releasePooledBuffer(buf)
		return nil, err
	}

	return buf, nil
}

// releasePooledBuffer returns buf to bufferPool. Buffers that grew past 512KB are dropped
// instead of pooled, matching the size cap marshalToPooledBuffer's callers rely on to avoid
// bloating the pool with oversized buffers from unusually large responses.
func releasePooledBuffer(buf *bytes.Buffer) {
	if buf.Cap() < 512*1024 {
		bufferPool.Put(buf)
	}
}

// marshalTo writes the JSON representation directly to buf.
// Nested *OrderedMap and []interface{} values write into the same buf, eliminating
// the per-entity bufferPool round-trip and intermediate copy that MarshalJSON performs.
func (om *OrderedMap) marshalTo(buf *bytes.Buffer) error {
	// enc is created lazily, only when a value needs the reflection-based
	// encoding/json fallback (decimal, UUID, arbitrary structs, ...). Entities
	// composed entirely of scalars, time.Time, pointers-to-scalars and nested
	// *OrderedMap values never allocate an encoder.
	var enc *json.Encoder

	buf.WriteByte('{')

	for i, key := range om.keys {
		if i > 0 {
			buf.WriteByte(',')
		}

		if needsEscaping(key) {
			keyBytes, err := json.Marshal(key)
			if err != nil {
				return err
			}
			buf.Write(keyBytes)
		} else {
			buf.WriteByte('"')
			buf.WriteString(key)
			buf.WriteByte('"')
		}
		buf.WriteByte(':')

		value := om.values[key]

		// Dereference the common nullable-scalar pointer types up front so the
		// main switch below handles them without falling through to reflection.
		// A nil pointer serializes as JSON null.
		switch p := value.(type) {
		case *string:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		case *int:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		case *int64:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		case *int32:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		case *uint:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		case *uint64:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		case *uint32:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		case *float64:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		case *float32:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		case *bool:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		case *time.Time:
			if p == nil {
				buf.WriteString("null")
				continue
			}
			value = *p
		}

		switch v := value.(type) {
		case string:
			if err := writeJSONString(buf, v, &enc); err != nil {
				return err
			}
		case time.Time:
			if !appendJSONTime(buf, v) {
				// Years outside [0,9999] can't be represented as strict RFC 3339;
				// defer to encoding/json, which reports the same error stdlib does.
				if err := encodeFallback(buf, &enc, v); err != nil {
					return err
				}
			}
		case *OrderedMap:
			if v == nil {
				buf.WriteString("null")
			} else {
				if err := v.marshalTo(buf); err != nil {
					return err
				}
			}
		case []interface{}:
			// Write *OrderedMap elements directly into buf to avoid per-entity
			// bufferPool round-trips and intermediate copies.
			buf.WriteByte('[')
			for j, item := range v {
				if j > 0 {
					buf.WriteByte(',')
				}
				if elem, ok := item.(*OrderedMap); ok {
					if elem == nil {
						buf.WriteString("null")
					} else {
						if err := elem.marshalTo(buf); err != nil {
							return err
						}
					}
				} else {
					if err := encodeFallback(buf, &enc, item); err != nil {
						return err
					}
				}
			}
			buf.WriteByte(']')
		case int:
			writeInt(buf, int64(v))
		case int64:
			writeInt(buf, v)
		case int32:
			writeInt(buf, int64(v))
		case uint:
			writeUint(buf, uint64(v))
		case uint64:
			writeUint(buf, v)
		case uint32:
			writeUint(buf, uint64(v))
		case float64:
			writeFloat(buf, v)
		case float32:
			writeFloat(buf, float64(v))
		case bool:
			if v {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}
		case nil:
			buf.WriteString("null")
		default:
			// Complex-type (embedded struct) values are written directly from a
			// cached field plan; only shapes the plan can't reproduce exactly fall
			// through to the reflection-based encoding/json encoder.
			if handled, err := writeComplexValue(buf, reflect.ValueOf(value), &enc); handled {
				if err != nil {
					return err
				}
			} else if err := encodeFallback(buf, &enc, value); err != nil {
				return err
			}
		}
	}

	buf.WriteByte('}')
	return nil
}

// encodeFallback serializes val through the reflection-based encoding/json encoder,
// creating the encoder on first use so callers pay the allocation only when a
// value actually needs the fallback. It strips the trailing newline that
// (*json.Encoder).Encode appends so the output composes with surrounding JSON.
func encodeFallback(buf *bytes.Buffer, enc **json.Encoder, val interface{}) error {
	if *enc == nil {
		e := json.NewEncoder(buf)
		e.SetEscapeHTML(false)
		*enc = e
	}
	if err := (*enc).Encode(val); err != nil {
		return err
	}
	buf.Truncate(buf.Len() - 1)
	return nil
}

// appendJSONTime writes t as a JSON string using the same strict RFC 3339 (nano)
// representation that time.Time.MarshalJSON produces, directly into buf without
// allocating. It returns false (writing nothing) when the year falls outside
// [0,9999], which strict RFC 3339 cannot represent; callers then fall back to
// encoding/json to reproduce stdlib's error behavior exactly.
func appendJSONTime(buf *bytes.Buffer, t time.Time) bool {
	if y := t.Year(); y < 0 || y > 9999 {
		return false
	}
	var tmp [len(time.RFC3339Nano) + 2]byte
	b := tmp[:0]
	b = append(b, '"')
	b = t.AppendFormat(b, time.RFC3339Nano)
	b = append(b, '"')
	buf.Write(b)
	return true
}

// writeInt writes an int64 to the buffer without allocation
func writeInt(buf *bytes.Buffer, v int64) {
	var b [20]byte // max int64 is 19 digits + sign
	i := len(b)
	neg := v < 0
	if neg {
		v = -v
	}
	for v >= 10 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	i--
	b[i] = byte('0' + v)
	if neg {
		i--
		b[i] = '-'
	}
	buf.Write(b[i:])
}

// writeUint writes a uint64 to the buffer without allocation
func writeUint(buf *bytes.Buffer, v uint64) {
	var b [20]byte // max uint64 is 20 digits
	i := len(b)
	for v >= 10 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	i--
	b[i] = byte('0' + v)
	buf.Write(b[i:])
}

// writeFloat writes a float64 to the buffer
func writeFloat(buf *bytes.Buffer, v float64) {
	// Use strconv for proper float formatting
	buf.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
}

// writeJSONKey writes a JSON object key (quoted string) to buf, escaping only
// when necessary. Mirrors the key handling inlined in marshalTo.
func writeJSONKey(buf *bytes.Buffer, key string) {
	if needsEscaping(key) {
		keyBytes, err := json.Marshal(key)
		if err == nil {
			buf.Write(keyBytes)
			return
		}
	}
	buf.WriteByte('"')
	buf.WriteString(key)
	buf.WriteByte('"')
}

// writeJSONString writes a JSON string value to buf. Strings that need escaping
// are routed through the encoder (which handles escaping identically to the rest
// of the response); the common no-escape case is written directly.
func writeJSONString(buf *bytes.Buffer, s string, enc **json.Encoder) error {
	if needsEscaping(s) {
		return encodeFallback(buf, enc, s)
	}
	buf.WriteByte('"')
	buf.WriteString(s)
	buf.WriteByte('"')
	return nil
}

// needsEscaping checks if a string needs JSON escaping
// Returns true if the string contains characters that need escaping
func needsEscaping(s string) bool {
	for i := 0; i < len(s); {
		c := s[i]
		if c < 0x20 || c == '"' || c == '\\' {
			return true
		}
		if c < utf8.RuneSelf {
			i++
			continue
		}
		// Multi-byte UTF-8 characters are safe
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return false
}
