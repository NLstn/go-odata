package odata_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
)

func TestDeltaEntriesUseConfiguredNamespace(t *testing.T) {
	db := setupNamespaceDB(t)
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(&NamespaceProduct{}); err != nil {
		t.Fatalf("register entity: %v", err)
	}
	if err := service.SetNamespace("Contoso"); err != nil {
		t.Fatalf("set namespace: %v", err)
	}
	if err := service.EnableChangeTracking("NamespaceProducts"); err != nil {
		t.Fatalf("enable change tracking: %v", err)
	}

	initialReq := httptest.NewRequest(http.MethodGet, "/NamespaceProducts", nil)
	initialReq.Header.Set("Prefer", "odata.track-changes")
	initialRes := httptest.NewRecorder()
	service.ServeHTTP(initialRes, initialReq)
	if initialRes.Code != http.StatusOK {
		t.Fatalf("initial delta request failed: %d", initialRes.Code)
	}
	initialToken := extractDeltaToken(t, initialRes.Body.Bytes())

	createReq := httptest.NewRequest(http.MethodPost, "/NamespaceProducts", strings.NewReader(`{"ID":1,"Name":"Widget"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	service.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create request failed: %d", createRes.Code)
	}

	deltaReq := httptest.NewRequest(http.MethodGet, "/NamespaceProducts?$deltatoken="+url.QueryEscape(initialToken), nil)
	deltaReq.Header.Set("Accept", "application/json;odata.metadata=full")
	deltaRes := httptest.NewRecorder()
	service.ServeHTTP(deltaRes, deltaReq)
	if deltaRes.Code != http.StatusOK {
		t.Fatalf("delta request failed: %d", deltaRes.Code)
	}

	payload := decodeJSON(t, deltaRes.Body.Bytes())
	entries := valueEntries(t, payload)
	if len(entries) != 1 {
		t.Fatalf("expected 1 delta entry, got %d", len(entries))
	}

	entryType, ok := entries[0]["@odata.type"].(string)
	if !ok {
		t.Fatalf("delta entry missing @odata.type: %v", entries[0])
	}
	if entryType != "#Contoso.NamespaceProduct" {
		t.Fatalf("expected @odata.type #Contoso.NamespaceProduct, got %s", entryType)
	}
}
