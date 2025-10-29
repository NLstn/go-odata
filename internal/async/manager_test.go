package async

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestManagerLifecycle(t *testing.T) {
	mgr := NewManager(0)
	t.Cleanup(mgr.Close)

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
	job.SetMonitorURL("/async/jobs/" + job.ID)

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
	mgr := NewManager(0)
	t.Cleanup(mgr.Close)

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
	job.SetMonitorURL("/async/jobs/" + job.ID)

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
	mgr := NewManager(ttl)
	t.Cleanup(mgr.Close)

	job, err := mgr.StartJob(context.Background(), func(ctx context.Context) (*StoredResponse, error) {
		return &StoredResponse{StatusCode: http.StatusOK}, nil
	})
	if err != nil {
		t.Fatalf("StartJob error: %v", err)
	}
	job.SetMonitorURL("/async/jobs/" + job.ID)

	job.Wait()
	if _, ok := mgr.GetJob(job.ID); !ok {
		t.Fatalf("expected job to be retained immediately after completion")
	}

	time.Sleep(3 * ttl)
	mgr.cleanupExpired()

	if _, ok := mgr.GetJob(job.ID); ok {
		t.Fatalf("expected job to be cleaned up after TTL")
	}
}
