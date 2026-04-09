package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type KeyAsSegmentsProduct struct {
	ID    int    `json:"ID" gorm:"primarykey" odata:"key"`
	Name  string `json:"Name"`
	Price int    `json:"Price"`
}

func setupKeyAsSegmentsService(t *testing.T) (*odata.Service, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&KeyAsSegmentsProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(KeyAsSegmentsProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestKeyAsSegments_EntityAccess verifies that /EntitySet/key resolves to the same
// entity as /EntitySet(key) when the client negotiates OData 4.01.
func TestKeyAsSegments_EntityAccess(t *testing.T) {
	service, db := setupKeyAsSegmentsService(t)
	db.Create(&KeyAsSegmentsProduct{ID: 1, Name: "Widget", Price: 100})

	// Parenthetical key (baseline)
	req1 := httptest.NewRequest(http.MethodGet, "/KeyAsSegmentsProducts(1)", nil)
	w1 := httptest.NewRecorder()
	service.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("parenthetical key: expected 200, got %d", w1.Code)
	}

	// Key-as-segment with OData-MaxVersion: 4.01
	req2 := httptest.NewRequest(http.MethodGet, "/KeyAsSegmentsProducts/1", nil)
	req2.Header.Set("OData-MaxVersion", "4.01")
	w2 := httptest.NewRecorder()
	service.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("key-as-segment: expected 200, got %d", w2.Code)
	}

	// Both responses should contain the same entity
	var r1, r2 map[string]interface{}
	if err := json.Unmarshal(w1.Body.Bytes(), &r1); err != nil {
		t.Fatalf("failed to parse parenthetical response: %v", err)
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &r2); err != nil {
		t.Fatalf("failed to parse key-as-segment response: %v", err)
	}

	if r1["ID"] != r2["ID"] {
		t.Errorf("responses differ: parenthetical ID=%v, key-as-segment ID=%v", r1["ID"], r2["ID"])
	}
	if r1["Name"] != r2["Name"] {
		t.Errorf("responses differ: parenthetical Name=%v, key-as-segment Name=%v", r1["Name"], r2["Name"])
	}
}

// TestKeyAsSegments_NotFoundEntity verifies that a non-existent key still returns 404
// under key-as-segments.
func TestKeyAsSegments_NotFoundEntity(t *testing.T) {
	service, _ := setupKeyAsSegmentsService(t)

	req := httptest.NewRequest(http.MethodGet, "/KeyAsSegmentsProducts/9999", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent entity, got %d", w.Code)
	}
}

// TestKeyAsSegments_NotActiveUnder40 verifies that key-as-segments does NOT apply
// when the client negotiates OData 4.0.
func TestKeyAsSegments_NotActiveUnder40(t *testing.T) {
	service, db := setupKeyAsSegmentsService(t)
	db.Create(&KeyAsSegmentsProduct{ID: 1, Name: "Widget", Price: 100})

	req := httptest.NewRequest(http.MethodGet, "/KeyAsSegmentsProducts/1", nil)
	req.Header.Set("OData-MaxVersion", "4.0")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	// Under OData 4.0, "1" is interpreted as a property/navigation segment, not a key.
	// Since "1" is not a valid property, the service should return 404.
	if w.Code != http.StatusNotFound {
		t.Fatalf("under OData 4.0: expected 404 (key-as-segments not active), got %d", w.Code)
	}
}

// TestKeyAsSegments_PropertyAccess verifies that /EntitySet/key/PropertyName works
// under OData 4.01 key-as-segments.
func TestKeyAsSegments_PropertyAccess(t *testing.T) {
	service, db := setupKeyAsSegmentsService(t)
	db.Create(&KeyAsSegmentsProduct{ID: 1, Name: "Widget", Price: 100})

	req := httptest.NewRequest(http.MethodGet, "/KeyAsSegmentsProducts/1/Name", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("key-as-segment property access: expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := result["value"]; !ok {
		t.Errorf("expected 'value' field in property response, got %v", result)
	}
}
