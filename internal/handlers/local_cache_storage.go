package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
)

// LocalCacheStorageOptions controls local in-memory cache behavior.
type LocalCacheStorageOptions struct {
	// TTL controls cache entry expiration. A non-positive value disables expiry.
	TTL time.Duration
	// MaxEntityEntries limits cached entity-by-key entries. Non-positive is unbounded.
	MaxEntityEntries int
	// MaxCollectionEntries limits cached collection query entries. Non-positive is unbounded.
	MaxCollectionEntries int
	// MaxCountEntries limits cached count query entries. Non-positive is unbounded.
	MaxCountEntries int
	// WarmEntitySets selects entity sets warmed at registration/startup.
	WarmEntitySets []string
	// EnabledEntitySets limits caching to listed entity sets.
	// Empty means all entity sets are eligible for caching.
	EnabledEntitySets []string
	// WarmTop limits warm-up collection size to avoid large startup reads.
	WarmTop int
}

// LocalCacheStorage composes a DB-backed storage with a local in-memory read cache.
//
// Safety note: scoped reads (for example auth/tenant scopes) are intentionally not
// cached to avoid cross-request data leakage.
type LocalCacheStorage struct {
	base Storage
	ttl  time.Duration

	maxEntityEntries     int
	maxCollectionEntries int
	maxCountEntries      int

	warmEntitySets    map[string]struct{}
	enabledEntitySets map[string]struct{}
	warmTop           int

	mu sync.RWMutex

	entityByKey         map[string]cacheEntry
	collectionByKey     map[string]cacheEntry
	countByKey          map[string]countCacheEntry
	entityKeysBySet     map[string]map[string]struct{}
	collectionKeysBySet map[string]map[string]struct{}
	countKeysBySet      map[string]map[string]struct{}
	globalVersion       uint64
}

type cacheEntry struct {
	value interface{}
	meta  cacheMetadata
}

type countCacheEntry struct {
	count int64
	meta  cacheMetadata
}

type cacheMetadata struct {
	VersionMarker uint64
	UpdatedAt     time.Time
	Dirty         bool
	PendingOpID   string
	ExpiresAt     time.Time
}

// NewLocalCacheStorage creates a local memory cache storage that delegates cache
// misses and writes to the provided base storage.
func NewLocalCacheStorage(base Storage, opts LocalCacheStorageOptions) Storage {
	if base == nil {
		base = NewDBStorage()
	}

	warmSets := make(map[string]struct{}, len(opts.WarmEntitySets))
	for _, setName := range opts.WarmEntitySets {
		trimmed := strings.TrimSpace(setName)
		if trimmed == "" {
			continue
		}
		warmSets[strings.ToLower(trimmed)] = struct{}{}
	}

	enabledSets := make(map[string]struct{}, len(opts.EnabledEntitySets))
	for _, setName := range opts.EnabledEntitySets {
		trimmed := strings.TrimSpace(setName)
		if trimmed == "" {
			continue
		}
		enabledSets[strings.ToLower(trimmed)] = struct{}{}
	}

	return &LocalCacheStorage{
		base:                 base,
		ttl:                  opts.TTL,
		maxEntityEntries:     opts.MaxEntityEntries,
		maxCollectionEntries: opts.MaxCollectionEntries,
		maxCountEntries:      opts.MaxCountEntries,
		warmEntitySets:       warmSets,
		enabledEntitySets:    enabledSets,
		warmTop:              opts.WarmTop,
		entityByKey:          make(map[string]cacheEntry),
		collectionByKey:      make(map[string]cacheEntry),
		countByKey:           make(map[string]countCacheEntry),
		entityKeysBySet:      make(map[string]map[string]struct{}),
		collectionKeysBySet:  make(map[string]map[string]struct{}),
		countKeysBySet:       make(map[string]map[string]struct{}),
	}
}

func (s *LocalCacheStorage) FetchEntityByKey(ctx context.Context, h *EntityHandler, entityKey string, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	if h == nil || h.metadata == nil || len(scopes) > 0 || !s.shouldCacheEntitySet(h.metadata.EntitySetName) {
		return s.base.FetchEntityByKey(ctx, h, entityKey, queryOptions, scopes)
	}

	cacheKey := s.buildEntityCacheKey(h.metadata.EntitySetName, canonicalEntityKeyFromRaw(entityKey, h.metadata.KeyProperties))
	if cached, ok := s.getEntity(cacheKey); ok {
		return cached, nil
	}

	result, err := s.base.FetchEntityByKey(ctx, h, entityKey, queryOptions, scopes)
	if err != nil {
		return nil, err
	}

	s.setEntity(h.metadata.EntitySetName, cacheKey, result)
	return cloneValue(result), nil
}

