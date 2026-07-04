package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NestedLambdaGroup/Item/Tag form a three-level navigation chain used to exercise
// nested lambda filters, i.e. Groups/any(x: x/Items/any(y: y/... )).
type NestedLambdaGroup struct {
	ID    uint               `json:"ID" gorm:"primaryKey" odata:"key"`
	Name  string             `json:"Name"`
	Items []NestedLambdaItem `json:"Items" gorm:"foreignKey:GroupID"`
}

type NestedLambdaItem struct {
	ID      uint              `json:"ID" gorm:"primaryKey" odata:"key"`
	GroupID uint              `json:"GroupID"`
	Name    string            `json:"Name"`
	Tags    []NestedLambdaTag `json:"Tags" gorm:"foreignKey:ItemID"`
}

type NestedLambdaTag struct {
	ID     uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	ItemID uint   `json:"ItemID"`
	Label  string `json:"Label"`
}

func setupLambdaNestedTest(t *testing.T) *odata.Service {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&NestedLambdaGroup{}, &NestedLambdaItem{}, &NestedLambdaTag{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Group 1 -> Item 1 -> Tag "target"
	db.Create(&NestedLambdaGroup{ID: 1, Name: "Group1"})
	db.Create(&NestedLambdaItem{ID: 1, GroupID: 1, Name: "Item1"})
	db.Create(&NestedLambdaTag{ID: 1, ItemID: 1, Label: "target"})

	// Group 2 -> Item 2 -> Tag "other" (no tag matches "target")
	db.Create(&NestedLambdaGroup{ID: 2, Name: "Group2"})
	db.Create(&NestedLambdaItem{ID: 2, GroupID: 2, Name: "Item2"})
	db.Create(&NestedLambdaTag{ID: 2, ItemID: 2, Label: "other"})

	// Group 3 has no items at all.
	db.Create(&NestedLambdaGroup{ID: 3, Name: "Group3"})

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&NestedLambdaGroup{}); err != nil {
		t.Fatalf("Failed to register NestedLambdaGroup entity: %v", err)
	}
	if err := service.RegisterEntity(&NestedLambdaItem{}); err != nil {
		t.Fatalf("Failed to register NestedLambdaItem entity: %v", err)
	}
	if err := service.RegisterEntity(&NestedLambdaTag{}); err != nil {
		t.Fatalf("Failed to register NestedLambdaTag entity: %v", err)
	}

	return service
}

func queryNestedLambdaGroupIDs(t *testing.T, service *odata.Service, filterQuery string) []float64 {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/NestedLambdaGroups?$filter="+url.QueryEscape(filterQuery), nil)
	rec := httptest.NewRecorder()
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Response missing 'value' array")
	}

	ids := make([]float64, 0, len(value))
	for _, item := range value {
		group := item.(map[string]interface{})
		ids = append(ids, group["ID"].(float64))
	}
	return ids
}

// TestNestedLambdaAny_MatchesOnlyDeepMatchingGroup reproduces #770: a two-level nested
// any() (Items/any(i: i/Tags/any(t: t/Label eq 'target'))) must only match groups whose
// items actually satisfy the innermost predicate, not silently pass through every group.
func TestNestedLambdaAny_MatchesOnlyDeepMatchingGroup(t *testing.T) {
	service := setupLambdaNestedTest(t)

	ids := queryNestedLambdaGroupIDs(t, service,
		"Items/any(i: i/Tags/any(t: t/Label eq 'target'))")

	if len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("Expected only Group 1, got %v", ids)
	}
}

// TestNestedLambdaAny_NoMatchIsVacuouslyFalse reproduces the exact scenario in #770:
// when nothing in the graph satisfies the innermost predicate, the filter must exclude
// every group (vacuously false), not return all of them.
func TestNestedLambdaAny_NoMatchIsVacuouslyFalse(t *testing.T) {
	service := setupLambdaNestedTest(t)

	ids := queryNestedLambdaGroupIDs(t, service,
		"Items/any(i: i/Tags/any(t: t/Label eq 'nonexistent'))")

	if len(ids) != 0 {
		t.Fatalf("Expected no groups to match, got %v", ids)
	}
}
