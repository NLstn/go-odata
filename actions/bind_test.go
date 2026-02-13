package actions

import (
	"reflect"
	"strings"
	"testing"
)

type discountInput struct {
	Percentage float64 `mapstructure:"percentage"`
	Note       *string `mapstructure:"note,omitempty"`
}

func TestBindParams_StructAndPointer(t *testing.T) {
	note := "seasonal"
	params := map[string]interface{}{
		"percentage": 12.5,
		"note":       &note,
	}

	bound, err := BindParams[discountInput](params)
	if err != nil {
		t.Fatalf("BindParams() unexpected error: %v", err)
	}

	if bound.Percentage != 12.5 {
		t.Fatalf("BindParams() percentage = %v, want 12.5", bound.Percentage)
	}
	if bound.Note == nil || *bound.Note != note {
		t.Fatalf("BindParams() note = %#v, want %q", bound.Note, note)
	}

	ptrResult, err := BindParams[*discountInput](params)
	if err != nil {
		t.Fatalf("BindParams() pointer unexpected error: %v", err)
	}

	if ptrResult == nil {
		t.Fatal("BindParams() pointer result is nil")
		return
	}
	if ptrResult.Percentage != 12.5 {
		t.Fatalf("BindParams() pointer percentage = %v, want 12.5", ptrResult.Percentage)
	}
	if ptrResult.Note == nil {
		t.Fatalf("BindParams() pointer note is nil, want %q", note)
	}
	if *ptrResult.Note != note {
		t.Fatalf("BindParams() pointer note = %q, want %q", *ptrResult.Note, note)
	}
}

