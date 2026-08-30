package metadata

import (
	"reflect"
	"testing"
)

func TestParsePrimitiveType(t *testing.T) {
	t.Parallel()

	primitiveType, err := ParsePrimitiveType("Edm.Date")
	if err != nil {
		t.Fatalf("ParsePrimitiveType() error = %v", err)
	}
	if primitiveType != PrimitiveTypeDate {
		t.Fatalf("ParsePrimitiveType() = %q, want %q", primitiveType, PrimitiveTypeDate)
	}

	if _, err := ParsePrimitiveType("Edm.NotAType"); err == nil {
		t.Fatal("ParsePrimitiveType() expected an error for an unsupported type")
	}
}

func TestPrimitiveTypeFromGoType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		goType reflect.Type
		want   PrimitiveType
	}{
		{name: "string", goType: reflect.TypeOf(""), want: PrimitiveTypeString},
		{name: "int8", goType: reflect.TypeOf(int8(0)), want: PrimitiveTypeSByte},
		{name: "uint8", goType: reflect.TypeOf(uint8(0)), want: PrimitiveTypeByte},
		{name: "uint32", goType: reflect.TypeOf(uint32(0)), want: PrimitiveTypeInt64},
		{name: "binary", goType: reflect.TypeOf([]byte(nil)), want: PrimitiveTypeBinary},
		{name: "pointer", goType: reflect.TypeOf((*string)(nil)), want: PrimitiveTypeString},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := PrimitiveTypeFromGoType(test.goType)
			if err != nil {
				t.Fatalf("PrimitiveTypeFromGoType() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("PrimitiveTypeFromGoType() = %q, want %q", got, test.want)
			}
		})
	}
}
