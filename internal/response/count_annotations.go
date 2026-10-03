package response

import (
	"reflect"
	"strconv"
	"strings"
)

// stringifyCountAnnotations handles nested counts produced by expansion and projection.
func stringifyCountAnnotations(value interface{}) {
	convert := func(key string, value interface{}) interface{} {
		if strings.HasSuffix(key, "@odata.count") {
			switch count := value.(type) {
			case int:
				return strconv.Itoa(count)
			case int64:
				return strconv.FormatInt(count, 10)
			}
		}
		stringifyCountAnnotations(value)
		return value
	}
	switch object := value.(type) {
	case *OrderedMap:
		for key, value := range object.values {
			object.values[key] = convert(key, value)
		}
	case map[string]interface{}:
		for key, value := range object {
			object[key] = convert(key, value)
		}
	default:
		v := reflect.ValueOf(value)
		if v.IsValid() && v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				stringifyCountAnnotations(v.Index(i).Interface())
			}
		}
	}
}
