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

func TestManagerDefaultRetentionCleanup(t *testing.T) {
	originalDefault := defaultJobRetention
	defaultJobRetention = 50 * time.Millisecond
	t.Cleanup(func() {
		defaultJobRetention = originalDefault
	})

	mgr, db := newTestManager(t, 0)

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

	var record JobRecord
	if err := db.First(&record, "id = ?", job.ID).Error; err != nil {
		t.Fatalf("expected job record to exist before cleanup, got %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var lookupErr error
	for time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		lookupErr = db.First(&record, "id = ?", job.ID).Error
		if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			break
		}
	}

	if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		t.Fatalf("expected job row to be deleted by background cleanup, got %v", lookupErr)
	}

	mgr.mu.Lock()
	_, exists := mgr.jobs[job.ID]
	mgr.mu.Unlock()
	if exists {
		t.Fatalf("expected job to be removed from memory after cleanup")
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

func TestWithMonitorURL(t *testing.T) {
	mgr, _ := newTestManager(t, 0)

	monitorURL := "/async/jobs/test-123"
	job, err := mgr.StartJob(context.Background(), func(ctx context.Context) (*StoredResponse, error) {
		return &StoredResponse{StatusCode: http.StatusOK}, nil
	}, WithMonitorURL(monitorURL))

	if err != nil {
		t.Fatalf("StartJob error: %v", err)
	}

	if job.MonitorURL() != monitorURL {
		t.Errorf("expected monitor URL %q, got %q", monitorURL, job.MonitorURL())
	}
}

func TestWithRetentionDisabled(t *testing.T) {
	db := newTestDB(t)
	mgr, err := NewManager(db, 0, WithRetentionDisabled())
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}
	t.Cleanup(mgr.Close)

	if mgr.ttl != 0 {
		t.Errorf("expected TTL to be 0 when retention is disabled, got %v", mgr.ttl)
	}

	if mgr.cleanupTicker != nil {
		t.Error("expected cleanup ticker to be nil when retention is disabled")
	}
}

func TestJobRetryAfter(t *testing.T) {
	mgr, _ := newTestManager(t, 0)

	t.Run("with retry-after set", func(t *testing.T) {
		retryDuration := 5 * time.Second
		job, err := mgr.StartJob(context.Background(), func(ctx context.Context) (*StoredResponse, error) {
			return &StoredResponse{StatusCode: http.StatusOK}, nil
		}, WithRetryAfter(retryDuration))

		if err != nil {
			t.Fatalf("StartJob error: %v", err)
		}

		duration, ok := job.RetryAfter()
		if !ok {
			t.Error("expected retry-after to be set")
		}
		if duration != retryDuration {
			t.Errorf("expected retry duration %v, got %v", retryDuration, duration)
		}
	})

	t.Run("without retry-after", func(t *testing.T) {
		job, err := mgr.StartJob(context.Background(), func(ctx context.Context) (*StoredResponse, error) {
			return &StoredResponse{StatusCode: http.StatusOK}, nil
		})

		if err != nil {
			t.Fatalf("StartJob error: %v", err)
		}

		_, ok := job.RetryAfter()
		if ok {
			t.Error("expected retry-after to not be set")
		}
	})
}

