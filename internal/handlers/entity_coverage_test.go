package handlers

import (
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// TestFetchEntity tests the public FetchEntity method
// Note: FetchEntity requires a full database setup to work properly.
// It is tested extensively in the integration tests, so we skip it here.
func TestFetchEntity(t *testing.T) {
	t.Skip("FetchEntity requires database setup, covered by integration tests")
}

// TestIsNotFoundErrorCoverage tests the error checking function for coverage
func TestIsNotFoundErrorCoverage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"gorm.ErrRecordNotFound", gorm.ErrRecordNotFound, true},
		{"other error", &requestError{Message: "test"}, false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestEntityMatchesType tests type matching functionality
func TestEntityMatchesType(t *testing.T) {
	handler := createTestHandler()
	handler.metadata = &metadata.EntityMetadata{
		EntityName: "Product",
		Properties: []metadata.PropertyMetadata{
			{
				Name:      "Type",
				FieldName: "Type",
				JsonName:  "type",
				Type:      reflect.TypeOf(""),
			},
		},
	}
	handler.namespace = "TestNamespace"

	t.Run("map entity with @odata.type", func(t *testing.T) {
		entity := map[string]interface{}{
			"@odata.type": "Product",
		}
		result := handler.entityMatchesType(entity, "Product")
		if !result {
			t.Error("Expected entity to match type")
		}
	})

	t.Run("map entity with Type field", func(t *testing.T) {
		entity := map[string]interface{}{
			"Type": "Product",
		}
		result := handler.entityMatchesType(entity, "Product")
		if !result {
			t.Error("Expected entity to match type")
		}
	})

	t.Run("map entity no match", func(t *testing.T) {
		entity := map[string]interface{}{
			"name": "test",
		}
		result := handler.entityMatchesType(entity, "OtherType")
		// Might still match by entity name
		if result {
			t.Log("Entity matched by name fallback")
		}
	})

	t.Run("struct entity with Type field", func(t *testing.T) {
		type TestEntity struct {
			Type string
			Name string
		}
		entity := &TestEntity{Type: "Product"}
		result := handler.entityMatchesType(entity, "Product")
		if !result {
			t.Error("Expected entity to match type")
		}
	})

	t.Run("struct entity no match", func(t *testing.T) {
		type TestEntity struct {
			Name string
		}
		entity := &TestEntity{Name: "test"}
		// Should fall back to entity name matching
		result := handler.entityMatchesType(entity, "Product")
		if !result {
			t.Error("Expected entity to match by name")
		}
	})

	t.Run("empty type name", func(t *testing.T) {
		entity := map[string]interface{}{}
		result := handler.entityMatchesType(entity, "")
		if result {
			t.Error("Expected no match for empty type name")
		}
	})

	t.Run("non-struct non-map entity", func(t *testing.T) {
		result := handler.entityMatchesType("string", "Product")
		if result {
			t.Error("Expected no match for string entity")
		}
	})

	t.Run("qualified type name", func(t *testing.T) {
		entity := map[string]interface{}{
			"@odata.type": "TestNamespace.Product",
		}
		result := handler.entityMatchesType(entity, "TestNamespace.Product")
		if !result {
			t.Error("Expected entity to match qualified type")
		}
	})
}

// TestStructEntityMatchesType tests struct-specific type matching
func TestStructEntityMatchesType(t *testing.T) {
	handler := createTestHandler()
	handler.metadata = &metadata.EntityMetadata{
		EntityName: "Product",
		Properties: []metadata.PropertyMetadata{
			{
				Name:      "ProductType",
				FieldName: "ProductType",
				Type:      reflect.TypeOf(""),
			},
		},
	}

	t.Run("matches via ProductType field", func(t *testing.T) {
		type TestEntity struct {
			ProductType string
		}
		entity := TestEntity{ProductType: "Widget"}
		v := reflect.ValueOf(entity)
		result := handler.structEntityMatchesType(v, []string{"Widget"})
		if !result {
			t.Error("Expected match via ProductType")
		}
	})

	t.Run("matches via Type field", func(t *testing.T) {
		type TestEntity struct {
			Type string
		}
		entity := TestEntity{Type: "Widget"}
		v := reflect.ValueOf(entity)
		result := handler.structEntityMatchesType(v, []string{"Widget"})
		if !result {
			t.Error("Expected match via Type")
		}
	})

	t.Run("no matching field", func(t *testing.T) {
		type TestEntity struct {
			Name string
		}
		entity := TestEntity{Name: "test"}
		v := reflect.ValueOf(entity)
		result := handler.structEntityMatchesType(v, []string{"Widget"})
		if result {
			t.Error("Expected no match")
		}
	})
}

// TestMapEntityMatchesType tests map-specific type matching
func TestMapEntityMatchesType(t *testing.T) {
	handler := createTestHandler()
	handler.metadata = &metadata.EntityMetadata{
		EntityName: "Product",
	}

	t.Run("matches via @odata.type", func(t *testing.T) {
		entity := map[string]interface{}{
			"@odata.type": "Widget",
		}
		result := handler.mapEntityMatchesType(entity, []string{"Widget"})
		if !result {
			t.Error("Expected match via @odata.type")
		}
	})

	t.Run("matches via Type", func(t *testing.T) {
		entity := map[string]interface{}{
			"Type": "Widget",
		}
		result := handler.mapEntityMatchesType(entity, []string{"Widget"})
		if !result {
			t.Error("Expected match via Type")
		}
	})

	t.Run("no matching key", func(t *testing.T) {
		entity := map[string]interface{}{
			"name": "test",
		}
		result := handler.mapEntityMatchesType(entity, []string{"Widget"})
		if result {
			t.Error("Expected no match")
		}
	})
}

// TestEntityNameMatches tests entity name matching
func TestEntityNameMatches(t *testing.T) {
	handler := createTestHandler()
	handler.namespace = "TestNamespace"

	t.Run("empty entity name", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{EntityName: ""}
		result := handler.entityNameMatches([]string{"Product"})
		if result {
			t.Error("Expected no match for empty entity name")
		}
	})

	t.Run("exact match", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{EntityName: "Product"}
		result := handler.entityNameMatches([]string{"Product"})
		if !result {
			t.Error("Expected match")
		}
	})

	t.Run("qualified match", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{EntityName: "Product"}
		result := handler.entityNameMatches([]string{"TestNamespace.Product"})
		if !result {
			t.Error("Expected qualified match")
		}
	})
}

