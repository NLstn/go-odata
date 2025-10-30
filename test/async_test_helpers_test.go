package odata_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
)

func enableAsyncProcessing(t *testing.T, service *odata.Service, retry time.Duration) {
	t.Helper()
	if err := service.EnableAsyncProcessing(odata.AsyncConfig{
		MonitorPathPrefix:    "/$async/jobs/",
		DefaultRetryInterval: retry,
		JobRetention:         0,
	}); err != nil {
		t.Fatalf("failed to enable async processing: %v", err)
	}
}

func waitForMonitorCompletion(t *testing.T, service *odata.Service, location string) *httptest.ResponseRecorder {
	t.Helper()
	if location == "" {
		t.Fatalf("monitor location must not be empty")
	}

	const attempts = 50
	for i := 0; i < attempts; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, location, nil)
		service.ServeHTTP(rec, req)

		if rec.Code == http.StatusAccepted {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		return rec
	}

	t.Fatalf("monitor %s did not reach terminal state after %d attempts", location, attempts)
	return nil
}

func issueMonitorRequest(t *testing.T, service *odata.Service, method, location string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, location, nil)
	service.ServeHTTP(rec, req)
	return rec
}
