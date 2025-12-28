package query

import (
	"strings"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
)

func filterSelectedProperties(selected []string, entityMetadata *metadata.EntityMetadata, policy auth.Policy, authCtx auth.AuthContext, computedAliases map[string]bool) []string {
	if policy == nil || len(selected) == 0 {
		return selected
	}

	filtered := make([]string, 0, len(selected))
	for _, propName := range selected {
		trimmed := strings.TrimSpace(propName)
		if trimmed == "" {
			continue
		}
		if computedAliases != nil && computedAliases[trimmed] {
			filtered = append(filtered, propName)
			continue
		}

		propertyPath := propertyPathSegments(entityMetadata, trimmed)
		if authorizePropertyPath(policy, authCtx, entityMetadata, propertyPath) {
			filtered = append(filtered, propName)
		}
	}

	return filtered
}

func filterExpandOptions(expand []ExpandOption, entityMetadata *metadata.EntityMetadata, policy auth.Policy, authCtx auth.AuthContext, prefix []string) []ExpandOption {
	if policy == nil || len(expand) == 0 {
		return expand
	}

	filtered := make([]ExpandOption, 0, len(expand))
	for _, expandOpt := range expand {
		navSegments := append([]string{}, prefix...)
		navSegments = append(navSegments, propertyPathSegments(entityMetadata, expandOpt.NavigationProperty)...)

		if !authorizePropertyPath(policy, authCtx, entityMetadata, navSegments) {
			continue
		}

		if expandOpt.SelectSpecified {
			expandOpt.Select = filterSelectedPropertiesWithPrefix(expandOpt.Select, entityMetadata, policy, authCtx, navSegments)
		}
		expandOpt.Expand = filterExpandOptions(expandOpt.Expand, entityMetadata, policy, authCtx, navSegments)
		filtered = append(filtered, expandOpt)
	}

	return filtered
}

func filterSelectedPropertiesWithPrefix(selected []string, entityMetadata *metadata.EntityMetadata, policy auth.Policy, authCtx auth.AuthContext, prefix []string) []string {
	if policy == nil || len(selected) == 0 {
		return selected
	}

	filtered := make([]string, 0, len(selected))
	for _, propName := range selected {
		trimmed := strings.TrimSpace(propName)
		if trimmed == "" {
			continue
		}

		path := append([]string{}, prefix...)
		path = append(path, propertyPathSegments(entityMetadata, trimmed)...)
		if authorizePropertyPath(policy, authCtx, entityMetadata, path) {
			filtered = append(filtered, propName)
		}
	}

	return filtered
}

func authorizePropertyPath(policy auth.Policy, authCtx auth.AuthContext, entityMetadata *metadata.EntityMetadata, propertyPath []string) bool {
	if policy == nil || len(propertyPath) == 0 {
		return true
	}

	resource := auth.ResourceDescriptor{
		PropertyPath: append([]string(nil), propertyPath...),
	}
	if entityMetadata != nil {
		resource.EntitySetName = entityMetadata.EntitySetName
		resource.EntityType = entityMetadata.EntityName
	}

	return policy.Authorize(authCtx, resource, auth.OperationRead).Allowed
}

func propertyPathSegments(entityMetadata *metadata.EntityMetadata, path string) []string {
	if entityMetadata != nil {
		if segments := entityMetadata.SplitPropertyPath(path); len(segments) > 0 {
			return segments
		}
	}

	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		segment := strings.TrimSpace(part)
		if segment == "" {
			continue
		}
		segments = append(segments, segment)
	}
	return segments
}