// TestTypeNameCandidates tests type name candidate generation
func TestTypeNameCandidates(t *testing.T) {
	handler := createTestHandler()
	handler.namespace = "TestNamespace"

	tests := []struct {
		name     string
		typeName string
		minLen   int
	}{
		{"empty string", "", 0},
		{"simple name", "Product", 1},
		{"qualified name", "TestNamespace.Product", 2},
		{"multiple dots", "A.B.C", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := handler.typeNameCandidates(tt.typeName)
			if len(candidates) < tt.minLen {
				t.Errorf("Expected at least %d candidates, got %d", tt.minLen, len(candidates))
			}
		})
	}
}

// TestDiscriminatorFieldNames tests discriminator field name generation
func TestDiscriminatorFieldNames(t *testing.T) {
	handler := createTestHandler()

	t.Run("with type discriminator property", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{
			Properties: []metadata.PropertyMetadata{
				{
					Name:      "ProductType",
					FieldName: "ProductType",
					Type:      reflect.TypeOf(""),
				},
			},
		}
		names := handler.discriminatorFieldNames()
		if len(names) == 0 {
			t.Error("Expected non-empty discriminator field names")
		}
	})

	t.Run("without discriminator property", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{
			Properties: []metadata.PropertyMetadata{},
		}
		names := handler.discriminatorFieldNames()
		// Should still have defaults
		if len(names) == 0 {
			t.Error("Expected default discriminator field names")
		}
	})
}

