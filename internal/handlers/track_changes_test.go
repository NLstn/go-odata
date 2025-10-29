package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandleCollectionTrackChanges(t *testing.T) {
	handler, db := setupTestHandler(t)

	if err := handler.EnableChangeTracking(); err != nil {
		t.Fatalf("failed to enable change tracking: %v", err)
	}

	if err := db.Create(&TestEntity{ID: 1, Name: "Initial"}).Error; err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/TestEntities", nil)
	req.Header.Set("Prefer", "odata.track-changes")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	applied := w.Header().Get(HeaderPreferenceApplied)
	if !strings.Contains(strings.ToLower(applied), "odata.track-changes") {
		t.Fatalf("expected Preference-Applied to include odata.track-changes, got %s", applied)
	}

	var initialBody map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &initialBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	deltaLinkValue, ok := initialBody["@odata.deltaLink"].(string)
	if !ok || deltaLinkValue == "" {
		t.Fatalf("expected delta link in response, got %v", initialBody["@odata.deltaLink"])
	}

	parsedLink, err := url.Parse(deltaLinkValue)
	if err != nil {
		t.Fatalf("failed to parse delta link: %v", err)
	}
	token := parsedLink.Query().Get("$deltatoken")
	if token == "" {
		t.Fatalf("expected delta token in delta link, got %s", deltaLinkValue)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/TestEntities", strings.NewReader(`{"id":2,"name":"Delta Created"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	handler.HandleCollection(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected 201 for create, got %d", createRes.Code)
	}

	deltaReq := httptest.NewRequest(http.MethodGet, "/TestEntities?$deltatoken="+token, nil)
	deltaRes := httptest.NewRecorder()
	handler.HandleCollection(deltaRes, deltaReq)
	if deltaRes.Code != http.StatusOK {
		t.Fatalf("expected 200 for delta response, got %d", deltaRes.Code)
	}

	var deltaBody map[string]interface{}
	if err := json.Unmarshal(deltaRes.Body.Bytes(), &deltaBody); err != nil {
		t.Fatalf("failed to decode delta response: %v", err)
	}

	valueArray, ok := deltaBody["value"].([]interface{})
	if !ok {
		t.Fatalf("expected value array in delta response")
	}
	if len(valueArray) == 0 {
		t.Fatalf("expected at least one change entry in delta response")
	}

	firstEntry, ok := valueArray[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map entries in delta response")
	}
	if firstEntry["name"] != "Delta Created" {
		t.Fatalf("expected created entity name, got %v", firstEntry["name"])
	}

	nextLink, ok := deltaBody["@odata.deltaLink"].(string)
	if !ok || nextLink == "" {
		t.Fatalf("expected next delta link, got %v", deltaBody["@odata.deltaLink"])
	}
	nextParsed, err := url.Parse(nextLink)
	if err != nil {
		t.Fatalf("failed to parse next delta link: %v", err)
	}
	nextToken := nextParsed.Query().Get("$deltatoken")
	if nextToken == "" {
		t.Fatalf("expected delta token in next delta link")
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/TestEntities(2)", nil)
	deleteRes := httptest.NewRecorder()
	handler.HandleEntity(deleteRes, deleteReq, "2")
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for delete, got %d", deleteRes.Code)
	}

	secondDeltaReq := httptest.NewRequest(http.MethodGet, "/TestEntities?$deltatoken="+nextToken, nil)
	secondDeltaRes := httptest.NewRecorder()
	handler.HandleCollection(secondDeltaRes, secondDeltaReq)
	if secondDeltaRes.Code != http.StatusOK {
		t.Fatalf("expected 200 for second delta response, got %d", secondDeltaRes.Code)
	}

	var secondBody map[string]interface{}
	if err := json.Unmarshal(secondDeltaRes.Body.Bytes(), &secondBody); err != nil {
		t.Fatalf("failed to decode second delta response: %v", err)
	}

	removedEntries, ok := secondBody["value"].([]interface{})
	if !ok {
		t.Fatalf("expected value array in second delta response")
	}
	if len(removedEntries) == 0 {
		t.Fatalf("expected removal entry in second delta response")
	}

	removalEntry, ok := removedEntries[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map entry for removal")
	}
	removedInfo, hasRemoved := removalEntry["@odata.removed"].(map[string]interface{})
	if !hasRemoved {
		t.Fatalf("expected @odata.removed in removal entry")
	}
	if removedInfo["reason"] != "deleted" {
		t.Fatalf("expected removal reason 'deleted', got %v", removedInfo["reason"])
	}
	if idValue, ok := removalEntry["id"].(float64); !ok || int(idValue) != 2 {
		t.Fatalf("expected id 2 in removal entry, got %v", removalEntry["id"])
	}
}

