package edm

import (
	"reflect"
	"testing"
)

func TestParseTypeFromTag(t *testing.T) {
	t.Run("Empty tag", func(t *testing.T) {
		typeName, facets, err := ParseTypeFromTag("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "" {
			t.Errorf("expected empty typeName, got '%s'", typeName)
		}
		if facets.Precision != nil {
			t.Error("expected nil precision")
		}
	})

	t.Run("Type only", func(t *testing.T) {
		typeName, facets, err := ParseTypeFromTag("type=Edm.Decimal")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Decimal" {
			t.Errorf("expected typeName 'Edm.Decimal', got '%s'", typeName)
		}
		if facets.Precision != nil {
			t.Error("expected nil precision")
		}
	})

	t.Run("Type with precision and scale", func(t *testing.T) {
		typeName, facets, err := ParseTypeFromTag("type=Edm.Decimal,precision=18,scale=4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Decimal" {
			t.Errorf("expected typeName 'Edm.Decimal', got '%s'", typeName)
		}
		if facets.Precision == nil || *facets.Precision != 18 {
			t.Errorf("expected precision 18, got %v", facets.Precision)
		}
		if facets.Scale == nil || *facets.Scale != 4 {
			t.Errorf("expected scale 4, got %v", facets.Scale)
		}
	})

	t.Run("Nullable flag", func(t *testing.T) {
		typeName, facets, err := ParseTypeFromTag("nullable,type=Edm.Date")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Date" {
			t.Errorf("expected typeName 'Edm.Date', got '%s'", typeName)
		}
		if !facets.Nullable {
			t.Error("expected Nullable to be true")
		}
	})

	t.Run("MaxLength facet", func(t *testing.T) {
		typeName, facets, err := ParseTypeFromTag("type=Edm.String,maxLength=50")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.String" {
			t.Errorf("expected typeName 'Edm.String', got '%s'", typeName)
		}
		if facets.MaxLength == nil || *facets.MaxLength != 50 {
			t.Errorf("expected maxLength 50, got %v", facets.MaxLength)
		}
	})

	t.Run("Unicode facet", func(t *testing.T) {
		typeName, facets, err := ParseTypeFromTag("type=Edm.String,unicode=true")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.String" {
			t.Errorf("expected typeName 'Edm.String', got '%s'", typeName)
		}
		if facets.Unicode == nil || *facets.Unicode != true {
			t.Errorf("expected unicode true, got %v", facets.Unicode)
		}
	})

	t.Run("SRID facet", func(t *testing.T) {
		_, facets, err := ParseTypeFromTag("type=Edm.Geography,srid=4326")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if facets.SRID == nil || *facets.SRID != 4326 {
			t.Errorf("expected srid 4326, got %v", facets.SRID)
		}
	})

	t.Run("Facets without type", func(t *testing.T) {
		typeName, facets, err := ParseTypeFromTag("precision=10,scale=2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "" {
			t.Errorf("expected empty typeName, got '%s'", typeName)
		}
		if facets.Precision == nil || *facets.Precision != 10 {
			t.Errorf("expected precision 10, got %v", facets.Precision)
		}
		if facets.Scale == nil || *facets.Scale != 2 {
			t.Errorf("expected scale 2, got %v", facets.Scale)
		}
	})

	t.Run("Unknown tags are ignored", func(t *testing.T) {
		typeName, _, err := ParseTypeFromTag("key,searchable,type=Edm.String")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.String" {
			t.Errorf("expected typeName 'Edm.String', got '%s'", typeName)
		}
		// key and searchable should be ignored
	})

	t.Run("Invalid precision value", func(t *testing.T) {
		_, _, err := ParseTypeFromTag("precision=invalid")
		if err == nil {
			t.Error("expected error for invalid precision")
		}
	})

	t.Run("Invalid scale value", func(t *testing.T) {
		_, _, err := ParseTypeFromTag("scale=abc")
		if err == nil {
			t.Error("expected error for invalid scale")
		}
	})

	t.Run("Real world example from user", func(t *testing.T) {
		// Revenue decimal.Decimal  `json:"Revenue" odata:"type=Edm.Decimal,precision=18,scale=4"`
		typeName, facets, err := ParseTypeFromTag("type=Edm.Decimal,precision=18,scale=4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Decimal" {
			t.Errorf("expected typeName 'Edm.Decimal', got '%s'", typeName)
		}
		if facets.Precision == nil || *facets.Precision != 18 {
			t.Errorf("expected precision 18, got %v", facets.Precision)
		}
		if facets.Scale == nil || *facets.Scale != 4 {
			t.Errorf("expected scale 4, got %v", facets.Scale)
		}
	})
}

