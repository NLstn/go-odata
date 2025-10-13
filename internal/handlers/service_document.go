package handlers

import (
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

// ServiceDocumentHandler handles requests for the OData service document.
type ServiceDocumentHandler struct {
	entities map[string]*metadata.EntityMetadata
}

// NewServiceDocumentHandler creates a new service document handler.
func NewServiceDocumentHandler(entities map[string]*metadata.EntityMetadata) *ServiceDocumentHandler {
	return &ServiceDocumentHandler{
		entities: entities,
	}
}

// HandleServiceDocument handles the service document endpoint
func (h *ServiceDocumentHandler) HandleServiceDocument(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetServiceDocument(w, r)
	case http.MethodOptions:
		h.handleOptionsServiceDocument(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for service document", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
	}
}

// handleGetServiceDocument handles GET requests for service document
func (h *ServiceDocumentHandler) handleGetServiceDocument(w http.ResponseWriter, r *http.Request) {
	// Build separate lists for entity sets and singletons
	entitySets := make([]string, 0)
	singletons := make([]string, 0)

	for name, meta := range h.entities {
		if meta.IsSingleton {
			singletons = append(singletons, name)
		} else {
			entitySets = append(entitySets, name)
		}
	}

	if err := response.WriteServiceDocument(w, r, entitySets, singletons); err != nil {
		fmt.Printf("Error writing service document: %v\n", err)
	}
}

// handleOptionsServiceDocument handles OPTIONS requests for service document
func (h *ServiceDocumentHandler) handleOptionsServiceDocument(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)
}
