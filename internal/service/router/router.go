package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/response"
)

// EntityHandler defines the behavior required from entity handlers by the router.
type EntityHandler interface {
	IsSingleton() bool
	HandleCollection(http.ResponseWriter, *http.Request)
	HandleEntity(http.ResponseWriter, *http.Request, string)
	HandleSingleton(http.ResponseWriter, *http.Request)
	HandleCount(http.ResponseWriter, *http.Request)
	HandleNavigationPropertyCount(http.ResponseWriter, *http.Request, string, string)
	HandleEntityRef(http.ResponseWriter, *http.Request, string)
	HandleCollectionRef(http.ResponseWriter, *http.Request)
	HandleNavigationProperty(http.ResponseWriter, *http.Request, string, string, bool)
	HandleStreamProperty(http.ResponseWriter, *http.Request, string, string, bool)
	HandleStructuralProperty(http.ResponseWriter, *http.Request, string, string, bool)
	HandleComplexTypeProperty(http.ResponseWriter, *http.Request, string, []string, bool)
	HandleMediaEntityValue(http.ResponseWriter, *http.Request, string)
	IsNavigationProperty(string) bool
	IsStreamProperty(string) bool
	IsStructuralProperty(string) bool
	IsComplexTypeProperty(string) bool
	FetchEntity(string) (interface{}, error)
}

// HandlerResolver resolves an entity handler for the given entity set.
type HandlerResolver func(string) (EntityHandler, bool)

// ActionInvoker invokes bound or unbound actions and functions.
type ActionInvoker func(http.ResponseWriter, *http.Request, string, string, bool, string)

// Router routes incoming HTTP requests to the appropriate handlers.
type Router struct {
	resolveHandler        HandlerResolver
	handleServiceDocument func(http.ResponseWriter, *http.Request)
	handleMetadata        func(http.ResponseWriter, *http.Request)
	handleBatch           func(http.ResponseWriter, *http.Request)
	actions               map[string][]*actions.ActionDefinition
	functions             map[string][]*actions.FunctionDefinition
	actionInvoker         ActionInvoker
}

// NewRouter creates a new Router instance.
func NewRouter(
	resolver HandlerResolver,
	serviceDocumentHandler func(http.ResponseWriter, *http.Request),
	metadataHandler func(http.ResponseWriter, *http.Request),
	batchHandler func(http.ResponseWriter, *http.Request),
	actions map[string][]*actions.ActionDefinition,
	functions map[string][]*actions.FunctionDefinition,
	actionInvoker ActionInvoker,
) *Router {
	return &Router{
		resolveHandler:        resolver,
		handleServiceDocument: serviceDocumentHandler,
		handleMetadata:        metadataHandler,
		handleBatch:           batchHandler,
		actions:               actions,
		functions:             functions,
		actionInvoker:         actionInvoker,
	}
}

