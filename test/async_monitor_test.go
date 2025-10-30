package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestAsyncMonitorGetMatchesSynchronousResponse(t *testing.T) {
	service, db := setupPreferTestService(t)
	enableAsyncProcessing(t, service, 2*time.Second)

	existing := []PreferTestProduct{
		{Name: "Monitor One", Price: 10},
		{Name: "Monitor Two", Price: 20},
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("failed to seed products: %v", err)
	}

	asyncReq := httptest.NewRequest(http.MethodGet, "/PreferTestProducts?$orderby=Name", nil)
	asyncReq.Header.Set("Prefer", "respond-async")
	asyncRec := httptest.NewRecorder()
	service.ServeHTTP(asyncRec, asyncReq)

	if asyncRec.Code != http.StatusAccepted {
		t.Fatalf("expected async acknowledgement, got %d", asyncRec.Code)
	}

	expected := httptest.NewRecorder()
	service.ServeHTTP(expected, httptest.NewRequest(http.MethodGet, "/PreferTestProducts?$orderby=Name", nil))

	monitorLocation := asyncRec.Header().Get("Location")
	if monitorLocation == "" {
		t.Fatal("missing monitor Location header")
	}

	monitorRec := waitForMonitorCompletion(t, service, monitorLocation)

	if monitorRec.Code != expected.Code {
		t.Fatalf("monitor status %d, want %d", monitorRec.Code, expected.Code)
	}

	if ct := monitorRec.Header().Get("Content-Type"); ct != expected.Header().Get("Content-Type") {
		t.Fatalf("monitor content type %q, want %q", ct, expected.Header().Get("Content-Type"))
	}

	var expectedBody map[string]any
	if err := json.NewDecoder(expected.Body).Decode(&expectedBody); err != nil {
		t.Fatalf("failed to decode expected body: %v", err)
	}

	var actualBody map[string]any
	if err := json.NewDecoder(monitorRec.Body).Decode(&actualBody); err != nil {
		t.Fatalf("failed to decode monitor body: %v", err)
	}

	if !reflect.DeepEqual(actualBody, expectedBody) {
		t.Fatalf("monitor payload mismatch: got %v, want %v", actualBody, expectedBody)
	}
}

func TestAsyncMonitorDeleteRemovesCompletedJob(t *testing.T) {
	service, _ := setupPreferTestService(t)
	enableAsyncProcessing(t, service, time.Second)

	payload := map[string]any{
		"name":  "Delete Monitor",
		"price": 77.0,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/PreferTestProducts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "respond-async")

	rec := httptest.NewRecorder()
	service.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected async acknowledgement, got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("missing monitor location")
	}

	monitorRec := waitForMonitorCompletion(t, service, location)
	if monitorRec.Code == http.StatusAccepted {
		t.Fatalf("monitor still pending after wait")
	}

	deleteRec := issueMonitorRequest(t, service, http.MethodDelete, location)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete monitor status %d, want %d", deleteRec.Code, http.StatusNoContent)
	}

	followUp := issueMonitorRequest(t, service, http.MethodGet, location)
	if followUp.Code != monitorRec.Code {
		t.Fatalf("expected persisted monitor status %d, got %d", monitorRec.Code, followUp.Code)
	}

	if !bytes.Equal(followUp.Body.Bytes(), monitorRec.Body.Bytes()) {
		t.Fatalf("expected persisted monitor body %q, got %q", monitorRec.Body.String(), followUp.Body.String())
	}
}
