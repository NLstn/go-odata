package response

import (
	"encoding/json"
)

// OrderedMap maintains insertion order of keys
type OrderedMap struct {
	keys   []string
	values map[string]interface{}
}

// NewOrderedMap creates a new OrderedMap
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		keys:   make([]string, 0),
		values: make(map[string]interface{}),
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
func (om *OrderedMap) MarshalJSON() ([]byte, error) {
	buf := []byte("{")
	for i, key := range om.keys {
		if i > 0 {
			buf = append(buf, ',')
		}
		// Marshal the key
		keyBytes, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf = append(buf, keyBytes...)
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
