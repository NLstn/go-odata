package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/metadata"
)

// OData version and header constants
const (
	// ODataVersionValue is the OData protocol version this library implements
	ODataVersionValue = "4.01"
	// HeaderODataVersion is the OData-Version header name (with exact capitalization)
	HeaderODataVersion = "OData-Version"
)

// SetODataVersionHeader sets the OData-Version header with the correct capitalization.
// Using direct map assignment to preserve exact capitalization as required by OData spec.
func SetODataVersionHeader(w http.ResponseWriter) {
	w.Header()[HeaderODataVersion] = []string{ODataVersionValue}
}

// ODataResponse represents the structure of an OData JSON response
type ODataResponse struct {
	Context  string      `json:"@odata.context,omitempty"`
	Count    *int64      `json:"@odata.count,omitempty"`
	NextLink *string     `json:"@odata.nextLink,omitempty"`
	Value    interface{} `json:"value"`
}

// EntityMetadataProvider is an interface for getting entity metadata
type EntityMetadataProvider interface {
	GetProperties() []PropertyMetadata
	GetKeyProperty() *PropertyMetadata    // Deprecated: Use GetKeyProperties for composite key support
	GetKeyProperties() []PropertyMetadata // Returns all key properties (single or composite)
	GetEntitySetName() string
	GetETagProperty() *PropertyMetadata // Returns the ETag property if configured
	GetNamespace() string
}

// PropertyMetadata represents metadata about a property
type PropertyMetadata struct {
	Name              string
	JsonName          string
	IsNavigationProp  bool
	NavigationTarget  string
	NavigationIsArray bool
}

