package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestProduct is a test entity for lambda applier tests
type TestProduct struct {
	ID           uint                     `json:"ID" gorm:"primaryKey" odata:"key"`
	Name         string                   `json:"Name"`
	Price        float64                  `json:"Price"`
	Descriptions []TestProductDescription `json:"Descriptions" gorm:"foreignKey:TestProductID;references:ID"`
}

// TableName specifies the table name for TestProduct
func (TestProduct) TableName() string {
	return "TestProducts"
}

// TestProductDescription is a test entity for lambda applier tests
type TestProductDescription struct {
	TestProductID uint         `json:"TestProductID" gorm:"primaryKey;column:test_product_id" odata:"key"`
	LanguageKey   string       `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key"`
	Description   string       `json:"Description"`
	CustomName    string       `json:"CustomName" gorm:"column:custom_name"`
	Locale        string       `json:"Locale" gorm:"column:locale_code"`
	Product       *TestProduct `gorm:"foreignKey:TestProductID;references:ID"`
}

// TableName specifies the table name for TestProductDescription
func (TestProductDescription) TableName() string {
	return "TestProductDescriptions"
}

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate the schema
	if err := db.AutoMigrate(&TestProduct{}, &TestProductDescription{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	products := []TestProduct{
		{ID: 1, Name: "Laptop", Price: 999.99},
		{ID: 2, Name: "Mouse", Price: 29.99},
		{ID: 3, Name: "Keyboard", Price: 79.99},
		{ID: 4, Name: "Monitor", Price: 299.99},
	}

	descriptions := []TestProductDescription{
		{TestProductID: 1, LanguageKey: "EN", Description: "High-performance laptop", CustomName: "Promo", Locale: "EN"},
		{TestProductID: 1, LanguageKey: "DE", Description: "Hochleistungs-Laptop", Locale: "DE"},
		{TestProductID: 2, LanguageKey: "EN", Description: "Wireless mouse", Locale: "EN"},
		{TestProductID: 3, LanguageKey: "EN", Description: "Mechanical keyboard", Locale: "EN"},
		{TestProductID: 3, LanguageKey: "FR", Description: "Clavier mécanique", Locale: "FR"},
	}

	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to seed products: %v", err)
	}

	if err := db.Create(&descriptions).Error; err != nil {
		t.Fatalf("Failed to seed descriptions: %v", err)
	}

	return db
}

// getTestProductMetadata creates test metadata for TestProduct
func getTestProductMetadata() *metadata.EntityMetadata {
	productMeta, err := metadata.AnalyzeEntity(TestProduct{})
	if err != nil {
		panic(err)
	}
	descriptionMeta, err := metadata.AnalyzeEntity(TestProductDescription{})
	if err != nil {
		panic(err)
	}

	registry := map[string]*metadata.EntityMetadata{
		"TestProduct":            productMeta,
		"TestProductDescription": descriptionMeta,
	}
	productMeta.SetEntitiesRegistry(registry)
	descriptionMeta.SetEntitiesRegistry(registry)

	return productMeta
}

func TestLambdaApplier_SimpleAny(t *testing.T) {
	db := setupTestDB(t)
	entityMetadata := getTestProductMetadata()

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "any with simple equality",
			filter:        "Descriptions/any(d: d/LanguageKey eq 'EN')",
			expectedCount: 3,
			description:   "Products that have EN descriptions: Laptop, Mouse, Keyboard",
		},
		{
			name:          "any with different language",
			filter:        "Descriptions/any(d: d/LanguageKey eq 'DE')",
			expectedCount: 1,
			description:   "Products that have DE descriptions: Laptop",
		},
		{
			name:          "any with non-existent language",
			filter:        "Descriptions/any(d: d/LanguageKey eq 'XX')",
			expectedCount: 0,
			description:   "No products have XX descriptions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the filter
			filterExpr, err := parseFilter(tt.filter, entityMetadata, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			// Apply the filter
			query := db.Model(&TestProduct{})
			query = ApplyFilterOnly(query, filterExpr, entityMetadata, nil)

			// Count the results
			var count int64
			if err := query.Count(&count).Error; err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if int(count) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d. %s", tt.expectedCount, count, tt.description)
			}
		})
	}
}

func TestLambdaApplier_AnyWithContains(t *testing.T) {
	db := setupTestDB(t)
	entityMetadata := getTestProductMetadata()

	tests := []struct {
		name          string
		filter        string
		expectedCount int
	}{
		{
			name:          "any with contains function",
			filter:        "Descriptions/any(d: contains(d/Description, 'laptop'))",
			expectedCount: 1,
		},
		{
			name:          "any with contains - case sensitive",
			filter:        "Descriptions/any(d: contains(d/Description, 'Laptop'))",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, entityMetadata, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			query := db.Model(&TestProduct{})
			query = ApplyFilterOnly(query, filterExpr, entityMetadata, nil)

			var count int64
			if err := query.Count(&count).Error; err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if int(count) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, count)
			}
		})
	}
}

