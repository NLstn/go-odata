package handlers

import (
	"context"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// BindTestCategory is a test entity for bind tests
type BindTestCategory struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name"`
}

// BindTestProduct is a test entity for bind tests with navigation properties
type BindTestProduct struct {
	ID         uint              `json:"ID" gorm:"primaryKey" odata:"key"`
	Name       string            `json:"Name"`
	CategoryID *uint             `json:"CategoryID"`
	Category   *BindTestCategory `json:"Category" gorm:"foreignKey:CategoryID"`
}

// BindTestOrder is a test entity for collection binding tests
type BindTestOrder struct {
	ID       uint              `json:"ID" gorm:"primaryKey" odata:"key"`
	Number   string            `json:"Number"`
	Products []BindTestProduct `json:"Products" gorm:"many2many:order_products"`
}

func setupBindTestHandler(t *testing.T) (*EntityHandler, *gorm.DB, map[string]*metadata.EntityMetadata) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&BindTestCategory{}, &BindTestProduct{}, &BindTestOrder{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	productMeta, err := metadata.AnalyzeEntity(BindTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze product entity: %v", err)
	}

	categoryMeta, err := metadata.AnalyzeEntity(BindTestCategory{})
	if err != nil {
		t.Fatalf("Failed to analyze category entity: %v", err)
	}

	orderMeta, err := metadata.AnalyzeEntity(BindTestOrder{})
	if err != nil {
		t.Fatalf("Failed to analyze order entity: %v", err)
	}

	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"BindTestProducts":   productMeta,
		"BindTestCategories": categoryMeta,
		"BindTestOrders":     orderMeta,
	}

	handler := NewEntityHandler(db, productMeta, nil)
	handler.SetEntitiesMetadata(entitiesMetadata)

	return handler, db, entitiesMetadata
}

func TestParseEntityReference(t *testing.T) {
	tests := []struct {
		name          string
		refURL        string
		wantEntitySet string
		wantEntityKey string
		wantErr       bool
	}{
		{
			name:          "simple reference",
			refURL:        "Categories(1)",
			wantEntitySet: "Categories",
			wantEntityKey: "1",
			wantErr:       false,
		},
		{
			name:          "composite key reference",
			refURL:        "Products(ProductID=1,LanguageKey='en')",
			wantEntitySet: "Products",
			wantEntityKey: "ProductID=1,LanguageKey='en'",
			wantErr:       false,
		},
		{
			name:          "absolute URL",
			refURL:        "http://host/service/Categories(1)",
			wantEntitySet: "Categories",
			wantEntityKey: "1",
			wantErr:       false,
		},
		{
			name:          "https URL",
			refURL:        "https://host/service/Products(42)",
			wantEntitySet: "Products",
			wantEntityKey: "42",
			wantErr:       false,
		},
		{
			name:          "root-relative URL",
			refURL:        "/service/Categories(1)",
			wantEntitySet: "Categories",
			wantEntityKey: "1",
			wantErr:       false,
		},
		{
			name:          "URL with spaces",
			refURL:        "  Categories(1)  ",
			wantEntitySet: "Categories",
			wantEntityKey: "1",
			wantErr:       false,
		},
		{
			name:    "missing key",
			refURL:  "Categories",
			wantErr: true,
		},
		{
			name:    "invalid format - no closing paren",
			refURL:  "Categories(1",
			wantErr: true,
		},
		{
			name:          "empty key",
			refURL:        "Categories()",
			wantEntitySet: "Categories",
			wantEntityKey: "",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entitySet, entityKey, err := parseEntityReference(tt.refURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseEntityReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if entitySet != tt.wantEntitySet {
					t.Errorf("parseEntityReference() entitySet = %v, want %v", entitySet, tt.wantEntitySet)
				}
				if entityKey != tt.wantEntityKey {
					t.Errorf("parseEntityReference() entityKey = %v, want %v", entityKey, tt.wantEntityKey)
				}
			}
		})
	}
}