// BuildEntityID constructs the entity ID path from entity set name and key values
// For single key: "Products(1)"
// For composite key: "ProductDescriptions(ProductID=1,LanguageKey='EN')"
func BuildEntityID(entitySetName string, keyValues map[string]interface{}) string {
	if len(keyValues) == 1 {
		// Single key - check if it's named or just a value
		for _, v := range keyValues {
			return fmt.Sprintf("%s(%v)", entitySetName, v)
		}
	}

	// Composite key or named single key
	var keyParts []string
	for k, v := range keyValues {
		// Quote string values
		if str, ok := v.(string); ok {
			keyParts = append(keyParts, fmt.Sprintf("%s='%s'", k, str))
		} else {
			keyParts = append(keyParts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	return fmt.Sprintf("%s(%s)", entitySetName, strings.Join(keyParts, ","))
}

// ExtractEntityKeys extracts key values from an entity using metadata
func ExtractEntityKeys(entity interface{}, keyProperties []metadata.PropertyMetadata) map[string]interface{} {
	keyValues := make(map[string]interface{})
	entityValue := reflect.ValueOf(entity)

	// Handle pointer
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

// WriteEntityReference writes an OData entity reference response for a single entity
func WriteEntityReference(w http.ResponseWriter, r *http.Request, entityID string) error {
	// Check if the requested format is supported
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses.")
	}

	baseURL := buildBaseURL(r)
	contextURL := baseURL + "/$metadata#$ref"

	response := map[string]interface{}{
		"@odata.context": contextURL,
		"@odata.id":      baseURL + "/" + entityID,
	}

	// Set OData-compliant headers with dynamic metadata level
	metadataLevel := GetODataMetadataLevel(r)
	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	SetODataVersionHeader(w)
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

// WriteEntityReferenceCollection writes an OData entity reference collection response
func WriteEntityReferenceCollection(w http.ResponseWriter, r *http.Request, entityIDs []string, count *int64, nextLink *string) error {
	// Check if the requested format is supported
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses.")
	}

	baseURL := buildBaseURL(r)
	contextURL := baseURL + "/$metadata#Collection($ref)"

	// Build the references
	refs := make([]map[string]string, len(entityIDs))
	for i, entityID := range entityIDs {
		refs[i] = map[string]string{
			"@odata.id": baseURL + "/" + entityID,
		}
	}

	response := map[string]interface{}{
		"@odata.context": contextURL,
		"value":          refs,
	}

	if count != nil {
		response["@odata.count"] = *count
	}

	if nextLink != nil && *nextLink != "" {
		response["@odata.nextLink"] = *nextLink
	}

	// Set OData-compliant headers with dynamic metadata level
	metadataLevel := GetODataMetadataLevel(r)
	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	SetODataVersionHeader(w)
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

// WriteODataCollection writes an OData collection response
func WriteODataCollection(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string) error {
	// Check if the requested format is supported
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses.")
	}

	// Get metadata level to determine which fields to include
	metadataLevel := GetODataMetadataLevel(r)

	// Build the context URL (only for minimal and full metadata)
	contextURL := ""
	if metadataLevel != "none" {
		contextURL = buildContextURL(r, entitySetName)
	}

	// Ensure empty collections are represented as [] not null per OData v4 spec
	if data == nil {
		data = []interface{}{}
	}

	response := ODataResponse{
		Context:  contextURL,
		Count:    count,
		NextLink: nextLink,
		Value:    data,
	}

	// Set OData-compliant headers with dynamic metadata level
	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return nil
	}

	// Encode and write the response
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

// WriteODataCollectionWithNavigation writes an OData collection response with navigation links
func WriteODataCollectionWithNavigation(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string, metadata EntityMetadataProvider, expandedProps []string, fullMetadata *metadata.EntityMetadata) error {
	// Check if the requested format is supported
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses.")
	}

	// Get metadata level to determine if we need to add @odata.type and @odata.context
	metadataLevel := GetODataMetadataLevel(r)

	// Build the context URL (only for minimal and full metadata)
	contextURL := ""
	if metadataLevel != "none" {
		contextURL = buildContextURL(r, entitySetName)
	}

	// Transform the data to add navigation links and type annotations
	transformedData := addNavigationLinks(data, metadata, expandedProps, r, entitySetName, metadataLevel, fullMetadata)

	// Ensure empty collections are represented as [] not null per OData v4 spec
	if transformedData == nil {
		transformedData = []interface{}{}
	}

	response := ODataResponse{
		Context:  contextURL,
		Count:    count,
		NextLink: nextLink,
		Value:    transformedData,
	}

	// Set OData-compliant headers with dynamic metadata level (already retrieved above)
	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return nil
	}

	// Encode and write the response
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

// addNavigationLinks adds @odata.navigationLink annotations for navigation properties
// and @odata.type annotations when full metadata is requested
func addNavigationLinks(data interface{}, metadata EntityMetadataProvider, expandedProps []string, r *http.Request, entitySetName string, metadataLevel string, fullMetadata *metadata.EntityMetadata) []interface{} {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		// Return empty slice instead of nil to ensure JSON marshaling produces []
		// instead of null, per OData v4 specification
		return []interface{}{}
	}

	result := make([]interface{}, dataValue.Len())
	baseURL := buildBaseURL(r)

	for i := 0; i < dataValue.Len(); i++ {
		entity := dataValue.Index(i)
		var entityMap interface{}

		if entity.Kind() == reflect.Map {
			entityMap = processMapEntity(entity, metadata, expandedProps, baseURL, entitySetName, metadataLevel, fullMetadata)
		} else {
			entityMap = processStructEntityOrdered(entity, metadata, expandedProps, baseURL, entitySetName, metadataLevel, fullMetadata)
		}

		if entityMap != nil {
			result[i] = entityMap
		}
	}

	return result
}

// processMapEntity processes an entity that is already a map and adds navigation links
// and @odata.type annotation when full metadata is requested
func processMapEntity(entity reflect.Value, metadata EntityMetadataProvider, expandedProps []string, baseURL, entitySetName string, metadataLevel string, fullMetadata *metadata.EntityMetadata) map[string]interface{} {
	entityMap, ok := entity.Interface().(map[string]interface{})
	if !ok {
		return nil
	}

	// Add @odata.etag annotation if ETag is configured
	// Per OData v4 spec, exclude all control information for metadata=none
	if fullMetadata != nil && fullMetadata.ETagProperty != nil && metadataLevel != "none" {
		etagValue := etag.Generate(entityMap, fullMetadata)
		if etagValue != "" {
			entityMap["@odata.etag"] = etagValue
		}
	}

	// Add @odata.id for full and minimal metadata levels
	// Per OData v4 spec section 4.5.1, @odata.id MUST be included in responses
	// except when odata.metadata=none
	keySegment := buildKeySegmentFromMap(entityMap, metadata)
	if keySegment != "" {
		entityID := fmt.Sprintf("%s/%s(%s)", baseURL, entitySetName, keySegment)

		switch metadataLevel {
		case "full", "minimal":
			// Always include @odata.id in full and minimal metadata
			entityMap["@odata.id"] = entityID
		}
		// For "none" metadata level, never include @odata.id
	}

	// Add @odata.type annotation for full metadata
	if metadataLevel == "full" {
		// Get entity type name from entity set name (remove trailing 's' for simple pluralization)
		// This is a simplified approach - in a real implementation, we'd get this from metadata
		entityTypeName := getEntityTypeFromSetName(entitySetName)
		entityMap["@odata.type"] = "#" + metadata.GetNamespace() + "." + entityTypeName
	}

	// Add navigation links only for full metadata (per OData v4 spec)
	// Minimal and none metadata levels do not include navigation links as they are computable by the client
	if metadataLevel == "full" {
		for _, prop := range metadata.GetProperties() {
			if !prop.IsNavigationProp {
				continue
			}

			if isPropertyExpanded(prop, expandedProps) {
				continue
			}

			// If property doesn't exist in map, add navigation link
			if _, exists := entityMap[prop.JsonName]; !exists {
				keySegment := buildKeySegmentFromMap(entityMap, metadata)
				if keySegment != "" {
					navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, prop.JsonName)
					entityMap[prop.JsonName+"@odata.navigationLink"] = navLink
				}
			}
		}
	}

	return entityMap
}

