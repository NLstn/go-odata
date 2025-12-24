package actions

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

type sampleAddress struct {
	Street   string            `json:"street"`
	Tags     []string          `json:"tags"`
	Metadata map[string]string `json:"metadata"`
}

type sampleOrder struct {
	Address sampleAddress `json:"address"`
	Counts  []int         `json:"counts"`
}

func TestParseActionParameters_ValidTypes(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		paramDefs  []ParameterDefinition
		wantParams map[string]interface{}
		wantErr    bool
	}{
		{
			name: "valid float parameter",
			body: `{"price": 10.5}`,
			paramDefs: []ParameterDefinition{
				{Name: "price", Type: reflect.TypeOf(float64(0)), Required: true},
			},
			wantParams: map[string]interface{}{"price": 10.5},
			wantErr:    false,
		},
		{
			name: "valid int parameter",
			body: `{"count": 5}`,
			paramDefs: []ParameterDefinition{
				{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
			},
			wantParams: map[string]interface{}{"count": int64(5)},
			wantErr:    false,
		},
		{
			name: "valid string parameter",
			body: `{"name": "test"}`,
			paramDefs: []ParameterDefinition{
				{Name: "name", Type: reflect.TypeOf(""), Required: true},
			},
			wantParams: map[string]interface{}{"name": "test"},
			wantErr:    false,
		},
		{
			name: "valid bool parameter",
			body: `{"active": true}`,
			paramDefs: []ParameterDefinition{
				{Name: "active", Type: reflect.TypeOf(true), Required: true},
			},
			wantParams: map[string]interface{}{"active": true},
			wantErr:    false,
		},
		{
			name: "multiple valid parameters",
			body: `{"name": "test", "count": 5, "price": 10.5, "active": true}`,
			paramDefs: []ParameterDefinition{
				{Name: "name", Type: reflect.TypeOf(""), Required: true},
				{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
				{Name: "price", Type: reflect.TypeOf(float64(0)), Required: true},
				{Name: "active", Type: reflect.TypeOf(true), Required: true},
			},
			wantParams: map[string]interface{}{
				"name":   "test",
				"count":  int64(5),
				"price":  10.5,
				"active": true,
			},
			wantErr: false,
		},
		{
			name: "optional parameter not provided",
			body: `{"name": "test"}`,
			paramDefs: []ParameterDefinition{
				{Name: "name", Type: reflect.TypeOf(""), Required: true},
				{Name: "count", Type: reflect.TypeOf(int64(0)), Required: false},
			},
			wantParams: map[string]interface{}{"name": "test"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			params, err := ParseActionParameters(req, tt.paramDefs, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseActionParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(params) != len(tt.wantParams) {
					t.Errorf("ParseActionParameters() got %d params, want %d", len(params), len(tt.wantParams))
				}

				for key, want := range tt.wantParams {
					got, ok := params[key]
					if !ok {
						t.Errorf("ParseActionParameters() missing param %s", key)
						continue
					}
					if !reflect.DeepEqual(got, want) {
						t.Errorf("ParseActionParameters() param %s = %v, want %v", key, got, want)
					}
				}
			}
		})
	}
}

func TestParseActionParameters_InvalidTypes(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		paramDefs []ParameterDefinition
		wantErr   string
	}{
		{
			name: "string instead of float",
			body: `{"price": "invalid"}`,
			paramDefs: []ParameterDefinition{
				{Name: "price", Type: reflect.TypeOf(float64(0)), Required: true},
			},
			wantErr: "parameter 'price' has invalid value: json: cannot unmarshal string into Go value of type float64",
		},
		{
			name: "string instead of int",
			body: `{"count": "invalid"}`,
			paramDefs: []ParameterDefinition{
				{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
			},
			wantErr: "parameter 'count' has invalid value: json: cannot unmarshal string into Go value of type int64",
		},
		{
			name: "number instead of string",
			body: `{"name": 123}`,
			paramDefs: []ParameterDefinition{
				{Name: "name", Type: reflect.TypeOf(""), Required: true},
			},
			wantErr: "parameter 'name' has invalid value: json: cannot unmarshal number into Go value of type string",
		},
		{
			name: "string instead of bool",
			body: `{"active": "true"}`,
			paramDefs: []ParameterDefinition{
				{Name: "active", Type: reflect.TypeOf(true), Required: true},
			},
			wantErr: "parameter 'active' has invalid value: json: cannot unmarshal string into Go value of type bool",
		},
		{
			name: "float instead of int (non-whole number)",
			body: `{"count": 5.5}`,
			paramDefs: []ParameterDefinition{
				{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
			},
			wantErr: "parameter 'count' has invalid value: json: cannot unmarshal number 5.5 into Go value of type int64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			_, err := ParseActionParameters(req, tt.paramDefs, nil)

			if err == nil {
				t.Error("ParseActionParameters() expected error but got nil")
				return
			}

			if err.Error() != tt.wantErr {
				t.Errorf("ParseActionParameters() error = %v, wantErr %v", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseActionParameters_MissingRequired(t *testing.T) {
	body := `{"name": "test"}`
	paramDefs := []ParameterDefinition{
		{Name: "name", Type: reflect.TypeOf(""), Required: true},
		{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
	}

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := ParseActionParameters(req, paramDefs, nil)

	if err == nil {
		t.Error("ParseActionParameters() expected error for missing required parameter")
		return
	}

	expectedErr := "required parameter 'count' is missing"
	if err.Error() != expectedErr {
		t.Errorf("ParseActionParameters() error = %v, want %v", err.Error(), expectedErr)
	}
}

func TestParseActionParameters_InvalidJSON(t *testing.T) {
	body := `{invalid json}`
	paramDefs := []ParameterDefinition{
		{Name: "name", Type: reflect.TypeOf(""), Required: true},
	}

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := ParseActionParameters(req, paramDefs, nil)

	if err == nil {
		t.Error("ParseActionParameters() expected error for invalid JSON")
	}
}

func TestParseActionParameters_ComplexTypes(t *testing.T) {
	body := `{
                "order": {
                        "address": {
                                "street": "Main St",
                                "tags": ["primary", "billing"],
                                "metadata": {"zone": "north"}
                        },
                        "counts": [1, 2, 3]
                },
                "labels": ["priority", "express"],
                "shipping": {
                        "street": "Second St",
                        "tags": ["shipping"],
                        "metadata": {"zone": "south"}
                }
        }`

	paramDefs := []ParameterDefinition{
		{Name: "order", Type: reflect.TypeOf(sampleOrder{}), Required: true},
		{Name: "labels", Type: reflect.TypeOf([]string{}), Required: true},
		{Name: "shipping", Type: reflect.TypeOf(&sampleAddress{}), Required: false},
	}

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	params, err := ParseActionParameters(req, paramDefs, nil)
	if err != nil {
		t.Fatalf("ParseActionParameters() unexpected error: %v", err)
	}

	expectedOrder := sampleOrder{
		Address: sampleAddress{
			Street: "Main St",
			Tags:   []string{"primary", "billing"},
			Metadata: map[string]string{
				"zone": "north",
			},
		},
		Counts: []int{1, 2, 3},
	}

	if got, ok := params["order"].(sampleOrder); !ok || !reflect.DeepEqual(got, expectedOrder) {
		t.Fatalf("ParseActionParameters() order = %#v, want %#v", params["order"], expectedOrder)
	}

	expectedLabels := []string{"priority", "express"}
	if got, ok := params["labels"].([]string); !ok || !reflect.DeepEqual(got, expectedLabels) {
		t.Fatalf("ParseActionParameters() labels = %#v, want %#v", params["labels"], expectedLabels)
	}

	if shippingVal, ok := params["shipping"]; ok {
		shippingPtr, ok := shippingVal.(*sampleAddress)
		if !ok {
			t.Fatalf("ParseActionParameters() shipping type = %T, want *sampleAddress", shippingVal)
		}
		expectedShipping := &sampleAddress{
			Street: "Second St",
			Tags:   []string{"shipping"},
			Metadata: map[string]string{
				"zone": "south",
			},
		}
		if !reflect.DeepEqual(shippingPtr, expectedShipping) {
			t.Fatalf("ParseActionParameters() shipping = %#v, want %#v", shippingPtr, expectedShipping)
		}
	} else {
		t.Fatal("ParseActionParameters() expected shipping parameter to be present")
	}
}

func TestParseFunctionParameters_ValidTypes(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		paramDefs  []ParameterDefinition
		wantParams map[string]interface{}
		wantErr    bool
	}{
		{
			name: "valid int parameter",
			url:  "/test?count=5",
			paramDefs: []ParameterDefinition{
				{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
			},
			wantParams: map[string]interface{}{"count": int64(5)},
			wantErr:    false,
		},
		{
			name: "valid float parameter",
			url:  "/test?rate=0.08",
			paramDefs: []ParameterDefinition{
				{Name: "rate", Type: reflect.TypeOf(float64(0)), Required: true},
			},
			wantParams: map[string]interface{}{"rate": 0.08},
			wantErr:    false,
		},
		{
			name: "valid string parameter",
			url:  "/test?name=test",
			paramDefs: []ParameterDefinition{
				{Name: "name", Type: reflect.TypeOf(""), Required: true},
			},
			wantParams: map[string]interface{}{"name": "test"},
			wantErr:    false,
		},
		{
			name: "valid bool parameter",
			url:  "/test?active=true",
			paramDefs: []ParameterDefinition{
				{Name: "active", Type: reflect.TypeOf(true), Required: true},
			},
			wantParams: map[string]interface{}{"active": true},
			wantErr:    false,
		},
		{
			name: "multiple parameters",
			url:  "/test?name=test&count=5&rate=0.08",
			paramDefs: []ParameterDefinition{
				{Name: "name", Type: reflect.TypeOf(""), Required: true},
				{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
				{Name: "rate", Type: reflect.TypeOf(float64(0)), Required: true},
			},
			wantParams: map[string]interface{}{
				"name":  "test",
				"count": int64(5),
				"rate":  0.08,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			params, err := ParseFunctionParameters(req, tt.paramDefs, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFunctionParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(params) != len(tt.wantParams) {
					t.Errorf("ParseFunctionParameters() got %d params, want %d", len(params), len(tt.wantParams))
				}

				for key, want := range tt.wantParams {
					got, ok := params[key]
					if !ok {
						t.Errorf("ParseFunctionParameters() missing param %s", key)
						continue
					}
					if !reflect.DeepEqual(got, want) {
						t.Errorf("ParseFunctionParameters() param %s = %v (type %T), want %v (type %T)", key, got, got, want, want)
					}
				}
			}
		})
	}
}

func TestParseFunctionParameters_InvalidTypes(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		paramDefs []ParameterDefinition
		wantErr   string
	}{
		{
			name: "invalid int parameter",
			url:  "/test?count=invalid",
			paramDefs: []ParameterDefinition{
				{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
			},
			wantErr: "parameter 'count' has invalid value: expected integer value",
		},
		{
			name: "invalid float parameter",
			url:  "/test?rate=invalid",
			paramDefs: []ParameterDefinition{
				{Name: "rate", Type: reflect.TypeOf(float64(0)), Required: true},
			},
			wantErr: "parameter 'rate' has invalid value: expected numeric value",
		},
		{
			name: "invalid bool parameter",
			url:  "/test?active=invalid",
			paramDefs: []ParameterDefinition{
				{Name: "active", Type: reflect.TypeOf(true), Required: true},
			},
			wantErr: "parameter 'active' has invalid value: expected boolean value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			_, err := ParseFunctionParameters(req, tt.paramDefs, nil)

			if err == nil {
				t.Error("ParseFunctionParameters() expected error but got nil")
				return
			}

			if err.Error() != tt.wantErr {
				t.Errorf("ParseFunctionParameters() error = %v, wantErr %v", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseFunctionParameters_ComplexTypes(t *testing.T) {
	t.Run("array query parameter", func(t *testing.T) {
		addressesLiteral := `[{"street":"First","tags":["primary"],"metadata":{"zone":"north"}},{"street":"Second","tags":[],"metadata":{"zone":"south"}}]`
		req := httptest.NewRequest(http.MethodGet, "/test?addresses="+url.QueryEscape(addressesLiteral), nil)

		paramDefs := []ParameterDefinition{
			{Name: "addresses", Type: reflect.TypeOf([]sampleAddress{}), Required: true},
		}

		params, err := ParseFunctionParameters(req, paramDefs, nil)
		if err != nil {
			t.Fatalf("ParseFunctionParameters() unexpected error: %v", err)
		}

		want := []sampleAddress{
			{
				Street:   "First",
				Tags:     []string{"primary"},
				Metadata: map[string]string{"zone": "north"},
			},
			{
				Street:   "Second",
				Tags:     []string{},
				Metadata: map[string]string{"zone": "south"},
			},
		}

		if got, ok := params["addresses"].([]sampleAddress); !ok || !reflect.DeepEqual(got, want) {
			t.Fatalf("ParseFunctionParameters() addresses = %#v, want %#v", params["addresses"], want)
		}
	})

	t.Run("object path parameter", func(t *testing.T) {
		filterLiteral := `{"address":{"street":"Warehouse","tags":["dock"],"metadata":{"zone":"west"}},"counts":[5,6]}`
		escaped := url.PathEscape(filterLiteral)
		req := httptest.NewRequest(http.MethodGet, "/DoSomething(filter="+escaped+")", nil)

		paramDefs := []ParameterDefinition{
			{Name: "filter", Type: reflect.TypeOf(sampleOrder{}), Required: true},
		}

		params, err := ParseFunctionParameters(req, paramDefs, nil)
		if err != nil {
			t.Fatalf("ParseFunctionParameters() unexpected error: %v", err)
		}

		want := sampleOrder{
			Address: sampleAddress{
				Street:   "Warehouse",
				Tags:     []string{"dock"},
				Metadata: map[string]string{"zone": "west"},
			},
			Counts: []int{5, 6},
		}

		if got, ok := params["filter"].(sampleOrder); !ok || !reflect.DeepEqual(got, want) {
			t.Fatalf("ParseFunctionParameters() filter = %#v, want %#v", params["filter"], want)
		}
	})
}

func TestParseFunctionParameters_MissingRequired(t *testing.T) {
	paramDefs := []ParameterDefinition{
		{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	_, err := ParseFunctionParameters(req, paramDefs, nil)

	if err == nil {
		t.Error("ParseFunctionParameters() expected error for missing required parameter")
		return
	}

	expectedErr := "required parameter 'count' is missing"
	if err.Error() != expectedErr {
		t.Errorf("ParseFunctionParameters() error = %v, want %v", err.Error(), expectedErr)
	}
}

func TestValidateParameterType_NullValues(t *testing.T) {
	// Null values should be allowed
	err := validateParameterType("test", nil, reflect.TypeOf(""))
	if err != nil {
		t.Errorf("validateParameterType() with nil value should not error, got %v", err)
	}
}

func TestValidateParameterType_IntegerEdgeCases(t *testing.T) {
	// Whole number as float should be accepted for integer type
	err := validateParameterType("count", float64(5), reflect.TypeOf(int64(0)))
	if err != nil {
		t.Errorf("validateParameterType() should accept whole number float for int, got %v", err)
	}

	// Non-whole number should be rejected for integer type
	err = validateParameterType("count", 5.5, reflect.TypeOf(int64(0)))
	if err == nil {
		t.Error("validateParameterType() should reject non-whole number for int")
	}
}

func TestValidateParameterType_NumericFlexibility(t *testing.T) {
	// Float type should accept both int and float
	err := validateParameterType("rate", float64(5), reflect.TypeOf(float64(0)))
	if err != nil {
		t.Errorf("validateParameterType() should accept int for float, got %v", err)
	}

	err = validateParameterType("rate", 5.5, reflect.TypeOf(float64(0)))
	if err != nil {
		t.Errorf("validateParameterType() should accept float for float, got %v", err)
	}
}

func TestResolveActionOverload_TypedParameters(t *testing.T) {
	countAction := &ActionDefinition{
		Name:    "Process",
		IsBound: false,
		Parameters: []ParameterDefinition{
			{Name: "count", Type: reflect.TypeOf(int(0)), Required: true},
		},
	}

	orderAction := &ActionDefinition{
		Name:    "Process",
		IsBound: false,
		Parameters: []ParameterDefinition{
			{Name: "order", Type: reflect.TypeOf(sampleOrder{}), Required: true},
		},
	}

	candidates := []*ActionDefinition{countAction, orderAction}

	t.Run("selects integer overload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/Process", bytes.NewBufferString(`{"count": 7}`))
		req.Header.Set("Content-Type", "application/json")

		selected, params, err := ResolveActionOverload(req, candidates, false, "")
		if err != nil {
			t.Fatalf("ResolveActionOverload() unexpected error: %v", err)
		}

		if selected != countAction {
			t.Fatalf("ResolveActionOverload() selected = %#v, want countAction", selected)
		}

		value, ok := params["count"].(int)
		if !ok {
			t.Fatalf("ResolveActionOverload() count type = %T, want int", params["count"])
		}

		if value != 7 {
			t.Fatalf("ResolveActionOverload() count = %d, want 7", value)
		}
	})

	t.Run("selects struct overload", func(t *testing.T) {
		body := `{"order": {"address": {"street": "Main"}, "counts": [1,2]}}`
		req := httptest.NewRequest(http.MethodPost, "/Process", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		selected, params, err := ResolveActionOverload(req, candidates, false, "")
		if err != nil {
			t.Fatalf("ResolveActionOverload() unexpected error: %v", err)
		}

		if selected != orderAction {
			t.Fatalf("ResolveActionOverload() selected = %#v, want orderAction", selected)
		}

		orderParam, ok := params["order"].(sampleOrder)
		if !ok {
			t.Fatalf("ResolveActionOverload() order type = %T, want sampleOrder", params["order"])
		}

		expected := sampleOrder{
			Address: sampleAddress{Street: "Main"},
			Counts:  []int{1, 2},
		}

		if !reflect.DeepEqual(orderParam, expected) {
			t.Fatalf("ResolveActionOverload() order = %#v, want %#v", orderParam, expected)
		}
	})
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
	} else if ptrResult.Percentage != 12.5 {
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

func TestParseActionParameters_WithStructBinding(t *testing.T) {
	type actionInput struct {
		Name  string  `mapstructure:"name"`
		Count int64   `mapstructure:"count"`
		Note  *string `mapstructure:"note,omitempty"`
	}

	body := `{"name":"Widget","count":5}`
	req := httptest.NewRequest(http.MethodPost, "/Apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	defs, err := ParameterDefinitionsFromStruct(reflect.TypeOf(actionInput{}))
	if err != nil {
		t.Fatalf("ParameterDefinitionsFromStruct() unexpected error: %v", err)
	}

	params, err := ParseActionParameters(req, defs, reflect.TypeOf(actionInput{}))
	if err != nil {
		t.Fatalf("ParseActionParameters() unexpected error: %v", err)
	}

	// Verify the params were populated correctly (struct binding happens during parse)
	if _, ok := params["name"]; !ok {
		t.Fatal("ParseActionParameters() missing 'name' parameter")
	}
	if _, ok := params["count"]; !ok {
		t.Fatal("ParseActionParameters() missing 'count' parameter")
	}
}
