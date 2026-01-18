package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ComplexAddress is a complex type for testing complex properties
type ComplexAddress struct {
	Street  string `json:"Street"`
	City    string `json:"City"`
	Country string `json:"Country"`
	ZipCode string `json:"ZipCode"`
}

// NestedComplex is a nested complex type for testing
type NestedComplex struct {
	Inner  string `json:"Inner"`
	Number int    `json:"Number"`
}

// ComplexPropertyTestEntity is a test entity with complex properties
type ComplexPropertyTestEntity struct {
	ID              uint              `json:"ID" gorm:"primaryKey" odata:"key"`
	Name            string            `json:"Name"`
	ShippingAddress ComplexAddress    `json:"ShippingAddress" gorm:"embedded;embeddedPrefix:shipping_"`
	BillingAddress  *ComplexAddress   `json:"BillingAddress" gorm:"embedded;embeddedPrefix:billing_"`
	Nested          *NestedComplex    `json:"Nested" gorm:"embedded;embeddedPrefix:nested_"`
}

func setupComplexPropertyHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ComplexPropertyTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(ComplexPropertyTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	return handler, db
}

func TestHandleComplexTypeProperty_GetComplex(t *testing.T) {
	handler, db := setupComplexPropertyHandler(t)

	// Insert test data
	entity := ComplexPropertyTestEntity{
		ID:   1,
		Name: "Test Entity",
		ShippingAddress: ComplexAddress{
			Street:  "123 Main St",
			City:    "New York",
			Country: "USA",
			ZipCode: "10001",
		},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyTestEntities(1)/ShippingAddress", nil)
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "1", []string{"ShippingAddress"}, false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["Street"] != "123 Main St" {
		t.Errorf("Street = %v, want '123 Main St'", response["Street"])
	}

	if response["City"] != "New York" {
		t.Errorf("City = %v, want 'New York'", response["City"])
	}

	// Check @odata.context
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Expected @odata.context in response")
	}
}

