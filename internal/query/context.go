package query

import (
	"context"
	"net/url"
)

// parsedQueryContextKey is the unexported context key for the pre-parsed query values.
type parsedQueryContextKey struct{}

// WithParsedQuery stores a pre-parsed url.Values map in the context so that the
// query string does not have to be parsed again by downstream handlers and response
// writers.
func WithParsedQuery(ctx context.Context, values url.Values) context.Context {
	return context.WithValue(ctx, parsedQueryContextKey{}, values)
}

// GetParsedQuery retrieves the pre-parsed url.Values from the context. If the
// values were not stored in the context (e.g. in unit tests that create bare
// http.Requests) it falls back to parsing r.URL.RawQuery on the request provided
// as the fallback raw query string.
func GetParsedQuery(ctx context.Context) url.Values {
	if values, ok := ctx.Value(parsedQueryContextKey{}).(url.Values); ok {
		return values
	}
	return nil
}

// GetOrParseParsedQuery retrieves the pre-parsed url.Values from the context,
// falling back to CachedParseRawQuery if no cached value is present.
// CachedParseRawQuery checks a process-level bounded cache before calling
// ParseRawQuery, reducing redundant allocations for repeated query strings.
func GetOrParseParsedQuery(ctx context.Context, rawQuery string) url.Values {
	if values, ok := ctx.Value(parsedQueryContextKey{}).(url.Values); ok {
		return values
	}
	return CachedParseRawQuery(rawQuery)
}
