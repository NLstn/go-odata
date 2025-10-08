package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ODataResponse represents the structure of an OData JSON response
type ODataResponse struct {
	Context  string      `json:"@odata.context"`
	Count    *int64      `json:"@odata.count,omitempty"`
	NextLink *string     `json:"@odata.nextLink,omitempty"`
	Value    interface{} `json:"value"`
}

// WriteODataCollection writes an OData collection response
func WriteODataCollection(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string) error {
	// Build the context URL
	contextURL := buildContextURL(r, entitySetName)

	response := ODataResponse{
		Context:  contextURL,
		Count:    count,
		NextLink: nextLink,
		Value:    data,
	}

	// Set OData-compliant headers
	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	// Encode and write the response
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(response)
}

// WriteError writes an OData error response
func WriteError(w http.ResponseWriter, code int, message string, details string) error {
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"details": details,
		},
	}

	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(code)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(errorResponse)
}

// WriteServiceDocument writes the OData service document
func WriteServiceDocument(w http.ResponseWriter, r *http.Request, entitySets []string) error {
	baseURL := buildBaseURL(r)

	entities := make([]map[string]interface{}, 0, len(entitySets))
	for _, entitySet := range entitySets {
		entities = append(entities, map[string]interface{}{
			"name": entitySet,
			"kind": "EntitySet",
			"url":  entitySet,
		})
	}

	serviceDoc := map[string]interface{}{
		"@odata.context": baseURL + "/$metadata",
		"value":          entities,
	}

	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(serviceDoc)
}

// buildContextURL builds the @odata.context URL for a response
func buildContextURL(r *http.Request, entitySetName string) string {
	baseURL := buildBaseURL(r)
	return baseURL + "/$metadata#" + entitySetName
}

// BuildBaseURL builds the base URL for the service (exported for use in handlers)
func BuildBaseURL(r *http.Request) string {
	return buildBaseURL(r)
}

// buildBaseURL builds the base URL for the service
func buildBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	// Handle X-Forwarded-Proto header for reverse proxies
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}

	return scheme + "://" + host
}

// BuildNextLink builds the next link URL for pagination
func BuildNextLink(r *http.Request, skipValue int) string {
	baseURL := buildBaseURL(r)

	// Clone the URL to avoid modifying the original
	nextURL := *r.URL

	// Get existing query parameters
	query := nextURL.Query()

	// Update the $skip parameter
	query.Set("$skip", fmt.Sprintf("%d", skipValue))

	// Rebuild the URL with updated query
	nextURL.RawQuery = query.Encode()

	return baseURL + nextURL.Path + "?" + nextURL.RawQuery
}

// ParseODataURL parses an OData URL and extracts components (exported for use in main package)
func ParseODataURL(path string) (entitySet string, entityKey string, err error) {
	return parseODataURL(path)
}

// parseODataURL parses an OData URL and extracts components
func parseODataURL(path string) (entitySet string, entityKey string, err error) {
	// Remove leading slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Parse URL
	u, err := url.Parse(path)
	if err != nil {
		return "", "", err
	}

	// Extract entity set and key
	pathParts := strings.Split(u.Path, "/")
	if len(pathParts) > 0 {
		entitySet = pathParts[0]

		// Check for key in parentheses: Products(1)
		if idx := strings.Index(entitySet, "("); idx != -1 {
			if strings.HasSuffix(entitySet, ")") {
				entityKey = entitySet[idx+1 : len(entitySet)-1]
				entitySet = entitySet[:idx]
			}
		}
	}

	return entitySet, entityKey, nil
}
