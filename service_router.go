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
	// Set OData-Version header for all responses
	// Using helper function to preserve exact capitalization (OData-Version with capital 'D')
	// as specified in OData v4 spec. Header.Set() would canonicalize to "Odata-Version".
	handlers.SetODataVersionHeader(w)

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
	// Functions MUST have parameters in parentheses: GetTopProducts() or GetTopProducts(count=5)
	// Actions do NOT use parentheses (they use POST)
	switch r.Method {
	case http.MethodGet:
		// For GET (functions), require parentheses
		if strings.Contains(path, "(") && strings.Contains(path, ")") {
			pathWithoutParams := path
			if idx := strings.Index(path, "("); idx != -1 {
				pathWithoutParams = path[:idx]
			}
			if s.isActionOrFunction(pathWithoutParams) {
				s.handleActionOrFunction(w, r, pathWithoutParams, "", false, "")
				return
			}
		}
	case http.MethodPost:
		// For POST (actions), no parentheses required
		if s.isActionOrFunction(path) {
			s.handleActionOrFunction(w, r, path, "", false, "")
			return
		}
	case http.MethodPut, http.MethodPatch, http.MethodDelete:
		// Check if this looks like an action/function call (has parentheses or matches registered name)
		pathWithoutParams := path
		if idx := strings.Index(path, "("); idx != -1 {
			pathWithoutParams = path[:idx]
		}
		if s.isActionOrFunction(pathWithoutParams) {
			// Actions/functions don't support these methods
			if writeErr := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
				fmt.Sprintf("Method %s is not allowed for actions or functions", r.Method)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
		// Also check without parentheses
		if s.isActionOrFunction(path) {
			if writeErr := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
				fmt.Sprintf("Method %s is not allowed for actions or functions", r.Method)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
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

	// Validate type cast if present and add to request context
	if components.TypeCast != "" {
		// Extract the type name from the type cast (Namespace.TypeName -> TypeName)
		parts := strings.Split(components.TypeCast, ".")
		if len(parts) < 2 {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid type cast",
				fmt.Sprintf("Type cast '%s' is not in the correct format (Namespace.TypeName)", components.TypeCast)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Add type cast to request context for handlers to use
		typeName := parts[len(parts)-1]
		ctx := r.Context()
		ctx = handlers.WithTypeCast(ctx, typeName)
		r = r.WithContext(ctx)
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
		// $count request: Products/$count or Products(1)/Descriptions/$count
		if hasKey && components.NavigationProperty != "" {
			// Navigation property count: Products(1)/Descriptions/$count
			keyString := s.getKeyString(components)
			handler.HandleNavigationPropertyCount(w, r, keyString, components.NavigationProperty)
		} else if !hasKey && components.NavigationProperty == "" {
			// Collection count: Products/$count
			handler.HandleCount(w, r)
		} else {
			// Invalid: count on entity without navigation property (Products(1)/$count)
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$count is not supported on individual entities. Use $count on collections or navigation properties."); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
		}
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
			// Property access on singleton: /Company/Name or /Company/Navigation
			// For singletons, pass an empty key to indicate we need to fetch the singleton first
			s.handlePropertyRequest(w, r, handler, components)
		} else {
			// Direct singleton access: /Me
			handler.HandleSingleton(w, r)
		}
	} else if !hasKey {
		// Check for invalid operations on collections
		if components.IsValue {
			// $value is not supported on collections
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$value is not supported on entity collections. Use $value on individual properties: EntitySet(key)/PropertyName/$value"); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
		// Check if this is a bound action/function on the collection
		if components.NavigationProperty != "" {
			// Strip parentheses from operation name for lookup
			operationName := components.NavigationProperty
			if idx := strings.Index(operationName, "("); idx != -1 {
				operationName = operationName[:idx]
			}
			if s.isActionOrFunction(operationName) {
				s.handleActionOrFunction(w, r, operationName, "", true, components.EntitySet)
				return
			}
		}
		if components.NavigationProperty != "" {
			// Navigation property or action/function not found on collection
			if writeErr := response.WriteError(w, http.StatusNotFound, "Property or operation not found",
				fmt.Sprintf("'%s' is not a valid property, action, or function for %s", components.NavigationProperty, components.EntitySet)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
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

	// Strip parentheses from operation name for lookup
	operationName := propertyOrAction
	if idx := strings.Index(operationName, "("); idx != -1 {
		operationName = operationName[:idx]
	}

	// Try action/function first (bound operations)
	if s.isActionOrFunction(operationName) {
		s.handleActionOrFunction(w, r, operationName, keyString, true, components.EntitySet)
		return
	}

	propertySegments := components.PropertySegments
	if len(propertySegments) == 0 && components.NavigationProperty != "" {
		propertySegments = []string{components.NavigationProperty}
	}

	// Try navigation property first, then structural property, then complex type
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
	} else if handler.IsComplexTypeProperty(components.NavigationProperty) {
		if components.IsRef {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$ref is not supported on complex properties"); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		handler.HandleComplexTypeProperty(w, r, keyString, propertySegments, components.IsValue)
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
