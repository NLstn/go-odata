package odata

import "net/http"

// ServeHTTP implements http.Handler by delegating to the internal router.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.router == nil {
		http.Error(w, "service router not initialized", http.StatusInternalServerError)
		return
	}
	s.router.ServeHTTP(w, r)
}
