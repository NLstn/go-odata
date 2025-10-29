package async

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// JobStatus represents the lifecycle state of an asynchronous job.
type JobStatus string

const (
	// JobPending indicates the job has been created but not yet completed.
	JobPending JobStatus = "pending"
	// JobRunning indicates the job handler is executing.
	JobRunning JobStatus = "running"
	// JobCompleted indicates the job handler succeeded and produced a response.
	JobCompleted JobStatus = "completed"
	// JobFailed indicates the job handler finished with an error.
	JobFailed JobStatus = "failed"
	// JobCanceled indicates the job was canceled before completion.
	JobCanceled JobStatus = "canceled"
)

// StoredResponse captures the output of an asynchronous job for later replay.
type StoredResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// Job represents the execution of an asynchronous handler.
type Job struct {
	ID          string
	Status      JobStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
	Response    *StoredResponse
	Error       string

	monitorURL string
	retryAfter *time.Duration

	cancel context.CancelFunc
	done   chan struct{}
}

// WithMonitorURL sets the URL clients should poll for job status.
func WithMonitorURL(url string) JobOption {
	return func(j *Job) {
		j.monitorURL = url
	}
}

// WithRetryAfter configures an optional Retry-After header for the job.
func WithRetryAfter(d time.Duration) JobOption {
	return func(j *Job) {
		j.retryAfter = &d
	}
}

// JobOption mutates a job at creation time.
type JobOption func(*Job)

// Handler is the unit of work executed for an asynchronous job.
type Handler func(ctx context.Context) (*StoredResponse, error)

// Manager supervises asynchronous jobs, tracking their lifecycle and cleanup.
type Manager struct {
	mu            sync.Mutex
	jobs          map[string]*Job
	ttl           time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewManager constructs a Manager with the supplied TTL for completed jobs.
func NewManager(ttl time.Duration) *Manager {
	m := &Manager{
		jobs:        make(map[string]*Job),
		ttl:         ttl,
		stopCleanup: make(chan struct{}),
	}

	if ttl > 0 {
		interval := ttl / 2
		if interval <= 0 {
			interval = ttl
		}
		m.cleanupTicker = time.NewTicker(interval)
		go func() {
			for {
				select {
				case <-m.cleanupTicker.C:
					m.cleanupExpired()
				case <-m.stopCleanup:
					m.cleanupTicker.Stop()
					return
				}
			}
		}()
	}

	return m
}

// Close stops the manager's background cleanup.
func (m *Manager) Close() {
	if m.cleanupTicker == nil {
		return
	}
	select {
	case <-m.stopCleanup:
		// already closed
	default:
		close(m.stopCleanup)
	}
}

// StartJob registers a new asynchronous job and launches the handler in a goroutine.
func (m *Manager) StartJob(ctx context.Context, handler Handler, opts ...JobOption) (*Job, error) {
	if handler == nil {
		return nil, errors.New("async: handler is required")
	}

	id, err := generateID()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	job := &Job{
		ID:        id,
		Status:    JobPending,
		CreatedAt: now,
		UpdatedAt: now,
		done:      make(chan struct{}),
	}

	for _, opt := range opts {
		opt(job)
	}

	jobCtx, cancel := context.WithCancel(ctx)
	job.cancel = cancel

	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	go m.run(job, jobCtx, handler)

	return job, nil
}

// GetJob retrieves a job by ID.
func (m *Manager) GetJob(id string) (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	return job, ok
}

// CancelJob requests cancellation of the specified job.
func (m *Manager) CancelJob(id string) bool {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return false
	}

	if job.Status == JobCompleted || job.Status == JobFailed || job.Status == JobCanceled {
		m.mu.Unlock()
		return false
	}

	cancel := job.cancel
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	return true
}

// Wait blocks until the job reaches a terminal state.
func (j *Job) Wait() {
	<-j.done
}

func (m *Manager) run(job *Job, ctx context.Context, handler Handler) {
	defer close(job.done)

	m.updateStatus(job, JobRunning)

	resp, err := handler(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
			m.finish(job, JobCanceled, resp, nil)
			return
		}
		m.finish(job, JobFailed, resp, err)
		return
	}

	m.finish(job, JobCompleted, resp, nil)
}

func (m *Manager) updateStatus(job *Job, status JobStatus) {
	m.mu.Lock()
	job.Status = status
	now := time.Now()
	job.UpdatedAt = now
	m.mu.Unlock()
}

func (m *Manager) finish(job *Job, status JobStatus, resp *StoredResponse, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job.Status = status
	now := time.Now()
	job.UpdatedAt = now
	job.CompletedAt = &now

	if resp != nil {
		job.Response = cloneStoredResponse(resp)
	}

	if err != nil {
		job.Error = err.Error()
	}
}

func cloneStoredResponse(resp *StoredResponse) *StoredResponse {
	if resp == nil {
		return nil
	}
	cloned := &StoredResponse{
		StatusCode: resp.StatusCode,
		Body:       append([]byte(nil), resp.Body...),
	}
	if resp.Header != nil {
		cloned.Header = make(http.Header, len(resp.Header))
		for k, vv := range resp.Header {
			cloned.Header[k] = append([]string(nil), vv...)
		}
	} else {
		cloned.Header = make(http.Header)
	}
	return cloned
}

