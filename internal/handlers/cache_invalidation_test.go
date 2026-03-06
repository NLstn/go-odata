package handlers

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCacheInvalidationPollerReplaysExternalEntityUpdate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	meta := mustAnalyzeCacheTestEntity(t)
	h := NewEntityHandler(db, meta, slog.Default())

	base := &cacheTestStorage{entityResult: &cacheTestProduct{ID: 1, Name: "from-db"}}
	cache := NewLocalCacheStorage(base, LocalCacheStorageOptions{}).(*LocalCacheStorage)
	h.SetStorage(cache)

	cache.OnEntityChanged(h, &cacheTestProduct{ID: 1, Name: "stale-local"}, trackchanges.ChangeTypeUpdated)

	logStore, err := NewDBCacheInvalidationLog(db, slog.Default())
	require.NoError(t, err)

	poller, err := NewCacheInvalidationPoller(
		db,
		slog.Default(),
		func(entitySet string) (*EntityHandler, bool) {
			if entitySet == h.metadata.EntitySetName {
				return h, true
			}
			return nil, false
		},
		CacheInvalidationPollerOptions{
			InstanceID:   "instance-a",
			PollInterval: 10 * time.Millisecond,
			BatchSize:    10,
			SkipOwnEvent: true,
		},
	)
	require.NoError(t, err)
	poller.Start()
	defer poller.Close()

	err = logStore.Append(context.Background(), CacheInvalidationEvent{
		EntitySet:      h.metadata.EntitySetName,
		ChangeType:     trackchanges.ChangeTypeUpdated,
		KeyValues:      map[string]interface{}{"id": 1},
		Data:           map[string]interface{}{"id": 1, "name": "from-peer"},
		SourceInstance: "instance-b",
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		entity, fetchErr := cache.FetchEntityByKey(context.Background(), h, namedEntityKey(meta, 1), &query.QueryOptions{}, nil)
		if fetchErr != nil {
			return false
		}
		return entityName(entity) == "from-peer"
	}, time.Second, 20*time.Millisecond)

	require.Equal(t, 0, base.fetchEntityCount)
}

func TestCacheInvalidationPollerSkipsOwnEvents(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	meta := mustAnalyzeCacheTestEntity(t)
	h := NewEntityHandler(db, meta, slog.Default())

	base := &cacheTestStorage{entityResult: &cacheTestProduct{ID: 1, Name: "from-db"}}
	cache := NewLocalCacheStorage(base, LocalCacheStorageOptions{}).(*LocalCacheStorage)
	h.SetStorage(cache)

	cache.OnEntityChanged(h, &cacheTestProduct{ID: 1, Name: "unchanged-local"}, trackchanges.ChangeTypeUpdated)

	logStore, err := NewDBCacheInvalidationLog(db, slog.Default())
	require.NoError(t, err)

	poller, err := NewCacheInvalidationPoller(
		db,
		slog.Default(),
		func(entitySet string) (*EntityHandler, bool) {
			if entitySet == h.metadata.EntitySetName {
				return h, true
			}
			return nil, false
		},
		CacheInvalidationPollerOptions{
			InstanceID:   "instance-a",
			PollInterval: 10 * time.Millisecond,
			BatchSize:    10,
			SkipOwnEvent: true,
		},
	)
	require.NoError(t, err)
	poller.Start()
	defer poller.Close()

	err = logStore.Append(context.Background(), CacheInvalidationEvent{
		EntitySet:      h.metadata.EntitySetName,
		ChangeType:     trackchanges.ChangeTypeUpdated,
		KeyValues:      map[string]interface{}{"id": 1},
		Data:           map[string]interface{}{"id": 1, "name": "own-event"},
		SourceInstance: "instance-a",
	})
	require.NoError(t, err)

	// Allow at least one poll cycle to process and checkpoint the skipped event.
	time.Sleep(80 * time.Millisecond)

	entity, err := cache.FetchEntityByKey(context.Background(), h, namedEntityKey(meta, 1), &query.QueryOptions{}, nil)
	require.NoError(t, err)
	require.Equal(t, "unchanged-local", entityName(entity))
	require.Equal(t, 0, base.fetchEntityCount)
}
