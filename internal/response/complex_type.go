package response

import (
	"bytes"
	"encoding"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Complex-type serialization writes an embedded complex-type struct value
// directly into the response buffer from a cached, reflection-free field plan,
// using the same scalar/time writers marshalTo uses for the top-level entity.
// This replaces the encoding/json reflection fallback (the
// ptrEncoder->structEncoder hotspot in #841) without building an intermediate
// map or boxing scalar fields into interface{}.
//
// It is deliberately conservative: a struct type is written this way only when
// its JSON shape can be reproduced byte-for-byte from its exported fields. Types
// that customize their own JSON (json.Marshaler / encoding.TextMarshaler) or use
// anonymous embedding (which encoding/json promotes/flattens) are left to the
// reflection fallback, so output never changes for those.

type complexFieldPlan struct {
	jsonName  string
	index     int
	omitEmpty bool
}

// complexPlan is the cached serialization plan for one struct type: the exported
// fields to emit, in declaration order, with resolved JSON names and omitempty.
type complexPlan struct {
	fields []complexFieldPlan
}

// complexPlanCache memoizes plans (and non-serializable verdicts) per reflect.Type.
// A stored nil *complexPlan means "not directly serializable — use the fallback".
var complexPlanCache sync.Map // reflect.Type -> *complexPlan

var (
	jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	timeType          = reflect.TypeOf(time.Time{})
)

// getComplexPlan returns the cached plan for t, or nil if t must use the fallback.
func getComplexPlan(t reflect.Type) *complexPlan {
	if cached, ok := complexPlanCache.Load(t); ok {
		p, _ := cached.(*complexPlan) //nolint:errcheck // value type is guaranteed by our Store calls
		return p
	}
	p := buildComplexPlan(t)
	actual, _ := complexPlanCache.LoadOrStore(t, p)
	cp, _ := actual.(*complexPlan) //nolint:errcheck // value type is guaranteed by our Store calls
	return cp
}

// buildComplexPlan inspects t and returns a plan, or nil when t must be left to
// the reflection fallback to preserve exact encoding/json output.
func buildComplexPlan(t reflect.Type) *complexPlan {
	if t.Kind() != reflect.Struct || t == timeType {
		return nil
	}
	// Custom marshalers: let encoding/json invoke them so output is unchanged.
	if implementsMarshaler(t) {
		return nil
	}
	n := t.NumField()
	fields := make([]complexFieldPlan, 0, n)
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if f.Anonymous {
			// encoding/json promotes/flattens embedded fields with dominance
			// rules we don't replicate; defer the whole struct to reflection.
			return nil
		}
		if !f.IsExported() {
			continue
		}
		name, omitEmpty, skip := parseJSONTag(f)
		if skip {
			continue
		}
		fields = append(fields, complexFieldPlan{jsonName: name, index: i, omitEmpty: omitEmpty})
	}
	return &complexPlan{fields: fields}
}

func implementsMarshaler(t reflect.Type) bool {
	pt := reflect.PointerTo(t)
	return t.Implements(jsonMarshalerType) || pt.Implements(jsonMarshalerType) ||
		t.Implements(textMarshalerType) || pt.Implements(textMarshalerType)
}

// parseJSONTag mirrors encoding/json's tag handling for the subset we support:
// name override, the omitempty option, and json:"-" exclusion.
func parseJSONTag(f reflect.StructField) (name string, omitEmpty, skip bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}
	name = f.Name
	if tag == "" {
		return name, false, false
	}
	first := tag
	if idx := strings.IndexByte(tag, ','); idx != -1 {
		first = tag[:idx]
		for _, opt := range strings.Split(tag[idx+1:], ",") {
			if opt == "omitempty" {
				omitEmpty = true
			}
		}
	}
	if first != "" {
		name = first
	}
	return name, omitEmpty, false
}

// writeComplexValue writes v (a struct, or pointer to one) to buf using its
// cached plan, and reports whether it handled the value. It returns false
// (writing nothing) when v is not a directly serializable struct — a non-struct
// kind, or a type with a custom marshaler / anonymous embedding — so the caller
// can fall back to encoding/json. A nil pointer to a serializable struct is
// handled here as JSON null.
func writeComplexValue(buf *bytes.Buffer, v reflect.Value, enc **json.Encoder) (bool, error) {
	if !v.IsValid() {
		return false, nil
	}
	t := v.Type()
	isPtr := t.Kind() == reflect.Ptr
	elem := t
	if isPtr {
		elem = t.Elem()
	}
	plan := getComplexPlan(elem)
	if plan == nil {
		return false, nil
	}
	if isPtr {
		if v.IsNil() {
			buf.WriteString("null")
			return true, nil
		}
		v = v.Elem()
	}
	return true, writeComplexStruct(buf, v, plan, enc)
}

func writeComplexStruct(buf *bytes.Buffer, v reflect.Value, plan *complexPlan, enc **json.Encoder) error {
	buf.WriteByte('{')
	first := true
	for i := range plan.fields {
		f := &plan.fields[i]
		fv := v.Field(f.index)
		if f.omitEmpty && isEmptyValue(fv) {
			continue
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false
		writeJSONKey(buf, f.jsonName)
		buf.WriteByte(':')
		if err := writeComplexField(buf, fv, enc); err != nil {
			return err
		}
	}
	buf.WriteByte('}')
	return nil
}

// writeComplexField writes a single struct-field value, recursing into nested
// pointers and complex structs and using the allocation-free scalar/time
// writers. Anything it can't handle directly is routed to encoding/json.
func writeComplexField(buf *bytes.Buffer, v reflect.Value, enc **json.Encoder) error {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			buf.WriteString("null")
			return nil
		}
		return writeComplexField(buf, v.Elem(), enc)
	case reflect.String:
		return writeJSONString(buf, v.String(), enc)
	case reflect.Bool:
		if v.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		writeInt(buf, v.Int())
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		writeUint(buf, v.Uint())
		return nil
	case reflect.Float32, reflect.Float64:
		writeFloat(buf, v.Float())
		return nil
	case reflect.Struct:
		if v.Type() == timeType {
			if appendJSONTime(buf, v.Interface().(time.Time)) { //nolint:errcheck // guarded by type check
				return nil
			}
			return encodeFallback(buf, enc, v.Interface())
		}
		if plan := getComplexPlan(v.Type()); plan != nil {
			return writeComplexStruct(buf, v, plan, enc)
		}
		return encodeFallback(buf, enc, v.Interface())
	default:
		return encodeFallback(buf, enc, v.Interface())
	}
}

// isEmptyValue reports whether v is the zero value for its type, matching
// encoding/json's omitempty semantics.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}