// processStructEntityOrdered processes an entity that is a struct and returns an OrderedMap
// and adds @odata.type annotation when full metadata is requested
func processStructEntityOrdered(entity reflect.Value, metadata EntityMetadataProvider, expandedProps []string, baseURL, entitySetName string, metadataLevel string, fullMetadata *metadata.EntityMetadata) *OrderedMap {
	entityMap := NewOrderedMap()
	entityType := entity.Type()

	// Add @odata.etag annotation if ETag is configured
	// Per OData v4 spec, exclude all control information for metadata=none
	if fullMetadata != nil && fullMetadata.ETagProperty != nil && metadataLevel != "none" {
		// Get the entity interface for etag.Generate
		var entityInterface interface{}
		if entity.Kind() == reflect.Ptr {
			entityInterface = entity.Interface()
		} else {
			// For non-pointer values, we need to get the address if possible
			if entity.CanAddr() {
				entityInterface = entity.Addr().Interface()
			} else {
				entityInterface = entity.Interface()
			}
		}
		etagValue := etag.Generate(entityInterface, fullMetadata)
		if etagValue != "" {
			entityMap.Set("@odata.etag", etagValue)
		}
	}

	// Add @odata.id for full and minimal metadata levels
	// Per OData v4 spec section 4.5.1, @odata.id MUST be included in responses
	// except when odata.metadata=none
	keySegment := BuildKeySegmentFromEntity(entity, metadata)
	if keySegment != "" {
		entityID := fmt.Sprintf("%s/%s(%s)", baseURL, entitySetName, keySegment)

		switch metadataLevel {
		case "full", "minimal":
			// Always include @odata.id in full and minimal metadata
			entityMap.Set("@odata.id", entityID)
		}
		// For "none" metadata level, never include @odata.id
	}

	// Add @odata.type annotation for full metadata
	if metadataLevel == "full" {
		entityTypeName := getEntityTypeFromSetName(entitySetName)
		entityMap.Set("@odata.type", "#"+metadata.GetNamespace()+"."+entityTypeName)
	}

	for j := 0; j < entity.NumField(); j++ {
		field := entityType.Field(j)
		if !field.IsExported() {
			continue
		}

		fieldValue := entity.Field(j)
		jsonName := getJsonFieldName(field)
		propMeta := findPropertyMetadata(field.Name, metadata)

		if propMeta != nil && propMeta.IsNavigationProp {
			processNavigationPropertyOrderedWithMetadata(entityMap, entity, propMeta, fieldValue, jsonName, expandedProps, baseURL, entitySetName, metadata, metadataLevel)
		} else {
			// Regular property - include its value
			entityMap.Set(jsonName, fieldValue.Interface())
		}
	}

	return entityMap
}

