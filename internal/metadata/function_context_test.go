package metadata

import (
	"reflect"
	"testing"
)

type testEntity struct {
	ID   int
	Name string
}

type testComplex struct {
	Street string
}

func TestFunctionContextFragment(t *testing.T) {
	entities := map[string]*EntityMetadata{
		"TestEntities": {
			EntityType:    reflect.TypeOf(testEntity{}),
			EntitySetName: "TestEntities",
		},
	}

	tests := []struct {
		name       string
		returnType reflect.Type
		entities   map[string]*EntityMetadata
		namespace  string
		want       string
	}{
		{
			name:       "primitive type",
			returnType: reflect.TypeOf(""),
			entities:   entities,
			namespace:  "",
			want:       "Edm.String",
		},
		{
			name:       "primitive collection",
			returnType: reflect.TypeOf([]int{}),
			entities:   entities,
			namespace:  "",
			want:       "Collection(Edm.Int32)",
		},
		{
			name:       "entity type",
			returnType: reflect.TypeOf(testEntity{}),
			entities:   entities,
			namespace:  "",
			want:       "TestEntities/$entity",
		},
		{
			name:       "entity collection",
			returnType: reflect.TypeOf([]testEntity{}),
			entities:   entities,
			namespace:  "",
			want:       "TestEntities",
		},
		{
			name:       "complex type",
			returnType: reflect.TypeOf(testComplex{}),
			entities:   entities,
			namespace:  "",
			want:       "ODataService.testComplex",
		},
		{
			name:       "complex type with custom namespace",
			returnType: reflect.TypeOf(testComplex{}),
			entities:   entities,
			namespace:  "Contoso.NS",
			want:       "Contoso.NS.testComplex",
		},
		{
			name:       "complex collection",
			returnType: reflect.TypeOf([]testComplex{}),
			entities:   entities,
			namespace:  "",
			want:       "Collection(ODataService.testComplex)",
		},
		{
			name:       "untyped map",
			returnType: reflect.TypeOf(map[string]interface{}{}),
			entities:   entities,
			namespace:  "",
			want:       "Edm.Untyped",
		},
		{
			name:       "untyped collection",
			returnType: reflect.TypeOf([]map[string]interface{}{}),
			entities:   entities,
			namespace:  "",
			want:       "Collection(Edm.Untyped)",
		},
		{
			name:       "nil return type",
			returnType: nil,
			entities:   entities,
			namespace:  "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FunctionContextFragment(tt.returnType, tt.entities, tt.namespace)
			if got != tt.want {
				t.Fatalf("FunctionContextFragment() = %q, want %q", got, tt.want)
			}
		})
	}
}
