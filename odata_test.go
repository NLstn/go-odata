package odata

import (
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
