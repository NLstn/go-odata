package handlers

import (
	"context"
	"fmt"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type cacheTestProduct struct {
	ID   int    `json:"id" odata:"key"`
	Name string `json:"name"`
}

type cacheTestStorage struct {
	fetchEntityCount     int
	fetchCollectionCount int

	entityResult     interface{}
	collectionResult interface{}
}

func (s *cacheTestStorage) FetchEntityByKey(_ context.Context, _ *EntityHandler, _ string, _ *query.QueryOptions, _ []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	s.fetchEntityCount++
	return s.entityResult, nil
}

func (s *cacheTestStorage) FetchCollection(_ context.Context, _ *EntityHandler, _ *query.QueryOptions, _ []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	s.fetchCollectionCount++
	return s.collectionResult, nil
}

func (s *cacheTestStorage) CountEntities(_ context.Context, _ *EntityHandler, _ *query.QueryOptions, _ []func(*gorm.DB) *gorm.DB) (int64, error) {
	return 0, nil
}

func (s *cacheTestStorage) Create(_ *gorm.DB, _ *EntityHandler, _ interface{}) error {
	return nil
}

func (s *cacheTestStorage) UpdatePartial(_ *gorm.DB, _ *EntityHandler, _ interface{}, _ map[string]interface{}) error {
	return nil
}

func (s *cacheTestStorage) UpdateFull(_ *gorm.DB, _ *EntityHandler, _ interface{}, _ interface{}) error {
	return nil
}

func (s *cacheTestStorage) Delete(_ *gorm.DB, _ *EntityHandler, _ interface{}) error {
	return nil
}

func (s *cacheTestStorage) Refresh(_ *gorm.DB, _ *EntityHandler, _ interface{}) error {
	return nil
}

func TestLocalCacheStorageCachesEntityReads(t *testing.T) {
	meta := mustAnalyzeCacheTestEntity(t)
	h := NewEntityHandler(nil, meta, nil)

	base := &cacheTestStorage{
		entityResult: &cacheTestProduct{ID: 1, Name: "from-db"},
	}
	cache := NewLocalCacheStorage(base, LocalCacheStorageOptions{}).(*LocalCacheStorage)
	entityKey := namedEntityKey(meta, 1)

	first, err := cache.FetchEntityByKey(context.Background(), h, entityKey, &query.QueryOptions{}, nil)
	require.NoError(t, err)
	second, err := cache.FetchEntityByKey(context.Background(), h, entityKey, &query.QueryOptions{}, nil)
	require.NoError(t, err)

	require.Equal(t, 1, base.fetchEntityCount)
	require.IsType(t, &cacheTestProduct{}, first)
	require.IsType(t, &cacheTestProduct{}, second)
	require.Equal(t, "from-db", first.(*cacheTestProduct).Name)
	require.Equal(t, "from-db", second.(*cacheTestProduct).Name)
}

func TestLocalCacheStorageInvalidatesCollectionsAndUpsertsEntityOnChange(t *testing.T) {
	meta := mustAnalyzeCacheTestEntity(t)
	h := NewEntityHandler(nil, meta, nil)

	base := &cacheTestStorage{
		entityResult:     &cacheTestProduct{ID: 1, Name: "stale-db"},
		collectionResult: []cacheTestProduct{{ID: 1, Name: "from-db"}},
	}
	cache := NewLocalCacheStorage(base, LocalCacheStorageOptions{}).(*LocalCacheStorage)

	_, err := cache.FetchCollection(context.Background(), h, &query.QueryOptions{}, nil)
	require.NoError(t, err)
	_, err = cache.FetchCollection(context.Background(), h, &query.QueryOptions{}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, base.fetchCollectionCount)

	cache.OnEntityChanged(h, &cacheTestProduct{ID: 1, Name: "fresh-cache"}, trackchanges.ChangeTypeUpdated)
	require.Equal(t,
		canonicalEntityKeyFromRaw(namedEntityKey(meta, 1), meta.KeyProperties),
		canonicalEntityKeyFromEntity(&cacheTestProduct{ID: 1, Name: "fresh-cache"}, meta),
	)

	_, err = cache.FetchCollection(context.Background(), h, &query.QueryOptions{}, nil)
	require.NoError(t, err)
	require.Equal(t, 2, base.fetchCollectionCount)

	entity, err := cache.FetchEntityByKey(context.Background(), h, namedEntityKey(meta, 1), &query.QueryOptions{}, nil)
	require.NoError(t, err)
	require.Equal(t, 0, base.fetchEntityCount)
	require.Equal(t, "fresh-cache", entityName(entity))
}

func TestLocalCacheStorageWarmEntitySet(t *testing.T) {
	meta := mustAnalyzeCacheTestEntity(t)
	h := NewEntityHandler(nil, meta, nil)

	base := &cacheTestStorage{
		collectionResult: []cacheTestProduct{{ID: 1, Name: "warm"}},
	}
	cache := NewLocalCacheStorage(base, LocalCacheStorageOptions{WarmEntitySets: []string{meta.EntitySetName}}).(*LocalCacheStorage)

	err := cache.WarmEntitySet(context.Background(), h)
	require.NoError(t, err)
	require.Equal(t, 1, base.fetchCollectionCount)

	_, err = cache.FetchCollection(context.Background(), h, &query.QueryOptions{}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, base.fetchCollectionCount)
}

func TestLocalCacheStorageReconcileEntitySetRefreshesCache(t *testing.T) {
	meta := mustAnalyzeCacheTestEntity(t)
	h := NewEntityHandler(nil, meta, nil)

	base := &cacheTestStorage{
		collectionResult: []cacheTestProduct{{ID: 1, Name: "initial"}},
	}
	cache := NewLocalCacheStorage(base, LocalCacheStorageOptions{}).(*LocalCacheStorage)

	_, err := cache.FetchCollection(context.Background(), h, &query.QueryOptions{}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, base.fetchCollectionCount)

	base.collectionResult = []cacheTestProduct{{ID: 1, Name: "reconciled"}}
	err = cache.ReconcileEntitySet(context.Background(), h)
	require.NoError(t, err)
	require.Equal(t, 2, base.fetchCollectionCount)

	entity, err := cache.FetchEntityByKey(context.Background(), h, namedEntityKey(meta, 1), &query.QueryOptions{}, nil)
	require.NoError(t, err)
	require.Equal(t, "reconciled", entityName(entity))
	require.Equal(t, 0, base.fetchEntityCount)
}

func TestLocalCacheStorageRespectsEntryLimits(t *testing.T) {
	meta := mustAnalyzeCacheTestEntity(t)
	cache := NewLocalCacheStorage(&cacheTestStorage{}, LocalCacheStorageOptions{
		MaxEntityEntries:     1,
		MaxCollectionEntries: 1,
		MaxCountEntries:      1,
	}).(*LocalCacheStorage)

	entityKeyOne := cache.buildEntityCacheKey(meta.EntitySetName, canonicalEntityKeyFromRaw(namedEntityKey(meta, 1), meta.KeyProperties))
	entityKeyTwo := cache.buildEntityCacheKey(meta.EntitySetName, canonicalEntityKeyFromRaw(namedEntityKey(meta, 2), meta.KeyProperties))

	cache.setEntity(meta.EntitySetName, entityKeyOne, &cacheTestProduct{ID: 1, Name: "one"})
	cache.setEntity(meta.EntitySetName, entityKeyTwo, &cacheTestProduct{ID: 2, Name: "two"})
	require.Len(t, cache.entityByKey, 1)
	_, stillHasFirst := cache.entityByKey[entityKeyOne]
	require.False(t, stillHasFirst)

	collectionKeyOne := cache.buildCollectionCacheKey(meta.EntitySetName, &query.QueryOptions{Top: ptrInt(1)})
	collectionKeyTwo := cache.buildCollectionCacheKey(meta.EntitySetName, &query.QueryOptions{Top: ptrInt(2)})
	cache.setCollection(meta.EntitySetName, collectionKeyOne, []cacheTestProduct{{ID: 1}})
	cache.setCollection(meta.EntitySetName, collectionKeyTwo, []cacheTestProduct{{ID: 2}})
	require.Len(t, cache.collectionByKey, 1)
	_, stillHasCollectionOne := cache.collectionByKey[collectionKeyOne]
	require.False(t, stillHasCollectionOne)

	countKeyOne := cache.buildCountCacheKey(meta.EntitySetName, &query.QueryOptions{Top: ptrInt(1)})
	countKeyTwo := cache.buildCountCacheKey(meta.EntitySetName, &query.QueryOptions{Top: ptrInt(2)})
	cache.setCount(meta.EntitySetName, countKeyOne, 1)
	cache.setCount(meta.EntitySetName, countKeyTwo, 2)
	require.Len(t, cache.countByKey, 1)
	_, stillHasCountOne := cache.countByKey[countKeyOne]
	require.False(t, stillHasCountOne)
}

func TestLocalCacheStorageEnabledEntitySetsScope(t *testing.T) {
	meta := mustAnalyzeCacheTestEntity(t)
	h := NewEntityHandler(nil, meta, nil)

	base := &cacheTestStorage{
		entityResult: &cacheTestProduct{ID: 1, Name: "from-db"},
	}
	cache := NewLocalCacheStorage(base, LocalCacheStorageOptions{
		EnabledEntitySets: []string{"AnotherSet"},
	}).(*LocalCacheStorage)

	entityKey := namedEntityKey(meta, 1)
	_, err := cache.FetchEntityByKey(context.Background(), h, entityKey, &query.QueryOptions{}, nil)
	require.NoError(t, err)
	_, err = cache.FetchEntityByKey(context.Background(), h, entityKey, &query.QueryOptions{}, nil)
	require.NoError(t, err)

	require.Equal(t, 2, base.fetchEntityCount)
}

func ptrInt(v int) *int {
	return &v
}

func mustAnalyzeCacheTestEntity(t *testing.T) *metadata.EntityMetadata {
	t.Helper()
	meta, err := metadata.AnalyzeEntity(&cacheTestProduct{})
	require.NoError(t, err)
	return meta
}

func entityName(entity interface{}) string {
	switch e := entity.(type) {
	case cacheTestProduct:
		return e.Name
	case *cacheTestProduct:
		return e.Name
	default:
		return ""
	}
}

func namedEntityKey(meta *metadata.EntityMetadata, id int) string {
	return fmt.Sprintf("%s=%d", meta.KeyProperties[0].JsonName, id)
}
