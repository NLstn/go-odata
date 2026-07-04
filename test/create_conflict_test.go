package odata_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ConflictTestDescription has a composite primary key (non-numeric, so it is never
// mistaken for a database-generated/auto-increment key) to isolate the duplicate-key
// conflict behavior under test from key-generation concerns.
type ConflictTestDescription struct {
	ProductCode string `json:"ProductCode" gorm:"primaryKey" odata:"key"`
	LanguageKey string `json:"LanguageKey" gorm:"primaryKey" odata:"key"`
	Description string `json:"Description"`
}

func setupCreateConflictTest(t *testing.T) *odata.Service {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ConflictTestDescription{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	db.Create(&ConflictTestDescription{ProductCode: "P1", LanguageKey: "EN", Description: "A portable computer"})

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&ConflictTestDescription{}); err != nil {
		t.Fatalf("RegisterEntity() error: %v", err)
	}

	return service
}

// TestCreateWithDuplicateCompositeKeyReturnsConflict tests that POSTing an entity whose
// composite key already exists returns 409 Conflict rather than a generic 500 Internal
// Server Error (OData JSON Format §5.1.2: services MUST respond with 409 Conflict if the
// entity already exists).
func TestCreateWithDuplicateCompositeKeyReturnsConflict(t *testing.T) {
	service := setupCreateConflictTest(t)

	body := []byte(`{"ProductCode": "P1", "LanguageKey": "EN", "Description": "Duplicate"}`)
	req := httptest.NewRequest(http.MethodPost, "/ConflictTestDescriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate composite key, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCreateWithNewCompositeKeySucceeds tests that POSTing an entity with a composite key
// that doesn't already exist still succeeds normally.
func TestCreateWithNewCompositeKeySucceeds(t *testing.T) {
	service := setupCreateConflictTest(t)

	body := []byte(`{"ProductCode": "P1", "LanguageKey": "FR", "Description": "Un ordinateur portable"}`)
	req := httptest.NewRequest(http.MethodPost, "/ConflictTestDescriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 for new composite key, got %d: %s", w.Code, w.Body.String())
	}
}