// isPropertyExpanded checks if a property is in the expanded properties list
func isPropertyExpanded(prop PropertyMetadata, expandedProps []string) bool {
	for _, expanded := range expandedProps {
		if expanded == prop.Name || expanded == prop.JsonName {
			return true
		}
	}
	return false
}

// findPropertyMetadata finds the metadata for a property by its field name
func findPropertyMetadata(fieldName string, metadata EntityMetadataProvider) *PropertyMetadata {
	props := metadata.GetProperties()
	for i := range props {
		if props[i].Name == fieldName {
			return &props[i]
		}
	}
	return nil
}

// processNavigationPropertyOrderedWithMetadata handles navigation properties with metadata level support
func processNavigationPropertyOrderedWithMetadata(entityMap *OrderedMap, entity reflect.Value, propMeta *PropertyMetadata, fieldValue reflect.Value, jsonName string, expandedProps []string, baseURL, entitySetName string, metadata EntityMetadataProvider, metadataLevel string) {
	if isPropertyExpanded(*propMeta, expandedProps) {
		// Include the expanded data
		entityMap.Set(jsonName, fieldValue.Interface())
	} else {
		// Add navigation link only for full metadata (per OData v4 spec)
		// Minimal and none metadata levels do not include navigation links as they are computable by the client
		if metadataLevel == "full" {
			keySegment := BuildKeySegmentFromEntity(entity, metadata)
			if keySegment != "" {
				navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, entitySetName, keySegment, propMeta.JsonName)
				entityMap.Set(jsonName+"@odata.navigationLink", navLink)
			}
		}
	}
}

// BuildKeySegmentFromEntity builds the key segment for URLs from an entity and metadata
// For single keys: returns "1"
// For composite keys: returns "ProductID=1,LanguageKey='EN'"
func BuildKeySegmentFromEntity(entity reflect.Value, metadata EntityMetadataProvider) string {
	keyProps := metadata.GetKeyProperties()
	if len(keyProps) == 0 {
		return ""
	}

	// Single key - return just the value
	if len(keyProps) == 1 {
		keyFieldValue := entity.FieldByName(keyProps[0].Name)
		if keyFieldValue.IsValid() {
			return fmt.Sprintf("%v", keyFieldValue.Interface())
		}
		return ""
	}

	// Composite keys - return key1=value1,key2=value2
	var parts []string
	for _, keyProp := range keyProps {
		keyFieldValue := entity.FieldByName(keyProp.Name)
		if keyFieldValue.IsValid() {
			keyValue := keyFieldValue.Interface()
			// Quote string values
			if keyFieldValue.Kind() == reflect.String {
				parts = append(parts, fmt.Sprintf("%s='%v'", keyProp.JsonName, keyValue))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%v", keyProp.JsonName, keyValue))
			}
		}
	}

	return strings.Join(parts, ",")
}

