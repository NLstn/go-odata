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
	s.runtime.ServeHTTP(w, r, allowAsync)
}
