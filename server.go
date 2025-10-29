package odata

import "net/http"

// Handler returns the Service as an http.Handler.
// This method provides an explicit way to use the Service as a handler,
// though the Service already implements http.Handler directly.
func (s *Service) Handler() http.Handler {
	return s
}
