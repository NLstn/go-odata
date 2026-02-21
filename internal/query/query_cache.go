package query

import (
	"net/url"
	"sync"
)

// queryValuesCache is a simple bounded cache that maps raw query strings to their
// parsed url.Values. It is designed to be used across requests so that repeated
// identical query strings (common in load tests and batch API traffic) do not
// incur the cost of splitting, URL-unescaping, and map-building every time.
//
// Eviction strategy: when the cache reaches its capacity limit the entire map is
// replaced. This is simpler than a true LRU and sufficient for the target use-case
// (a small number of distinct query templates repeated many times).
//
// Thread safety: all public methods are safe for concurrent use.
type queryValuesCache struct {
	mu    sync.RWMutex
	items map[string]url.Values
	max   int
}

var globalQueryCache = &queryValuesCache{
	items: make(map[string]url.Values, 256),
	max:   256,
}

func (c *queryValuesCache) get(rawQuery string) (url.Values, bool) {
	c.mu.RLock()
	v, ok := c.items[rawQuery]
	c.mu.RUnlock()
	return v, ok
}

func (c *queryValuesCache) put(rawQuery string, v url.Values) {
	c.mu.Lock()
	if len(c.items) >= c.max {
		// Evict everything and start fresh rather than tracking individual entry ages.
		c.items = make(map[string]url.Values, c.max)
	}
	c.items[rawQuery] = v
	c.mu.Unlock()
}

// CachedParseRawQuery retrieves the parsed url.Values for rawQuery from the global
// cache, falling back to ParseRawQuery on a cache miss. The returned url.Values
// MUST NOT be modified by the caller because it may be shared with other goroutines.
// Callers that need to modify the map (e.g. resolveParameterAliases) already
// create their own copy before doing so, so this constraint is safe in practice.
func CachedParseRawQuery(rawQuery string) url.Values {
	if rawQuery == "" {
		return make(url.Values)
	}
	if v, ok := globalQueryCache.get(rawQuery); ok {
		return v
	}
	v := ParseRawQuery(rawQuery)
	globalQueryCache.put(rawQuery, v)
	return v
}
