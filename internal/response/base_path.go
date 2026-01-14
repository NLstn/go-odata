package response

import "net/http"

// ContextKey is the type for context keys used by the response package.
type ContextKey string

// BasePathContextKey is the context key for storing the base path.
const BasePathContextKey ContextKey = "odata.basePath"

// getBasePath retrieves the base path from the request context.
// Returns empty string if not configured.
func getBasePath(r *http.Request) string {
	if basePath, ok := r.Context().Value(BasePathContextKey).(string); ok {
		return basePath
	}
	return ""
}
