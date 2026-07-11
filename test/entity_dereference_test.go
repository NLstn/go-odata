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

type EntityDereferenceProduct struct {
	ID   int    `json:"ID" gorm:"primarykey" odata:"key"`
	Name string `json:"Name"`
}

func TestEntityDereferenceViaCanonicalID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&EntityDereferenceProduct{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	if err := db.Create(&EntityDereferenceProduct{ID: 1, Name: "Widget"}).Error; err != nil {
		t.Fatalf("seed database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := service.RegisterEntity(EntityDereferenceProduct{}); err != nil {
		t.Fatalf("RegisterEntity: %v", err)
	}

	canonicalID := "http://example.com/EntityDereferenceProducts(1)"
	req := httptest.NewRequest(http.MethodGet, "/$entity?$id="+url.QueryEscape(canonicalID), nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /$entity returned %d: %s", w.Code, w.Body.String())
	}

	var entity map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&entity); err != nil {
		t.Fatalf("decode entity: %v", err)
	}
	if entity["ID"] != float64(1) || entity["Name"] != "Widget" {
		t.Fatalf("dereferenced entity = %v, want ID=1 and Name=Widget", entity)
	}
}

func TestEntityDereferenceRequiresID(t *testing.T) {
	service := setupErrorTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/$entity", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("GET /$entity without $id returned %d, want 400", w.Code)
	}
}
