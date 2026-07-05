package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/async"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/version"
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
	NavigationTargetSet(string) (string, bool)
	FetchEntity(string) (interface{}, error)
	FetchNavEntityKey(entityKey, navPropName string) (string, error)
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
	logger                *slog.Logger

	// Protects async monitor configuration to avoid races between
	// SetAsyncMonitor and concurrent request handling.
	asyncMu            sync.RWMutex
	asyncManager       *async.Manager
	asyncMonitorPrefix string

	// namespace is the schema namespace advertised in $metadata. It is used to
	// resolve namespace-qualified action/function path segments (e.g.
	// "ComplianceService.GetTotalPrice") to the unqualified name under which
	// the operation is registered. Protected by namespaceMu since it can be
	// updated concurrently with request handling via Service.SetNamespace.
	namespaceMu sync.RWMutex
	namespace   string
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
	logger *slog.Logger,
) *Router {
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{
		resolveHandler:        resolver,
		handleServiceDocument: serviceDocumentHandler,
		handleMetadata:        metadataHandler,
		handleBatch:           batchHandler,
		actions:               actions,
		functions:             functions,
		actionInvoker:         actionInvoker,
		logger:                logger,
	}
}

// SetLogger sets the logger for the router.
func (r *Router) SetLogger(logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	r.logger = logger
}

// SetNamespace updates the schema namespace used to resolve namespace-qualified
// action/function invocations (OData v4 Part 2 §4.5).
func (r *Router) SetNamespace(namespace string) {
	r.namespaceMu.Lock()
	r.namespace = namespace
	r.namespaceMu.Unlock()
}

func (r *Router) getNamespace() string {
	r.namespaceMu.RLock()
	defer r.namespaceMu.RUnlock()
	return r.namespace
}

