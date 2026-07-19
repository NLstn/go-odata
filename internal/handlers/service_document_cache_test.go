package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

func serviceDocTestHandler(t *testing.T) *ServiceDocumentHandler {
	t.Helper()
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	meta2, _ := metadata.AnalyzeEntity(TestEntity{})
	singleton, _ := metadata.AnalyzeSingleton(ServiceDocumentTestEntity{}, "Settings")

	return NewServiceDocumentHandler(map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
		"TestEntities":                meta2,
		"Settings":                    singleton,
	}, nil)
}

func getServiceDoc(t *testing.T, h *ServiceDocumentHandler, target string) []byte {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	h.HandleServiceDocument(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}
	return w.Body.Bytes()
}

// TestServiceDocumentCache_ReusedAcrossRequests verifies that the cached "value"
// array is built once and the same bytes are reused for repeated requests.
func TestServiceDocumentCache_ReusedAcrossRequests(t *testing.T) {
	h := serviceDocTestHandler(t)

	if h.cachedValueJSON.Load() != nil {
		t.Fatal("cache should be empty before the first request")
	}

	first := getServiceDoc(t, h, "http://example.com/")

	cached := h.cachedValueJSON.Load()
	if cached == nil {
		t.Fatal("cache should be populated after the first request")
	}

	second := getServiceDoc(t, h, "http://example.com/")
	if string(first) != string(second) {
		t.Errorf("repeated requests produced different bodies:\n%s\n%s", first, second)
	}

	// The cached pointer must be unchanged after the second request (no rebuild).
	if h.cachedValueJSON.Load() != cached {
		t.Error("expected the cached value to be reused, not rebuilt")
	}
}

// TestServiceDocumentCache_VariesByBaseURL verifies that the cached value array
// is spliced with the request base URL, so different hosts get correct
// @odata.context values from the same cache.
func TestServiceDocumentCache_VariesByBaseURL(t *testing.T) {
	h := serviceDocTestHandler(t)

	decodeContext := func(body []byte) string {
		var resp map[string]interface{}
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		ctx, _ := resp["@odata.context"].(string)
		return ctx
	}

	a := decodeContext(getServiceDoc(t, h, "http://a.example.com/"))
	b := decodeContext(getServiceDoc(t, h, "http://b.example.com/"))

	if a != "http://a.example.com/$metadata" {
		t.Errorf("context = %q, want http://a.example.com/$metadata", a)
	}
	if b != "http://b.example.com/$metadata" {
		t.Errorf("context = %q, want http://b.example.com/$metadata", b)
	}
}

// TestServiceDocumentCache_ClearCacheRebuilds verifies ClearCache forces a
// rebuild that reflects the current entity set.
func TestServiceDocumentCache_ClearCacheRebuilds(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	entities := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
	}
	h := NewServiceDocumentHandler(entities, nil)

	countEntitySets := func(body []byte) int {
		var resp map[string]interface{}
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		value, _ := resp["value"].([]interface{})
		return len(value)
	}

	if got := countEntitySets(getServiceDoc(t, h, "http://example.com/")); got != 1 {
		t.Fatalf("expected 1 entity set, got %d", got)
	}

	// Add an entity to the shared map, then invalidate the cache.
	meta2, _ := metadata.AnalyzeEntity(TestEntity{})
	entities["TestEntities"] = meta2

	// Without invalidation the stale cache is served.
	if got := countEntitySets(getServiceDoc(t, h, "http://example.com/")); got != 1 {
		t.Errorf("expected stale cache to still report 1 entity set, got %d", got)
	}

	h.ClearCache()
	if got := countEntitySets(getServiceDoc(t, h, "http://example.com/")); got != 2 {
		t.Errorf("expected 2 entity sets after ClearCache, got %d", got)
	}
}

// TestServiceDocumentCache_HeadMatchesGet verifies HEAD reports the Content-Length
// that a GET body would have, and returns no body.
func TestServiceDocumentCache_HeadMatchesGet(t *testing.T) {
	h := serviceDocTestHandler(t)

	getBody := getServiceDoc(t, h, "http://example.com/")

	req := httptest.NewRequest(http.MethodHead, "http://example.com/", nil)
	w := httptest.NewRecorder()
	h.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}
	if w.Body.Len() != 0 {
		t.Errorf("HEAD body should be empty, got %d bytes", w.Body.Len())
	}
	if got, want := w.Header().Get("Content-Length"), len(getBody); got != strconv.Itoa(want) {
		t.Errorf("HEAD Content-Length = %q, want %d", got, want)
	}
}
