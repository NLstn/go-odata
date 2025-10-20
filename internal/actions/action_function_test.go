package actions

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

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
			wantParams: map[string]interface{}{"count": float64(5)},
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
				"count":  float64(5),
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

			params, err := ParseActionParameters(req, tt.paramDefs)

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
			wantErr: "parameter 'price' must be a number",
		},
		{
			name: "string instead of int",
			body: `{"count": "invalid"}`,
			paramDefs: []ParameterDefinition{
				{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
			},
			wantErr: "parameter 'count' must be an integer",
		},
		{
			name: "number instead of string",
			body: `{"name": 123}`,
			paramDefs: []ParameterDefinition{
				{Name: "name", Type: reflect.TypeOf(""), Required: true},
			},
			wantErr: "parameter 'name' must be a string",
		},
		{
			name: "string instead of bool",
			body: `{"active": "true"}`,
			paramDefs: []ParameterDefinition{
				{Name: "active", Type: reflect.TypeOf(true), Required: true},
			},
			wantErr: "parameter 'active' must be a boolean",
		},
		{
			name: "float instead of int (non-whole number)",
			body: `{"count": 5.5}`,
			paramDefs: []ParameterDefinition{
				{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
			},
			wantErr: "parameter 'count' must be an integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			_, err := ParseActionParameters(req, tt.paramDefs)

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

	_, err := ParseActionParameters(req, paramDefs)

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

	_, err := ParseActionParameters(req, paramDefs)

	if err == nil {
		t.Error("ParseActionParameters() expected error for invalid JSON")
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

			params, err := ParseFunctionParameters(req, tt.paramDefs)

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
			wantErr: "parameter 'count' must be an integer",
		},
		{
			name: "invalid float parameter",
			url:  "/test?rate=invalid",
			paramDefs: []ParameterDefinition{
				{Name: "rate", Type: reflect.TypeOf(float64(0)), Required: true},
			},
			wantErr: "parameter 'rate' must be a number",
		},
		{
			name: "invalid bool parameter",
			url:  "/test?active=invalid",
			paramDefs: []ParameterDefinition{
				{Name: "active", Type: reflect.TypeOf(true), Required: true},
			},
			wantErr: "parameter 'active' must be a boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			_, err := ParseFunctionParameters(req, tt.paramDefs)

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

func TestParseFunctionParameters_MissingRequired(t *testing.T) {
	paramDefs := []ParameterDefinition{
		{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	_, err := ParseFunctionParameters(req, paramDefs)

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
