package odata

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

type invocationError struct {
	status  int
	message string
	detail  string
}

type overloadResolver[T any] func(*http.Request, []T, bool, string) (T, map[string]interface{}, error)

func resolveInvocation[T any](r *http.Request, name string, ctxType string, definitions map[string][]T,
	resolver overloadResolver[T], isBound bool, entitySet string) (T, map[string]interface{}, *invocationError) {
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

func loadBoundContext(handler *handlers.EntityHandler, key string) (interface{}, *invocationError) {
	if handler == nil || key == "" {
		return nil, nil
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

// handleActionOrFunction handles action or function invocation
func (s *Service) handleActionOrFunction(w http.ResponseWriter, r *http.Request, name string, key string, isBound bool, entitySet string) {
	// Check if it's an action (POST) or function (GET)
	switch r.Method {
	case http.MethodPost:
		actionDef, params, invErr := resolveInvocation(r, name, "Action", s.actions, actions.ResolveActionOverload, isBound, entitySet)
		if invErr != nil {
			if writeErr := response.WriteError(w, invErr.status, invErr.message, invErr.detail); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		var ctx interface{}
		if isBound {
			var ctxErr *invocationError
			ctx, ctxErr = loadBoundContext(s.handlers[entitySet], key)
			if ctxErr != nil {
				if writeErr := response.WriteError(w, ctxErr.status, ctxErr.message, ctxErr.detail); writeErr != nil {
					fmt.Printf("Error writing error response: %v\n", writeErr)
				}
				return
			}
		}

		if err := actionDef.Handler(w, r, ctx, params); err != nil {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, "Action failed", err.Error()); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

	case http.MethodGet:
		functionDef, params, invErr := resolveInvocation(r, name, "Function", s.functions, actions.ResolveFunctionOverload, isBound, entitySet)
		if invErr != nil {
			if writeErr := response.WriteError(w, invErr.status, invErr.message, invErr.detail); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		var ctx interface{}
		if isBound {
			var ctxErr *invocationError
			ctx, ctxErr = loadBoundContext(s.handlers[entitySet], key)
			if ctxErr != nil {
				if writeErr := response.WriteError(w, ctxErr.status, ctxErr.message, ctxErr.detail); writeErr != nil {
					fmt.Printf("Error writing error response: %v\n", writeErr)
				}
				return
			}
		}

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

		contextFragment := metadata.FunctionContextFragment(functionDef.ReturnType, s.entities)
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

// containsAny checks if a string contains any of the given substrings
func containsAny(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