func TestLambdaApplier_MultipleAny(t *testing.T) {
	db := setupTestDB(t)
	entityMetadata := getTestProductMetadata()

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "multiple any with and",
			filter:        "Descriptions/any(d: d/LanguageKey eq 'EN') and Descriptions/any(d: d/LanguageKey eq 'DE')",
			expectedCount: 1,
			description:   "Only Laptop has both EN and DE",
		},
		{
			name:          "multiple any with or",
			filter:        "Descriptions/any(d: d/LanguageKey eq 'DE') or Descriptions/any(d: d/LanguageKey eq 'FR')",
			expectedCount: 2,
			description:   "Laptop has DE, Keyboard has FR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, entityMetadata, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			query := db.Model(&TestProduct{})
			query = ApplyFilterOnly(query, filterExpr, entityMetadata, nil)

			var count int64
			if err := query.Count(&count).Error; err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if int(count) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d. %s", tt.expectedCount, count, tt.description)
			}
		})
	}
}

func TestLambdaApplier_NotAny(t *testing.T) {
	db := setupTestDB(t)
	entityMetadata := getTestProductMetadata()

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "not any with specific language",
			filter:        "not Descriptions/any(d: d/LanguageKey eq 'XX')",
			expectedCount: 4,
			description:   "All products don't have XX language",
		},
		{
			name:          "not any with EN",
			filter:        "not Descriptions/any(d: d/LanguageKey eq 'EN')",
			expectedCount: 1,
			description:   "Only Monitor doesn't have EN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, entityMetadata, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			query := db.Model(&TestProduct{})
			query = ApplyFilterOnly(query, filterExpr, entityMetadata, nil)

			var count int64
			if err := query.Count(&count).Error; err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if int(count) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d. %s", tt.expectedCount, count, tt.description)
			}
		})
	}
}

func TestLambdaApplier_CombinedFilters(t *testing.T) {
	db := setupTestDB(t)
	entityMetadata := getTestProductMetadata()

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "entity property and lambda",
			filter:        "Price gt 50 and Descriptions/any(d: d/LanguageKey eq 'EN')",
			expectedCount: 2,
			description:   "Laptop and Keyboard have price > 50 and EN description",
		},
		{
			name:          "lambda with entity property or",
			filter:        "Price lt 100 or Descriptions/any(d: d/LanguageKey eq 'DE')",
			expectedCount: 3,
			description:   "Mouse, Keyboard have price < 100, and Laptop has DE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, entityMetadata, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			query := db.Model(&TestProduct{})
			query = ApplyFilterOnly(query, filterExpr, entityMetadata, nil)

			var count int64
			if err := query.Count(&count).Error; err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if int(count) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d. %s", tt.expectedCount, count, tt.description)
			}
		})
	}
}

func TestLambdaApplier_All(t *testing.T) {
	db := setupTestDB(t)
	entityMetadata := getTestProductMetadata()

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "all with inequality",
			filter:        "Descriptions/all(d: d/LanguageKey ne 'XX')",
			expectedCount: 4,
			description:   "All products satisfy: Monitor has no descriptions (vacuous truth), others have descriptions != 'XX'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, entityMetadata, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			query := db.Model(&TestProduct{})
			query = ApplyFilterOnly(query, filterExpr, entityMetadata, nil)

			var count int64
			if err := query.Count(&count).Error; err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if int(count) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d. %s", tt.expectedCount, count, tt.description)
			}
		})
	}
}

func TestLambdaApplier_ComplexPredicates(t *testing.T) {
	db := setupTestDB(t)
	entityMetadata := getTestProductMetadata()

	tests := []struct {
		name          string
		filter        string
		expectedCount int
	}{
		{
			name:          "any with complex and condition",
			filter:        "Descriptions/any(d: d/LanguageKey eq 'EN' and contains(d/Description, 'laptop'))",
			expectedCount: 1,
		},
		{
			name:          "any with or condition",
			filter:        "Descriptions/any(d: d/LanguageKey eq 'EN' or d/LanguageKey eq 'DE')",
			expectedCount: 4, // Monitor has no descriptions, so it won't match, but all others have EN or DE
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, entityMetadata, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			query := db.Model(&TestProduct{})
			query = ApplyFilterOnly(query, filterExpr, entityMetadata, nil)

			var count int64
			if err := query.Count(&count).Error; err != nil {
				t.Fatalf("Failed to execute query: %v", err)
			}

			if int(count) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, count)
			}
		})
	}
}

func TestLambdaApplier_CustomColumnAny(t *testing.T) {
	db := setupTestDB(t)
	entityMetadata := getTestProductMetadata()

	filterExpr, err := parseFilter("Descriptions/any(d: d/CustomName eq 'Promo')", entityMetadata, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	query := db.Model(&TestProduct{})
	query = ApplyFilterOnly(query, filterExpr, entityMetadata, nil)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 result, got %d", count)
	}
}

func TestLambdaApplier_CustomColumnRegistryLookup(t *testing.T) {
	db := setupTestDB(t)
	entityMetadata := getTestProductMetadata()

	filterExpr, err := parseFilter("Descriptions/any(d: d/Locale eq 'DE')", entityMetadata, nil, 0)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	query := db.Model(&TestProduct{})
	query = ApplyFilterOnly(query, filterExpr, entityMetadata, nil)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 result, got %d", count)
	}
}
