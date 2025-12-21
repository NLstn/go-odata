package response

import (
	"bytes"
	"encoding/json"
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
// Optimized version using sync.Pool for buffer reuse
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

		// Marshal the value
		valueBytes, err := json.Marshal(om.values[key])
		if err != nil {
			return nil, err
		}
		buf.Write(valueBytes)
	}

	buf.WriteByte('}')
	
	// Create a copy of the buffer contents since we're reusing the buffer
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
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