func TestBindParams_OptionalAndRequiredValidation(t *testing.T) {
	params := map[string]interface{}{
		"percentage": 30.0,
	}

	bound, err := BindParams[discountInput](params)
	if err != nil {
		t.Fatalf("BindParams() unexpected error for optional field: %v", err)
	}

	if bound.Note != nil {
		t.Fatalf("BindParams() expected nil optional field, got %#v", bound.Note)
	}

	_, err = BindParams[discountInput](map[string]interface{}{"note": "hello"})
	if err == nil {
		t.Fatal("BindParams() expected error for missing required field, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "required parameter 'percentage' is missing") {
		t.Fatalf("BindParams() error = %q, want missing required message", got)
	}
}

func TestBindParams_TypeMismatch(t *testing.T) {
	params := map[string]interface{}{
		"percentage": "not-a-number",
	}

	if _, err := BindParams[discountInput](params); err == nil {
		t.Fatal("BindParams() expected error for type mismatch, got nil")
	}
}

func TestBindParams_RepeatCallsUseConsistentBinding(t *testing.T) {
	params := map[string]interface{}{
		"percentage": 55.0,
	}

	first, err := BindParams[discountInput](params)
	if err != nil {
		t.Fatalf("BindParams() first call unexpected error: %v", err)
	}
	if first.Percentage != 55 {
		t.Fatalf("BindParams() first call percentage = %v, want 55", first.Percentage)
	}

	second, err := BindParams[discountInput](params)
	if err != nil {
		t.Fatalf("BindParams() second call unexpected error: %v", err)
	}
	if second.Percentage != 55 {
		t.Fatalf("BindParams() second call percentage = %v, want 55", second.Percentage)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("BindParams() first and second results differ: first=%#v, second=%#v", first, second)
	}

	ptrResult, err := BindParams[*discountInput](params)
	if err != nil {
		t.Fatalf("BindParams() pointer call unexpected error: %v", err)
	}
	if ptrResult == nil || ptrResult.Percentage != 55 {
		t.Fatalf("BindParams() pointer result = %#v, want percentage 55", ptrResult)
	}
}

func TestBindParams_NilParams(t *testing.T) {
	_, err := BindParams[discountInput](nil)
	if err == nil {
		t.Fatal("BindParams() expected error for nil params, got nil")
	}
	if got := err.Error(); got != "parameters map cannot be nil" {
		t.Fatalf("BindParams() error = %q, want %q", got, "parameters map cannot be nil")
	}
}

type cachedInput struct {
	Name string `json:"name"`
}

func TestBindParams_UsesCachedStruct(t *testing.T) {
	params := map[string]interface{}{
		"name": "primary",
	}

	first, err := BindParams[cachedInput](params)
	if err != nil {
		t.Fatalf("BindParams() first call error: %v", err)
	}
	if first.Name != "primary" {
		t.Fatalf("BindParams() first call name = %q, want %q", first.Name, "primary")
	}

	params["name"] = 123

	second, err := BindParams[cachedInput](params)
	if err != nil {
		t.Fatalf("BindParams() cached call error: %v", err)
	}
	if second.Name != "primary" {
		t.Fatalf("BindParams() cached call name = %q, want %q", second.Name, "primary")
	}
	if _, ok := params[BoundStructKey]; !ok {
		t.Fatal("BindParams() expected cached struct in params map")
	}
}

type EmbeddedInfo struct {
	EmbeddedName string `json:"embedded_name"`
	OptionalPtr  *int   `json:"optional_ptr"`
}

type bindingInput struct {
	EmbeddedInfo
	Title    string `mapstructure:"title"`
	Nickname string `json:"nickname,omitempty"`
	Alias    string `mapstructure:"alias" json:"alias_json"`
	Count    int
}

func TestCollectFieldBindings_Metadata(t *testing.T) {
	bindings, err := CollectFieldBindings(reflect.TypeOf(bindingInput{}))
	if err != nil {
		t.Fatalf("CollectFieldBindings() error: %v", err)
	}

	byName := make(map[string]StructFieldBinding, len(bindings))
	for _, binding := range bindings {
		byName[binding.Name] = binding
	}
	bindingNames := make([]string, 0, len(byName))
	for name := range byName {
		bindingNames = append(bindingNames, name)
	}

	expectRequired := map[string]bool{
		"embedded_name": true,
		"optional_ptr":  false,
		"title":         true,
		"nickname":      false,
		"alias":         true,
		"Count":         true,
	}

	for name, required := range expectRequired {
		binding, ok := byName[name]
		if !ok {
			t.Fatalf("CollectFieldBindings() missing %q binding (available: %v)", name, bindingNames)
		}
		if binding.Required != required {
			t.Fatalf("CollectFieldBindings() %q required = %v, want %v", name, binding.Required, required)
		}
	}

	if _, ok := byName["alias_json"]; ok {
		t.Fatal("CollectFieldBindings() should prefer mapstructure tag over json tag")
	}

	if len(byName["embedded_name"].Field.Index) <= 1 {
		t.Fatalf("CollectFieldBindings() embedded field index = %v, want nested index", byName["embedded_name"].Field.Index)
	}
	if len(byName["optional_ptr"].Field.Index) <= 1 {
		t.Fatalf("CollectFieldBindings() embedded field index = %v, want nested index", byName["optional_ptr"].Field.Index)
	}
}

func TestNewDecodeTarget(t *testing.T) {
	t.Run("struct type", func(t *testing.T) {
		value, err := NewDecodeTarget(reflect.TypeOf(discountInput{}))
		if err != nil {
			t.Fatalf("NewDecodeTarget() error: %v", err)
		}
		if value.Kind() != reflect.Ptr || value.Elem().Type() != reflect.TypeOf(discountInput{}) {
			t.Fatalf("NewDecodeTarget() = %v (%v), want pointer to %v", value, value.Type(), reflect.TypeOf(discountInput{}))
		}
	})

	t.Run("pointer to struct type", func(t *testing.T) {
		value, err := NewDecodeTarget(reflect.TypeOf(&discountInput{}))
		if err != nil {
			t.Fatalf("NewDecodeTarget() error: %v", err)
		}
		if value.Kind() != reflect.Ptr || value.Elem().Type() != reflect.TypeOf(discountInput{}) {
			t.Fatalf("NewDecodeTarget() = %v (%v), want pointer to %v", value, value.Type(), reflect.TypeOf(discountInput{}))
		}
	})

	t.Run("non struct types", func(t *testing.T) {
		nonStructTypes := []reflect.Type{reflect.TypeOf(0), reflect.TypeOf(new(int))}
		for _, tt := range nonStructTypes {
			_, err := NewDecodeTarget(tt)
			if err == nil {
				t.Fatalf("NewDecodeTarget(%v) expected error, got nil", tt)
			}
			if !strings.Contains(err.Error(), "BindParams target type must be a struct or pointer to struct") {
				t.Fatalf("NewDecodeTarget(%v) error = %q, want type error", tt, err)
			}
		}
	})
}

func TestNormalizeStructType(t *testing.T) {
	structType, err := NormalizeStructType(reflect.TypeOf(discountInput{}))
	if err != nil {
		t.Fatalf("NormalizeStructType() struct error: %v", err)
	}
	if structType != reflect.TypeOf(discountInput{}) {
		t.Fatalf("NormalizeStructType() struct = %v, want %v", structType, reflect.TypeOf(discountInput{}))
	}

	ptrType, err := NormalizeStructType(reflect.TypeOf(&discountInput{}))
	if err != nil {
		t.Fatalf("NormalizeStructType() pointer error: %v", err)
	}
	if ptrType != reflect.TypeOf(discountInput{}) {
		t.Fatalf("NormalizeStructType() pointer = %v, want %v", ptrType, reflect.TypeOf(discountInput{}))
	}

	_, err = NormalizeStructType(reflect.TypeOf(0))
	if err == nil {
		t.Fatal("NormalizeStructType() non-struct expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parameter binding target must be a struct or pointer to struct") {
		t.Fatalf("NormalizeStructType() non-struct error = %q, want type error", err)
	}
}

func TestResolveFieldValue(t *testing.T) {
	t.Run("non pointer target", func(t *testing.T) {
		field, ok := reflect.TypeOf(discountInput{}).FieldByName("Percentage")
		if !ok {
			t.Fatal("missing Percentage field")
		}

		_, err := ResolveFieldValue(reflect.ValueOf(discountInput{}), field)
		if err == nil {
			t.Fatal("ResolveFieldValue() expected error for non-pointer target, got nil")
		}
		if !strings.Contains(err.Error(), "target must be a pointer to struct") {
			t.Fatalf("ResolveFieldValue() error = %q, want pointer target error", err)
		}
	})

	t.Run("nil pointer target gets allocated", func(t *testing.T) {
		type outer struct {
			Inner *discountInput
		}

		field, ok := reflect.TypeOf(outer{}).FieldByName("Inner")
		if !ok {
			t.Fatal("missing Inner field")
		}

		var target *outer
		targetVal := reflect.ValueOf(&target).Elem()
		fieldVal, err := ResolveFieldValue(targetVal, field)
		if err != nil {
			t.Fatalf("ResolveFieldValue() error: %v", err)
		}

		if target == nil {
			t.Fatal("ResolveFieldValue() did not allocate nil pointer target")
		}
		if fieldVal.Kind() != reflect.Ptr || !fieldVal.IsNil() {
			t.Fatalf("ResolveFieldValue() field = %v, want nil pointer field", fieldVal)
		}
	})

	t.Run("invalid intermediate path", func(t *testing.T) {
		type invalidIntermediate struct {
			Value int
		}

		target := &invalidIntermediate{}
		_, err := ResolveFieldValue(reflect.ValueOf(target), reflect.StructField{Name: "Value", Index: []int{0, 0}})
		if err == nil {
			t.Fatal("ResolveFieldValue() expected error for invalid intermediate path, got nil")
		}
		if !strings.Contains(err.Error(), "intermediate field int is not addressable") {
			t.Fatalf("ResolveFieldValue() error = %q, want intermediate field error", err)
		}
	})
}

func TestApplyBindings(t *testing.T) {
	t.Run("success with mixed present and missing params", func(t *testing.T) {
		target := &assignTarget{}
		typeOfTarget := reflect.TypeOf(*target)
		nameField, _ := typeOfTarget.FieldByName("Name")
		countField, _ := typeOfTarget.FieldByName("Count")
		scoreField, _ := typeOfTarget.FieldByName("Score")

		bindings := []StructFieldBinding{
			{Field: nameField, Name: "name"},
			{Field: countField, Name: "count"},
			{Field: scoreField, Name: "score"},
		}

		err := ApplyBindings(reflect.ValueOf(target), bindings, map[string]interface{}{
			"name":  "widget",
			"score": 99.5,
		})
		if err != nil {
			t.Fatalf("ApplyBindings() error: %v", err)
		}

		if target.Name != "widget" {
			t.Fatalf("ApplyBindings() name = %q, want %q", target.Name, "widget")
		}
		if target.Count != 0 {
			t.Fatalf("ApplyBindings() count = %d, want unchanged zero", target.Count)
		}
		if target.Score != 99.5 {
			t.Fatalf("ApplyBindings() score = %v, want 99.5", target.Score)
		}
	})

	t.Run("assignment error wraps parameter name", func(t *testing.T) {
		target := &assignTarget{}
		countField, _ := reflect.TypeOf(*target).FieldByName("Count")

		err := ApplyBindings(reflect.ValueOf(target), []StructFieldBinding{{Field: countField, Name: "count"}}, map[string]interface{}{
			"count": "not-an-int",
		})
		if err == nil {
			t.Fatal("ApplyBindings() expected assignment error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to bind parameter 'count'") {
			t.Fatalf("ApplyBindings() error = %q, want wrapped parameter name", err)
		}
	})
}

type assignTarget struct {
	Name  string
	Count int
	Score float64
	Ptr   *int
}

func TestAssignValue_Conversions(t *testing.T) {
	target := assignTarget{}
	targetVal := reflect.ValueOf(&target).Elem()

	if err := AssignValue(targetVal.FieldByName("Name"), "alpha"); err != nil {
		t.Fatalf("AssignValue() direct assign error: %v", err)
	}
	if target.Name != "alpha" {
		t.Fatalf("AssignValue() direct assign name = %q, want %q", target.Name, "alpha")
	}

	if err := AssignValue(targetVal.FieldByName("Count"), int32(42)); err != nil {
		t.Fatalf("AssignValue() convertible error: %v", err)
	}
	if target.Count != 42 {
		t.Fatalf("AssignValue() convertible count = %d, want 42", target.Count)
	}

	ptr := 21
	if err := AssignValue(targetVal.FieldByName("Count"), &ptr); err != nil {
		t.Fatalf("AssignValue() pointer-to-value error: %v", err)
	}
	if target.Count != 21 {
		t.Fatalf("AssignValue() pointer-to-value count = %d, want 21", target.Count)
	}

	if err := AssignValue(targetVal.FieldByName("Ptr"), 88); err != nil {
		t.Fatalf("AssignValue() value-to-pointer error: %v", err)
	}
	if target.Ptr == nil || *target.Ptr != 88 {
		t.Fatalf("AssignValue() value-to-pointer ptr = %#v, want 88", target.Ptr)
	}

	if err := AssignValue(targetVal.FieldByName("Score"), "nope"); err == nil {
		t.Fatal("AssignValue() expected error for incompatible types, got nil")
	}
}
