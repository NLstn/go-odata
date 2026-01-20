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

// EntityWithCollectionAnnotation tests collection annotation values
type EntityWithCollectionAnnotation struct {
	ID   uint   `json:"ID" gorm:"primaryKey"`
	Name string `json:"Name"`
}

// EntityWithRecordAnnotation tests record annotation values
type EntityWithRecordAnnotation struct {
	ID   uint   `json:"ID" gorm:"primaryKey"`
	Name string `json:"Name"`
}

// EntityWithNestedAnnotation tests nested collection/record annotation values
type EntityWithNestedAnnotation struct {
	ID   uint   `json:"ID" gorm:"primaryKey"`
	Name string `json:"Name"`
}

func TestAnnotations_CollectionValues_JSON(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityWithCollectionAnnotation{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&EntityWithCollectionAnnotation{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register collection annotation (e.g., OptimisticConcurrency)
	err = service.RegisterEntityAnnotation("EntityWithCollectionAnnotations",
		"Org.OData.Core.V1.OptimisticConcurrency",
		[]string{"Name"})
	if err != nil {
		t.Fatalf("Failed to register collection annotation: %v", err)
	}

	t.Run("Collection annotation in JSON metadata", func(t *testing.T) {
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

		entityType, ok := namespace["EntityWithCollectionAnnotation"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing EntityWithCollectionAnnotation in metadata")
		}

		// Check for collection annotation
		annotation, ok := entityType["@Org.OData.Core.V1.OptimisticConcurrency"]
		if !ok {
			t.Fatal("Missing @Org.OData.Core.V1.OptimisticConcurrency annotation")
		}

		// Should be wrapped in $Collection
		collectionWrapper, ok := annotation.(map[string]interface{})
		if !ok {
			t.Fatal("Collection annotation should be wrapped in a map")
		}

		collection, ok := collectionWrapper["$Collection"].([]interface{})
		if !ok {
			t.Fatal("Collection annotation should have $Collection key with array value")
		}

		if len(collection) != 1 {
			t.Errorf("Collection should have 1 item, got %d", len(collection))
		}

		if collection[0] != "Name" {
			t.Errorf("Collection item = %v, want 'Name'", collection[0])
		}
	})
}

func TestAnnotations_CollectionValues_XML(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityWithCollectionAnnotation{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&EntityWithCollectionAnnotation{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register collection annotation
	err = service.RegisterEntityAnnotation("EntityWithCollectionAnnotations",
		"Org.OData.Core.V1.OptimisticConcurrency",
		[]string{"Name"})
	if err != nil {
		t.Fatalf("Failed to register collection annotation: %v", err)
	}

	t.Run("Collection annotation in XML metadata", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/xml")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		body := w.Body.String()

		// Should contain Annotation element with Collection
		if !strings.Contains(body, "<Annotation Term=\"Org.OData.Core.V1.OptimisticConcurrency\">") {
			t.Error("XML metadata should contain Annotation element with OptimisticConcurrency term")
		}

		if !strings.Contains(body, "<Collection>") {
			t.Error("XML metadata should contain <Collection> element")
		}

		if !strings.Contains(body, "</Collection>") {
			t.Error("XML metadata should contain closing </Collection> element")
		}

		// Should contain the string element inside collection
		if !strings.Contains(body, "<String>Name</String>") {
			t.Error("XML metadata should contain <String>Name</String> element inside Collection")
		}
	})
}

func TestAnnotations_RecordValues_JSON(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityWithRecordAnnotation{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&EntityWithRecordAnnotation{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register record annotation
	recordValue := map[string]interface{}{
		"LongDescription": "A detailed description",
		"ChangedAt":       "2025-01-01T00:00:00Z",
	}
	err = service.RegisterEntityAnnotation("EntityWithRecordAnnotations",
		"Org.OData.Core.V1.Example",
		recordValue)
	if err != nil {
		t.Fatalf("Failed to register record annotation: %v", err)
	}

	t.Run("Record annotation in JSON metadata", func(t *testing.T) {
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

		entityType, ok := namespace["EntityWithRecordAnnotation"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing EntityWithRecordAnnotation in metadata")
		}

		// Check for record annotation
		annotation, ok := entityType["@Org.OData.Core.V1.Example"]
		if !ok {
			t.Fatal("Missing @Org.OData.Core.V1.Example annotation")
		}

		// Should be wrapped in $Record
		recordWrapper, ok := annotation.(map[string]interface{})
		if !ok {
			t.Fatal("Record annotation should be wrapped in a map")
		}

		record, ok := recordWrapper["$Record"].(map[string]interface{})
		if !ok {
			t.Fatal("Record annotation should have $Record key with map value")
		}

		if len(record) != 2 {
			t.Errorf("Record should have 2 properties, got %d", len(record))
		}

		if record["LongDescription"] != "A detailed description" {
			t.Errorf("Record LongDescription = %v, want 'A detailed description'", record["LongDescription"])
		}

		if record["ChangedAt"] != "2025-01-01T00:00:00Z" {
			t.Errorf("Record ChangedAt = %v, want '2025-01-01T00:00:00Z'", record["ChangedAt"])
		}
	})
}

func TestAnnotations_RecordValues_XML(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityWithRecordAnnotation{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&EntityWithRecordAnnotation{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register record annotation
	recordValue := map[string]interface{}{
		"LongDescription": "A detailed description",
		"ChangedAt":       "2025-01-01T00:00:00Z",
	}
	err = service.RegisterEntityAnnotation("EntityWithRecordAnnotations",
		"Org.OData.Core.V1.Example",
		recordValue)
	if err != nil {
		t.Fatalf("Failed to register record annotation: %v", err)
	}

	t.Run("Record annotation in XML metadata", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/xml")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		body := w.Body.String()

		// Should contain Annotation element with Record
		if !strings.Contains(body, "<Annotation Term=\"Org.OData.Core.V1.Example\">") {
			t.Error("XML metadata should contain Annotation element with Example term")
		}

		if !strings.Contains(body, "<Record>") {
			t.Error("XML metadata should contain <Record> element")
		}

		if !strings.Contains(body, "</Record>") {
			t.Error("XML metadata should contain closing </Record> element")
		}

		// Should contain PropertyValue elements for record properties
		if !strings.Contains(body, "<PropertyValue Property=\"ChangedAt\" String=\"2025-01-01T00:00:00Z\" />") &&
			!strings.Contains(body, "<PropertyValue Property=\"LongDescription\" String=\"A detailed description\" />") {
			t.Error("XML metadata should contain PropertyValue elements for record properties")
		}
	})
}

func TestAnnotations_NestedCollectionAndRecord_JSON(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityWithNestedAnnotation{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&EntityWithNestedAnnotation{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register nested annotation: collection of records
	nestedValue := []map[string]interface{}{
		{
			"PropertyPath": "Name",
			"Filterable":   true,
		},
		{
			"PropertyPath": "ID",
			"Filterable":   false,
		},
	}
	err = service.RegisterEntityAnnotation("EntityWithNestedAnnotations",
		"Org.OData.Capabilities.V1.FilterRestrictions",
		nestedValue)
	if err != nil {
		t.Fatalf("Failed to register nested annotation: %v", err)
	}

	t.Run("Nested collection of records in JSON metadata", func(t *testing.T) {
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

		entityType, ok := namespace["EntityWithNestedAnnotation"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing EntityWithNestedAnnotation in metadata")
		}

		// Check for nested annotation
		annotation, ok := entityType["@Org.OData.Capabilities.V1.FilterRestrictions"]
		if !ok {
			t.Fatal("Missing @Org.OData.Capabilities.V1.FilterRestrictions annotation")
		}

		// Should be wrapped in $Collection
		collectionWrapper, ok := annotation.(map[string]interface{})
		if !ok {
			t.Fatal("Nested annotation should be wrapped in a map")
		}

		collection, ok := collectionWrapper["$Collection"].([]interface{})
		if !ok {
			t.Fatal("Nested annotation should have $Collection key with array value")
		}

		if len(collection) != 2 {
			t.Errorf("Collection should have 2 items, got %d", len(collection))
		}

		// First item should be a record
		firstItem, ok := collection[0].(map[string]interface{})
		if !ok {
			t.Fatal("Collection item should be a map")
		}

		record1, ok := firstItem["$Record"].(map[string]interface{})
		if !ok {
			t.Fatal("Collection item should have $Record key")
		}

		if record1["PropertyPath"] != "Name" {
			t.Errorf("First record PropertyPath = %v, want 'Name'", record1["PropertyPath"])
		}

		if record1["Filterable"] != true {
			t.Errorf("First record Filterable = %v, want true", record1["Filterable"])
		}

		// Second item should be a record
		secondItem, ok := collection[1].(map[string]interface{})
		if !ok {
			t.Fatal("Collection item should be a map")
		}

		record2, ok := secondItem["$Record"].(map[string]interface{})
		if !ok {
			t.Fatal("Collection item should have $Record key")
		}

		if record2["PropertyPath"] != "ID" {
			t.Errorf("Second record PropertyPath = %v, want 'ID'", record2["PropertyPath"])
		}

		if record2["Filterable"] != false {
			t.Errorf("Second record Filterable = %v, want false", record2["Filterable"])
		}
	})
}

func TestAnnotations_NestedCollectionAndRecord_XML(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityWithNestedAnnotation{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&EntityWithNestedAnnotation{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register nested annotation: collection of records
	nestedValue := []map[string]interface{}{
		{
			"PropertyPath": "Name",
			"Filterable":   true,
		},
		{
			"PropertyPath": "ID",
			"Filterable":   false,
		},
	}
	err = service.RegisterEntityAnnotation("EntityWithNestedAnnotations",
		"Org.OData.Capabilities.V1.FilterRestrictions",
		nestedValue)
	if err != nil {
		t.Fatalf("Failed to register nested annotation: %v", err)
	}

	t.Run("Nested collection of records in XML metadata", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/xml")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		body := w.Body.String()

		// Should contain Annotation element with nested Collection
		if !strings.Contains(body, "<Annotation Term=\"Org.OData.Capabilities.V1.FilterRestrictions\">") {
			t.Error("XML metadata should contain Annotation element with FilterRestrictions term")
		}

		if !strings.Contains(body, "<Collection>") {
			t.Error("XML metadata should contain <Collection> element")
		}

		if !strings.Contains(body, "<Record>") {
			t.Error("XML metadata should contain <Record> element inside Collection")
		}

		// Should contain PropertyValue elements for record properties
		if !strings.Contains(body, "Property=\"PropertyPath\"") {
			t.Error("XML metadata should contain PropertyValue elements with PropertyPath property")
		}

		if !strings.Contains(body, "Property=\"Filterable\"") {
			t.Error("XML metadata should contain PropertyValue elements with Filterable property")
		}
	})
}

func TestAnnotations_PrimitiveCollectionTypes_XML(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityWithCollectionAnnotation{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&EntityWithCollectionAnnotation{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register collection with different primitive types
	err = service.RegisterEntityAnnotation("EntityWithCollectionAnnotations",
		"Org.OData.Core.V1.MixedTypes",
		[]interface{}{
			"string value",
			42,
			3.14,
			true,
			false,
		})
	if err != nil {
		t.Fatalf("Failed to register collection annotation: %v", err)
	}

	t.Run("Collection with mixed primitive types in XML metadata", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/xml")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		body := w.Body.String()

		// Should contain Collection with various typed elements
		if !strings.Contains(body, "<String>string value</String>") {
			t.Error("XML metadata should contain <String> element for string value")
		}

		if !strings.Contains(body, "<Int>42</Int>") {
			t.Error("XML metadata should contain <Int> element for integer value")
		}

		if !strings.Contains(body, "<Float>3.14</Float>") {
			t.Error("XML metadata should contain <Float> element for float value")
		}

		if !strings.Contains(body, "<Bool>true</Bool>") {
			t.Error("XML metadata should contain <Bool>true</Bool> element")
		}

		if !strings.Contains(body, "<Bool>false</Bool>") {
			t.Error("XML metadata should contain <Bool>false</Bool> element")
		}
	})
}

func TestAnnotations_RecordWithVariousTypes_XML(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityWithRecordAnnotation{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&EntityWithRecordAnnotation{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Register record with various property types
	recordValue := map[string]interface{}{
		"StringProp": "test",
		"IntProp":    123,
		"FloatProp":  45.67,
		"BoolProp":   true,
	}
	err = service.RegisterEntityAnnotation("EntityWithRecordAnnotations",
		"Org.OData.Core.V1.RecordExample",
		recordValue)
	if err != nil {
		t.Fatalf("Failed to register record annotation: %v", err)
	}

	t.Run("Record with various typed properties in XML metadata", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/xml")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		body := w.Body.String()

		// Should contain PropertyValue elements with correct types
		if !strings.Contains(body, "Property=\"StringProp\" String=\"test\"") {
			t.Error("XML metadata should contain PropertyValue with String attribute for StringProp")
		}

		if !strings.Contains(body, "Property=\"IntProp\" Int=\"123\"") {
			t.Error("XML metadata should contain PropertyValue with Int attribute for IntProp")
		}

		if !strings.Contains(body, "Property=\"FloatProp\" Float=\"45.67\"") {
			t.Error("XML metadata should contain PropertyValue with Float attribute for FloatProp")
		}

		if !strings.Contains(body, "Property=\"BoolProp\" Bool=\"true\"") {
			t.Error("XML metadata should contain PropertyValue with Bool attribute for BoolProp")
		}
	})
}
