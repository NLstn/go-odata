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

type ComplexQueryAddress struct {
	Street string `json:"Street"`
	City   string `json:"City"`
}

type ComplexQueryDimensions struct {
	Length float64 `json:"Length"`
}

type ComplexQueryProduct struct {
	ID              uint                    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name            string                  `json:"Name"`
	ShippingAddress *ComplexQueryAddress    `json:"ShippingAddress,omitempty" gorm:"embedded;embeddedPrefix:shipping_" odata:"nullable"`
	Dimensions      *ComplexQueryDimensions `json:"Dimensions,omitempty" gorm:"embedded;embeddedPrefix:dim_" odata:"nullable"`
}

func setupComplexQueryService(t *testing.T) (*odata.Service, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&ComplexQueryProduct{}, &ComplexQueryAddress{}, &ComplexQueryDimensions{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(ComplexQueryProduct{}); err != nil {
		t.Fatalf("failed to register entity: %v", err)
	}

	return service, db
}

func seedComplexQueryProducts(t *testing.T, db *gorm.DB) {
	t.Helper()

	products := []ComplexQueryProduct{
		{
			ID:   1,
			Name: "Laptop",
			ShippingAddress: &ComplexQueryAddress{
				Street: "123 Tech Way",
				City:   "Seattle",
			},
			Dimensions: &ComplexQueryDimensions{
				Length: 35.5,
			},
		},
		{
			ID:   2,
			Name: "Tablet",
			ShippingAddress: &ComplexQueryAddress{
				Street: "456 Market St",
				City:   "Portland",
			},
			Dimensions: &ComplexQueryDimensions{
				Length: 25.0,
			},
		},
	}

	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("failed to seed products: %v", err)
	}
}

func TestFilterByComplexNestedProperty(t *testing.T) {
	service, db := setupComplexQueryService(t)
	seedComplexQueryProducts(t, db)

	filter := url.QueryEscape("ShippingAddress/City eq 'Seattle'")
	req := httptest.NewRequest(http.MethodGet, "/ComplexQueryProducts?%24filter="+filter, nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(body.Value) != 1 {
		t.Fatalf("expected 1 product in response, got %d", len(body.Value))
	}

	if name, ok := body.Value[0]["Name"].(string); !ok || name != "Laptop" {
		t.Fatalf("expected Laptop result, got %v", body.Value[0]["Name"])
	}
}

func TestOrderByComplexNestedProperty(t *testing.T) {
	service, db := setupComplexQueryService(t)
	seedComplexQueryProducts(t, db)

	order := url.QueryEscape("Dimensions/Length desc")
	req := httptest.NewRequest(http.MethodGet, "/ComplexQueryProducts?%24orderby="+order, nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(body.Value) < 2 {
		t.Fatalf("expected at least 2 products in response, got %d", len(body.Value))
	}

	firstID, ok := body.Value[0]["ID"].(float64)
	if !ok {
		t.Fatalf("expected numeric ID in first result, got %T", body.Value[0]["ID"])
	}
	if int(firstID) != 1 {
		t.Fatalf("expected first result to have ID 1, got %d", int(firstID))
	}
}