func generateID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (m *Manager) cleanupExpired() {
	if m.ttl <= 0 {
		return
	}

	cutoff := time.Now().Add(-m.ttl)

	m.mu.Lock()
	defer m.mu.Unlock()

	for id, job := range m.jobs {
		if job.CompletedAt == nil {
			continue
		}
		if job.CompletedAt.Before(cutoff) {
			delete(m.jobs, id)
		}
	}
}

// WriteInitialResponse writes the HTTP 202 response acknowledging an async job.
func WriteInitialResponse(w http.ResponseWriter, job *Job) {
	if job == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	location := job.monitorURL
	if location == "" {
		location = job.ID
	}
	w.Header().Set("Location", location)
	w.Header().Set("Preference-Applied", "respond-async")
	if job.retryAfter != nil {
		w.Header().Set("Retry-After", formatRetryAfter(*job.retryAfter))
	}
	w.WriteHeader(http.StatusAccepted)
}

func formatRetryAfter(d time.Duration) string {
	if d <= 0 {
		return "0"
	}
	seconds := int64(math.Ceil(d.Seconds()))
	return strconv.FormatInt(seconds, 10)
}

// jobSnapshot holds a read-only view of a job's state.
type jobSnapshot struct {
	status     JobStatus
	retryAfter *time.Duration
	response   *StoredResponse
	err        string
}

// ServeMonitor handles HTTP requests for job status monitoring.
func (m *Manager) ServeMonitor(w http.ResponseWriter, r *http.Request) {
	id := extractJobID(r)
	if id == "" {
		http.NotFound(w, r)
		return
	}

	m.mu.Lock()
	job, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	// Create snapshot while holding lock to avoid data races
	snapshot := jobSnapshot{
		status:     job.Status,
		retryAfter: job.retryAfter,
		response:   job.Response,
		err:        job.Error,
	}
	m.mu.Unlock()

	switch r.Method {
	case http.MethodDelete:
		if m.CancelJob(id) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if snapshot.status == JobCanceled {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if snapshot.status == JobCompleted || snapshot.status == JobFailed {
			m.mu.Lock()
			delete(m.jobs, id)
			m.mu.Unlock()
		}
		w.WriteHeader(http.StatusNoContent)
		return
	case http.MethodGet, http.MethodHead, "":
		// continue below
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	switch snapshot.status {
	case JobPending, JobRunning:
		w.Header().Set("Preference-Applied", "respond-async")
		if snapshot.retryAfter != nil {
			w.Header().Set("Retry-After", formatRetryAfter(*snapshot.retryAfter))
		}
		w.WriteHeader(http.StatusAccepted)
	case JobCompleted:
		writeStoredResponse(w, snapshot.response, r.Method != http.MethodHead)
	case JobFailed:
		if snapshot.response != nil {
			writeStoredResponse(w, snapshot.response, r.Method != http.MethodHead)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		if snapshot.err != "" && r.Method != http.MethodHead {
			writeBytes(w, []byte(snapshot.err))
		}
	case JobCanceled:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusAccepted)
	}
}

func writeStoredResponse(w http.ResponseWriter, resp *StoredResponse, includeBody bool) {
	if resp == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	copyHeader(w.Header(), resp.Header)
	status := resp.StatusCode
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if includeBody && len(resp.Body) > 0 {
		writeBytes(w, resp.Body)
	}
}

func copyHeader(dst, src http.Header) {
	for k := range dst {
		dst.Del(k)
	}
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func writeBytes(w http.ResponseWriter, body []byte) {
	if len(body) == 0 {
		return
	}
	if _, err := w.Write(body); err != nil {
		return
	}
}

func extractJobID(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	p := r.URL.Path
	if p == "" || p == "/" {
		return ""
	}
	p = strings.TrimSuffix(p, "/")
	return path.Base(p)
}

// MonitorURL returns the URL clients should poll for this job.
func (j *Job) MonitorURL() string {
	return j.monitorURL
}

// RetryAfter returns the configured retry duration if present.
func (j *Job) RetryAfter() (time.Duration, bool) {
	if j.retryAfter == nil {
		return 0, false
	}
	return *j.retryAfter, true
}

// SetRetryAfter updates the retry-after duration for the job. A non-positive
// duration clears the setting.
func (j *Job) SetRetryAfter(d time.Duration) {
	if d <= 0 {
		j.retryAfter = nil
		return
	}
	j.retryAfter = &d
}

// SetMonitorURL updates the monitoring URL for the job.
func (j *Job) SetMonitorURL(url string) {
	j.monitorURL = url
}

// ErrorMessage returns the stored error for the job.
func (j *Job) ErrorMessage() string {
	return j.Error
}

// IsTerminal reports whether the job has completed execution.
func (j *Job) IsTerminal() bool {
	switch j.Status {
	case JobCompleted, JobFailed, JobCanceled:
		return true
	default:
		return false
	}
}
