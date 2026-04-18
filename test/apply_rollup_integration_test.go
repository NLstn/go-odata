package odata_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// RollupSale represents a sales record used in rollup integration tests.
type RollupSale struct {
	ID     uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Region string  `json:"Region"`
	Month  string  `json:"Month"`
	Amount float64 `json:"Amount"`
}

// setupRollupService creates an OData service with sample sales data for rollup tests.
func setupRollupService(t *testing.T) *httptest.Server {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if err := db.AutoMigrate(&RollupSale{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	sales := []RollupSale{
		{ID: 1, Region: "West", Month: "Jan", Amount: 100},
		{ID: 2, Region: "West", Month: "Feb", Amount: 150},
		{ID: 3, Region: "East", Month: "Jan", Amount: 200},
		{ID: 4, Region: "East", Month: "Feb", Amount: 250},
	}
	if err := db.Create(&sales).Error; err != nil {
		t.Fatalf("Failed to create sales: %v", err)
	}

	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = svc.RegisterEntity(&RollupSale{})

	return httptest.NewServer(svc)
}

func getJSONBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nbody: %s", err, body)
	}
	return result
}

func TestIntegrationApplyRollup(t *testing.T) {
	srv := setupRollupService(t)
	defer srv.Close()

	t.Run("rollup single property with null includes grand total", func(t *testing.T) {
		// groupby((rollup(null,Region)), aggregate(Amount with sum as Total))
		// Expected rows:
		//   {Region: "West", Total: 250}
		//   {Region: "East", Total: 450}
		//   {Region: nil,    Total: 700}  ← grand total
		url := srv.URL + "/RollupSales?$apply=groupby((rollup(null,Region)),aggregate(Amount%20with%20sum%20as%20Total))"
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := getJSONBody(t, resp)
		values, ok := result["value"].([]interface{})
		if !ok {
			t.Fatalf("expected 'value' array, got: %T", result["value"])
		}

		if len(values) != 3 {
			t.Fatalf("expected 3 rows (West, East, grand total), got %d: %v", len(values), values)
		}

		// Verify grand total row exists (Region is null)
		var grandTotalFound bool
		var grandTotalAmount float64
		for _, v := range values {
			row := v.(map[string]interface{})
			if row["Region"] == nil {
				grandTotalFound = true
				grandTotalAmount = toFloat64Interface(row["Total"])
			}
		}
		if !grandTotalFound {
			t.Error("expected grand total row with Region=null")
		}
		if grandTotalAmount != 700 {
			t.Errorf("expected grand total Amount=700, got %v", grandTotalAmount)
		}
	})

	t.Run("rollup single property without null excludes grand total", func(t *testing.T) {
		// groupby((rollup(Region)), aggregate(Amount with sum as Total))
		// Expected rows:
		//   {Region: "West", Total: 250}
		//   {Region: "East", Total: 450}
		//   (no grand total)
		url := srv.URL + "/RollupSales?$apply=groupby((rollup(Region)),aggregate(Amount%20with%20sum%20as%20Total))"
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := getJSONBody(t, resp)
		values, ok := result["value"].([]interface{})
		if !ok {
			t.Fatalf("expected 'value' array, got: %T", result["value"])
		}

		if len(values) != 2 {
			t.Fatalf("expected 2 rows (West, East), got %d: %v", len(values), values)
		}
		for _, v := range values {
			row := v.(map[string]interface{})
			if row["Region"] == nil {
				t.Error("unexpected grand total row (rollup without null should not include grand total)")
			}
		}
	})

	t.Run("rollup two properties with null produces three levels", func(t *testing.T) {
		// groupby((rollup(null,Region,Month)), aggregate(Amount with sum as Total))
		// Levels:
		//   (Region, Month): 4 rows
		//   (Region, nil):   2 rows (West total=250, East total=450)
		//   (nil, nil):      1 grand total row (700)
		url := srv.URL + "/RollupSales?$apply=groupby((rollup(null,Region,Month)),aggregate(Amount%20with%20sum%20as%20Total))"
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := getJSONBody(t, resp)
		values, ok := result["value"].([]interface{})
		if !ok {
			t.Fatalf("expected 'value' array, got: %T", result["value"])
		}

		if len(values) != 7 {
			t.Fatalf("expected 7 rows (4 fine-grain + 2 region subtotals + 1 grand total), got %d: %v", len(values), values)
		}

		// Find grand total (both Region and Month are null)
		var grandTotalFound bool
		for _, v := range values {
			row := v.(map[string]interface{})
			if row["Region"] == nil && row["Month"] == nil {
				grandTotalFound = true
				total := toFloat64Interface(row["Total"])
				if total != 700 {
					t.Errorf("expected grand total=700, got %v", total)
				}
			}
		}
		if !grandTotalFound {
			t.Error("expected grand total row with Region=nil and Month=nil")
		}
	})

	t.Run("rollup with preceding filter", func(t *testing.T) {
		// filter(Region eq 'West')/groupby((rollup(null,Region)),aggregate(Amount with sum as Total))
		// Expected: only West rows, plus grand total for West (250)
		url := srv.URL + "/RollupSales?$apply=filter(Region%20eq%20'West')/groupby((rollup(null,Region)),aggregate(Amount%20with%20sum%20as%20Total))"
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := getJSONBody(t, resp)
		values, ok := result["value"].([]interface{})
		if !ok {
			t.Fatalf("expected 'value' array, got: %T", result["value"])
		}

		// 1 region row (West=250) + 1 grand total (250)
		if len(values) != 2 {
			t.Fatalf("expected 2 rows after filtering West, got %d: %v", len(values), values)
		}
	})
}

