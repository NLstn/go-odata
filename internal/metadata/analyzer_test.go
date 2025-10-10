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

func TestPropertyFacets(t *testing.T) {
	type EntityWithFacets struct {
		ID          int     `json:"id" odata:"key"`
		Name        string  `json:"name" odata:"maxlength=100"`
		Description string  `json:"description" odata:"maxlength=500,nullable"`
		Price       float64 `json:"price" odata:"precision=10,scale=2"`
		SKU         string  `json:"sku" odata:"default=AUTO"`
		Active      bool    `json:"active" odata:"nullable=false"`
	}

	meta, err := AnalyzeEntity(EntityWithFacets{})
	if err != nil {
		t.Fatalf("AnalyzeEntity() error = %v", err)
	}

	// Check name with maxlength
	var nameProp *PropertyMetadata
	for i, prop := range meta.Properties {
		if prop.Name == "Name" {
			nameProp = &meta.Properties[i]
			break
		}
	}
	if nameProp == nil {
		t.Fatal("Name property not found")
	}
	if nameProp.MaxLength != 100 {
		t.Errorf("Name MaxLength = %v, want 100", nameProp.MaxLength)
	}

	// Check description with maxlength and nullable
	var descProp *PropertyMetadata
	for i, prop := range meta.Properties {
		if prop.Name == "Description" {
			descProp = &meta.Properties[i]
			break
		}
	}
	if descProp == nil {
		t.Fatal("Description property not found")
	}
	if descProp.MaxLength != 500 {
		t.Errorf("Description MaxLength = %v, want 500", descProp.MaxLength)
	}
	if descProp.Nullable == nil || !*descProp.Nullable {
		t.Error("Description should be nullable")
	}

	// Check price with precision and scale
	var priceProp *PropertyMetadata
	for i, prop := range meta.Properties {
		if prop.Name == "Price" {
			priceProp = &meta.Properties[i]
			break
		}
	}
	if priceProp == nil {
		t.Fatal("Price property not found")
	}
	if priceProp.Precision != 10 {
		t.Errorf("Price Precision = %v, want 10", priceProp.Precision)
	}
	if priceProp.Scale != 2 {
		t.Errorf("Price Scale = %v, want 2", priceProp.Scale)
	}

	// Check SKU with default value
	var skuProp *PropertyMetadata
	for i, prop := range meta.Properties {
		if prop.Name == "SKU" {
			skuProp = &meta.Properties[i]
			break
		}
	}
	if skuProp == nil {
		t.Fatal("SKU property not found")
	}
	if skuProp.DefaultValue != "AUTO" {
		t.Errorf("SKU DefaultValue = %v, want AUTO", skuProp.DefaultValue)
	}

	// Check Active with nullable=false
	var activeProp *PropertyMetadata
	for i, prop := range meta.Properties {
		if prop.Name == "Active" {
			activeProp = &meta.Properties[i]
			break
		}
	}
	if activeProp == nil {
		t.Fatal("Active property not found")
	}
	if activeProp.Nullable == nil || *activeProp.Nullable {
		t.Error("Active should not be nullable")
	}
}

func TestNavigationPropertyWithReferentialConstraints(t *testing.T) {
	type User struct {
		ID   int    `json:"id" odata:"key"`
		Name string `json:"name"`
	}

	type Order struct {
		ID         int   `json:"id" odata:"key"`
		CustomerID int   `json:"customerId"`
		Customer   *User `json:"customer" gorm:"foreignKey:CustomerID;references:ID"`
	}

	meta, err := AnalyzeEntity(Order{})
	if err != nil {
		t.Fatalf("AnalyzeEntity() error = %v", err)
	}

	// Find customer navigation property
	var customerProp *PropertyMetadata
	for i, prop := range meta.Properties {
		if prop.Name == "Customer" {
			customerProp = &meta.Properties[i]
			break
		}
	}

	if customerProp == nil {
		t.Fatal("Customer navigation property not found")
	}

	if !customerProp.IsNavigationProp {
		t.Error("Customer should be a navigation property")
	}

	if customerProp.NavigationTarget != "User" {
		t.Errorf("NavigationTarget = %v, want User", customerProp.NavigationTarget)
	}

	if len(customerProp.ReferentialConstraints) == 0 {
		t.Fatal("ReferentialConstraints should not be empty")
	}

	if referencedProp, ok := customerProp.ReferentialConstraints["CustomerID"]; !ok || referencedProp != "ID" {
		t.Errorf("Expected CustomerID -> ID constraint, got %v", customerProp.ReferentialConstraints)
	}
}
