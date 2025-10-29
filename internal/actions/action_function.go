package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
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

	// Parse JSON body using RawMessage so we can convert into the expected types later
	var bodyParams map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&bodyParams); err != nil {
		return nil, fmt.Errorf("failed to parse request body: %w", err)
	}

	// Validate and convert parameters
	for _, paramDef := range paramDefs {
		rawValue, exists := bodyParams[paramDef.Name]
		if !exists {
			if paramDef.Required {
				return nil, fmt.Errorf("required parameter '%s' is missing", paramDef.Name)
			}
			continue
		}

		value, err := decodeJSONToType(rawValue, paramDef.Type)
		if err != nil {
			return nil, fmt.Errorf("parameter '%s' has invalid value: %w", paramDef.Name, err)
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
			if values, ok := query[paramDef.Name]; ok && len(values) > 0 {
				value = values[0]
				found = true
			}
		}

		if !found {
			if paramDef.Required {
				return nil, fmt.Errorf("required parameter '%s' is missing", paramDef.Name)
			}
			continue
		}

		converted, err := convertStringToType(value, paramDef.Type)
		if err != nil {
			return nil, fmt.Errorf("parameter '%s' has invalid value: %w", paramDef.Name, err)
		}

		if err := validateParameterType(paramDef.Name, converted, paramDef.Type); err != nil {
			return nil, err
		}

		params[paramDef.Name] = converted
	}

	return params, nil
}

// decodeJSONToType converts a JSON raw message into the expected Go type using reflection.
func decodeJSONToType(raw []byte, expectedType reflect.Type) (interface{}, error) {
	if expectedType == nil {
		var v interface{}
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		return v, nil
	}

	// Create a pointer to the expected type (or pointer to pointer for pointer types)
	target := reflect.New(expectedType)
	if err := json.Unmarshal(raw, target.Interface()); err != nil {
		return nil, err
	}

	return target.Elem().Interface(), nil
}

// convertStringToType converts a string representation of a parameter to the expected type.
func convertStringToType(raw string, expectedType reflect.Type) (interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		// Handle empty strings explicitly for string parameters
		baseType := expectedType
		if expectedType != nil && expectedType.Kind() == reflect.Ptr {
			baseType = expectedType.Elem()
		}
		if baseType != nil && baseType.Kind() == reflect.String {
			data, err := json.Marshal("")
			if err != nil {
				return nil, err
			}
			return decodeJSONToType(data, expectedType)
		}
	}

	if expectedType == nil {
		return trimmed, nil
	}

	baseType := expectedType
	if expectedType.Kind() == reflect.Ptr {
		baseType = expectedType.Elem()
	}

	lowerTrimmed := strings.ToLower(trimmed)
	if lowerTrimmed == "null" {
		return decodeJSONToType([]byte("null"), expectedType)
	}

	if baseType.Kind() == reflect.Struct || baseType.Kind() == reflect.Slice || baseType.Kind() == reflect.Map || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return decodeJSONToType([]byte(trimmed), expectedType)
	}

	if baseType.Kind() == reflect.String {
		str, err := parseStringLiteral(trimmed)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(str)
		if err != nil {
			return nil, err
		}
		return decodeJSONToType(data, expectedType)
	}

	switch baseType.Kind() {
	case reflect.Bool:
		switch lowerTrimmed {
		case "true", "false":
			return decodeJSONToType([]byte(lowerTrimmed), expectedType)
		default:
			return nil, fmt.Errorf("expected boolean value")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// Ensure the value is a valid numeric literal by attempting to parse first
		if baseType.Kind() == reflect.Float32 || baseType.Kind() == reflect.Float64 {
			if _, err := strconv.ParseFloat(trimmed, 64); err != nil {
				return nil, fmt.Errorf("expected numeric value")
			}
		} else if baseType.Kind() == reflect.Uint || baseType.Kind() == reflect.Uint8 || baseType.Kind() == reflect.Uint16 ||
			baseType.Kind() == reflect.Uint32 || baseType.Kind() == reflect.Uint64 {
			if _, err := strconv.ParseUint(trimmed, 10, 64); err != nil {
				return nil, fmt.Errorf("expected unsigned integer value")
			}
		} else {
			if _, err := strconv.ParseInt(trimmed, 10, 64); err != nil {
				return nil, fmt.Errorf("expected integer value")
			}
		}
		return decodeJSONToType([]byte(trimmed), expectedType)
	}

	// Fallback: treat as JSON string literal for other cases
	data, err := json.Marshal(trimmed)
	if err != nil {
		return nil, err
	}
	return decodeJSONToType(data, expectedType)
}