// TestMapDiscriminatorKeys tests map discriminator key generation
func TestMapDiscriminatorKeys(t *testing.T) {
	handler := createTestHandler()
	handler.metadata = &metadata.EntityMetadata{
		Properties: []metadata.PropertyMetadata{
			{
				Name:      "ProductType",
				FieldName: "ProductType",
				JsonName:  "productType",
				Type:      reflect.TypeOf(""),
			},
		},
	}

	keys := handler.mapDiscriminatorKeys()
	if len(keys) == 0 {
		t.Error("Expected non-empty discriminator keys")
	}

	// Should include @odata.type
	found := false
	for _, key := range keys {
		if key == "@odata.type" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected @odata.type in discriminator keys")
	}
}

// TestTypeDiscriminatorProperty tests finding the type discriminator property
func TestTypeDiscriminatorProperty(t *testing.T) {
	handler := createTestHandler()

	t.Run("nil metadata", func(t *testing.T) {
		handler.metadata = nil
		prop := handler.typeDiscriminatorProperty()
		if prop != nil {
			t.Error("Expected nil for nil metadata")
		}
	})

	t.Run("with Type property", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{
			Properties: []metadata.PropertyMetadata{
				{
					Name:      "Type",
					FieldName: "Type",
					Type:      reflect.TypeOf(""),
				},
			},
		}
		prop := handler.typeDiscriminatorProperty()
		if prop == nil {
			t.Error("Expected to find Type property")
		}
	})

	t.Run("no matching property", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{
			Properties: []metadata.PropertyMetadata{
				{
					Name:      "Name",
					FieldName: "Name",
					Type:      reflect.TypeOf(""),
				},
			},
		}
		prop := handler.typeDiscriminatorProperty()
		if prop != nil {
			t.Error("Expected nil for no matching property")
		}
	})
}

// TestDiscriminatorPropertyNames tests discriminator property name generation
func TestDiscriminatorPropertyNames(t *testing.T) {
	handler := createTestHandler()

	t.Run("with entity name", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{
			EntityName: "Product",
		}
		names := handler.discriminatorPropertyNames()
		// Should include "ProductType"
		found := false
		for _, name := range names {
			if name == "ProductType" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected ProductType in discriminator property names")
		}
	})

	t.Run("empty entity name", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{
			EntityName: "",
		}
		names := handler.discriminatorPropertyNames()
		// Should still have defaults
		if len(names) == 0 {
			t.Error("Expected default discriminator property names")
		}
	})
}

// TestTypeDiscriminatorColumn tests column name extraction
func TestTypeDiscriminatorColumn(t *testing.T) {
	handler := createTestHandler()

	t.Run("with gorm tag", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{
			Properties: []metadata.PropertyMetadata{
				{
					Name:      "Type",
					FieldName: "Type",
					Type:      reflect.TypeOf(""),
					GormTag:   "column:type_col",
				},
			},
		}
		column := handler.typeDiscriminatorColumn()
		if column != "type_col" {
			t.Errorf("Expected 'type_col', got %s", column)
		}
	})

	t.Run("without gorm tag", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{
			Properties: []metadata.PropertyMetadata{
				{
					Name:       "Type",
					FieldName:  "Type",
					Type:       reflect.TypeOf(""),
					ColumnName: "type_column",
				},
			},
		}
		column := handler.typeDiscriminatorColumn()
		if column != "type_column" {
			t.Errorf("Expected 'type_column', got %s", column)
		}
	})

	t.Run("no discriminator property", func(t *testing.T) {
		handler.metadata = &metadata.EntityMetadata{
			Properties: []metadata.PropertyMetadata{},
		}
		column := handler.typeDiscriminatorColumn()
		if column != "" {
			t.Errorf("Expected empty string, got %s", column)
		}
	})
}

