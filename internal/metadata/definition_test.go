package metadata

import (
	"reflect"
	"strings"
	"testing"
)

func TestAnalyzeEntityDefinition(t *testing.T) {
	t.Parallel()

	nullable := true
	definition := EntityDefinition{
		Name:          "Produkt",
		EntitySetName: "Produkte",
		Properties: []PropertyDefinition{
			{Name: "Locale", Type: PrimitiveTypeString},
			{Name: "ID", Type: PrimitiveTypeInt64},
			{Name: "Description", Type: PrimitiveTypeString, Nullable: &nullable},
		},
		Keys:            []string{"ID", "Locale"},
		DisabledMethods: []string{" post ", "delete"},
	}

	entityMetadata, err := AnalyzeEntityDefinition(definition)
	if err != nil {
		t.Fatalf("AnalyzeEntityDefinition() error = %v", err)
	}
	if entityMetadata.EntityType != nil {
		t.Fatalf("EntityType = %v, want nil", entityMetadata.EntityType)
	}
	if !entityMetadata.IsVirtual {
		t.Fatal("IsVirtual = false, want true")
	}
	if entityMetadata.KeyProperty != nil {
		t.Fatalf("KeyProperty = %v, want nil for a composite key", entityMetadata.KeyProperty)
	}
	if got := []string{entityMetadata.KeyProperties[0].Name, entityMetadata.KeyProperties[1].Name}; !reflect.DeepEqual(got, definition.Keys) {
		t.Fatalf("key order = %v, want %v", got, definition.Keys)
	}
	if entityMetadata.Properties[0].Nullable == nil || *entityMetadata.Properties[0].Nullable {
		t.Fatal("key property Nullable must resolve to false")
	}
	if entityMetadata.Properties[2].Nullable == nil || !*entityMetadata.Properties[2].Nullable {
		t.Fatal("ordinary property Nullable must resolve to true")
	}
	if !entityMetadata.DisabledMethods["POST"] || !entityMetadata.DisabledMethods["DELETE"] {
		t.Fatalf("DisabledMethods = %v", entityMetadata.DisabledMethods)
	}
	if definition.DisabledMethods[0] != " post " || !nullable {
		t.Fatal("AnalyzeEntityDefinition mutated its input")
	}
}

func TestAnalyzeEntityDefinitionSetsSingleKeyProperty(t *testing.T) {
	t.Parallel()

	entityMetadata, err := AnalyzeEntityDefinition(EntityDefinition{
		Name:          "Product",
		EntitySetName: "Products",
		Properties:    []PropertyDefinition{{Name: "ID", Type: PrimitiveTypeInt64}},
		Keys:          []string{"ID"},
	})
	if err != nil {
		t.Fatalf("AnalyzeEntityDefinition() error = %v", err)
	}
	if entityMetadata.KeyProperty != &entityMetadata.KeyProperties[0] {
		t.Fatal("KeyProperty does not reference the single ordered key")
	}
}

