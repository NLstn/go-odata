package async

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use file-based database for better concurrency support
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	return db
}

func newTestManager(t *testing.T, ttl time.Duration) (*Manager, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	mgr, err := NewManager(db, ttl)
	if err != nil {
		t.Fatalf("failed to create async manager: %v", err)
	}
	t.Cleanup(mgr.Close)
	return mgr, db
}

func TestManagerLifecycle(t *testing.T) {
	mgr, _ := newTestManager(t, 0)

	started := make(chan struct{})
	finish := make(chan struct{})

	handler := func(ctx context.Context) (*StoredResponse, error) {
		close(started)
		select {
		case <-finish:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		hdr := http.Header{}
		hdr.Set("Content-Type", "application/json")
		return &StoredResponse{
			StatusCode: http.StatusCreated,
			Header:     hdr,
			Body:       []byte(`{"status":"ok"}`),
		}, nil
	}

	job, err := mgr.StartJob(context.Background(), handler, WithRetryAfter(2*time.Second))
	if err != nil {
		t.Fatalf("StartJob error: %v", err)
	}
	if err := job.SetMonitorURL("/async/jobs/" + job.ID); err != nil {
		t.Fatalf("SetMonitorURL error: %v", err)
	}

	initial := httptest.NewRecorder()
	WriteInitialResponse(initial, job)
	if initial.Code != http.StatusAccepted {
		t.Fatalf("expected initial status 202, got %d", initial.Code)
	}
	if got := initial.Header().Get("Location"); got != job.MonitorURL() {
		t.Fatalf("expected Location %q, got %q", job.MonitorURL(), got)
	}
	if got := initial.Header().Get("Preference-Applied"); got != "respond-async" {
		t.Fatalf("expected Preference-Applied header, got %q", got)
	}
	if got := initial.Header().Get("Retry-After"); got != "2" {
		t.Fatalf("expected Retry-After of 2, got %q", got)
	}

	<-started

	pendingReq := httptest.NewRequest(http.MethodGet, job.MonitorURL(), nil)
	pendingRec := httptest.NewRecorder()
	mgr.ServeMonitor(pendingRec, pendingReq)
	if pendingRec.Code != http.StatusAccepted {
		t.Fatalf("expected pending status 202, got %d", pendingRec.Code)
	}
	if got := pendingRec.Header().Get("Preference-Applied"); got != "respond-async" {
		t.Fatalf("expected pending Preference-Applied header, got %q", got)
	}

	close(finish)
	job.Wait()

	completeReq := httptest.NewRequest(http.MethodGet, job.MonitorURL(), nil)
	completeRec := httptest.NewRecorder()
	mgr.ServeMonitor(completeRec, completeReq)

	if completeRec.Code != http.StatusCreated {
		t.Fatalf("expected completed status 201, got %d", completeRec.Code)
	}
	if got := completeRec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type to be propagated, got %q", got)
	}
	if body := completeRec.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("unexpected body %q", body)
	}

	headReq := httptest.NewRequest(http.MethodHead, job.MonitorURL(), nil)
	headRec := httptest.NewRecorder()
	mgr.ServeMonitor(headRec, headReq)
	if headRec.Code != http.StatusCreated {
		t.Fatalf("expected HEAD status to match final response, got %d", headRec.Code)
	}
	if headRec.Body.Len() != 0 {
		t.Fatalf("expected no body for HEAD request, got %q", headRec.Body.String())
	}
}

func TestManagerCancellation(t *testing.T) {
	mgr, _ := newTestManager(t, 0)

	release := make(chan struct{})
	handler := func(ctx context.Context) (*StoredResponse, error) {
		select {
		case <-release:
			return &StoredResponse{StatusCode: http.StatusOK}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	job, err := mgr.StartJob(context.Background(), handler)
	if err != nil {
		t.Fatalf("StartJob error: %v", err)
	}
	if err := job.SetMonitorURL("/async/jobs/" + job.ID); err != nil {
		t.Fatalf("SetMonitorURL error: %v", err)
	}

	cancelReq := httptest.NewRequest(http.MethodDelete, job.MonitorURL(), nil)
	cancelRec := httptest.NewRecorder()
	mgr.ServeMonitor(cancelRec, cancelReq)

	if cancelRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on cancel, got %d", cancelRec.Code)
	}

	job.Wait()
	if job.Status != JobCanceled {
		t.Fatalf("expected job status canceled, got %s", job.Status)
	}

	afterReq := httptest.NewRequest(http.MethodGet, job.MonitorURL(), nil)
	afterRec := httptest.NewRecorder()
	mgr.ServeMonitor(afterRec, afterReq)
	if afterRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 after cancellation, got %d", afterRec.Code)
	}
}

func TestManagerCleanupRemovesOldJobs(t *testing.T) {
	ttl := 10 * time.Millisecond
	mgr, db := newTestManager(t, ttl)

	job, err := mgr.StartJob(context.Background(), func(ctx context.Context) (*StoredResponse, error) {
		return &StoredResponse{StatusCode: http.StatusOK}, nil
	})
	if err != nil {
		t.Fatalf("StartJob error: %v", err)
	}
	if err := job.SetMonitorURL("/async/jobs/" + job.ID); err != nil {
		t.Fatalf("SetMonitorURL error: %v", err)
	}

	job.Wait()
	if _, ok := mgr.GetJob(job.ID); ok {
		t.Fatalf("expected job to be removed from active set after completion")
	}

	time.Sleep(3 * ttl)
	mgr.cleanupExpired()

	if _, ok := mgr.GetJob(job.ID); ok {
		t.Fatalf("expected job to remain absent from active set after cleanup")
	}

	var record JobRecord
	err = db.First(&record, "id = ?", job.ID).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected job row to be deleted, got error %v", err)
	}
}

func TestManagerPersistenceAcrossInstances(t *testing.T) {
	mgr1, db := newTestManager(t, 0)

	release := make(chan struct{})
	handler := func(ctx context.Context) (*StoredResponse, error) {
		<-release
		hdr := http.Header{}
		hdr.Set("X-Async-Result", "done")
		return &StoredResponse{
			StatusCode: http.StatusCreated,
			Header:     hdr,
			Body:       []byte(`{"status":"ok"}`),
		}, nil
	}

	job, err := mgr1.StartJob(context.Background(), handler)
	if err != nil {
		t.Fatalf("StartJob error: %v", err)
	}
	if err := job.SetMonitorURL("/async/jobs/" + job.ID); err != nil {
		t.Fatalf("SetMonitorURL error: %v", err)
	}

	close(release)
	job.Wait()

	mgr2, err := NewManager(db, 0)
	if err != nil {
		t.Fatalf("failed to create second manager: %v", err)
	}
	t.Cleanup(mgr2.Close)

	req := httptest.NewRequest(http.MethodGet, "/async/jobs/"+job.ID, nil)
	rec := httptest.NewRecorder()
	mgr2.ServeMonitor(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected persisted monitor response 201, got %d", rec.Code)
	}
	if got := rec.Header().Get("X-Async-Result"); got != "done" {
		t.Fatalf("expected persisted header, got %q", got)
	}
	if body := rec.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("expected persisted body, got %q", body)
	}
}
