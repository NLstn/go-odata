package response

import (
	"encoding/json"
	"unicode/utf8"
)

// OrderedMap maintains insertion order of keys
type OrderedMap struct {
	keys   []string
	values map[string]interface{}
}

// NewOrderedMap creates a new OrderedMap
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		keys:   make([]string, 0, 8), // Pre-allocate for typical entity size
		values: make(map[string]interface{}, 8),
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
// Optimized version that pre-allocates buffer and avoids unnecessary marshaling
func (om *OrderedMap) MarshalJSON() ([]byte, error) {
	if len(om.keys) == 0 {
		return []byte("{}"), nil
	}

	// Estimate buffer size: average 50 bytes per field (key + value + formatting)
	estimatedSize := len(om.keys) * 50
	buf := make([]byte, 0, estimatedSize)
	buf = append(buf, '{')

	for i, key := range om.keys {
		if i > 0 {
			buf = append(buf, ',')
		}

		// Optimize for simple string keys (no special characters)
		// This avoids the overhead of json.Marshal for keys
		if needsEscaping(key) {
			// Use json.Marshal for keys that need escaping
			keyBytes, err := json.Marshal(key)
			if err != nil {
				return nil, err
			}
			buf = append(buf, keyBytes...)
		} else {
			// Fast path: write key directly with quotes
			buf = append(buf, '"')
			buf = append(buf, key...)
			buf = append(buf, '"')
		}

		buf = append(buf, ':')

		// Marshal the value
		valueBytes, err := json.Marshal(om.values[key])
		if err != nil {
			return nil, err
		}
		buf = append(buf, valueBytes...)
	}

	buf = append(buf, '}')
	return buf, nil
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
