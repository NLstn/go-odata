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
	"github.com/nlstn/go-odata/internal/observability"
	"github.com/nlstn/go-odata/internal/preference"
	servrouter "github.com/nlstn/go-odata/internal/service/router"
	"go.opentelemetry.io/otel/trace"
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

	// observability holds the OpenTelemetry configuration
	observability *observability.Config
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

// SetObservability configures observability for the runtime.
func (rt *Runtime) SetObservability(cfg *observability.Config) {
	rt.observability = cfg
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

	ctx := r.Context()
	start := time.Now()

	// Start tracing span if observability is configured
	var tracer *observability.Tracer
	var metrics *observability.Metrics
	var span trace.Span
	if rt.observability != nil {
		tracer = rt.observability.Tracer()
		metrics = rt.observability.Metrics()

		ctx, span = tracer.StartRequest(ctx, r)
		defer span.End()

		// Record request start in metrics
		metrics.RecordRequestStart(ctx)
		r = r.WithContext(ctx)
	}

	if allowAsync && rt.tryHandleAsync(w, r) {
		return
	}

	// Wrap response writer to capture status code for metrics
	wrapped := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
	rt.router.ServeHTTP(wrapped, r)

	// Record metrics if observability is configured
	if rt.observability != nil {
		duration := time.Since(start)
		entitySet := extractEntitySetFromPath(r.URL.Path)
		operation := extractOperationType(r)
		metrics.RecordRequestEnd(ctx, duration, wrapped.statusCode)
		if wrapped.statusCode >= 400 {
			metrics.RecordError(ctx, entitySet, operation, http.StatusText(wrapped.statusCode))
		}
		tracer.SetHTTPStatus(ctx, wrapped.statusCode)
		if entitySet != "" && operation != "" {
			metrics.RecordRequestDuration(ctx, duration, entitySet, operation, wrapped.statusCode)
		}
	}
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

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	if !r.written {
		r.statusCode = statusCode
		r.written = true
	}
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.written {
		r.written = true
	}
	return r.ResponseWriter.Write(b)
}

// extractEntitySetFromPath extracts the entity set name from a URL path.
func extractEntitySetFromPath(path string) string {
	path = strings.TrimPrefix(path, "/")

	if path == "" || path == "$metadata" || path == "$batch" {
		return ""
	}

	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 {
		return ""
	}

	entitySet := parts[0]
	if idx := strings.Index(entitySet, "("); idx > 0 {
		entitySet = entitySet[:idx]
	}

	return entitySet
}

// extractOperationType determines the OData operation type from the request.
func extractOperationType(r *http.Request) string {
	path := strings.TrimPrefix(r.URL.Path, "/")

	if path == "$metadata" {
		return observability.OpMetadata
	}
	if path == "" {
		return observability.OpServiceDoc
	}
	if path == "$batch" || strings.HasSuffix(path, "/$batch") {
		return observability.OpBatch
	}
	if strings.HasSuffix(path, "/$count") {
		return observability.OpCount
	}
	if strings.Contains(path, "/$ref") {
		return observability.OpRef
	}

	hasKey := strings.Contains(path, "(") && strings.Contains(path, ")")

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if hasKey {
			return observability.OpReadEntity
		}
		return observability.OpReadCollection
	case http.MethodPost:
		return observability.OpCreate
	case http.MethodPatch:
		return observability.OpPatch
	case http.MethodPut:
		return observability.OpUpdate
	case http.MethodDelete:
		return observability.OpDelete
	}

	return ""
}
