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

type ComplexPropertyAddress struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

type ComplexPropertyProduct struct {
	ID              int                     `json:"id" gorm:"primarykey" odata:"key"`
	Name            string                  `json:"name"`
	ShippingAddress *ComplexPropertyAddress `json:"shippingAddress,omitempty" gorm:"embedded;embeddedPrefix:ship_" odata:"nullable"`
}

func setupComplexPropertyService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&ComplexPropertyProduct{}, &ComplexPropertyAddress{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(ComplexPropertyProduct{}); err != nil {
		t.Fatalf("failed to register entity: %v", err)
	}

	return service, db
}

func TestComplexPropertyGET(t *testing.T) {
	service, db := setupComplexPropertyService(t)

	product := ComplexPropertyProduct{
		ID:   1,
		Name: "Widget",
		ShippingAddress: &ComplexPropertyAddress{
			Street: "123 Main St",
			City:   "Metropolis",
		},
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("failed to insert product: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyProducts(1)/shippingAddress", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected application/json content type, got %s", contentType)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if context, ok := body["@odata.context"].(string); !ok || !strings.Contains(context, "shippingAddress") {
		t.Fatalf("expected context to reference shippingAddress, got %v", body["@odata.context"])
	}

	if body["street"] != "123 Main St" {
		t.Fatalf("expected street to be '123 Main St', got %v", body["street"])
	}
	if body["city"] != "Metropolis" {
		t.Fatalf("expected city to be 'Metropolis', got %v", body["city"])
	}
}

func TestComplexPropertyHEAD(t *testing.T) {
	service, db := setupComplexPropertyService(t)

	product := ComplexPropertyProduct{
		ID:   1,
		Name: "Widget",
		ShippingAddress: &ComplexPropertyAddress{
			Street: "123 Main St",
			City:   "Metropolis",
		},
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("failed to insert product: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/ComplexPropertyProducts(1)/shippingAddress", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if w.Body.Len() != 0 {
		t.Fatalf("expected empty body for HEAD request, got %q", w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected json content type, got %s", contentType)
	}
}

func TestComplexPropertyOPTIONS(t *testing.T) {
	service, _ := setupComplexPropertyService(t)

	req := httptest.NewRequest(http.MethodOptions, "/ComplexPropertyProducts(1)/shippingAddress", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	allow := w.Header().Get("Allow")
	if allow != "GET, HEAD, OPTIONS" {
		t.Fatalf("expected Allow header 'GET, HEAD, OPTIONS', got %s", allow)
	}
}

func TestComplexPropertyNestedGET(t *testing.T) {
	service, db := setupComplexPropertyService(t)

	product := ComplexPropertyProduct{
		ID:   1,
		Name: "Widget",
		ShippingAddress: &ComplexPropertyAddress{
			Street: "123 Main St",
			City:   "Metropolis",
		},
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("failed to insert product: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyProducts(1)/shippingAddress/city", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["value"] != "Metropolis" {
		t.Fatalf("expected value 'Metropolis', got %v", body["value"])
	}

	if context, ok := body["@odata.context"].(string); !ok || !strings.Contains(context, "shippingAddress/city") {
		t.Fatalf("expected context to include nested path, got %v", body["@odata.context"])
	}
}

func TestComplexPropertyNestedValue(t *testing.T) {
	service, db := setupComplexPropertyService(t)

	product := ComplexPropertyProduct{
		ID:   1,
		Name: "Widget",
		ShippingAddress: &ComplexPropertyAddress{
			Street: "123 Main St",
			City:   "Metropolis",
		},
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("failed to insert product: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyProducts(1)/shippingAddress/city/$value", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if body := strings.TrimSpace(w.Body.String()); body != "Metropolis" {
		t.Fatalf("expected raw body 'Metropolis', got %q", body)
	}

	if contentType := w.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		t.Fatalf("expected text/plain content type, got %s", contentType)
	}
}

func TestComplexPropertyNull(t *testing.T) {
	service, db := setupComplexPropertyService(t)

	product := ComplexPropertyProduct{
		ID:              1,
		Name:            "Widget",
		ShippingAddress: nil,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("failed to insert product: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ComplexPropertyProducts(1)/shippingAddress", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}
}
