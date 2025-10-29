package odata

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/nlstn/go-odata/internal/async"
	"github.com/nlstn/go-odata/internal/preference"
)

// ServeHTTP implements http.Handler by delegating to the internal router.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.serveHTTP(w, r, true)
}

func (s *Service) serveHTTP(w http.ResponseWriter, r *http.Request, allowAsync bool) {
	if s.router == nil {
		http.Error(w, "service router not initialized", http.StatusInternalServerError)
		return
	}

	if allowAsync && s.tryHandleAsync(w, r) {
		return
	}

	s.router.ServeHTTP(w, r)
}

func (s *Service) tryHandleAsync(w http.ResponseWriter, r *http.Request) bool {
	if s.asyncManager == nil || s.asyncConfig == nil {
		return false
	}

	if s.asyncMonitorPrefix != "" && strings.HasPrefix(r.URL.Path, s.asyncMonitorPrefix) {
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

	queueToken := s.acquireAsyncSlot()
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
		s.router.ServeHTTP(recorder, cloned)

		return &async.StoredResponse{
			StatusCode: recorder.Code,
			Header:     cloneHeader(recorder.Header()),
			Body:       append([]byte(nil), recorder.Body.Bytes()...),
		}, nil
	}

	jobOpts := []async.JobOption{}
	if s.asyncConfig.DefaultRetryInterval > 0 {
		jobOpts = append(jobOpts, async.WithRetryAfter(s.asyncConfig.DefaultRetryInterval))
	}

	job, err := s.asyncManager.StartJob(r.Context(), handler, jobOpts...)
	if err != nil {
		restoreRequestBody(r, body)
		queueToken.release()
		http.Error(w, "failed to start async job", http.StatusInternalServerError)
		return true
	}

	restoreRequestBody(r, body)

	if s.asyncMonitorPrefix != "" {
		job.SetMonitorURL(s.asyncMonitorPrefix + job.ID)
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

func (s *Service) acquireAsyncSlot() *queueToken {
	if s.asyncQueue == nil {
		return &queueToken{}
	}

	select {
	case s.asyncQueue <- struct{}{}:
		return &queueToken{ch: s.asyncQueue}
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
