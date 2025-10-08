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

	// Parse the OData URL to extract entity set, key, and navigation property
	components, err := response.ParseODataURLComponents(path)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid URL", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Find the handler for the entity set
	handler, exists := s.handlers[components.EntitySet]
	if !exists {
		if writeErr := response.WriteError(w, http.StatusNotFound, "Entity set not found",
			fmt.Sprintf("Entity set '%s' is not registered", components.EntitySet)); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	// Route to appropriate handler method
	if components.EntityKey == "" {
		// Collection request
		handler.HandleCollection(w, r)
	} else if components.NavigationProperty != "" {
		// Navigation property request: Products(1)/Descriptions
		handler.HandleNavigationProperty(w, r, components.EntityKey, components.NavigationProperty)
	} else {
		// Individual entity request
		handler.HandleEntity(w, r, components.EntityKey)
	}
}

// ListenAndServe starts the OData service on the specified address.
func (s *Service) ListenAndServe(addr string) error {
	fmt.Printf("Starting OData service on %s\n", addr)
	return http.ListenAndServe(addr, s)
}
