package handlers

import (
	"reflect"
	"sort"
)

func annotationCollectionValues(value interface{}) ([]interface{}, bool) {
	if value == nil {
		return nil, false
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}

	if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
		return nil, false
	}

	values := make([]interface{}, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		values[i] = rv.Index(i).Interface()
	}
	return values, true
}

func annotationRecordValues(value interface{}) (map[string]interface{}, bool) {
	if value == nil {
		return nil, false
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}

	values := make(map[string]interface{}, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		values[iter.Key().String()] = iter.Value().Interface()
	}

	return values, true
}

func sortedAnnotationKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
