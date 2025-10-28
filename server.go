package odata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/metadata"
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

		// Check if this is a media entity $value request
		if components.IsValue {
			handler.HandleMediaEntityValue(w, r, keyString)
		} else {
			handler.HandleEntity(w, r, keyString)
		}
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
	} else if handler.IsStreamProperty(components.NavigationProperty) {
		// Stream properties support $value for binary content access
		if components.IsRef {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$ref is not supported on stream properties"); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
		handler.HandleStreamProperty(w, r, keyString, components.NavigationProperty, components.IsValue)
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
		actionCandidates, exists := s.actions[name]
		if !exists {
			if writeErr := response.WriteError(w, http.StatusNotFound, "Action not found",
				fmt.Sprintf("Action '%s' is not registered", name)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Resolve the appropriate overload
		actionDef, params, err := actions.ResolveActionOverload(r, actionCandidates, isBound, entitySet)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid parameters", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Get entity context for bound actions
		var ctx interface{}
		if isBound && key != "" {
			// Fetch the entity from database to verify it exists
			handler := s.handlers[entitySet]
			if handler != nil {
				// Try to fetch the entity to ensure it exists
				entity, err := handler.FetchEntity(key)
				if err != nil {
					// Check if it's a "not found" error
					if handlers.IsNotFoundError(err) {
						if writeErr := response.WriteError(w, http.StatusNotFound, "Entity not found",
							fmt.Sprintf("Entity with key '%s' not found", key)); writeErr != nil {
							fmt.Printf("Error writing error response: %v\n", writeErr)
						}
						return
					}
					// Other database errors
					if writeErr := response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error()); writeErr != nil {
						fmt.Printf("Error writing error response: %v\n", writeErr)
					}
					return
				}
				ctx = entity
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
		functionCandidates, exists := s.functions[name]
		if !exists {
			if writeErr := response.WriteError(w, http.StatusNotFound, "Function not found",
				fmt.Sprintf("Function '%s' is not registered", name)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Resolve the appropriate overload
		functionDef, params, err := actions.ResolveFunctionOverload(r, functionCandidates, isBound, entitySet)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid parameters", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Get entity context for bound functions
		var ctx interface{}
		if isBound && key != "" {
			// Fetch the entity from database to verify it exists
			handler := s.handlers[entitySet]
			if handler != nil {
				// Try to fetch the entity to ensure it exists
				entity, err := handler.FetchEntity(key)
				if err != nil {
					// Check if it's a "not found" error
					if handlers.IsNotFoundError(err) {
						if writeErr := response.WriteError(w, http.StatusNotFound, "Entity not found",
							fmt.Sprintf("Entity with key '%s' not found", key)); writeErr != nil {
							fmt.Printf("Error writing error response: %v\n", writeErr)
						}
						return
					}
					// Other database errors
					if writeErr := response.WriteError(w, http.StatusInternalServerError, "Database error", err.Error()); writeErr != nil {
						fmt.Printf("Error writing error response: %v\n", writeErr)
					}
					return
				}
				ctx = entity
			}
		}

		// Invoke the function handler
		result, err := functionDef.Handler(w, r, ctx, params)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, "Function failed", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		if !response.IsAcceptableFormat(r) {
			if writeErr := response.WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
				"The requested format is not supported. Only application/json is supported for data responses."); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		metadataLevel := response.GetODataMetadataLevel(r)
		w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))

		contextFragment := s.functionContextFragment(functionDef.ReturnType)
		if contextFragment == "" {
			contextFragment = "Edm.String"
		}

		contextURL := ""
		if metadataLevel != "none" && contextFragment != "" {
			contextURL = fmt.Sprintf("%s/$metadata#%s", response.BuildBaseURL(r), contextFragment)
		}

		odataResponse := response.ODataResponse{
			Context: contextURL,
			Value:   result,
		}

		if metadataLevel == "none" {
			odataResponse.Context = ""
		}

		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodHead {
			return
		}

		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(odataResponse); err != nil {
			fmt.Printf("Error encoding response: %v\n", err)
		}

	default:
		if writeErr := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not allowed for actions or functions", r.Method)); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
	}
}

// functionContextFragment builds the metadata fragment for a function return type
func (s *Service) functionContextFragment(returnType reflect.Type) string {
	if returnType == nil {
		return ""
	}

	typ := returnType
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	isCollection := false

	switch typ.Kind() {
	case reflect.Slice:
		if typ.Elem().Kind() != reflect.Uint8 {
			isCollection = true
			typ = typ.Elem()
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
		}
	case reflect.Array:
		if typ.Elem().Kind() != reflect.Uint8 {
			isCollection = true
			typ = typ.Elem()
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
		}
	}

	if edmType, ok := primitiveEdmType(typ); ok {
		if isCollection {
			return fmt.Sprintf("Collection(%s)", edmType)
		}
		return edmType
	}

	if entityMeta := s.entityMetadataByType(typ); entityMeta != nil {
		if isCollection {
			return entityMeta.EntitySetName
		}
		return fmt.Sprintf("%s/$entity", entityMeta.EntitySetName)
	}

	if typ.Kind() == reflect.Struct {
		qualifiedName := buildQualifiedComplexTypeName(typ)
		if qualifiedName == "" {
			return ""
		}
		if isCollection {
			return fmt.Sprintf("Collection(%s)", qualifiedName)
		}
		return qualifiedName
	}

	if typ.Kind() == reflect.Map || typ.Kind() == reflect.Interface {
		if isCollection {
			return "Collection(Edm.Untyped)"
		}
		return "Edm.Untyped"
	}

	return ""
}

func (s *Service) entityMetadataByType(goType reflect.Type) *metadata.EntityMetadata {
	if goType == nil {
		return nil
	}

	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	for _, meta := range s.entities {
		if meta == nil {
			continue
		}
		entityType := meta.EntityType
		if entityType.Kind() == reflect.Ptr {
			entityType = entityType.Elem()
		}
		if entityType == goType {
			return meta
		}
	}

	return nil
}

var (
	timeType = reflect.TypeOf(time.Time{})
)

func primitiveEdmType(goType reflect.Type) (string, bool) {
	if goType == nil {
		return "", false
	}

	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	if goType == timeType {
		return "Edm.DateTimeOffset", true
	}

	if goType.Kind() == reflect.Slice && goType.Elem().Kind() == reflect.Uint8 {
		return "Edm.Binary", true
	}

	if goType.Kind() == reflect.Array && goType.Elem().Kind() == reflect.Uint8 {
		return "Edm.Binary", true
	}

	if pkgPath := goType.PkgPath(); pkgPath != "" {
		switch pkgPath + "." + goType.Name() {
		case "github.com/google/uuid.UUID":
			return "Edm.Guid", true
		}
	}

	switch goType.Kind() {
	case reflect.String:
		return "Edm.String", true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "Edm.Int32", true
	case reflect.Int64:
		return "Edm.Int64", true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "Edm.Int32", true
	case reflect.Uint64:
		return "Edm.Int64", true
	case reflect.Float32:
		return "Edm.Single", true
	case reflect.Float64:
		return "Edm.Double", true
	case reflect.Bool:
		return "Edm.Boolean", true
	}

	return "", false
}

func buildQualifiedComplexTypeName(goType reflect.Type) string {
	if goType == nil {
		return ""
	}

	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	if goType.Name() == "" {
		return ""
	}

	return fmt.Sprintf("ODataService.%s", goType.Name())
}

// Handler returns the Service as an http.Handler.
// This method provides an explicit way to use the Service as a handler,
// though the Service already implements http.Handler directly.
func (s *Service) Handler() http.Handler {
	return s
}
