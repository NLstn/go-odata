package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
)

// TestCacheSnapshot_FallbackForUnsupportedFilter verifies that a query using a
// filter function outside the in-memory subset (length()) still returns correct
// results by transparently falling back to the primary database.
func TestCacheSnapshot_FallbackForUnsupportedFilter(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	// Warm the cache first so the snapshot exists and the fallback branch is taken.
	warm := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	service.ServeHTTP(httptest.NewRecorder(), warm)

	// length(Name) eq 5 matches only "Books" (Electronics=11, Books=5, Clothing=8).
	req := httptest.NewRequest(http.MethodGet, "/CachedCategories?$filter=length(Name)%20eq%205", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	values := decodeValues(t, w)
	if len(values) != 1 {
		t.Fatalf("expected 1 entity for length(Name) eq 5, got %d", len(values))
	}
	if name, _ := values[0].(map[string]interface{})["Name"].(string); name != "Books" {
		t.Fatalf("expected Books, got %q", name)
	}
}

// TestCacheSnapshot_OrderByDescFromCache verifies ordering is applied in memory.
func TestCacheSnapshot_OrderByDescFromCache(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	req := httptest.NewRequest(http.MethodGet, "/CachedCategories?$orderby=Name%20desc", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	values := decodeValues(t, w)
	if len(values) != 3 {
		t.Fatalf("expected 3 entities, got %d", len(values))
	}
	got := make([]string, len(values))
	for i, v := range values {
		got[i], _ = v.(map[string]interface{})["Name"].(string)
	}
	want := []string{"Electronics", "Clothing", "Books"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("orderby desc: expected %v, got %v", want, got)
		}
	}
}

// TestCacheSnapshot_InFilterFromCache verifies the in operator is served in memory.
func TestCacheSnapshot_InFilterFromCache(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	req := httptest.NewRequest(http.MethodGet, "/CachedCategories?$filter=Name%20in%20('Books','Clothing')&$orderby=Name", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	values := decodeValues(t, w)
	if len(values) != 2 {
		t.Fatalf("expected 2 entities for in filter, got %d", len(values))
	}
}

// TestCacheSnapshot_KeyMissReturns404 verifies a key not present in a warm
// snapshot yields a 404 without falling back to the database.
func TestCacheSnapshot_KeyMissReturns404(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	// Warm the cache.
	service.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/CachedCategories", nil))

	req := httptest.NewRequest(http.MethodGet, "/CachedCategories(9999)", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing key, got %d: %s", w.Code, w.Body.String())
	}
}

// TestCacheSnapshot_CountFromCache verifies $count is served from the snapshot
// and reflects the filter.
func TestCacheSnapshot_CountFromCache(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	req := httptest.NewRequest(http.MethodGet, "/CachedCategories?$count=true&$filter=Name%20ne%20'Books'", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	count, ok := resp["@odata.count"]
	if !ok {
		t.Fatalf("expected @odata.count in response: %v", resp)
	}
	if n, _ := count.(float64); n != 2 {
		t.Fatalf("expected count 2, got %v", count)
	}
}

// TestCacheSnapshot_ConcurrentReads drives many concurrent cached reads to
// exercise the lock-free snapshot path; run with -race.
func TestCacheSnapshot_ConcurrentReads(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	// Warm the cache.
	service.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/CachedCategories", nil))

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/CachedCategories?$filter=Name%20eq%20'Electronics'", nil)
			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("concurrent read failed: %d", w.Code)
				return
			}
			if len(decodeValues(t, w)) != 1 {
				t.Errorf("expected 1 result from concurrent read")
			}
		}()
	}
	wg.Wait()
}

func decodeValues(t *testing.T, w *httptest.ResponseRecorder) []interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	values, ok := resp["value"].([]interface{})
	if !ok {
		t.Fatalf("expected 'value' array, got: %v", resp)
	}
	return values
}