func TestJobSetRetryAfter(t *testing.T) {
	mgr, _ := newTestManager(t, 0)

	job, err := mgr.StartJob(context.Background(), func(ctx context.Context) (*StoredResponse, error) {
		return &StoredResponse{StatusCode: http.StatusOK}, nil
	})
	if err != nil {
		t.Fatalf("StartJob error: %v", err)
	}
	if err := job.SetMonitorURL("/async/jobs/" + job.ID); err != nil {
		t.Fatalf("SetMonitorURL error: %v", err)
	}

	t.Run("set retry-after", func(t *testing.T) {
		retryDuration := 10 * time.Second
		err := job.SetRetryAfter(retryDuration)
		if err != nil {
			t.Errorf("SetRetryAfter error: %v", err)
		}

		duration, ok := job.RetryAfter()
		if !ok {
			t.Error("expected retry-after to be set")
		}
		if duration != retryDuration {
			t.Errorf("expected retry duration %v, got %v", retryDuration, duration)
		}
	})

	t.Run("clear retry-after with zero duration", func(t *testing.T) {
		err := job.SetRetryAfter(0)
		if err != nil {
			t.Errorf("SetRetryAfter error: %v", err)
		}

		_, ok := job.RetryAfter()
		if ok {
			t.Error("expected retry-after to be cleared")
		}
	})

	t.Run("clear retry-after with negative duration", func(t *testing.T) {
		err := job.SetRetryAfter(-1 * time.Second)
		if err != nil {
			t.Errorf("SetRetryAfter error: %v", err)
		}

		_, ok := job.RetryAfter()
		if ok {
			t.Error("expected retry-after to be cleared")
		}
	})
}

func TestJobErrorMessage(t *testing.T) {
	mgr, _ := newTestManager(t, 0)

	expectedError := "test error message"
	job, err := mgr.StartJob(context.Background(), func(ctx context.Context) (*StoredResponse, error) {
		return nil, errors.New(expectedError)
	})

	if err != nil {
		t.Fatalf("StartJob error: %v", err)
	}
	if err := job.SetMonitorURL("/async/jobs/" + job.ID); err != nil {
		t.Fatalf("SetMonitorURL error: %v", err)
	}

	job.Wait()

	if job.ErrorMessage() != expectedError {
		t.Errorf("expected error message %q, got %q", expectedError, job.ErrorMessage())
	}
}

func TestJobIsTerminal(t *testing.T) {
	mgr, _ := newTestManager(t, 0)

	tests := []struct {
		name         string
		handler      Handler
		shouldCancel bool
		wantTerminal bool
	}{
		{
			name: "completed job is terminal",
			handler: func(ctx context.Context) (*StoredResponse, error) {
				return &StoredResponse{StatusCode: http.StatusOK}, nil
			},
			shouldCancel: false,
			wantTerminal: true,
		},
		{
			name: "failed job is terminal",
			handler: func(ctx context.Context) (*StoredResponse, error) {
				return nil, errors.New("job failed")
			},
			shouldCancel: false,
			wantTerminal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, err := mgr.StartJob(context.Background(), tt.handler)
			if err != nil {
				t.Fatalf("StartJob error: %v", err)
			}
			if err := job.SetMonitorURL("/async/jobs/" + job.ID); err != nil {
				t.Fatalf("SetMonitorURL error: %v", err)
			}

			if tt.shouldCancel {
				job.cancel()
			}

			job.Wait()

			if got := job.IsTerminal(); got != tt.wantTerminal {
				t.Errorf("IsTerminal() = %v, want %v (status: %v)", got, tt.wantTerminal, job.Status)
			}
		})
	}

	t.Run("canceled job is terminal", func(t *testing.T) {
		job, err := mgr.StartJob(context.Background(), func(ctx context.Context) (*StoredResponse, error) {
			// Wait for cancellation or completion
			<-ctx.Done()
			return nil, ctx.Err()
		})
		if err != nil {
			t.Fatalf("StartJob error: %v", err)
		}
		if err := job.SetMonitorURL("/async/jobs/" + job.ID); err != nil {
			t.Fatalf("SetMonitorURL error: %v", err)
		}

		// Give the job a moment to start
		time.Sleep(10 * time.Millisecond)

		// Cancel the job
		req := httptest.NewRequest(http.MethodDelete, job.MonitorURL(), nil)
		rec := httptest.NewRecorder()
		mgr.ServeMonitor(rec, req)

		job.Wait()

		if !job.IsTerminal() {
			t.Errorf("canceled job should be terminal (status: %v)", job.Status)
		}
	})
}
