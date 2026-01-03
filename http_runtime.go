package odata

import "net/http"

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
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if ctx != nil {
			r = r.WithContext(ctx)
		}
	}

	s.runtime.ServeHTTP(w, r, allowAsync)
}
