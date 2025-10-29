package handlers

import (
	"context"
)

// Context keys for request-scoped values
type contextKey string

const (
	typeCastKey contextKey = "odata_type_cast"
)

// WithTypeCast adds a type cast filter to the request context
func WithTypeCast(ctx context.Context, typeName string) context.Context {
	return context.WithValue(ctx, typeCastKey, typeName)
}

// GetTypeCast retrieves the type cast filter from the request context
// Returns empty string if no type cast is present
func GetTypeCast(ctx context.Context) string {
	if typeCast, ok := ctx.Value(typeCastKey).(string); ok {
		return typeCast
	}
	return ""
}
