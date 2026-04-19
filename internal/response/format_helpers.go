package response

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	oquery "github.com/nlstn/go-odata/internal/query"
)

// Valid OData metadata levels per OData v4 specification
const (
	MetadataMinimal = "minimal"
	MetadataFull    = "full"
	MetadataNone    = "none"
)

// BuildEntityID constructs the entity ID path from entity set name and key values
// For single key: "Products(1)"
// For composite key: "ProductDescriptions(ProductID=1,LanguageKey='EN')"
func BuildEntityID(entitySetName string, keyValues map[string]interface{}) string {
	if len(keyValues) == 1 {
		for _, v := range keyValues {
			return fmt.Sprintf("%s(%s)", entitySetName, formatKeyValueLiteral(v))
		}
	}

	keys := make([]string, 0, len(keyValues))
	for k := range keyValues {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	keyParts := make([]string, len(keys))
	for i, k := range keys {
		keyParts[i] = fmt.Sprintf("%s=%s", k, formatKeyValueLiteral(keyValues[k]))
	}

	return fmt.Sprintf("%s(%s)", entitySetName, strings.Join(keyParts, ","))
}

func formatKeyValueLiteral(value interface{}) string {
	switch v := value.(type) {
	case string:
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ExtractEntityKeys extracts key values from an entity using metadata
func ExtractEntityKeys(entity interface{}, keyProperties []metadata.PropertyMetadata) map[string]interface{} {
	keyValues := make(map[string]interface{})
	entityValue := reflect.ValueOf(entity)

	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	for _, keyProp := range keyProperties {
		fieldValue := entityValue.FieldByName(keyProp.Name)
		if fieldValue.IsValid() {
			keyValues[keyProp.JsonName] = fieldValue.Interface()
		}
	}

	return keyValues
}

// isValidMetadataLevel checks if the given value is a valid OData metadata level
func isValidMetadataLevel(value string) bool {
	return value == MetadataMinimal || value == MetadataFull || value == MetadataNone
}

// validateMetadataValue returns an error if the value is not a valid metadata level
func validateMetadataValue(value string) error {
	if !isValidMetadataLevel(value) {
		return fmt.Errorf("invalid odata.metadata value: %s (valid values are: minimal, full, none)", value)
	}
	return nil
}

// ValidateODataMetadata checks if the odata.metadata parameter in the request is valid.
// Returns an error if an invalid metadata value is specified.
// Valid values are: "minimal", "full", "none"
func ValidateODataMetadata(r *http.Request) error {
	format := getFormatParameter(r.URL.RawQuery)
	if format != "" {
		if err := validateMetadataInFormat(format); err != nil {
			return err
		}
	}

	accept := r.Header.Get("Accept")
	if accept != "" {
		if err := validateMetadataInAccept(accept); err != nil {
			return err
		}
	}

	return nil
}

// GetODataMetadataLevel extracts the odata.metadata parameter value from the request
// Returns "minimal" (default), "full", or "none" based on Accept header or $format parameter
func GetODataMetadataLevel(r *http.Request) string {
	format := getFormatParameter(r.URL.RawQuery)
	if format != "" {
		return extractMetadataFromFormat(format)
	}

	accept := r.Header.Get("Accept")
	if accept != "" {
		return extractMetadataFromAccept(accept)
	}

	return MetadataMinimal
}

// GetIEEE754Compatible extracts IEEE754Compatible format parameter value from
// $format or Accept. Returns false when unspecified.
func GetIEEE754Compatible(r *http.Request) bool {
	format := getFormatParameter(r.URL.RawQuery)
	if format != "" {
		if value, ok := extractIEEE754FromMediaType(format); ok {
			return value
		}
	}

	accept := r.Header.Get("Accept")
	if accept != "" {
		parts := strings.Split(accept, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			subparts := strings.Split(part, ";")
			mimeType := strings.TrimSpace(subparts[0])
			if mimeType != "application/json" && mimeType != "*/*" && mimeType != "application/*" {
				continue
			}

			if value, ok := extractIEEE754FromParts(subparts[1:]); ok {
				return value
			}
		}
	}

	return false
}

// BuildJSONContentType builds the OData JSON Content-Type value from request format negotiation.
func BuildJSONContentType(r *http.Request) string {
	metadataLevel := GetODataMetadataLevel(r)
	if GetIEEE754Compatible(r) {
		return fmt.Sprintf("application/json;odata.metadata=%s;IEEE754Compatible=true", metadataLevel)
	}
	return fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel)
}

func getFormatParameter(rawQuery string) string {
	params := strings.Split(rawQuery, "&")
	for _, param := range params {
		if strings.HasPrefix(param, "$format=") || strings.HasPrefix(param, "%24format=") {
			parts := strings.SplitN(param, "=", 2)
			if len(parts) == 2 {
				decoded, err := url.QueryUnescape(parts[1])
				if err != nil {
					return parts[1]
				}
				return decoded
			}
		}
	}
	return ""
}

func validateMetadataInFormat(format string) error {
	parts := strings.Split(format, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "odata.metadata=") {
			value := strings.TrimPrefix(part, "odata.metadata=")
			value = strings.TrimSpace(value)
			if err := validateMetadataValue(value); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractMetadataFromFormat(format string) string {
	parts := strings.Split(format, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "odata.metadata=") {
			value := strings.TrimPrefix(part, "odata.metadata=")
			value = strings.TrimSpace(value)
			if isValidMetadataLevel(value) {
				return value
			}
		}
	}

	return MetadataMinimal
}

func validateMetadataInAccept(accept string) error {
	parts := strings.Split(accept, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		subparts := strings.Split(part, ";")
		mimeType := strings.TrimSpace(subparts[0])

		if mimeType == "application/json" || mimeType == "*/*" || mimeType == "application/*" {
			for _, param := range subparts[1:] {
				param = strings.TrimSpace(param)
				if strings.HasPrefix(param, "odata.metadata=") {
					value := strings.TrimPrefix(param, "odata.metadata=")
					value = strings.TrimSpace(value)
					if err := validateMetadataValue(value); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func extractMetadataFromAccept(accept string) string {
	parts := strings.Split(accept, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		subparts := strings.Split(part, ";")
		mimeType := strings.TrimSpace(subparts[0])

		if mimeType == "application/json" || mimeType == "*/*" || mimeType == "application/*" {
			for _, param := range subparts[1:] {
				param = strings.TrimSpace(param)
				if strings.HasPrefix(param, "odata.metadata=") {
					value := strings.TrimPrefix(param, "odata.metadata=")
					value = strings.TrimSpace(value)
					if isValidMetadataLevel(value) {
						return value
					}
				}
			}
		}
	}

	return MetadataMinimal
}

func extractIEEE754FromMediaType(mediaType string) (bool, bool) {
	parts := strings.Split(mediaType, ";")
	return extractIEEE754FromParts(parts[1:])
}

func extractIEEE754FromParts(parts []string) (bool, bool) {
	for _, param := range parts {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}

		segments := strings.SplitN(param, "=", 2)
		if len(segments) != 2 {
			continue
		}

		key := strings.TrimSpace(segments[0])
		value := strings.TrimSpace(segments[1])
		if !strings.EqualFold(key, "IEEE754Compatible") {
			continue
		}

		if strings.EqualFold(value, "true") {
			return true, true
		}
		if strings.EqualFold(value, "false") {
			return false, true
		}
	}

	return false, false
}

// IsAcceptableFormat checks if the requested format via Accept header or $format is supported
// Returns true if the format is acceptable (JSON, Atom/XML, or wildcard), false otherwise
func IsAcceptableFormat(r *http.Request) bool {
	format := getFormatParameter(r.URL.RawQuery)
	if format != "" {
		parts := strings.Split(format, ";")
		baseFormat := strings.TrimSpace(parts[0])
		return baseFormat == "json" || baseFormat == "application/json" ||
			baseFormat == "atom" || baseFormat == "application/atom+xml"
	}

	accept := r.Header.Get("Accept")
	if accept == "" {
		return true
	}

	type mediaType struct {
		mimeType string
		quality  float64
	}

	parts := strings.Split(accept, ",")
	mediaTypes := make([]mediaType, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		subparts := strings.Split(part, ";")
		mimeType := strings.TrimSpace(subparts[0])
		quality := 1.0

		for _, param := range subparts[1:] {
			param = strings.TrimSpace(param)
			if strings.HasPrefix(param, "q=") {
				var q float64
				if _, err := fmt.Sscanf(param[2:], "%f", &q); err == nil {
					if q >= 0 && q <= 1 {
						quality = q
					}
				}
			}
		}

		mediaTypes = append(mediaTypes, mediaType{mimeType: mimeType, quality: quality})
	}

	var bestJSON, bestAtom, bestXML, bestWildcard float64
	for _, mt := range mediaTypes {
		switch mt.mimeType {
		case "application/json":
			if mt.quality > bestJSON {
				bestJSON = mt.quality
			}
		case "application/atom+xml":
			if mt.quality > bestAtom {
				bestAtom = mt.quality
			}
		case "application/xml", "text/xml":
			if mt.quality > bestXML {
				bestXML = mt.quality
			}
		case "*/*", "application/*":
			if mt.quality > bestWildcard {
				bestWildcard = mt.quality
			}
		}
	}

	if bestWildcard > 0 {
		return true
	}
	if bestJSON > 0 {
		return true
	}
	if bestAtom > 0 {
		return true
	}
	if bestXML > 0 {
		return false
	}

	return true
}

// BuildBaseURL builds the base URL for the service (exported for use in handlers)
func BuildBaseURL(r *http.Request) string {
	return buildBaseURL(r)
}

func buildBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}

	pathPrefix := getBasePath(r)

	// Use strings.Builder to avoid string concatenation allocations
	var b strings.Builder
	b.Grow(len(scheme) + 3 + len(host) + len(pathPrefix)) // "://" is 3 chars
	b.WriteString(scheme)
	b.WriteString("://")
	b.WriteString(host)
	b.WriteString(pathPrefix)
	return b.String()
}

// BuildNextLink builds the next link URL for pagination using $skip
func BuildNextLink(r *http.Request, skipValue int) string {
	baseURL := buildBaseURL(r)

	nextURL := *r.URL
	q := oquery.NormalizeQueryParams(nextURL.Query())
	q.Del("$skiptoken")
	q.Set("$skip", fmt.Sprintf("%d", skipValue))
	nextURL.RawQuery = q.Encode()

	return baseURL + nextURL.Path + "?" + nextURL.RawQuery
}

// BuildNextLinkWithSkipToken builds the next link URL for server-driven pagination using $skiptoken
func BuildNextLinkWithSkipToken(r *http.Request, skipToken string) string {
	baseURL := buildBaseURL(r)

	nextURL := *r.URL
	q := oquery.NormalizeQueryParams(nextURL.Query())
	q.Del("$skip")
	q.Del("$skiptoken")
	q.Set("$skiptoken", skipToken)
	nextURL.RawQuery = q.Encode()

	return baseURL + nextURL.Path + "?" + nextURL.RawQuery
}

// BuildDeltaLink builds a delta link URL using the supplied delta token.
func BuildDeltaLink(r *http.Request, deltaToken string) string {
	baseURL := buildBaseURL(r)

	deltaURL := *r.URL
	query := deltaURL.Query()
	query.Set("$deltatoken", deltaToken)
	deltaURL.RawQuery = query.Encode()

	if deltaURL.RawQuery != "" {
		return baseURL + deltaURL.Path + "?" + deltaURL.RawQuery
	}

	return baseURL + deltaURL.Path
}

// buildContextURLWithSelect builds an OData context URL with optional select properties.
// When selectedProps is non-empty, the context URL includes the property list in parentheses:
//
//	#EntitySet(prop1,prop2)
//
// When empty, returns the plain collection context URL:
//
//	#EntitySet
func buildContextURLWithSelect(r *http.Request, entitySetName string, selectedProps []string) string {
	baseURL := buildBaseURL(r)
	if len(selectedProps) == 0 {
		return baseURL + "/$metadata#" + entitySetName
	}
	return baseURL + "/$metadata#" + entitySetName + "(" + strings.Join(selectedProps, ",") + ")"
}

// BuildEntityContextURL builds an OData context URL for a single entity with optional select properties.
// When selectedProps is non-empty, the context URL includes the property list:
//
//	#EntitySet(prop1,prop2)/$entity
//
// When empty, returns the plain entity context URL:
//
//	#EntitySet/$entity
func BuildEntityContextURL(r *http.Request, entitySetName string, selectedProps []string) string {
	baseURL := buildBaseURL(r)
	if len(selectedProps) == 0 {
		return baseURL + "/$metadata#" + entitySetName + "/$entity"
	}
	return baseURL + "/$metadata#" + entitySetName + "(" + strings.Join(selectedProps, ",") + ")/$entity"
}

func buildDeltaContextURL(r *http.Request, entitySetName string) string {
	baseURL := buildBaseURL(r)
	return baseURL + "/$metadata#" + entitySetName + "/$delta"
}

// getEntityTypeFromSetName derives the entity type name from the entity set name
func getEntityTypeFromSetName(entitySetName string) string {
	if strings.HasSuffix(entitySetName, "ies") {
		return entitySetName[:len(entitySetName)-3] + "y"
	}
	if strings.HasSuffix(entitySetName, "ses") || strings.HasSuffix(entitySetName, "xes") || strings.HasSuffix(entitySetName, "ches") || strings.HasSuffix(entitySetName, "shes") {
		return entitySetName[:len(entitySetName)-2]
	}
	if strings.HasSuffix(entitySetName, "s") {
		return entitySetName[:len(entitySetName)-1]
	}
	return entitySetName
}
