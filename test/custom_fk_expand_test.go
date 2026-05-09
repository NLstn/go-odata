package odata_test

// Tests for:
// Bug A: Collection expand with nested $top returns null
// Bug B: Direct navigation collection query uses wrong foreign key column

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// CustomFKSuite is a parent entity with a navigation property whose FK field name
// differs from the default convention (<EntityName><KeyName>).
// Here the FK is "SuiteID" not "CustomFKSuiteID", which triggers Bug B if not fixed.
type CustomFKSuite struct {
	ID   string         `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string         `json:"Name"`
	Runs []CustomFKRun  `json:"Runs,omitempty" gorm:"foreignKey:SuiteID;references:ID" odata:"nav"`
}

// CustomFKRun is a child entity whose FK column is "suite_id" (from field SuiteID),
// NOT "custom_fk_suite_id" (the default convention-based name).
type CustomFKRun struct {
	ID      string `json:"ID" gorm:"primaryKey" odata:"key"`
	SuiteID string `json:"SuiteID"`
	Name    string `json:"Name"`
	Order   int    `json:"Order"`
}

func setupCustomFKTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&CustomFKSuite{}, &CustomFKRun{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	suites := []CustomFKSuite{
		{ID: "suite-1", Name: "Suite One"},
		{ID: "suite-2", Name: "Suite Two"},
	}
	for _, s := range suites {
		db.Create(&s)
	}

	runs := []CustomFKRun{
		{ID: "run-1a", SuiteID: "suite-1", Name: "Run 1A", Order: 3},
		{ID: "run-1b", SuiteID: "suite-1", Name: "Run 1B", Order: 1},
		{ID: "run-1c", SuiteID: "suite-1", Name: "Run 1C", Order: 2},
		{ID: "run-2a", SuiteID: "suite-2", Name: "Run 2A", Order: 1},
	}
	for _, r := range runs {
		db.Create(&r)
	}

	return db
}

func setupCustomFKService(t *testing.T, db *gorm.DB) *odata.Service {
	t.Helper()

	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := svc.RegisterEntity(&CustomFKSuite{}); err != nil {
		t.Fatalf("RegisterEntity(CustomFKSuite) error: %v", err)
	}
	if err := svc.RegisterEntity(&CustomFKRun{}); err != nil {
		t.Fatalf("RegisterEntity(CustomFKRun) error: %v", err)
	}

	return svc
}

// TestCustomFKDirectNavigation verifies that a direct navigation collection query
// (e.g. /CustomFKSuites('suite-1')/Runs?$top=1) correctly uses the configured
// GORM foreignKey ("SuiteID" → column "suite_id") rather than the convention-based
// name derived from <EntityName><KeyName> ("custom_fk_suite_id").  Before the fix
// this would return HTTP 500 with a "column does not exist" SQL error.
func TestCustomFKDirectNavigation(t *testing.T) {
	db := setupCustomFKTestDB(t)
	svc := setupCustomFKService(t, db)

	tests := []struct {
		name          string
		url           string
		wantStatus    int
		wantResultLen int
	}{
		{
			name:          "direct navigation without query options",
			url:           "/CustomFKSuites('suite-1')/Runs",
			wantStatus:    http.StatusOK,
			wantResultLen: 3,
		},
		{
			name:          "direct navigation with $top",
			url:           "/CustomFKSuites('suite-1')/Runs?$top=1",
			wantStatus:    http.StatusOK,
			wantResultLen: 1,
		},
		{
			name:          "direct navigation with $orderby",
			url:           "/CustomFKSuites('suite-1')/Runs?$orderby=Order%20asc",
			wantStatus:    http.StatusOK,
			wantResultLen: 3,
		},
		{
			name:          "direct navigation with $filter",
			url:           "/CustomFKSuites('suite-1')/Runs?$filter=Name%20eq%20'Run%201A'",
			wantStatus:    http.StatusOK,
			wantResultLen: 1,
		},
		{
			name:          "direct navigation with combined query options",
			url:           "/CustomFKSuites('suite-1')/Runs?$orderby=Order%20asc&$top=2",
			wantStatus:    http.StatusOK,
			wantResultLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			svc.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			values, ok := response["value"].([]interface{})
			if !ok {
				t.Fatalf("expected array in 'value', got %T", response["value"])
			}

			if len(values) != tt.wantResultLen {
				t.Errorf("expected %d results, got %d: %s", tt.wantResultLen, len(values), w.Body.String())
			}
		})
	}
}

// TestCustomFKExpandWithTop verifies that $expand with nested $top on a collection
// navigation property whose FK differs from the default convention returns the correct
// (non-null) data.  Before the fix, Runs would be null for each suite.
func TestCustomFKExpandWithTop(t *testing.T) {
	db := setupCustomFKTestDB(t)
	svc := setupCustomFKService(t, db)

	t.Run("expand collection with $top=1 on single entity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/CustomFKSuites('suite-1')?$expand=Runs($top=1)", nil)
		w := httptest.NewRecorder()
		svc.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		runs, ok := response["Runs"].([]interface{})
		if !ok {
			t.Fatalf("expected Runs to be an array, got %T (%v)", response["Runs"], response["Runs"])
		}
		if len(runs) != 1 {
			t.Errorf("expected 1 run (due to $top=1), got %d", len(runs))
		}
	})

	t.Run("expand collection with $top=2 on single entity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/CustomFKSuites('suite-1')?$expand=Runs($top=2)", nil)
		w := httptest.NewRecorder()
		svc.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		runs, ok := response["Runs"].([]interface{})
		if !ok {
			t.Fatalf("expected Runs to be an array, got %T", response["Runs"])
		}
		if len(runs) != 2 {
			t.Errorf("expected 2 runs (due to $top=2), got %d", len(runs))
		}
	})

	t.Run("expand collection with $top on collection endpoint populates each parent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/CustomFKSuites?$expand=Runs($top=1)", nil)
		w := httptest.NewRecorder()
		svc.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		value, ok := response["value"].([]interface{})
		if !ok {
			t.Fatal("expected 'value' array in response")
		}
		if len(value) != 2 {
			t.Fatalf("expected 2 suites, got %d", len(value))
		}

		for i, v := range value {
			suite := v.(map[string]interface{})
			runs, ok := suite["Runs"].([]interface{})
			if !ok {
				t.Errorf("suite[%d] Runs should be an array, got %T (%v)", i, suite["Runs"], suite["Runs"])
				continue
			}
			if len(runs) != 1 {
				t.Errorf("suite[%d] expected 1 run (due to $top=1), got %d", i, len(runs))
			}
		}
	})

	t.Run("expand collection with $top and $orderby returns correct items", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet,
			"/CustomFKSuites('suite-1')?$expand=Runs($orderby=Order%20asc;$top=1)", nil)
		w := httptest.NewRecorder()
		svc.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		runs, ok := response["Runs"].([]interface{})
		if !ok {
			t.Fatalf("expected Runs to be an array, got %T", response["Runs"])
		}
		if len(runs) != 1 {
			t.Fatalf("expected 1 run, got %d", len(runs))
		}

		run := runs[0].(map[string]interface{})
		if run["Name"] != "Run 1B" {
			t.Errorf("expected the run with lowest Order (Run 1B), got %v", run["Name"])
		}
	})
}

// TestNestedExpandWithTopOnCustomFK is the most important regression test.
// It reproduces the scenario from the bug report:
//
//	$expand=OuterNav($expand=InnerCollection($top=1))
//
// where OuterNav has no $top/$skip itself (so it goes through GORM preload),
// but InnerCollection has $top=1 (which previously caused it to be skipped).
func TestNestedExpandWithTopOnCustomFK(t *testing.T) {
	// Use independent entities to build a three-level hierarchy System → Suite → Run
	// where the inner navigation FK name differs from the default convention.
	type NSRun struct {
		ID      string `json:"ID" gorm:"primaryKey" odata:"key"`
		SuiteID string `json:"SuiteID"`
		Name    string `json:"Name"`
		Order   int    `json:"Order"`
	}
	type NSSuite struct {
		ID       string  `json:"ID" gorm:"primaryKey" odata:"key"`
		SystemID string  `json:"SystemID"`
		Name     string  `json:"Name"`
		Runs     []NSRun `json:"Runs,omitempty" gorm:"foreignKey:SuiteID;references:ID" odata:"nav"`
	}
	type NSSystem struct {
		ID     string    `json:"ID" gorm:"primaryKey" odata:"key"`
		Name   string    `json:"Name"`
		Suites []NSSuite `json:"Suites,omitempty" gorm:"foreignKey:SystemID;references:ID" odata:"nav"`
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&NSSystem{}, &NSSuite{}, &NSRun{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	systems := []NSSystem{
		{ID: "sys-1", Name: "System One"},
	}
	suites := []NSSuite{
		{ID: "ns-suite-1", SystemID: "sys-1", Name: "Suite A"},
		{ID: "ns-suite-2", SystemID: "sys-1", Name: "Suite B"},
	}
	runs := []NSRun{
		{ID: "ns-run-1a", SuiteID: "ns-suite-1", Name: "Run 1A", Order: 2},
		{ID: "ns-run-1b", SuiteID: "ns-suite-1", Name: "Run 1B", Order: 1},
		{ID: "ns-run-2a", SuiteID: "ns-suite-2", Name: "Run 2A", Order: 1},
	}
	db.Create(&systems)
	db.Create(&suites)
	db.Create(&runs)

	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.RegisterEntity(&NSSystem{}); err != nil {
		t.Fatalf("register NSSystem: %v", err)
	}
	if err := svc.RegisterEntity(&NSSuite{}); err != nil {
		t.Fatalf("register NSSuite: %v", err)
	}
	if err := svc.RegisterEntity(&NSRun{}); err != nil {
		t.Fatalf("register NSRun: %v", err)
	}

	t.Run("three-level expand: Suites(no $top) -> Runs($top=1)", func(t *testing.T) {
		// This is the core bug: Suites has no $top so it goes via GORM preload;
		// Runs has $top=1 which was previously skipped, yielding null Runs.
		req := httptest.NewRequest(http.MethodGet,
			"/NSSystems('sys-1')?$expand=Suites($expand=Runs($top=1))", nil)
		w := httptest.NewRecorder()
		svc.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("parse: %v", err)
		}

		suitesList, ok := response["Suites"].([]interface{})
		if !ok {
			t.Fatalf("Suites should be an array, got %T", response["Suites"])
		}
		if len(suitesList) != 2 {
			t.Fatalf("expected 2 suites, got %d", len(suitesList))
		}

		for i, sv := range suitesList {
			suite := sv.(map[string]interface{})
			runsVal, hasRuns := suite["Runs"]
			if !hasRuns {
				t.Errorf("suite[%d] missing Runs key", i)
				continue
			}
			runsList, ok := runsVal.([]interface{})
			if !ok {
				// null or wrong type – this is the bug
				t.Errorf("suite[%d] Runs should be an array, got %T (%v)", i, runsVal, runsVal)
				continue
			}
			if len(runsList) != 1 {
				t.Errorf("suite[%d] expected 1 run ($top=1), got %d", i, len(runsList))
			}
		}
	})

	t.Run("three-level expand on collection: Systems -> Suites -> Runs($top=1)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet,
			"/NSSystems?$expand=Suites($expand=Runs($top=1))", nil)
		w := httptest.NewRecorder()
		svc.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("parse: %v", err)
		}

		systemsList, ok := response["value"].([]interface{})
		if !ok {
			t.Fatalf("expected 'value' array, got %T", response["value"])
		}

		for si, sv := range systemsList {
			system := sv.(map[string]interface{})
			suitesList, ok := system["Suites"].([]interface{})
			if !ok {
				t.Errorf("system[%d] Suites should be an array, got %T", si, system["Suites"])
				continue
			}
			for i, sv2 := range suitesList {
				suite := sv2.(map[string]interface{})
				runsVal, hasRuns := suite["Runs"]
				if !hasRuns {
					t.Errorf("system[%d] suite[%d] missing Runs key", si, i)
					continue
				}
				runsList, ok := runsVal.([]interface{})
				if !ok {
					t.Errorf("system[%d] suite[%d] Runs should be an array, got %T (%v)", si, i, runsVal, runsVal)
					continue
				}
				if len(runsList) != 1 {
					t.Errorf("system[%d] suite[%d] expected 1 run ($top=1), got %d", si, i, len(runsList))
				}
			}
		}
	})

	t.Run("direct navigation on suite returns correct runs with custom FK", func(t *testing.T) {
		// Ensure the direct navigation path also works for NSSuite->NSRun (custom FK "SuiteID").
		url := fmt.Sprintf("/NSSuites('%s')/Runs?$orderby=Order%%20asc&$top=1", "ns-suite-1")
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		svc.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("parse: %v", err)
		}

		values, ok := response["value"].([]interface{})
		if !ok {
			t.Fatalf("expected value array, got %T", response["value"])
		}
		if len(values) != 1 {
			t.Fatalf("expected 1 run, got %d", len(values))
		}
		run := values[0].(map[string]interface{})
		if run["Name"] != "Run 1B" {
			t.Errorf("expected Run 1B (lowest Order), got %v", run["Name"])
		}
	})
}
