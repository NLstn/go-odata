package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	ODataVersionValue  = "4.01"
	HeaderODataVersion = "OData-Version"
)

// SetODataVersionHeader sets the OData-Version header with the correct capitalization.
func SetODataVersionHeader(w http.ResponseWriter) {
	w.Header()[HeaderODataVersion] = []string{ODataVersionValue}
}

// ODataResponse represents the structure of an OData JSON response.
type ODataResponse struct {
	Context  string      `json:"@odata.context,omitempty"`
	Count    *int64      `json:"@odata.count,omitempty"`
	NextLink *string     `json:"@odata.nextLink,omitempty"`
	Value    interface{} `json:"value"`
}

// EntityMetadataProvider describes metadata required by response writers.
type EntityMetadataProvider interface {
	GetProperties() []PropertyMetadata
	GetKeyProperty() *PropertyMetadata
	GetKeyProperties() []PropertyMetadata
	GetEntitySetName() string
	GetETagProperty() *PropertyMetadata
	GetNamespace() string
}

// PropertyMetadata represents metadata about a property.
type PropertyMetadata struct {
	Name              string
	JsonName          string
	IsNavigationProp  bool
	NavigationTarget  string
	NavigationIsArray bool
}

// ODataErrorDetail represents an additional error detail in an OData error response.
type ODataErrorDetail struct {
	Code    string `json:"code,omitempty"`
	Target  string `json:"target,omitempty"`
	Message string `json:"message"`
}

// ODataInnerError represents nested error information in an OData error response.
type ODataInnerError struct {
	Message    string           `json:"message,omitempty"`
	TypeName   string           `json:"type,omitempty"`
	StackTrace string           `json:"stacktrace,omitempty"`
	InnerError *ODataInnerError `json:"innererror,omitempty"`
}

// ODataError represents the OData v4 compliant error structure.
type ODataError struct {
	Code       string             `json:"code"`
	Message    string             `json:"message"`
	Target     string             `json:"target,omitempty"`
	Details    []ODataErrorDetail `json:"details,omitempty"`
	InnerError *ODataInnerError   `json:"innererror,omitempty"`
}

// WriteError writes an OData v4 compliant error response.
func WriteError(w http.ResponseWriter, code int, message string, details string) error {
	odataErr := &ODataError{
		Code:    fmt.Sprintf("%d", code),
		Message: message,
	}

	if details != "" {
		odataErr.Details = []ODataErrorDetail{{Message: details}}
	}

	return WriteODataError(w, code, odataErr)
}

// WriteODataError writes an OData v4 compliant error response with full error structure.
func WriteODataError(w http.ResponseWriter, httpStatusCode int, odataError *ODataError) error {
	errorResponse := map[string]interface{}{
		"error": odataError,
	}

	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	SetODataVersionHeader(w)
	w.WriteHeader(httpStatusCode)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(errorResponse)
}

// WriteErrorWithTarget writes an OData error with target information.
func WriteErrorWithTarget(w http.ResponseWriter, code int, message string, target string, details string) error {
	odataErr := &ODataError{
		Code:    fmt.Sprintf("%d", code),
		Message: message,
		Target:  target,
	}

	if details != "" {
		odataErr.Details = []ODataErrorDetail{{
			Message: details,
			Target:  target,
		}}
	}

	return WriteODataError(w, code, odataErr)
}

// WriteServiceDocument writes the OData service document.
func WriteServiceDocument(w http.ResponseWriter, r *http.Request, entitySets []string, singletons []string) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for service documents.")
	}

	baseURL := buildBaseURL(r)

	entities := make([]map[string]interface{}, 0, len(entitySets)+len(singletons))
	for _, entitySet := range entitySets {
		entities = append(entities, map[string]interface{}{
			"name": entitySet,
			"kind": "EntitySet",
			"url":  entitySet,
		})
	}
	for _, singleton := range singletons {
		entities = append(entities, map[string]interface{}{
			"name": singleton,
			"kind": "Singleton",
			"url":  singleton,
		})
	}

	serviceDoc := map[string]interface{}{
		"@odata.context": baseURL + "/$metadata",
		"value":          entities,
	}

	metadataLevel := GetODataMetadataLevel(r)
	if r.Method == http.MethodHead {
		jsonBytes, err := json.Marshal(serviceDoc)
		if err != nil {
			return WriteError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to marshal service document.")
		}
		w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))
		w.WriteHeader(http.StatusOK)
		return nil
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(serviceDoc)
}

// ODataURLComponents represents the parsed components of an OData URL.
type ODataURLComponents struct {
	EntitySet          string
	EntityKey          string
	EntityKeyMap       map[string]string
	NavigationProperty string
	PropertyPath       string
	PropertySegments   []string
	IsCount            bool
	IsValue            bool
	IsRef              bool
	ActionName         string
	FunctionName       string
	IsAction           bool
	IsFunction         bool
	TypeCast           string
}

// ParseODataURL parses an OData URL and extracts components.
func ParseODataURL(path string) (entitySet string, entityKey string, err error) {
	components, err := ParseODataURLComponents(path)
	if err != nil {
		return "", "", err
	}
	return components.EntitySet, components.EntityKey, err
}

