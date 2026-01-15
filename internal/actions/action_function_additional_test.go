package actions

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestActionAndFunctionSignaturesMatch(t *testing.T) {
	stringType := reflect.TypeOf("")

	actionA := &ActionDefinition{
		Name:      "DoThing",
		IsBound:   true,
		EntitySet: "Products",
		Parameters: []ParameterDefinition{
			{Name: "name", Type: stringType, Required: true},
			{Name: "note", Type: stringType, Required: false},
		},
	}
	actionB := &ActionDefinition{
		Name:      "DoThing",
		IsBound:   true,
		EntitySet: "Products",
		Parameters: []ParameterDefinition{
			{Name: "note", Type: stringType, Required: false},
			{Name: "name", Type: stringType, Required: true},
		},
	}
	if !ActionSignaturesMatch(actionA, actionB) {
		t.Fatalf("expected action signatures to match")
	}

	actionB.EntitySet = "Orders"
	if ActionSignaturesMatch(actionA, actionB) {
		t.Fatalf("expected action signatures to differ by entity set")
	}

	functionA := &FunctionDefinition{
		Name:      "GetStatus",
		IsBound:   false,
		EntitySet: "",
		Parameters: []ParameterDefinition{
			{Name: "id", Type: reflect.TypeOf(0), Required: true},
		},
	}
	functionB := &FunctionDefinition{
		Name:      "GetStatus",
		IsBound:   false,
		EntitySet: "",
		Parameters: []ParameterDefinition{
			{Name: "id", Type: reflect.TypeOf(0), Required: true},
		},
	}
	if !FunctionSignaturesMatch(functionA, functionB) {
		t.Fatalf("expected function signatures to match")
	}

	functionB.Name = "GetOther"
	if FunctionSignaturesMatch(functionA, functionB) {
		t.Fatalf("expected function signatures to differ by name")
	}
}

func TestFunctionParameterNamesMatch(t *testing.T) {
	paramDefs := []ParameterDefinition{
		{Name: "id", Type: reflect.TypeOf(0), Required: true},
		{Name: "filter", Type: reflect.TypeOf(""), Required: false},
	}

	if !functionParameterNamesMatch(paramDefs, map[string]string{"id": "1"}) {
		t.Fatalf("expected required parameter to match")
	}

	if functionParameterNamesMatch(paramDefs, map[string]string{"filter": "name"}) {
		t.Fatalf("expected missing required parameter to fail")
	}

	if functionParameterNamesMatch(paramDefs, map[string]string{"id": "1", "extra": "nope"}) {
		t.Fatalf("expected extra parameter to fail")
	}
}

func TestResolveFunctionOverload_ByParameters(t *testing.T) {
	idFunction := &FunctionDefinition{
		Name: "GetItem",
		Parameters: []ParameterDefinition{
			{Name: "id", Type: reflect.TypeOf(0), Required: true},
		},
	}
	nameFunction := &FunctionDefinition{
		Name: "GetItem",
		Parameters: []ParameterDefinition{
			{Name: "name", Type: reflect.TypeOf(""), Required: true},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/GetItem(id=1)", nil)
	selected, params, err := ResolveFunctionOverload(req, []*FunctionDefinition{idFunction, nameFunction}, false, "")
	if err != nil {
		t.Fatalf("ResolveFunctionOverload() unexpected error: %v", err)
	}
	if selected != idFunction {
		t.Fatalf("ResolveFunctionOverload() selected = %#v, want idFunction", selected)
	}
	if got := params["id"]; got != 1 {
		t.Fatalf("ResolveFunctionOverload() id param = %v, want 1", got)
	}
}