// buildKeySegmentFromMap builds the key segment for URLs from a map
// For single keys: returns "1"
// For composite keys: returns "ProductID=1,LanguageKey='EN'"
func buildKeySegmentFromMap(entityMap map[string]interface{}, metadata EntityMetadataProvider) string {
	keyProps := metadata.GetKeyProperties()
	if len(keyProps) == 0 {
		return ""
	}

	// Single key - return just the value
	if len(keyProps) == 1 {
		if keyValue := entityMap[keyProps[0].JsonName]; keyValue != nil {
			return fmt.Sprintf("%v", keyValue)
		}
		return ""
	}

	// Composite keys - return key1=value1,key2=value2
	var parts []string
	for _, keyProp := range keyProps {
		if keyValue := entityMap[keyProp.JsonName]; keyValue != nil {
			// Quote string values
			if strVal, ok := keyValue.(string); ok {
				parts = append(parts, fmt.Sprintf("%s='%s'", keyProp.JsonName, strVal))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%v", keyProp.JsonName, keyValue))
			}
		}
	}

	return strings.Join(parts, ",")
}

// getJsonFieldName extracts the JSON field name from struct tags
func getJsonFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return field.Name
	}

	// Handle json:",omitempty" or json:"fieldname,omitempty"
	parts := strings.Split(jsonTag, ",")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return field.Name
}

// ODataErrorDetail represents an additional error detail in an OData error response
type ODataErrorDetail struct {
	Code    string `json:"code,omitempty"`
	Target  string `json:"target,omitempty"`
	Message string `json:"message"`
}

// ODataInnerError represents nested error information in an OData error response
type ODataInnerError struct {
	Message    string           `json:"message,omitempty"`
	TypeName   string           `json:"type,omitempty"`
	StackTrace string           `json:"stacktrace,omitempty"`
	InnerError *ODataInnerError `json:"innererror,omitempty"`
}

// ODataError represents the OData v4 compliant error structure
type ODataError struct {
	Code       string             `json:"code"`
	Message    string             `json:"message"`
	Target     string             `json:"target,omitempty"`
	Details    []ODataErrorDetail `json:"details,omitempty"`
	InnerError *ODataInnerError   `json:"innererror,omitempty"`
}

// WriteError writes an OData v4 compliant error response
// For backwards compatibility, message and details params are used to create a basic error
func WriteError(w http.ResponseWriter, code int, message string, details string) error {
	odataErr := &ODataError{
		Code:    fmt.Sprintf("%d", code),
		Message: message,
	}

	// If details are provided, add them as a detail entry
	if details != "" {
		odataErr.Details = []ODataErrorDetail{
			{
				Message: details,
			},
		}
	}

	return WriteODataError(w, code, odataErr)
}

// WriteODataError writes an OData v4 compliant error response with full error structure
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

// WriteErrorWithTarget writes an OData error with target information
func WriteErrorWithTarget(w http.ResponseWriter, code int, message string, target string, details string) error {
	odataErr := &ODataError{
		Code:    fmt.Sprintf("%d", code),
		Message: message,
		Target:  target,
	}

	if details != "" {
		odataErr.Details = []ODataErrorDetail{
			{
				Message: details,
				Target:  target,
			},
		}
	}

	return WriteODataError(w, code, odataErr)
}

// WriteServiceDocument writes the OData service document
func WriteServiceDocument(w http.ResponseWriter, r *http.Request, entitySets []string, singletons []string) error {
	// Check if the requested format is supported
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for service documents.")
	}

	baseURL := buildBaseURL(r)

	entities := make([]map[string]interface{}, 0, len(entitySets)+len(singletons))

	// Add entity sets
	for _, entitySet := range entitySets {
		entities = append(entities, map[string]interface{}{
			"name": entitySet,
			"kind": "EntitySet",
			"url":  entitySet,
		})
	}

	// Add singletons
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

	// Set OData-compliant headers with dynamic metadata level
	metadataLevel := GetODataMetadataLevel(r)
	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return nil
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(serviceDoc)
}

