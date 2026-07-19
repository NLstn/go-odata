package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

// TestCacheSnapshot_AndOrFilterFromCache verifies a nested AND/OR filter is
// evaluated correctly from the in-memory snapshot's prepared filter tree.
func TestCacheSnapshot_AndOrFilterFromCache(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	// (ID eq 1 or ID eq 3) and Name ne 'Clothing' -> only Electronics (ID 1).
	req := httptest.NewRequest(http.MethodGet, "/CachedCategories?$filter=(ID%20eq%201%20or%20ID%20eq%203)%20and%20Name%20ne%20'Clothing'", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	values := decodeValues(t, w)
	if len(values) != 1 {
		t.Fatalf("expected 1 entity, got %d: %v", len(values), values)
	}
	if name, _ := values[0].(map[string]interface{})["Name"].(string); name != "Electronics" {
		t.Fatalf("expected Electronics, got %q", name)
	}
}

// TestCacheSnapshot_NotFilterFromCache verifies a negated leaf comparison is
// evaluated correctly from the in-memory snapshot's prepared filter tree.
func TestCacheSnapshot_NotFilterFromCache(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	req := httptest.NewRequest(http.MethodGet, "/CachedCategories?$filter=not%20(Name%20eq%20'Books')&$orderby=Name", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	values := decodeValues(t, w)
	if len(values) != 2 {
		t.Fatalf("expected 2 entities, got %d: %v", len(values), values)
	}
	got := make([]string, len(values))
	for i, v := range values {
		got[i], _ = v.(map[string]interface{})["Name"].(string)
	}
	want := []string{"Clothing", "Electronics"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("not filter: expected %v, got %v", want, got)
		}
	}
}

// CachedRanking has a deliberately duplicated Group value so multi-key
// $orderby exercises the tie-breaking second key (see sortMatches).
type CachedRanking struct {
	ID    uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Group string `json:"Group"`
	Score int    `json:"Score"`
}

// TestCacheSnapshot_MultiKeyOrderByTieBreak verifies that $orderby with more
// than one property breaks ties using the later keys, from the in-memory
// snapshot's precomputed, prepared orderby.
func TestCacheSnapshot_MultiKeyOrderByTieBreak(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(&CachedRanking{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}
	rankings := []CachedRanking{
		{ID: 1, Group: "A", Score: 10},
		{ID: 2, Group: "A", Score: 30},
		{ID: 3, Group: "B", Score: 20},
	}
	if err := db.Create(&rankings).Error; err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	if err := service.RegisterEntity(&CachedRanking{}, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	}); err != nil {
		t.Fatalf("failed to register entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/CachedRankings?$orderby=Group%20asc,Score%20desc", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	values := decodeValues(t, w)
	if len(values) != 3 {
		t.Fatalf("expected 3 entities, got %d: %v", len(values), values)
	}
	gotIDs := make([]float64, len(values))
	for i, v := range values {
		gotIDs[i], _ = v.(map[string]interface{})["ID"].(float64)
	}
	// Group A first (asc), Score 30 before 10 within group A (desc), then Group B.
	want := []float64{2, 1, 3}
	for i := range want {
		if gotIDs[i] != want[i] {
			t.Fatalf("multi-key orderby: expected IDs %v, got %v", want, gotIDs)
		}
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