func (s *LocalCacheStorage) FetchCollection(ctx context.Context, h *EntityHandler, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	if h == nil || h.metadata == nil || len(scopes) > 0 || !s.shouldCacheEntitySet(h.metadata.EntitySetName) {
		return s.base.FetchCollection(ctx, h, queryOptions, scopes)
	}

	cacheKey := s.buildCollectionCacheKey(h.metadata.EntitySetName, queryOptions)
	if cached, ok := s.getCollection(cacheKey); ok {
		return cached, nil
	}

	result, err := s.base.FetchCollection(ctx, h, queryOptions, scopes)
	if err != nil {
		return nil, err
	}

	s.setCollection(h.metadata.EntitySetName, cacheKey, result)
	s.populateEntityCacheFromCollection(h.metadata, result)
	return cloneValue(result), nil
}

func (s *LocalCacheStorage) CountEntities(ctx context.Context, h *EntityHandler, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (int64, error) {
	if h == nil || h.metadata == nil || len(scopes) > 0 || !s.shouldCacheEntitySet(h.metadata.EntitySetName) {
		return s.base.CountEntities(ctx, h, queryOptions, scopes)
	}

	cacheKey := s.buildCountCacheKey(h.metadata.EntitySetName, queryOptions)
	if count, ok := s.getCount(cacheKey); ok {
		return count, nil
	}

	count, err := s.base.CountEntities(ctx, h, queryOptions, scopes)
	if err != nil {
		return 0, err
	}

	s.setCount(h.metadata.EntitySetName, cacheKey, count)
	return count, nil
}

func (s *LocalCacheStorage) Create(tx *gorm.DB, h *EntityHandler, entity interface{}) error {
	return s.base.Create(tx, h, entity)
}

func (s *LocalCacheStorage) UpdatePartial(tx *gorm.DB, h *EntityHandler, entity interface{}, updateData map[string]interface{}) error {
	return s.base.UpdatePartial(tx, h, entity, updateData)
}

func (s *LocalCacheStorage) UpdateFull(tx *gorm.DB, h *EntityHandler, entity interface{}, replacement interface{}) error {
	return s.base.UpdateFull(tx, h, entity, replacement)
}

func (s *LocalCacheStorage) Delete(tx *gorm.DB, h *EntityHandler, entity interface{}) error {
	return s.base.Delete(tx, h, entity)
}

func (s *LocalCacheStorage) Refresh(tx *gorm.DB, h *EntityHandler, entity interface{}) error {
	return s.base.Refresh(tx, h, entity)
}

func (s *LocalCacheStorage) OnEntityChanged(h *EntityHandler, entity interface{}, changeType trackchanges.ChangeType) {
	if h == nil || h.metadata == nil {
		return
	}
	if !s.shouldCacheEntitySet(h.metadata.EntitySetName) {
		return
	}

	setName := h.metadata.EntitySetName
	s.invalidateCollectionSet(setName)
	s.invalidateEntitySet(setName)

	entityKey := canonicalEntityKeyFromEntity(entity, h.metadata)
	if entityKey == "" {
		return
	}

	cacheKey := s.buildEntityCacheKey(setName, entityKey)
	switch changeType {
	case trackchanges.ChangeTypeDeleted:
		s.deleteEntity(setName, cacheKey)
	default:
		s.setEntity(setName, cacheKey, entity)
	}
}

// ReplayEntityChange applies an externally observed committed change to local cache state.
func (s *LocalCacheStorage) ReplayEntityChange(h *EntityHandler, keyValues map[string]interface{}, data map[string]interface{}, changeType trackchanges.ChangeType) {
	if h == nil || h.metadata == nil {
		return
	}
	if !s.shouldCacheEntitySet(h.metadata.EntitySetName) {
		return
	}

	setName := h.metadata.EntitySetName
	s.invalidateCollectionSet(setName)

	entityKey := canonicalEntityKeyFromMap(keyValues, h.metadata.KeyProperties)
	if entityKey == "" {
		return
	}

	cacheKey := s.buildEntityCacheKey(setName, entityKey)
	if changeType == trackchanges.ChangeTypeDeleted {
		s.deleteEntity(setName, cacheKey)
		return
	}

	entity, err := materializeEntityFromData(h.metadata.EntityType, data)
	if err != nil {
		// If payload materialization fails, fail-safe by dropping stale entity cache.
		s.deleteEntity(setName, cacheKey)
		return
	}

	s.setEntity(setName, cacheKey, entity)
}

func (s *LocalCacheStorage) WarmEntitySet(ctx context.Context, h *EntityHandler) error {
	if h == nil || h.metadata == nil {
		return nil
	}
	if !s.shouldCacheEntitySet(h.metadata.EntitySetName) {
		return nil
	}
	if len(s.warmEntitySets) == 0 {
		return nil
	}
	if _, ok := s.warmEntitySets[strings.ToLower(h.metadata.EntitySetName)]; !ok {
		return nil
	}

	return s.reconcileEntitySet(ctx, h, s.warmTop)
}

// ReconcileEntitySet force-refreshes cached collection/entity snapshots for an entity set.
func (s *LocalCacheStorage) ReconcileEntitySet(ctx context.Context, h *EntityHandler) error {
	if h == nil || h.metadata == nil {
		return nil
	}
	if !s.shouldCacheEntitySet(h.metadata.EntitySetName) {
		return nil
	}
	return s.reconcileEntitySet(ctx, h, 0)
}

func (s *LocalCacheStorage) reconcileEntitySet(ctx context.Context, h *EntityHandler, top int) error {
	if h == nil || h.metadata == nil {
		return nil
	}

	queryOptions := &query.QueryOptions{}
	if top > 0 {
		warmTop := top
		queryOptions.Top = &warmTop
	}

	results, err := s.base.FetchCollection(ctx, h, queryOptions, nil)
	if err != nil {
		return err
	}

	setName := h.metadata.EntitySetName
	s.invalidateCollectionSet(setName)
	s.invalidateEntitySet(setName)

	collectionKey := s.buildCollectionCacheKey(setName, queryOptions)
	s.setCollection(setName, collectionKey, results)
	s.populateEntityCacheFromCollection(h.metadata, results)
	return nil
}

func (s *LocalCacheStorage) shouldCacheEntitySet(entitySet string) bool {
	if len(s.enabledEntitySets) == 0 {
		return true
	}
	_, ok := s.enabledEntitySets[strings.ToLower(strings.TrimSpace(entitySet))]
	return ok
}

func (s *LocalCacheStorage) buildEntityCacheKey(entitySet, canonicalKey string) string {
	return fmt.Sprintf("entity|%s|%s", entitySet, canonicalKey)
}

func (s *LocalCacheStorage) buildCollectionCacheKey(entitySet string, queryOptions *query.QueryOptions) string {
	return fmt.Sprintf("collection|%s|%s", entitySet, normalizeQueryOptions(queryOptions))
}

func (s *LocalCacheStorage) buildCountCacheKey(entitySet string, queryOptions *query.QueryOptions) string {
	return fmt.Sprintf("count|%s|%s", entitySet, normalizeQueryOptions(queryOptions))
}

func (s *LocalCacheStorage) setEntity(entitySet, key string, value interface{}) {
	entry := cacheEntry{value: cloneValue(value), meta: s.newMetadata()}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entityByKey[key] = entry
	indexKeyBySet(s.entityKeysBySet, entitySet, key)
	s.enforceEntityCapacityLocked()
}

func (s *LocalCacheStorage) setEntityIfAbsent(entitySet, key string, value interface{}) {
	entry := cacheEntry{value: cloneValue(value), meta: s.newMetadata()}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.entityByKey[key]; exists {
		return
	}
	s.entityByKey[key] = entry
	indexKeyBySet(s.entityKeysBySet, entitySet, key)
	s.enforceEntityCapacityLocked()
}

func (s *LocalCacheStorage) getEntity(key string) (interface{}, bool) {
	s.mu.RLock()
	entry, ok := s.entityByKey[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if s.isExpired(entry.meta.ExpiresAt) {
		s.mu.Lock()
		delete(s.entityByKey, key)
		removeIndexedKeyFromAllSets(s.entityKeysBySet, key)
		s.mu.Unlock()
		return nil, false
	}
	return cloneValue(entry.value), true
}

func (s *LocalCacheStorage) deleteEntity(entitySet, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entityByKey, key)
	deleteIndexedKey(s.entityKeysBySet, entitySet, key)
}

func (s *LocalCacheStorage) setCollection(entitySet, key string, value interface{}) {
	entry := cacheEntry{value: cloneValue(value), meta: s.newMetadata()}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.collectionByKey[key] = entry
	indexKeyBySet(s.collectionKeysBySet, entitySet, key)
	s.enforceCollectionCapacityLocked()
}

func (s *LocalCacheStorage) getCollection(key string) (interface{}, bool) {
	s.mu.RLock()
	entry, ok := s.collectionByKey[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if s.isExpired(entry.meta.ExpiresAt) {
		s.mu.Lock()
		delete(s.collectionByKey, key)
		removeIndexedKeyFromAllSets(s.collectionKeysBySet, key)
		s.mu.Unlock()
		return nil, false
	}
	return cloneValue(entry.value), true
}

func (s *LocalCacheStorage) setCount(entitySet, key string, value int64) {
	entry := countCacheEntry{count: value, meta: s.newMetadata()}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.countByKey[key] = entry
	indexKeyBySet(s.countKeysBySet, entitySet, key)
	s.enforceCountCapacityLocked()
}

func (s *LocalCacheStorage) getCount(key string) (int64, bool) {
	s.mu.RLock()
	entry, ok := s.countByKey[key]
	s.mu.RUnlock()
	if !ok {
		return 0, false
	}
	if s.isExpired(entry.meta.ExpiresAt) {
		s.mu.Lock()
		delete(s.countByKey, key)
		removeIndexedKeyFromAllSets(s.countKeysBySet, key)
		s.mu.Unlock()
		return 0, false
	}
	return entry.count, true
}

func (s *LocalCacheStorage) invalidateCollectionSet(entitySet string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.collectionKeysBySet[entitySet] {
		delete(s.collectionByKey, key)
	}
	delete(s.collectionKeysBySet, entitySet)

	for key := range s.countKeysBySet[entitySet] {
		delete(s.countByKey, key)
	}
	delete(s.countKeysBySet, entitySet)
}

func (s *LocalCacheStorage) invalidateEntitySet(entitySet string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.entityKeysBySet[entitySet] {
		delete(s.entityByKey, key)
	}
	delete(s.entityKeysBySet, entitySet)
}

func (s *LocalCacheStorage) populateEntityCacheFromCollection(meta *metadata.EntityMetadata, collection interface{}) {
	if meta == nil {
		return
	}
	value := reflect.ValueOf(collection)
	if value.Kind() != reflect.Slice {
		return
	}

	for i := 0; i < value.Len(); i++ {
		item := value.Index(i).Interface()
		entityKey := canonicalEntityKeyFromEntity(item, meta)
		if entityKey == "" {
			continue
		}
		s.setEntityIfAbsent(meta.EntitySetName, s.buildEntityCacheKey(meta.EntitySetName, entityKey), item)
	}
}

func (s *LocalCacheStorage) newMetadata() cacheMetadata {
	meta := cacheMetadata{
		VersionMarker: atomic.AddUint64(&s.globalVersion, 1),
		UpdatedAt:     time.Now().UTC(),
		Dirty:         false,
		PendingOpID:   "",
	}
	if s.ttl > 0 {
		meta.ExpiresAt = meta.UpdatedAt.Add(s.ttl)
	}
	return meta
}

func (s *LocalCacheStorage) isExpired(expiresAt time.Time) bool {
	if expiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().After(expiresAt)
}

func (s *LocalCacheStorage) enforceEntityCapacityLocked() {
	if s.maxEntityEntries <= 0 {
		return
	}
	for len(s.entityByKey) > s.maxEntityEntries {
		oldestKey := oldestCacheEntryKey(s.entityByKey)
		if oldestKey == "" {
			return
		}
		delete(s.entityByKey, oldestKey)
		removeIndexedKeyFromAllSets(s.entityKeysBySet, oldestKey)
	}
}

func (s *LocalCacheStorage) enforceCollectionCapacityLocked() {
	if s.maxCollectionEntries <= 0 {
		return
	}
	for len(s.collectionByKey) > s.maxCollectionEntries {
		oldestKey := oldestCacheEntryKey(s.collectionByKey)
		if oldestKey == "" {
			return
		}
		delete(s.collectionByKey, oldestKey)
		removeIndexedKeyFromAllSets(s.collectionKeysBySet, oldestKey)
	}
}

func (s *LocalCacheStorage) enforceCountCapacityLocked() {
	if s.maxCountEntries <= 0 {
		return
	}
	for len(s.countByKey) > s.maxCountEntries {
		oldestKey := oldestCountCacheEntryKey(s.countByKey)
		if oldestKey == "" {
			return
		}
		delete(s.countByKey, oldestKey)
		removeIndexedKeyFromAllSets(s.countKeysBySet, oldestKey)
	}
}

func oldestCacheEntryKey(entries map[string]cacheEntry) string {
	var (
		oldestKey string
		oldestAt  time.Time
	)
	for key, entry := range entries {
		if oldestKey == "" || entry.meta.UpdatedAt.Before(oldestAt) {
			oldestKey = key
			oldestAt = entry.meta.UpdatedAt
		}
	}
	return oldestKey
}

func oldestCountCacheEntryKey(entries map[string]countCacheEntry) string {
	var (
		oldestKey string
		oldestAt  time.Time
	)
	for key, entry := range entries {
		if oldestKey == "" || entry.meta.UpdatedAt.Before(oldestAt) {
			oldestKey = key
			oldestAt = entry.meta.UpdatedAt
		}
	}
	return oldestKey
}

func normalizeQueryOptions(queryOptions *query.QueryOptions) string {
	if queryOptions == nil {
		return "{}"
	}
	encoded, err := json.Marshal(queryOptions)
	if err != nil {
		return fmt.Sprintf("fallback:%v", queryOptions)
	}
	return string(encoded)
}

func canonicalEntityKeyFromRaw(raw string, keyProperties []metadata.PropertyMetadata) string {
	parsed := parseEntityKeyValues(strings.TrimSpace(raw), keyProperties)
	return canonicalEntityKeyFromMap(parsed, keyProperties)
}

func canonicalEntityKeyFromEntity(entity interface{}, meta *metadata.EntityMetadata) string {
	if entity == nil || meta == nil {
		return ""
	}
	values := extractEntityKeyValues(entity, meta)
	return canonicalEntityKeyFromMap(values, meta.KeyProperties)
}

func extractEntityKeyValues(entity interface{}, meta *metadata.EntityMetadata) map[string]interface{} {
	keyValues := make(map[string]interface{}, len(meta.KeyProperties))
	value := reflect.ValueOf(entity)
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return keyValues
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return keyValues
	}

	for _, keyProp := range meta.KeyProperties {
		fieldNames := []string{keyProp.FieldName, keyProp.Name}
		for _, fieldName := range fieldNames {
			if fieldName == "" {
				continue
			}
			field := value.FieldByName(fieldName)
			if !field.IsValid() {
				continue
			}
			keyValues[keyProp.JsonName] = field.Interface()
			break
		}
	}

	return keyValues
}

func canonicalEntityKeyFromMap(values map[string]interface{}, keyProperties []metadata.PropertyMetadata) string {
	if len(values) == 0 {
		return ""
	}

	parts := make([]string, 0, len(values))
	for _, keyProp := range keyProperties {
		value, ok := values[keyProp.JsonName]
		if !ok {
			value, ok = values[keyProp.Name]
		}
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", keyProp.JsonName, stableValueString(value)))
	}

	if len(parts) > 0 {
		return strings.Join(parts, ",")
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, stableValueString(values[key])))
	}
	return strings.Join(parts, ",")
}

