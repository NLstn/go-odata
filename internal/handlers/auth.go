package handlers

import (
	"log/slog"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
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

	if err := response.WriteError(w, r, statusCode, message, decision.Reason); err != nil {
		if logger == nil {
			logger = slog.Default()
		}
		logger.Error("Error writing authorization response", "error", err)
	}
	return false
}

func policyQueryFilter(r *http.Request, policy auth.Policy, resource auth.ResourceDescriptor, operation auth.Operation) (*query.FilterExpression, error) {
	if policy == nil {
		return nil, nil
	}
	filterProvider, ok := policy.(auth.QueryFilterProvider)
	if !ok {
		return nil, nil
	}
	return filterProvider.QueryFilter(buildAuthContext(r), resource, operation)
}

func applyPolicyFilter(r *http.Request, policy auth.Policy, resource auth.ResourceDescriptor, queryOptions *query.QueryOptions) error {
	if queryOptions == nil {
		return nil
	}
	return applyPolicyFilterToExpression(r, policy, resource, &queryOptions.Filter)
}

func applyPolicyFilterToExpression(r *http.Request, policy auth.Policy, resource auth.ResourceDescriptor, filter **query.FilterExpression) error {
	if filter == nil {
		return nil
	}
	policyFilter, err := policyQueryFilter(r, policy, resource, auth.OperationQuery)
	if err != nil {
		return err
	}
	*filter = query.MergeFilterExpressions(*filter, policyFilter)
	return nil
}

func applyPolicyFiltersToExpand(r *http.Request, policy auth.Policy, entityMetadata *metadata.EntityMetadata, expand []query.ExpandOption) error {
	if policy == nil || entityMetadata == nil || len(expand) == 0 {
		return nil
	}

	for i := range expand {
		expandOpt := &expand[i]
		targetMetadata, err := entityMetadata.ResolveNavigationTarget(expandOpt.NavigationProperty)
		if err != nil || targetMetadata == nil {
			continue
		}

		resource := buildEntityResourceDescriptor(targetMetadata, "", nil)
		if err := applyPolicyFilterToExpression(r, policy, resource, &expandOpt.Filter); err != nil {
			return err
		}

		if len(expandOpt.Expand) > 0 {
			if err := applyPolicyFiltersToExpand(r, policy, targetMetadata, expandOpt.Expand); err != nil {
				return err
			}
		}
	}

	return nil
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

func buildEntityResourceDescriptorWithEntity(entityMetadata *metadata.EntityMetadata, entityKey string, entity interface{}, propertyPath []string) auth.ResourceDescriptor {
	resource := buildEntityResourceDescriptor(entityMetadata, entityKey, propertyPath)
	resource.Entity = entity
	if resource.EntitySetName == "" || entityMetadata == nil || entity == nil {
		return resource
	}
	if keyValues := buildKeyValuesFromEntity(entityMetadata, entity); len(keyValues) > 0 {
		resource.KeyValues = keyValues
	}
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

func buildKeyValuesFromEntity(entityMetadata *metadata.EntityMetadata, entity interface{}) map[string]interface{} {
	if entityMetadata == nil || entity == nil {
		return nil
	}

	value := reflect.ValueOf(entity)
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}

	keyValues := make(map[string]interface{}, len(entityMetadata.KeyProperties))
	for _, keyProp := range entityMetadata.KeyProperties {
		field := value.FieldByName(keyProp.Name)
		if !field.IsValid() {
			continue
		}
		if field.Kind() == reflect.Ptr && field.IsNil() {
			keyValues[keyProp.JsonName] = nil
			continue
		}
		keyValues[keyProp.JsonName] = field.Interface()
	}

	if len(keyValues) == 0 {
		return nil
	}
	return keyValues
}
