package odata

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/response"
)

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
