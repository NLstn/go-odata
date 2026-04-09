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

// navigationBindingContextKey is the unexported key type used to store
// navigation binding context in the HTTP request context.
type navigationBindingContextKey struct{}

// NavigationBindingContext carries parent-resource information when a bound
// action or function is invoked through navigation composition.
//
// For a request such as /Categories(1)/Products/GetAveragePrice() the fields
// are populated as follows:
//
//   - ParentEntitySet   = "Categories"
//   - ParentKey         = "1"
//   - NavigationProperty = "Products"
type NavigationBindingContext struct {
	// ParentEntitySet is the entity set of the parent resource in the
	// navigation path (e.g. "Categories").
	ParentEntitySet string
	// ParentKey is the key value of the parent resource
	// (e.g. "1" for /Categories(1)/...).
	ParentKey string
	// NavigationProperty is the name of the navigation property that was
	// traversed to reach the bound target (e.g. "Products" or
	// "FeaturedProducts").
	NavigationProperty string
}

// WithNavigationBindingContext returns a shallow copy of r whose context
// carries the provided navigation binding context. The stored value can later
// be retrieved by NavigationBindingContextFromRequest. If navCtx is nil the
// original request is returned unchanged.
func WithNavigationBindingContext(r *http.Request, navCtx *NavigationBindingContext) *http.Request {
	if navCtx == nil {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), navigationBindingContextKey{}, navCtx))
}

// NavigationBindingContextFromRequest retrieves the navigation binding context
// that was injected into the request context when a bound action or function is
// invoked through navigation composition. Returns nil when the request was not
// dispatched via navigation composition.
func NavigationBindingContextFromRequest(r *http.Request) *NavigationBindingContext {
	if r == nil {
		return nil
	}
	navCtx, ok := r.Context().Value(navigationBindingContextKey{}).(*NavigationBindingContext)
	if !ok {
		return nil
	}
	return navCtx
}
