package odata

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

// SliceFilterFunc evaluates a $filter expression against an item.
type SliceFilterFunc[T any] func(item T, filter *FilterExpression) (bool, error)

// ApplyQueryOptionsToSlice applies basic OData query options ($filter, $orderby, $skip, $top)
// to an in-memory slice. Ordering is based on struct field names or json tag values.
// The returned slice is a new slice and does not modify the input.
func ApplyQueryOptionsToSlice[T any](items []T, options *QueryOptions, filterFunc SliceFilterFunc[T]) ([]T, error) {
	if options == nil || len(items) == 0 {
		return append([]T(nil), items...), nil
	}

	result := append([]T(nil), items...)

	if options.Filter != nil {
		if filterFunc == nil {
			return nil, fmt.Errorf("filter query option requires a filter evaluator")
		}
		filtered := make([]T, 0, len(result))
		for _, item := range result {
			match, err := filterFunc(item, options.Filter)
			if err != nil {
				return nil, err
			}
			if match {
				filtered = append(filtered, item)
			}
		}
		result = filtered
	}

	if len(options.OrderBy) > 0 && len(result) > 1 {
		var sortErr error
		sort.SliceStable(result, func(i, j int) bool {
			if sortErr != nil {
				return false
			}
			left := result[i]
			right := result[j]
			for _, order := range options.OrderBy {
				cmp, err := compareOrderByValues(left, right, order.Property)
				if err != nil {
					sortErr = err
					return false
				}
				if cmp == 0 {
					continue
				}
				if order.Descending {
					return cmp > 0
				}
				return cmp < 0
			}
			return false
		})
		if sortErr != nil {
			return nil, sortErr
		}
	}

	if options.Skip != nil {
		if *options.Skip >= len(result) {
			return []T{}, nil
		}
		result = result[*options.Skip:]
	}

	if options.Top != nil && *options.Top < len(result) {
		result = result[:*options.Top]
	}

	return result, nil
}

func compareOrderByValues[T any](left, right T, property string) (int, error) {
	leftValue, ok := lookupPropertyValue(left, property)
	if !ok {
		return 0, fmt.Errorf("order by property %q not found", property)
	}
	rightValue, ok := lookupPropertyValue(right, property)
	if !ok {
		return 0, fmt.Errorf("order by property %q not found", property)
	}

	return compareValues(leftValue, rightValue)
}

func lookupPropertyValue(item interface{}, property string) (reflect.Value, bool) {
	value := reflect.ValueOf(item)
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		itemType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := itemType.Field(i)
			if !field.IsExported() {
				continue
			}
			jsonName := jsonFieldName(field)
			if jsonName == "-" {
				continue
			}
			if jsonName == "" {
				jsonName = field.Name
			}
			if jsonName == property || field.Name == property {
				return value.Field(i), true
			}
		}
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return reflect.Value{}, false
		}
		fieldValue := value.MapIndex(reflect.ValueOf(property))
		if fieldValue.IsValid() {
			return fieldValue, true
		}
	}

	return reflect.Value{}, false
}

func jsonFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return ""
	}
	parts := strings.Split(jsonTag, ",")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func compareValues(left, right reflect.Value) (int, error) {
	leftValue, leftNil := normalizeValue(left)
	rightValue, rightNil := normalizeValue(right)

	switch {
	case leftNil && rightNil:
		return 0, nil
	case leftNil:
		return -1, nil
	case rightNil:
		return 1, nil
	}

	if leftValue.Kind() != rightValue.Kind() {
		return 0, fmt.Errorf("cannot compare values of different kinds: %s and %s", leftValue.Kind(), rightValue.Kind())
	}

	switch leftValue.Kind() {
	case reflect.String:
		leftString := leftValue.String()
		rightString := rightValue.String()
		switch {
		case leftString < rightString:
			return -1, nil
		case leftString > rightString:
			return 1, nil
		default:
			return 0, nil
		}
	case reflect.Bool:
		leftBool := leftValue.Bool()
		rightBool := rightValue.Bool()
		switch {
		case leftBool == rightBool:
			return 0, nil
		case !leftBool && rightBool:
			return -1, nil
		default:
			return 1, nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		leftInt := leftValue.Int()
		rightInt := rightValue.Int()
		switch {
		case leftInt < rightInt:
			return -1, nil
		case leftInt > rightInt:
			return 1, nil
		default:
			return 0, nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		leftUint := leftValue.Uint()
		rightUint := rightValue.Uint()
		switch {
		case leftUint < rightUint:
			return -1, nil
		case leftUint > rightUint:
			return 1, nil
		default:
			return 0, nil
		}
	case reflect.Float32, reflect.Float64:
		leftFloat := leftValue.Float()
		rightFloat := rightValue.Float()
		switch {
		case leftFloat < rightFloat:
			return -1, nil
		case leftFloat > rightFloat:
			return 1, nil
		default:
			return 0, nil
		}
	case reflect.Struct:
		timeType := reflect.TypeOf(time.Time{})
		if leftValue.Type() == timeType && rightValue.Type() == timeType {
			leftTime := leftValue.Interface().(time.Time)
			rightTime := rightValue.Interface().(time.Time)
			switch {
			case leftTime.Before(rightTime):
				return -1, nil
			case leftTime.After(rightTime):
				return 1, nil
			default:
				return 0, nil
			}
		}
	}

	return 0, fmt.Errorf("unsupported order by value type: %s", leftValue.Type())
}

func normalizeValue(value reflect.Value) (reflect.Value, bool) {
	current := value
	for current.Kind() == reflect.Pointer || current.Kind() == reflect.Interface {
		if current.IsNil() {
			return reflect.Value{}, true
		}
		current = current.Elem()
	}

	if !current.IsValid() {
		return reflect.Value{}, true
	}

	return current, false
}
