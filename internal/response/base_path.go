package response

import "sync"

var (
	basePath   string
	basePathMu sync.RWMutex
)

// SetBasePath configures the base path for URL generation.
// This is called by the Service when SetBasePath() is configured.
// Thread-safe for concurrent access.
func SetBasePath(path string) {
	basePathMu.Lock()
	basePath = path
	basePathMu.Unlock()
}

// getBasePath returns the configured base path.
// Returns empty string if not configured.
// Thread-safe for concurrent access.
func getBasePath() string {
	basePathMu.RLock()
	defer basePathMu.RUnlock()
	return basePath
}
