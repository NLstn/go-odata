package odata

import (
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestRegisterEntityAnnotation verifies entity annotation registration
func TestRegisterEntityAnnotation(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register entity
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test registering entity annotation
	err = service.RegisterEntityAnnotation("Products", "Org.OData.Core.V1.Description", "Product catalog items")
	if err != nil {
		t.Fatalf("Failed to register entity annotation: %v", err)
	}

	// Verify annotation appears in metadata
	req := httptest.NewRequest("GET", "/$metadata", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Description") {
		t.Error("Expected annotation to appear in metadata")
	}

	// Test with non-existent entity set
	err = service.RegisterEntityAnnotation("NonExistent", "Org.OData.Core.V1.Description", "test")
	if err == nil {
		t.Error("Expected error for non-existent entity set")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("Expected 'not registered' error, got: %v", err)
	}

	// Test with annotation with qualifier
	err = service.RegisterEntityAnnotation("Products", "Org.OData.Core.V1.Description#Summary", "Short description")
	if err != nil {
		t.Fatalf("Failed to register annotation with qualifier: %v", err)
	}

	// Test with computed annotation
	err = service.RegisterEntityAnnotation("Products", "Org.OData.Core.V1.OptimisticConcurrency", []string{"ID"})
	if err != nil {
		t.Fatalf("Failed to register optimistic concurrency annotation: %v", err)
	}
}

// TestRegisterEntitySetAnnotation verifies entity set annotation registration
func TestRegisterEntitySetAnnotation(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register entity
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test registering entity set annotation
	err = service.RegisterEntitySetAnnotation("Products", "Org.OData.Capabilities.V1.DeleteRestrictions",
		map[string]interface{}{"Deletable": false})
	if err != nil {
		t.Fatalf("Failed to register entity set annotation: %v", err)
	}

	// Test with non-existent entity set
	err = service.RegisterEntitySetAnnotation("NonExistent", "Org.OData.Core.V1.Description", "test")
	if err == nil {
		t.Error("Expected error for non-existent entity set")
	}

	// Test with insert restrictions
	err = service.RegisterEntitySetAnnotation("Products", "Org.OData.Capabilities.V1.InsertRestrictions",
		map[string]interface{}{"Insertable": true})
	if err != nil {
		t.Fatalf("Failed to register insert restrictions: %v", err)
	}
}

// TestRegisterEntitySetAnnotationOnSingleton verifies error for singleton
func TestRegisterEntitySetAnnotationOnSingleton(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register as singleton
	if err := service.RegisterSingleton(&Product{}, "CurrentProduct"); err != nil {
		t.Fatalf("Failed to register singleton: %v", err)
	}

	// Try to register entity set annotation on singleton
	err = service.RegisterEntitySetAnnotation("CurrentProduct", "Org.OData.Core.V1.Description", "test")
	if err == nil {
		t.Error("Expected error when registering entity set annotation on singleton")
	}
	if !strings.Contains(err.Error(), "singleton") {
		t.Errorf("Expected error mentioning 'singleton', got: %v", err)
	}
}

// TestRegisterSingletonAnnotation verifies singleton annotation registration
func TestRegisterSingletonAnnotation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register as singleton
	if err := service.RegisterSingleton(&Product{}, "CurrentProduct"); err != nil {
		t.Fatalf("Failed to register singleton: %v", err)
	}

	// Test registering singleton annotation
	err = service.RegisterSingletonAnnotation("CurrentProduct", "Org.OData.Core.V1.Description", "Current product information")
	if err != nil {
		t.Fatalf("Failed to register singleton annotation: %v", err)
	}

	// Test with non-existent singleton
	err = service.RegisterSingletonAnnotation("NonExistent", "Org.OData.Core.V1.Description", "test")
	if err == nil {
		t.Error("Expected error for non-existent singleton")
	}

	// Register a regular entity
	if err := service.RegisterEntity(&Article{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Try to register singleton annotation on regular entity
	err = service.RegisterSingletonAnnotation("Articles", "Org.OData.Core.V1.Description", "test")
	if err == nil {
		t.Error("Expected error when registering singleton annotation on regular entity")
	}
	if !strings.Contains(err.Error(), "not a singleton") {
		t.Errorf("Expected error mentioning 'not a singleton', got: %v", err)
	}
}

// TestRegisterEntityContainerAnnotation verifies entity container annotation registration
func TestRegisterEntityContainerAnnotation(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test registering entity container annotation
	err = service.RegisterEntityContainerAnnotation("Org.OData.Core.V1.Description", "Primary service container")
	if err != nil {
		t.Fatalf("Failed to register entity container annotation: %v", err)
	}

	// Register multiple annotations
	err = service.RegisterEntityContainerAnnotation("Org.OData.Capabilities.V1.ConformanceLevel", "Advanced")
	if err != nil {
		t.Fatalf("Failed to register conformance level: %v", err)
	}

	err = service.RegisterEntityContainerAnnotation("Org.OData.Capabilities.V1.SupportedFormats", []string{"application/json", "application/xml"})
	if err != nil {
		t.Fatalf("Failed to register supported formats: %v", err)
	}

	// Register entity and check metadata
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	req := httptest.NewRequest("GET", "/$metadata", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

// TestRegisterPropertyAnnotation verifies property annotation registration
func TestRegisterPropertyAnnotation(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register entity
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test registering property annotation (using Name property)
	err = service.RegisterPropertyAnnotation("Products", "Name", "Org.OData.Core.V1.Description", "The product display name")
	if err != nil {
		t.Fatalf("Failed to register property annotation: %v", err)
	}

	// Test registering computed annotation
	err = service.RegisterPropertyAnnotation("Products", "ID", "Org.OData.Core.V1.Computed", true)
	if err != nil {
		t.Fatalf("Failed to register computed annotation: %v", err)
	}

	// Test registering immutable annotation
	err = service.RegisterPropertyAnnotation("Products", "ID", "Org.OData.Core.V1.Immutable", true)
	if err != nil {
		t.Fatalf("Failed to register immutable annotation: %v", err)
	}

	// Test with annotation with qualifier
	err = service.RegisterPropertyAnnotation("Products", "Name", "Org.OData.Core.V1.Description#Long", "Full product description")
	if err != nil {
		t.Fatalf("Failed to register property annotation with qualifier: %v", err)
	}

	// Test with non-existent entity set
	err = service.RegisterPropertyAnnotation("NonExistent", "Name", "Org.OData.Core.V1.Description", "test")
	if err == nil {
		t.Error("Expected error for non-existent entity set")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("Expected 'not registered' error, got: %v", err)
	}

	// Test with non-existent property
	err = service.RegisterPropertyAnnotation("Products", "NonExistentProperty", "Org.OData.Core.V1.Description", "test")
	if err == nil {
		t.Error("Expected error for non-existent property")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	// Verify annotations appear in metadata
	req := httptest.NewRequest("GET", "/$metadata", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

// TestMultipleAnnotationRegistrations verifies multiple annotations can be registered
func TestMultipleAnnotationRegistrations(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register entity
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register multiple annotations on entity
	annotations := []struct {
		term  string
		value interface{}
	}{
		{"Org.OData.Core.V1.Description", "Products in the catalog"},
		{"Org.OData.Core.V1.LongDescription", "All products available for purchase"},
		{"Org.OData.Capabilities.V1.SearchRestrictions", map[string]interface{}{"Searchable": true}},
	}

	for _, ann := range annotations {
		err = service.RegisterEntityAnnotation("Products", ann.term, ann.value)
		if err != nil {
			t.Fatalf("Failed to register annotation %s: %v", ann.term, err)
		}
	}

	// Register multiple property annotations
	propertyAnnotations := []struct {
		property string
		term     string
		value    interface{}
	}{
		{"Name", "Org.OData.Core.V1.Description", "Product name"},
		{"Price", "Org.OData.Core.V1.Description", "Product price"},
		{"ID", "Org.OData.Core.V1.Computed", true},
	}

	for _, ann := range propertyAnnotations {
		err = service.RegisterPropertyAnnotation("Products", ann.property, ann.term, ann.value)
		if err != nil {
			t.Fatalf("Failed to register property annotation %s on %s: %v", ann.term, ann.property, err)
		}
	}

	// Verify metadata is valid
	req := httptest.NewRequest("GET", "/$metadata", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}
