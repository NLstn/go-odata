package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// EntityCache provides a file-based SQLite-backed cache for a single entity type.
// It stores the full dataset from the primary database and serves read queries
// against the local cache copy, avoiding repeated round-trips to the primary database.
//
// Cache invalidation:
//   - Automatic: data expires after the configured TTL
//   - Manual: call Invalidate() after any write operation to force a refresh
//
// The cache is safe for concurrent use. Only one goroutine refreshes the cache at a
// time; concurrent refresh requests block until the first refresh completes, after
// which they reuse the freshly populated cache rather than triggering another fetch.
type EntityCache struct {
	mu         sync.Mutex
	refreshMu  sync.Mutex // serializes refresh operations to prevent thundering herd
	current    *cacheStore
	stale      []*cacheStore
	expiresAt  time.Time
	ttl        time.Duration
	cacheDir   string
	entityType reflect.Type
	valid      bool // whether the cache has been populated at least once
}

type cacheStore struct {
	db      *gorm.DB
	path    string
	readers int64
}

var cacheFileCounter uint64

// New creates a new EntityCache for the given entity type and TTL.
// entityType should be the non-pointer struct type of the entity.
func New(entityType reflect.Type, ttl time.Duration) (*EntityCache, error) {
	if entityType == nil {
		return nil, fmt.Errorf("entityType must not be nil")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("ttl must be positive")
	}

	cacheDir, err := os.MkdirTemp("", "go-odata-entity-cache-")
	if err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &EntityCache{
		ttl:        ttl,
		cacheDir:   cacheDir,
		entityType: entityType,
	}

	return cache, nil
}

// IsValid reports whether the cache contains fresh data that has not expired.
func (c *EntityCache) IsValid() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid && time.Now().Before(c.expiresAt)
}

// Invalidate marks the cache as stale so that the next read triggers a refresh
// from the primary database.
func (c *EntityCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.valid = false
}

// AcquireDB returns the currently active cache database with a release function.
// Callers must invoke release when done to allow safe cleanup of stale cache files.
func (c *EntityCache) AcquireDB() (*gorm.DB, func(), bool) {
	c.mu.Lock()
	store := c.current
	if !c.valid || store == nil {
		c.mu.Unlock()
		return nil, func() {}, false
	}
	store.readers++
	c.mu.Unlock()

	release := func() {
		c.mu.Lock()
		store.readers--
		c.cleanupStaleStoresLocked()
		c.mu.Unlock()
	}

	return store.db, release, true
}

// Refresh reloads the entire dataset from the primary database into the cache.
// It serialises concurrent refresh attempts so that only one fetch is performed
// at a time; subsequent callers reuse the result of the first fetch.
// It swaps in a freshly populated SQLite file atomically and resets the expiry timer.
func (c *EntityCache) Refresh(sourceDB *gorm.DB) error {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	// Another goroutine may have already refreshed the cache while we were waiting
	// for the refresh lock.  Reuse that result rather than doing an unnecessary fetch.
	if c.IsValid() {
		return nil
	}

	// Fetch all records from the primary database
	sliceType := reflect.SliceOf(c.entityType)
	entities := reflect.New(sliceType).Interface()
	if err := sourceDB.Find(entities).Error; err != nil {
		return fmt.Errorf("failed to load entities from primary database: %w", err)
	}

	newStore, err := c.createStore()
	if err != nil {
		return err
	}

	// Bulk-insert the fetched entities
	sliceVal := reflect.ValueOf(entities).Elem().Interface()
	if reflect.ValueOf(sliceVal).Len() > 0 {
		if err := newStore.db.Create(sliceVal).Error; err != nil {
			cleanupStore(newStore)
			return fmt.Errorf("failed to populate cache: %w", err)
		}
	}

	c.mu.Lock()
	if c.current != nil {
		c.stale = append(c.stale, c.current)
	}
	c.current = newStore
	c.expiresAt = time.Now().Add(c.ttl)
	c.valid = true
	c.cleanupStaleStoresLocked()
	c.mu.Unlock()

	return nil
}

func (c *EntityCache) createStore() (*cacheStore, error) {
	fileName := fmt.Sprintf("cache_%s_%d_%d.db", c.entityType.Name(), time.Now().UnixNano(), atomic.AddUint64(&cacheFileCounter, 1))
	path := filepath.Join(c.cacheDir, fileName)

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cache database: %w", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}

	entityPtr := reflect.New(c.entityType).Interface()
	if err := db.AutoMigrate(entityPtr); err != nil {
		cleanupStore(&cacheStore{db: db, path: path})
		return nil, fmt.Errorf("failed to migrate entity schema into cache: %w", err)
	}

	return &cacheStore{db: db, path: path}, nil
}

func (c *EntityCache) cleanupStaleStoresLocked() {
	if len(c.stale) == 0 {
		return
	}

	remaining := c.stale[:0]
	for _, store := range c.stale {
		if store.readers > 0 {
			remaining = append(remaining, store)
			continue
		}
		cleanupStore(store)
	}
	c.stale = remaining
}

func cleanupStore(store *cacheStore) {
	if store == nil {
		return
	}
	if store.db != nil {
		if sqlDB, err := store.db.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				// Best-effort cleanup; nothing actionable for callers.
				_ = err
			}
		}
	}
	if store.path != "" {
		if err := os.Remove(store.path); err != nil {
			// Best-effort cleanup; file may already be removed.
			_ = err
		}
	}
}
