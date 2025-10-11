package odata

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/handlers"
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

	// Handle batch requests
	if path == "$batch" {
		s.batchHandler.HandleBatch(w, r)
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
	s.routeRequest(w, r, handler, components)
}

// routeRequest routes the request to the appropriate handler method based on URL components
func (s *Service) routeRequest(w http.ResponseWriter, r *http.Request, handler *handlers.EntityHandler, components *response.ODataURLComponents) {
	hasKey := components.EntityKey != "" || len(components.EntityKeyMap) > 0

	if components.IsCount {
		// $count request: Products/$count
		handler.HandleCount(w, r)
	} else if !hasKey {
		// Collection request
		handler.HandleCollection(w, r)
	} else if components.NavigationProperty != "" {
		s.handlePropertyRequest(w, r, handler, components)
	} else {
		// Individual entity request
		keyString := s.getKeyString(components)
		handler.HandleEntity(w, r, keyString)
	}
}

// handlePropertyRequest handles navigation and structural property requests
func (s *Service) handlePropertyRequest(w http.ResponseWriter, r *http.Request, handler *handlers.EntityHandler, components *response.ODataURLComponents) {
	keyString := s.getKeyString(components)

	// Try navigation property first, then structural property
	if handler.IsNavigationProperty(components.NavigationProperty) {
		if components.IsValue {
			// /$value is not supported on navigation properties
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$value is not supported on navigation properties"); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
		handler.HandleNavigationProperty(w, r, keyString, components.NavigationProperty)
	} else if handler.IsStructuralProperty(components.NavigationProperty) {
		handler.HandleStructuralProperty(w, r, keyString, components.NavigationProperty, components.IsValue)
	} else {
		// Property not found
		if writeErr := response.WriteError(w, http.StatusNotFound, "Property not found",
			fmt.Sprintf("'%s' is not a valid property for %s", components.NavigationProperty, components.EntitySet)); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
	}
}

// getKeyString returns the key string from components, serializing the key map if needed
func (s *Service) getKeyString(components *response.ODataURLComponents) string {
	if components.EntityKey != "" {
		return components.EntityKey
	}
	return serializeKeyMap(components.EntityKeyMap)
}

// serializeKeyMap converts a key map to a string format for handlers
// For composite keys: returns "ProductID=1,LanguageKey='EN'"
func serializeKeyMap(keyMap map[string]string) string {
	if len(keyMap) == 0 {
		return ""
	}

	var parts []string
	for key, value := range keyMap {
		// Check if value looks like a string (simple heuristic)
		// If it contains only digits, treat as number, otherwise quote it
		isNumeric := true
		for _, ch := range value {
			if ch < '0' || ch > '9' {
				isNumeric = false
				break
			}
		}

		if isNumeric {
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
		} else {
			parts = append(parts, fmt.Sprintf("%s='%s'", key, value))
		}
	}

	// Sort for consistency (optional, but helps with testing)
	return strings.Join(parts, ",")
}

// ListenAndServe starts the OData service on the specified address.
func (s *Service) ListenAndServe(addr string) error {
	fmt.Printf("Starting OData service on %s\n", addr)
	return http.ListenAndServe(addr, s)
}
