package response

import "reflect"

// OmitNullValues removes properties with a null value from item, honoring the
// Prefer: omit-values=nulls preference (OData v4.01 §11.2.8.6). item must be an
// *OrderedMap or map[string]interface{}; any other type is a no-op. Nested
// OrderedMap/map values (e.g. expanded navigation properties) and slices of them
// are processed recursively, since the preference applies to every property in
// the response, not just the top level.
func OmitNullValues(item interface{}) {
	switch v := item.(type) {
	case *OrderedMap:
		omitNullValuesFromOrderedMap(v)
	case map[string]interface{}:
		omitNullValuesFromMap(v)
	}
}

func omitNullValuesFromOrderedMap(om *OrderedMap) {
	if om == nil {
		return
	}
	keys := append([]string(nil), om.keys...)
	for _, key := range keys {
		val := om.values[key]
		if isNullValue(val) {
			om.Delete(key)
			continue
		}
		omitNullValuesFromNested(val)
	}
}

func omitNullValuesFromMap(m map[string]interface{}) {
	for key, val := range m {
		if isNullValue(val) {
			delete(m, key)
			continue
		}
		omitNullValuesFromNested(val)
	}
}

func omitNullValuesFromNested(val interface{}) {
	switch v := val.(type) {
	case *OrderedMap:
		omitNullValuesFromOrderedMap(v)
	case map[string]interface{}:
		omitNullValuesFromMap(v)
	case []interface{}:
		for _, item := range v {
			omitNullValuesFromNested(item)
		}
	}
}

// isNullValue reports whether v represents a JSON null: either a nil interface,
// or a typed nil (nil pointer/map/slice) that would otherwise marshal to "null".
func isNullValue(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}
