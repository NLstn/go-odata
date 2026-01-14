package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

// MetadataHandler handles metadata document requests
//
// # Locking Strategy
//
// This handler uses a simplified locking approach that eliminates lock ordering concerns:
//
// 1. sync.Map for caches (cachedXML, cachedJSON): Lock-free reads, internal synchronization for writes
// 2. namespaceMu: Protects only the namespace field (not the caches)
// 3. No lock ordering requirements - each lock is independent
//
// ## Why This Design Prevents Deadlocks
//
// The cache and namespace are deliberately decoupled:
// - Cache operations never acquire namespaceMu
// - Namespace operations (SetNamespace) clear the cache but don't hold locks while doing so
// - sync.Map provides lock-free reads, eliminating the most common contention point
//
// ## Thread Safety Guarantees
//
// - Concurrent metadata requests are safe (lock-free reads from sync.Map)
// - SetNamespace is safe to call concurrently (protected by namespaceMu write lock)
// - Cache eviction is safe (uses sync.Map's atomic operations and atomic.Int64 for size)
//
// ## Performance Characteristics
//
// - Cache hits: Lock-free (sync.Map.Load)
// - Cache misses: Single sync.Map.LoadOrStore per build
// - Namespace reads: Single RLock (fast path in namespaceOrDefault)
// - Namespace writes: Single Lock + cache clear (rare operation)
type MetadataHandler struct {
	entities map[string]*metadata.EntityMetadata
	// Lock-free cached metadata documents by version (key: version string)
	cachedXML     sync.Map     // map[string]string
	cachedJSON    sync.Map     // map[string][]byte
	cacheSizeXML  atomic.Int64 // Tracks XML cache entries for eviction
	cacheSizeJSON atomic.Int64 // Tracks JSON cache entries for eviction
	namespace     string
	namespaceMu   sync.RWMutex // Protects namespace field ONLY (not the caches)
	logger        *slog.Logger
	policy        auth.Policy
}

const defaultNamespace = "ODataService"

// NewMetadataHandler creates a new metadata handler
func NewMetadataHandler(entities map[string]*metadata.EntityMetadata) *MetadataHandler {
	h := &MetadataHandler{
		entities:  entities,
		namespace: defaultNamespace,
		logger:    slog.Default(),
	}
	// sync.Map fields are initialized to their zero value (empty map)
	h.cacheSizeXML.Store(0)
	h.cacheSizeJSON.Store(0)
	return h
}

// SetLogger sets the logger for the metadata handler.
func (h *MetadataHandler) SetLogger(logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	h.logger = logger
}

// SetPolicy sets the authorization policy for the handler.
func (h *MetadataHandler) SetPolicy(policy auth.Policy) {
	h.policy = policy
}

// SetNamespace updates the namespace used for metadata generation and clears cached documents.
//
// Thread-safety: This method is safe to call concurrently. The namespaceMu write lock
// ensures that only one SetNamespace operation executes at a time. Cache clearing uses
// sync.Map.Delete which is internally synchronized.
//
// Design note: This method does NOT create lock ordering issues because:
// - It only acquires namespaceMu (never acquires cache locks - sync.Map is lock-free)
// - Cache operations (in metadata handlers) never acquire namespaceMu
// - Therefore, no circular wait condition can occur
func (h *MetadataHandler) SetNamespace(namespace string) {
	trimmed := strings.TrimSpace(namespace)
	if trimmed == "" {
		trimmed = defaultNamespace
	}

	// Acquire write lock for namespace update
	h.namespaceMu.Lock()
	defer h.namespaceMu.Unlock()

	// Check if namespace actually changed
	if trimmed == h.namespace {
		return
	}

	// Update namespace
	h.namespace = trimmed

	// Clear all cached versions since namespace changed
	// Delete all entries from sync.Map instead of replacing it
	h.cachedXML.Range(func(key, value interface{}) bool {
		h.cachedXML.Delete(key)
		return true
	})
	h.cachedJSON.Range(func(key, value interface{}) bool {
		h.cachedJSON.Delete(key)
		return true
	})
	h.cacheSizeXML.Store(0)
	h.cacheSizeJSON.Store(0)
}

// namespaceOrDefault returns the current namespace or the default if empty.
//
// Thread-safety: Uses RLock for fast concurrent reads. This is safe because:
// - Multiple readers can hold RLock simultaneously (high throughput)
// - SetNamespace's write lock blocks until all readers release RLock
// - No cache operations occur while holding this lock (prevents lock ordering issues)
func (h *MetadataHandler) namespaceOrDefault() string {
	h.namespaceMu.RLock()
	defer h.namespaceMu.RUnlock()
	if strings.TrimSpace(h.namespace) == "" {
		return defaultNamespace
	}
	return h.namespace
}

