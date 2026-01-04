package odata

import (
	"net/http"

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

	// Call the pre-request hook if configured
	if s.preRequestHook != nil {
		ctx, err := s.preRequestHook(r)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusForbidden, "Forbidden", err.Error()); writeErr != nil {
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
