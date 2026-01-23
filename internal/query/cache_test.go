package query

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestParserCacheBasic tests that the cache works correctly
func TestParserCacheBasic(t *testing.T) {
	cache := newParserCache()

	// Create simple metadata
	meta := &metadata.EntityMetadata{
		EntitySetName: "Products",
		EntityName:    "Product",
		EntityType: reflect.TypeOf(struct {
			ID   string
			Name string
		}{}),
		Properties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Name: "ID", Type: reflect.TypeOf(""), IsKey: true, ColumnName: "id"},
			{JsonName: "Name", FieldName: "Name", Name: "Name", Type: reflect.TypeOf(""), ColumnName: "name"},
		},
	}

	// First call - should compute and cache
	result1 := cache.propertyExistsWithCache("Name", meta)
	if !result1 {
		t.Error("Expected Name to exist")
	}

	// Check cache has entry
	if len(cache.resolvedPaths) != 1 {
		t.Errorf("Expected cache to have 1 entry, got %d", len(cache.resolvedPaths))
	}

	// Second call - should hit cache
	result2 := cache.propertyExistsWithCache("Name", meta)
	if !result2 {
		t.Error("Expected Name to exist (from cache)")
	}

	// Cache should still have 1 entry
	if len(cache.resolvedPaths) != 1 {
		t.Errorf("Expected cache to still have 1 entry, got %d", len(cache.resolvedPaths))
	}

	// Non-existent property
	result3 := cache.propertyExistsWithCache("NonExistent", meta)
	if result3 {
		t.Error("Expected NonExistent to not exist")
	}

	// Cache should now have 2 entries
	if len(cache.resolvedPaths) != 2 {
		t.Errorf("Expected cache to have 2 entries, got %d", len(cache.resolvedPaths))
	}
}

// TestCacheWithNavigationPaths tests caching with navigation properties
func TestCacheWithNavigationPaths(t *testing.T) {
	// Create metadata with navigation properties
	categoryMeta := &metadata.EntityMetadata{
		EntitySetName: "Categories",
		EntityName:    "Category",
		EntityType: reflect.TypeOf(struct {
			ID   string
			Name string
		}{}),
		Properties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Name: "ID", Type: reflect.TypeOf(""), IsKey: true, ColumnName: "id"},
			{JsonName: "Name", FieldName: "Name", Name: "Name", Type: reflect.TypeOf(""), ColumnName: "name"},
		},
	}

	productMeta := &metadata.EntityMetadata{
		EntitySetName: "Products",
		EntityName:    "Product",
		EntityType: reflect.TypeOf(struct {
			ID         string
			Name       string
			CategoryID string
		}{}),
		Properties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Name: "ID", Type: reflect.TypeOf(""), IsKey: true, ColumnName: "id"},
			{JsonName: "Name", FieldName: "Name", Name: "Name", Type: reflect.TypeOf(""), ColumnName: "name"},
			{JsonName: "CategoryID", FieldName: "CategoryID", Name: "CategoryID", Type: reflect.TypeOf(""), ColumnName: "category_id"},
			{JsonName: "Category", FieldName: "Category", Name: "Category", Type: reflect.TypeOf(categoryMeta), IsNavigationProp: true, NavigationIsArray: false, NavigationTarget: "Category"},
		},
	}

	registry := map[string]*metadata.EntityMetadata{
		"Category": categoryMeta,
		"Product":  productMeta,
	}
	productMeta.SetEntitiesRegistry(registry)

	// Parse a query with navigation paths multiple times
	params := url.Values{
		"$filter": []string{"Category/Name eq 'Electronics' and Name eq 'Laptop'"},
	}

	// First parse
	_, err := ParseQueryOptions(params, productMeta)
	if err != nil {
		t.Fatalf("First parse failed: %v", err)
	}

	// Second parse - should benefit from cache
	_, err = ParseQueryOptions(params, productMeta)
	if err != nil {
		t.Fatalf("Second parse failed: %v", err)
	}
}

// TestConcurrentCacheAccess tests that the cache is thread-safe
func TestConcurrentCacheAccess(t *testing.T) {
	cache := newParserCache()

	meta := &metadata.EntityMetadata{
		EntitySetName: "Products",
		EntityName:    "Product",
		EntityType: reflect.TypeOf(struct {
			ID   string
			Name string
		}{}),
		Properties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Name: "ID", Type: reflect.TypeOf(""), IsKey: true, ColumnName: "id"},
			{JsonName: "Name", FieldName: "Name", Name: "Name", Type: reflect.TypeOf(""), ColumnName: "name"},
		},
	}

	// Run concurrent accesses
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cache.propertyExistsWithCache("Name", meta)
				cache.propertyExistsWithCache("ID", meta)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify cache has entries
	if len(cache.resolvedPaths) != 2 {
		t.Errorf("Expected cache to have 2 entries, got %d", len(cache.resolvedPaths))
	}
}
