package odata_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MaxLengthTestProduct declares a MaxLength=10 facet on Name, matching how the CSDL
// document advertises `<Property Name="Name" Type="Edm.String" MaxLength="10" />`.
type MaxLengthTestProduct struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name" gorm:"size:10" odata:"maxlength=10"`
}

func setupMaxLengthTest(t *testing.T) *odata.Service {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&MaxLengthTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&MaxLengthTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service
}

// TestMaxLength_PostRejectsOversizedString reproduces #768: a string value longer than
// the declared MaxLength facet must be rejected with 400 Bad Request, per OData CSDL
// §6.2.3 (MaxLength) and Protocol §11.4.2 (400 for a request body invalid per the model).
func TestMaxLength_PostRejectsOversizedString(t *testing.T) {
	service := setupMaxLengthTest(t)

	body := []byte(`{"Name": "ThisNameIsWayTooLong"}`)
	req := httptest.NewRequest(http.MethodPost, "/MaxLengthTestProducts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for oversized Name, got %d: %s", w.Code, w.Body.String())
	}
}

// TestMaxLength_PostAllowsStringWithinLimit tests that a string at or under the
// MaxLength limit is still accepted.
func TestMaxLength_PostAllowsStringWithinLimit(t *testing.T) {
	service := setupMaxLengthTest(t)

	body := []byte(`{"Name": "ExactlyTen"}`) // 10 characters
	req := httptest.NewRequest(http.MethodPost, "/MaxLengthTestProducts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 for Name within MaxLength, got %d: %s", w.Code, w.Body.String())
	}
}

// TestMaxLength_PatchRejectsOversizedString tests that a PATCH updating a property to
// exceed its MaxLength is also rejected.
func TestMaxLength_PatchRejectsOversizedString(t *testing.T) {
	service := setupMaxLengthTest(t)

	createBody := []byte(`{"Name": "Short"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/MaxLengthTestProducts", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	service.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("setup: expected 201 creating product, got %d: %s", createW.Code, createW.Body.String())
	}

	location := createW.Header().Get("Location")
	if location == "" {
		t.Fatal("setup: missing Location header")
	}
	path := location[strings.Index(location, "/MaxLengthTestProducts"):]

	patchBody := []byte(`{"Name": "ThisNameIsWayTooLong"}`)
	patchReq := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	service.ServeHTTP(patchW, patchReq)

	if patchW.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for oversized Name on PATCH, got %d: %s", patchW.Code, patchW.Body.String())
	}
}
