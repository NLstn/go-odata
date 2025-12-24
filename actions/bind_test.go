package actions

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	internalactions "github.com/nlstn/go-odata/internal/actions"
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
	if ptrResult.Note == nil || *ptrResult.Note != note {
		t.Fatalf("BindParams() pointer note = %#v, want %q", ptrResult.Note, note)
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

func TestBindParams_UsesExistingBinding(t *testing.T) {
	params := map[string]interface{}{
		boundStructKey: &discountInput{Percentage: 55},
	}

	result, err := BindParams[discountInput](params)
	if err != nil {
		t.Fatalf("BindParams() unexpected error: %v", err)
	}
	if result.Percentage != 55 {
		t.Fatalf("BindParams() percentage = %v, want 55", result.Percentage)
	}

	ptrResult, err := BindParams[*discountInput](params)
	if err != nil {
		t.Fatalf("BindParams() pointer unexpected error: %v", err)
	}
	if ptrResult == nil || ptrResult.Percentage != 55 {
		t.Fatalf("BindParams() pointer = %#v, want percentage 55", ptrResult)
	}
}

func TestBindParams_WithStructBinding(t *testing.T) {
	type actionInput struct {
		Name  string  `mapstructure:"name"`
		Count int64   `mapstructure:"count"`
		Note  *string `mapstructure:"note,omitempty"`
	}

	body := `{"name":"Widget","count":5}`
	req := httptest.NewRequest(http.MethodPost, "/Apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	defs, err := internalactions.ParameterDefinitionsFromStruct(reflect.TypeOf(actionInput{}))
	if err != nil {
		t.Fatalf("ParameterDefinitionsFromStruct() unexpected error: %v", err)
	}

	params, err := internalactions.ParseActionParameters(req, defs, reflect.TypeOf(actionInput{}))
	if err != nil {
		t.Fatalf("ParseActionParameters() unexpected error: %v", err)
	}

	bound, err := BindParams[actionInput](params)
	if err != nil {
		t.Fatalf("BindParams() unexpected error: %v", err)
	}

	if bound.Name != "Widget" {
		t.Fatalf("BindParams() name = %q, want %q", bound.Name, "Widget")
	}
	if bound.Count != 5 {
		t.Fatalf("BindParams() count = %d, want 5", bound.Count)
	}
	if bound.Note != nil {
		t.Fatalf("BindParams() expected nil note, got %#v", bound.Note)
	}

	ptrResult, err := BindParams[*actionInput](params)
	if err != nil {
		t.Fatalf("BindParams() pointer unexpected error: %v", err)
	}
	if ptrResult == nil || ptrResult.Count != 5 {
		t.Fatalf("BindParams() pointer = %#v, want count 5", ptrResult)
	}
}