// parseStringLiteral handles OData-style quoted strings using single quotes as well as JSON strings.
func parseStringLiteral(value string) (string, error) {
	if len(value) >= 2 {
		if value[0] == '\'' && value[len(value)-1] == '\'' {
			inner := value[1 : len(value)-1]
			inner = strings.ReplaceAll(inner, "''", "'")
			return inner, nil
		}
		if value[0] == '"' && value[len(value)-1] == '"' {
			unquoted, err := strconv.Unquote(value)
			if err != nil {
				return "", err
			}
			return unquoted, nil
		}
	}
	return value, nil
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
		if unescaped, err := url.PathUnescape(paramValue); err == nil {
			paramValue = unescaped
		}

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
	braceDepth := 0
	bracketDepth := 0
	parenDepth := 0

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
		case (ch == '{' || ch == '[' || ch == '(') && !inQuote:
			switch ch {
			case '{':
				braceDepth++
			case '[':
				bracketDepth++
			case '(':
				parenDepth++
			}
			current.WriteRune(ch)
		case (ch == '}' || ch == ']' || ch == ')') && !inQuote:
			switch ch {
			case '}':
				if braceDepth > 0 {
					braceDepth--
				}
			case ']':
				if bracketDepth > 0 {
					bracketDepth--
				}
			case ')':
				if parenDepth > 0 {
					parenDepth--
				}
			}
			current.WriteRune(ch)
		case ch == ',' && !inQuote && braceDepth == 0 && bracketDepth == 0 && parenDepth == 0:
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

	actualValue := reflect.ValueOf(value)
	actualKind := actualValue.Kind()
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
		actualType := actualValue.Type()
		if actualType.AssignableTo(expectedType) {
			return nil
		}
		if expectedKind == reflect.Ptr && actualType.AssignableTo(expectedType.Elem()) {
			return nil
		}
		if actualKind == reflect.Ptr && actualValue.Elem().IsValid() && actualValue.Elem().Type().AssignableTo(expectedType) {
			return nil
		}
		return fmt.Errorf("parameter '%s' has invalid type", paramName)
	}

	return nil
}

// ActionSignaturesMatch checks if two action definitions have the same signature
// Two actions have the same signature if they have the same name, binding, entity set, and parameters
func ActionSignaturesMatch(a1, a2 *ActionDefinition) bool {
	if a1.Name != a2.Name {
		return false
	}
	if a1.IsBound != a2.IsBound {
		return false
	}
	if a1.EntitySet != a2.EntitySet {
		return false
	}
	return parametersMatch(a1.Parameters, a2.Parameters)
}

// FunctionSignaturesMatch checks if two function definitions have the same signature
// Two functions have the same signature if they have the same name, binding, entity set, and parameters
func FunctionSignaturesMatch(f1, f2 *FunctionDefinition) bool {
	if f1.Name != f2.Name {
		return false
	}
	if f1.IsBound != f2.IsBound {
		return false
	}
	if f1.EntitySet != f2.EntitySet {
		return false
	}
	return parametersMatch(f1.Parameters, f2.Parameters)
}

// parametersMatch checks if two parameter lists are identical
func parametersMatch(p1, p2 []ParameterDefinition) bool {
	if len(p1) != len(p2) {
		return false
	}

	// Create maps for comparison (order shouldn't matter for signature matching)
	params1 := make(map[string]reflect.Type)
	for _, p := range p1 {
		params1[p.Name] = p.Type
	}

	for _, p := range p2 {
		if t, exists := params1[p.Name]; !exists || t != p.Type {
			return false
		}
	}

	return true
}

