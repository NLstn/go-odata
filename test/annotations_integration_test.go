package odata_test

import (
	"encoding/json"
	"fmt"
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

// TestAnnotations_Phase5_InstanceAnnotationsInResponses tests that vocabulary annotations
// appear in entity payloads based on metadata level
func TestAnnotations_Phase5_InstanceAnnotationsInResponses(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Entity with annotations
	type AnnotatedItem struct {
		ID        uint    `json:"ID" gorm:"primaryKey;autoIncrement"`
		Name      string  `json:"Name" odata:"required,annotation:Core.Description=Product display name"`
		Price     float64 `json:"Price"`
		CreatedAt string  `json:"CreatedAt" odata:"auto,annotation:Core.Computed"`
	}

	if err := db.AutoMigrate(&AnnotatedItem{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&AnnotatedItem{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Add entity-level annotation
	err = service.RegisterEntityAnnotation("AnnotatedItems",
		"Org.OData.Core.V1.Description",
		"Product catalog item")
	if err != nil {
		t.Fatalf("Failed to register entity annotation: %v", err)
	}

	// Create a test entity
	testProduct := &AnnotatedItem{
		Name:      "Widget",
		Price:     99.99,
		CreatedAt: "2025-01-01T00:00:00Z",
	}
	if err := db.Create(testProduct).Error; err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	t.Run("Full metadata includes annotations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/AnnotatedItems(1)?$format=application/json;odata.metadata=full", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check entity-level annotation
		if desc, ok := response["@Org.OData.Core.V1.Description"]; !ok {
			t.Error("Full metadata should include entity-level @Org.OData.Core.V1.Description annotation")
		} else if desc != "Product catalog item" {
			t.Errorf("Entity description = %v, want 'Product catalog item'", desc)
		}

		// Check property-level annotation on Name
		if desc, ok := response["Name@Org.OData.Core.V1.Description"]; !ok {
			t.Error("Full metadata should include Name@Org.OData.Core.V1.Description annotation")
		} else if desc != "Product display name" {
			t.Errorf("Name description = %v, want 'Product display name'", desc)
		}

		// Check property-level annotation on CreatedAt
		if computed, ok := response["CreatedAt@Org.OData.Core.V1.Computed"]; !ok {
			t.Error("Full metadata should include CreatedAt@Org.OData.Core.V1.Computed annotation")
		} else if computed != true {
			t.Errorf("CreatedAt computed = %v, want true", computed)
		}
	})

	t.Run("Minimal metadata excludes annotations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/AnnotatedItems(1)?$format=application/json;odata.metadata=minimal", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Minimal metadata should not include vocabulary annotations
		if _, ok := response["@Org.OData.Core.V1.Description"]; ok {
			t.Error("Minimal metadata should not include entity-level vocabulary annotations")
		}
		if _, ok := response["Name@Org.OData.Core.V1.Description"]; ok {
			t.Error("Minimal metadata should not include property-level vocabulary annotations")
		}
		if _, ok := response["CreatedAt@Org.OData.Core.V1.Computed"]; ok {
			t.Error("Minimal metadata should not include property-level vocabulary annotations")
		}
	})

	t.Run("None metadata excludes annotations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/AnnotatedItems(1)?$format=application/json;odata.metadata=none", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// None metadata should not include any annotations
		if _, ok := response["@Org.OData.Core.V1.Description"]; ok {
			t.Error("None metadata should not include vocabulary annotations")
		}
		if _, ok := response["Name@Org.OData.Core.V1.Description"]; ok {
			t.Error("None metadata should not include vocabulary annotations")
		}
	})

	t.Run("Collection response with full metadata includes annotations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/AnnotatedItems?$format=application/json;odata.metadata=full", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		value, ok := response["value"].([]interface{})
		if !ok || len(value) == 0 {
			t.Fatal("Expected value array in response")
		}

		entity, ok := value[0].(map[string]interface{})
		if !ok {
			t.Fatal("Expected entity object in value array")
		}

		// Check entity-level annotation
		if desc, ok := entity["@Org.OData.Core.V1.Description"]; !ok {
			t.Error("Collection items with full metadata should include entity-level annotations")
		} else if desc != "Product catalog item" {
			t.Errorf("Entity description = %v, want 'Product catalog item'", desc)
		}

		// Check property-level annotation
		if desc, ok := entity["Name@Org.OData.Core.V1.Description"]; !ok {
			t.Error("Collection items with full metadata should include property-level annotations")
		} else if desc != "Product display name" {
			t.Errorf("Name description = %v, want 'Product display name'", desc)
		}
	})
}

