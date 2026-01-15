package handlers

import (
	"context"
	"log/slog"
	"testing"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/observability"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// EntityTestProduct is a test entity for entity handler tests
type EntityTestProduct struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name"`
	Price       float64 `json:"Price"`
	Category    string  `json:"Category"`
	Description string  `json:"Description"`
}

func setupEntityHandlerTest(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(EntityTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	return handler, db
}

func TestEntityHandler_SetLogger(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	// Test with custom logger
	customLogger := slog.Default()
	handler.SetLogger(customLogger)
	if handler.logger != customLogger {
		t.Error("Expected custom logger to be set")
	}

	// Test with nil logger (should use default)
	handler.SetLogger(nil)
	if handler.logger == nil {
		t.Error("Expected default logger when nil is passed")
	}
}

func TestEntityHandler_SetKeyGeneratorResolver(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	resolver := func(name string) (func(context.Context) (interface{}, error), bool) {
		if name == "test" {
			return func(ctx context.Context) (interface{}, error) {
				return 123, nil
			}, true
		}
		return nil, false
	}

	handler.SetKeyGeneratorResolver(resolver)

	// Verify the resolver was set by checking if it's not nil
	if handler.keyGeneratorResolver == nil {
		t.Error("Expected key generator resolver to be set")
	}

	// Test the resolver works
	gen, ok := handler.keyGeneratorResolver("test")
	if !ok {
		t.Error("Expected resolver to find 'test' generator")
	}
	if gen == nil {
		t.Error("Expected generator function to be returned")
	}

	result, err := gen(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 123 {
		t.Errorf("Expected 123, got %v", result)
	}
}

func TestEntityHandler_namespaceOrDefault(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	// Test default namespace
	result := handler.namespaceOrDefault()
	if result != defaultNamespace {
		t.Errorf("Expected default namespace %s, got %s", defaultNamespace, result)
	}

	// Test custom namespace
	handler.SetNamespace("CustomNamespace")
	result = handler.namespaceOrDefault()
	if result != "CustomNamespace" {
		t.Errorf("Expected CustomNamespace, got %s", result)
	}

	// Test empty namespace returns default
	handler.SetNamespace("")
	result = handler.namespaceOrDefault()
	if result != defaultNamespace {
		t.Errorf("Expected default namespace %s, got %s", defaultNamespace, result)
	}

	// Test whitespace namespace returns default
	handler.SetNamespace("   ")
	result = handler.namespaceOrDefault()
	if result != defaultNamespace {
		t.Errorf("Expected default namespace %s, got %s", defaultNamespace, result)
	}
}

func TestEntityHandler_qualifiedTypeName(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	// Test with default namespace
	result := handler.qualifiedTypeName("Product")
	expected := defaultNamespace + ".Product"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test with custom namespace
	handler.SetNamespace("CustomNamespace")
	result = handler.qualifiedTypeName("Product")
	expected = "CustomNamespace.Product"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestEntityHandler_SetDefaultMaxTop(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	maxTop := 100
	handler.SetDefaultMaxTop(&maxTop)

	if handler.defaultMaxTop == nil {
		t.Error("Expected defaultMaxTop to be set")
	} else if *handler.defaultMaxTop != 100 {
		t.Errorf("Expected defaultMaxTop to be 100, got %d", *handler.defaultMaxTop)
	}

	// Test with nil
	handler.SetDefaultMaxTop(nil)
	if handler.defaultMaxTop != nil {
		t.Error("Expected defaultMaxTop to be nil")
	}
}

func TestEntityHandler_SetObservability(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	cfg := &observability.Config{}
	handler.SetObservability(cfg)

	if handler.observability != cfg {
		t.Error("Expected observability config to be set")
	}
}

func TestEntityHandler_SetGeospatialEnabled(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	handler.SetGeospatialEnabled(true)
	if !handler.IsGeospatialEnabled() {
		t.Error("Expected geospatial to be enabled")
	}

	handler.SetGeospatialEnabled(false)
	if handler.IsGeospatialEnabled() {
		t.Error("Expected geospatial to be disabled")
	}
}

func TestEntityHandler_IsGeospatialEnabled(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	// Default should be false
	if handler.IsGeospatialEnabled() {
		t.Error("Expected geospatial to be disabled by default")
	}

	handler.SetGeospatialEnabled(true)
	if !handler.IsGeospatialEnabled() {
		t.Error("Expected geospatial to be enabled")
	}
}

func TestEntityHandler_SetMaxInClauseSize(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	handler.SetMaxInClauseSize(1000)
	if handler.maxInClauseSize != 1000 {
		t.Errorf("Expected maxInClauseSize to be 1000, got %d", handler.maxInClauseSize)
	}
}

func TestEntityHandler_SetMaxExpandDepth(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	handler.SetMaxExpandDepth(5)
	if handler.maxExpandDepth != 5 {
		t.Errorf("Expected maxExpandDepth to be 5, got %d", handler.maxExpandDepth)
	}
}

func TestEntityHandler_HasEntityLevelDefaultMaxTop(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	// Default should be false (no entity-level max top set)
	if handler.HasEntityLevelDefaultMaxTop() {
		t.Error("Expected HasEntityLevelDefaultMaxTop to be false by default")
	}

	// Set entity-level max top
	maxTop := 50
	handler.metadata.DefaultMaxTop = &maxTop
	if !handler.HasEntityLevelDefaultMaxTop() {
		t.Error("Expected HasEntityLevelDefaultMaxTop to be true after setting")
	}
}

func TestEntityHandler_SetDeltaTracker(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	tracker := trackchanges.NewTracker()
	handler.SetDeltaTracker(tracker)

	if handler.tracker != tracker {
		t.Error("Expected delta tracker to be set")
	}
}

func TestEntityHandler_SetEntitiesMetadata(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	entityMeta, err := metadata.AnalyzeEntity(EntityTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"Products": entityMeta,
	}

	handler.SetEntitiesMetadata(entitiesMetadata)

	if handler.entitiesMetadata == nil {
		t.Error("Expected entities metadata to be set")
	}

	if len(handler.entitiesMetadata) != 1 {
		t.Errorf("Expected 1 entity metadata, got %d", len(handler.entitiesMetadata))
	}
}

func TestIsNotFoundError(t *testing.T) {
	// Test with record not found error
	err := gorm.ErrRecordNotFound
	if !IsNotFoundError(err) {
		t.Error("Expected IsNotFoundError to return true for ErrRecordNotFound")
	}

	// Test with other error
	err = gorm.ErrInvalidData
	if IsNotFoundError(err) {
		t.Error("Expected IsNotFoundError to return false for non-NotFound error")
	}

	// Test with nil error
	if IsNotFoundError(nil) {
		t.Error("Expected IsNotFoundError to return false for nil error")
	}
}

func TestEntityHandler_SetOverwriteMethods(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	// Test SetGetCollectionOverwrite
	getCollectionFunc := func(ctx *OverwriteContext) (*CollectionResult, error) {
		return &CollectionResult{}, nil
	}
	handler.SetGetCollectionOverwrite(getCollectionFunc)
	handler.ensureOverwrite()
	if handler.overwrite.getCollection == nil {
		t.Error("Expected GetCollectionOverwrite to be set")
	}

	// Test SetGetEntityOverwrite
	getEntityFunc := func(ctx *OverwriteContext) (interface{}, error) {
		return nil, nil
	}
	handler.SetGetEntityOverwrite(getEntityFunc)
	handler.ensureOverwrite()
	if handler.overwrite.getEntity == nil {
		t.Error("Expected GetEntityOverwrite to be set")
	}

	// Test SetCreateOverwrite
	createFunc := func(ctx *OverwriteContext, entity interface{}) (interface{}, error) {
		return entity, nil
	}
	handler.SetCreateOverwrite(createFunc)
	handler.ensureOverwrite()
	if handler.overwrite.create == nil {
		t.Error("Expected CreateOverwrite to be set")
	}

	// Test SetUpdateOverwrite
	updateFunc := func(ctx *OverwriteContext, updateData map[string]interface{}, isFullReplace bool) (interface{}, error) {
		return nil, nil
	}
	handler.SetUpdateOverwrite(updateFunc)
	handler.ensureOverwrite()
	if handler.overwrite.update == nil {
		t.Error("Expected UpdateOverwrite to be set")
	}

	// Test SetDeleteOverwrite
	deleteFunc := func(ctx *OverwriteContext) error {
		return nil
	}
	handler.SetDeleteOverwrite(deleteFunc)
	handler.ensureOverwrite()
	if handler.overwrite.delete == nil {
		t.Error("Expected DeleteOverwrite to be set")
	}

	// Test SetGetCountOverwrite
	getCountFunc := func(ctx *OverwriteContext) (int64, error) {
		return 0, nil
	}
	handler.SetGetCountOverwrite(getCountFunc)
	handler.ensureOverwrite()
	if handler.overwrite.getCount == nil {
		t.Error("Expected GetCountOverwrite to be set")
	}
}

func TestEntityHandler_ensureOverwrite(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	// Initially overwrite should be nil
	if handler.overwrite != nil {
		t.Error("Expected overwrite to be nil initially")
	}

	// Call ensureOverwrite
	handler.ensureOverwrite()

	// Now overwrite should be initialized
	if handler.overwrite == nil {
		t.Error("Expected overwrite to be initialized")
	}

	// Calling again should not create a new instance
	originalOverwrite := handler.overwrite
	handler.ensureOverwrite()
	if handler.overwrite != originalOverwrite {
		t.Error("Expected ensureOverwrite to not create new instance")
	}
}

// mockPolicy is a simple mock implementation of auth.Policy for testing
type mockPolicy struct{}

func (m *mockPolicy) Authorize(ctx auth.AuthContext, resource auth.ResourceDescriptor, operation auth.Operation) auth.Decision {
	return auth.Allow()
}

func TestEntityHandler_SetPolicy(t *testing.T) {
	handler, _ := setupEntityHandlerTest(t)

	// Create a simple test policy
	policy := &mockPolicy{}

	handler.SetPolicy(policy)

	// Verify the policy was set
	if handler.policy != policy {
		t.Error("Expected policy to be set to the provided policy")
	}
}
