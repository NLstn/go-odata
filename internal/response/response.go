package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/version"
)

const (
	HeaderODataVersion = "OData-Version"
)

// SetODataVersionHeaderFromRequest sets the OData-Version header based on the negotiated version in the request context.
func SetODataVersionHeaderFromRequest(w http.ResponseWriter, r *http.Request) {
	ver := version.GetVersion(r.Context())
	w.Header().Set(HeaderODataVersion, ver.String())
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
	Code    string `json:"code"`
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

// WriteMethodNotAllowed writes a 405 Method Not Allowed response with the Allow header.
func WriteMethodNotAllowed(w http.ResponseWriter, r *http.Request, allow string, message string, details string) error {
	w.Header().Set("Allow", allow)
	return WriteError(w, r, http.StatusMethodNotAllowed, message, details)
}

// WriteError writes an OData v4 compliant error response.
func WriteError(w http.ResponseWriter, r *http.Request, code int, message string, details string) error {
	odataErr := &ODataError{
		Code:    fmt.Sprintf("%d", code),
		Message: message,
	}

	if details != "" {
		odataErr.Details = []ODataErrorDetail{{
			Code:    fmt.Sprintf("%d", code),
			Message: details,
		}}
	}

	return WriteODataError(w, r, code, odataErr)
}

// WriteODataError writes an OData v4 compliant error response with full error structure.
func WriteODataError(w http.ResponseWriter, r *http.Request, httpStatusCode int, odataError *ODataError) error {
	errorResponse := map[string]interface{}{
		"error": odataError,
	}

	w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
	// Built-in error messages are currently written in English. OData JSON
	// error responses must identify the language used for error.message.
	w.Header().Set("Content-Language", "en")
	SetODataVersionHeaderFromRequest(w, r)
	w.WriteHeader(httpStatusCode)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(errorResponse)
}

// WriteErrorWithTarget writes an OData error with target information.
func WriteErrorWithTarget(w http.ResponseWriter, r *http.Request, code int, message string, target string, details string) error {
	odataErr := &ODataError{
		Code:    fmt.Sprintf("%d", code),
		Message: message,
		Target:  target,
	}

	if details != "" {
		odataErr.Details = []ODataErrorDetail{{
			Code:    fmt.Sprintf("%d", code),
			Message: details,
			Target:  target,
		}}
	}

	return WriteODataError(w, r, code, odataErr)
}

// WriteServiceDocument writes the OData service document.
func WriteServiceDocument(w http.ResponseWriter, r *http.Request, entitySets []string, singletons []string) error {
	valueJSON, err := BuildServiceDocumentValueJSON(entitySets, singletons)
	if err != nil {
		return WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to marshal service document.")
	}
	return WriteServiceDocumentValue(w, r, valueJSON)
}

// BuildServiceDocumentValueJSON serializes the service document "value" array —
// the list of entity sets and singletons — into JSON. Each entry uses a relative
// url (the set/singleton name), so the result is independent of the request's
// base URL and can be built once and reused across requests.
//
// An empty input produces "[]" (never "null").
func BuildServiceDocumentValueJSON(entitySets []string, singletons []string) ([]byte, error) {
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

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(entities); err != nil {
		return nil, err
	}
	// Encode appends a trailing newline; drop it so the bytes splice cleanly
	// into the response envelope.
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// WriteServiceDocumentValue writes an OData service document whose "value" array
// has already been serialized (see BuildServiceDocumentValueJSON). Only the
// @odata.context — which embeds the request base URL — is assembled per request,
// avoiding a full re-marshal of the entity/singleton list on every call.
func WriteServiceDocumentValue(w http.ResponseWriter, r *http.Request, valueJSON []byte) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, r, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for service documents.")
	}

	if len(valueJSON) == 0 {
		valueJSON = []byte("[]")
	}

	// JSON-encode the context string so any characters in the base URL that
	// require escaping are handled correctly.
	contextJSON, err := json.Marshal(buildBaseURL(r) + "/$metadata")
	if err != nil {
		return WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to marshal service document.")
	}

	// OData JSON key order: @odata.context before value.
	var buf bytes.Buffer
	buf.Grow(len(contextJSON) + len(valueJSON) + 32)
	buf.WriteString(`{"@odata.context":`)
	buf.Write(contextJSON)
	buf.WriteString(`,"value":`)
	buf.Write(valueJSON)
	buf.WriteByte('}')
	body := buf.Bytes()

	metadataLevel := GetODataMetadataLevel(r)
	w.Header().Set("Content-Type", "application/json;odata.metadata="+metadataLevel)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(body)
	return err
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

	// Reject URLs with consecutive slashes before parsing
	// url.Parse treats //path as scheme-relative URL with "path" as host, not path
	// This would cause ///Products to be incorrectly accepted
	if strings.Contains(path, "//") {
		return nil, fmt.Errorf("invalid URL: empty path segments are not allowed per OData specification")
	}

	components := &ODataURLComponents{
		EntityKeyMap: make(map[string]string),
	}

	pathParts := strings.Split(path, "/")

	// Remove leading/trailing empty segments from leading/trailing slashes
	// We already rejected consecutive slashes above, so any empty segments here
	// are just from single leading/trailing slashes
	filteredParts := make([]string, 0, len(pathParts))
	for _, part := range pathParts {
		if part != "" {
			filteredParts = append(filteredParts, part)
		}
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
		cleanKey := keyPart
		if unquoted, ok, err := unquoteODataStringLiteral(cleanKey); err != nil {
			return err
		} else if ok {
			cleanKey = unquoted
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

		if unquoted, ok, err := unquoteODataStringLiteral(keyValue); err != nil {
			return err
		} else if ok {
			keyValue = unquoted
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

func unquoteODataStringLiteral(value string) (string, bool, error) {
	if len(value) < 2 {
		return value, false, nil
	}

	quote := value[0]
	if quote != '\'' && quote != '"' {
		return value, false, nil
	}
	if value[len(value)-1] != quote {
		return "", false, fmt.Errorf("unterminated string literal")
	}

	unquoted := value[1 : len(value)-1]
	if quote == '\'' {
		unquoted = strings.ReplaceAll(unquoted, "''", "'")
	}
	return unquoted, true, nil
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
