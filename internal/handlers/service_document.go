package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync/atomic"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

// ServiceDocumentHandler handles requests for the OData service document.
//
// The service document lists every exposed entity set and singleton with a
// relative url, so its serialized "value" array depends only on the registered
// schema — not on the request. The handler builds that array once and caches the
// bytes, rebuilding lazily after ClearCache. This mirrors the metadata handler
// and assumes entities are registered before the service starts handling
// requests.
type ServiceDocumentHandler struct {
	entities map[string]*metadata.EntityMetadata
	logger   *slog.Logger
	policy   auth.Policy

	// cachedValueJSON holds the serialized service document "value" array,
	// rebuilt lazily on the first request after construction or ClearCache.
	cachedValueJSON atomic.Pointer[[]byte]
}

// NewServiceDocumentHandler creates a new service document handler.
func NewServiceDocumentHandler(entities map[string]*metadata.EntityMetadata, logger *slog.Logger) *ServiceDocumentHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ServiceDocumentHandler{
		entities: entities,
		logger:   logger,
	}
}

// SetLogger sets the logger for the handler.
func (h *ServiceDocumentHandler) SetLogger(logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	h.logger = logger
}

// SetPolicy sets the authorization policy for the handler.
func (h *ServiceDocumentHandler) SetPolicy(policy auth.Policy) {
	h.policy = policy
}

// ClearCache discards the cached service document so the next request rebuilds
// it. Call this if the set of exposed entity sets or singletons changes.
func (h *ServiceDocumentHandler) ClearCache() {
	h.cachedValueJSON.Store(nil)
}

// valueJSON returns the serialized service document "value" array, building and
// caching it on first use. The build is idempotent, so a race between two
// initial requests simply recomputes identical bytes.
func (h *ServiceDocumentHandler) valueJSON() ([]byte, error) {
	if cached := h.cachedValueJSON.Load(); cached != nil {
		return *cached, nil
	}

	// Build separate, sorted lists for entity sets and singletons so the cached
	// document has a stable, deterministic ordering.
	entitySets := make([]string, 0, len(h.entities))
	singletons := make([]string, 0)
	for name, meta := range h.entities {
		// Skip entities that are only accessible via navigation properties.
		if meta.IsAccessibleOnlyViaNavigation {
			continue
		}
		if meta.IsSingleton {
			singletons = append(singletons, name)
		} else {
			entitySets = append(entitySets, name)
		}
	}
	sort.Strings(entitySets)
	sort.Strings(singletons)

	valueJSON, err := response.BuildServiceDocumentValueJSON(entitySets, singletons)
	if err != nil {
		return nil, err
	}

	h.cachedValueJSON.Store(&valueJSON)
	return valueJSON, nil
}

// HandleServiceDocument handles the service document endpoint
func (h *ServiceDocumentHandler) HandleServiceDocument(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if !authorizeRequest(w, r, h.policy, auth.ResourceDescriptor{}, auth.OperationRead, h.logger) {
			return
		}
		h.handleGetServiceDocument(w, r)
	case http.MethodOptions:
		if !authorizeRequest(w, r, h.policy, auth.ResourceDescriptor{}, auth.OperationRead, h.logger) {
			return
		}
		h.handleOptionsServiceDocument(w)
	default:
		if err := response.WriteMethodNotAllowed(w, r, "GET, HEAD, OPTIONS", "Method not allowed",
			fmt.Sprintf("Method %s is not supported for service document", r.Method)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
	}
}

// handleGetServiceDocument handles GET requests for service document
func (h *ServiceDocumentHandler) handleGetServiceDocument(w http.ResponseWriter, r *http.Request) {
	valueJSON, err := h.valueJSON()
	if err != nil {
		h.logger.Error("Error building service document", "error", err)
		if writeErr := response.WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to build service document."); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	if err := response.WriteServiceDocumentValue(w, r, valueJSON); err != nil {
		h.logger.Error("Error writing service document", "error", err)
	}
}

// handleOptionsServiceDocument handles OPTIONS requests for service document
func (h *ServiceDocumentHandler) handleOptionsServiceDocument(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}