// ServeHTTP implements http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Check if client's OData-MaxVersion is below 4.0 (reject old versions)
	// Per OData spec 8.2.9, invalid version formats should be ignored (treated as if no header was sent)
	clientMaxVersion := req.Header.Get(handlers.HeaderODataMaxVersion)
	if clientMaxVersion != "" {
		clientVersion := version.ParseVersionString(clientMaxVersion)
		// Invalid version strings are parsed as 0.0 and should be ignored per spec
		// Only reject versions that are validly parsed but below 4.0
		if clientVersion.Major > 0 && clientVersion.Major < 4 {
			if err := response.WriteError(w, req, http.StatusNotAcceptable,
				handlers.ErrMsgVersionNotSupported,
				handlers.ErrDetailVersionNotSupported); err != nil {
				r.logger.Error("Error writing error response", "error", err)
			}
			return
		}
		// Log invalid formats but don't reject (treat as no header per OData spec)
		if clientVersion.Major == 0 && clientVersion.Minor == 0 {
			r.logger.Debug("Invalid OData-MaxVersion header ignored, using default", "version", clientMaxVersion)
		}
	}

	// Negotiate OData version based on client's OData-MaxVersion header
	negotiatedVersion := version.NegotiateVersion(clientMaxVersion)

	// Store the negotiated version in the request context
	ctx := version.WithVersion(req.Context(), negotiatedVersion)
	req = req.WithContext(ctx)

	// Set the OData-Version header based on the negotiated version
	// Use direct assignment to preserve exact casing per OData spec
	w.Header()[response.HeaderODataVersion] = []string{negotiatedVersion.String()}

	if r.tryServeAsyncMonitor(w, req) {
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
	case http.MethodGet, http.MethodHead:
		if !strings.Contains(path, "(") && !strings.Contains(path, ")") {
			if resolved, ok := r.resolveBoundName(path); ok {
				if _, exists := r.functions[resolved]; exists {
					supportsNoParens := negotiatedVersion.Major > 4 || (negotiatedVersion.Major == 4 && negotiatedVersion.Minor >= 1)
					if !supportsNoParens {
						if writeErr := response.WriteError(w, req, http.StatusBadRequest, "Invalid function invocation",
							"Function invocations without parentheses require OData 4.01 negotiation; use parentheses syntax (FunctionName())"); writeErr != nil {
							r.logger.Error("Error writing error response", "error", writeErr)
						}
						return
					}

					if r.hasUnboundParameterlessFunction(resolved) && !hasNonSystemQueryParams(req.URL.Query()) {
						r.actionInvoker(w, req, resolved, "", false, "")
						return
					}
				}
			}
		}

		if strings.Contains(path, "(") && strings.Contains(path, ")") {
			pathWithoutParams := path
			if idx := strings.Index(path, "("); idx != -1 {
				pathWithoutParams = path[:idx]
			}
			if resolved, ok := r.resolveBoundName(pathWithoutParams); ok && r.isActionOrFunction(resolved) {
				r.actionInvoker(w, req, resolved, "", false, "")
				return
			}
		}
	case http.MethodPost:
		if resolved, ok := r.resolveBoundName(path); ok && r.isActionOrFunction(resolved) {
			r.actionInvoker(w, req, resolved, "", false, "")
			return
		}
	case http.MethodPut, http.MethodPatch, http.MethodDelete:
		pathWithoutParams := path
		if idx := strings.Index(path, "("); idx != -1 {
			pathWithoutParams = path[:idx]
		}
		if r.isBoundOperationSegment(pathWithoutParams) {
			if writeErr := response.WriteMethodNotAllowed(w, req, "GET, HEAD, POST, OPTIONS", "Method not allowed",
				fmt.Sprintf("Method %s is not allowed for actions or functions", req.Method)); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
		if r.isBoundOperationSegment(path) {
			if writeErr := response.WriteMethodNotAllowed(w, req, "GET, HEAD, POST, OPTIONS", "Method not allowed",
				fmt.Sprintf("Method %s is not allowed for actions or functions", req.Method)); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
	}

	components, err := response.ParseODataURLComponents(path)
	if err != nil {
		if writeErr := response.WriteError(w, req, http.StatusBadRequest, "Invalid URL", err.Error()); writeErr != nil {
			r.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	handler, exists := r.resolveHandler(components.EntitySet)
	if !exists {
		if writeErr := response.WriteError(w, req, http.StatusNotFound, "Entity set not found",
			fmt.Sprintf("Entity set '%s' is not registered", components.EntitySet)); writeErr != nil {
			r.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// A namespace-qualified action/function segment without parentheses (e.g.
	// "ComplianceService.ApplyDiscount") is syntactically indistinguishable
	// from a derived-type cast segment (e.g. "ComplianceService.Manager") and
	// so gets parsed as components.TypeCast above. Reinterpret it as an
	// operation invocation when it actually resolves to a registered bound or
	// unbound action/function; otherwise leave the type-cast parsing as-is.
	if components.TypeCast != "" && components.NavigationProperty == "" {
		if resolved, ok := r.resolveBoundName(components.TypeCast); ok && r.isActionOrFunction(resolved) {
			components.NavigationProperty = resolved
			components.PropertySegments = []string{resolved}
			components.PropertyPath = resolved
			components.TypeCast = ""
		} else if r.isUnresolvableQualifiedOperation(components.TypeCast) {
			if writeErr := response.WriteError(w, req, http.StatusNotFound, "Action or function not found",
				fmt.Sprintf("'%s' does not match the service namespace", components.TypeCast)); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
	}

	if components.TypeCast != "" {
		parts := strings.Split(components.TypeCast, ".")
		if len(parts) < 2 {
			if writeErr := response.WriteError(w, req, http.StatusBadRequest, "Invalid type cast",
				fmt.Sprintf("Type cast '%s' is not in the correct format (Namespace.TypeName)", components.TypeCast)); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
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

func (r *Router) hasUnboundParameterlessFunction(name string) bool {
	defs, ok := r.functions[name]
	if !ok {
		return false
	}

	for _, def := range defs {
		if def == nil {
			continue
		}
		if !def.IsBound && len(def.Parameters) == 0 {
			return true
		}
	}

	return false
}

func hasNonSystemQueryParams(values map[string][]string) bool {
	for key := range values {
		if !strings.HasPrefix(key, "$") {
			return true
		}
	}
	return false
}

// SetAsyncMonitor configures the router to delegate monitor requests to the async manager.
func (r *Router) SetAsyncMonitor(prefix string, manager *async.Manager) {
	if prefix != "" {
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
	}

	r.asyncMu.Lock()
	r.asyncMonitorPrefix = prefix
	r.asyncManager = manager
	r.asyncMu.Unlock()
}

func (r *Router) tryServeAsyncMonitor(w http.ResponseWriter, req *http.Request) bool {
	r.asyncMu.RLock()
	manager := r.asyncManager
	prefix := r.asyncMonitorPrefix
	r.asyncMu.RUnlock()

	if manager == nil || prefix == "" || req == nil || req.URL == nil {
		return false
	}

	path := req.URL.Path
	if !strings.HasPrefix(path, prefix) {
		return false
	}

	suffix := strings.TrimPrefix(path, prefix)
	if suffix == "" {
		http.NotFound(w, req)
		return true
	}
	if strings.Contains(suffix, "/") {
		http.NotFound(w, req)
		return true
	}
	if !isValidAsyncJobID(suffix) {
		http.Error(w, "invalid async job identifier", http.StatusBadRequest)
		return true
	}

	manager.ServeMonitor(w, req)
	return true
}

func isValidAsyncJobID(id string) bool {
	if id == "" {
		return false
	}
	for _, ch := range id {
		switch {
		case ch >= '0' && ch <= '9':
			continue
		case ch >= 'a' && ch <= 'z':
			continue
		case ch >= 'A' && ch <= 'Z':
			continue
		case ch == '-' || ch == '_':
			continue
		default:
			if ch > 127 {
				return false
			}
			return false
		}
	}
	return true
}

func (r *Router) routeRequest(w http.ResponseWriter, req *http.Request, handler EntityHandler, components *response.ODataURLComponents) {
	hasKey := components.EntityKey != "" || len(components.EntityKeyMap) > 0
	isSingleton := handler.IsSingleton()

	// OData 4.01 key-as-segments convention: if no parenthetical key was parsed
	// but there is a segment after the entity set that is not a known property,
	// treat it as the entity key (e.g., /Products/1 → key "1").
	if !hasKey && !isSingleton && components.NavigationProperty != "" {
		ver := version.GetVersion(req.Context())
		if ver.Supports("key-as-segments") {
			potentialKey := components.NavigationProperty
			operationName := potentialKey
			if idx := strings.Index(operationName, "("); idx != -1 {
				operationName = operationName[:idx]
			}
			if !handler.IsNavigationProperty(potentialKey) &&
				!handler.IsStructuralProperty(potentialKey) &&
				!handler.IsStreamProperty(potentialKey) &&
				!handler.IsComplexTypeProperty(potentialKey) &&
				!r.isBoundOperationSegment(operationName) {
				components = resolveKeyAsSegment(components)
				hasKey = true
			}
		}
	}

	if components.IsCount {
		if hasKey && components.NavigationProperty != "" {
			keyString := r.getKeyString(components)

			// A trailing $count segment can also follow a bound function-call
			// segment, e.g. Products(1)/GetRelatedProducts()/$count. Per OData
			// v4.0 Part 1 §12.1, functions returning a collection are
			// composable, so $count must resolve against the function's
			// result collection rather than being treated as an (invalid)
			// navigation property named "GetRelatedProducts()".
			operationName := components.NavigationProperty
			if idx := strings.Index(operationName, "("); idx != -1 {
				operationName = operationName[:idx]
			}
			if r.isActionOrFunction(operationName) {
				req = actions.WithCountRequested(req)
				r.actionInvoker(w, req, operationName, keyString, true, components.EntitySet)
				return
			}

			handler.HandleNavigationPropertyCount(w, req, keyString, components.NavigationProperty)
		} else if !hasKey && components.NavigationProperty == "" {
			handler.HandleCount(w, req)
		} else {
			if writeErr := response.WriteError(w, req, http.StatusBadRequest, "Invalid request",
				"$count is not supported on individual entities. Use $count on collections or navigation properties."); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
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
			if writeErr := response.WriteError(w, req, http.StatusBadRequest, "Invalid request",
				"$value is not supported on entity collections. Use $value on individual properties: EntitySet(key)/PropertyName/$value"); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
		if components.NavigationProperty != "" {
			operationName := components.NavigationProperty
			if idx := strings.Index(operationName, "("); idx != -1 {
				operationName = operationName[:idx]
			}
			if resolved, ok := r.resolveBoundName(operationName); ok && r.isActionOrFunction(resolved) {
				r.actionInvoker(w, req, resolved, "", true, components.EntitySet)
				return
			}
		}
		if components.NavigationProperty != "" {
			if writeErr := response.WriteError(w, req, http.StatusNotFound, "Property or operation not found",
				fmt.Sprintf("'%s' is not a valid property, action, or function for %s", components.NavigationProperty, components.EntitySet)); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
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

	if resolved, ok := r.resolveBoundName(operationName); ok && r.isActionOrFunction(resolved) {
		r.actionInvoker(w, req, resolved, keyString, true, components.EntitySet)
		return
	}

	propertySegments := components.PropertySegments
	if len(propertySegments) == 0 && components.NavigationProperty != "" {
		propertySegments = []string{components.NavigationProperty}
	}

	// Check for function composition after navigation property
	// e.g., Categories(1)/Products/GetAveragePrice()
	if len(propertySegments) > 1 {
		firstSegment := propertySegments[0]
		lastSegment := propertySegments[len(propertySegments)-1]

		// Extract operation name from last segment (remove parameters)
		lastOperationName := lastSegment
		if idx := strings.Index(lastSegment, "("); idx != -1 {
			lastOperationName = lastSegment[:idx]
		}
		if resolved, ok := r.resolveBoundName(lastOperationName); ok {
			lastOperationName = resolved
		}

		// Check if first segment is a navigation property and last segment is an operation
		if handler.IsNavigationProperty(firstSegment) && r.isActionOrFunction(lastOperationName) {
			// This is function composition: navigate first, then invoke operation
			// We need to get the target entity set for the navigation property
			targetEntitySet := r.getNavigationTargetEntitySet(handler, firstSegment)
			if targetEntitySet == "" {
				if writeErr := response.WriteError(w, req, http.StatusInternalServerError, "Internal error",
					fmt.Sprintf("Could not determine target entity set for navigation property '%s'", firstSegment)); writeErr != nil {
					r.logger.Error("Error writing error response", "error", writeErr)
				}
				return
			}

			// Inject parent binding context so that action/function handlers can
			// access the parent entity set, key, and navigation property that were
			// used to reach the bound target (e.g. Categories(1)/Products/...).
			req = actions.WithNavigationBindingContext(req, &actions.NavigationBindingContext{
				ParentEntitySet:    components.EntitySet,
				ParentKey:          keyString,
				NavigationProperty: firstSegment,
			})

			r.actionInvoker(w, req, lastOperationName, "", true, targetEntitySet)
			return
		}

		// Handle chained navigation properties (e.g. Products(1)/Category/Products).
		// OData §11.2.4.2: each navigation segment is resolved in turn against its parent entity.
		if handler.IsNavigationProperty(firstSegment) {
			targetEntitySet := r.getNavigationTargetEntitySet(handler, firstSegment)
			targetHandler, targetExists := r.resolveHandler(targetEntitySet)
			if targetExists {
				intermediateKey, err := handler.FetchNavEntityKey(keyString, firstSegment)
				if err != nil {
					statusCode := http.StatusInternalServerError
					if handlers.IsNotFoundError(err) {
						statusCode = http.StatusNotFound
					}
					if writeErr := response.WriteError(w, req, statusCode, "Navigation path error",
						fmt.Sprintf("Could not resolve navigation path: %v", err)); writeErr != nil {
						r.logger.Error("Error writing error response", "error", writeErr)
					}
					return
				}
				newComponents := &response.ODataURLComponents{
					EntitySet:          targetEntitySet,
					EntityKey:          intermediateKey,
					EntityKeyMap:       make(map[string]string),
					NavigationProperty: propertySegments[1],
					PropertySegments:   propertySegments[1:],
					PropertyPath:       strings.Join(propertySegments[1:], "/"),
					IsRef:              components.IsRef,
					IsCount:            components.IsCount,
				}
				r.handlePropertyRequest(w, req, targetHandler, newComponents)
				return
			}
		}
	}

	if handler.IsNavigationProperty(components.NavigationProperty) {
		if components.IsValue {
			if writeErr := response.WriteError(w, req, http.StatusBadRequest, "Invalid request",
				"$value is not supported on navigation properties"); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
		handler.HandleNavigationProperty(w, req, keyString, components.NavigationProperty, components.IsRef)
	} else if handler.IsStreamProperty(components.NavigationProperty) {
		if components.IsRef {
			if writeErr := response.WriteError(w, req, http.StatusBadRequest, "Invalid request",
				"$ref is not supported on stream properties"); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
		handler.HandleStreamProperty(w, req, keyString, components.NavigationProperty, components.IsValue)
	} else if handler.IsStructuralProperty(components.NavigationProperty) {
		if components.IsRef {
			if writeErr := response.WriteError(w, req, http.StatusBadRequest, "Invalid request",
				"$ref is not supported on structural properties"); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
		handler.HandleStructuralProperty(w, req, keyString, components.NavigationProperty, components.IsValue)
	} else if handler.IsComplexTypeProperty(components.NavigationProperty) {
		if components.IsRef {
			if writeErr := response.WriteError(w, req, http.StatusBadRequest, "Invalid request",
				"$ref is not supported on complex properties"); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
		handler.HandleComplexTypeProperty(w, req, keyString, propertySegments, components.IsValue)
	} else {
		if writeErr := response.WriteError(w, req, http.StatusNotFound, "Property not found",
			fmt.Sprintf("'%s' is not a valid property for %s", components.NavigationProperty, components.EntitySet)); writeErr != nil {
			r.logger.Error("Error writing error response", "error", writeErr)
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

	// Sort keys to ensure deterministic ordering
	keys := make([]string, 0, len(keyMap))
	for key := range keyMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		value := keyMap[key]
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

// splitLastDot splits name at its last '.' character, returning the portion
// before ("prefix") and after ("suffix") the dot. ok is false when name
// contains no dot, in which case suffix equals name unchanged.
func splitLastDot(name string) (prefix, suffix string, ok bool) {
	idx := strings.LastIndex(name, ".")
	if idx == -1 {
		return "", name, false
	}
	return name[:idx], name[idx+1:], true
}

// resolveBoundName resolves a possibly namespace- or alias-qualified action or
// function segment to its unqualified registered name.
//
// Per OData v4.0 Part 2 §4.5 ("Invoking Actions") and the equivalent function
// invocation rules, a bound (or unbound) action or function MUST be invocable
// using its namespace-qualified name (e.g. "ComplianceService.GetTotalPrice"),
// in addition to the short/unqualified name where that is unambiguous.
//
// If name contains no '.', it is not qualified and is returned unchanged. If
// it is qualified, the namespace portion must exactly match the service's
// configured schema namespace; otherwise resolution fails so that operations
// cannot be invoked under an arbitrary or incorrect namespace.
func (r *Router) resolveBoundName(name string) (string, bool) {
	ns, local, hasNamespace := splitLastDot(name)
	if !hasNamespace {
		return name, true
	}
	if local == "" {
		return "", false
	}
	serviceNamespace := r.getNamespace()
	if serviceNamespace == "" || ns != serviceNamespace {
		return "", false
	}
	return local, true
}

// isBoundOperationSegment reports whether name (which may be namespace-
// qualified) resolves to a registered action or function.
func (r *Router) isBoundOperationSegment(name string) bool {
	resolved, ok := r.resolveBoundName(name)
	return ok && r.isActionOrFunction(resolved)
}

// isUnresolvableQualifiedOperation reports whether name looks like an attempt
// to invoke a registered action or function using a namespace qualifier that
// does not match the service's schema namespace (e.g.
// "WrongNamespace.ApplyDiscount" when the real action is "ApplyDiscount").
// This lets callers distinguish "unknown namespace for a real operation"
// (which should 404) from an unrelated segment such as a genuine derived-type
// cast (e.g. "ComplianceService.SpecialProduct").
func (r *Router) isUnresolvableQualifiedOperation(name string) bool {
	_, local, hasNamespace := splitLastDot(name)
	if !hasNamespace {
		return false
	}
	if _, ok := r.resolveBoundName(name); ok {
		return false
	}
	return r.isActionOrFunction(local)
}

// getNavigationTargetEntitySet returns the target entity set name for a navigation property
func (r *Router) getNavigationTargetEntitySet(handler EntityHandler, navigationProperty string) string {
	if handler == nil {
		return ""
	}

	if target, ok := handler.NavigationTargetSet(navigationProperty); ok {
		return target
	}

	return ""
}

// resolveKeyAsSegment reinterprets the first property segment as an entity key,
// implementing the OData 4.01 key-as-segments URL convention.
// For example, /Products/1 → EntitySet="Products", EntityKey="1".
// If additional property segments remain after the key, they are preserved.
func resolveKeyAsSegment(components *response.ODataURLComponents) *response.ODataURLComponents {
	newComponents := *components
	newComponents.EntityKey = components.NavigationProperty
	newComponents.EntityKeyMap = make(map[string]string)

	if len(components.PropertySegments) > 1 {
		remainingSegments := components.PropertySegments[1:]
		newComponents.NavigationProperty = remainingSegments[0]
		newComponents.PropertySegments = remainingSegments
		newComponents.PropertyPath = strings.Join(remainingSegments, "/")
	} else {
		newComponents.NavigationProperty = ""
		newComponents.PropertySegments = nil
		newComponents.PropertyPath = ""
	}

	return &newComponents
}
