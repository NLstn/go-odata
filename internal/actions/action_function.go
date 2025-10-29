package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
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

// ParseActionParameters parses action parameters from request body
func ParseActionParameters(r *http.Request, paramDefs []ParameterDefinition) (map[string]interface{}, error) {
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

		// Validate parameter type
		if err := validateParameterType(paramDef.Name, value, paramDef.Type); err != nil {
			return nil, err
		}

		params[paramDef.Name] = value
	}

	return params, nil
}

// ParseFunctionParameters parses function parameters from URL query string or path
func ParseFunctionParameters(r *http.Request, paramDefs []ParameterDefinition) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	if len(paramDefs) == 0 {
		return params, nil
	}

	// First, try to parse parameters from the URL path (OData function call syntax)
	// e.g., /FunctionName(param1=value1,param2=value2)
	pathParams, err := parseFunctionPathParameters(r.URL.Path)
	if err != nil {
		return nil, err
	}

	// Then, get parameters from query string (alternative syntax)
	// e.g., /FunctionName?param1=value1&param2=value2
	query := r.URL.Query()

	// Validate and convert parameters
	for _, paramDef := range paramDefs {
		var value string
		var found bool

		// Try path parameters first, then query string
		if pathValue, ok := pathParams[paramDef.Name]; ok {
			value = pathValue
			found = true
		} else {
			value = query.Get(paramDef.Name)
			found = value != ""
		}

		if !found {
			if paramDef.Required {
				return nil, fmt.Errorf("required parameter '%s' is missing", paramDef.Name)
			}
			continue
		}

		// Convert string value to appropriate type
		var converted interface{}
		switch paramDef.Type.Kind() {
		case reflect.String:
			// Remove quotes if present
			if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
				value = value[1 : len(value)-1]
			}
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

// parseFunctionPathParameters extracts function parameters from the URL path
// e.g., /FunctionName(param1=value1,param2=value2) -> map[param1:value1 param2:value2]
// For bound functions on entities, the path might be /EntitySet(key)/FunctionName(params)
// We need to find the LAST occurrence of parentheses (after the last /)
func parseFunctionPathParameters(path string) (map[string]string, error) {
	params := make(map[string]string)

	// Find the last path segment (after the last /)
	lastSlash := strings.LastIndex(path, "/")
	var lastSegment string
	if lastSlash != -1 {
		lastSegment = path[lastSlash+1:]
	} else {
		lastSegment = path
	}

	// Find the opening parenthesis in the last segment
	startIdx := strings.Index(lastSegment, "(")
	if startIdx == -1 {
		return params, nil // No parameters in path
	}

	// Find the closing parenthesis in the last segment
	endIdx := strings.LastIndex(lastSegment, ")")
	if endIdx == -1 || endIdx < startIdx {
		return nil, fmt.Errorf("malformed function call: missing closing parenthesis")
	}

	// Extract parameter string from the last segment
	paramStr := lastSegment[startIdx+1 : endIdx]
	if paramStr == "" {
		return params, nil // Empty parameters: FunctionName()
	}

	// Split by comma, but be careful of quoted values
	pairs, err := splitParameterPairs(paramStr)
	if err != nil {
		return nil, err
	}

	for _, pair := range pairs {
		// Split by '='
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid parameter pair: %s", pair)
		}

		paramName := strings.TrimSpace(parts[0])
		paramValue := strings.TrimSpace(parts[1])

		params[paramName] = paramValue
	}

	return params, nil
}

// splitParameterPairs splits parameter pairs by comma, respecting quoted values
func splitParameterPairs(input string) ([]string, error) {
	var pairs []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, ch := range input {
		switch {
		case (ch == '\'' || ch == '"') && !inQuote:
			inQuote = true
			quoteChar = ch
			current.WriteRune(ch)
		case ch == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
			current.WriteRune(ch)
		case ch == ',' && !inQuote:
			pairs = append(pairs, current.String())
			current.Reset()
		default:
			current.WriteRune(ch)
		}

		// Check for unclosed quote at end
		if i == len(input)-1 && inQuote {
			return nil, fmt.Errorf("unclosed quote in function parameters")
		}
	}

	// Add the last pair
	if current.Len() > 0 {
		pairs = append(pairs, current.String())
	}

	return pairs, nil
}

// validateParameterType validates that a parameter value matches the expected type
func validateParameterType(paramName string, value interface{}, expectedType reflect.Type) error {
	if value == nil {
		return nil // null values are allowed
	}

	// Get the actual type of the value
	actualValue := reflect.ValueOf(value)
	actualKind := actualValue.Kind()

	// Get the expected kind
	expectedKind := expectedType.Kind()

	// JSON unmarshaling converts numbers to float64
	// So we need to handle numeric type conversions
	switch expectedKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Accept float64 (from JSON) if it represents a whole number
		if actualKind == reflect.Float64 {
			floatVal := actualValue.Float()
			if floatVal != float64(int64(floatVal)) {
				return fmt.Errorf("parameter '%s' must be an integer", paramName)
			}
			return nil
		}
		if actualKind != reflect.Int && actualKind != reflect.Int8 && actualKind != reflect.Int16 &&
			actualKind != reflect.Int32 && actualKind != reflect.Int64 {
			return fmt.Errorf("parameter '%s' must be an integer", paramName)
		}

	case reflect.Float32, reflect.Float64:
		// Accept both int and float from JSON
		if actualKind != reflect.Float64 && actualKind != reflect.Float32 &&
			actualKind != reflect.Int && actualKind != reflect.Int8 && actualKind != reflect.Int16 &&
			actualKind != reflect.Int32 && actualKind != reflect.Int64 {
			return fmt.Errorf("parameter '%s' must be a number", paramName)
		}

	case reflect.String:
		if actualKind != reflect.String {
			return fmt.Errorf("parameter '%s' must be a string", paramName)
		}

	case reflect.Bool:
		if actualKind != reflect.Bool {
			return fmt.Errorf("parameter '%s' must be a boolean", paramName)
		}

	default:
		// For other types, try to check if they are assignable
		if !actualValue.Type().AssignableTo(expectedType) {
			return fmt.Errorf("parameter '%s' has invalid type", paramName)
		}
	}

	return nil
}