// maxCacheEntries defines the maximum number of cached metadata versions to keep
// This prevents unbounded memory growth when many different OData-MaxVersion values are requested
const maxCacheEntries = 10

// evictOldCacheEntriesXML removes the oldest cached XML entries if the cache exceeds maxCacheEntries.
// Uses sync.Map.Range to iterate and delete entries.
func (h *MetadataHandler) evictOldCacheEntriesXML() {
	if h.cacheSizeXML.Load() <= maxCacheEntries {
		return
	}

	// Keep priority versions (most common)
	priorityVersions := map[string]bool{"4.01": true, "4.0": true}
	keysToKeep := maxCacheEntries / 2

	// First pass: collect all entries and count priority versions
	var priorityKeys []string
	var nonPriorityKeys []string
	h.cachedXML.Range(func(key, value interface{}) bool {
		keyStr, ok := key.(string)
		if !ok {
			return true // Skip invalid entries
		}
		if priorityVersions[keyStr] {
			priorityKeys = append(priorityKeys, keyStr)
		} else {
			nonPriorityKeys = append(nonPriorityKeys, keyStr)
		}
		return true
	})

	// Calculate how many non-priority entries to keep
	// Total to keep = keysToKeep, minus however many priority versions we actually have
	nonPriorityToKeep := keysToKeep - len(priorityKeys)
	if nonPriorityToKeep < 0 {
		nonPriorityToKeep = 0
	}

	// Evict excess non-priority entries
	evicted := 0
	if len(nonPriorityKeys) > nonPriorityToKeep {
		for i := nonPriorityToKeep; i < len(nonPriorityKeys); i++ {
			h.cachedXML.Delete(nonPriorityKeys[i])
			evicted++
		}
	}

	// Update cache size
	newSize := h.cacheSizeXML.Add(-int64(evicted))
	h.logger.Info("Evicted old metadata XML cache entries", "newSize", newSize, "evicted", evicted)
}

// evictOldCacheEntriesJSON removes the oldest cached JSON entries if the cache exceeds maxCacheEntries.
// Uses sync.Map.Range to iterate and delete entries.
func (h *MetadataHandler) evictOldCacheEntriesJSON() {
	if h.cacheSizeJSON.Load() <= maxCacheEntries {
		return
	}

	// Keep priority versions (most common)
	priorityVersions := map[string]bool{"4.01": true, "4.0": true}
	keysToKeep := maxCacheEntries / 2

	// First pass: collect all entries and count priority versions
	var priorityKeys []string
	var nonPriorityKeys []string
	h.cachedJSON.Range(func(key, value interface{}) bool {
		keyStr, ok := key.(string)
		if !ok {
			return true // Skip invalid entries
		}
		if priorityVersions[keyStr] {
			priorityKeys = append(priorityKeys, keyStr)
		} else {
			nonPriorityKeys = append(nonPriorityKeys, keyStr)
		}
		return true
	})

	// Calculate how many non-priority entries to keep
	// Total to keep = keysToKeep, minus however many priority versions we actually have
	nonPriorityToKeep := keysToKeep - len(priorityKeys)
	if nonPriorityToKeep < 0 {
		nonPriorityToKeep = 0
	}

	// Evict excess non-priority entries
	evicted := 0
	if len(nonPriorityKeys) > nonPriorityToKeep {
		for i := nonPriorityToKeep; i < len(nonPriorityKeys); i++ {
			h.cachedJSON.Delete(nonPriorityKeys[i])
			evicted++
		}
	}

	// Update cache size
	newSize := h.cacheSizeJSON.Add(-int64(evicted))
	h.logger.Info("Evicted old metadata JSON cache entries", "newSize", newSize, "evicted", evicted)
}

// HandleMetadata handles the metadata document endpoint
func (h *MetadataHandler) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if !authorizeRequest(w, r, h.policy, auth.ResourceDescriptor{}, auth.OperationMetadata, h.logger) {
			return
		}
		h.handleGetMetadata(w, r)
	case http.MethodOptions:
		if !authorizeRequest(w, r, h.policy, auth.ResourceDescriptor{}, auth.OperationMetadata, h.logger) {
			return
		}
		h.handleOptionsMetadata(w)
	default:
		if err := response.WriteError(w, r, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for metadata document", r.Method)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
	}
}