func TestValidateDecimalFacets(t *testing.T) {
	t.Run("No facets", func(t *testing.T) {
		err := ValidateDecimalFacets("123.456", Facets{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Precision validation passes", func(t *testing.T) {
		precision := 6
		err := ValidateDecimalFacets("123.45", Facets{Precision: &precision})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Precision validation fails", func(t *testing.T) {
		precision := 5
		err := ValidateDecimalFacets("123.456", Facets{Precision: &precision})
		if err == nil {
			t.Error("expected error for exceeding precision")
		}
	})

	t.Run("Scale validation passes", func(t *testing.T) {
		scale := 3
		err := ValidateDecimalFacets("123.456", Facets{Scale: &scale})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Scale validation fails", func(t *testing.T) {
		scale := 2
		err := ValidateDecimalFacets("123.456", Facets{Scale: &scale})
		if err == nil {
			t.Error("expected error for exceeding scale")
		}
	})

	t.Run("Negative number", func(t *testing.T) {
		precision := 6
		err := ValidateDecimalFacets("-123.45", Facets{Precision: &precision})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Integer with no fractional part", func(t *testing.T) {
		precision := 5
		scale := 2
		err := ValidateDecimalFacets("12345", Facets{Precision: &precision, Scale: &scale})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestValidateLengthFacet(t *testing.T) {
	t.Run("No maxLength", func(t *testing.T) {
		err := ValidateLengthFacet(1000, Facets{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Within maxLength", func(t *testing.T) {
		maxLen := 10
		err := ValidateLengthFacet(5, Facets{MaxLength: &maxLen})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Exceeds maxLength", func(t *testing.T) {
		maxLen := 5
		err := ValidateLengthFacet(10, Facets{MaxLength: &maxLen})
		if err == nil {
			t.Error("expected error for exceeding maxLength")
		}
	})

	t.Run("Exactly at maxLength", func(t *testing.T) {
		maxLen := 10
		err := ValidateLengthFacet(10, Facets{MaxLength: &maxLen})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestTypeRegistry(t *testing.T) {
	t.Run("IsValidType for registered types", func(t *testing.T) {
		if !IsValidType("Edm.String") {
			t.Error("expected Edm.String to be valid")
		}
		if !IsValidType("Edm.Int32") {
			t.Error("expected Edm.Int32 to be valid")
		}
		if !IsValidType("Edm.Decimal") {
			t.Error("expected Edm.Decimal to be valid")
		}
		if !IsValidType("Edm.Boolean") {
			t.Error("expected Edm.Boolean to be valid")
		}
	})

	t.Run("IsValidType for unregistered type", func(t *testing.T) {
		if IsValidType("Edm.Unknown") {
			t.Error("expected Edm.Unknown to be invalid")
		}
	})

	t.Run("ParseType with valid type", func(t *testing.T) {
		typ, err := ParseType("Edm.Int32", 42, Facets{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typ.TypeName() != "Edm.Int32" {
			t.Errorf("expected Edm.Int32, got %s", typ.TypeName())
		}
		if typ.Value() != int32(42) {
			t.Errorf("expected value 42, got %v", typ.Value())
		}
	})

	t.Run("ParseType with unknown type", func(t *testing.T) {
		_, err := ParseType("Edm.Unknown", "value", Facets{})
		if err == nil {
			t.Error("expected error for unknown type")
		}
	})
}

func TestFromGoType(t *testing.T) {
	t.Run("string to Edm.String", func(t *testing.T) {
		typeName, err := FromGoType(reflect.TypeOf(""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.String" {
			t.Errorf("expected Edm.String, got %s", typeName)
		}
	})

	t.Run("int32 to Edm.Int32", func(t *testing.T) {
		typeName, err := FromGoType(reflect.TypeOf(int32(0)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Int32" {
			t.Errorf("expected Edm.Int32, got %s", typeName)
		}
	})

	t.Run("int64 to Edm.Int64", func(t *testing.T) {
		typeName, err := FromGoType(reflect.TypeOf(int64(0)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Int64" {
			t.Errorf("expected Edm.Int64, got %s", typeName)
		}
	})

	t.Run("int16 to Edm.Int16", func(t *testing.T) {
		typeName, err := FromGoType(reflect.TypeOf(int16(0)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Int16" {
			t.Errorf("expected Edm.Int16, got %s", typeName)
		}
	})

	t.Run("uint8 to Edm.Byte", func(t *testing.T) {
		typeName, err := FromGoType(reflect.TypeOf(uint8(0)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Byte" {
			t.Errorf("expected Edm.Byte, got %s", typeName)
		}
	})

	t.Run("int8 to Edm.SByte", func(t *testing.T) {
		typeName, err := FromGoType(reflect.TypeOf(int8(0)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.SByte" {
			t.Errorf("expected Edm.SByte, got %s", typeName)
		}
	})

	t.Run("float64 to Edm.Double", func(t *testing.T) {
		typeName, err := FromGoType(reflect.TypeOf(float64(0)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Double" {
			t.Errorf("expected Edm.Double, got %s", typeName)
		}
	})

	t.Run("float32 to Edm.Single", func(t *testing.T) {
		typeName, err := FromGoType(reflect.TypeOf(float32(0)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Single" {
			t.Errorf("expected Edm.Single, got %s", typeName)
		}
	})

	t.Run("bool to Edm.Boolean", func(t *testing.T) {
		typeName, err := FromGoType(reflect.TypeOf(true))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Boolean" {
			t.Errorf("expected Edm.Boolean, got %s", typeName)
		}
	})

	t.Run("[]byte to Edm.Binary", func(t *testing.T) {
		typeName, err := FromGoType(reflect.TypeOf([]byte{}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Binary" {
			t.Errorf("expected Edm.Binary, got %s", typeName)
		}
	})

	t.Run("pointer type", func(t *testing.T) {
		var i *int32
		typeName, err := FromGoType(reflect.TypeOf(i))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typeName != "Edm.Int32" {
			t.Errorf("expected Edm.Int32, got %s", typeName)
		}
	})

	t.Run("nil type error", func(t *testing.T) {
		_, err := FromGoType(nil)
		if err == nil {
			t.Error("expected error for nil type")
		}
	})
}

func TestFromGoValue(t *testing.T) {
	t.Run("int32 value", func(t *testing.T) {
		typ, err := FromGoValue(int32(42))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typ.TypeName() != "Edm.Int32" {
			t.Errorf("expected Edm.Int32, got %s", typ.TypeName())
		}
		if typ.Value() != int32(42) {
			t.Errorf("expected value 42, got %v", typ.Value())
		}
	})

	t.Run("string value", func(t *testing.T) {
		typ, err := FromGoValue("hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typ.TypeName() != "Edm.String" {
			t.Errorf("expected Edm.String, got %s", typ.TypeName())
		}
		if typ.Value() != "hello" {
			t.Errorf("expected value 'hello', got %v", typ.Value())
		}
	})

	t.Run("nil value error", func(t *testing.T) {
		_, err := FromGoValue(nil)
		if err == nil {
			t.Error("expected error for nil value")
		}
	})
}

func TestFromStructField(t *testing.T) {
	type TestStruct struct {
		Name          string  `odata:"type=Edm.String,maxLength=50"`
		Age           int32   `odata:"type=Edm.Int32"`
		Price         string  `odata:"type=Edm.Decimal,precision=18,scale=4"`
		Active        bool    // No tag, should infer
		NullableValue *string `odata:"nullable"`
	}

	structType := reflect.TypeOf(TestStruct{})

	t.Run("Field with explicit type and facets", func(t *testing.T) {
		field, _ := structType.FieldByName("Name")
		typ, err := FromStructField(field, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typ.TypeName() != "Edm.String" {
			t.Errorf("expected Edm.String, got %s", typ.TypeName())
		}
		facets := typ.GetFacets()
		if facets.MaxLength == nil || *facets.MaxLength != 50 {
			t.Errorf("expected maxLength 50, got %v", facets.MaxLength)
		}
	})

	t.Run("Field with inferred type", func(t *testing.T) {
		field, _ := structType.FieldByName("Active")
		typ, err := FromStructField(field, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typ.TypeName() != "Edm.Boolean" {
			t.Errorf("expected Edm.Boolean, got %s", typ.TypeName())
		}
	})

	t.Run("Field with nullable flag", func(t *testing.T) {
		field, _ := structType.FieldByName("NullableValue")
		typ, err := FromStructField(field, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// For nil pointer values, the type should handle null appropriately
		if typ == nil {
			t.Error("expected non-nil type for nullable field")
		} else if !typ.IsNull() {
			t.Error("expected IsNull() to be true for nil value")
		}
	})
}
