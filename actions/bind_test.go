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
