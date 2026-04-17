package metadata

import (
	"reflect"
	"testing"
)

type testTypeDef1 float64
type testTypeDef2 int32
type testTypeDef3 string

func TestRegisterTypeDefinition(t *testing.T) {
	tests := []struct {
		name    string
		goType  reflect.Type
		info    TypeDefinitionInfo
		wantErr bool
	}{
		{
			name:   "float64 type infers Edm.Double",
			goType: reflect.TypeOf(testTypeDef1(0)),
			info:   TypeDefinitionInfo{},
		},
		{
			name:   "int32 type infers Edm.Int32",
			goType: reflect.TypeOf(testTypeDef2(0)),
			info:   TypeDefinitionInfo{},
		},
		{
			name:   "string type infers Edm.String",
			goType: reflect.TypeOf(testTypeDef3("")),
			info:   TypeDefinitionInfo{Name: "MyLabel"},
		},
		{
			name:    "nil type returns error",
			goType:  nil,
			info:    TypeDefinitionInfo{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterTypeDefinition(tt.goType, tt.info)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify retrieval
			info, ok := GetTypeDefinition(tt.goType)
			if !ok {
				t.Fatal("Expected TypeDefinition to be registered")
			}
			if info.UnderlyingType == "" {
				t.Error("Expected UnderlyingType to be inferred")
			}
		})
	}
}

func TestGetTypeDefinition(t *testing.T) {
	type customFloat float64
	goType := reflect.TypeOf(customFloat(0))

	// Register
	err := RegisterTypeDefinition(goType, TypeDefinitionInfo{Name: "CustomFloat"})
	if err != nil {
		t.Fatalf("RegisterTypeDefinition error: %v", err)
	}

	info, ok := GetTypeDefinition(goType)
	if !ok {
		t.Fatal("Expected TypeDefinition to be found")
	}
	if info.Name != "CustomFloat" {
		t.Errorf("Expected Name=CustomFloat, got %s", info.Name)
	}
	if info.UnderlyingType != "Edm.Double" {
		t.Errorf("Expected UnderlyingType=Edm.Double, got %s", info.UnderlyingType)
	}

	// Non-registered type returns false
	type unregistered string
	_, ok = GetTypeDefinition(reflect.TypeOf(unregistered("")))
	if ok {
		t.Error("Expected unregistered type to return false")
	}

	// nil type returns false
	_, ok = GetTypeDefinition(nil)
	if ok {
		t.Error("Expected nil type to return false")
	}
}

func TestInferUnderlyingEdmType(t *testing.T) {
	tests := []struct {
		name     string
		kind     reflect.Kind
		typeName string
		want     string
	}{
		{"string", reflect.String, "", "Edm.String"},
		{"bool", reflect.Bool, "", "Edm.Boolean"},
		{"int8", reflect.Int8, "", "Edm.SByte"},
		{"int16", reflect.Int16, "", "Edm.Int16"},
		{"int32", reflect.Int32, "", "Edm.Int32"},
		{"int64", reflect.Int64, "", "Edm.Int64"},
		{"uint8", reflect.Uint8, "", "Edm.Byte"},
		{"uint16", reflect.Uint16, "", "Edm.Int32"},
		{"uint32", reflect.Uint32, "", "Edm.Int64"},
		{"uint64", reflect.Uint64, "", "Edm.Int64"},
		{"float32", reflect.Float32, "", "Edm.Single"},
		{"float64", reflect.Float64, "", "Edm.Double"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a type alias to get the right reflect.Type with the desired kind
			var goType reflect.Type
			switch tt.kind {
			case reflect.String:
				goType = reflect.TypeOf("")
			case reflect.Bool:
				goType = reflect.TypeOf(false)
			case reflect.Int8:
				goType = reflect.TypeOf(int8(0))
			case reflect.Int16:
				goType = reflect.TypeOf(int16(0))
			case reflect.Int32:
				goType = reflect.TypeOf(int32(0))
			case reflect.Int64:
				goType = reflect.TypeOf(int64(0))
			case reflect.Uint8:
				goType = reflect.TypeOf(uint8(0))
			case reflect.Uint16:
				goType = reflect.TypeOf(uint16(0))
			case reflect.Uint32:
				goType = reflect.TypeOf(uint32(0))
			case reflect.Uint64:
				goType = reflect.TypeOf(uint64(0))
			case reflect.Float32:
				goType = reflect.TypeOf(float32(0))
			case reflect.Float64:
				goType = reflect.TypeOf(float64(0))
			}

			got, err := inferUnderlyingEdmType(goType)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("inferUnderlyingEdmType(%s) = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestRegisterTypeDefinitionWithFacets(t *testing.T) {
	type typeWithFacets float64
	goType := reflect.TypeOf(typeWithFacets(0))

	err := RegisterTypeDefinition(goType, TypeDefinitionInfo{
		Name:           "Measurement",
		UnderlyingType: "Edm.Decimal",
		Precision:      12,
		Scale:          4,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	info, ok := GetTypeDefinition(goType)
	if !ok {
		t.Fatal("Expected TypeDefinition to be registered")
	}

	if info.Name != "Measurement" {
		t.Errorf("Expected Name=Measurement, got %s", info.Name)
	}
	if info.UnderlyingType != "Edm.Decimal" {
		t.Errorf("Expected UnderlyingType=Edm.Decimal, got %s", info.UnderlyingType)
	}
	if info.Precision != 12 {
		t.Errorf("Expected Precision=12, got %d", info.Precision)
	}
	if info.Scale != 4 {
		t.Errorf("Expected Scale=4, got %d", info.Scale)
	}
}