// GetODataMetadataLevel extracts the odata.metadata parameter value from the request
// Returns "minimal" (default), "full", or "none" based on Accept header or $format parameter
func GetODataMetadataLevel(r *http.Request) string {
	// Check $format query parameter first (highest priority)
	// Use raw query string parsing to handle semicolons in $format value
	// (Go's standard Query() treats semicolons as parameter separators per HTML spec)
	format := getFormatParameter(r.URL.RawQuery)
	if format != "" {
		return extractMetadataFromFormat(format)
	}

	// Check Accept header
	accept := r.Header.Get("Accept")
	if accept != "" {
		return extractMetadataFromAccept(accept)
	}

	// Default to minimal
	return "minimal"
}

// getFormatParameter extracts the $format parameter from raw query string
// Handles semicolons in the value by parsing manually instead of using url.Query()
func getFormatParameter(rawQuery string) string {
	// Split by & to get individual parameters
	params := strings.Split(rawQuery, "&")
	for _, param := range params {
		// Find the parameter that starts with $format=
		if strings.HasPrefix(param, "$format=") || strings.HasPrefix(param, "%24format=") {
			// Extract the value after the equals sign
			parts := strings.SplitN(param, "=", 2)
			if len(parts) == 2 {
				// URL decode the value
				decoded, err := url.QueryUnescape(parts[1])
				if err != nil {
					// If decoding fails, return the raw value
					return parts[1]
				}
				return decoded
			}
		}
	}
	return ""
}

// extractMetadataFromFormat parses odata.metadata from $format parameter
func extractMetadataFromFormat(format string) string {
	// Format can be:
	// - "json" or "application/json" -> minimal (default)
	// - "application/json;odata.metadata=minimal"
	// - "application/json;odata.metadata=full"
	// - "application/json;odata.metadata=none"

	// Split by semicolon
	parts := strings.Split(format, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "odata.metadata=") {
			value := strings.TrimPrefix(part, "odata.metadata=")
			value = strings.TrimSpace(value)
			switch value {
			case "full", "none", "minimal":
				return value
			}
		}
	}

	// Default to minimal
	return "minimal"
}

// extractMetadataFromAccept parses odata.metadata from Accept header
func extractMetadataFromAccept(accept string) string {
	// Accept header can contain multiple media types with parameters
	// e.g., "application/json;odata.metadata=full, application/xml;q=0.8"

	parts := strings.Split(accept, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by semicolon to get media type and parameters
		subparts := strings.Split(part, ";")
		mimeType := strings.TrimSpace(subparts[0])

		// Only check JSON media types
		if mimeType == "application/json" || mimeType == "*/*" || mimeType == "application/*" {
			// Look for odata.metadata parameter
			for _, param := range subparts[1:] {
				param = strings.TrimSpace(param)
				if strings.HasPrefix(param, "odata.metadata=") {
					value := strings.TrimPrefix(param, "odata.metadata=")
					value = strings.TrimSpace(value)
					switch value {
					case "full", "none", "minimal":
						return value
					}
				}
			}
		}
	}

	// Default to minimal
	return "minimal"
}

