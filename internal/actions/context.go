package actions

import (
	"context"
	"net/http"

	"github.com/nlstn/go-odata/internal/query"
)

// queryOptionsContextKey is the unexported key type used to store parsed query
// options in the HTTP request context. Using a private struct type prevents
// collisions with keys defined in other packages.
type queryOptionsContextKey struct{}

// WithQueryOptions returns a shallow copy of r whose context carries the
// provided query options. The stored options can later be retrieved by
// QueryOptionsFromRequest.
func WithQueryOptions(r *http.Request, opts *query.QueryOptions) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), queryOptionsContextKey{}, opts))
}

// QueryOptionsFromRequest retrieves the parsed OData query options that were
// injected into the request context before the action or function handler was
// called. Returns nil when no options are present.
func QueryOptionsFromRequest(r *http.Request) *query.QueryOptions {
	if r == nil {
		return nil
	}
	opts, ok := r.Context().Value(queryOptionsContextKey{}).(*query.QueryOptions)
	if !ok {
		return nil
	}
	return opts
}
