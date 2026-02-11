package auth

import (
	"context"
	"net/http"

	"github.com/nlstn/go-odata/internal/query"
)

// AuthContext contains authentication and request metadata for authorization decisions.
type AuthContext struct {
	Principal interface{}
	Roles     []string
	Claims    map[string]interface{}
	Scopes    []string
	Request   RequestMetadata
}

// RequestMetadata captures HTTP request details relevant to authorization.
type RequestMetadata struct {
	Method     string
	Path       string
	Headers    http.Header
	Query      map[string][]string
	RemoteAddr string
}

// ResourceDescriptor describes the resource being accessed.
type ResourceDescriptor struct {
	EntitySetName string
	EntityType    string
	KeyValues     map[string]interface{}
	PropertyPath  []string
	Entity        interface{}
}

// Operation defines the type of action being authorized.
type Operation int

// Operation values for authorization checks.
const (
	OperationRead Operation = iota
	OperationCreate
	OperationUpdate
	OperationDelete
	OperationQuery
	OperationMetadata
	OperationAction
	OperationFunction
)

// Decision represents the result of an authorization check.
type Decision struct {
	Allowed bool
	Reason  string
}

// Allow returns an allow decision.
func Allow() Decision {
	return Decision{Allowed: true}
}

// Deny returns a deny decision with an optional reason.
func Deny(reason string) Decision {
	return Decision{Allowed: false, Reason: reason}
}

// Policy defines the interface for authorization decisions.
type Policy interface {
	Authorize(ctx AuthContext, resource ResourceDescriptor, operation Operation) Decision
}

// QueryFilterProvider defines an optional extension point for providing additional
// query filters based on authorization policy.
type QueryFilterProvider interface {
	Policy
	QueryFilter(ctx AuthContext, resource ResourceDescriptor, operation Operation) (*query.FilterExpression, error)
}

// Context keys for standard auth data that can be stored in request context.
// Users can store auth data using these keys in PreRequestHook, and it will be
// automatically extracted by the authorization framework.
type contextKey string

const (
	// PrincipalContextKey is the context key for the authenticated principal/user identifier
	PrincipalContextKey contextKey = "odata_auth_principal"
	// RolesContextKey is the context key for user roles (should be []string)
	RolesContextKey contextKey = "odata_auth_roles"
	// ClaimsContextKey is the context key for additional claims (should be map[string]interface{})
	ClaimsContextKey contextKey = "odata_auth_claims"
	// ScopesContextKey is the context key for OAuth scopes (should be []string)
	ScopesContextKey contextKey = "odata_auth_scopes"
)

// ExtractFromContext extracts authentication data from the request context using
// the standard context keys. This allows PreRequestHook to populate auth data
// that will be automatically used by the authorization framework.
func ExtractFromContext(ctx context.Context) (principal interface{}, roles []string, claims map[string]interface{}, scopes []string) {
	if ctx == nil {
		return nil, nil, nil, nil
	}

	if p := ctx.Value(PrincipalContextKey); p != nil {
		principal = p
	}

	if r, ok := ctx.Value(RolesContextKey).([]string); ok {
		roles = r
	}

	if c, ok := ctx.Value(ClaimsContextKey).(map[string]interface{}); ok {
		claims = c
	}

	if s, ok := ctx.Value(ScopesContextKey).([]string); ok {
		scopes = s
	}

	return principal, roles, claims, scopes
}
