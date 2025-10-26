package odata

import (
	"net/http"
	"reflect"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Product struct {
	ID    int     `json:"id" gorm:"primarykey" odata:"key"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type InvalidEntity struct {
	Name string `json:"name"`
	// No key field
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)

	service := NewService(db)

	if service == nil {
		t.Fatal("NewService() returned nil")
	}

	if service.db == nil {
		t.Error("Service.db is nil")
	}

	if service.entities == nil {
		t.Error("Service.entities is nil")
	}

	if service.handlers == nil {
		t.Error("Service.handlers is nil")
	}

	if service.metadataHandler == nil {
		t.Error("Service.metadataHandler is nil")
	}

	if service.serviceDocumentHandler == nil {
		t.Error("Service.serviceDocumentHandler is nil")
	}
}

func TestServiceRegisterEntity(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Test successful registration
	err := service.RegisterEntity(Product{})
	if err != nil {
		t.Errorf("RegisterEntity() error = %v, want nil", err)
	}

	// Check that entity metadata was stored
	if _, exists := service.entities["Products"]; !exists {
		t.Error("Entity metadata not stored after registration")
	}

	// Check that handler was created
	if _, exists := service.handlers["Products"]; !exists {
		t.Error("Handler not created after registration")
	}

	// Test registration with invalid entity
	err = service.RegisterEntity(InvalidEntity{})
	if err == nil {
		t.Error("RegisterEntity() with invalid entity should return error")
	}

	// Test registration with pointer
	err = service.RegisterEntity(&Product{})
	if err != nil {
		t.Errorf("RegisterEntity() with pointer error = %v, want nil", err)
	}
}

func TestServiceRegisterMultipleEntities(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	type Category struct {
		ID   int    `json:"id" odata:"key"`
		Name string `json:"name"`
	}

	// Register multiple entities
	if err := service.RegisterEntity(Product{}); err != nil {
		t.Errorf("RegisterEntity(Product) error = %v", err)
	}

	if err := service.RegisterEntity(Category{}); err != nil {
		t.Errorf("RegisterEntity(Category) error = %v", err)
	}

	// Verify both are registered
	if len(service.entities) != 2 {
		t.Errorf("Number of registered entities = %v, want 2", len(service.entities))
	}

	if len(service.handlers) != 2 {
		t.Errorf("Number of handlers = %v, want 2", len(service.handlers))
	}

	// Verify entity sets exist
	expectedSets := map[string]bool{
		"Products":   true,
		"Categories": true,
	}

	for setName := range expectedSets {
		if _, exists := service.entities[setName]; !exists {
			t.Errorf("Entity set %s not found", setName)
		}
		if _, exists := service.handlers[setName]; !exists {
			t.Errorf("Handler for %s not found", setName)
		}
	}
}

func TestRegisterActionValidation(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(Product{}); err != nil {
		t.Fatalf("failed to register product entity: %v", err)
	}

	dummyHandler := func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) error {
		return nil
	}

	tests := []struct {
		name        string
		action      ActionDefinition
		expectedErr string
	}{
		{
			name: "empty name",
			action: ActionDefinition{
				Handler: dummyHandler,
			},
			expectedErr: "action name cannot be empty",
		},
		{
			name: "nil handler",
			action: ActionDefinition{
				Name: "TestAction",
			},
			expectedErr: "action handler cannot be nil",
		},
		{
			name: "bound action missing entity set",
			action: ActionDefinition{
				Name:    "BoundAction",
				IsBound: true,
				Handler: dummyHandler,
			},
			expectedErr: "bound action must specify entity set",
		},
		{
			name: "bound action unregistered entity set",
			action: ActionDefinition{
				Name:      "BoundAction",
				IsBound:   true,
				EntitySet: "UnknownSet",
				Handler:   dummyHandler,
			},
			expectedErr: "entity set 'UnknownSet' not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := service.RegisterAction(tc.action)
			if err == nil {
				t.Fatalf("RegisterAction() error = nil, want %q", tc.expectedErr)
			}
			if err.Error() != tc.expectedErr {
				t.Fatalf("RegisterAction() error = %q, want %q", err.Error(), tc.expectedErr)
			}
		})
	}
}

func TestRegisterFunctionValidation(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	if err := service.RegisterEntity(Product{}); err != nil {
		t.Fatalf("failed to register product entity: %v", err)
	}

	dummyHandler := func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
		return nil, nil
	}

	returnType := reflect.TypeOf("")

	tests := []struct {
		name        string
		function    FunctionDefinition
		expectedErr string
	}{
		{
			name: "empty name",
			function: FunctionDefinition{
				Handler:    dummyHandler,
				ReturnType: returnType,
			},
			expectedErr: "function name cannot be empty",
		},
		{
			name: "nil handler",
			function: FunctionDefinition{
				Name:       "TestFunction",
				ReturnType: returnType,
			},
			expectedErr: "function handler cannot be nil",
		},
		{
			name: "nil return type",
			function: FunctionDefinition{
				Name:    "TestFunction",
				Handler: dummyHandler,
			},
			expectedErr: "function must have a return type",
		},
		{
			name: "bound function missing entity set",
			function: FunctionDefinition{
				Name:       "BoundFunction",
				IsBound:    true,
				Handler:    dummyHandler,
				ReturnType: returnType,
			},
			expectedErr: "bound function must specify entity set",
		},
		{
			name: "bound function unregistered entity set",
			function: FunctionDefinition{
				Name:       "BoundFunction",
				IsBound:    true,
				EntitySet:  "UnknownSet",
				Handler:    dummyHandler,
				ReturnType: returnType,
			},
			expectedErr: "entity set 'UnknownSet' not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := service.RegisterFunction(tc.function)
			if err == nil {
				t.Fatalf("RegisterFunction() error = nil, want %q", tc.expectedErr)
			}
			if err.Error() != tc.expectedErr {
				t.Fatalf("RegisterFunction() error = %q, want %q", err.Error(), tc.expectedErr)
			}
		})
	}
}
