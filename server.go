package odata

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/response"
)

// ServeHTTP implements http.Handler interface
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	// Handle root path - service document
	if path == "" {
		s.serviceDocumentHandler.HandleServiceDocument(w, r)
		return
	}

	// Handle metadata document
	if path == "$metadata" {
		s.metadataHandler.HandleMetadata(w, r)
		return
	}

	// Parse the OData URL to extract entity set and key
	entitySet, entityKey, err := response.ParseODataURL(path)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid URL", err.Error())
		return
	}

	// Find the handler for the entity set
	handler, exists := s.handlers[entitySet]
	if !exists {
		response.WriteError(w, http.StatusNotFound, "Entity set not found",
			fmt.Sprintf("Entity set '%s' is not registered", entitySet))
		return
	}

	// Route to appropriate handler method
	if entityKey == "" {
		// Collection request
		handler.HandleCollection(w, r)
	} else {
		// Individual entity request
		handler.HandleEntity(w, r, entityKey)
	}
}

// ListenAndServe starts the OData service on the specified address.
func (s *Service) ListenAndServe(addr string) error {
	fmt.Printf("Starting OData service on %s\n", addr)
	return http.ListenAndServe(addr, s)
}