func stableValueString(value interface{}) string {
	encoded, err := json.Marshal(value)
	if err == nil {
		return string(encoded)
	}
	return fmt.Sprintf("%v", value)
}

func cloneValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return value
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return value
	}

	typeValue := v.Type()
	if typeValue.Kind() == reflect.Ptr {
		clone := reflect.New(typeValue.Elem())
		if err := json.Unmarshal(encoded, clone.Interface()); err != nil {
			return value
		}
		return clone.Interface()
	}

	clone := reflect.New(typeValue)
	if err := json.Unmarshal(encoded, clone.Interface()); err != nil {
		return value
	}
	return clone.Elem().Interface()
}

func indexKeyBySet(index map[string]map[string]struct{}, setName, key string) {
	if setName == "" || key == "" {
		return
	}
	if _, ok := index[setName]; !ok {
		index[setName] = make(map[string]struct{})
	}
	index[setName][key] = struct{}{}
}

func deleteIndexedKey(index map[string]map[string]struct{}, setName, key string) {
	set, ok := index[setName]
	if !ok {
		return
	}
	delete(set, key)
	if len(set) == 0 {
		delete(index, setName)
	}
}

func removeIndexedKeyFromAllSets(index map[string]map[string]struct{}, key string) {
	for setName, set := range index {
		if _, ok := set[key]; !ok {
			continue
		}
		delete(set, key)
		if len(set) == 0 {
			delete(index, setName)
		}
	}
}
