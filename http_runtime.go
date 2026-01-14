package odata

import (
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/response"
)

// ServeHTTP implements http.Handler by delegating to the runtime.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.serveHTTP(w, r, true)
}

func (s *Service) serveHTTP(w http.ResponseWriter, r *http.Request, allowAsync bool) {
	if s.runtime == nil {
		http.Error(w, "service runtime not initialized", http.StatusInternalServerError)
		return
	}

	// Strip base path from incoming request if configured
	if s.basePath != "" && strings.HasPrefix(r.URL.Path, s.basePath) {
		newPath := strings.TrimPrefix(r.URL.Path, s.basePath)
		// Handle exact match of base path
		if newPath == "" {
			newPath = "/"
		}
		// Ensure we only strip if followed by "/" or exact match
		// This prevents /odatax/foo from being incorrectly stripped to x/foo
		if newPath == "/" || strings.HasPrefix(newPath, "/") {
			r.URL.Path = newPath
		}
	}

	// Call the pre-request hook if configured
	if s.preRequestHook != nil {
		ctx, err := s.preRequestHook(r)
		if err != nil {
			if writeErr := response.WriteError(w, r, http.StatusForbidden, "Forbidden", err.Error()); writeErr != nil {
				http.Error(w, writeErr.Error(), http.StatusInternalServerError)
			}
			return
		}
		if ctx != nil {
			r = r.WithContext(ctx)
		}
	}

	s.runtime.ServeHTTP(w, r, allowAsync)
}