// toFloat64Interface converts a JSON number value to float64 for assertions.
func toFloat64Interface(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

// RollupProduct mirrors the dev-server Product structure: it has a nullable FK
// (CategoryID *uint) whose SQL column name is "category_id" (snake_case), while
// the OData/JSON name is "CategoryID" (PascalCase). This case exposed the bug
// where getNestedMapValueCaseInsensitive would fail to match snake_case keys.
type RollupProduct struct {
	ID         uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name       string  `json:"Name"`
	Price      float64 `json:"Price"`
	CategoryID *uint   `json:"CategoryID" odata:"nullable"`
}

func setupRollupProductService(t *testing.T) *httptest.Server {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if err := db.AutoMigrate(&RollupProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	cat1 := uint(1)
	cat2 := uint(2)
	cat3 := uint(3)
	products := []RollupProduct{
		{ID: 1, Name: "A", Price: 100, CategoryID: &cat1},
		{ID: 2, Name: "B", Price: 200, CategoryID: &cat1},
		{ID: 3, Name: "C", Price: 300, CategoryID: &cat1},
		{ID: 4, Name: "D", Price: 400, CategoryID: &cat2},
		{ID: 5, Name: "E", Price: 500, CategoryID: &cat2},
		{ID: 6, Name: "F", Price: 600, CategoryID: &cat3},
		{ID: 7, Name: "G", Price: 700, CategoryID: &cat3},
	}
	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to create products: %v", err)
	}

	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	_ = svc.RegisterEntity(&RollupProduct{})

	return httptest.NewServer(svc)
}

// TestIntegrationApplyRollupNullableFKProperty verifies rollup works correctly
// when the groupby property is a nullable FK column whose SQL name is snake_case
// (e.g. category_id) but OData/JSON name is PascalCase (e.g. CategoryID).
// This is the scenario that was failing in the compliance tests.
func TestIntegrationApplyRollupNullableFKProperty(t *testing.T) {
	srv := setupRollupProductService(t)
	defer srv.Close()

	t.Run("rollup with null includes grand total, CategoryID is nullable FK", func(t *testing.T) {
		// 3 categories × totals + 1 grand total = 4 rows
		url := srv.URL + "/RollupProducts?$apply=groupby((rollup(null,CategoryID)),aggregate(Price%20with%20sum%20as%20Total))"
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := getJSONBody(t, resp)
		values, ok := result["value"].([]interface{})
		if !ok {
			t.Fatalf("expected 'value' array, got: %T", result["value"])
		}

		if len(values) != 4 {
			t.Fatalf("expected 4 rows (3 categories + grand total), got %d: %v", len(values), values)
		}

		var grandTotalFound bool
		for _, v := range values {
			row := v.(map[string]interface{})
			if row["CategoryID"] == nil {
				grandTotalFound = true
				total := toFloat64Interface(row["Total"])
				if total != 2800 {
					t.Errorf("expected grand total=2800, got %v", total)
				}
			}
		}
		if !grandTotalFound {
			t.Error("expected grand total row with CategoryID=null")
		}
	})

	t.Run("rollup without null does not include grand total, CategoryID is nullable FK", func(t *testing.T) {
		// 3 category rows, no grand total
		url := srv.URL + "/RollupProducts?$apply=groupby((rollup(CategoryID)),aggregate(Price%20with%20sum%20as%20Total))"
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := getJSONBody(t, resp)
		values, ok := result["value"].([]interface{})
		if !ok {
			t.Fatalf("expected 'value' array, got: %T", result["value"])
		}

		if len(values) != 3 {
			t.Fatalf("expected 3 rows (one per category, no grand total), got %d: %v", len(values), values)
		}
		for _, v := range values {
			row := v.(map[string]interface{})
			if row["CategoryID"] == nil {
				t.Error("unexpected grand total row (rollup without null should not include grand total)")
			}
		}
	})
}
