package auth

import (
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
