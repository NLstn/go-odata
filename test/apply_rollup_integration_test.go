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

func getRollupValue(t *testing.T, row map[string]interface{}, keys ...string) interface{} {
	t.Helper()
	for _, k := range keys {
		if v, ok := row[k]; ok {
			return v
		}
	}
	return nil
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

// Ensure getRollupValue doesn't cause unused warning
var _ = getRollupValue
