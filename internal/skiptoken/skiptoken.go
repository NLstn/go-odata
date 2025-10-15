package skiptoken

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// SkipToken represents the state needed to resume a query
type SkipToken struct {
	// KeyValues holds the last entity's key values (for composite keys)
	KeyValues map[string]interface{} `json:"k"`
	// OrderByValues holds the last entity's orderby values (if orderby is used)
	OrderByValues map[string]interface{} `json:"o,omitempty"`
}

// Encode encodes a skip token into a base64-encoded JSON string
func Encode(token *SkipToken) (string, error) {
	if token == nil {
		return "", fmt.Errorf("token cannot be nil")
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token: %w", err)
	}

	// Encode to base64
	encoded := base64.URLEncoding.EncodeToString(jsonBytes)
	return encoded, nil
}

// Decode decodes a base64-encoded JSON string into a SkipToken
func Decode(encoded string) (*SkipToken, error) {
	if encoded == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Decode from base64
	jsonBytes, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	// Unmarshal from JSON
	var token SkipToken
	if err := json.Unmarshal(jsonBytes, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// ExtractFromEntity extracts key and orderby values from an entity to create a skip token
func ExtractFromEntity(entity interface{}, keyProperties []string, orderByProperties []string) (*SkipToken, error) {
	token := &SkipToken{
		KeyValues: make(map[string]interface{}),
	}

	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Handle map-based entities (from $select or $apply)
	if entityValue.Kind() == reflect.Map {
		return extractFromMap(entityValue, keyProperties, orderByProperties)
	}

	// Handle struct-based entities
	entityType := entityValue.Type()

	// Extract key values
	for _, keyProp := range keyProperties {
		field, err := findFieldByJSONTag(entityType, entityValue, keyProp)
		if err != nil {
			return nil, fmt.Errorf("key property '%s' not found: %w", keyProp, err)
		}
		token.KeyValues[keyProp] = field.Interface()
	}

	// Extract orderby values if present
	if len(orderByProperties) > 0 {
		token.OrderByValues = make(map[string]interface{})
		for _, orderByProp := range orderByProperties {
			// Remove " desc" or " asc" suffix if present
			propName := strings.TrimSuffix(strings.TrimSuffix(orderByProp, " desc"), " asc")
			propName = strings.TrimSpace(propName)

			field, err := findFieldByJSONTag(entityType, entityValue, propName)
			if err != nil {
				return nil, fmt.Errorf("orderby property '%s' not found: %w", propName, err)
			}
			token.OrderByValues[propName] = field.Interface()
		}
	}

	return token, nil
}

// extractFromMap extracts values from a map-based entity
func extractFromMap(entityValue reflect.Value, keyProperties []string, orderByProperties []string) (*SkipToken, error) {
	token := &SkipToken{
		KeyValues: make(map[string]interface{}),
	}

	mapValue, ok := entityValue.Interface().(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("value is not a map[string]interface{}")
	}

	// Extract key values
	for _, keyProp := range keyProperties {
		val, ok := mapValue[keyProp]
		if !ok {
			return nil, fmt.Errorf("key property '%s' not found in map", keyProp)
		}
		token.KeyValues[keyProp] = val
	}

	// Extract orderby values if present
	if len(orderByProperties) > 0 {
		token.OrderByValues = make(map[string]interface{})
		for _, orderByProp := range orderByProperties {
			// Remove " desc" or " asc" suffix if present
			propName := strings.TrimSuffix(strings.TrimSuffix(orderByProp, " desc"), " asc")
			propName = strings.TrimSpace(propName)

			val, ok := mapValue[propName]
			if !ok {
				return nil, fmt.Errorf("orderby property '%s' not found in map", propName)
			}
			token.OrderByValues[propName] = val
		}
	}

	return token, nil
}

// findFieldByJSONTag finds a struct field by its JSON tag name
func findFieldByJSONTag(entityType reflect.Type, entityValue reflect.Value, jsonTag string) (reflect.Value, error) {
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		tag := field.Tag.Get("json")
		// Remove ",omitempty" and other options
		tagName := strings.Split(tag, ",")[0]
		if tagName == jsonTag {
			return entityValue.Field(i), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("field with json tag '%s' not found", jsonTag)
}