// ServeHTTP implements http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handlers.SetODataVersionHeader(w)

	if !handlers.ValidateODataVersion(req) {
		if err := response.WriteError(w, http.StatusNotAcceptable,
			handlers.ErrMsgVersionNotSupported,
			handlers.ErrDetailVersionNotSupported); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/")

	if path == "" {
		r.handleServiceDocument(w, req)
		return
	}

	if path == "$metadata" {
		r.handleMetadata(w, req)
		return
	}

	if path == "$batch" {
		r.handleBatch(w, req)
		return
	}

	switch req.Method {
	case http.MethodGet:
		if strings.Contains(path, "(") && strings.Contains(path, ")") {
			pathWithoutParams := path
			if idx := strings.Index(path, "("); idx != -1 {
				pathWithoutParams = path[:idx]
			}
			if r.isActionOrFunction(pathWithoutParams) {
				r.actionInvoker(w, req, pathWithoutParams, "", false, "")
				return
			}
		}
	case http.MethodPost:
		if r.isActionOrFunction(path) {
			r.actionInvoker(w, req, path, "", false, "")
			return
		}
	case http.MethodPut, http.MethodPatch, http.MethodDelete:
		pathWithoutParams := path
		if idx := strings.Index(path, "("); idx != -1 {
			pathWithoutParams = path[:idx]
		}
		if r.isActionOrFunction(pathWithoutParams) {
			if writeErr := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
				fmt.Sprintf("Method %s is not allowed for actions or functions", req.Method)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
		if r.isActionOrFunction(path) {
			if writeErr := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
				fmt.Sprintf("Method %s is not allowed for actions or functions", req.Method)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
	}

	components, err := response.ParseODataURLComponents(path)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid URL", err.Error()); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	handler, exists := r.resolveHandler(components.EntitySet)
	if !exists {
		if writeErr := response.WriteError(w, http.StatusNotFound, "Entity set not found",
			fmt.Sprintf("Entity set '%s' is not registered", components.EntitySet)); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
		return
	}

	if components.TypeCast != "" {
		parts := strings.Split(components.TypeCast, ".")
		if len(parts) < 2 {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid type cast",
				fmt.Sprintf("Type cast '%s' is not in the correct format (Namespace.TypeName)", components.TypeCast)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		typeName := parts[len(parts)-1]
		ctx := req.Context()
		ctx = handlers.WithTypeCast(ctx, typeName)
		req = req.WithContext(ctx)
	}

	r.routeRequest(w, req, handler, components)
}

func (r *Router) routeRequest(w http.ResponseWriter, req *http.Request, handler EntityHandler, components *response.ODataURLComponents) {
	hasKey := components.EntityKey != "" || len(components.EntityKeyMap) > 0
	isSingleton := handler.IsSingleton()

	if components.IsCount {
		if hasKey && components.NavigationProperty != "" {
			keyString := r.getKeyString(components)
			handler.HandleNavigationPropertyCount(w, req, keyString, components.NavigationProperty)
		} else if !hasKey && components.NavigationProperty == "" {
			handler.HandleCount(w, req)
		} else {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$count is not supported on individual entities. Use $count on collections or navigation properties."); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
		}
	} else if components.IsRef {
		if hasKey && components.NavigationProperty == "" {
			keyString := r.getKeyString(components)
			handler.HandleEntityRef(w, req, keyString)
		} else if !hasKey && components.NavigationProperty == "" {
			handler.HandleCollectionRef(w, req)
		} else {
			r.handlePropertyRequest(w, req, handler, components)
		}
	} else if isSingleton {
		if components.NavigationProperty != "" {
			r.handlePropertyRequest(w, req, handler, components)
		} else {
			handler.HandleSingleton(w, req)
		}
	} else if !hasKey {
		if components.IsValue {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$value is not supported on entity collections. Use $value on individual properties: EntitySet(key)/PropertyName/$value"); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
		if components.NavigationProperty != "" {
			operationName := components.NavigationProperty
			if idx := strings.Index(operationName, "("); idx != -1 {
				operationName = operationName[:idx]
			}
			if r.isActionOrFunction(operationName) {
				r.actionInvoker(w, req, operationName, "", true, components.EntitySet)
				return
			}
		}
		if components.NavigationProperty != "" {
			if writeErr := response.WriteError(w, http.StatusNotFound, "Property or operation not found",
				fmt.Sprintf("'%s' is not a valid property, action, or function for %s", components.NavigationProperty, components.EntitySet)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
		} else {
			handler.HandleCollection(w, req)
		}
	} else if components.NavigationProperty != "" {
		r.handlePropertyRequest(w, req, handler, components)
	} else {
		keyString := r.getKeyString(components)
		if components.IsValue {
			handler.HandleMediaEntityValue(w, req, keyString)
		} else {
			handler.HandleEntity(w, req, keyString)
		}
	}
}

func (r *Router) handlePropertyRequest(w http.ResponseWriter, req *http.Request, handler EntityHandler, components *response.ODataURLComponents) {
	keyString := r.getKeyString(components)
	propertyOrAction := components.NavigationProperty

	operationName := propertyOrAction
	if idx := strings.Index(operationName, "("); idx != -1 {
		operationName = operationName[:idx]
	}

	if r.isActionOrFunction(operationName) {
		r.actionInvoker(w, req, operationName, keyString, true, components.EntitySet)
		return
	}

	propertySegments := components.PropertySegments
	if len(propertySegments) == 0 && components.NavigationProperty != "" {
		propertySegments = []string{components.NavigationProperty}
	}

	if handler.IsNavigationProperty(components.NavigationProperty) {
		if components.IsValue {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$value is not supported on navigation properties"); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
		handler.HandleNavigationProperty(w, req, keyString, components.NavigationProperty, components.IsRef)
	} else if handler.IsStreamProperty(components.NavigationProperty) {
		if components.IsRef {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$ref is not supported on stream properties"); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
		handler.HandleStreamProperty(w, req, keyString, components.NavigationProperty, components.IsValue)
	} else if handler.IsStructuralProperty(components.NavigationProperty) {
		if components.IsRef {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$ref is not supported on structural properties"); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
		handler.HandleStructuralProperty(w, req, keyString, components.NavigationProperty, components.IsValue)
	} else if handler.IsComplexTypeProperty(components.NavigationProperty) {
		if components.IsRef {
			if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$ref is not supported on complex properties"); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}
		handler.HandleComplexTypeProperty(w, req, keyString, propertySegments, components.IsValue)
	} else {
		if writeErr := response.WriteError(w, http.StatusNotFound, "Property not found",
			fmt.Sprintf("'%s' is not a valid property for %s", components.NavigationProperty, components.EntitySet)); writeErr != nil {
			fmt.Printf("Error writing error response: %v\n", writeErr)
		}
	}
}

func (r *Router) getKeyString(components *response.ODataURLComponents) string {
	if components.EntityKey != "" {
		return components.EntityKey
	}
	return r.serializeKeyMap(components.EntityKeyMap)
}

func (r *Router) serializeKeyMap(keyMap map[string]string) string {
	if len(keyMap) == 0 {
		return ""
	}

	var parts []string
	for key, value := range keyMap {
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

	return strings.Join(parts, ",")
}

func (r *Router) isActionOrFunction(name string) bool {
	if name == "" {
		return false
	}
	if _, ok := r.actions[name]; ok {
		return true
	}
	if _, ok := r.functions[name]; ok {
		return true
	}
	return false
}
