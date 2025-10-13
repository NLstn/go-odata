package odata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

// ParameterDefinition defines a parameter for an action or function
type ParameterDefinition struct {
	Name     string
	Type     reflect.Type
	Required bool
}

// ActionDefinition defines an OData action
type ActionDefinition struct {
	Name       string
	IsBound    bool
	EntitySet  string // For bound actions, the entity set it's bound to
	Handler    ActionHandler
	Parameters []ParameterDefinition
	ReturnType reflect.Type // nil if no return value
}

// FunctionDefinition defines an OData function
type FunctionDefinition struct {
	Name       string
	IsBound    bool
	EntitySet  string // For bound functions, the entity set it's bound to
	Handler    FunctionHandler
	Parameters []ParameterDefinition
	ReturnType reflect.Type
}

// ActionHandler is the function signature for action handlers
// ctx contains the binding parameter (entity) for bound actions, nil for unbound
// params contains the action parameters
type ActionHandler func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error

// FunctionHandler is the function signature for function handlers
// ctx contains the binding parameter (entity) for bound functions, nil for unbound
// params contains the function parameters
type FunctionHandler func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error)

// RegisterAction registers an action with the OData service
func (s *Service) RegisterAction(action ActionDefinition) error {
	if action.Name == "" {
		return fmt.Errorf("action name cannot be empty")
	}
	if action.Handler == nil {
		return fmt.Errorf("action handler cannot be nil")
	}
	if action.IsBound && action.EntitySet == "" {
		return fmt.Errorf("bound action must specify entity set")
	}
	if action.IsBound {
		// Verify entity set exists
		if _, exists := s.entities[action.EntitySet]; !exists {
			return fmt.Errorf("entity set '%s' not found", action.EntitySet)
		}
	}

	s.actions[action.Name] = &action
	fmt.Printf("Registered action: %s (Bound: %v, EntitySet: %s)\n", action.Name, action.IsBound, action.EntitySet)
	return nil
}

// RegisterFunction registers a function with the OData service
func (s *Service) RegisterFunction(function FunctionDefinition) error {
	if function.Name == "" {
		return fmt.Errorf("function name cannot be empty")
	}
	if function.Handler == nil {
		return fmt.Errorf("function handler cannot be nil")
	}
	if function.ReturnType == nil {
		return fmt.Errorf("function must have a return type")
	}
	if function.IsBound && function.EntitySet == "" {
		return fmt.Errorf("bound function must specify entity set")
	}
	if function.IsBound {
		// Verify entity set exists
		if _, exists := s.entities[function.EntitySet]; !exists {
			return fmt.Errorf("entity set '%s' not found", function.EntitySet)
		}
	}

	s.functions[function.Name] = &function
	fmt.Printf("Registered function: %s (Bound: %v, EntitySet: %s)\n", function.Name, function.IsBound, function.EntitySet)
	return nil
}

// parseActionParameters parses action parameters from request body
func parseActionParameters(r *http.Request, paramDefs []ParameterDefinition) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	if len(paramDefs) == 0 {
		return params, nil
	}

	// Parse JSON body
	var bodyParams map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&bodyParams); err != nil {
		return nil, fmt.Errorf("failed to parse request body: %w", err)
	}

	// Validate and convert parameters
	for _, paramDef := range paramDefs {
		value, exists := bodyParams[paramDef.Name]
		if !exists {
			if paramDef.Required {
				return nil, fmt.Errorf("required parameter '%s' is missing", paramDef.Name)
			}
			continue
		}
		params[paramDef.Name] = value
	}

	return params, nil
}

// parseFunctionParameters parses function parameters from URL query string
func parseFunctionParameters(r *http.Request, paramDefs []ParameterDefinition) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	if len(paramDefs) == 0 {
		return params, nil
	}

	query := r.URL.Query()

	// Validate and convert parameters
	for _, paramDef := range paramDefs {
		value := query.Get(paramDef.Name)
		if value == "" {
			if paramDef.Required {
				return nil, fmt.Errorf("required parameter '%s' is missing", paramDef.Name)
			}
			continue
		}

		// Convert string value to appropriate type
		var converted interface{}
		switch paramDef.Type.Kind() {
		case reflect.String:
			converted = value
		case reflect.Int, reflect.Int32, reflect.Int64:
			var intVal int64
			if _, err := fmt.Sscanf(value, "%d", &intVal); err != nil {
				return nil, fmt.Errorf("parameter '%s' must be an integer", paramDef.Name)
			}
			converted = intVal
		case reflect.Float32, reflect.Float64:
			var floatVal float64
			if _, err := fmt.Sscanf(value, "%f", &floatVal); err != nil {
				return nil, fmt.Errorf("parameter '%s' must be a number", paramDef.Name)
			}
			converted = floatVal
		case reflect.Bool:
			switch value {
			case "true":
				converted = true
			case "false":
				converted = false
			default:
				return nil, fmt.Errorf("parameter '%s' must be a boolean", paramDef.Name)
			}
		default:
			converted = value
		}

		params[paramDef.Name] = converted
	}

	return params, nil
}
