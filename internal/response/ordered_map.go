package response

import (
	"bytes"
	"encoding/json"
	"strconv"
	"sync"
	"unicode/utf8"
)

// bufferPool is a sync.Pool for reusing bytes.Buffer instances
var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := new(bytes.Buffer)
		buf.Grow(512) // Pre-allocate reasonable capacity for typical JSON responses
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
	// Ensure we have enough capacity
	if cap(om.keys) < capacity {
		om.keys = make([]string, 0, capacity)
	}
	// For the map, we can't resize it, but we can check if it needs recreation
	// Only recreate if significantly undersized (avoids thrashing)
	if len(om.values) == 0 && capacity > 16 {
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

// MarshalJSON implements json.Marshaler to maintain field order
// Optimized version using streaming encoder to avoid per-value allocations
func (om *OrderedMap) MarshalJSON() ([]byte, error) {
	if len(om.keys) == 0 {
		return []byte("{}"), nil
	}

	// Get buffer from pool
	buf := bufferPool.Get().(*bytes.Buffer) //nolint:errcheck // sync.Pool.Get() doesn't return error
	buf.Reset()
	defer func() {
		// Only return buffers to pool if they're not too large (< 64KB)
		// This prevents unbounded memory growth from large responses
		if buf.Cap() < 65536 {
			bufferPool.Put(buf)
		}
	}()

	// Estimate initial capacity
	estimatedSize := len(om.keys) * 100
	buf.Grow(estimatedSize)

	// Create a streaming encoder - this avoids intermediate allocations
	enc := json.NewEncoder(buf)
	// Disable HTML escaping for better performance (not needed for OData responses)
	enc.SetEscapeHTML(false)

	buf.WriteByte('{')

	for i, key := range om.keys {
		if i > 0 {
			buf.WriteByte(',')
		}

		// Optimize for simple string keys (no special characters)
		// This avoids the overhead of json.Marshal for keys
		if needsEscaping(key) {
			// Use json.Marshal for keys that need escaping
			keyBytes, err := json.Marshal(key)
			if err != nil {
				return nil, err
			}
			buf.Write(keyBytes)
		} else {
			// Fast path: write key directly with quotes
			buf.WriteByte('"')
			buf.WriteString(key)
			buf.WriteByte('"')
		}

		buf.WriteByte(':')

		// Use streaming encoder for values - avoids intermediate []byte allocation
		value := om.values[key]

		// Fast path for common types to avoid encoder overhead
		switch v := value.(type) {
		case string:
			if needsEscaping(v) {
				if err := enc.Encode(v); err != nil {
					return nil, err
				}
				// Remove trailing newline added by Encode
				buf.Truncate(buf.Len() - 1)
			} else {
				buf.WriteByte('"')
				buf.WriteString(v)
				buf.WriteByte('"')
			}
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
			// Fall back to streaming encoder for complex types
			if err := enc.Encode(value); err != nil {
				return nil, err
			}
			// Remove trailing newline added by Encode
			buf.Truncate(buf.Len() - 1)
		}
	}

	buf.WriteByte('}')

	// Create a copy of the buffer contents since we're reusing the buffer
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
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
