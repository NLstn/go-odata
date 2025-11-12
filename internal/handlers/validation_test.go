package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// Test entity for validation tests
type ValidationTestEntity struct {
	ID       uint    `json:"id" gorm:"primaryKey" odata:"key"`
	Name     string  `json:"name" odata:"required"`
	Price    float64 `json:"price" odata:"required"`
	Category string  `json:"category"`
	Active   bool    `json:"active"`
}

func TestValidateDataTypes_ValidTypes(t *testing.T) {
	// Create test entity metadata
	testMetadata := &metadata.EntityMetadata{
		Properties: []metadata.PropertyMetadata{
			{Name: "Name", JsonName: "name", Type: reflect.TypeOf(""), IsRequired: true},
			{Name: "Price", JsonName: "price", Type: reflect.TypeOf(float64(0)), IsRequired: true},
			{Name: "Category", JsonName: "category", Type: reflect.TypeOf("")},
			{Name: "Active", JsonName: "active", Type: reflect.TypeOf(false)},
		},
	}

	handler := &EntityHandler{metadata: testMetadata}

	tests := []struct {
		name       string
		updateData map[string]interface{}
		wantErr    bool
	}{
		{
			name: "Valid string and number",
			updateData: map[string]interface{}{
				"name":  "Test Product",
				"price": 99.99,
			},
			wantErr: false,
		},
		{
			name: "Valid boolean",
			updateData: map[string]interface{}{
				"active": true,
			},
			wantErr: false,
		},
		{
			name: "Null value for nullable field",
			updateData: map[string]interface{}{
				"category": nil,
			},
			wantErr: false,
		},
		{
			name: "Integer as float64 (from JSON)",
			updateData: map[string]interface{}{
				"price": float64(100),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateDataTypes(tt.updateData)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDataTypes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDataTypes_InvalidTypes(t *testing.T) {
	testMetadata := &metadata.EntityMetadata{
		Properties: []metadata.PropertyMetadata{
			{Name: "Name", JsonName: "name", Type: reflect.TypeOf(""), IsRequired: true},
			{Name: "Price", JsonName: "price", Type: reflect.TypeOf(float64(0)), IsRequired: true},
			{Name: "Active", JsonName: "active", Type: reflect.TypeOf(false)},
		},
	}

	handler := &EntityHandler{metadata: testMetadata}

	tests := []struct {
		name       string
		updateData map[string]interface{}
		wantErr    bool
	}{
		{
			name: "String for number field",
			updateData: map[string]interface{}{
				"price": "invalid",
			},
			wantErr: true,
		},
		{
			name: "Number for string field",
			updateData: map[string]interface{}{
				"name": 123,
			},
			wantErr: true,
		},
		{
			name: "String for boolean field",
			updateData: map[string]interface{}{
				"active": "true",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateDataTypes(tt.updateData)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDataTypes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRequiredFieldsNotNull(t *testing.T) {
	testMetadata := &metadata.EntityMetadata{
		Properties: []metadata.PropertyMetadata{
			{Name: "Name", JsonName: "name", Type: reflect.TypeOf(""), IsRequired: true},
			{Name: "Price", JsonName: "price", Type: reflect.TypeOf(float64(0)), IsRequired: true},
			{Name: "Category", JsonName: "category", Type: reflect.TypeOf(""), IsRequired: false},
		},
	}

	handler := &EntityHandler{metadata: testMetadata}

	tests := []struct {
		name       string
		updateData map[string]interface{}
		wantErr    bool
	}{
		{
			name: "Null for optional field",
			updateData: map[string]interface{}{
				"category": nil,
			},
			wantErr: false,
		},
		{
			name: "Non-null for required field",
			updateData: map[string]interface{}{
				"name": "Test",
			},
			wantErr: false,
		},
		{
			name: "Null for required field",
			updateData: map[string]interface{}{
				"name": nil,
			},
			wantErr: true,
		},
		{
			name: "Multiple nulls including required",
			updateData: map[string]interface{}{
				"category": nil,
				"price":    nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateRequiredFieldsNotNull(tt.updateData)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequiredFieldsNotNull() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantErr     bool
		wantStatus  int
	}{
		{
			name:        "Valid application/json",
			contentType: "application/json",
			wantErr:     false,
		},
		{
			name:        "Valid with charset",
			contentType: "application/json; charset=utf-8",
			wantErr:     false,
		},
		{
			name:        "Valid with odata.metadata=minimal",
			contentType: "application/json;odata.metadata=minimal",
			wantErr:     false,
		},
		{
			name:        "Valid with odata.metadata=full",
			contentType: "application/json;odata.metadata=full",
			wantErr:     false,
		},
		{
			name:        "Missing Content-Type",
			contentType: "",
			wantErr:     true,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "Invalid Content-Type",
			contentType: "text/plain",
			wantErr:     true,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "Invalid Content-Type XML",
			contentType: "application/xml",
			wantErr:     true,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			err := validateContentType(w, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateContentType() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.wantStatus != 0 {
				if w.Code != tt.wantStatus {
					t.Errorf("validateContentType() status = %v, want %v", w.Code, tt.wantStatus)
				}
			}
		})
	}
}

func TestClearAutoIncrementKeys(t *testing.T) {
	tests := []struct {
		name             string
		entity           interface{}
		keyProperties    []metadata.PropertyMetadata
		expectedID       interface{}
		expectedIDIsZero bool
	}{
		{
			name: "Single uint key - should be cleared",
			entity: &struct {
				ID   uint   `json:"id" gorm:"primaryKey"`
				Name string `json:"name"`
			}{
				ID:   99999,
				Name: "Test",
			},
			keyProperties: []metadata.PropertyMetadata{
				{Name: "ID", JsonName: "id", Type: reflect.TypeOf(uint(0)), GormTag: "primaryKey", DatabaseGenerated: true},
			},
			expectedIDIsZero: true,
		},
		{
			name: "Single int key - should NOT be cleared",
			entity: &struct {
				ID   int    `json:"id" gorm:"primaryKey"`
				Name string `json:"name"`
			}{
				ID:   42,
				Name: "Test",
			},
			keyProperties: []metadata.PropertyMetadata{
				{Name: "ID", JsonName: "id", Type: reflect.TypeOf(int(0)), GormTag: "primaryKey"},
			},
			expectedID: 42,
		},
		{
			name: "Composite keys - should NOT be cleared",
			entity: &struct {
				ProductID   int    `json:"productId" gorm:"primaryKey"`
				LanguageKey string `json:"languageKey" gorm:"primaryKey"`
				Name        string `json:"name"`
			}{
				ProductID:   1,
				LanguageKey: "EN",
				Name:        "Test",
			},
			keyProperties: []metadata.PropertyMetadata{
				{Name: "ProductID", JsonName: "productId", Type: reflect.TypeOf(int(0)), GormTag: "primaryKey"},
				{Name: "LanguageKey", JsonName: "languageKey", Type: reflect.TypeOf(""), GormTag: "primaryKey"},
			},
			expectedID: 1,
		},
		{
			name: "Uint key with autoIncrement:false - should NOT be cleared",
			entity: &struct {
				ID   uint   `json:"id" gorm:"primaryKey;autoIncrement:false"`
				Name string `json:"name"`
			}{
				ID:   99,
				Name: "Test",
			},
			keyProperties: []metadata.PropertyMetadata{
				{Name: "ID", JsonName: "id", Type: reflect.TypeOf(uint(0)), GormTag: "primaryKey;autoIncrement:false"},
			},
			expectedID: uint(99),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &EntityHandler{
				metadata: &metadata.EntityMetadata{
					KeyProperties: tt.keyProperties,
				},
			}

			if err := handler.initializeEntityKeys(context.Background(), tt.entity); err != nil {
				t.Fatalf("initializeEntityKeys() error = %v", err)
			}

			// Check the ID field value
			entityValue := reflect.ValueOf(tt.entity).Elem()
			idField := entityValue.FieldByName("ID")
			if idField.IsValid() {
				if tt.expectedIDIsZero {
					if !idField.IsZero() {
						t.Errorf("Expected ID to be zero, got %v", idField.Interface())
					}
				} else if tt.expectedID != nil {
					if idField.Interface() != tt.expectedID {
						t.Errorf("Expected ID to be %v, got %v", tt.expectedID, idField.Interface())
					}
				}
			}

			// For composite keys, check ProductID
			productIDField := entityValue.FieldByName("ProductID")
			if productIDField.IsValid() && tt.expectedID != nil {
				if productIDField.Interface() != tt.expectedID {
					t.Errorf("Expected ProductID to be %v, got %v", tt.expectedID, productIDField.Interface())
				}
			}
		})
	}
}

func TestEntityHandlerInitializeEntityKeys_Generator(t *testing.T) {
	makeResolver := func(value interface{}, err error) func(string) (func(context.Context) (interface{}, error), bool) {
		return func(name string) (func(context.Context) (interface{}, error), bool) {
			if name != "custom" {
				return nil, false
			}
			return func(context.Context) (interface{}, error) {
				return value, err
			}, true
		}
	}

	t.Run("assign generated string", func(t *testing.T) {
		entity := &struct {
			ID string
		}{}

		handler := &EntityHandler{
			metadata: &metadata.EntityMetadata{
				KeyProperties: []metadata.PropertyMetadata{
					{Name: "ID", JsonName: "id", Type: reflect.TypeOf(""), IsKey: true, KeyGenerator: "custom"},
				},
			},
			keyGeneratorResolver: makeResolver("generated", nil),
		}

		if err := handler.initializeEntityKeys(context.Background(), entity); err != nil {
			t.Fatalf("initializeEntityKeys() error = %v", err)
		}

		if entity.ID != "generated" {
			t.Fatalf("expected generated ID to be 'generated', got %q", entity.ID)
		}
	})

	t.Run("assign pointer field", func(t *testing.T) {
		type identifier struct{ value string }
		entity := &struct {
			ID *identifier
		}{}

		generated := identifier{value: "abc"}

		handler := &EntityHandler{
			metadata: &metadata.EntityMetadata{
				KeyProperties: []metadata.PropertyMetadata{
					{Name: "ID", JsonName: "id", Type: reflect.TypeOf(&identifier{}), IsKey: true, KeyGenerator: "custom"},
				},
			},
			keyGeneratorResolver: makeResolver(generated, nil),
		}

		if err := handler.initializeEntityKeys(context.Background(), entity); err != nil {
			t.Fatalf("initializeEntityKeys() error = %v", err)
		}

		if entity.ID == nil || entity.ID.value != "abc" {
			t.Fatalf("expected generated pointer value, got %#v", entity.ID)
		}
	})

	t.Run("error when resolver missing", func(t *testing.T) {
		entity := &struct {
			ID string
		}{}

		handler := &EntityHandler{
			metadata: &metadata.EntityMetadata{
				KeyProperties: []metadata.PropertyMetadata{
					{Name: "ID", JsonName: "id", Type: reflect.TypeOf(""), IsKey: true, KeyGenerator: "custom"},
				},
			},
		}

		if err := handler.initializeEntityKeys(context.Background(), entity); err == nil {
			t.Fatal("expected error when resolver is missing")
		}
	})

	t.Run("error when generator returns nil for non-pointer", func(t *testing.T) {
		entity := &struct {
			ID string
		}{}

		handler := &EntityHandler{
			metadata: &metadata.EntityMetadata{
				KeyProperties: []metadata.PropertyMetadata{
					{Name: "ID", JsonName: "id", Type: reflect.TypeOf(""), IsKey: true, KeyGenerator: "custom"},
				},
			},
			keyGeneratorResolver: makeResolver(nil, nil),
		}

		if err := handler.initializeEntityKeys(context.Background(), entity); err == nil {
			t.Fatal("expected error when generator returned nil")
		}
	})
}
