package operations

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

// Handler orchestrates the execution of OData actions and functions.
type Handler struct {
	actions   map[string][]*actions.ActionDefinition
	functions map[string][]*actions.FunctionDefinition
	handlers  map[string]*handlers.EntityHandler
	entities  map[string]*metadata.EntityMetadata
	namespace string
	logger    Logger
}

// Logger captures the subset of slog.Logger functionality required by the handler.
type Logger interface {
	Error(msg string, args ...any)
}

// NewHandler creates a new Handler with the provided dependencies.
func NewHandler(
	actions map[string][]*actions.ActionDefinition,
	functions map[string][]*actions.FunctionDefinition,
	handlers map[string]*handlers.EntityHandler,
	entities map[string]*metadata.EntityMetadata,
	namespace string,
	logger Logger,
) *Handler {
	return &Handler{
		actions:   actions,
		functions: functions,
		handlers:  handlers,
		entities:  entities,
		namespace: namespace,
		logger:    logger,
	}
}

// SetLogger updates the logger used for error reporting.
func (h *Handler) SetLogger(logger Logger) {
	h.logger = logger
}

// SetNamespace updates the namespace used for function context fragments.
func (h *Handler) SetNamespace(namespace string) {
	h.namespace = namespace
}

type invocationError struct {
	status  int
	message string
	detail  string
}

type overloadResolver[T any] func(*http.Request, []T, bool, string) (T, map[string]interface{}, error)

// HandleActionOrFunction executes the requested action or function.
func (h *Handler) HandleActionOrFunction(w http.ResponseWriter, r *http.Request, name, key string, isBound bool, entitySet string) {
	switch r.Method {
	case http.MethodPost:
		actionDef, params, invErr := resolveInvocation(r, name, "Action", h.actions, actions.ResolveActionOverload, isBound, entitySet)
		if invErr != nil {
			h.writeError(w, invErr)
			return
		}

		var ctx interface{}
		if isBound {
			var ctxErr *invocationError
			ctx, ctxErr = h.loadBoundContext(entitySet, key)
			if ctxErr != nil {
				h.writeError(w, ctxErr)
				return
			}
		}

		if err := actionDef.Handler(w, r, ctx, params); err != nil {
			if writeErr := response.WriteError(w, r, http.StatusInternalServerError, "Action failed", err.Error()); writeErr != nil {
				h.logError("Error writing error response", writeErr)
			}
			return
		}
	case http.MethodGet:
		functionDef, params, invErr := resolveInvocation(r, name, "Function", h.functions, actions.ResolveFunctionOverload, isBound, entitySet)
		if invErr != nil {
			h.writeError(w, invErr)
			return
		}

		var ctx interface{}
		if isBound {
			var ctxErr *invocationError
			ctx, ctxErr = h.loadBoundContext(entitySet, key)
			if ctxErr != nil {
				h.writeError(w, ctxErr)
				return
			}
		}

		result, err := functionDef.Handler(w, r, ctx, params)
		if err != nil {
			if writeErr := response.WriteError(w, r, http.StatusInternalServerError, "Function failed", err.Error()); writeErr != nil {
				h.logError("Error writing error response", writeErr)
			}
			return
		}

		if !response.IsAcceptableFormat(r) {
			if writeErr := response.WriteError(w, r, http.StatusNotAcceptable, "Not Acceptable",
				"The requested format is not supported. Only application/json is supported for data responses."); writeErr != nil {
				h.logError("Error writing error response", writeErr)
			}
			return
		}

		metadataLevel := response.GetODataMetadataLevel(r)
		w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))

		contextFragment := metadata.FunctionContextFragment(functionDef.ReturnType, h.entities, h.namespace)
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
			h.logError("Error encoding response", err)
		}
	default:
		if writeErr := response.WriteError(w, r, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not allowed for actions or functions", r.Method)); writeErr != nil {
			h.logError("Error writing error response", writeErr)
		}
	}
}

func (h *Handler) writeError(w http.ResponseWriter, invErr *invocationError) {
	if invErr == nil {
		return
	}
	if writeErr := response.WriteError(w, r, invErr.status, invErr.message, invErr.detail); writeErr != nil {
		h.logError("Error writing error response", writeErr)
	}
}

func (h *Handler) logError(msg string, err error) {
	if h.logger == nil {
		return
	}
	h.logger.Error(msg, "error", err)
}

func (h *Handler) loadBoundContext(entitySet, key string) (interface{}, *invocationError) {
	if key == "" {
		return nil, nil
	}
	handler := h.handlers[entitySet]
	if handler == nil {
		return nil, &invocationError{
			status:  http.StatusNotFound,
			message: "Entity not found",
			detail:  fmt.Sprintf("Entity set '%s' is not registered", entitySet),
		}
	}

	entity, err := handler.FetchEntity(key)
	if err != nil {
		if handlers.IsNotFoundError(err) {
			return nil, &invocationError{
				status:  http.StatusNotFound,
				message: "Entity not found",
				detail:  fmt.Sprintf("Entity with key '%s' not found", key),
			}
		}

		return nil, &invocationError{
			status:  http.StatusInternalServerError,
			message: "Database error",
			detail:  err.Error(),
		}
	}

	return entity, nil
}

func resolveInvocation[T any](
	r *http.Request,
	name string,
	ctxType string,
	definitions map[string][]T,
	resolver overloadResolver[T],
	isBound bool,
	entitySet string,
) (T, map[string]interface{}, *invocationError) {
	var zero T

	defs, exists := definitions[name]
	if !exists {
		return zero, nil, &invocationError{
			status:  http.StatusNotFound,
			message: fmt.Sprintf("%s not found", ctxType),
			detail:  fmt.Sprintf("%s '%s' is not registered", ctxType, name),
		}
	}

	def, params, err := resolver(r, defs, isBound, entitySet)
	if err != nil {
		errStr := err.Error()
		lowerCtx := strings.ToLower(ctxType)
		errorMsg := fmt.Sprintf("Invalid %s invocation", lowerCtx)
		if containsAny(errStr, "parameter", "required", "type", "missing") {
			errorMsg = "Invalid parameters"
		}

		return zero, nil, &invocationError{
			status:  http.StatusBadRequest,
			message: errorMsg,
			detail:  errStr,
		}
	}

	return def, params, nil
}

// containsAny checks if a string contains any of the given substrings.
func containsAny(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
