package odata_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// UserSession is a test entity with string UUID primary key to test DELETE with quoted keys
type UserSession struct {
	ID          string `json:"id" gorm:"primarykey" odata:"key"`
	UserID      string `json:"userId"`
	AccessToken string `json:"accessToken"`
}

// Global variables to track hook calls
// Note: These are safe in sequential test execution (Go's default).
// Tests using these variables reset them at the start of each test.
var (
	capturedSessionID string
	hookWasCalled     bool
)

// ODataBeforeDelete hook to capture the ID value
func (s UserSession) ODataBeforeDelete(ctx context.Context, r *http.Request) error {
	capturedSessionID = s.ID
	hookWasCalled = true
	return nil
}

func setupDeleteStringKeyTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&UserSession{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(UserSession{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestDeleteEntity_StringKeyWithQuotes_HookReceivesCleanKey(t *testing.T) {
	service, db := setupDeleteStringKeyTestService(t)

	// Reset global variables
	capturedSessionID = ""
	hookWasCalled = false

	// Insert test data with a UUID-like string key
	session := UserSession{
		ID:          "1f8d7b3b-3271-4ce2-8ea9-a875ad35bbd6",
		UserID:      "user123",
		AccessToken: "token-abc-123",
	}
	db.Create(&session)

	// Verify entity exists
	var count int64
	db.Model(&UserSession{}).Where("id = ?", "1f8d7b3b-3271-4ce2-8ea9-a875ad35bbd6").Count(&count)
	if count != 1 {
		t.Fatalf("Entity not created, count = %d", count)
	}

	// Delete the entity using OData syntax with quoted string key
	// This simulates: DELETE /UserSessions('1f8d7b3b-3271-4ce2-8ea9-a875ad35bbd6')
	req := httptest.NewRequest(http.MethodDelete, "/UserSessions('1f8d7b3b-3271-4ce2-8ea9-a875ad35bbd6')", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the hook was called
	if !hookWasCalled {
		t.Error("BeforeDelete hook was not called")
	}

	// IMPORTANT: Verify that the ID passed to the hook does NOT have surrounding quotes
	// This is the core bug being fixed
	expectedID := "1f8d7b3b-3271-4ce2-8ea9-a875ad35bbd6"
	if capturedSessionID != expectedID {
		t.Errorf("Hook received ID with quotes: got '%s', want '%s'", capturedSessionID, expectedID)
		if strings.HasPrefix(capturedSessionID, "'") && strings.HasSuffix(capturedSessionID, "'") {
			t.Error("BUG CONFIRMED: ID still contains surrounding single quotes")
		}
	}

	// Verify entity was deleted
	db.Model(&UserSession{}).Where("id = ?", "1f8d7b3b-3271-4ce2-8ea9-a875ad35bbd6").Count(&count)
	if count != 0 {
		t.Errorf("Entity not deleted, count = %d", count)
	}
}

func TestDeleteEntity_StringKeyWithoutQuotes_StillWorks(t *testing.T) {
	service, db := setupDeleteStringKeyTestService(t)

	// Reset global variables
	capturedSessionID = ""
	hookWasCalled = false

	// Insert test data
	session := UserSession{
		ID:          "simple-key",
		UserID:      "user456",
		AccessToken: "token-xyz-789",
	}
	db.Create(&session)

	// Delete the entity without quotes (numeric-style access)
	req := httptest.NewRequest(http.MethodDelete, "/UserSessions(simple-key)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the hook was called with the correct ID
	if !hookWasCalled {
		t.Error("BeforeDelete hook was not called")
	}

	expectedID := "simple-key"
	if capturedSessionID != expectedID {
		t.Errorf("Hook received incorrect ID: got '%s', want '%s'", capturedSessionID, expectedID)
	}

	// Verify entity was deleted
	var count int64
	db.Model(&UserSession{}).Where("id = ?", "simple-key").Count(&count)
	if count != 0 {
		t.Errorf("Entity not deleted, count = %d", count)
	}
}

func TestDeleteEntity_CompositeKeyWithQuotes_ValuesClean(t *testing.T) {
	// Test entity with composite key including a string
	type CompositeKeyEntity struct {
		ProductID   int    `json:"productID" gorm:"primaryKey" odata:"key"`
		LanguageKey string `json:"languageKey" gorm:"primaryKey" odata:"key"`
		Name        string `json:"name"`
	}

	// We test the URL parsing directly through a GET request to verify
	// the keys are parsed correctly without quotes

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&CompositeKeyEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(CompositeKeyEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	entity := CompositeKeyEntity{
		ProductID:   1,
		LanguageKey: "EN",
		Name:        "Product Name",
	}
	db.Create(&entity)

	// GET the entity to verify key parsing works correctly with quotes
	req := httptest.NewRequest(http.MethodGet, "/CompositeKeyEntities(productID=1,languageKey='EN')", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify the response contains the entity
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if productID, ok := response["productID"].(float64); !ok || int(productID) != 1 {
		t.Errorf("Expected productID 1, got %v", response["productID"])
	}

	if languageKey, ok := response["languageKey"].(string); !ok || languageKey != "EN" {
		t.Errorf("Expected languageKey 'EN', got %v", response["languageKey"])
		if languageKey == "'EN'" {
			t.Error("BUG: languageKey still contains quotes")
		}
	}
}
