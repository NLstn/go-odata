package metadata

import (
	"reflect"
	"strings"
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

type TestOrder struct {
	ID          int             `json:"id" odata:"key"`
	Customer    string          `json:"customer"`
	ShipAddress OrderAddress    `json:"shipAddress" gorm:"embedded"`
	Lines       []TestOrderLine `json:"lines" gorm:"foreignKey:OrderID"`
}

type TestOrderLine struct {
	OrderID int    `json:"orderId"`
	SKU     string `json:"sku"`
}

type OrderAddress struct {
	City string `json:"city"`
}

type MethodEnum int32

const (
	MethodEnumZero MethodEnum = 0
	MethodEnumOne  MethodEnum = 1
)

func (MethodEnum) EnumMembers() map[string]int {
	return map[string]int{
		"Zero": int(MethodEnumZero),
		"One":  int(MethodEnumOne),
	}
}

type MethodEnumEntity struct {
	ID    int        `json:"id" odata:"key"`
	Value MethodEnum `json:"value" odata:"enum=MethodEnum"`
}

type MissingEnum int

type MissingEnumEntity struct {
	ID    int         `json:"id" odata:"key"`
	Value MissingEnum `json:"value" odata:"enum=MissingEnum"`
}

type PrefixedAddress struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

type PrefixedOrder struct {
	ID              int              `json:"id" odata:"key"`
	ShippingAddress *PrefixedAddress `json:"shippingAddress,omitempty" gorm:"embedded;embeddedPrefix:shipping_" odata:"nullable"`
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

func TestAnalyzeEntity_KeyGenerator(t *testing.T) {
	type GeneratedEntity struct {
		ID   string `json:"id" odata:"key,generate=uuid"`
		Name string `json:"name"`
	}

	meta, err := AnalyzeEntity(GeneratedEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity() error = %v", err)
	}

	if len(meta.KeyProperties) != 1 {
		t.Fatalf("expected 1 key property, got %d", len(meta.KeyProperties))
	}

	key := meta.KeyProperties[0]
	if key.KeyGenerator != "uuid" {
		t.Fatalf("expected key generator 'uuid', got %q", key.KeyGenerator)
	}

	if key.DatabaseGenerated {
		t.Fatal("expected generated key to disable database-managed flag")
	}

	type UnknownGenerator struct {
		ID string `json:"id" odata:"key,generate=custom"`
	}

	if _, err := AnalyzeEntity(UnknownGenerator{}); err == nil {
		t.Fatal("expected error for unknown generator")
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

func TestEntityMetadataFinders(t *testing.T) {
	meta, err := AnalyzeEntity(TestOrder{})
	if err != nil {
		t.Fatalf("AnalyzeEntity(TestOrder) returned error: %v", err)
	}

	t.Run("find property by json name", func(t *testing.T) {
		prop := meta.FindProperty("customer")
		if prop == nil {
			t.Fatalf("FindProperty returned nil")
		}
		if prop.Name != "Customer" {
			t.Fatalf("expected struct field name Customer, got %s", prop.Name)
		}
	})

	t.Run("find property by struct name", func(t *testing.T) {
		prop := meta.FindProperty("Customer")
		if prop == nil {
			t.Fatal("FindProperty by struct name returned nil")
		}
	})

	t.Run("find navigation property", func(t *testing.T) {
		nav := meta.FindNavigationProperty("lines")
		if nav == nil {
			t.Fatal("FindNavigationProperty returned nil for navigation field")
		}
		if !nav.IsNavigationProp {
			t.Fatal("expected navigation property to be marked as such")
		}
	})

	t.Run("find complex type property", func(t *testing.T) {
		complexProp := meta.FindComplexTypeProperty("shipAddress")
		if complexProp == nil {
			t.Fatal("FindComplexTypeProperty returned nil for complex field")
		}
		if !complexProp.IsComplexType {
			t.Fatal("expected property to be marked as complex type")
		}
	})

	t.Run("find structural property excludes navigation and complex", func(t *testing.T) {
		if prop := meta.FindStructuralProperty("lines"); prop != nil {
			t.Fatal("expected navigation property to be excluded from structural lookup")
		}
		if prop := meta.FindStructuralProperty("shipAddress"); prop != nil {
			t.Fatal("expected complex type to be excluded from structural lookup")
		}
		if prop := meta.FindStructuralProperty("customer"); prop == nil {
			t.Fatal("expected structural property to be returned")
		}
	})

	t.Run("returns nil for missing property", func(t *testing.T) {
		if meta.FindProperty("unknown") != nil {
			t.Fatal("expected FindProperty to return nil for missing property")
		}
	})
}

func TestAnalyzeEntityEnumMembers(t *testing.T) {
	meta, err := AnalyzeEntity(MethodEnumEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity(MethodEnumEntity) error: %v", err)
	}

	prop := meta.FindProperty("Value")
	if prop == nil {
		t.Fatalf("expected to find enum property")
	}
	if !prop.IsEnum {
		t.Fatalf("expected property to be enum")
	}
	if len(prop.EnumMembers) != 2 {
		t.Fatalf("expected 2 enum members, got %d", len(prop.EnumMembers))
	}
	if prop.EnumUnderlyingType != "Edm.Int32" {
		t.Fatalf("expected underlying type Edm.Int32, got %s", prop.EnumUnderlyingType)
	}
}

func TestAnalyzeEntityEnumMissingMembers(t *testing.T) {
	if _, err := AnalyzeEntity(MissingEnumEntity{}); err == nil {
		t.Fatalf("expected error for enum without registered members")
	}
}

func TestResolvePropertyPathForComplexType(t *testing.T) {
	meta, err := AnalyzeEntity(PrefixedOrder{})
	if err != nil {
		t.Fatalf("AnalyzeEntity(PrefixedOrder) returned error: %v", err)
	}

	complexProp := meta.FindComplexTypeProperty("ShippingAddress")
	if complexProp == nil {
		t.Fatal("expected ShippingAddress to be recognized as complex type")
	}
	if complexProp.EmbeddedPrefix != "shipping_" {
		t.Fatalf("expected embedded prefix 'shipping_', got %q", complexProp.EmbeddedPrefix)
	}

	if nested := complexProp.FindComplexField("City"); nested == nil {
		t.Fatal("expected to find nested City property by struct name")
	}
	if nested := complexProp.FindComplexField("city"); nested == nil {
		t.Fatal("expected to find nested City property by json name")
	}

	resolved, prefix, err := meta.ResolvePropertyPath("ShippingAddress/City")
	if err != nil {
		t.Fatalf("ResolvePropertyPath returned error: %v", err)
	}
	if prefix != "shipping_" {
		t.Fatalf("expected prefix 'shipping_', got %q", prefix)
	}
	if resolved == nil || resolved.Name != "City" {
		t.Fatalf("expected resolved property 'City', got %+v", resolved)
	}

	// Support resolving via json names
	resolved, prefix, err = meta.ResolvePropertyPath("shippingAddress/city")
	if err != nil {
		t.Fatalf("ResolvePropertyPath for json names returned error: %v", err)
	}
	if prefix != "shipping_" {
		t.Fatalf("expected prefix 'shipping_' for json path, got %q", prefix)
	}
	if resolved == nil || resolved.JsonName != "city" {
		t.Fatalf("expected resolved json property 'city', got %+v", resolved)
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
		ID     int     `json:"id" odata:"key"`
		Name   string  `json:"name" odata:"maxlength=100"`
		Price  float64 `json:"price" odata:"precision=10,scale=2"`
		SKU    string  `json:"sku" odata:"default=AUTO"`
		Active bool    `json:"active" odata:"nullable=false"`
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

func TestAutoDetectNullability(t *testing.T) {
	type TestEntity struct {
		ID                int     `json:"id" odata:"key"`
		NonNullableInt    int     `json:"nonNullableInt"`                         // Value type - should be non-nullable
		NullableIntPtr    *int    `json:"nullableIntPtr"`                         // Pointer - can be nullable
		StringWithNotNull string  `json:"stringWithNotNull" gorm:"not null"`      // GORM not null - should be non-nullable
		StringWithDefault string  `json:"stringWithDefault" gorm:"default:test"`  // Non-pointer with default - should be non-nullable
		NullableString    *string `json:"nullableString"`                         // Pointer - can be nullable
		ExplicitNonNull   *string `json:"explicitNonNull" odata:"nullable=false"` // Explicit non-nullable - should respect
		SliceField        []byte  `json:"sliceField"`                             // Slice - can be nullable
		RequiredField     string  `json:"requiredField" odata:"required"`         // Required - handled separately
	}

	meta, err := AnalyzeEntity(TestEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity() error = %v", err)
	}

	tests := []struct {
		fieldName      string
		expectNullable *bool // nil means should be nil (let metadata handler decide), true/false means should be set
	}{
		{"id", boolPtr(false)},                // Key field, int type is non-nullable
		{"nonNullableInt", boolPtr(false)},    // Value type
		{"nullableIntPtr", nil},               // Pointer type, no constraints
		{"stringWithNotNull", boolPtr(false)}, // GORM not null
		{"stringWithDefault", boolPtr(false)}, // Value type with default
		{"nullableString", nil},               // Pointer type
		{"explicitNonNull", boolPtr(false)},   // Explicit non-nullable
		{"sliceField", nil},                   // Slice type
		{"requiredField", boolPtr(false)},     // Required field, string type is non-nullable
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			var prop *PropertyMetadata
			for i := range meta.Properties {
				if meta.Properties[i].JsonName == tt.fieldName {
					prop = &meta.Properties[i]
					break
				}
			}

			if prop == nil {
				t.Fatalf("Property %s not found", tt.fieldName)
			}

			if tt.expectNullable == nil {
				if prop.Nullable != nil {
					t.Errorf("Property %s: expected Nullable to be nil, got %v", tt.fieldName, *prop.Nullable)
				}
			} else {
				if prop.Nullable == nil {
					t.Errorf("Property %s: expected Nullable to be %v, got nil", tt.fieldName, *tt.expectNullable)
				} else if *prop.Nullable != *tt.expectNullable {
					t.Errorf("Property %s: expected Nullable to be %v, got %v", tt.fieldName, *tt.expectNullable, *prop.Nullable)
				}
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestNullableMismatchError(t *testing.T) {
	// Test that we get an error when a non-nullable type has odata:"nullable" tag
	type InvalidEntity struct {
		ID          int    `json:"id" odata:"key"`
		Description string `json:"description" odata:"nullable"` // string is not nullable, but tag says it is
	}

	_, err := AnalyzeEntity(InvalidEntity{})
	if err == nil {
		t.Fatal("Expected error for non-nullable type with nullable tag, got nil")
	}

	expectedErrMsg := "property Description is marked as nullable"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error message to contain %q, got: %v", expectedErrMsg, err)
	}
}

func TestSearchSimilarityValidation(t *testing.T) {
	t.Run("valid similarity", func(t *testing.T) {
		type ValidEntity struct {
			ID   int    `json:"id" odata:"key"`
			Name string `json:"name" odata:"searchable,similarity=0.95"`
		}

		meta, err := AnalyzeEntity(ValidEntity{})
		if err != nil {
			t.Fatalf("Expected no error for valid similarity, got: %v", err)
		}

		// Check that similarity is properly set
		nameProp := meta.FindProperty("Name")
		if nameProp == nil {
			t.Fatal("Expected to find Name property")
		}
		if nameProp.SearchSimilarity != 0.95 {
			t.Errorf("Expected SearchSimilarity to be 0.95, got %f", nameProp.SearchSimilarity)
		}
		if !nameProp.IsSearchable {
			t.Error("Expected Name property to be marked as searchable")
		}
	})

	t.Run("both fuzziness and similarity defined", func(t *testing.T) {
		type InvalidEntity struct {
			ID   int    `json:"id" odata:"key"`
			Name string `json:"name" odata:"searchable,fuzziness=2,similarity=0.95"`
		}

		_, err := AnalyzeEntity(InvalidEntity{})
		if err == nil {
			t.Fatal("Expected error for field with both fuzziness and similarity, got nil")
		}

		expectedErrMsg := "cannot have both fuzziness and similarity defined"
		if !strings.Contains(err.Error(), expectedErrMsg) {
			t.Errorf("Expected error message to contain %q, got: %v", expectedErrMsg, err)
		}
	})

	t.Run("similarity out of range - too high", func(t *testing.T) {
		type InvalidEntity struct {
			ID   int    `json:"id" odata:"key"`
			Name string `json:"name" odata:"searchable,similarity=1.5"`
		}

		_, err := AnalyzeEntity(InvalidEntity{})
		if err == nil {
			t.Fatal("Expected error for similarity > 1.0, got nil")
		}

		expectedErrMsg := "must be between 0.0 and 1.0"
		if !strings.Contains(err.Error(), expectedErrMsg) {
			t.Errorf("Expected error message to contain %q, got: %v", expectedErrMsg, err)
		}
	})

	t.Run("similarity out of range - negative", func(t *testing.T) {
		type InvalidEntity struct {
			ID   int    `json:"id" odata:"key"`
			Name string `json:"name" odata:"searchable,similarity=-0.5"`
		}

		_, err := AnalyzeEntity(InvalidEntity{})
		if err == nil {
			t.Fatal("Expected error for negative similarity, got nil")
		}

		expectedErrMsg := "must be between 0.0 and 1.0"
		if !strings.Contains(err.Error(), expectedErrMsg) {
			t.Errorf("Expected error message to contain %q, got: %v", expectedErrMsg, err)
		}
	})

	t.Run("similarity without searchable tag", func(t *testing.T) {
		type ValidEntity struct {
			ID   int    `json:"id" odata:"key"`
			Name string `json:"name" odata:"similarity=0.8"`
		}

		meta, err := AnalyzeEntity(ValidEntity{})
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check that the property is marked as searchable when similarity is set
		nameProp := meta.FindProperty("Name")
		if nameProp == nil {
			t.Fatal("Expected to find Name property")
		}
		if !nameProp.IsSearchable {
			t.Error("Expected Name property to be marked as searchable when similarity is set")
		}
		if nameProp.SearchSimilarity != 0.8 {
			t.Errorf("Expected SearchSimilarity to be 0.8, got %f", nameProp.SearchSimilarity)
		}
	})
}
