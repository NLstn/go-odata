package runtime

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/nlstn/go-odata/internal/async"
	"github.com/nlstn/go-odata/internal/preference"
	servrouter "github.com/nlstn/go-odata/internal/service/router"
)

// Logger captures the subset of slog.Logger functionality required by the runtime.
type Logger interface {
	Error(msg string, args ...any)
}

// Runtime coordinates HTTP request handling for a service instance.
type Runtime struct {
	router *servrouter.Router
	logger Logger

	asyncEnabled         bool
	asyncManager         *async.Manager
	asyncQueue           chan struct{}
	asyncMonitorPrefix   string
	defaultRetryInterval time.Duration
}

// New creates a new Runtime.
func New(router *servrouter.Router, logger Logger) *Runtime {
	return &Runtime{
		router: router,
		logger: logger,
	}
}

// SetRouter updates the router used to dispatch requests.
func (rt *Runtime) SetRouter(router *servrouter.Router) {
	rt.router = router
}

// SetLogger updates the logger used for error reporting.
func (rt *Runtime) SetLogger(logger Logger) {
	rt.logger = logger
}

// ConfigureAsync configures asynchronous request processing dependencies.
func (rt *Runtime) ConfigureAsync(manager *async.Manager, queue chan struct{}, monitorPrefix string, defaultRetryInterval time.Duration) {
	if manager == nil {
		rt.asyncEnabled = false
		rt.asyncManager = nil
		rt.asyncQueue = nil
		rt.asyncMonitorPrefix = ""
		rt.defaultRetryInterval = 0
		return
	}

	rt.asyncEnabled = true
	rt.asyncManager = manager
	rt.asyncQueue = queue
	rt.asyncMonitorPrefix = monitorPrefix
	rt.defaultRetryInterval = defaultRetryInterval
}

// ServeHTTP dispatches the request via the configured router.
func (rt *Runtime) ServeHTTP(w http.ResponseWriter, r *http.Request, allowAsync bool) {
	if rt.router == nil {
		http.Error(w, "service router not initialized", http.StatusInternalServerError)
		return
	}

	if allowAsync && rt.tryHandleAsync(w, r) {
		return
	}

	rt.router.ServeHTTP(w, r)
}

func (rt *Runtime) tryHandleAsync(w http.ResponseWriter, r *http.Request) bool {
	if !rt.asyncEnabled || rt.asyncManager == nil {
		return false
	}

	if rt.asyncMonitorPrefix != "" && strings.HasPrefix(r.URL.Path, rt.asyncMonitorPrefix) {
		return false
	}

	pref := preference.ParsePrefer(r)
	if pref == nil || !pref.RespondAsyncRequested {
		return false
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/")
	if !isAsyncEligiblePath(trimmed) {
		return false
	}

	body, err := bufferRequestBody(r)
	if err != nil {
		http.Error(w, "failed to buffer request body", http.StatusInternalServerError)
		return true
	}

	sanitizedPrefer := preference.SanitizeForAsyncDispatch(r.Header.Get("Prefer"))

	queueToken := rt.acquireAsyncSlot()
	if queueToken == nil {
		// Fallback to synchronous execution
		restoreRequestBody(r, body)
		return false
	}

	handler := func(ctx context.Context) (*async.StoredResponse, error) {
		defer queueToken.release()

		reqCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		cloned := r.Clone(reqCtx)
		restoreRequestBody(cloned, body)
		if sanitizedPrefer == "" {
			cloned.Header.Del("Prefer")
		} else {
			cloned.Header.Set("Prefer", sanitizedPrefer)
		}

		recorder := httptest.NewRecorder()
		rt.router.ServeHTTP(recorder, cloned)

		return &async.StoredResponse{
			StatusCode: recorder.Code,
			Header:     cloneHeader(recorder.Header()),
			Body:       append([]byte(nil), recorder.Body.Bytes()...),
		}, nil
	}

	jobOpts := []async.JobOption{}
	if rt.defaultRetryInterval > 0 {
		jobOpts = append(jobOpts, async.WithRetryAfter(rt.defaultRetryInterval))
	}

	job, err := rt.asyncManager.StartJob(r.Context(), handler, jobOpts...)
	if err != nil {
		restoreRequestBody(r, body)
		queueToken.release()
		http.Error(w, "failed to start async job", http.StatusInternalServerError)
		return true
	}

	// Don't restore body on original request - the async worker clones the request
	// and restores the body on the clone. Restoring here would race with r.Clone()
	// in the worker goroutine.

	if rt.asyncMonitorPrefix != "" {
		if err := job.SetMonitorURL(rt.asyncMonitorPrefix + job.ID); err != nil {
			log.Printf("odata: failed to persist async monitor URL for job %s: %v", job.ID, err)
		}
	}

	async.WriteInitialResponse(w, job)
	return true
}

type queueToken struct {
	ch chan struct{}
}

func (t *queueToken) release() {
	if t == nil || t.ch == nil {
		return
	}
	<-t.ch
}

func (rt *Runtime) acquireAsyncSlot() *queueToken {
	if rt.asyncQueue == nil {
		return &queueToken{}
	}

	select {
	case rt.asyncQueue <- struct{}{}:
		return &queueToken{ch: rt.asyncQueue}
	default:
		return nil
	}
}

func bufferRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	bodyReader := r.Body
	defer func(body io.ReadCloser) {
		_ = body.Close() //nolint:errcheck // best effort close
	}(bodyReader)

	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func restoreRequestBody(r *http.Request, body []byte) {
	if r.Body != nil {
		_ = r.Body.Close() //nolint:errcheck // replace body regardless of close error
	}
	reader := bytes.NewReader(body)
	r.Body = io.NopCloser(reader)
	r.ContentLength = int64(len(body))
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
}

func cloneHeader(h http.Header) http.Header {
	cloned := make(http.Header, len(h))
	for k, vals := range h {
		cloned[k] = append([]string(nil), vals...)
	}
	return cloned
}

func isAsyncEligiblePath(path string) bool {
	switch path {
	case "", "$metadata", "$batch":
		return false
	}
	return true
}
