package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// HandlerTestProduct is a test entity for entity handler tests
type HandlerTestProduct struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name" odata:"required"`
	Price       float64 `json:"Price"`
	Quantity    int     `json:"Quantity"`
	Category    string  `json:"Category"`
	IsAvailable bool    `json:"IsAvailable"`
}

func setupEntityTestHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&HandlerTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(HandlerTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	return handler, db
}

func TestNewEntityHandler(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(HandlerTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	t.Run("With nil logger", func(t *testing.T) {
		handler := NewEntityHandler(db, entityMeta, nil)
		if handler == nil {
			t.Fatal("Expected handler to be created")
		}
		if handler.db != db {
			t.Error("Expected handler.db to be set")
		}
		if handler.metadata != entityMeta {
			t.Error("Expected handler.metadata to be set")
		}
	})
}

func TestEntityHandler_SetFTSManager(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	// SetFTSManager accepts nil
	handler.SetFTSManager(nil)
	if handler.ftsManager != nil {
		t.Error("Expected ftsManager to be nil")
	}
}

func TestEntityHandler_SetNamespace(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	handler.SetNamespace("Custom.Namespace")
	if handler.namespace != "Custom.Namespace" {
		t.Errorf("Expected namespace to be 'Custom.Namespace', got %s", handler.namespace)
	}
}

func TestEntityHandler_IsSingleton(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	// Normal entity is not a singleton
	if handler.IsSingleton() {
		t.Error("Expected IsSingleton() to be false for normal entity")
	}
}

func TestHandleEntity_SelectProperties(t *testing.T) {
	handler, db := setupEntityTestHandler(t)

	// Create entity
	entity := HandlerTestProduct{ID: 1, Name: "Test", Price: 10.0, Quantity: 5, Category: "Electronics", IsAvailable: true}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/HandlerTestProducts(1)?$select=Name,Price", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have Name and Price (plus ID which is always included as key)
	if _, ok := response["Name"]; !ok {
		t.Error("Expected Name to be present")
	}
	if _, ok := response["Price"]; !ok {
		t.Error("Expected Price to be present")
	}
}

func TestHandleCollection_InvalidFilter(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/HandlerTestProducts?$filter=InvalidProperty%20eq%20%27test%27", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCollection_InvalidOrderBy(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/HandlerTestProducts?$orderby=InvalidProperty", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCollection_InvalidSelect(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/HandlerTestProducts?$select=InvalidProperty", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandleEntity_InvalidSelect(t *testing.T) {
	handler, db := setupEntityTestHandler(t)

	// Create entity
	entity := HandlerTestProduct{ID: 1, Name: "Test", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/HandlerTestProducts(1)?$select=InvalidProperty", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePatchEntity_EmptyBody(t *testing.T) {
	handler, db := setupEntityTestHandler(t)

	// Create entity
	entity := HandlerTestProduct{ID: 1, Name: "Test", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/HandlerTestProducts(1)", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	// Empty PATCH should succeed (no changes made)
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}
}

func TestHandlePatchEntity_InvalidJSON(t *testing.T) {
	handler, db := setupEntityTestHandler(t)

	// Create entity
	entity := HandlerTestProduct{ID: 1, Name: "Test", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/HandlerTestProducts(1)", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePatchEntity_UnknownProperty(t *testing.T) {
	handler, db := setupEntityTestHandler(t)

	// Create entity
	entity := HandlerTestProduct{ID: 1, Name: "Test", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	body := `{"UnknownProperty": "value"}`
	req := httptest.NewRequest(http.MethodPatch, "/HandlerTestProducts(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePutEntity_InvalidJSON(t *testing.T) {
	handler, db := setupEntityTestHandler(t)

	// Create entity
	entity := HandlerTestProduct{ID: 1, Name: "Test", Price: 10.0}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/HandlerTestProducts(1)", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePutEntity_NotFound(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	body := `{"ID": 999, "Name": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/HandlerTestProducts(999)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "999")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestHandlePostEntity_InvalidJSON(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/HandlerTestProducts", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePostEntity_UnknownProperty(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	// Unknown properties are typically ignored by OData servers, so this should succeed
	body := `{"Name": "Test", "UnknownProperty": "value"}`
	req := httptest.NewRequest(http.MethodPost, "/HandlerTestProducts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	// This might return 201 (ignoring unknown property) or 400 depending on configuration
	// The default behavior is to create the entity, ignoring unknown properties
	if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want 201 or 400", w.Code)
	}
}

func TestHandlePostEntity_WithPreferRepresentation(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	body := `{"Name": "Test Product", "Price": 99.99}`
	req := httptest.NewRequest(http.MethodPost, "/HandlerTestProducts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	// Check that representation is returned
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["Name"] != "Test Product" {
		t.Errorf("Name = %v, want 'Test Product'", response["Name"])
	}
}

func TestHandlePostEntity_WithPreferMinimal(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)

	body := `{"Name": "Test Product", "Price": 99.99}`
	req := httptest.NewRequest(http.MethodPost, "/HandlerTestProducts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}
}

func TestEntityHandler_InitPropertyMap(t *testing.T) {
	// Test that property map is initialized correctly
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(HandlerTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)

	// Verify property map was initialized
	if handler.propertyMap == nil {
		t.Fatal("Expected propertyMap to be initialized")
	}

	// Verify we can look up properties by name
	if _, ok := handler.propertyMap["Name"]; !ok {
		t.Error("Expected to find 'Name' property in map")
	}
	if _, ok := handler.propertyMap["Price"]; !ok {
		t.Error("Expected to find 'Price' property in map")
	}
}

func TestEntityHandler_EmptyMetadata(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Test with nil metadata - using direct instantiation to test edge cases
	// that would not normally occur via the constructor
	handler := &EntityHandler{
		db:       db,
		metadata: nil,
	}
	handler.initPropertyMap()

	// Should not panic and propertyMap should be nil
	if handler.propertyMap != nil {
		t.Error("Expected propertyMap to be nil for nil metadata")
	}

	// Test with empty properties - using direct instantiation for same reason
	handler = &EntityHandler{
		db: db,
		metadata: &metadata.EntityMetadata{
			Properties: []metadata.PropertyMetadata{},
		},
	}
	handler.initPropertyMap()

	// Should not panic
	if handler.propertyMap != nil {
		t.Error("Expected propertyMap to be nil for empty properties")
	}
}

func TestValidateValueType_VariousTypes(t *testing.T) {
	tests := []struct {
		name         string
		value        interface{}
		expectedType reflect.Type
		fieldName    string
		wantErr      bool
	}{
		{
			name:         "Float for int field",
			value:        float64(42),
			expectedType: reflect.TypeOf(int(0)),
			fieldName:    "count",
			wantErr:      false,
		},
		{
			name:         "Float for uint field",
			value:        float64(42),
			expectedType: reflect.TypeOf(uint(0)),
			fieldName:    "count",
			wantErr:      false,
		},
		{
			name:         "Float for float32 field",
			value:        float64(42.5),
			expectedType: reflect.TypeOf(float32(0)),
			fieldName:    "price",
			wantErr:      false,
		},
		{
			name:         "String for string field",
			value:        "test",
			expectedType: reflect.TypeOf(""),
			fieldName:    "name",
			wantErr:      false,
		},
		{
			name:         "Bool for bool field",
			value:        true,
			expectedType: reflect.TypeOf(false),
			fieldName:    "active",
			wantErr:      false,
		},
		{
			name:         "String for int field",
			value:        "not a number",
			expectedType: reflect.TypeOf(int(0)),
			fieldName:    "count",
			wantErr:      true,
		},
		{
			name:         "Bool for string field",
			value:        true,
			expectedType: reflect.TypeOf(""),
			fieldName:    "name",
			wantErr:      true,
		},
		{
			name:         "Slice for slice field",
			value:        []interface{}{"a", "b"},
			expectedType: reflect.TypeOf([]string{}),
			fieldName:    "tags",
			wantErr:      false,
		},
		{
			name:         "Map for struct field",
			value:        map[string]interface{}{"key": "value"},
			expectedType: reflect.TypeOf(struct{}{}),
			fieldName:    "nested",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateValueType(tt.value, tt.expectedType, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateValueType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateValueType_PointerTypes(t *testing.T) {
	// Test pointer types
	ptrType := reflect.TypeOf((*int)(nil))
	err := validateValueType(float64(42), ptrType, "ptrField")
	if err != nil {
		t.Errorf("validateValueType() should handle pointer types, got error: %v", err)
	}
}
