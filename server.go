package odata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/response"
)

// ServeHTTP implements http.Handler interface
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validate OData version before processing any request
	if !handlers.ValidateODataVersion(r) {
		if err := response.WriteError(w, http.StatusNotAcceptable,
			handlers.ErrMsgVersionNotSupported,
			handlers.ErrDetailVersionNotSupported); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

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

	// Check if this is an unbound action or function (no entity set in path)
	if s.isActionOrFunction(path) {
		s.handleActionOrFunction(w, r, path, "", false, "")
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

	// Check if this is a singleton
	isSingleton := handler.IsSingleton()

	if components.IsCount {
		// $count request: Products/$count
		handler.HandleCount(w, r)
	} else if components.IsRef {
		// $ref request: Products/$ref or Products(1)/$ref
		if hasKey && components.NavigationProperty == "" {
			// Entity reference: Products(1)/$ref
			keyString := s.getKeyString(components)
			handler.HandleEntityRef(w, r, keyString)
		} else if !hasKey && components.NavigationProperty == "" {
			// Collection reference: Products/$ref
			handler.HandleCollectionRef(w, r)
		} else {
			// Navigation property reference handled in handlePropertyRequest
			s.handlePropertyRequest(w, r, handler, components)
		}
	} else if isSingleton {
		// Singleton request - treat as single entity without key
		if components.NavigationProperty != "" {
			// Navigation property on singleton: /Me/Friends
			s.handlePropertyRequest(w, r, handler, components)
		} else {
			// Direct singleton access: /Me
			handler.HandleSingleton(w, r)
		}
	} else if !hasKey {
		// Check if this is an unbound action/function on the collection
		if components.NavigationProperty != "" && s.isActionOrFunction(components.NavigationProperty) {
			s.handleActionOrFunction(w, r, components.NavigationProperty, "", false, components.EntitySet)
		} else {
			// Collection request
			handler.HandleCollection(w, r)
		}
	} else if components.NavigationProperty != "" {
		s.handlePropertyRequest(w, r, handler, components)
	} else {
		// Individual entity request
		keyString := s.getKeyString(components)
		handler.HandleEntity(w, r, keyString)
	}
}

// handlePropertyRequest handles navigation and structural property requests, as well as actions/functions
func (s *Service) handlePropertyRequest(w http.ResponseWriter, r *http.Request, handler *handlers.EntityHandler, components *response.ODataURLComponents) {
	keyString := s.getKeyString(components)

	// Check if this is an action or function invocation (bound to entity)
	propertyOrAction := components.NavigationProperty

	// Try action/function first (bound operations)
	if s.isActionOrFunction(propertyOrAction) {
		s.handleActionOrFunction(w, r, propertyOrAction, keyString, true, components.EntitySet)
		return
	}

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
		handler.HandleNavigationProperty(w, r, keyString, components.NavigationProperty, components.IsRef)
	} else if handler.IsStructuralProperty(components.NavigationProperty) {
		if components.IsRef {
			// /$ref is not supported on structural properties
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$ref is not supported on structural properties"); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
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

// isActionOrFunction checks if a name corresponds to a registered action or function
func (s *Service) isActionOrFunction(name string) bool {
	_, isAction := s.actions[name]
	_, isFunction := s.functions[name]
	return isAction || isFunction
}

// handleActionOrFunction handles action or function invocation
func (s *Service) handleActionOrFunction(w http.ResponseWriter, r *http.Request, name string, key string, isBound bool, entitySet string) {
	// Check if it's an action (POST) or function (GET)
	switch r.Method {
	case http.MethodPost:
		// Handle action
		actionDef, exists := s.actions[name]
		if !exists {
			if writeErr := response.WriteError(w, http.StatusNotFound, "Action not found",
				fmt.Sprintf("Action '%s' is not registered", name)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Verify binding matches
		if isBound != actionDef.IsBound {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid action binding",
				fmt.Sprintf("Action '%s' binding mismatch", name)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		if isBound && actionDef.EntitySet != entitySet {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid entity set",
				fmt.Sprintf("Action '%s' is not bound to entity set '%s'", name, entitySet)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Parse parameters from request body
		params, err := actions.ParseActionParameters(r, actionDef.Parameters)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid parameters", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Get entity context for bound actions
		var ctx interface{}
		if isBound && key != "" {
			// Fetch the entity from database
			handler := s.handlers[entitySet]
			if handler != nil {
				// For now, we'll pass nil as context
				// In a full implementation, we'd fetch the entity here
				ctx = nil
			}
		}

		// Invoke the action handler
		if err := actionDef.Handler(w, r, ctx, params); err != nil {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, "Action failed", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

	case http.MethodGet:
		// Handle function
		functionDef, exists := s.functions[name]
		if !exists {
			if writeErr := response.WriteError(w, http.StatusNotFound, "Function not found",
				fmt.Sprintf("Function '%s' is not registered", name)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Verify binding matches
		if isBound != functionDef.IsBound {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid function binding",
				fmt.Sprintf("Function '%s' binding mismatch", name)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		if isBound && functionDef.EntitySet != entitySet {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid entity set",
				fmt.Sprintf("Function '%s' is not bound to entity set '%s'", name, entitySet)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Parse parameters from query string
		params, err := actions.ParseFunctionParameters(r, functionDef.Parameters)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid parameters", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Get entity context for bound functions
		var ctx interface{}
		if isBound && key != "" {
			// For now, we'll pass nil as context
			// In a full implementation, we'd fetch the entity here
			ctx = nil
		}

		// Invoke the function handler
		result, err := functionDef.Handler(w, r, ctx, params)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, "Function failed", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Write the result with dynamic metadata level
		metadataLevel := response.GetODataMetadataLevel(r)
		w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
		w.Header().Set("OData-Version", "4.0")
		w.WriteHeader(http.StatusOK)

		responseMap := map[string]interface{}{
			"@odata.context": "$metadata#Edm.String",
			"value":          result,
		}

		if err := json.NewEncoder(w).Encode(responseMap); err != nil {
			fmt.Printf("Error encoding response: %v\n", err)
		}

	default:
		if writeErr := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not allowed for actions or functions", r.Method)); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
	}
}

// ListenAndServe starts the OData service on the specified address.
func (s *Service) ListenAndServe(addr string) error {
	fmt.Printf("Starting OData service on %s\n", addr)
	return http.ListenAndServe(addr, s)
}
