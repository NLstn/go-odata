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

// handleActionOrFunction handles action or function invocation
func (s *Service) handleActionOrFunction(w http.ResponseWriter, r *http.Request, name string, key string, isBound bool, entitySet string) {
	// Check if it's an action (POST) or function (GET)
	switch r.Method {
	case http.MethodPost:
		// Handle action
		actionDefs, exists := s.actions[name]
		if !exists {
			if writeErr := response.WriteError(w, http.StatusNotFound, "Action not found",
				fmt.Sprintf("Action '%s' is not registered", name)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Resolve the appropriate action overload
		actionDef, params, err := actions.ResolveActionOverload(r, actionDefs, isBound, entitySet)
		if err != nil {
			// Determine error message based on error content
			errorMsg := "Invalid action invocation"
			errStr := err.Error()
			if containsAny(errStr, "parameter", "required", "type", "missing") {
				errorMsg = "Invalid parameters"
			}
			if writeErr := response.WriteError(w, http.StatusBadRequest, errorMsg, errStr); writeErr != nil {
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
		functionDefs, exists := s.functions[name]
		if !exists {
			if writeErr := response.WriteError(w, http.StatusNotFound, "Function not found",
				fmt.Sprintf("Function '%s' is not registered", name)); writeErr != nil {
				fmt.Printf("Error writing error response: %v\n", writeErr)
			}
			return
		}

		// Resolve the appropriate function overload
		functionDef, params, err := actions.ResolveFunctionOverload(r, functionDefs, isBound, entitySet)
		if err != nil {
			// Determine error message based on error content
			errorMsg := "Invalid function invocation"
			errStr := err.Error()
			if containsAny(errStr, "parameter", "required", "type", "missing") {
				errorMsg = "Invalid parameters"
			}
			if writeErr := response.WriteError(w, http.StatusBadRequest, errorMsg, errStr); writeErr != nil {
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