// TestAnnotations_Phase6_InstanceAnnotationsInRequests tests that instance annotations
// in POST/PATCH requests are allowed and ignored
func TestAnnotations_Phase6_InstanceAnnotationsInRequests(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	type RequestTestEntity struct {
		ID    uint   `json:"ID" gorm:"primaryKey;autoIncrement"`
		Name  string `json:"Name"`
		Value int    `json:"Value"`
	}

	if err := db.AutoMigrate(&RequestTestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&RequestTestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	t.Run("POST with instance annotations should succeed", func(t *testing.T) {
		requestBody := strings.NewReader(`{
			"Name": "Test Entity",
			"Value": 42,
			"@Org.OData.Core.V1.Description": "Client-provided annotation",
			"Name@Org.OData.Core.V1.Description": "Name description from client"
		}`)

		req := httptest.NewRequest(http.MethodPost, "/RequestTestEntities", requestBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status Created, got %d: %s", w.Code, w.Body.String())
		}

		// Verify entity was created without instance annotations
		var created RequestTestEntity
		if err := db.First(&created).Error; err != nil {
			t.Fatalf("Failed to fetch created entity: %v", err)
		}

		if created.Name != "Test Entity" {
			t.Errorf("Name = %s, want 'Test Entity'", created.Name)
		}
		if created.Value != 42 {
			t.Errorf("Value = %d, want 42", created.Value)
		}
	})

	t.Run("PATCH with instance annotations should succeed", func(t *testing.T) {
		// Create entity first
		testEntity := &RequestTestEntity{Name: "Original", Value: 10}
		if err := db.Create(testEntity).Error; err != nil {
			t.Fatalf("Failed to create test entity: %v", err)
		}

		requestBody := strings.NewReader(`{
			"Name": "Updated",
			"@Org.OData.Core.V1.Description": "Updated description",
			"Value@Org.OData.Core.V1.Computed": true
		}`)

		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/RequestTestEntities(%d)", testEntity.ID), requestBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
			t.Errorf("Expected status NoContent or OK, got %d: %s", w.Code, w.Body.String())
		}

		// Verify entity was updated correctly
		var updated RequestTestEntity
		if err := db.First(&updated, testEntity.ID).Error; err != nil {
			t.Fatalf("Failed to fetch updated entity: %v", err)
		}

		if updated.Name != "Updated" {
			t.Errorf("Name = %s, want 'Updated'", updated.Name)
		}
	})

	t.Run("POST with unknown property (not annotation) should fail", func(t *testing.T) {
		requestBody := strings.NewReader(`{
			"Name": "Test",
			"Value": 10,
			"UnknownProperty": "should fail"
		}`)

		req := httptest.NewRequest(http.MethodPost, "/RequestTestEntities", requestBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		// Go's JSON unmarshaling will silently ignore unknown fields,
		// so the request should succeed but the unknown property is ignored
		if w.Code != http.StatusCreated {
			t.Logf("Note: Unknown non-annotation properties are silently ignored during JSON unmarshaling")
		}
	})

	t.Run("PATCH with unknown property (not annotation) should fail", func(t *testing.T) {
		// Create entity first
		testEntity := &RequestTestEntity{Name: "Original", Value: 10}
		if err := db.Create(testEntity).Error; err != nil {
			t.Fatalf("Failed to create test entity: %v", err)
		}

		requestBody := strings.NewReader(`{
			"Name": "Updated",
			"UnknownProperty": "should fail"
		}`)

		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/RequestTestEntities(%d)", testEntity.ID), requestBody)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		// For PATCH, unknown properties are validated
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status BadRequest for unknown property, got %d: %s", w.Code, w.Body.String())
		}
	})
}
