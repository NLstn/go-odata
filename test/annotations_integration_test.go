package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// AnnotatedProduct is a test entity with annotation tags
type AnnotatedProduct struct {
	ID        uint    `json:"ID" gorm:"primaryKey"`
	Name      string  `json:"Name" odata:"required,annotation:Core.Description=Product display name"`
	Price     float64 `json:"Price"`
	CreatedAt string  `json:"CreatedAt" odata:"auto"`
}

// ComputedFieldEntity tests auto-detection of annotations from property flags
type ComputedFieldEntity struct {
	ID        uint   `json:"ID" gorm:"primaryKey;autoIncrement"` // Database-generated key
	Name      string `json:"Name"`
	UpdatedAt string `json:"UpdatedAt" odata:"auto"` // Auto property should have Computed annotation
}

func TestAnnotations_PropertyAnnotationFromTag(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&AnnotatedProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&AnnotatedProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test JSON metadata
	t.Run("JSON metadata contains annotations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		body := w.Body.String()

		// Should contain the Description annotation on Name property
		if !strings.Contains(body, "@Org.OData.Core.V1.Description") {
			t.Error("JSON metadata should contain @Org.OData.Core.V1.Description annotation")
		}

		// Should contain the Computed annotation on CreatedAt property
		if !strings.Contains(body, "@Org.OData.Core.V1.Computed") {
			t.Error("JSON metadata should contain @Org.OData.Core.V1.Computed annotation")
		}
	})

	// Test XML metadata
	t.Run("XML metadata contains annotations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/xml")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		body := w.Body.String()

		// Should contain Annotations element
		if !strings.Contains(body, "<Annotations Target=") {
			t.Error("XML metadata should contain <Annotations> element")
		}

		// Should contain Annotation element with Core.Computed term
		if !strings.Contains(body, "Org.OData.Core.V1.Computed") {
			t.Error("XML metadata should contain Org.OData.Core.V1.Computed annotation")
		}

		// Should contain vocabulary reference
		if !strings.Contains(body, "edmx:Reference") {
			t.Error("XML metadata should contain edmx:Reference element for vocabulary")
		}
	})
}

func TestAnnotations_AutoDetectedFromPropertyFlags(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ComputedFieldEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&ComputedFieldEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	t.Run("Auto property has Computed annotation in JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &metadata); err != nil {
			t.Fatalf("Failed to parse JSON metadata: %v", err)
		}

		// Navigate to the entity type
		namespace, ok := metadata["ODataService"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing ODataService namespace in metadata")
		}

		entityType, ok := namespace["ComputedFieldEntity"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing ComputedFieldEntity in metadata")
		}

		// Check UpdatedAt property for Computed annotation
		updatedAt, ok := entityType["UpdatedAt"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing UpdatedAt property in entity type")
		}

		if _, hasComputed := updatedAt["@Org.OData.Core.V1.Computed"]; !hasComputed {
			t.Error("UpdatedAt property should have @Org.OData.Core.V1.Computed annotation auto-detected from odata:\"auto\"")
		}
	})
}

func TestAnnotations_RegisterEntityAnnotation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Simple entity without annotation tags
	type SimpleEntity struct {
		ID   uint   `json:"ID" gorm:"primaryKey"`
		Name string `json:"Name"`
	}

	if err := db.AutoMigrate(&SimpleEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&SimpleEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register entity-level annotation via API
	err = service.RegisterEntityAnnotation("SimpleEntities",
		"Org.OData.Core.V1.Description",
		"A simple test entity")
	if err != nil {
		t.Fatalf("Failed to register entity annotation: %v", err)
	}

	t.Run("Entity annotation appears in JSON metadata", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "@Org.OData.Core.V1.Description") {
			t.Error("JSON metadata should contain the registered entity annotation")
		}
		if !strings.Contains(body, "A simple test entity") {
			t.Error("JSON metadata should contain the annotation value")
		}
	})
}

func TestAnnotations_RegisterPropertyAnnotation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Simple entity without annotation tags
	type PropertyAnnotationEntity struct {
		ID       uint   `json:"ID" gorm:"primaryKey"`
		Email    string `json:"Email"`
		Password string `json:"Password"`
	}

	if err := db.AutoMigrate(&PropertyAnnotationEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&PropertyAnnotationEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register property-level annotation via API
	err = service.RegisterPropertyAnnotation("PropertyAnnotationEntities", "Email",
		"Org.OData.Core.V1.Description",
		"User email address")
	if err != nil {
		t.Fatalf("Failed to register property annotation: %v", err)
	}

	// Register Permissions annotation to Password (write-only)
	err = service.RegisterPropertyAnnotation("PropertyAnnotationEntities", "Password",
		"Org.OData.Core.V1.Permissions",
		"Write")
	if err != nil {
		t.Fatalf("Failed to register property annotation: %v", err)
	}

	t.Run("Property annotation appears in JSON metadata", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &metadata); err != nil {
			t.Fatalf("Failed to parse JSON metadata: %v", err)
		}

		namespace, ok := metadata["ODataService"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing ODataService namespace in metadata")
		}

		entityType, ok := namespace["PropertyAnnotationEntity"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing PropertyAnnotationEntity in metadata")
		}

		// Check Email property for Description annotation
		email, ok := entityType["Email"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing Email property in entity type")
		}

		if desc, ok := email["@Org.OData.Core.V1.Description"]; !ok {
			t.Error("Email property should have @Org.OData.Core.V1.Description annotation")
		} else if desc != "User email address" {
			t.Errorf("Email annotation value = %v, want 'User email address'", desc)
		}

		// Check Password property for Permissions annotation
		password, ok := entityType["Password"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing Password property in entity type")
		}

		if perm, ok := password["@Org.OData.Core.V1.Permissions"]; !ok {
			t.Error("Password property should have @Org.OData.Core.V1.Permissions annotation")
		} else if perm != "Write" {
			t.Errorf("Password annotation value = %v, want 'Write'", perm)
		}
	})
}

func TestAnnotations_RegisterAnnotation_InvalidEntitySet(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Try to register annotation on non-existent entity
	err = service.RegisterEntityAnnotation("NonExistentEntities",
		"Org.OData.Core.V1.Description",
		"Test")
	if err == nil {
		t.Error("Expected error when registering annotation on non-existent entity")
	}
}

func TestAnnotations_RegisterPropertyAnnotation_InvalidProperty(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	type TestEntity struct {
		ID   uint   `json:"ID" gorm:"primaryKey"`
		Name string `json:"Name"`
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&TestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Try to register annotation on non-existent property
	err = service.RegisterPropertyAnnotation("TestEntities", "NonExistentProperty",
		"Org.OData.Core.V1.Description",
		"Test")
	if err == nil {
		t.Error("Expected error when registering annotation on non-existent property")
	}
}