func TestHandleComplexTypeProperty_GetNestedProperty(t *testing.T) {
	handler, db := setupComplexPropertyHandler(t)

	// Insert test data
	entity := ComplexPropertyTestEntity{
		ID:   1,
		Name: "Test Entity",
		ShippingAddress: ComplexAddress{
			Street:  "123 Main St",
			City:    "New York",
			Country: "USA",
			ZipCode: "10001",
		},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyTestEntities(1)/ShippingAddress/City", nil)
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "1", []string{"ShippingAddress", "City"}, false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["value"] != "New York" {
		t.Errorf("value = %v, want 'New York'", response["value"])
	}
}

func TestHandleComplexTypeProperty_GetNestedRawValue(t *testing.T) {
	handler, db := setupComplexPropertyHandler(t)

	// Insert test data
	entity := ComplexPropertyTestEntity{
		ID:   1,
		Name: "Test Entity",
		ShippingAddress: ComplexAddress{
			Street:  "123 Main St",
			City:    "New York",
			Country: "USA",
			ZipCode: "10001",
		},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyTestEntities(1)/ShippingAddress/City/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "1", []string{"ShippingAddress", "City"}, true)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	body := w.Body.String()
	if body != "New York" {
		t.Errorf("body = %v, want 'New York'", body)
	}
}

func TestHandleComplexTypeProperty_ValueOnComplexNotAllowed(t *testing.T) {
	handler, db := setupComplexPropertyHandler(t)

	// Insert test data
	entity := ComplexPropertyTestEntity{
		ID:   1,
		Name: "Test Entity",
		ShippingAddress: ComplexAddress{
			Street:  "123 Main St",
			City:    "New York",
			Country: "USA",
			ZipCode: "10001",
		},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	// $value is not supported on complex properties directly
	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyTestEntities(1)/ShippingAddress/$value", nil)
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "1", []string{"ShippingAddress"}, true)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleComplexTypeProperty_EntityNotFound(t *testing.T) {
	handler, _ := setupComplexPropertyHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyTestEntities(999)/ShippingAddress", nil)
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "999", []string{"ShippingAddress"}, false)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleComplexTypeProperty_PropertyNotFound(t *testing.T) {
	handler, db := setupComplexPropertyHandler(t)

	// Insert test data
	entity := ComplexPropertyTestEntity{
		ID:   1,
		Name: "Test Entity",
		ShippingAddress: ComplexAddress{
			Street:  "123 Main St",
			City:    "New York",
			Country: "USA",
			ZipCode: "10001",
		},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyTestEntities(1)/NonExistentProperty", nil)
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "1", []string{"NonExistentProperty"}, false)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleComplexTypeProperty_NestedPropertyNotFound(t *testing.T) {
	handler, db := setupComplexPropertyHandler(t)

	// Insert test data
	entity := ComplexPropertyTestEntity{
		ID:   1,
		Name: "Test Entity",
		ShippingAddress: ComplexAddress{
			Street:  "123 Main St",
			City:    "New York",
			Country: "USA",
			ZipCode: "10001",
		},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyTestEntities(1)/ShippingAddress/InvalidProperty", nil)
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "1", []string{"ShippingAddress", "InvalidProperty"}, false)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleComplexTypeProperty_EmptySegments(t *testing.T) {
	handler, _ := setupComplexPropertyHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyTestEntities(1)/", nil)
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "1", []string{}, false)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandleComplexTypeProperty_Head(t *testing.T) {
	handler, db := setupComplexPropertyHandler(t)

	// Insert test data
	entity := ComplexPropertyTestEntity{
		ID:   1,
		Name: "Test Entity",
		ShippingAddress: ComplexAddress{
			Street:  "123 Main St",
			City:    "New York",
			Country: "USA",
			ZipCode: "10001",
		},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/ComplexPropertyTestEntities(1)/ShippingAddress", nil)
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "1", []string{"ShippingAddress"}, false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestHandleComplexTypeProperty_Options(t *testing.T) {
	handler, _ := setupComplexPropertyHandler(t)

	req := httptest.NewRequest(http.MethodOptions, "/ComplexPropertyTestEntities(1)/ShippingAddress", nil)
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "1", []string{"ShippingAddress"}, false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}
}

func TestHandleComplexTypeProperty_MethodNotAllowed(t *testing.T) {
	handler, _ := setupComplexPropertyHandler(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/ComplexPropertyTestEntities(1)/ShippingAddress", nil)
			w := httptest.NewRecorder()

			handler.HandleComplexTypeProperty(w, req, "1", []string{"ShippingAddress"}, false)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestHandleComplexTypeProperty_MetadataLevel_None(t *testing.T) {
	handler, db := setupComplexPropertyHandler(t)

	// Insert test data
	entity := ComplexPropertyTestEntity{
		ID:   1,
		Name: "Test Entity",
		ShippingAddress: ComplexAddress{
			Street:  "123 Main St",
			City:    "New York",
			Country: "USA",
			ZipCode: "10001",
		},
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyTestEntities(1)/ShippingAddress", nil)
	req.Header.Set("Accept", "application/json;odata.metadata=none")
	w := httptest.NewRecorder()

	handler.HandleComplexTypeProperty(w, req, "1", []string{"ShippingAddress"}, false)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// With metadata=none, @odata.context should not be present
	if _, ok := response["@odata.context"]; ok {
		t.Error("Expected @odata.context to NOT be present with metadata=none")
	}

	// But data should still be present
	if response["Street"] != "123 Main St" {
		t.Errorf("Street = %v, want '123 Main St'", response["Street"])
	}
}

// Test helper functions from complex_properties.go
func TestResolveStructField(t *testing.T) {
	type TestStruct struct {
		Name    string  `json:"name"`
		Value   int     `json:"value"`
		Private string  // unexported, no json tag
		Hidden  float64 `json:"-"`
	}

	s := TestStruct{
		Name:    "test",
		Value:   42,
		Private: "private",
		Hidden:  3.14,
	}

	t.Run("resolve by json name", func(t *testing.T) {
		v := reflect.ValueOf(s)
		val, _, jsonName, ok := resolveStructField(v, "name")
		if !ok {
			t.Error("Expected to find field 'name'")
		}
		if jsonName != "name" {
			t.Errorf("jsonName = %v, want 'name'", jsonName)
		}
		if val.String() != "test" {
			t.Errorf("value = %v, want 'test'", val.String())
		}
	})

	t.Run("resolve by struct field name", func(t *testing.T) {
		v := reflect.ValueOf(s)
		val, _, _, ok := resolveStructField(v, "Name")
		if !ok {
			t.Error("Expected to find field 'Name'")
		}
		if val.String() != "test" {
			t.Errorf("value = %v, want 'test'", val.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		v := reflect.ValueOf(s)
		_, _, _, ok := resolveStructField(v, "nonexistent")
		if ok {
			t.Error("Expected to not find field 'nonexistent'")
		}
	})

	t.Run("non-struct type", func(t *testing.T) {
		v := reflect.ValueOf(42)
		_, _, _, ok := resolveStructField(v, "any")
		if ok {
			t.Error("Expected to not find field in non-struct")
		}
	})
}

func TestResolveMapField(t *testing.T) {
	m := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}

	t.Run("exact match", func(t *testing.T) {
		v := reflect.ValueOf(m)
		val, _, _, ok := resolveMapField(v, "name")
		if !ok {
			t.Error("Expected to find key 'name'")
		}
		if val.Interface() != "test" {
			t.Errorf("value = %v, want 'test'", val.Interface())
		}
	})

	t.Run("case insensitive match", func(t *testing.T) {
		v := reflect.ValueOf(m)
		val, _, _, ok := resolveMapField(v, "NAME")
		if !ok {
			t.Error("Expected to find key 'NAME' (case-insensitive)")
		}
		if val.Interface() != "test" {
			t.Errorf("value = %v, want 'test'", val.Interface())
		}
	})

	t.Run("not found", func(t *testing.T) {
		v := reflect.ValueOf(m)
		_, _, _, ok := resolveMapField(v, "nonexistent")
		if ok {
			t.Error("Expected to not find key 'nonexistent'")
		}
	})

	t.Run("non-map type", func(t *testing.T) {
		v := reflect.ValueOf(42)
		_, _, _, ok := resolveMapField(v, "any")
		if ok {
			t.Error("Expected to not find key in non-map")
		}
	})

	t.Run("map with non-string keys", func(t *testing.T) {
		m2 := map[int]string{1: "one"}
		v := reflect.ValueOf(m2)
		_, _, _, ok := resolveMapField(v, "1")
		if ok {
			t.Error("Expected to not find in map with non-string keys")
		}
	})
}

func TestIsNilPointer(t *testing.T) {
	t.Run("nil pointer", func(t *testing.T) {
		var p *int
		v := reflect.ValueOf(p)
		if !isNilPointer(v) {
			t.Error("Expected isNilPointer to be true for nil pointer")
		}
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		i := 42
		p := &i
		v := reflect.ValueOf(p)
		if isNilPointer(v) {
			t.Error("Expected isNilPointer to be false for non-nil pointer")
		}
	})

	t.Run("nil map", func(t *testing.T) {
		var m map[string]string
		v := reflect.ValueOf(m)
		if !isNilPointer(v) {
			t.Error("Expected isNilPointer to be true for nil map")
		}
	})

	t.Run("nil slice", func(t *testing.T) {
		var s []int
		v := reflect.ValueOf(s)
		if !isNilPointer(v) {
			t.Error("Expected isNilPointer to be true for nil slice")
		}
	})

	t.Run("value type", func(t *testing.T) {
		v := reflect.ValueOf(42)
		if isNilPointer(v) {
			t.Error("Expected isNilPointer to be false for value type")
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		var v reflect.Value
		if !isNilPointer(v) {
			t.Error("Expected isNilPointer to be true for invalid value")
		}
	})
}

func TestDereferenceValue(t *testing.T) {
	t.Run("pointer value", func(t *testing.T) {
		i := 42
		p := &i
		v := reflect.ValueOf(p)
		deref := dereferenceValue(v)
		if deref.Int() != 42 {
			t.Errorf("value = %v, want 42", deref.Int())
		}
	})

	t.Run("double pointer", func(t *testing.T) {
		i := 42
		p := &i
		pp := &p
		v := reflect.ValueOf(pp)
		deref := dereferenceValue(v)
		if deref.Int() != 42 {
			t.Errorf("value = %v, want 42", deref.Int())
		}
	})

	t.Run("nil pointer", func(t *testing.T) {
		var p *int
		v := reflect.ValueOf(p)
		deref := dereferenceValue(v)
		if deref.IsValid() {
			t.Error("Expected invalid value for nil pointer")
		}
	})

	t.Run("value type", func(t *testing.T) {
		v := reflect.ValueOf(42)
		deref := dereferenceValue(v)
		if deref.Int() != 42 {
			t.Errorf("value = %v, want 42", deref.Int())
		}
	})
}

func TestDereferenceType(t *testing.T) {
	t.Run("pointer type", func(t *testing.T) {
		var i *int
		typ := reflect.TypeOf(i)
		deref := dereferenceType(typ)
		if deref.Kind() != reflect.Int {
			t.Errorf("kind = %v, want int", deref.Kind())
		}
	})

	t.Run("double pointer", func(t *testing.T) {
		var i **int
		typ := reflect.TypeOf(i)
		deref := dereferenceType(typ)
		if deref.Kind() != reflect.Int {
			t.Errorf("kind = %v, want int", deref.Kind())
		}
	})

	t.Run("nil type", func(t *testing.T) {
		deref := dereferenceType(nil)
		if deref != nil {
			t.Error("Expected nil for nil type")
		}
	})

	t.Run("value type", func(t *testing.T) {
		typ := reflect.TypeOf(42)
		deref := dereferenceType(typ)
		if deref.Kind() != reflect.Int {
			t.Errorf("kind = %v, want int", deref.Kind())
		}
	})
}

func TestIsComplexValue(t *testing.T) {
	t.Run("struct value", func(t *testing.T) {
		type TestStruct struct{ Name string }
		s := TestStruct{Name: "test"}
		v := reflect.ValueOf(s)
		if !isComplexValue(v, v.Type()) {
			t.Error("Expected isComplexValue to be true for struct")
		}
	})

	t.Run("map value", func(t *testing.T) {
		m := map[string]interface{}{"name": "test"}
		v := reflect.ValueOf(m)
		if !isComplexValue(v, v.Type()) {
			t.Error("Expected isComplexValue to be true for map")
		}
	})

	t.Run("time.Time value", func(t *testing.T) {
		// time.Time is a struct but should be treated as primitive
		now := reflect.ValueOf(time.Now())
		if isComplexValue(now, now.Type()) {
			t.Error("Expected isComplexValue to be false for time.Time")
		}
	})

	t.Run("primitive value", func(t *testing.T) {
		v := reflect.ValueOf(42)
		if isComplexValue(v, v.Type()) {
			t.Error("Expected isComplexValue to be false for primitive")
		}
	})

	t.Run("nil type", func(t *testing.T) {
		var v reflect.Value
		if isComplexValue(v, nil) {
			t.Error("Expected isComplexValue to be false for nil type")
		}
	})
}

func TestStructValueToMap(t *testing.T) {
	type TestStruct struct {
		Name   string  `json:"name"`
		Value  int     `json:"value"`
		Hidden float64 `json:"-"`
	}

	s := TestStruct{
		Name:   "test",
		Value:  42,
		Hidden: 3.14,
	}

	v := reflect.ValueOf(s)
	m := structValueToMap(v)

	if m["name"] != "test" {
		t.Errorf("name = %v, want 'test'", m["name"])
	}

	if m["value"] != 42 {
		t.Errorf("value = %v, want 42", m["value"])
	}

	// Hidden field should be excluded (json:"-")
	if _, ok := m["Hidden"]; ok {
		t.Error("Hidden field should be excluded")
	}
}

func TestGetJSONFieldName(t *testing.T) {
	type TestStruct struct {
		Name      string  `json:"name"`
		Value     int     `json:"value,omitempty"`
		NoTag     string  // no json tag
		Empty     string  `json:""`
		Hidden    float64 `json:"-"`
		OmitEmpty string  `json:",omitempty"` // empty name but has options
	}

	typ := reflect.TypeOf(TestStruct{})

	tests := []struct {
		fieldName    string
		expectedJSON string
	}{
		{"Name", "name"},
		{"Value", "value"},
		{"NoTag", "NoTag"},
		{"Empty", "Empty"},
		{"Hidden", "-"},
		{"OmitEmpty", "OmitEmpty"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field, _ := typ.FieldByName(tt.fieldName)
			result := getJSONFieldName(field)
			if result != tt.expectedJSON {
				t.Errorf("getJSONFieldName(%s) = %v, want %v", tt.fieldName, result, tt.expectedJSON)
			}
		})
	}
}
