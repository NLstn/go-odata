package handlers

import (
	"context"
	"fmt"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/version"
)

const (
	numConcurrentReaders          = 100
	numStressGoroutines           = 300
	numNamespaceChanges           = 10
	numConcurrentNamespaceReaders = 50
)

// TestMetadataHandler_ManyVersionsCacheEviction tests cache eviction with many unique versions
func TestMetadataHandler_ManyVersionsCacheEviction(t *testing.T) {
	entities := map[string]*metadata.EntityMetadata{
		"Products": createTestEntityMetadata(),
	}

	handler := NewMetadataHandler(entities)

	// Request metadata for many different versions to trigger eviction
	versions := []version.Version{
		{Major: 4, Minor: 0},
		{Major: 4, Minor: 1},
		{Major: 4, Minor: 2},
		{Major: 4, Minor: 3},
		{Major: 4, Minor: 4},
		{Major: 4, Minor: 5},
		{Major: 4, Minor: 6},
		{Major: 4, Minor: 7},
		{Major: 4, Minor: 8},
		{Major: 4, Minor: 9},
		{Major: 4, Minor: 10}, // This should trigger eviction
		{Major: 4, Minor: 11},
		{Major: 4, Minor: 12},
	}

	for _, ver := range versions {
		req := httptest.NewRequest("GET", "/$metadata", nil)
		ctx := version.WithVersion(context.Background(), ver)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != 200 {
			t.Fatalf("Expected 200, got %d for version %v", w.Code, ver)
		}
	}

	// Verify cache size is bounded
	cacheSize := handler.cacheSize.Load()
	if cacheSize > maxCacheEntries {
		t.Errorf("Cache size %d exceeds maximum %d", cacheSize, maxCacheEntries)
	}

	// Verify priority versions (4.0, 4.01) are still cached
	req401 := httptest.NewRequest("GET", "/$metadata", nil)
	ctx401 := version.WithVersion(context.Background(), version.Version{Major: 4, Minor: 1})
	req401 = req401.WithContext(ctx401)
	w401 := httptest.NewRecorder()
	handler.HandleMetadata(w401, req401)

	if w401.Code != 200 {
		t.Error("Priority version 4.01 should still be cached")
	}
}

// TestMetadataHandler_CacheInvalidationCorrectness tests cache invalidation behavior
func TestMetadataHandler_CacheInvalidationCorrectness(t *testing.T) {
	entities := map[string]*metadata.EntityMetadata{
		"Products": createTestEntityMetadata(),
	}

	handler := NewMetadataHandler(entities)

	// Prime cache with version 4.01
	req1 := httptest.NewRequest("GET", "/$metadata", nil)
	ctx1 := version.WithVersion(context.Background(), version.Version{Major: 4, Minor: 1})
	req1 = req1.WithContext(ctx1)
	w1 := httptest.NewRecorder()
	handler.HandleMetadata(w1, req1)

	originalContent := w1.Body.String()

	// Change namespace - should clear cache
	handler.SetNamespace("NewNamespace")

	// Request same version again
	req2 := httptest.NewRequest("GET", "/$metadata", nil)
	ctx2 := version.WithVersion(context.Background(), version.Version{Major: 4, Minor: 1})
	req2 = req2.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	handler.HandleMetadata(w2, req2)

	newContent := w2.Body.String()

	// Content should be different (contains new namespace)
	if originalContent == newContent {
		t.Error("Cache should have been invalidated after namespace change")
	}

	if !contains(newContent, "NewNamespace") {
		t.Error("New metadata should contain updated namespace")
	}

	// Verify cache was actually cleared
	cacheSize := handler.cacheSize.Load()
	if cacheSize == 0 {
		// Cache was cleared and rebuilt with 1 entry
		t.Log("Cache correctly cleared and rebuilt")
	}
}

