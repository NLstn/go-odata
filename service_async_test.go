package odata

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type asyncTestEntity struct {
	ID   uint `gorm:"primaryKey" odata:"key"`
	Name string
}

func TestServiceRespondAsyncFlow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&asyncTestEntity{}); err != nil {
		t.Fatalf("failed to migrate test entity: %v", err)
	}

	svc := NewService(db)
	if err := svc.RegisterEntity(&asyncTestEntity{}); err != nil {
		t.Fatalf("failed to register entity: %v", err)
	}

	if err := svc.EnableAsyncProcessing(AsyncConfig{
		MonitorPathPrefix:    "/$async/jobs",
		DefaultRetryInterval: 3 * time.Second,
		JobRetention:         time.Minute,
	}); err != nil {
		t.Fatalf("failed to enable async processing: %v", err)
	}
	if svc.asyncManager != nil {
		t.Cleanup(svc.asyncManager.Close)
	}

	req := httptest.NewRequest(http.MethodGet, "/AsyncTestEntities", nil)
	req.Header.Set("Prefer", "return=minimal, respond-async")

	rec := httptest.NewRecorder()
	svc.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", rec.Code)
	}

	if applied := rec.Header().Get("Preference-Applied"); applied != "respond-async" {
		t.Fatalf("expected Preference-Applied to be respond-async, got %q", applied)
	}

	if retry := rec.Header().Get("Retry-After"); retry != "3" {
		t.Fatalf("expected Retry-After header of 3, got %q", retry)
	}

	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header with monitor URL")
	}

	if !strings.HasPrefix(location, svc.asyncMonitorPrefix) {
		t.Fatalf("monitor URL %q does not start with prefix %q", location, svc.asyncMonitorPrefix)
	}

	jobID := strings.TrimPrefix(location, svc.asyncMonitorPrefix)
	job, ok := svc.asyncManager.GetJob(jobID)
	if !ok {
		t.Fatalf("expected job %q to be registered", jobID)
	}

	job.Wait()

	monitorReq := httptest.NewRequest(http.MethodGet, location, nil)
	monitorRec := httptest.NewRecorder()
	svc.ServeHTTP(monitorRec, monitorReq)

	if monitorRec.Code == http.StatusAccepted {
		t.Fatalf("expected terminal monitor response, got status %d", monitorRec.Code)
	}
}
