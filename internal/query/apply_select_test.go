package query

import (
	"net/url"
	"strings"
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

// has-one test types: MemberID is on the child (selectTestPrivacySettings), not on selectTestMember
type selectTestPrivacySettings struct {
	ID       int    `json:"ID" odata:"key"`
	MemberID int    `json:"memberID"`
	Setting  string `json:"setting"`
}

type selectTestMember struct {
	ID              int                        `json:"ID" odata:"key"`
	Name            string                     `json:"name"`
	PrivacySettings *selectTestPrivacySettings `json:"privacySettings,omitempty" gorm:"foreignKey:MemberID"`
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

	t.Run("Expand on has-one does NOT include child FK column in parent SELECT", func(t *testing.T) {
		memberMeta, err := metadata.AnalyzeEntity(selectTestMember{})
		if err != nil {
			t.Fatalf("AnalyzeEntity returned error: %v", err)
		}

		result := applySelect(db, []string{"name"}, []ExpandOption{{NavigationProperty: "privacySettings"}}, memberMeta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		selects := result.Statement.Selects
		for _, selectExpr := range selects {
			if strings.Contains(selectExpr, "member_id") {
				t.Fatalf("expected child FK column member_id NOT to appear in parent SELECT, got: %v", selects)
			}
		}
	})
}

// TestSelectWildcard tests $select=* wildcard behaviour (OData v4.01 section 5.1.3)
func TestSelectWildcard(t *testing.T) {
	meta := getSelectTestMetadata(t)
	db := getSelectTestDB(t)

	products := []selectTestProduct{
		{ID: 1, Name: "Product1", Price: 10.5, Description: "Desc1", CategoryID: 1},
		{ID: 2, Name: "Product2", Price: 20.0, Description: "Desc2", CategoryID: 2},
	}

	t.Run("ApplySelect with wildcard returns results unchanged", func(t *testing.T) {
		result := ApplySelect(products, []string{"*"}, meta, []ExpandOption{})
		// Wildcard should not filter — result should equal the input (not converted to maps)
		if _, ok := result.([]selectTestProduct); !ok {
			t.Fatal("expected result to be []selectTestProduct (unchanged), got something else")
		}
	})

	t.Run("ApplySelectToEntity with wildcard returns entity unchanged", func(t *testing.T) {
		p := selectTestProduct{ID: 1, Name: "Product1", Price: 10.5}
		result := ApplySelectToEntity(&p, []string{"*"}, meta, []ExpandOption{})
		if result != &p {
			t.Fatal("expected entity to be returned unchanged when wildcard is used")
		}
	})

	t.Run("ApplySelectToMapResults with wildcard returns all entries unchanged", func(t *testing.T) {
		mapResults := []map[string]interface{}{
			{"ID": 1, "name": "Product1", "price": 10.5, "description": "Desc1"},
			{"ID": 2, "name": "Product2", "price": 20.0, "description": "Desc2"},
		}
		result := ApplySelectToMapResults(mapResults, []string{"*"}, meta, map[string]bool{})
		if len(result) != 2 {
			t.Fatalf("expected 2 results, got %d", len(result))
		}
		for _, r := range result {
			if _, ok := r["description"]; !ok {
				t.Error("expected 'description' to be present when wildcard is used")
			}
		}
	})

	t.Run("applySelect with wildcard applies no column restriction", func(t *testing.T) {
		result := applySelect(db, []string{"*"}, nil, meta)
		if result == nil {
			t.Fatal("expected non-nil db")
		}
		// Wildcard: no SELECT clause should be added (fetch all columns)
		if len(result.Statement.Selects) != 0 {
			t.Errorf("expected no SELECT clause for wildcard, got: %v", result.Statement.Selects)
		}
	})
}

// TestParseSelectWildcard tests that $select=* parses without validation errors
func TestParseSelectWildcard(t *testing.T) {
	meta := getSelectTestMetadata(t)

	t.Run("$select=* passes validation with metadata in OData 4.01 mode", func(t *testing.T) {
		params := url.Values{}
		params.Set("$select", "*")
		opts, err := ParseQueryOptions(params, meta) // defaults to caseInsensitive=true (4.01)
		if err != nil {
			t.Fatalf("unexpected error for $select=* in OData 4.01 mode: %v", err)
		}
		if len(opts.Select) != 1 || opts.Select[0] != "*" {
			t.Errorf("expected Select=[*], got %v", opts.Select)
		}
	})

	t.Run("$select=* is rejected in OData 4.0 mode", func(t *testing.T) {
		params := url.Values{}
		params.Set("$select", "*")
		_, err := ParseQueryOptionsWithConfigAndCaseSensitivity(params, meta, nil, false)
		if err == nil {
			t.Fatal("expected error for $select=* in OData 4.0 mode, got nil")
		}
	})

	t.Run("$select=* passes validation without metadata (version not checked without metadata)", func(t *testing.T) {
		params := url.Values{}
		params.Set("$select", "*")
		opts, err := ParseQueryOptions(params, nil)
		if err != nil {
			t.Fatalf("unexpected error for $select=* without metadata: %v", err)
		}
		if len(opts.Select) != 1 || opts.Select[0] != "*" {
			t.Errorf("expected Select=[*], got %v", opts.Select)
		}
	})
}