// ResolveActionOverload resolves the appropriate action overload based on the request parameters
func ResolveActionOverload(r *http.Request, candidates []*ActionDefinition, isBound bool, entitySet string) (*ActionDefinition, map[string]interface{}, error) {
	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("no action candidates found")
	}

	// Filter by binding and entity set first
	var filtered []*ActionDefinition
	for _, candidate := range candidates {
		if candidate.IsBound == isBound {
			if !isBound || candidate.EntitySet == entitySet {
				filtered = append(filtered, candidate)
			}
		}
	}

	if len(filtered) == 0 {
		return nil, nil, fmt.Errorf("no matching action found for binding context")
	}

	// If only one candidate after filtering, try to parse with its parameters
	if len(filtered) == 1 {
		params, err := ParseActionParameters(r, filtered[0].Parameters)
		if err != nil {
			return nil, nil, err
		}
		return filtered[0], params, nil
	}

	// Multiple candidates - try to find the best match based on provided parameters
	// Parse the request body to get parameter names
	var bodyParams map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&bodyParams); err != nil {
		return nil, nil, fmt.Errorf("failed to parse request body: %w", err)
	}

	// Try to find an exact match based on parameter names
	for _, candidate := range filtered {
		if parameterNamesMatch(candidate.Parameters, bodyParams) {
			// Validate and convert parameters
			params := make(map[string]interface{})
			allMatch := true
			for _, paramDef := range candidate.Parameters {
				value, exists := bodyParams[paramDef.Name]
				if !exists {
					if paramDef.Required {
						allMatch = false
						break
					}
					continue
				}

				// Validate parameter type
				if err := validateParameterType(paramDef.Name, value, paramDef.Type); err != nil {
					allMatch = false
					break
				}

				params[paramDef.Name] = value
			}

			if allMatch {
				return candidate, params, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("no matching action overload found for provided parameters")
}

// ResolveFunctionOverload resolves the appropriate function overload based on the request parameters
func ResolveFunctionOverload(r *http.Request, candidates []*FunctionDefinition, isBound bool, entitySet string) (*FunctionDefinition, map[string]interface{}, error) {
	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("no function candidates found")
	}

	// Filter by binding and entity set first
	var filtered []*FunctionDefinition
	for _, candidate := range candidates {
		if candidate.IsBound == isBound {
			if !isBound || candidate.EntitySet == entitySet {
				filtered = append(filtered, candidate)
			}
		}
	}

	if len(filtered) == 0 {
		return nil, nil, fmt.Errorf("no matching function found for binding context")
	}

	// Parse parameters from the URL to get parameter names and values
	pathParams, err := parseFunctionPathParameters(r.URL.Path)
	if err != nil {
		return nil, nil, err
	}
	query := r.URL.Query()

	// Combine path and query parameters to get all provided parameter names
	providedParams := make(map[string]string)
	for k, v := range pathParams {
		providedParams[k] = v
	}
	for k := range query {
		if _, exists := providedParams[k]; !exists {
			providedParams[k] = query.Get(k)
		}
	}

	// If only one candidate after filtering and parameter count matches, use it
	if len(filtered) == 1 {
		params, err := ParseFunctionParameters(r, filtered[0].Parameters)
		if err != nil {
			return nil, nil, err
		}
		return filtered[0], params, nil
	}

	// Multiple candidates - try to find the best match based on provided parameters
	for _, candidate := range filtered {
		if functionParameterNamesMatch(candidate.Parameters, providedParams) {
			params, err := ParseFunctionParameters(r, candidate.Parameters)
			if err != nil {
				continue // Try next candidate if this one fails
			}
			return candidate, params, nil
		}
	}

	return nil, nil, fmt.Errorf("no matching function overload found for provided parameters")
}

// parameterNamesMatch checks if the provided parameters match the expected parameter definitions
func parameterNamesMatch(paramDefs []ParameterDefinition, providedParams map[string]interface{}) bool {
	// Check that all required parameters are provided
	for _, paramDef := range paramDefs {
		if paramDef.Required {
			if _, exists := providedParams[paramDef.Name]; !exists {
				return false
			}
		}
	}

	// Check that no extra parameters are provided
	definedParams := make(map[string]bool)
	for _, paramDef := range paramDefs {
		definedParams[paramDef.Name] = true
	}
	for paramName := range providedParams {
		if !definedParams[paramName] {
			return false
		}
	}

	return true
}

// functionParameterNamesMatch checks if the provided parameters match the expected parameter definitions for functions
func functionParameterNamesMatch(paramDefs []ParameterDefinition, providedParams map[string]string) bool {
	// Check that all required parameters are provided
	for _, paramDef := range paramDefs {
		if paramDef.Required {
			if _, exists := providedParams[paramDef.Name]; !exists {
				return false
			}
		}
	}

	// Check that no extra parameters are provided
	definedParams := make(map[string]bool)
	for _, paramDef := range paramDefs {
		definedParams[paramDef.Name] = true
	}
	for paramName := range providedParams {
		if !definedParams[paramName] {
			return false
		}
	}

	return true
}
