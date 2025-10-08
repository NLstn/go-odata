package metadata

import (
	"reflect"
	"testing"
)

type TestProduct struct {
	ID          int     `json:"id" odata:"key"`
	Name        string  `json:"name" odata:"required"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type TestProductNoKey struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type TestProductWithAutoKey struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func TestAnalyzeEntity(t *testing.T) {
	tests := []struct {
		name        string
		entity      interface{}
		wantErr     bool
		checkResult func(*testing.T, *EntityMetadata)
	}{
		{
			name:    "valid entity with explicit key",
			entity:  TestProduct{},
			wantErr: false,
			checkResult: func(t *testing.T, meta *EntityMetadata) {
				if meta.EntityName != "TestProduct" {
					t.Errorf("EntityName = %v, want TestProduct", meta.EntityName)
				}
				if meta.EntitySetName != "TestProducts" {
					t.Errorf("EntitySetName = %v, want TestProducts", meta.EntitySetName)
				}
				if meta.KeyProperty == nil {
					t.Fatal("KeyProperty is nil")
				}
				if meta.KeyProperty.Name != "ID" {
					t.Errorf("KeyProperty.Name = %v, want ID", meta.KeyProperty.Name)
				}
				if len(meta.Properties) != 4 {
					t.Errorf("len(Properties) = %v, want 4", len(meta.Properties))
				}
			},
		},
		{
			name:    "valid entity with auto-detected key",
			entity:  TestProductWithAutoKey{},
			wantErr: false,
			checkResult: func(t *testing.T, meta *EntityMetadata) {
				if meta.KeyProperty == nil {
					t.Fatal("KeyProperty is nil")
				}
				if meta.KeyProperty.Name != "ID" {
					t.Errorf("KeyProperty.Name = %v, want ID", meta.KeyProperty.Name)
				}
			},
		},
		{
			name:    "entity with pointer",
			entity:  &TestProduct{},
			wantErr: false,
			checkResult: func(t *testing.T, meta *EntityMetadata) {
				if meta.EntityName != "TestProduct" {
					t.Errorf("EntityName = %v, want TestProduct", meta.EntityName)
				}
			},
		},
		{
			name:    "entity without key",
			entity:  TestProductNoKey{},
			wantErr: true,
		},
		{
			name:    "non-struct entity",
			entity:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AnalyzeEntity(tt.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, got)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Product", "Products"},
		{"Category", "Categories"},
		{"Box", "Boxes"},
		{"Buzz", "Buzzes"},
		{"Church", "Churches"},
		{"Dish", "Dishes"},
		{"Class", "Classes"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := pluralize(tt.input)
			if got != tt.want {
				t.Errorf("pluralize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetJsonName(t *testing.T) {
	type TestStruct struct {
		NoTag     string
		WithTag   string `json:"custom_name"`
		OmitEmpty string `json:",omitempty"`
		Both      string `json:"another_name,omitempty"`
	}

	entityType := reflect.TypeOf(TestStruct{})

	tests := []struct {
		fieldName string
		want      string
	}{
		{"NoTag", "NoTag"},
		{"WithTag", "custom_name"},
		{"OmitEmpty", "OmitEmpty"},
		{"Both", "another_name"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field, _ := entityType.FieldByName(tt.fieldName)
			got := getJsonName(field)
			if got != tt.want {
				t.Errorf("getJsonName(%q) = %q, want %q", tt.fieldName, got, tt.want)
			}
		})
	}
}

func TestPropertyMetadata(t *testing.T) {
	entity := TestProduct{}
	meta, err := AnalyzeEntity(entity)
	if err != nil {
		t.Fatalf("AnalyzeEntity() error = %v", err)
	}

	// Check required property
	var nameProperty *PropertyMetadata
	for i, prop := range meta.Properties {
		if prop.Name == "Name" {
			nameProperty = &meta.Properties[i]
			break
		}
	}

	if nameProperty == nil {
		t.Fatal("Name property not found")
	}

	if !nameProperty.IsRequired {
		t.Error("Name property should be required")
	}

	if nameProperty.JsonName != "name" {
		t.Errorf("Name property JsonName = %v, want name", nameProperty.JsonName)
	}
}
