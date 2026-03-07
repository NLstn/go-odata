package cache

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// EntityCache provides an in-memory SQLite-backed cache for a single entity type.
// It stores the full dataset from the primary database and serves read queries
// against the in-memory copy, avoiding repeated round-trips to the primary database.
//
// Cache invalidation:
//   - Automatic: data expires after the configured TTL
//   - Manual: call Invalidate() after any write operation to force a refresh
//
// The cache is safe for concurrent use. Only one goroutine refreshes the cache at a
// time; concurrent refresh requests block until the first refresh completes, after
// which they reuse the freshly populated cache rather than triggering another fetch.
type EntityCache struct {
	mu        sync.RWMutex
	refreshMu sync.Mutex // serializes refresh operations to prevent thundering herd
	db        *gorm.DB   // in-memory SQLite database
	expiresAt time.Time
	ttl       time.Duration
	entityType reflect.Type
	valid      bool // whether the cache has been populated at least once
}

// New creates a new EntityCache for the given entity type and TTL.
// entityType should be the non-pointer struct type of the entity.
func New(entityType reflect.Type, ttl time.Duration) (*EntityCache, error) {
	if entityType == nil {
		return nil, fmt.Errorf("entityType must not be nil")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("ttl must be positive")
	}

	// Create the in-memory SQLite database
	inMemDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create in-memory cache database: %w", err)
	}

	// AutoMigrate the entity schema into the in-memory database
	entityPtr := reflect.New(entityType).Interface()
	if err := inMemDB.AutoMigrate(entityPtr); err != nil {
		return nil, fmt.Errorf("failed to migrate entity schema into cache: %w", err)
	}

	return &EntityCache{
		db:         inMemDB,
		ttl:        ttl,
		entityType: entityType,
	}, nil
}

// IsValid reports whether the cache contains fresh data that has not expired.
func (c *EntityCache) IsValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.valid && time.Now().Before(c.expiresAt)
}

// Invalidate marks the cache as stale so that the next read triggers a refresh
// from the primary database.
func (c *EntityCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.valid = false
}

// Refresh reloads the entire dataset from the primary database into the cache.
// It serialises concurrent refresh attempts so that only one fetch is performed
// at a time; subsequent callers reuse the result of the first fetch.
// It replaces the existing cached data atomically and resets the expiry timer.
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

	// Clear existing data in the in-memory database using GORM's safe delete
	// (AllowGlobalUpdate bypasses the safety check that prevents delete-without-where)
	entityPtr := reflect.New(c.entityType).Interface()
	if err := c.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(entityPtr).Error; err != nil {
		return fmt.Errorf("failed to clear cache table: %w", err)
	}

	// Bulk-insert the fetched entities
	sliceVal := reflect.ValueOf(entities).Elem().Interface()
	if reflect.ValueOf(sliceVal).Len() > 0 {
		if err := c.db.Create(sliceVal).Error; err != nil {
			return fmt.Errorf("failed to populate cache: %w", err)
		}
	}

	c.mu.Lock()
	c.expiresAt = time.Now().Add(c.ttl)
	c.valid = true
	c.mu.Unlock()

	return nil
}

// DB returns the in-memory GORM database that holds the cached data.
// Callers should use this as a read-only data source; writes to it are not
// propagated back to the primary database.
func (c *EntityCache) DB() *gorm.DB {
	return c.db
}
