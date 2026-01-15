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