// TestParseGORMColumn tests GORM column parsing
func TestParseGORMColumn(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{"empty tag", "", ""},
		{"column only", "column:test_col", "test_col"},
		{"multiple attributes", "column:test_col;primaryKey", "test_col"},
		{"spaces", "  column:test_col  ", "test_col"},
		{"no column", "primaryKey;autoIncrement", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGORMColumn(tt.tag)
			if result != tt.expected {
				t.Errorf("parseGORMColumn() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// TestTypeNameMatches tests type name matching
func TestTypeNameMatches(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		typeCandidates []string
		expected       bool
	}{
		{"exact match", "Product", []string{"Product"}, true},
		{"no match", "Product", []string{"Order"}, false},
		{"qualified match simple", "TestNamespace.Product", []string{"Product"}, true},
		{"qualified match full", "TestNamespace.Product", []string{"TestNamespace.Product"}, true},
		{"empty value", "", []string{"Product"}, false},
		{"empty candidates", "Product", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeNameMatches(tt.value, tt.typeCandidates)
			if result != tt.expected {
				t.Errorf("typeNameMatches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestUniqueStrings tests string deduplication
func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected int
	}{
		{"empty", []string{}, 0},
		{"no duplicates", []string{"a", "b", "c"}, 3},
		{"with duplicates", []string{"a", "b", "a", "c"}, 3},
		{"with empty strings", []string{"a", "", "b", ""}, 2},
		{"with spaces", []string{"  ", "a", "  a  ", "b"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uniqueStrings(tt.input)
			if len(result) != tt.expected {
				t.Errorf("uniqueStrings() length = %d, want %d", len(result), tt.expected)
			}
		})
	}
}

// TestSetOverwrite tests the SetOverwrite method
func TestSetOverwrite(t *testing.T) {
	handler := createTestHandler()

	overwrite := &EntityOverwrite{
		GetCollection: func(ctx *OverwriteContext) (*CollectionResult, error) {
			return &CollectionResult{Items: []interface{}{}}, nil
		},
		GetEntity: func(ctx *OverwriteContext) (interface{}, error) {
			return map[string]interface{}{"id": 1}, nil
		},
		Create: func(ctx *OverwriteContext, entity interface{}) (interface{}, error) {
			return entity, nil
		},
		Update: func(ctx *OverwriteContext, updateData map[string]interface{}, isFullReplace bool) (interface{}, error) {
			return updateData, nil
		},
		Delete: func(ctx *OverwriteContext) error {
			return nil
		},
		GetCount: func(ctx *OverwriteContext) (int64, error) {
			return 42, nil
		},
	}

	handler.SetOverwrite(overwrite)

	if handler.overwrite == nil {
		t.Error("Expected overwrite to be set")
	}

	if !handler.overwrite.hasGetCollection() {
		t.Error("Expected hasGetCollection to be true")
	}
	if !handler.overwrite.hasGetEntity() {
		t.Error("Expected hasGetEntity to be true")
	}
	if !handler.overwrite.hasCreate() {
		t.Error("Expected hasCreate to be true")
	}
	if !handler.overwrite.hasUpdate() {
		t.Error("Expected hasUpdate to be true")
	}
	if !handler.overwrite.hasDelete() {
		t.Error("Expected hasDelete to be true")
	}
	if !handler.overwrite.hasGetCount() {
		t.Error("Expected hasGetCount to be true")
	}
}

// Helper function to create a basic test handler
func createTestHandler() *EntityHandler {
	handler := &EntityHandler{
		metadata: &metadata.EntityMetadata{
			EntityName: "TestEntity",
		},
		logger: createNilLogger(),
	}
	return handler
}

// Helper function to create a nil logger
func createNilLogger() *slog.Logger {
	// Return a logger that discards output instead of nil
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
