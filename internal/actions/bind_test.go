package actions

import (
	"reflect"
	"strings"
	"testing"

	publicactions "github.com/nlstn/go-odata/actions"
)

type testParams struct {
	Percentage float64 `mapstructure:"percentage"`
	Note       *string `json:"note,omitempty"`
	Ignored    string  `mapstructure:"-"`
}

func TestParameterDefinitionsFromStruct(t *testing.T) {
	defs, err := ParameterDefinitionsFromStruct(reflect.TypeOf(testParams{}))
	if err != nil {
		t.Fatalf("ParameterDefinitionsFromStruct error: %v", err)
	}

	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}

	if defs[0].Name != "percentage" {
		t.Fatalf("first definition name = %q, want percentage", defs[0].Name)
	}
	if defs[0].Type != reflect.TypeOf(float64(0)) {
		t.Fatalf("first definition type = %v, want float64", defs[0].Type)
	}
	if !defs[0].Required {
		t.Fatalf("first definition should be required")
	}

	if defs[1].Name != "note" {
		t.Fatalf("second definition name = %q, want note", defs[1].Name)
	}
	if defs[1].Type != reflect.TypeOf((*string)(nil)) {
		t.Fatalf("second definition type = %v, want *string", defs[1].Type)
	}
	if defs[1].Required {
		t.Fatalf("second definition should be optional")
	}
}

func TestParameterDefinitionsFromStructPointer(t *testing.T) {
	defs, err := ParameterDefinitionsFromStruct(reflect.TypeOf(&testParams{}))
	if err != nil {
		t.Fatalf("ParameterDefinitionsFromStruct pointer error: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}
}

func TestParameterDefinitionsFromStruct_InvalidType(t *testing.T) {
	_, err := ParameterDefinitionsFromStruct(reflect.TypeOf(42))
	if err == nil {
		t.Fatal("expected error for non-struct type, got nil")
	}
	if !strings.Contains(err.Error(), "parameter binding target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBindStructToParams(t *testing.T) {
	t.Run("nil type", func(t *testing.T) {
		params := map[string]interface{}{"percentage": 10.0}
		if err := bindStructToParams(params, nil); err != nil {
			t.Fatalf("bindStructToParams nil type error: %v", err)
		}
		if _, ok := params[publicactions.BoundStructKey]; ok {
			t.Fatal("expected no bound struct for nil type")
		}
	})

	t.Run("missing required", func(t *testing.T) {
		params := map[string]interface{}{"note": "seasonal"}
		err := bindStructToParams(params, reflect.TypeOf(testParams{}))
		if err == nil {
			t.Fatal("expected error for missing required parameter, got nil")
		}
		if !strings.Contains(err.Error(), "required parameter 'percentage' is missing") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("binds struct", func(t *testing.T) {
		params := map[string]interface{}{"percentage": 12.5, "note": "seasonal"}
		if err := bindStructToParams(params, reflect.TypeOf(testParams{})); err != nil {
			t.Fatalf("bindStructToParams error: %v", err)
		}

		bound, ok := params[publicactions.BoundStructKey]
		if !ok {
			t.Fatal("expected bound struct in params")
		}
		boundParams, ok := bound.(*testParams)
		if !ok {
			t.Fatalf("expected bound type *testParams, got %T", bound)
		}
		if boundParams.Percentage != 12.5 {
			t.Fatalf("bound percentage = %v, want 12.5", boundParams.Percentage)
		}
		if boundParams.Note == nil || *boundParams.Note != "seasonal" {
			t.Fatalf("bound note = %#v, want seasonal", boundParams.Note)
		}
	})
}
