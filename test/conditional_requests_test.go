package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ConditionalRequestEntity is a test entity for conditional request scenarios
type ConditionalRequestEntity struct {
	ID          int       `json:"id" gorm:"primarykey;autoIncrement" odata:"key"`
	Name        string    `json:"name" odata:"required"`
	Description string    `json:"description"`
	Version     int       `json:"version" odata:"etag"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func setupConditionalRequestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ConditionalRequestEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(ConditionalRequestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestConditionalRequests_CompleteFlow tests the complete conditional request flow per OData v4 spec
func TestConditionalRequests_CompleteFlow(t *testing.T) {
	service, db := setupConditionalRequestService(t)

	// Step 1: Create an entity
	entity := ConditionalRequestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "Original description",
		Version:     1,
		UpdatedAt:   time.Now(),
	}
	db.Create(&entity)

	// Step 2: GET entity and retrieve ETag
	req := httptest.NewRequest(http.MethodGet, "/ConditionalRequestEntities(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET failed: status = %v, want %v", w.Code, http.StatusOK)
	}

	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag header is missing in GET response")
	}
	t.Logf("Retrieved ETag: %s", etag)

	// Step 3: Conditional GET with If-None-Match (should return 304 since ETag matches)
	req = httptest.NewRequest(http.MethodGet, "/ConditionalRequestEntities(1)", nil)
	req.Header.Set("If-None-Match", etag)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotModified {
		t.Errorf("If-None-Match with matching ETag: status = %v, want %v", w.Code, http.StatusNotModified)
	}
	if w.Body.Len() > 0 {
		t.Error("304 response should have empty body")
	}
	t.Log("✓ If-None-Match with matching ETag returned 304")

	// Step 4: Conditional GET with If-None-Match using different ETag (should return 200)
	req = httptest.NewRequest(http.MethodGet, "/ConditionalRequestEntities(1)", nil)
	req.Header.Set("If-None-Match", "W/\"different-etag\"")
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("If-None-Match with non-matching ETag: status = %v, want %v", w.Code, http.StatusOK)
	}
	t.Log("✓ If-None-Match with non-matching ETag returned 200")

	// Step 5: Conditional PATCH with If-Match (should succeed)
	update := map[string]interface{}{
		"description": "Updated description",
	}
	body, _ := json.Marshal(update)

	req = httptest.NewRequest(http.MethodPatch, "/ConditionalRequestEntities(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", etag)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("If-Match with matching ETag for PATCH: status = %v, want 200 or 204", w.Code)
	}
	t.Log("✓ If-Match with matching ETag allowed PATCH")

	// Step 6: Manually increment version to simulate optimistic locking behavior
	// Note: Version auto-increment is application responsibility, not the OData library
	var entityToUpdate ConditionalRequestEntity
	db.First(&entityToUpdate, 1)
	entityToUpdate.Version++
	db.Save(&entityToUpdate)

	// Get new ETag after version increment
	req = httptest.NewRequest(http.MethodGet, "/ConditionalRequestEntities(1)", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)
	newEtag := w.Header().Get("ETag")

	if newEtag == etag {
		t.Error("ETag should change after version is incremented")
	}
	t.Logf("New ETag after version increment: %s", newEtag)

	// Step 7: Conditional PATCH with old ETag (should fail with 412)
	req = httptest.NewRequest(http.MethodPatch, "/ConditionalRequestEntities(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", etag) // Using old ETag
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusPreconditionFailed {
		t.Errorf("If-Match with non-matching ETag: status = %v, want %v", w.Code, http.StatusPreconditionFailed)
	}
	t.Log("✓ If-Match with non-matching ETag returned 412")

	// Step 8: Conditional PATCH with wildcard (should always succeed)
	req = httptest.NewRequest(http.MethodPatch, "/ConditionalRequestEntities(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "*")
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("If-Match with wildcard: status = %v, want 200 or 204", w.Code)
	}
	t.Log("✓ If-Match with wildcard (*) allowed PATCH")

	// Step 9: Verify the final update was successful
	var finalEntity ConditionalRequestEntity
	db.First(&finalEntity, 1)
	if finalEntity.Description != "Updated description" {
		t.Errorf("Description = %v, want 'Updated description'", finalEntity.Description)
	}
}

// TestConditionalRequests_IfNoneMatchWildcard tests If-None-Match with wildcard
func TestConditionalRequests_IfNoneMatchWildcard(t *testing.T) {
	service, db := setupConditionalRequestService(t)

	// Create an entity
	entity := ConditionalRequestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "Description",
		Version:     1,
		UpdatedAt:   time.Now(),
	}
	db.Create(&entity)

	// If-None-Match with wildcard should return 304 for existing entity
	req := httptest.NewRequest(http.MethodGet, "/ConditionalRequestEntities(1)", nil)
	req.Header.Set("If-None-Match", "*")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotModified {
		t.Errorf("If-None-Match with * for existing entity: status = %v, want %v", w.Code, http.StatusNotModified)
	}
}

// TestConditionalRequests_DeleteWithIfMatch tests DELETE with conditional headers
func TestConditionalRequests_DeleteWithIfMatch(t *testing.T) {
	service, db := setupConditionalRequestService(t)

	// Create an entity
	entity := ConditionalRequestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "To be deleted",
		Version:     1,
		UpdatedAt:   time.Now(),
	}
	db.Create(&entity)

	// Get ETag
	req := httptest.NewRequest(http.MethodGet, "/ConditionalRequestEntities(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	etag := w.Header().Get("ETag")

	// DELETE with matching If-Match should succeed
	req = httptest.NewRequest(http.MethodDelete, "/ConditionalRequestEntities(1)", nil)
	req.Header.Set("If-Match", etag)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("DELETE with matching If-Match: status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Verify entity is deleted
	var count int64
	db.Model(&ConditionalRequestEntity{}).Where("id = ?", 1).Count(&count)
	if count != 0 {
		t.Error("Entity should be deleted")
	}
}

// TestConditionalRequests_PutWithIfMatch tests PUT with conditional headers
func TestConditionalRequests_PutWithIfMatch(t *testing.T) {
	service, db := setupConditionalRequestService(t)

	// Create an entity
	entity := ConditionalRequestEntity{
		ID:          1,
		Name:        "Test Product",
		Description: "Original",
		Version:     1,
		UpdatedAt:   time.Now(),
	}
	db.Create(&entity)

	// Get ETag
	req := httptest.NewRequest(http.MethodGet, "/ConditionalRequestEntities(1)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	etag := w.Header().Get("ETag")

	// PUT with matching If-Match should succeed
	replacement := map[string]interface{}{
		"name":        "Replaced Product",
		"description": "Replaced description",
		"version":     1,
	}
	body, _ := json.Marshal(replacement)

	req = httptest.NewRequest(http.MethodPut, "/ConditionalRequestEntities(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", etag)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("PUT with matching If-Match: status = %v, want 200 or 204", w.Code)
	}

	// Verify replacement
	var updated ConditionalRequestEntity
	db.First(&updated, 1)
	if updated.Name != "Replaced Product" {
		t.Errorf("Name = %v, want 'Replaced Product'", updated.Name)
	}
	if updated.Description != "Replaced description" {
		t.Errorf("Description = %v, want 'Replaced description'", updated.Description)
	}

	// Manually increment version to test If-Match with stale ETag
	updated.Version++
	db.Save(&updated)

	// PUT with wrong If-Match should fail with 412
	req = httptest.NewRequest(http.MethodPut, "/ConditionalRequestEntities(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", etag) // Using old ETag
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusPreconditionFailed {
		t.Errorf("PUT with non-matching If-Match: status = %v, want %v", w.Code, http.StatusPreconditionFailed)
	}
}
