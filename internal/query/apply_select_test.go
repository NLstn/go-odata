package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type selectTestCategory struct {
	ID   int    `json:"ID" odata:"key"`
	Name string `json:"name"`
}

type selectTestProduct struct {
	ID          int                 `json:"ID" odata:"key"`
	Name        string              `json:"name"`
	Price       float64             `json:"price"`
	Description string              `json:"description"`
	CategoryID  int                 `json:"categoryID"`
	Category    *selectTestCategory `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Tags        []selectTestTag     `json:"tags,omitempty" gorm:"foreignKey:ProductID"`
}

type selectTestTag struct {
	ID        int    `json:"ID" odata:"key"`
	Name      string `json:"name"`
	ProductID int    `json:"productID"`
}

func getSelectTestMetadata(t *testing.T) *metadata.EntityMetadata {
	t.Helper()
	meta, err := metadata.AnalyzeEntity(selectTestProduct{})
	if err != nil {
		t.Fatalf("AnalyzeEntity returned error: %v", err)
	}
	return meta
}

func getSelectTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	return db
}

func TestApplySelect(t *testing.T) {
	meta := getSelectTestMetadata(t)

	products := []selectTestProduct{
		{ID: 1, Name: "Product1", Price: 10.5, Description: "Desc1", CategoryID: 1},
		{ID: 2, Name: "Product2", Price: 20.0, Description: "Desc2", CategoryID: 2},
	}

	t.Run("Empty select returns results unchanged", func(t *testing.T) {
		result := ApplySelect(products, []string{}, meta, []ExpandOption{})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("Non-slice input returns results unchanged", func(t *testing.T) {
		notASlice := "not a slice"
		result := ApplySelect(notASlice, []string{"name"}, meta, []ExpandOption{})
		if result != notASlice {
			t.Fatal("expected result to be unchanged")
		}
	})

	t.Run("Select specific properties", func(t *testing.T) {
		result := ApplySelect(products, []string{"name", "price"}, meta, []ExpandOption{})
		maps, ok := result.([]map[string]interface{})
		if !ok {
			t.Fatal("expected result to be []map[string]interface{}")
		}

		if len(maps) != 2 {
			t.Fatalf("expected 2 results, got %d", len(maps))
		}

		// Should include selected properties and key properties
		firstProduct := maps[0]
		if _, ok := firstProduct["name"]; !ok {
			t.Error("expected 'name' to be in result")
		}
		if _, ok := firstProduct["price"]; !ok {
			t.Error("expected 'price' to be in result")
		}
		if _, ok := firstProduct["ID"]; !ok {
			t.Error("expected 'ID' (key) to be in result")
		}
		// Description should not be included
		if _, ok := firstProduct["description"]; ok {
			t.Error("expected 'description' to NOT be in result")
		}
	})

	t.Run("Select with navigation property path", func(t *testing.T) {
		productsWithNav := []selectTestProduct{
			{
				ID:   1,
				Name: "Product1",
				Category: &selectTestCategory{
					ID:   1,
					Name: "Category1",
				},
			},
		}

		expandOpts := []ExpandOption{
			{NavigationProperty: "category"},
		}

		result := ApplySelect(productsWithNav, []string{"name", "category/name"}, meta, expandOpts)
		maps, ok := result.([]map[string]interface{})
		if !ok {
			t.Fatal("expected result to be []map[string]interface{}")
		}

		if len(maps) != 1 {
			t.Fatalf("expected 1 result, got %d", len(maps))
		}

		firstProduct := maps[0]
		if _, ok := firstProduct["category"]; !ok {
			t.Error("expected 'category' to be in result")
		}
	})
}

func TestApplySelectToEntity(t *testing.T) {
	meta := getSelectTestMetadata(t)

	product := selectTestProduct{
		ID:          1,
		Name:        "Product1",
		Price:       10.5,
		Description: "Desc1",
		CategoryID:  1,
	}

	t.Run("Empty select returns entity unchanged", func(t *testing.T) {
		result := ApplySelectToEntity(&product, []string{}, meta, []ExpandOption{})
		if result != &product {
			t.Fatal("expected result to be unchanged")
		}
	})

	t.Run("Select specific properties on single entity", func(t *testing.T) {
		result := ApplySelectToEntity(&product, []string{"name", "price"}, meta, []ExpandOption{})
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("expected result to be map[string]interface{}")
		}

		// Should include selected properties and key properties
		if _, ok := resultMap["name"]; !ok {
			t.Error("expected 'name' to be in result")
		}
		if _, ok := resultMap["price"]; !ok {
			t.Error("expected 'price' to be in result")
		}
		if _, ok := resultMap["ID"]; !ok {
			t.Error("expected 'ID' (key) to be in result")
		}
		// Description should not be included
		if _, ok := resultMap["description"]; ok {
			t.Error("expected 'description' to NOT be in result")
		}
	})

	t.Run("Select with navigation property", func(t *testing.T) {
		productWithNav := selectTestProduct{
			ID:   1,
			Name: "Product1",
			Category: &selectTestCategory{
				ID:   1,
				Name: "Category1",
			},
		}

		expandOpts := []ExpandOption{
			{NavigationProperty: "category"},
		}

		result := ApplySelectToEntity(&productWithNav, []string{"name", "category"}, meta, expandOpts)
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("expected result to be map[string]interface{}")
		}

		if _, ok := resultMap["category"]; !ok {
			t.Error("expected 'category' to be in result")
		}
	})

	t.Run("Select with navigation property path", func(t *testing.T) {
		productWithNav := selectTestProduct{
			ID:   1,
			Name: "Product1",
			Category: &selectTestCategory{
				ID:   1,
				Name: "Category1",
			},
		}

		expandOpts := []ExpandOption{
			{NavigationProperty: "category"},
		}

		result := ApplySelectToEntity(&productWithNav, []string{"name", "category/name"}, meta, expandOpts)
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("expected result to be map[string]interface{}")
		}

		if _, ok := resultMap["category"]; !ok {
			t.Error("expected 'category' to be in result")
		}
	})
}

func TestApplySelectToMapResults(t *testing.T) {
	meta := getSelectTestMetadata(t)

	results := []map[string]interface{}{
		{
			"ID":          1,
			"name":        "Product1",
			"price":       10.5,
			"description": "Desc1",
		},
		{
			"ID":          2,
			"name":        "Product2",
			"price":       20.0,
			"description": "Desc2",
		},
	}

	t.Run("Empty select returns results unchanged", func(t *testing.T) {
		result := ApplySelectToMapResults(results, []string{}, meta, map[string]bool{})
		if len(result) != 2 {
			t.Fatalf("expected 2 results, got %d", len(result))
		}
	})

	t.Run("Select specific properties", func(t *testing.T) {
		result := ApplySelectToMapResults(results, []string{"name", "price"}, meta, map[string]bool{})

		if len(result) != 2 {
			t.Fatalf("expected 2 results, got %d", len(result))
		}

		firstProduct := result[0]
		// Should include selected properties and key properties
		if _, ok := firstProduct["name"]; !ok {
			t.Error("expected 'name' to be in result")
		}
		if _, ok := firstProduct["price"]; !ok {
			t.Error("expected 'price' to be in result")
		}
		if _, ok := firstProduct["ID"]; !ok {
			t.Error("expected 'ID' (key) to be in result")
		}
		// Description should not be included
		if _, ok := firstProduct["description"]; ok {
			t.Error("expected 'description' to NOT be in result")
		}
	})

	t.Run("Select only keys", func(t *testing.T) {
		result := ApplySelectToMapResults(results, []string{"name"}, meta, map[string]bool{})

		if len(result) != 2 {
			t.Fatalf("expected 2 results, got %d", len(result))
		}

		firstProduct := result[0]
		if _, ok := firstProduct["name"]; !ok {
			t.Error("expected 'name' to be in result")
		}
		if _, ok := firstProduct["ID"]; !ok {
			t.Error("expected 'ID' (key) to be in result")
		}
	})
}

func TestApplySelectDatabaseLevel(t *testing.T) {
	meta := getSelectTestMetadata(t)
	db := getSelectTestDB(t)

	t.Run("Empty select returns db unchanged", func(t *testing.T) {
		result := applySelect(db, []string{}, nil, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("Select specific properties modifies db", func(t *testing.T) {
		result := applySelect(db, []string{"name", "price"}, nil, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		// We can't easily test the internal state of GORM, but we verify it doesn't panic
	})

	t.Run("Select properties always includes key properties", func(t *testing.T) {
		result := applySelect(db, []string{"name"}, nil, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		// Key properties should be automatically included
	})

	t.Run("Expand on belongs-to includes foreign key column", func(t *testing.T) {
		result := applySelect(db, []string{"name"}, []ExpandOption{{NavigationProperty: "category"}}, meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		selects := result.Statement.Selects
		if len(selects) == 0 {
			t.Fatal("expected select clause to include columns")
		}

		foundFK := false
		for _, selectExpr := range selects {
			if selectExpr == "`select_test_products`.`category_id`" || selectExpr == "\"select_test_products\".\"category_id\"" {
				foundFK = true
				break
			}
		}

		if !foundFK {
			t.Fatalf("expected select clause to include category_id foreign key, got: %v", selects)
		}
	})
}