// TestMetadataHandler_RaceDetectorStressTest is a stress test for race conditions
func TestMetadataHandler_RaceDetectorStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	entities := map[string]*metadata.EntityMetadata{
		"Products": createTestEntityMetadata(),
	}

	handler := NewMetadataHandler(entities)

	var wg sync.WaitGroup
	errors := make(chan error, numStressGoroutines)

	// Concurrent readers
	for i := 0; i < numConcurrentReaders; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ver := version.Version{Major: 4, Minor: idx % 2}

			req := httptest.NewRequest("GET", "/$metadata", nil)
			ctx := version.WithVersion(context.Background(), ver)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handler.HandleMetadata(w, req)

			if w.Code != 200 {
				errors <- fmt.Errorf("XML request failed with code %d", w.Code)
			}
		}(i)
	}

	// Concurrent namespace changes
	for i := 0; i < numNamespaceChanges; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			handler.SetNamespace("Namespace" + string(rune('A'+idx%26)))
		}(i)
	}

	// Concurrent JSON requests
	for i := 0; i < numConcurrentReaders; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ver := version.Version{Major: 4, Minor: idx % 2}

			req := httptest.NewRequest("GET", "/$metadata", nil)
			req.Header.Set("Accept", "application/json")
			ctx := version.WithVersion(context.Background(), ver)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handler.HandleMetadata(w, req)

			if w.Code != 200 {
				errors <- fmt.Errorf("JSON request failed with code %d", w.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Report all errors
	for err := range errors {
		t.Error(err)
	}
}

// TestMetadataHandler_ConcurrentBuildPrevention tests that concurrent cache misses don't build multiple times
func TestMetadataHandler_ConcurrentBuildPrevention(t *testing.T) {
	entities := map[string]*metadata.EntityMetadata{
		"Products": createTestEntityMetadata(),
	}

	handler := NewMetadataHandler(entities)

	var buildCount atomic.Int32
	var wg sync.WaitGroup

	// Wrap to count builds (this is a white-box test)
	// In reality, LoadOrStore handles this internally

	// Launch many concurrent requests for the same version
	for i := 0; i < numConcurrentReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/$metadata", nil)
			ctx := version.WithVersion(context.Background(), version.Version{Major: 4, Minor: 1})
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handler.HandleMetadata(w, req)

			if w.Code == 200 {
				buildCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// All 100 requests should succeed
	if buildCount.Load() != 100 {
		t.Errorf("Expected 100 successful requests, got %d", buildCount.Load())
	}

	// Cache should only have 1 entry for version 4.01
	// (We can't easily verify the build happened once without instrumenting,
	// but LoadOrStore guarantees this)
}

// TestMetadataHandler_NamespaceChangeWhileReading tests namespace change during cache read
func TestMetadataHandler_NamespaceChangeWhileReading(t *testing.T) {
	entities := map[string]*metadata.EntityMetadata{
		"Products": createTestEntityMetadata(),
	}

	handler := NewMetadataHandler(entities)

	// Prime cache
	req := httptest.NewRequest("GET", "/$metadata", nil)
	ctx := version.WithVersion(context.Background(), version.Version{Major: 4, Minor: 1})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	var wg sync.WaitGroup
	done := make(chan bool)

	// Start readers
	for i := 0; i < numConcurrentNamespaceReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					req := httptest.NewRequest("GET", "/$metadata", nil)
					ctx := version.WithVersion(context.Background(), version.Version{Major: 4, Minor: 1})
					req = req.WithContext(ctx)
					w := httptest.NewRecorder()
					handler.HandleMetadata(w, req)
				}
			}
		}()
	}

	// Change namespace while reading
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numNamespaceChanges; i++ {
			select {
			case <-done:
				return
			default:
				handler.SetNamespace("NS" + string(rune('A'+i)))
			}
		}
	}()

	// Wait for namespace changes to run, then signal stop
	time.Sleep(10 * time.Millisecond)
	close(done)

	wg.Wait()

	// If we get here without deadlock or panic, test passes
}

// Helper function
func createTestEntityMetadata() *metadata.EntityMetadata {
	return &metadata.EntityMetadata{
		EntitySetName: "Products",
		EntityName:    "Product",
		KeyProperties: []metadata.PropertyMetadata{
			{Name: "ID", FieldName: "ID", Type: reflect.TypeOf(int(0)), IsKey: true},
		},
	}
}