func TestBindSingleNavigationProperty_Success(t *testing.T) {
	handler, db, _ := setupBindTestHandler(t)

	// Create a category
	category := BindTestCategory{ID: 1, Name: "Electronics"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Create a product instance
	product := BindTestProduct{ID: 1, Name: "Laptop"}

	// Find the Category navigation property
	navProp := handler.findNavigationProperty("Category")
	if navProp == nil {
		t.Fatal("Category navigation property not found")
	}

	// Test binding
	ctx := context.Background()
	productValue := reflect.ValueOf(&product).Elem()
	err := handler.bindSingleNavigationProperty(ctx, productValue, navProp, "BindTestCategories(1)", db)

	if err != nil {
		t.Errorf("bindSingleNavigationProperty() error = %v", err)
	}

	// Verify the foreign key was set (if there are referential constraints)
	if product.CategoryID != nil && *product.CategoryID != 1 {
		t.Errorf("CategoryID was not set correctly, got %v, want 1", *product.CategoryID)
	}
}

func TestBindSingleNavigationProperty_InvalidReference(t *testing.T) {
	handler, db, _ := setupBindTestHandler(t)

	product := BindTestProduct{ID: 1, Name: "Laptop"}
	navProp := handler.findNavigationProperty("Category")
	if navProp == nil {
		t.Fatal("Category navigation property not found")
	}

	tests := []struct {
		name    string
		refURL  interface{}
		wantErr bool
	}{
		{
			name:    "non-string reference",
			refURL:  123,
			wantErr: true,
		},
		{
			name:    "invalid URL format",
			refURL:  "InvalidFormat",
			wantErr: true,
		},
		{
			name:    "non-existent entity",
			refURL:  "BindTestCategories(999)",
			wantErr: true,
		},
		{
			name:    "wrong entity set",
			refURL:  "WrongEntitySet(1)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			productValue := reflect.ValueOf(&product).Elem()
			err := handler.bindSingleNavigationProperty(ctx, productValue, navProp, tt.refURL, db)

			if (err != nil) != tt.wantErr {
				t.Errorf("bindSingleNavigationProperty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBindSingleNavigationPropertyForUpdate_Success(t *testing.T) {
	handler, db, _ := setupBindTestHandler(t)

	// Create a category
	category := BindTestCategory{ID: 2, Name: "Books"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Create a product instance
	product := BindTestProduct{ID: 2, Name: "Novel"}

	// Find the Category navigation property
	navProp := handler.findNavigationProperty("Category")
	if navProp == nil {
		t.Fatal("Category navigation property not found")
	}

	// Test binding for update
	ctx := context.Background()
	productValue := reflect.ValueOf(&product).Elem()
	foreignKeyValues, err := handler.bindSingleNavigationPropertyForUpdate(ctx, productValue, navProp, "BindTestCategories(2)", db)

	if err != nil {
		t.Errorf("bindSingleNavigationPropertyForUpdate() error = %v", err)
	}

	// Verify foreign key values were returned
	if len(foreignKeyValues) > 0 {
		t.Logf("Foreign key values returned: %v", foreignKeyValues)
	}
}

func TestBindCollectionNavigationProperty_Success(t *testing.T) {
	_, db, entitiesMetadata := setupBindTestHandler(t)

	// Create products
	product1 := BindTestProduct{ID: 1, Name: "Product 1"}
	product2 := BindTestProduct{ID: 2, Name: "Product 2"}
	if err := db.Create(&product1).Error; err != nil {
		t.Fatalf("Failed to create product1: %v", err)
	}
	if err := db.Create(&product2).Error; err != nil {
		t.Fatalf("Failed to create product2: %v", err)
	}

	// Setup order handler
	orderHandler := NewEntityHandler(db, entitiesMetadata["BindTestOrders"], nil)
	orderHandler.SetEntitiesMetadata(entitiesMetadata)

	order := BindTestOrder{ID: 1, Number: "ORD-001"}

	// Find the Products navigation property
	navProp := orderHandler.findNavigationProperty("Products")
	if navProp == nil {
		t.Fatal("Products navigation property not found")
	}

	// Test binding collection
	ctx := context.Background()
	refArray := []interface{}{
		"BindTestProducts(1)",
		"BindTestProducts(2)",
	}

	orderValue := reflect.ValueOf(&order).Elem()
	targetEntities, err := orderHandler.bindCollectionNavigationProperty(ctx, orderValue, navProp, refArray, db)

	if err != nil {
		t.Errorf("bindCollectionNavigationProperty() error = %v", err)
	}

	if len(targetEntities) != 2 {
		t.Errorf("Expected 2 target entities, got %d", len(targetEntities))
	}
}

func TestBindCollectionNavigationProperty_EmptyArray(t *testing.T) {
	_, db, entitiesMetadata := setupBindTestHandler(t)

	orderHandler := NewEntityHandler(db, entitiesMetadata["BindTestOrders"], nil)
	orderHandler.SetEntitiesMetadata(entitiesMetadata)

	order := BindTestOrder{ID: 1, Number: "ORD-001"}

	navProp := orderHandler.findNavigationProperty("Products")
	if navProp == nil {
		t.Fatal("Products navigation property not found")
	}

	ctx := context.Background()
	refArray := []interface{}{}

	orderValue := reflect.ValueOf(&order).Elem()
	targetEntities, err := orderHandler.bindCollectionNavigationProperty(ctx, orderValue, navProp, refArray, db)

	if err != nil {
		t.Errorf("bindCollectionNavigationProperty() error = %v", err)
	}

	if len(targetEntities) != 0 {
		t.Errorf("Expected 0 target entities for empty array, got %d", len(targetEntities))
	}
}

func TestBindCollectionNavigationProperty_InvalidInputs(t *testing.T) {
	_, db, entitiesMetadata := setupBindTestHandler(t)

	orderHandler := NewEntityHandler(db, entitiesMetadata["BindTestOrders"], nil)
	orderHandler.SetEntitiesMetadata(entitiesMetadata)

	order := BindTestOrder{ID: 1, Number: "ORD-001"}

	navProp := orderHandler.findNavigationProperty("Products")
	if navProp == nil {
		t.Fatal("Products navigation property not found")
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "non-array value",
			value:   "not-an-array",
			wantErr: true,
		},
		{
			name:    "array with non-string element",
			value:   []interface{}{123},
			wantErr: true,
		},
		{
			name:    "array with invalid reference",
			value:   []interface{}{"InvalidFormat"},
			wantErr: true,
		},
		{
			name:    "non-existent entity",
			value:   []interface{}{"BindTestProducts(999)"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			orderValue := reflect.ValueOf(&order).Elem()
			_, err := orderHandler.bindCollectionNavigationProperty(ctx, orderValue, navProp, tt.value, db)

			if (err != nil) != tt.wantErr {
				t.Errorf("bindCollectionNavigationProperty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyPendingCollectionBindings_Success(t *testing.T) {
	_, db, entitiesMetadata := setupBindTestHandler(t)

	// Create products
	product1 := BindTestProduct{ID: 10, Name: "Product 10"}
	product2 := BindTestProduct{ID: 11, Name: "Product 11"}
	if err := db.Create(&product1).Error; err != nil {
		t.Fatalf("Failed to create product1: %v", err)
	}
	if err := db.Create(&product2).Error; err != nil {
		t.Fatalf("Failed to create product2: %v", err)
	}

	// Create and save order first
	order := BindTestOrder{ID: 10, Number: "ORD-010"}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	// Setup order handler
	orderHandler := NewEntityHandler(db, entitiesMetadata["BindTestOrders"], nil)
	orderHandler.SetEntitiesMetadata(entitiesMetadata)

	navProp := orderHandler.findNavigationProperty("Products")
	if navProp == nil {
		t.Fatal("Products navigation property not found")
	}

	// Create pending bindings
	pendingBindings := []PendingCollectionBinding{
		{
			NavigationProperty: navProp,
			TargetEntities:     []interface{}{&product1, &product2},
		},
	}

	ctx := context.Background()
	err := orderHandler.applyPendingCollectionBindings(ctx, db, &order, pendingBindings)

	if err != nil {
		t.Errorf("applyPendingCollectionBindings() error = %v", err)
	}

	// Verify the bindings were applied
	var reloadedOrder BindTestOrder
	if err := db.Preload("Products").First(&reloadedOrder, order.ID).Error; err != nil {
		t.Fatalf("Failed to reload order: %v", err)
	}

	if len(reloadedOrder.Products) != 2 {
		t.Errorf("Expected 2 products bound to order, got %d", len(reloadedOrder.Products))
	}
}

func TestApplyPendingCollectionBindings_EmptyBindings(t *testing.T) {
	handler, db, _ := setupBindTestHandler(t)

	product := BindTestProduct{ID: 1, Name: "Test"}

	ctx := context.Background()
	err := handler.applyPendingCollectionBindings(ctx, db, &product, []PendingCollectionBinding{})

	if err != nil {
		t.Errorf("applyPendingCollectionBindings() with empty bindings should not error, got %v", err)
	}
}