// IsAcceptableFormat checks if the requested format via Accept header or $format is supported
// Returns true if the format is acceptable (JSON or wildcard), false otherwise (e.g., XML)
func IsAcceptableFormat(r *http.Request) bool {
	// Check $format query parameter first (highest priority)
	format := r.URL.Query().Get("$format")
	if format != "" {
		// Extract the base format (remove parameters like odata.metadata)
		parts := strings.Split(format, ";")
		baseFormat := strings.TrimSpace(parts[0])
		// Only JSON format is supported for data responses
		return baseFormat == "json" || baseFormat == "application/json"
	}

	// Check Accept header with proper content negotiation
	accept := r.Header.Get("Accept")
	if accept == "" {
		// No Accept header - default to JSON (acceptable)
		return true
	}

	// Parse Accept header to find the best match
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

		// Split by semicolon to separate media type from parameters
		subparts := strings.Split(part, ";")
		mimeType := strings.TrimSpace(subparts[0])
		quality := 1.0 // Default quality

		// Parse quality value if present
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

	// Find the best matching media type
	var bestJSON, bestXML, bestWildcard float64
	for _, mt := range mediaTypes {
		switch mt.mimeType {
		case "application/json":
			if mt.quality > bestJSON {
				bestJSON = mt.quality
			}
		case "application/xml", "text/xml", "application/atom+xml":
			if mt.quality > bestXML {
				bestXML = mt.quality
			}
		case "*/*", "application/*":
			if mt.quality > bestWildcard {
				bestWildcard = mt.quality
			}
		}
	}

	// If there's a wildcard, we can always return JSON (it matches the wildcard)
	if bestWildcard > 0 {
		return true
	}

	// If JSON is explicitly requested, accept
	if bestJSON > 0 {
		return true
	}

	// If only XML is specified (no JSON, no wildcard), reject
	if bestXML > 0 {
		return false
	}

	// No specific format requested - accept (defaults to JSON)
	return true
}

// buildContextURL builds the @odata.context URL for a response
func buildContextURL(r *http.Request, entitySetName string) string {
	baseURL := buildBaseURL(r)
	return baseURL + "/$metadata#" + entitySetName
}