// handleGetMetadata handles GET requests for metadata document
func (h *MetadataHandler) handleGetMetadata(w http.ResponseWriter, r *http.Request) {
	useJSON := shouldReturnJSON(r)

	if useJSON {
		h.handleMetadataJSON(w, r)
	} else {
		h.handleMetadataXML(w, r)
	}
}

// handleOptionsMetadata handles OPTIONS requests for metadata document
func (h *MetadataHandler) handleOptionsMetadata(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

func (h *MetadataHandler) newMetadataModel() metadataModel {
	model := metadataModel{
		namespace: h.namespaceOrDefault(),
		entities:  h.entities,
	}
	model.buildEntityTypeToSetNameMap()
	return model
}

type metadataModel struct {
	namespace              string
	entities               map[string]*metadata.EntityMetadata
	entityTypeToSetNameMap map[string]string // Cache for EntityName -> EntitySetName lookups
}

type enumTypeInfo struct {
	Members        []metadata.EnumMember
	IsFlags        bool
	UnderlyingType string
}

func (m metadataModel) qualifiedTypeName(typeName string) string {
	return fmt.Sprintf("%s.%s", m.namespace, typeName)
}

func (m metadataModel) collectEnumDefinitions() map[string]*enumTypeInfo {
	enumDefinitions := make(map[string]*enumTypeInfo)

	for _, entityMeta := range m.entities {
		for _, prop := range entityMeta.Properties {
			if !prop.IsEnum || prop.EnumTypeName == "" {
				continue
			}

			info, exists := enumDefinitions[prop.EnumTypeName]
			if !exists {
				info = &enumTypeInfo{}
				enumDefinitions[prop.EnumTypeName] = info
			}

			if len(info.Members) == 0 && len(prop.EnumMembers) > 0 {
				info.Members = append([]metadata.EnumMember(nil), prop.EnumMembers...)
			}
			if info.UnderlyingType == "" && prop.EnumUnderlyingType != "" {
				info.UnderlyingType = prop.EnumUnderlyingType
			}
			if prop.IsFlags {
				info.IsFlags = true
			}
		}
	}

	return enumDefinitions
}

// buildEntityTypeToSetNameMap creates a reverse lookup map from EntityName to EntitySetName
// for efficient navigation property binding resolution.
func (m *metadataModel) buildEntityTypeToSetNameMap() {
	m.entityTypeToSetNameMap = make(map[string]string, len(m.entities))
	for entitySetName, entityMeta := range m.entities {
		m.entityTypeToSetNameMap[entityMeta.EntityName] = entitySetName
	}
}

// getEntitySetNameForType looks up the entity set name for a given entity type name.
// This respects custom EntitySetName() methods by using the cached lookup map.
// If the entity type is not found, it falls back to pluralization.
func (m metadataModel) getEntitySetNameForType(entityTypeName string) string {
	// Use cached lookup map for O(1) performance
	if entitySetName, exists := m.entityTypeToSetNameMap[entityTypeName]; exists {
		return entitySetName
	}

	// Fall back to pluralization if entity not found
	return pluralize(entityTypeName)
}

// getEdmType converts a Go type to an EDM (Entity Data Model) type
func getEdmType(goType reflect.Type) string {
	// Handle pointer types
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	// Check for specific types by name
	typeName := goType.String()
	switch typeName {
	case "time.Time":
		return "Edm.DateTimeOffset"
	case "uuid.UUID", "github.com/google/uuid.UUID":
		return "Edm.Guid"
	}

	switch goType.Kind() {
	case reflect.String:
		return "Edm.String"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "Edm.Int32"
	case reflect.Int64:
		return "Edm.Int64"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "Edm.Int32"
	case reflect.Uint64:
		return "Edm.Int64"
	case reflect.Float32:
		return "Edm.Single"
	case reflect.Float64:
		return "Edm.Double"
	case reflect.Bool:
		return "Edm.Boolean"
	default:
		if goType.Kind() == reflect.Slice && goType.Elem().Kind() == reflect.Uint8 {
			return "Edm.Binary"
		}
		return "Edm.String"
	}
}

// pluralize creates a simple pluralized form of the entity name
func pluralize(word string) string {
	if word == "" {
		return word
	}

	switch {
	case strings.HasSuffix(word, "y"):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") || strings.HasSuffix(word, "z") ||
		strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh"):
		return word + "es"
	default:
		return word + "s"
	}
}
