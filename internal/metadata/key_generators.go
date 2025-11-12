package metadata

import (
	"strings"
	"sync"
)

var (
	keyGeneratorNamesMu sync.RWMutex
	keyGeneratorNames   = make(map[string]struct{})
)

// RegisterKeyGeneratorName registers a key generator name so metadata analysis can validate usage.
func RegisterKeyGeneratorName(name string) {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "" {
		return
	}

	keyGeneratorNamesMu.Lock()
	defer keyGeneratorNamesMu.Unlock()
	keyGeneratorNames[trimmed] = struct{}{}
}

// KnownKeyGeneratorName reports whether the provided generator name has been registered.
func KnownKeyGeneratorName(name string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "" {
		return false
	}

	keyGeneratorNamesMu.RLock()
	defer keyGeneratorNamesMu.RUnlock()
	_, ok := keyGeneratorNames[trimmed]
	return ok
}

func init() {
	RegisterKeyGeneratorName("uuid")
}
