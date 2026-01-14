package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/response"
)

// ActionFunctionHandler handles OData action and function invocations
type ActionFunctionHandler struct {
	// actionsGetter provides access to registered actions
	actionsGetter func() map[string]interface{}
	// functionsGetter provides access to registered functions
	functionsGetter func() map[string]interface{}
	// entityHandler provides access to entity operations for bound actions/functions
	entityHandler *EntityHandler
}

// NewActionFunctionHandler creates a new action/function handler
func NewActionFunctionHandler(actionsGetter, functionsGetter func() map[string]interface{}, entityHandler *EntityHandler) *ActionFunctionHandler {
	return &ActionFunctionHandler{
		actionsGetter:   actionsGetter,
		functionsGetter: functionsGetter,
		entityHandler:   entityHandler,
	}
}

// HandleActionOrFunction handles action or function invocation
func (h *ActionFunctionHandler) HandleActionOrFunction(w http.ResponseWriter, r *http.Request, name string, key string, isBound bool) {
	switch r.Method {
	case http.MethodPost:
		// Actions are invoked with POST
		h.handleAction(w, r, name, key, isBound)
	case http.MethodGet:
		// Functions are invoked with GET
		h.handleFunction(w, r, name, key, isBound)
	default:
		if err := response.WriteError(w, r, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not allowed for actions or functions", r.Method)); err != nil {
			slog.Default().Error("Error writing error response", "error", err)
		}
	}
}

// handleAction handles action invocation
func (h *ActionFunctionHandler) handleAction(w http.ResponseWriter, r *http.Request, name string, key string, isBound bool) {
	actions := h.actionsGetter()
	actionDef, exists := actions[name]
	if !exists {
		if err := response.WriteError(w, r, http.StatusNotFound, "Action not found",
			fmt.Sprintf("Action '%s' is not registered", name)); err != nil {
			slog.Default().Error("Error writing error response", "error", err)
		}
		return
	}

	// Write a simple success response for now with dynamic metadata level
	// In a real implementation, this would invoke the action handler
	metadataLevel := response.GetODataMetadataLevel(r)
	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	responseMap := map[string]interface{}{
		"@odata.context": "$metadata#Edm.String",
		"value":          fmt.Sprintf("Action '%s' invoked successfully (bound: %v, key: %s, def: %v)", name, isBound, key, actionDef != nil),
	}

	if err := json.NewEncoder(w).Encode(responseMap); err != nil {
		slog.Default().Error("Error encoding response", "error", err)
	}
}

// handleFunction handles function invocation
func (h *ActionFunctionHandler) handleFunction(w http.ResponseWriter, r *http.Request, name string, key string, isBound bool) {
	functions := h.functionsGetter()
	functionDef, exists := functions[name]
	if !exists {
		if err := response.WriteError(w, r, http.StatusNotFound, "Function not found",
			fmt.Sprintf("Function '%s' is not registered", name)); err != nil {
			slog.Default().Error("Error writing error response", "error", err)
		}
		return
	}

	// Write a simple success response for now with dynamic metadata level
	// In a real implementation, this would invoke the function handler
	metadataLevel := response.GetODataMetadataLevel(r)
	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	responseMap := map[string]interface{}{
		"@odata.context": "$metadata#Edm.String",
		"value":          fmt.Sprintf("Function '%s' invoked successfully (bound: %v, key: %s, def: %v)", name, isBound, key, functionDef != nil),
	}

	if err := json.NewEncoder(w).Encode(responseMap); err != nil {
		slog.Default().Error("Error encoding response", "error", err)
	}
}

// ParseActionFunctionURL parses an action or function URL
// Returns: actionOrFunctionName, entityKey, isBound, error
func ParseActionFunctionURL(path string) (string, string, bool, error) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "", "", false, fmt.Errorf("invalid URL")
	}

	// Check for unbound action/function: /ActionName or /FunctionName
	if len(parts) == 1 {
		return parts[0], "", false, nil
	}

	// Check for bound action/function: /EntitySet(key)/ActionName or /EntitySet(key)/FunctionName
	if len(parts) == 2 {
		// Extract entity set and key from first part
		firstPart := parts[0]
		if strings.Contains(firstPart, "(") && strings.Contains(firstPart, ")") {
			// Has key, e.g., Products(1)
			openParen := strings.Index(firstPart, "(")
			closeParen := strings.Index(firstPart, ")")
			key := firstPart[openParen+1 : closeParen]
			actionOrFunction := parts[1]
			return actionOrFunction, key, true, nil
		}
	}

	return "", "", false, fmt.Errorf("invalid action/function URL format")
}