func TestEntityDefinitionValidation(t *testing.T) {
	t.Parallel()

	trueValue := true
	valid := EntityDefinition{
		Name:          "Product",
		EntitySetName: "Products",
		Properties: []PropertyDefinition{
			{Name: "ID", Type: PrimitiveTypeInt64},
			{Name: "Name", Type: PrimitiveTypeString},
		},
		Keys: []string{"ID"},
	}

	tests := []struct {
		name      string
		mutate    func(*EntityDefinition)
		wantError string
	}{
		{name: "empty entity name", mutate: func(value *EntityDefinition) { value.Name = "" }, wantError: "entity name cannot be empty"},
		{name: "invalid entity name", mutate: func(value *EntityDefinition) { value.Name = "1Product" }, wantError: "not a valid OData identifier"},
		{name: "long entity set name", mutate: func(value *EntityDefinition) { value.EntitySetName = "P" + strings.Repeat("x", 128) }, wantError: "not a valid OData identifier"},
		{name: "no properties", mutate: func(value *EntityDefinition) { value.Properties = nil }, wantError: "at least one property"},
		{name: "duplicate property", mutate: func(value *EntityDefinition) { value.Properties = append(value.Properties, value.Properties[0]) }, wantError: "defined more than once"},
		{name: "invalid property type", mutate: func(value *EntityDefinition) { value.Properties[0].Type = PrimitiveType("Edm.Nope") }, wantError: "unsupported EDM primitive type"},
		{name: "no key", mutate: func(value *EntityDefinition) { value.Keys = nil }, wantError: "at least one key property"},
		{name: "duplicate key", mutate: func(value *EntityDefinition) { value.Keys = []string{"ID", "ID"} }, wantError: "listed more than once"},
		{name: "missing key", mutate: func(value *EntityDefinition) { value.Keys = []string{"Missing"} }, wantError: "is not defined"},
		{name: "nullable key", mutate: func(value *EntityDefinition) { value.Properties[0].Nullable = &trueValue }, wantError: "must not be nullable"},
		{name: "invalid key type", mutate: func(value *EntityDefinition) { value.Properties[0].Type = PrimitiveTypeDouble }, wantError: "cannot use Edm.Double"},
		{name: "unsupported method", mutate: func(value *EntityDefinition) { value.DisabledMethods = []string{"TRACE"} }, wantError: "unsupported HTTP method"},
		{name: "duplicate method", mutate: func(value *EntityDefinition) { value.DisabledMethods = []string{"post", " POST "} }, wantError: "disabled more than once"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := valid
			definition.Properties = append([]PropertyDefinition(nil), valid.Properties...)
			definition.Keys = append([]string(nil), valid.Keys...)
			test.mutate(&definition)
			if err := definition.Validate(); err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Validate() error = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}

func TestEntityDefinitionAcceptsPrimitivePropertiesAndKeyTypes(t *testing.T) {
	t.Parallel()

	for primitiveType := range validPrimitiveTypes {
		definition := EntityDefinition{
			Name:          "Record",
			EntitySetName: "Records",
			Properties: []PropertyDefinition{
				{Name: "ID", Type: PrimitiveTypeInt64},
				{Name: "Value", Type: primitiveType},
			},
			Keys: []string{"ID"},
		}
		if err := definition.Validate(); err != nil {
			t.Errorf("Validate() rejected property type %s: %v", primitiveType, err)
		}
	}

	for _, primitiveType := range []PrimitiveType{
		PrimitiveTypeBoolean, PrimitiveTypeByte, PrimitiveTypeDate, PrimitiveTypeDateTimeOffset,
		PrimitiveTypeDecimal, PrimitiveTypeDuration, PrimitiveTypeGuid, PrimitiveTypeInt16,
		PrimitiveTypeInt32, PrimitiveTypeInt64, PrimitiveTypeSByte, PrimitiveTypeString,
		PrimitiveTypeTimeOfDay,
	} {
		definition := EntityDefinition{
			Name:          "Record",
			EntitySetName: "Records",
			Properties:    []PropertyDefinition{{Name: "ID", Type: primitiveType}},
			Keys:          []string{"ID"},
		}
		if err := definition.Validate(); err != nil {
			t.Errorf("Validate() rejected key type %s: %v", primitiveType, err)
		}
	}
}

func TestEntityDefinitionRejectsInvalidPrimitiveKeyTypes(t *testing.T) {
	t.Parallel()

	invalidKeyTypes := []PrimitiveType{
		PrimitiveTypeBinary,
		PrimitiveTypeDouble,
		PrimitiveTypeSingle,
		PrimitiveTypeStream,
		PrimitiveTypeUntyped,
		PrimitiveTypeGeography,
		PrimitiveTypeGeographyCollection,
		PrimitiveTypeGeographyLineString,
		PrimitiveTypeGeographyMultiLineString,
		PrimitiveTypeGeographyMultiPoint,
		PrimitiveTypeGeographyMultiPolygon,
		PrimitiveTypeGeographyPoint,
		PrimitiveTypeGeographyPolygon,
		PrimitiveTypeGeometry,
		PrimitiveTypeGeometryCollection,
		PrimitiveTypeGeometryLineString,
		PrimitiveTypeGeometryMultiLineString,
		PrimitiveTypeGeometryMultiPoint,
		PrimitiveTypeGeometryMultiPolygon,
		PrimitiveTypeGeometryPoint,
		PrimitiveTypeGeometryPolygon,
	}

	for _, primitiveType := range invalidKeyTypes {
		t.Run(string(primitiveType), func(t *testing.T) {
			definition := EntityDefinition{
				Name:          "Record",
				EntitySetName: "Records",
				Properties:    []PropertyDefinition{{Name: "ID", Type: primitiveType}},
				Keys:          []string{"ID"},
			}
			if err := definition.Validate(); err == nil || !strings.Contains(err.Error(), "cannot use") {
				t.Fatalf("Validate() error = %v, want invalid key type error", err)
			}
		})
	}
}
