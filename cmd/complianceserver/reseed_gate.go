package main

import (
	"net/http"
	"strings"
	"sync"
)

// reseedGate prevents destructive database reseeding from overlapping requests
// that use the same compliance-server database. The compliance suite may run
// suites concurrently, so a reseed must wait for active requests and block new
// ones until the schema and reference data are ready again.
type reseedGate struct {
	next http.Handler
	mu   sync.RWMutex
}

func newReseedGate(next http.Handler) http.Handler {
	return &reseedGate{next: next}
}

func (g *reseedGate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && strings.Trim(r.URL.Path, "/") == "Reseed" {
		g.mu.Lock()
		defer g.mu.Unlock()
	} else {
		g.mu.RLock()
		defer g.mu.RUnlock()
	}

	g.next.ServeHTTP(w, r)
}
