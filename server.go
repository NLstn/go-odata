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
	hasKey := components.EntityKey != "" || len(components.EntityKeyMap) > 0

	if !hasKey {
		// Collection request
		handler.HandleCollection(w, r)
	} else if components.NavigationProperty != "" {
		// Navigation property request: Products(1)/Descriptions
		// For composite keys, serialize the key map back to a string
		keyString := components.EntityKey
		if keyString == "" {
			keyString = serializeKeyMap(components.EntityKeyMap)
		}
		handler.HandleNavigationProperty(w, r, keyString, components.NavigationProperty)
	} else {
		// Individual entity request
		// For composite keys, serialize the key map back to a string
		keyString := components.EntityKey
		if keyString == "" {
			keyString = serializeKeyMap(components.EntityKeyMap)
		}
		handler.HandleEntity(w, r, keyString)
	}
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
