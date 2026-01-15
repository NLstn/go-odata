package runtime_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type asyncTestEntity struct {
	ID   uint `gorm:"primaryKey" odata:"key"`
	Name string
}

func TestServiceRespondAsyncFlow(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&asyncTestEntity{}); err != nil {
		t.Fatalf("failed to migrate test entity: %v", err)
	}

	svc := odata.NewService(db)
	if err := svc.RegisterEntity(&asyncTestEntity{}); err != nil {
		t.Fatalf("failed to register entity: %v", err)
	}

	if err := svc.EnableAsyncProcessing(odata.AsyncConfig{
		MonitorPathPrefix:    "/$async/jobs",
		DefaultRetryInterval: 3 * time.Second,
		JobRetention:         time.Minute,
	}); err != nil {
		t.Fatalf("failed to enable async processing: %v", err)
	}

	// Wait for async tables to be created (AutoMigrate is asynchronous in some cases)
	// Verify table exists before proceeding with async requests
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		var count int64
		if err := db.Raw("SELECT COUNT(*) FROM _odata_async_jobs").Scan(&count).Error; err == nil {
			break // Table exists
		}
		if i == maxAttempts-1 {
			t.Fatal("async jobs table not created after EnableAsyncProcessing")
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Fatalf("failed to close service: %v", err)
		}
	})

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

	if !strings.HasPrefix(location, svc.AsyncMonitorPrefix()) {
		t.Fatalf("monitor URL %q does not start with prefix %q", location, svc.AsyncMonitorPrefix())
	}

	deadline := time.Now().Add(2 * time.Second)
	var (
		monitorRec *httptest.ResponseRecorder
		status     = http.StatusAccepted
	)
	for time.Now().Before(deadline) {
		monitorReq := httptest.NewRequest(http.MethodGet, location, nil)
		monitorRec = httptest.NewRecorder()
		svc.ServeHTTP(monitorRec, monitorReq)
		status = monitorRec.Code
		if status != http.StatusAccepted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if status == http.StatusAccepted {
		t.Fatalf("expected terminal monitor response before deadline, last status %d", status)
	}
}
