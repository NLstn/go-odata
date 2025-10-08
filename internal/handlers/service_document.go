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
	if r.Method != http.MethodGet {
		response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for service document", r.Method))
		return
	}

	// Build list of entity sets
	entitySets := make([]string, 0, len(h.entities))
	for entitySetName := range h.entities {
		entitySets = append(entitySets, entitySetName)
	}

	if err := response.WriteServiceDocument(w, r, entitySets); err != nil {
		fmt.Printf("Error writing service document: %v\n", err)
	}
}
