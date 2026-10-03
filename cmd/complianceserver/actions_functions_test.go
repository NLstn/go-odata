package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"github.com/nlstn/go-odata/cmd/complianceserver/entities"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApplyDiscountReturnsEntityAndPersistsPrice(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := seedDatabase(db); err != nil {
		t.Fatal(err)
	}
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetNamespace("ComplianceService"); err != nil {
		t.Fatal(err)
	}
	if err := service.RegisterEntity(&entities.Product{}); err != nil {
		t.Fatal(err)
	}
	registerActions(service, db)
	var product entities.Product
	if err := db.First(&product).Error; err != nil {
		t.Fatal(err)
	}
	wantPrice := product.Price * 0.9
	r := httptest.NewRequest(http.MethodPost, "/Products("+product.ID.String()+")/ComplianceService.ApplyDiscount", strings.NewReader(`{"percentage":10}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["value"] != nil || body["ID"] != product.ID.String() || body["Price"] != wantPrice || body["@odata.context"] == nil {
		t.Fatalf("expected updated entity at the root: %s", w.Body.String())
	}
	var updated entities.Product
	if err := db.First(&updated, "id = ?", product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Price != wantPrice {
		t.Fatalf("persisted price %f, want %f", updated.Price, wantPrice)
	}
}
