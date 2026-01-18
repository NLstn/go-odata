package edm

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestFromGoTypeTable(t *testing.T) {
	tests := []struct {
		name     string
		goType   reflect.Type
		expected string
		wantErr  string
	}{
		{
			name:     "nil type",
			goType:   nil,
			wantErr:  "nil type",
			expected: "",
		},
		{
			name:     "pointer to string",
			goType:   reflect.TypeOf((*string)(nil)),
			expected: "Edm.String",
		},
		{
			name:     "time.Time",
			goType:   reflect.TypeOf(time.Time{}),
			expected: "Edm.DateTimeOffset",
		},
		{
			name:     "decimal.Decimal",
			goType:   reflect.TypeOf(decimal.Decimal{}),
			expected: "Edm.Decimal",
		},
		{
			name:     "uuid.UUID",
			goType:   reflect.TypeOf(uuid.UUID{}),
			expected: "Edm.Guid",
		},
		{
			name:     "byte slice",
			goType:   reflect.TypeOf([]byte{}),
			expected: "Edm.Binary",
		},
		{
			name:     "byte array",
			goType:   reflect.TypeOf([16]byte{}),
			expected: "Edm.Binary",
		},
		{
			name:     "int maps to Edm.Int32",
			goType:   reflect.TypeOf(int(0)),
			expected: "Edm.Int32",
		},
		{
			name:     "int32 maps to Edm.Int32",
			goType:   reflect.TypeOf(int32(0)),
			expected: "Edm.Int32",
		},
		{
			name:     "int64 maps to Edm.Int64",
			goType:   reflect.TypeOf(int64(0)),
			expected: "Edm.Int64",
		},
		{
			name:     "int16 maps to Edm.Int16",
			goType:   reflect.TypeOf(int16(0)),
			expected: "Edm.Int16",
		},
		{
			name:     "int8 maps to Edm.SByte",
			goType:   reflect.TypeOf(int8(0)),
			expected: "Edm.SByte",
		},
		{
			name:     "uint maps to Edm.Int64",
			goType:   reflect.TypeOf(uint(0)),
			expected: "Edm.Int64",
		},
		{
			name:     "uint32 maps to Edm.Int64",
			goType:   reflect.TypeOf(uint32(0)),
			expected: "Edm.Int64",
		},
		{
			name:     "uint64 maps to Edm.Int64",
			goType:   reflect.TypeOf(uint64(0)),
			expected: "Edm.Int64",
		},
		{
			name:     "uint16 maps to Edm.Int32",
			goType:   reflect.TypeOf(uint16(0)),
			expected: "Edm.Int32",
		},
		{
			name:     "uint8 maps to Edm.Byte",
			goType:   reflect.TypeOf(uint8(0)),
			expected: "Edm.Byte",
		},
		{
			name:    "unsupported kind",
			goType:  reflect.TypeOf(map[string]string{}),
			wantErr: "unsupported Go type: map[string]string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := FromGoType(tt.goType)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}

func TestFromGoValueTable(t *testing.T) {
	if _, err := FromGoValue(nil); err == nil {
		t.Fatal("expected error for nil input")
	}

	value := "hello"
	parsed, err := FromGoValue(value)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.TypeName() != "Edm.String" {
		t.Fatalf("expected type Edm.String, got %s", parsed.TypeName())
	}
	if parsed.Value() != value {
		t.Fatalf("expected value %q, got %v", value, parsed.Value())
	}
}

func TestFromStructFieldTable(t *testing.T) {
	type sample struct {
		Amount decimal.Decimal `odata:"type=Edm.Decimal,precision=5,scale=2"`
		Name   string          `odata:"maxLength=10,nullable"`
		Bad    string          `odata:"precision=bad"`
	}

	field, ok := reflect.TypeOf(sample{}).FieldByName("Amount")
	if !ok {
		t.Fatal("expected Amount field")
	}
	amountValue := decimal.RequireFromString("12.34")
	expectedTypeName, expectedFacets, err := ParseTypeFromTag(field.Tag.Get("odata"))
	if err != nil {
		t.Fatalf("unexpected tag parse error: %v", err)
	}
	expectedType, err := ParseType(expectedTypeName, amountValue, expectedFacets)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	actualType, err := FromStructField(field, amountValue)
	if err != nil {
		t.Fatalf("unexpected FromStructField error: %v", err)
	}
	if actualType.TypeName() != expectedType.TypeName() {
		t.Fatalf("expected type %s, got %s", expectedType.TypeName(), actualType.TypeName())
	}
	if !reflect.DeepEqual(actualType.GetFacets(), expectedType.GetFacets()) {
		t.Fatalf("expected facets %#v, got %#v", expectedType.GetFacets(), actualType.GetFacets())
	}
	if actualType.String() != expectedType.String() {
		t.Fatalf("expected value %s, got %s", expectedType.String(), actualType.String())
	}

	nameField, ok := reflect.TypeOf(sample{}).FieldByName("Name")
	if !ok {
		t.Fatal("expected Name field")
	}
	nameValue := "alpha"
	nameTypeName, nameFacets, err := ParseTypeFromTag(nameField.Tag.Get("odata"))
	if err != nil {
		t.Fatalf("unexpected tag parse error: %v", err)
	}
	if nameTypeName != "" {
		t.Fatalf("expected empty type name from tag, got %q", nameTypeName)
	}
	expectedNameType, err := ParseType("Edm.String", nameValue, nameFacets)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	actualNameType, err := FromStructField(nameField, nameValue)
	if err != nil {
		t.Fatalf("unexpected FromStructField error: %v", err)
	}
	if actualNameType.TypeName() != expectedNameType.TypeName() {
		t.Fatalf("expected type %s, got %s", expectedNameType.TypeName(), actualNameType.TypeName())
	}
	if !reflect.DeepEqual(actualNameType.GetFacets(), expectedNameType.GetFacets()) {
		t.Fatalf("expected facets %#v, got %#v", expectedNameType.GetFacets(), actualNameType.GetFacets())
	}
	if actualNameType.String() != expectedNameType.String() {
		t.Fatalf("expected value %s, got %s", expectedNameType.String(), actualNameType.String())
	}

	badField, ok := reflect.TypeOf(sample{}).FieldByName("Bad")
	if !ok {
		t.Fatal("expected Bad field")
	}
	_, err = FromStructField(badField, "bad")
	if err == nil {
		t.Fatal("expected error for invalid tag")
	}
	expectedErr := "failed to parse odata tag: invalid precision value: bad"
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}
