package handlers

import (
	"log/slog"
	"net/http"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

func buildAuthContext(r *http.Request) auth.AuthContext {
	if r == nil {
		return auth.AuthContext{}
	}
	return auth.AuthContext{
		Request: auth.RequestMetadata{
			Method:     r.Method,
			Path:       r.URL.Path,
			Headers:    r.Header.Clone(),
			Query:      r.URL.Query(),
			RemoteAddr: r.RemoteAddr,
		},
	}
}

func authorizeRequest(w http.ResponseWriter, r *http.Request, policy auth.Policy, resource auth.ResourceDescriptor, operation auth.Operation, logger *slog.Logger) bool {
	if policy == nil {
		return true
	}

	decision := policy.Authorize(buildAuthContext(r), resource, operation)
	if decision.Allowed {
		return true
	}

	statusCode := http.StatusForbidden
	message := "Forbidden"
	if r != nil && r.Header.Get("Authorization") == "" {
		statusCode = http.StatusUnauthorized
		message = "Unauthorized"
	}

	if err := response.WriteError(w, statusCode, message, decision.Reason); err != nil {
		if logger == nil {
			logger = slog.Default()
		}
		logger.Error("Error writing authorization response", "error", err)
	}
	return false
}

func buildEntityResourceDescriptor(entityMetadata *metadata.EntityMetadata, entityKey string, propertyPath []string) auth.ResourceDescriptor {
	resource := auth.ResourceDescriptor{
		PropertyPath: append([]string(nil), propertyPath...),
	}
	if entityMetadata == nil {
		return resource
	}
	resource.EntitySetName = entityMetadata.EntitySetName
	resource.EntityType = entityMetadata.EntityName
	if entityKey == "" {
		return resource
	}
	resource.KeyValues = buildKeyValues(entityMetadata, entityKey)
	return resource
}

func buildKeyValues(entityMetadata *metadata.EntityMetadata, entityKey string) map[string]interface{} {
	if entityMetadata == nil || entityKey == "" {
		return nil
	}

	components := &response.ODataURLComponents{
		EntityKeyMap: make(map[string]string),
	}
	if err := parseCompositeKey(entityKey, components); err != nil {
		if len(entityMetadata.KeyProperties) != 1 {
			return nil
		}
		keyProp := entityMetadata.KeyProperties[0]
		return map[string]interface{}{keyProp.JsonName: entityKey}
	}

	if len(components.EntityKeyMap) == 0 {
		return nil
	}

	keyValues := make(map[string]interface{}, len(components.EntityKeyMap))
	for _, keyProp := range entityMetadata.KeyProperties {
		if value, ok := components.EntityKeyMap[keyProp.JsonName]; ok {
			keyValues[keyProp.JsonName] = value
			continue
		}
		if value, ok := components.EntityKeyMap[keyProp.Name]; ok {
			keyValues[keyProp.JsonName] = value
		}
	}
	if len(keyValues) > 0 {
		return keyValues
	}
	for key, value := range components.EntityKeyMap {
		keyValues[key] = value
	}
	return keyValues
}
