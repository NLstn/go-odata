package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// CachedCategory is a small lookup entity used by the caching tests.
type CachedCategory struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name"`
}

func setupCacheTestService(t *testing.T, cacheConfigs ...odata.EntityCacheConfig) (*gorm.DB, *odata.Service) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&CachedCategory{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Seed initial data
	categories := []CachedCategory{
		{ID: 1, Name: "Electronics"},
		{ID: 2, Name: "Books"},
		{ID: 3, Name: "Clothing"},
	}
	if err := db.Create(&categories).Error; err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&CachedCategory{}, cacheConfigs...); err != nil {
		t.Fatalf("failed to register entity: %v", err)
	}

	return db, service
}

// TestEntityCaching_RegisterWithCacheLevelNone verifies that CacheLevelNone is a no-op.
func TestEntityCaching_RegisterWithCacheLevelNone(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelNone,
	})

	req := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET failed with CacheLevelNone: %d %s", w.Code, w.Body.String())
	}
}

// TestEntityCaching_RegisterWithMultipleConfigs verifies that only one cache config
// can be provided during entity registration.
func TestEntityCaching_RegisterWithMultipleConfigs(t *testing.T) {
	_, service := setupCacheTestService(t)

	err := service.RegisterEntity(&CachedCategory{},
		odata.EntityCacheConfig{Level: odata.CacheLevelNone},
		odata.EntityCacheConfig{Level: odata.CacheLevelFull},
	)
	if err == nil {
		t.Fatal("expected error when registering entity with multiple cache configs, got nil")
	}
}

// TestEntityCaching_FullCacheReturnsAllEntities verifies that a cached entity set
// returns all entities correctly.
func TestEntityCaching_FullCacheReturnsAllEntities(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	req := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	values, ok := resp["value"].([]interface{})
	if !ok {
		t.Fatalf("expected 'value' array in response, got: %v", resp)
	}

	if len(values) != 3 {
		t.Errorf("expected 3 entities, got %d", len(values))
	}
}

// TestEntityCaching_FullCacheWithFilter verifies that OData $filter works correctly
// when data is served from the in-memory cache.
func TestEntityCaching_FullCacheWithFilter(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	req := httptest.NewRequest(http.MethodGet, "/CachedCategories?$filter=Name%20eq%20'Electronics'", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	values, ok := resp["value"].([]interface{})
	if !ok {
		t.Fatalf("expected 'value' array in response")
	}

	if len(values) != 1 {
		t.Errorf("expected 1 entity after filter, got %d", len(values))
	}
}

// TestEntityCaching_InvalidatedAfterCreate verifies that creating a new entity
// invalidates the cache so that subsequent reads include the new entity.
func TestEntityCaching_InvalidatedAfterCreate(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Hour, // long TTL so it doesn't expire between requests
	})

	// Warm the cache with an initial read.
	warmReq := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	warmW := httptest.NewRecorder()
	service.ServeHTTP(warmW, warmReq)
	if warmW.Code != http.StatusOK {
		t.Fatalf("initial GET failed: %d", warmW.Code)
	}

	// Create a new entity via POST.
	body := `{"ID":4,"Name":"Sports"}`
	postReq := httptest.NewRequest(http.MethodPost, "/CachedCategories", strings.NewReader(body))
	postReq.Header.Set("Content-Type", "application/json")
	postW := httptest.NewRecorder()
	service.ServeHTTP(postW, postReq)
	if postW.Code != http.StatusCreated {
		t.Fatalf("POST failed: %d %s", postW.Code, postW.Body.String())
	}

	// Read again — the new entity should now be present.
	getReq := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	getW := httptest.NewRecorder()
	service.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET after POST failed: %d", getW.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(getW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	values, ok := resp["value"].([]interface{})
	if !ok {
		t.Fatalf("expected 'value' array in response")
	}

	if len(values) != 4 {
		t.Errorf("expected 4 entities after POST, got %d", len(values))
	}
}

