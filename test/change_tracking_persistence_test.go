package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type PersistentProduct struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
}

func TestChangeTrackingPersistsAcrossRestart(t *testing.T) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "tracker.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&PersistentProduct{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{PersistentChangeTracking: true})
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	if err := service.RegisterEntity(&PersistentProduct{}); err != nil {
		t.Fatalf("register entity: %v", err)
	}
	if err := service.EnableChangeTracking("PersistentProducts"); err != nil {
		t.Fatalf("enable change tracking: %v", err)
	}

	initialReq := httptest.NewRequest(http.MethodGet, "/PersistentProducts", nil)
	initialReq.Header.Set("Prefer", "odata.track-changes")
	initialRes := httptest.NewRecorder()
	service.ServeHTTP(initialRes, initialReq)
	if initialRes.Code != http.StatusOK {
		t.Fatalf("initial response status: %d", initialRes.Code)
	}
	initialToken := extractDeltaToken(t, initialRes.Body.Bytes())

	createReq := httptest.NewRequest(http.MethodPost, "/PersistentProducts", strings.NewReader(`{"id":1,"name":"Persisted"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	service.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status: %d", createRes.Code)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	dbRestarted, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("reconnect database: %v", err)
	}

	serviceRestarted, err := odata.NewServiceWithConfig(dbRestarted, odata.ServiceConfig{PersistentChangeTracking: true})
	if err != nil {
		t.Fatalf("recreate service: %v", err)
	}
	if err := dbRestarted.AutoMigrate(&PersistentProduct{}); err != nil {
		t.Fatalf("remigrate: %v", err)
	}
	if err := serviceRestarted.RegisterEntity(&PersistentProduct{}); err != nil {
		t.Fatalf("register entity after restart: %v", err)
	}
	if err := serviceRestarted.EnableChangeTracking("PersistentProducts"); err != nil {
		t.Fatalf("enable change tracking after restart: %v", err)
	}

	deltaReq := httptest.NewRequest(http.MethodGet, "/PersistentProducts?$deltatoken="+url.QueryEscape(initialToken), nil)
	deltaRes := httptest.NewRecorder()
	serviceRestarted.ServeHTTP(deltaRes, deltaReq)
	if deltaRes.Code != http.StatusOK {
		t.Fatalf("delta response status: %d", deltaRes.Code)
	}
	deltaBody := decodeJSON(t, deltaRes.Body.Bytes())
	changes := valueEntries(t, deltaBody)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change after restart, got %d", len(changes))
	}
	if id, ok := changes[0]["id"].(float64); !ok || int(id) != 1 {
		t.Fatalf("expected persisted entity id 1, got %v", changes[0]["id"])
	}
	nextToken := extractDeltaToken(t, deltaRes.Body.Bytes())

	createAfterReq := httptest.NewRequest(http.MethodPost, "/PersistentProducts", strings.NewReader(`{"id":2,"name":"Restarted"}`))
	createAfterReq.Header.Set("Content-Type", "application/json")
	createAfterRes := httptest.NewRecorder()
	serviceRestarted.ServeHTTP(createAfterRes, createAfterReq)
	if createAfterRes.Code != http.StatusCreated {
		t.Fatalf("create after restart status: %d", createAfterRes.Code)
	}

	followReq := httptest.NewRequest(http.MethodGet, "/PersistentProducts?$deltatoken="+url.QueryEscape(nextToken), nil)
	followRes := httptest.NewRecorder()
	serviceRestarted.ServeHTTP(followRes, followReq)
	if followRes.Code != http.StatusOK {
		t.Fatalf("follow-up delta status: %d", followRes.Code)
	}
	followBody := decodeJSON(t, followRes.Body.Bytes())
	followChanges := valueEntries(t, followBody)
	if len(followChanges) != 1 {
		t.Fatalf("expected 1 change after restart write, got %d", len(followChanges))
	}
	if id, ok := followChanges[0]["id"].(float64); !ok || int(id) != 2 {
		t.Fatalf("expected new entity id 2, got %v", followChanges[0]["id"])
	}
}

func extractDeltaToken(t *testing.T, body []byte) string {
	t.Helper()

	payload := decodeJSON(t, body)
	link, ok := payload["@odata.deltaLink"].(string)
	if !ok || link == "" {
		t.Fatalf("delta link missing: %v", payload["@odata.deltaLink"])
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("parse delta link: %v", err)
	}
	token := parsed.Query().Get("$deltatoken")
	if token == "" {
		t.Fatalf("delta token missing in link: %s", link)
	}
	return token
}

func decodeJSON(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return payload
}

func valueEntries(t *testing.T, payload map[string]interface{}) []map[string]interface{} {
	t.Helper()

	raw, ok := payload["value"].([]interface{})
	if !ok {
		t.Fatalf("missing value array in payload")
	}
	result := make([]map[string]interface{}, 0, len(raw))
	for _, entry := range raw {
		item, ok := entry.(map[string]interface{})
		if !ok {
			t.Fatalf("value entry is not object: %T", entry)
		}
		result = append(result, item)
	}
	return result
}