type trackChangesETagEntity struct {
	ID      int    `json:"id" gorm:"primarykey" odata:"key"`
	Version int    `json:"version" odata:"etag"`
	Name    string `json:"name"`
}

func setupTrackChangesHandlerWithETag(t *testing.T) (*EntityHandler, *gorm.DB, *metadata.EntityMetadata) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&trackChangesETagEntity{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(trackChangesETagEntity{})
	if err != nil {
		t.Fatalf("failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta)
	tracker := trackchanges.NewTracker()
	tracker.RegisterEntity(entityMeta.EntitySetName)
	handler.SetDeltaTracker(tracker)

	if err := handler.EnableChangeTracking(); err != nil {
		t.Fatalf("failed to enable change tracking: %v", err)
	}

	return handler, db, entityMeta
}

func TestHandleCollectionTrackChangesIncludesETag(t *testing.T) {
	handler, db, meta := setupTrackChangesHandlerWithETag(t)
	seed := trackChangesETagEntity{ID: 1, Name: "Initial", Version: 1}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	collectionPath := "/" + meta.EntitySetName
	req := httptest.NewRequest(http.MethodGet, collectionPath, nil)
	req.Header.Set("Prefer", "odata.track-changes")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var initialBody map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &initialBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	deltaLinkValue, ok := initialBody["@odata.deltaLink"].(string)
	if !ok || deltaLinkValue == "" {
		t.Fatalf("expected delta link in response, got %v", initialBody["@odata.deltaLink"])
	}

	parsedLink, err := url.Parse(deltaLinkValue)
	if err != nil {
		t.Fatalf("failed to parse delta link: %v", err)
	}
	token := parsedLink.Query().Get("$deltatoken")
	if token == "" {
		t.Fatalf("expected delta token in delta link")
	}

	patchPayload := `{"name":"Updated","version":2}`
	patchReq := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("%s(%d)", collectionPath, seed.ID), strings.NewReader(patchPayload))
	patchReq.Header.Set("Content-Type", "application/json")
	patchRes := httptest.NewRecorder()

	handler.HandleEntity(patchRes, patchReq, "1")
	if patchRes.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for patch update, got %d", patchRes.Code)
	}

	deltaReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("%s?$deltatoken=%s", collectionPath, token), nil)
	deltaRes := httptest.NewRecorder()

	handler.HandleCollection(deltaRes, deltaReq)
	if deltaRes.Code != http.StatusOK {
		t.Fatalf("expected 200 for delta response, got %d", deltaRes.Code)
	}

	var deltaBody map[string]interface{}
	if err := json.Unmarshal(deltaRes.Body.Bytes(), &deltaBody); err != nil {
		t.Fatalf("failed to decode delta response: %v", err)
	}

	entries, ok := deltaBody["value"].([]interface{})
	if !ok || len(entries) == 0 {
		t.Fatalf("expected at least one delta entry")
	}

	entry, ok := entries[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map entry in delta response")
	}

	etagValue, hasETag := entry["@odata.etag"].(string)
	if !hasETag || etagValue == "" {
		t.Fatalf("expected @odata.etag in delta entry, got %v", entry["@odata.etag"])
	}

	if _, hasID := entry["@odata.id"].(string); !hasID {
		t.Fatalf("expected @odata.id in delta entry")
	}

	expectedETag := etag.Generate(map[string]interface{}{"version": 2}, meta)
	if etagValue != expectedETag {
		t.Fatalf("expected etag %s, got %s", expectedETag, etagValue)
	}
}
