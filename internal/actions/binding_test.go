package actions

import (
	"reflect"
	"testing"
)

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