// ParseODataURLComponents parses an OData URL and returns detailed components.
func ParseODataURLComponents(path string) (*ODataURLComponents, error) {
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	components := &ODataURLComponents{
		EntityKeyMap: make(map[string]string),
	}

	pathParts := strings.Split(u.Path, "/")
	
	// Normalize path by removing leading/trailing empty segments coming from
	// leading/trailing slashes, but reject empty segments in the middle,
	// which arise from consecutive slashes like /Products//Details.
	filteredParts := make([]string, 0, len(pathParts))
	for i, part := range pathParts {
		if part == "" {
			// Allow empty segments only at the start or end (leading/trailing slash),
			// but disallow empty segments in the middle of the path.
			if i != 0 && i != len(pathParts)-1 {
				return nil, fmt.Errorf("invalid URL: empty path segments are not allowed per OData specification")
			}
			continue
		}
		filteredParts = append(filteredParts, part)
	}
	
	pathParts = filteredParts
	if len(pathParts) > 0 {
		entitySet := pathParts[0]

		if idx := strings.Index(entitySet, "("); idx != -1 {
			if strings.HasSuffix(entitySet, ")") {
				keyPart := entitySet[idx+1 : len(entitySet)-1]
				components.EntitySet = entitySet[:idx]

				if err := parseKeyPart(keyPart, components); err != nil {
					return nil, fmt.Errorf("invalid key format: %w", err)
				}
			}
		} else {
			components.EntitySet = entitySet
		}

		if len(pathParts) > 1 {
			remainingParts := pathParts[1:]
			propertySegments := make([]string, 0, len(remainingParts))

			firstSegment := remainingParts[0]
			if isTypeCastSegment(firstSegment) {
				components.TypeCast = firstSegment
				if len(remainingParts) > 1 {
					remainingParts = remainingParts[1:]
					firstSegment = remainingParts[0]
				} else {
					return components, nil
				}
			}

			switch firstSegment {
			case "$count":
				components.IsCount = true
			case "$ref":
				components.IsRef = true
			case "$value":
				components.IsValue = true
			default:
				propertySegments = append(propertySegments, firstSegment)
				components.NavigationProperty = firstSegment

				for _, segment := range remainingParts[1:] {
					switch segment {
					case "$value":
						components.IsValue = true
					case "$ref":
						components.IsRef = true
					case "$count":
						components.IsCount = true
					default:
						propertySegments = append(propertySegments, segment)
					}
				}
			}

			if len(propertySegments) > 0 {
				components.PropertySegments = propertySegments
				components.PropertyPath = strings.Join(propertySegments, "/")
			}
		}
	}

	return components, nil
}

func parseKeyPart(keyPart string, components *ODataURLComponents) error {
	if !strings.Contains(keyPart, "=") {
		// For single keys, strip surrounding quotes (same as composite keys)
		cleanKey := keyPart
		if len(cleanKey) >= 2 {
			// Check if the key is surrounded by matching quotes
			if (cleanKey[0] == '\'' && cleanKey[len(cleanKey)-1] == '\'') ||
				(cleanKey[0] == '"' && cleanKey[len(cleanKey)-1] == '"') {
				cleanKey = cleanKey[1 : len(cleanKey)-1]
			}
		}
		components.EntityKey = cleanKey
		return nil
	}

	pairs, err := splitKeyPairs(keyPart)
	if err != nil {
		return err
	}

	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key-value pair: %s", pair)
		}

		keyName := strings.TrimSpace(parts[0])
		keyValue := strings.TrimSpace(parts[1])

		// Strip surrounding quotes if they match
		if len(keyValue) >= 2 {
			if (keyValue[0] == '\'' && keyValue[len(keyValue)-1] == '\'') ||
				(keyValue[0] == '"' && keyValue[len(keyValue)-1] == '"') {
				keyValue = keyValue[1 : len(keyValue)-1]
			}
		}

		components.EntityKeyMap[keyName] = keyValue
	}

	if len(components.EntityKeyMap) == 1 {
		for _, v := range components.EntityKeyMap {
			components.EntityKey = v
			break
		}
	}

	return nil
}

func splitKeyPairs(input string) ([]string, error) {
	var pairs []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, ch := range input {
		switch {
		case (ch == '\'' || ch == '"') && !inQuote:
			inQuote = true
			quoteChar = ch
			current.WriteRune(ch)
		case ch == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
			current.WriteRune(ch)
		case ch == ',' && !inQuote:
			pairs = append(pairs, current.String())
			current.Reset()
		default:
			current.WriteRune(ch)
		}

		if i == len(input)-1 && inQuote {
			return nil, fmt.Errorf("unclosed quote in key part")
		}
	}

	if current.Len() > 0 {
		pairs = append(pairs, current.String())
	}

	return pairs, nil
}

func isTypeCastSegment(segment string) bool {
	if !strings.Contains(segment, ".") {
		return false
	}

	if strings.HasPrefix(segment, "$") {
		return false
	}

	if strings.Contains(segment, "(") || strings.Contains(segment, ")") {
		return false
	}

	parts := strings.Split(segment, ".")
	if len(parts) < 2 {
		return false
	}

	typeName := parts[len(parts)-1]
	if len(typeName) == 0 {
		return false
	}

	firstChar := rune(typeName[0])
	if firstChar < 'A' || firstChar > 'Z' {
		return false
	}

	return true
}
