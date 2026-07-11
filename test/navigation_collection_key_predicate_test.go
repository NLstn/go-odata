package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Entities used to reproduce issue #786: addressing a single entity through a
// collection-valued navigation property with a key predicate, e.g.
// Categories(1)/Products(1).
type NavKeyPredicateCategory struct {
	ID       uint                     `json:"ID" gorm:"primarykey" odata:"key"`
	Name     string                   `json:"Name"`
	Products []NavKeyPredicateProduct `json:"Products" gorm:"foreignKey:CategoryID"`
}

type NavKeyPredicateProduct struct {
	ID         uint   `json:"ID" gorm:"primarykey" odata:"key"`
	Name       string `json:"Name"`
	CategoryID *uint  `json:"CategoryID"`
}

func setupNavKeyPredicateService(t *testing.T) (*odata.Service, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&NavKeyPredicateCategory{}, &NavKeyPredicateProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&NavKeyPredicateCategory{}); err != nil {
		t.Fatalf("Failed to register NavKeyPredicateCategory: %v", err)
	}
	if err := service.RegisterEntity(&NavKeyPredicateProduct{}); err != nil {
		t.Fatalf("Failed to register NavKeyPredicateProduct: %v", err)
	}

	categoryID := uint(1)
	db.Create(&NavKeyPredicateCategory{ID: categoryID, Name: "Electronics"})
	db.Create(&NavKeyPredicateProduct{ID: 1, Name: "Mouse", CategoryID: &categoryID})
	db.Create(&NavKeyPredicateProduct{ID: 2, Name: "Keyboard", CategoryID: &categoryID})
	// A product that does not belong to the category above, to exercise the
	// "not related to parent" 404 case.
	db.Create(&NavKeyPredicateProduct{ID: 3, Name: "Orphan"})

	return service, db
}

// assertSingleNavEntity checks that body represents a single-entity JSON object (not a
// collection wrapped in "value": [...]) whose @odata.context ends in "/$entity", per OData
// v4.0 Part 2 §4.11.
func assertSingleNavEntity(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse response: %v (body: %s)", err, body)
	}

	if _, ok := result["value"]; ok {
		t.Fatalf("expected a single-entity object, got a collection wrapped in \"value\": %s", body)
	}

	contextURL, _ := result["@odata.context"].(string)
	if !strings.HasSuffix(contextURL, "/$entity") {
		t.Errorf("expected @odata.context to end with '/$entity', got %q", contextURL)
	}

	return result
}

// TestNavigationCollectionKeyPredicate_ParentheticalForm covers the primary regression from
// issue #786: GET /Categories(id)/Products(id) must return the Product entity directly, not a
// collection response.
func TestNavigationCollectionKeyPredicate_ParentheticalForm(t *testing.T) {
	service, _ := setupNavKeyPredicateService(t)

	req := httptest.NewRequest(http.MethodGet, "/NavKeyPredicateCategories(1)/Products(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	entity := assertSingleNavEntity(t, w.Body.Bytes())
	if entity["ID"] != float64(1) {
		t.Errorf("expected ID 1, got %v", entity["ID"])
	}
	if entity["Name"] != "Mouse" {
		t.Errorf("expected Name 'Mouse', got %v", entity["Name"])
	}
}

// TestNavigationCollectionKeyPredicate_KeyAsSegments covers the second regression from issue
// #786: under OData 4.01 key-as-segments negotiation, GET /Categories/id/Products/id must
// resolve the same way as the parenthetical form instead of crashing with a 500.
func TestNavigationCollectionKeyPredicate_KeyAsSegments(t *testing.T) {
	service, _ := setupNavKeyPredicateService(t)

	req := httptest.NewRequest(http.MethodGet, "/NavKeyPredicateCategories/1/Products/1", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	entity := assertSingleNavEntity(t, w.Body.Bytes())
	if entity["ID"] != float64(1) {
		t.Errorf("expected ID 1, got %v", entity["ID"])
	}
	if entity["Name"] != "Mouse" {
		t.Errorf("expected Name 'Mouse', got %v", entity["Name"])
	}
}

// TestNavigationCollectionKeyPredicate_KeyAsSegmentsActiveUnder40 verifies that
// response negotiation does not disable the key-as-segments request convention.
func TestNavigationCollectionKeyPredicate_KeyAsSegmentsActiveUnder40(t *testing.T) {
	service, _ := setupNavKeyPredicateService(t)

	req := httptest.NewRequest(http.MethodGet, "/NavKeyPredicateCategories/1/Products/1", nil)
	req.Header.Set("OData-MaxVersion", "4.0")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with OData-MaxVersion 4.0, got %d: %s", w.Code, w.Body.String())
	}
	if got := exactHeaderValues(w.Header(), "OData-Version"); len(got) != 1 || got[0] != "4.0" {
		t.Fatalf("OData-Version = %q, want 4.0", got)
	}
}

// TestNavigationCollectionKeyPredicate_NotFound verifies that a key that does not belong to the
// parent's collection still 404s, for both URL forms.
func TestNavigationCollectionKeyPredicate_NotFound(t *testing.T) {
	service, _ := setupNavKeyPredicateService(t)

	t.Run("parenthetical", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/NavKeyPredicateCategories(1)/Products(999)", nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for nonexistent key, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not related to parent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/NavKeyPredicateCategories(1)/Products(3)", nil)
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for a key not related to the parent, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("key-as-segments", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/NavKeyPredicateCategories/1/Products/999", nil)
		req.Header.Set("OData-MaxVersion", "4.01")
		w := httptest.NewRecorder()
		service.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for nonexistent key under key-as-segments, got %d: %s", w.Code, w.Body.String())
		}
	})
}