// getEntityTypeFromSetName derives the entity type name from the entity set name
// This uses simple pluralization rules - removes trailing 's' or 'es'
func getEntityTypeFromSetName(entitySetName string) string {
	// Handle common pluralization patterns
	if strings.HasSuffix(entitySetName, "ies") {
		// Categories -> Category
		return entitySetName[:len(entitySetName)-3] + "y"
	}
	if strings.HasSuffix(entitySetName, "ses") || strings.HasSuffix(entitySetName, "xes") || strings.HasSuffix(entitySetName, "ches") || strings.HasSuffix(entitySetName, "shes") {
		// Classes -> Class, Boxes -> Box, Churches -> Church, Dishes -> Dish
		return entitySetName[:len(entitySetName)-2]
	}
	if strings.HasSuffix(entitySetName, "s") {
		// Products -> Product
		return entitySetName[:len(entitySetName)-1]
	}
	// No change needed
	return entitySetName
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

// BuildNextLink builds the next link URL for pagination using $skip
func BuildNextLink(r *http.Request, skipValue int) string {
	baseURL := buildBaseURL(r)

	// Clone the URL to avoid modifying the original
	nextURL := *r.URL

	// Get existing query parameters
	query := nextURL.Query()

	// Remove $skiptoken if present (we're using $skip)
	query.Del("$skiptoken")

	// Update the $skip parameter
	query.Set("$skip", fmt.Sprintf("%d", skipValue))

	// Rebuild the URL with updated query
	nextURL.RawQuery = query.Encode()

	return baseURL + nextURL.Path + "?" + nextURL.RawQuery
}

// BuildNextLinkWithSkipToken builds the next link URL for server-driven pagination using $skiptoken
func BuildNextLinkWithSkipToken(r *http.Request, skipToken string) string {
	baseURL := buildBaseURL(r)

	// Clone the URL to avoid modifying the original
	nextURL := *r.URL

	// Get existing query parameters
	query := nextURL.Query()

	// Remove $skip and $skiptoken if present
	query.Del("$skip")
	query.Del("$skiptoken")

	// Add the new $skiptoken parameter
	query.Set("$skiptoken", skipToken)

	// Rebuild the URL with updated query
	nextURL.RawQuery = query.Encode()

	return baseURL + nextURL.Path + "?" + nextURL.RawQuery
}

// ODataURLComponents represents the parsed components of an OData URL
type ODataURLComponents struct {
        EntitySet          string
        EntityKey          string            // For single keys: the value, for composite keys: empty (use EntityKeyMap)
        EntityKeyMap       map[string]string // For composite keys: map of key names to values
        NavigationProperty string            // For paths like Products(1)/Descriptions
        PropertyPath       string            // For structural property paths like Products(1)/Name
        PropertySegments   []string          // Property path segments excluding $value/$ref/$count
        IsCount            bool              // For paths like Products/$count
        IsValue            bool              // For paths like Products(1)/Name/$value
        IsRef              bool              // For paths like Products(1)/Descriptions/$ref
        ActionName         string            // For action invocations like Products(1)/Namespace.ActionName
        FunctionName       string            // For function invocations like Products(1)/Namespace.FunctionName
	IsAction           bool              // True if this is an action invocation
	IsFunction         bool              // True if this is a function invocation
}

// ParseODataURL parses an OData URL and extracts components (exported for use in main package)
func ParseODataURL(path string) (entitySet string, entityKey string, err error) {
	components, err := ParseODataURLComponents(path)
	if err != nil {
		return "", "", err
	}
	return components.EntitySet, components.EntityKey, err
}

// ParseODataURLComponents parses an OData URL and returns detailed components
func ParseODataURLComponents(path string) (*ODataURLComponents, error) {
	// Remove leading slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Parse URL
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	components := &ODataURLComponents{
		EntityKeyMap: make(map[string]string),
	}

	// Extract entity set and key
	pathParts := strings.Split(u.Path, "/")
        if len(pathParts) > 0 {
                entitySet := pathParts[0]

		// Check for key in parentheses: Products(1) or ProductDescriptions(ProductID=1,LanguageKey='EN')
		if idx := strings.Index(entitySet, "("); idx != -1 {
			if strings.HasSuffix(entitySet, ")") {
				keyPart := entitySet[idx+1 : len(entitySet)-1]
				components.EntitySet = entitySet[:idx]

				// Parse the key part - could be single key or composite
				if err := parseKeyPart(keyPart, components); err != nil {
					return nil, fmt.Errorf("invalid key format: %w", err)
				}
			}
		} else {
			components.EntitySet = entitySet
		}

                // Process additional path segments after the entity or entity(key)
                if len(pathParts) > 1 {
                        remainingParts := pathParts[1:]
                        propertySegments := make([]string, 0, len(remainingParts))

                        firstSegment := remainingParts[0]
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

// parseKeyPart parses the key portion of an OData URL
// Supports both single keys: (1) or (ID=1)
// and composite keys: (ProductID=1,LanguageKey='EN')
func parseKeyPart(keyPart string, components *ODataURLComponents) error {
	// Check if it contains '=' - if not, it's a simple single key value
	if !strings.Contains(keyPart, "=") {
		components.EntityKey = keyPart
		return nil
	}

	// Parse composite key format: key1=value1,key2=value2
	// Split by comma, but be careful of quoted values
	pairs, err := splitKeyPairs(keyPart)
	if err != nil {
		return err
	}

	for _, pair := range pairs {
		// Split by '='
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key-value pair: %s", pair)
		}

		keyName := strings.TrimSpace(parts[0])
		keyValue := strings.TrimSpace(parts[1])

		// Remove quotes from value if present (OData allows 'value' or just value)
		if len(keyValue) > 0 && (keyValue[0] == '\'' || keyValue[0] == '"') {
			keyValue = strings.Trim(keyValue, "'\"")
		}

		components.EntityKeyMap[keyName] = keyValue
	}

	// If only one key-value pair, also set EntityKey for backwards compatibility
	if len(components.EntityKeyMap) == 1 {
		for _, v := range components.EntityKeyMap {
			components.EntityKey = v
			break
		}
	}

	return nil
}

// splitKeyPairs splits key pairs by comma, respecting quoted values
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

		// Check for unclosed quote at end
		if i == len(input)-1 && inQuote {
			return nil, fmt.Errorf("unclosed quote in key part")
		}
	}

	// Add the last pair
	if current.Len() > 0 {
		pairs = append(pairs, current.String())
	}

	return pairs, nil
}