// TestEntityCaching_InvalidatedAfterDelete verifies that deleting an entity
// invalidates the cache so that subsequent reads exclude the deleted entity.
func TestEntityCaching_InvalidatedAfterDelete(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Hour,
	})

	// Warm the cache.
	warmReq := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	warmW := httptest.NewRecorder()
	service.ServeHTTP(warmW, warmReq)
	if warmW.Code != http.StatusOK {
		t.Fatalf("initial GET failed: %d", warmW.Code)
	}

	// Delete entity with ID=1.
	delReq := httptest.NewRequest(http.MethodDelete, "/CachedCategories(1)", nil)
	delW := httptest.NewRecorder()
	service.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusNoContent {
		t.Fatalf("DELETE failed: %d %s", delW.Code, delW.Body.String())
	}

	// Read again — the deleted entity should be gone.
	getReq := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	getW := httptest.NewRecorder()
	service.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET after DELETE failed: %d", getW.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(getW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	values, ok := resp["value"].([]interface{})
	if !ok {
		t.Fatalf("expected 'value' array in response")
	}

	if len(values) != 2 {
		t.Errorf("expected 2 entities after DELETE, got %d", len(values))
	}
}

// TestEntityCaching_InvalidatedAfterPatch verifies that patching an entity
// invalidates the cache so that subsequent reads reflect the update.
func TestEntityCaching_InvalidatedAfterPatch(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Hour,
	})

	// Warm the cache.
	warmReq := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	warmW := httptest.NewRecorder()
	service.ServeHTTP(warmW, warmReq)
	if warmW.Code != http.StatusOK {
		t.Fatalf("initial GET failed: %d", warmW.Code)
	}

	// Patch entity with ID=1.
	body := `{"Name":"Updated Electronics"}`
	patchReq := httptest.NewRequest(http.MethodPatch, "/CachedCategories(1)", strings.NewReader(body))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	service.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusNoContent && patchW.Code != http.StatusOK {
		t.Fatalf("PATCH failed: %d %s", patchW.Code, patchW.Body.String())
	}

	// Read collection again — the updated name should be present.
	getReq := httptest.NewRequest(http.MethodGet, "/CachedCategories?$filter=Name%20eq%20'Updated%20Electronics'", nil)
	getW := httptest.NewRecorder()
	service.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET after PATCH failed: %d", getW.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(getW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	values, ok := resp["value"].([]interface{})
	if !ok {
		t.Fatalf("expected 'value' array in response")
	}

	if len(values) != 1 {
		t.Errorf("expected 1 entity with updated name, got %d", len(values))
	}
}

// TestEntityCaching_TTLExpiry verifies that an expired cache is refreshed automatically
// on the next read.
func TestEntityCaching_TTLExpiry(t *testing.T) {
	// Short TTL to make the cache expire quickly in the test.
	db, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   100 * time.Millisecond,
	})

	// First read — warms the cache.
	req1 := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	w1 := httptest.NewRecorder()
	service.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first GET failed: %d", w1.Code)
	}

	// Directly insert into the primary DB, bypassing the OData service so the
	// cache is NOT explicitly invalidated.
	if err := db.Create(&CachedCategory{ID: 10, Name: "Toys"}).Error; err != nil {
		t.Fatalf("direct DB insert failed: %v", err)
	}

	// Sleep until the cache expires with some margin.
	time.Sleep(200 * time.Millisecond)

	// Second read — cache should be stale and refreshed from the primary DB.
	req2 := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	w2 := httptest.NewRecorder()
	service.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second GET failed: %d", w2.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	values, ok := resp["value"].([]interface{})
	if !ok {
		t.Fatalf("expected 'value' array in response")
	}

	if len(values) != 4 {
		t.Errorf("expected 4 entities after TTL expiry, got %d", len(values))
	}
}

// TestEntityCaching_TopAndSkip verifies that $top and $skip work correctly
// when data is served from the cache.
func TestEntityCaching_TopAndSkip(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   time.Minute,
	})

	req := httptest.NewRequest(http.MethodGet, "/CachedCategories?$top=2&$skip=1", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	values, ok := resp["value"].([]interface{})
	if !ok {
		t.Fatalf("expected 'value' array in response")
	}

	if len(values) != 2 {
		t.Errorf("expected 2 entities with $top=2&$skip=1, got %d", len(values))
	}
}

// TestEntityCaching_DefaultTTLUsedWhenZero verifies that a zero TTL defaults to
// 5 minutes without returning an error.
func TestEntityCaching_DefaultTTLUsedWhenZero(t *testing.T) {
	_, service := setupCacheTestService(t, odata.EntityCacheConfig{
		Level: odata.CacheLevelFull,
		TTL:   0, // should default to 5 minutes
	})

	req := httptest.NewRequest(http.MethodGet, "/CachedCategories", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET failed after zero TTL setup: %d %s", w.Code, w.Body.String())
	}
}
