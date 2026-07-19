package cache

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

// EntityCache provides an in-memory snapshot cache for a single entity type.
// It loads the full dataset from the primary database once per TTL window and
// serves read queries directly from Go data structures, avoiding both repeated
// round-trips to the primary database and the per-connection locking of an
// embedded SQL engine.
//
// Concurrency model:
//   - The active dataset is held in an immutable *Snapshot behind an atomic
//     pointer. Readers load the pointer once and operate on the snapshot with no
//     locking whatsoever, so an unbounded number of reads proceed in parallel.
//   - A snapshot is never mutated after it is published. Refresh builds a brand
//     new snapshot and swaps it in atomically; the old snapshot is reclaimed by
//     the garbage collector once the last in-flight reader releases it.
//
// Cache invalidation:
//   - Automatic: a snapshot expires after the configured TTL.
//   - Manual: call Invalidate() after any write operation to force a refresh on
//     the next read.
type EntityCache struct {
	refreshMu   sync.Mutex // serializes refreshes to prevent a thundering herd
	snap        atomic.Pointer[Snapshot]
	ttl         time.Duration
	entityType  reflect.Type
	keyFn       KeyFunc
	normalizeFn NormalizeFunc
}

// KeyFunc derives the canonical string key of an entity. The argument is a
// single element of the loaded []T slice (a struct value, not a pointer). The
// returned string must match the one produced for the same key by the caller
// that performs key lookups, so that Snapshot.Lookup can find the entity.
type KeyFunc func(entity reflect.Value) string

// NormalizeFunc precomputes comparison-ready values for one entity, e.g. one
// slot per filterable/sortable property, indexed however the caller's own
// property lookup agrees to. It runs once per entity when a snapshot is
// built (Refresh), not on every comparison of every request, so callers that
// otherwise re-derive and re-box the same values via reflection on each
// filter/sort comparison can instead read them straight from the snapshot.
type NormalizeFunc func(entity reflect.Value) []interface{}

// Snapshot is an immutable view of the cached dataset. It is safe for concurrent
// reads and is never modified after New/Refresh publishes it.
type Snapshot struct {
	entities   reflect.Value   // []T, treated as read-only after construction
	byKey      map[string]int  // canonical key -> index into entities
	normalized [][]interface{} // parallel to entities; nil if no NormalizeFunc was configured
	expiresAt  time.Time
}

// New creates a new EntityCache for the given entity type and TTL.
// entityType should be the non-pointer struct type of the entity. keyFn derives
// the canonical key used for O(1) key lookups and must not be nil. normalizeFn
// is optional (nil disables precomputed normalization); when set, its output
// for every entity is precomputed once per Refresh and exposed via
// Snapshot.Normalized.
func New(entityType reflect.Type, ttl time.Duration, keyFn KeyFunc, normalizeFn NormalizeFunc) (*EntityCache, error) {
	if entityType == nil {
		return nil, fmt.Errorf("entityType must not be nil")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("ttl must be positive")
	}
	if keyFn == nil {
		return nil, fmt.Errorf("keyFn must not be nil")
	}

	return &EntityCache{
		ttl:         ttl,
		entityType:  entityType,
		keyFn:       keyFn,
		normalizeFn: normalizeFn,
	}, nil
}

// Current returns the active snapshot if it has been populated and has not
// expired. The returned snapshot is immutable and safe to read concurrently.
func (c *EntityCache) Current() (*Snapshot, bool) {
	s := c.snap.Load()
	if s == nil || !time.Now().Before(s.expiresAt) {
		return nil, false
	}
	return s, true
}

// IsValid reports whether the cache contains fresh data that has not expired.
func (c *EntityCache) IsValid() bool {
	_, ok := c.Current()
	return ok
}

// Invalidate drops the current snapshot so that the next read triggers a refresh
// from the primary database.
func (c *EntityCache) Invalidate() {
	c.snap.Store(nil)
}

// Refresh reloads the entire dataset from the primary database into a fresh
// snapshot and swaps it in atomically. It serialises concurrent refresh attempts
// so that only one fetch runs at a time; callers that arrive while a refresh is
// in flight reuse its result rather than issuing a redundant fetch.
func (c *EntityCache) Refresh(sourceDB *gorm.DB) error {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	// Another goroutine may have refreshed the cache while we waited on the
	// refresh lock. Reuse that result rather than doing an unnecessary fetch.
	if c.IsValid() {
		return nil
	}

	sliceType := reflect.SliceOf(c.entityType)
	entitiesPtr := reflect.New(sliceType)
	if err := sourceDB.Find(entitiesPtr.Interface()).Error; err != nil {
		return fmt.Errorf("failed to load entities from primary database: %w", err)
	}

	entities := entitiesPtr.Elem()
	byKey := make(map[string]int, entities.Len())
	var normalized [][]interface{}
	if c.normalizeFn != nil {
		normalized = make([][]interface{}, entities.Len())
	}
	for i := 0; i < entities.Len(); i++ {
		entity := entities.Index(i)
		byKey[c.keyFn(entity)] = i
		if c.normalizeFn != nil {
			normalized[i] = c.normalizeFn(entity)
		}
	}

	c.snap.Store(&Snapshot{
		entities:   entities,
		byKey:      byKey,
		normalized: normalized,
		expiresAt:  time.Now().Add(c.ttl),
	})

	return nil
}

// Len returns the number of entities in the snapshot.
func (s *Snapshot) Len() int {
	return s.entities.Len()
}

// At returns the entity at index i. The returned reflect.Value aliases the
// snapshot's backing array and must not be mutated; callers that need a mutable
// copy should copy the struct value out first.
func (s *Snapshot) At(i int) reflect.Value {
	return s.entities.Index(i)
}

// Lookup returns the entity with the given canonical key. The returned value
// aliases the snapshot's backing array and must not be mutated.
func (s *Snapshot) Lookup(key string) (reflect.Value, bool) {
	i, ok := s.byKey[key]
	if !ok {
		return reflect.Value{}, false
	}
	return s.entities.Index(i), true
}

// Normalized returns the precomputed comparison-ready values for the entity
// at index i, as produced by the NormalizeFunc passed to New. It returns nil
// if the cache was created without one.
func (s *Snapshot) Normalized(i int) []interface{} {
	if s.normalized == nil {
		return nil
	}
	return s.normalized[i]
}
