package handlers

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
)

// parseEntityKeyValues parses an entity key string into a map of key-value pairs.
// For single keys like "42", returns a map with one entry using the first key property's name.
// For composite keys like "OrderID=1,ProductID=5", returns map with multiple entries.
// Returns nil if entityKey is empty (for collection operations).
// Returns an empty map if keyProperties is empty/nil but entityKey is non-empty (missing metadata).
func parseEntityKeyValues(entityKey string, keyProperties []metadata.PropertyMetadata) map[string]interface{} {
	if entityKey == "" {
		return nil
	}

	keyValues := make(map[string]interface{})

	// Try to parse as composite key format first
	components := &response.ODataURLComponents{
		EntityKeyMap: make(map[string]string),
	}

	if err := parseCompositeKey(entityKey, components); err == nil && len(components.EntityKeyMap) > 0 {
		// Composite key format: key1=value1,key2=value2
		for key, value := range components.EntityKeyMap {
			// Convert string value to appropriate type based on metadata
			keyValues[key] = convertKeyValue(value, key, keyProperties)
		}
	} else {
		// Single key - use the first key property name
		if len(keyProperties) > 0 {
			keyValues[keyProperties[0].JsonName] = convertKeyValue(entityKey, keyProperties[0].JsonName, keyProperties)
		}
	}

	return keyValues
}

// convertKeyValue attempts to convert a string key value to the appropriate type
// based on the property metadata. Returns the value with the exact type from metadata.
// Returns string if conversion fails or type is unknown.
func convertKeyValue(value string, keyName string, keyProperties []metadata.PropertyMetadata) interface{} {
	// Find the property metadata for this key
	var propType reflect.Type
	for _, prop := range keyProperties {
		if prop.JsonName == keyName || prop.Name == keyName {
			propType = prop.Type
			break
		}
	}

	// If we didn't find the type or type is nil, return string
	if propType == nil {
		return value
	}

	// Try to convert based on type kind, returning the exact type from metadata
	switch propType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := strconv.ParseInt(value, 10, propType.Bits()); err == nil {
			v := reflect.New(propType).Elem()
			v.SetInt(intVal)
			return v.Interface()
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := strconv.ParseUint(value, 10, propType.Bits()); err == nil {
			v := reflect.New(propType).Elem()
			v.SetUint(uintVal)
			return v.Interface()
		}
	case reflect.Float32, reflect.Float64:
		if floatVal, err := strconv.ParseFloat(value, propType.Bits()); err == nil {
			v := reflect.New(propType).Elem()
			v.SetFloat(floatVal)
			return v.Interface()
		}
	case reflect.Bool:
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	case reflect.String:
		return value
	}

	// Default to string if conversion fails or type is unknown
	return value
}

// SetODataHeader sets an HTTP header preserving the exact capitalization.
// This is needed for OData-specific headers like "OData-Version" and "OData-EntityId"
// which do not follow Go's standard HTTP header canonicalization (which would turn them into "Odata-Version").
// The OData v4 specification requires these headers to have exact capitalization with a capital 'D'.
//
// This function sets the header using direct map assignment to preserve the exact key,
// ensuring the header is sent over the wire with the correct capitalization as required by OData spec.
//
// Note: Due to the non-canonical key, Header.Get() will not work for accessing these headers.
// Use direct map access (w.Header()["OData-Version"]) instead when you need to read them.
func SetODataHeader(w http.ResponseWriter, key, value string) {
	// Set with exact capitalization for wire format (OData spec compliance)
	w.Header()[key] = []string{value}
}

// SetODataVersionHeader sets the OData-Version header with the correct version value.
// This centralizes the version header setting to ensure consistency across all responses.
//
// Deprecated: The version header is now set automatically in the router middleware.
// This function will be removed in v2.0.0 (target: June 2026).
//
// Migration:
//
//	// Old (deprecated):
//	handlers.SetODataVersionHeader(w)
//
//	// New (context-aware):
//	response.SetODataVersionHeaderFromRequest(w, r)
//
// Note: In most cases, you don't need to call this manually as the router
// automatically sets the version header based on client negotiation.
func SetODataVersionHeader(w http.ResponseWriter) {
	SetODataHeader(w, HeaderODataVersion, response.ODataVersionValue)
}

// buildKeyQuery builds a GORM query with WHERE conditions for the entity key(s)
// Supports both single keys and composite keys. When db is nil the handler's default
// database handle is used.
func (h *EntityHandler) buildKeyQuery(db *gorm.DB, entityKey string) (*gorm.DB, error) {
	if db == nil {
		db = h.db
	}

	// Parse the key - could be single value or composite key format
	components := &response.ODataURLComponents{
		EntityKeyMap: make(map[string]string),
	}

	// Try to parse as composite key format first
	keyPart := entityKey
	if err := parseCompositeKey(keyPart, components); err != nil {
		// If parsing fails, treat as simple single key
		components.EntityKey = entityKey
		components.EntityKeyMap = nil
	}

	// If we have a composite key map, use it
	if len(components.EntityKeyMap) > 0 {
		// Build WHERE clause for each key property
		for _, keyProp := range h.metadata.KeyProperties {
			keyValue, found := components.EntityKeyMap[keyProp.JsonName]
			if !found {
				// Also try the field name
				keyValue, found = components.EntityKeyMap[keyProp.Name]
			}
			if !found {
				return nil, fmt.Errorf("missing key property: %s", keyProp.JsonName)
			}
			// Use cached column name from metadata
			db = db.Where(fmt.Sprintf("%s = ?", keyProp.ColumnName), keyValue)
		}
	} else {
		// Single key - use backwards compatible logic
		if len(h.metadata.KeyProperties) != 1 {
			return nil, fmt.Errorf("entity has composite keys, please use composite key format: key1=value1,key2=value2")
		}
		// Use cached column name from metadata
		db = db.Where(fmt.Sprintf("%s = ?", h.metadata.KeyProperties[0].ColumnName), entityKey)
	}

	return db, nil
}

// toSnakeCase converts a camelCase or PascalCase string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if the previous character was lowercase or if this is the start of a new word
			// For "ProductID", we want "product_id" not "product_i_d"
			prevRune := rune(s[i-1])
			if prevRune >= 'a' && prevRune <= 'z' {
				result.WriteRune('_')
			} else if i < len(s)-1 {
				// Check if next character is lowercase (e.g., "XMLParser" -> "xml_parser")
				nextRune := rune(s[i+1])
				if nextRune >= 'a' && nextRune <= 'z' {
					result.WriteRune('_')
				}
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// parseCompositeKey attempts to parse a composite key string
func parseCompositeKey(keyPart string, components *response.ODataURLComponents) error {
	// Check if it contains '=' - if not, it's a simple single key value
	if !strings.Contains(keyPart, "=") {
		return fmt.Errorf("not a composite key")
	}

	// Parse composite key format: key1=value1,key2=value2
	pairs := strings.Split(keyPart, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key-value pair: %s", pair)
		}

		keyName := strings.TrimSpace(parts[0])
		keyValue := strings.TrimSpace(parts[1])

		// Remove quotes from value if they match
		if len(keyValue) >= 2 {
			if (keyValue[0] == '\'' && keyValue[len(keyValue)-1] == '\'') ||
				(keyValue[0] == '"' && keyValue[len(keyValue)-1] == '"') {
				keyValue = keyValue[1 : len(keyValue)-1]
			}
		}

		components.EntityKeyMap[keyName] = keyValue
	}

	return nil
}

// handleFetchError writes appropriate error responses based on the fetch error type
func (h *EntityHandler) handleFetchError(w http.ResponseWriter, r *http.Request, err error, entityKey string) {
	if err == gorm.ErrRecordNotFound {
		target := fmt.Sprintf(ODataEntityKeyFormat, h.metadata.EntitySetName, entityKey)
		if writeErr := response.WriteErrorWithTarget(w, r, http.StatusNotFound, ErrMsgEntityNotFound,
			target, fmt.Sprintf(EntityKeyNotExistFmt, entityKey)); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
	} else {
		if writeErr := response.WriteError(w, r, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
	}
}

func (h *EntityHandler) supportsTrackChanges() bool {
	if h.metadata == nil {
		return false
	}
	return h.tracker != nil && h.metadata.ChangeTrackingEnabled && !h.metadata.IsSingleton
}

func (h *EntityHandler) recordChange(entity interface{}, changeType trackchanges.ChangeType) {
	if !h.supportsTrackChanges() {
		return
	}

	keyValues := h.extractKeyValues(entity)
	var data map[string]interface{}
	if changeType != trackchanges.ChangeTypeDeleted {
		data = h.entityToMap(entity)
	}
	if _, err := h.tracker.RecordChange(h.metadata.EntitySetName, keyValues, data, changeType); err != nil {
		if h.logger != nil {
			h.logger.Error("failed to record change event", "entitySet", h.metadata.EntitySetName, "err", err)
		}
	}
}

func (h *EntityHandler) finalizeChangeEvents(ctx context.Context, events []changeEvent) {
	if len(events) == 0 {
		return
	}
	if _, ok := TransactionFromContext(ctx); ok {
		addPendingChangeEvents(ctx, h, events)
		return
	}
	for _, event := range events {
		h.recordChange(event.entity, event.changeType)
	}
}

func (h *EntityHandler) extractKeyValues(entity interface{}) map[string]interface{} {
	keyValues := make(map[string]interface{})
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	if !entityValue.IsValid() {
		return keyValues
	}

	for _, keyProp := range h.metadata.KeyProperties {
		field := entityValue.FieldByName(keyProp.Name)
		if field.IsValid() {
			keyValues[keyProp.JsonName] = field.Interface()
		}
	}

	return keyValues
}

func (h *EntityHandler) entityToMap(entity interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	if !entityValue.IsValid() {
		return result
	}

	for _, prop := range h.metadata.Properties {
		field := entityValue.FieldByName(prop.Name)
		if field.IsValid() {
			result[prop.JsonName] = field.Interface()
		}
	}

	return result
}

// buildEntityResponseWithMetadata builds an OData entity response with metadata level support
func (h *EntityHandler) buildEntityResponseWithMetadata(navValue reflect.Value, contextURL string, metadataLevel string) map[string]interface{} {
	odataResponse := response.NewOrderedMap()

	// Only include @odata.context for minimal and full metadata (not for none)
	if metadataLevel != "none" {
		odataResponse.Set(ODataContextProperty, contextURL)
	}

	// Add @odata.type for full metadata
	if metadataLevel == "full" {
		odataResponse.Set("@odata.type", "#"+h.qualifiedTypeName(h.metadata.EntityName))
	}

	navType := navValue.Type()
	for i := 0; i < navValue.NumField(); i++ {
		field := navType.Field(i)
		if field.IsExported() {
			fieldValue := navValue.Field(i)
			// Use cached metadata instead of reflection for performance
			jsonName := field.Name // Default to field name
			if propMeta := h.findPropertyMetadata(field.Name); propMeta != nil {
				jsonName = propMeta.JsonName
			}
			odataResponse.Set(jsonName, fieldValue.Interface())
		}
	}

	return odataResponse.ToMap()
}

// buildOrderedEntityResponseWithMetadata builds an ordered OData entity response with metadata level support
func (h *EntityHandler) buildOrderedEntityResponseWithMetadata(result interface{}, contextURL string, metadataLevel string, r *http.Request, etagValue string, expandOptions []query.ExpandOption) *response.OrderedMap {
	odataResponse := response.NewOrderedMap()

	// Only include @odata.context for minimal and full metadata (not for none)
	if metadataLevel != "none" {
		odataResponse.Set(ODataContextProperty, contextURL)
	}

	// Add @odata.etag annotation if ETag value is available
	// Per OData v4 spec, exclude all control information for metadata=none
	if etagValue != "" && metadataLevel != "none" {
		odataResponse.Set("@odata.etag", etagValue)
	}

	// Check if result is a map (from $select) or a struct
	resultValue := reflect.ValueOf(result)

	// Build the entity ID for @odata.id
	var entityID string
	switch resultValue.Kind() {
	case reflect.Ptr:
		entityID = h.buildEntityIDFromValue(resultValue.Elem(), r)
	case reflect.Struct:
		entityID = h.buildEntityIDFromValue(resultValue, r)
	case reflect.Map:
		if mapResult, ok := result.(map[string]interface{}); ok {
			entityID = h.buildEntityIDFromMap(mapResult, r)
		}
	}

	// Add @odata.id based on metadata level
	if entityID != "" {
		switch metadataLevel {
		case "full":
			// Always include @odata.id in full metadata
			odataResponse.Set("@odata.id", entityID)
		case "minimal":
			// For minimal metadata, check later after processing fields
			odataResponse.Set("__temp_entity_id", entityID)
		}
		// For "none" metadata level, never include @odata.id
	}

	// Add @odata.type for full metadata
	if metadataLevel == "full" {
		odataResponse.Set("@odata.type", "#"+h.qualifiedTypeName(h.metadata.EntityName))

		// Add entity-level vocabulary annotations for full metadata
		if h.metadata.Annotations != nil && h.metadata.Annotations.Len() > 0 {
			for _, annotation := range h.metadata.Annotations.Get() {
				annotationKey := "@" + annotation.QualifiedTerm()
				odataResponse.Set(annotationKey, annotation.Value)
			}
		}
	}

	// Add media link annotations for media entities
	if h.metadata.HasStream && metadataLevel != "none" {
		if entityID != "" {
			mediaReadLink := entityID + "/$value"
			mediaEditLink := entityID + "/$value"
			odataResponse.Set("@odata.mediaReadLink", mediaReadLink)
			odataResponse.Set("@odata.mediaEditLink", mediaEditLink)

			// Add media content type if available
			entityValue := reflect.ValueOf(result)
			if entityValue.Kind() == reflect.Ptr {
				entityValue = entityValue.Elem()
			}
			if entityValue.Kind() == reflect.Struct && entityValue.CanAddr() {
				// Try to get content type from the entity
				if hasMethod := entityValue.Addr().MethodByName("GetMediaContentType"); hasMethod.IsValid() {
					results := hasMethod.Call(nil)
					if len(results) > 0 && results[0].Kind() == reflect.String {
						contentType := results[0].String()
						if contentType != "" {
							odataResponse.Set("@odata.mediaContentType", contentType)
						}
					}
				}
			}
		}
	}

	// Handle map[string]interface{} (from $select filtering)
	if resultValue.Kind() == reflect.Map {
		if mapResult, ok := result.(map[string]interface{}); ok {
			if len(expandOptions) > 0 && h.metadata != nil {
				mapResult = response.ApplyExpandAnnotationsToMap(mapResult, expandOptions, h.metadata)
			}

			// Iterate over map keys in a consistent order
			for _, key := range reflect.ValueOf(mapResult).MapKeys() {
				keyStr := key.String()
				value := reflect.ValueOf(mapResult).MapIndex(key)

				// Add property-level annotations for full metadata
				if metadataLevel == "full" {
					// Use findPropertyMetadata which now handles JsonName lookups via O(1) map
					propMeta := h.findPropertyMetadata(keyStr)
					if propMeta != nil && propMeta.Annotations != nil && propMeta.Annotations.Len() > 0 {
						for _, annotation := range propMeta.Annotations.Get() {
							annotationKey := keyStr + "@" + annotation.QualifiedTerm()
							odataResponse.Set(annotationKey, annotation.Value)
						}
					}
				}

				odataResponse.Set(keyStr, value.Interface())
			}
		}

		// For minimal metadata with map, check if all key fields are present
		if metadataLevel == "minimal" {
			if tempID, exists := odataResponse.ToMap()["__temp_entity_id"]; exists {
				// Remove the temporary ID
				odataResponse.Delete("__temp_entity_id")

				// Per OData v4 spec: For individual entity responses (with /$entity in context URL),
				// @odata.id MUST be included even if all key fields are present
				// For collection items, @odata.id is only needed when key fields are omitted
				isIndividualEntity := strings.Contains(contextURL, "/$entity")

				if isIndividualEntity {
					// Always add @odata.id for individual entity responses
					odataResponse.InsertAfter(ODataContextProperty, "@odata.id", tempID)
				} else {
					// For collection items, check if all key fields are present
					allKeysPresent := true
					for _, keyProp := range h.metadata.KeyProperties {
						if _, exists := odataResponse.ToMap()[keyProp.JsonName]; !exists {
							allKeysPresent = false
							break
						}
					}

					if !allKeysPresent {
						// Add @odata.id after @odata.context
						odataResponse.InsertAfter(ODataContextProperty, "@odata.id", tempID)
					}
				}
			}
		}

		return odataResponse
	}

	// Handle struct/pointer to struct (original entity)
	if resultValue.Kind() == reflect.Ptr {
		resultValue = resultValue.Elem()
	}

	entityType := resultValue.Type()
	for i := 0; i < resultValue.NumField(); i++ {
		field := entityType.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldValue := resultValue.Field(i)

		// Check if this field is a navigation property and get cached JsonName
		propMeta := h.findPropertyMetadata(field.Name)
		// Use cached metadata instead of reflection for performance
		jsonName := field.Name // Default to field name
		if propMeta != nil {
			jsonName = propMeta.JsonName
		}
		if propMeta != nil && propMeta.IsNavigationProp {
			// Check if the navigation property is populated (expanded)
			isExpanded := false
			if fieldValue.Kind() == reflect.Ptr {
				// For pointer types, check if not nil
				isExpanded = !fieldValue.IsNil()
			} else if fieldValue.Kind() == reflect.Slice {
				// For slices, check if not nil (empty slices are valid expanded data)
				isExpanded = !fieldValue.IsNil()
			}

			if isExpanded {
				updatedValue := fieldValue.Interface()
				if propMeta != nil && len(expandOptions) > 0 {
					expandOpt := query.FindExpandOption(expandOptions, propMeta.Name, propMeta.JsonName)
					if expandOpt != nil {
						if targetMetadata, err := h.metadata.ResolveNavigationTarget(propMeta.Name); err == nil {
							var count *int
							updatedValue, count = response.ApplyExpandOptionToValue(updatedValue, expandOpt, targetMetadata)
							if count != nil {
								odataResponse.Set(jsonName+"@odata.count", *count)
							}
						}
					}
				}

				// Include the expanded data
				odataResponse.Set(jsonName, updatedValue)
			} else {
				// Handle navigation property based on metadata level
				// Only add navigation link for full metadata (per OData v4 spec)
				// Minimal and none metadata levels do not include navigation links
				if metadataLevel == "full" && r != nil {
					// Build the key segment for the navigation link
					keySegment := h.buildKeySegmentFromEntity(resultValue)
					if keySegment != "" {
						baseURL := response.BuildBaseURL(r)
						navLink := fmt.Sprintf("%s/%s(%s)/%s", baseURL, h.metadata.EntitySetName, keySegment, propMeta.JsonName)
						odataResponse.Set(jsonName+"@odata.navigationLink", navLink)
					}
				}
			}
		} else {
			// Regular property - add property-level annotations first (for full metadata)
			if metadataLevel == "full" && propMeta != nil && propMeta.Annotations != nil && propMeta.Annotations.Len() > 0 {
				for _, annotation := range propMeta.Annotations.Get() {
					annotationKey := jsonName + "@" + annotation.QualifiedTerm()
					odataResponse.Set(annotationKey, annotation.Value)
				}
			}
			// Then include the property value
			odataResponse.Set(jsonName, fieldValue.Interface())
		}
	}

	// For minimal metadata, check if all key fields are present in struct
	if metadataLevel == "minimal" {
		if tempID, exists := odataResponse.ToMap()["__temp_entity_id"]; exists {
			// Remove the temporary ID
			odataResponse.Delete("__temp_entity_id")

			// Per OData v4 spec: For individual entity responses (with /$entity in context URL),
			// @odata.id MUST be included even if all key fields are present
			// For collection items, @odata.id is only needed when key fields are omitted
			isIndividualEntity := strings.Contains(contextURL, "/$entity")
			if isIndividualEntity {
				// Always add @odata.id for individual entity responses
				odataResponse.InsertAfter(ODataContextProperty, "@odata.id", tempID)
			}
			// For collection items (not individual entities), @odata.id is only added when key fields are omitted
			// This is already handled in the map case above (lines 230-248)
		}
	}

	return odataResponse
}

// findPropertyMetadata finds metadata for a property by field name, JSON name, or FieldName
// Uses the pre-computed property map for O(1) lookup instead of O(n) iteration
func (h *EntityHandler) findPropertyMetadata(fieldName string) *metadata.PropertyMetadata {
	if h.propertyMap != nil {
		if prop, ok := h.propertyMap[fieldName]; ok {
			return prop
		}
	}
	// Fallback to linear search if map not initialized (shouldn't happen normally)
	for i := range h.metadata.Properties {
		if h.metadata.Properties[i].Name == fieldName || h.metadata.Properties[i].FieldName == fieldName || h.metadata.Properties[i].JsonName == fieldName {
			return &h.metadata.Properties[i]
		}
	}
	return nil
}

// buildKeySegmentFromEntity builds the key segment for URLs from an entity value
func (h *EntityHandler) buildKeySegmentFromEntity(entity reflect.Value) string {
	keyProps := h.metadata.KeyProperties
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

// metadataAdapter adapts metadata.EntityMetadata to response.EntityMetadataProvider
// It pre-computes and caches converted property metadata to avoid allocations on every request
type metadataAdapter struct {
	metadata *metadata.EntityMetadata
	// Cached converted properties to avoid repeated allocations
	cachedProperties    []response.PropertyMetadata
	cachedKeyProperty   *response.PropertyMetadata
	cachedKeyProperties []response.PropertyMetadata
	cachedETagProperty  *response.PropertyMetadata
	namespace           string
}

// newMetadataAdapter creates a new metadataAdapter with pre-computed cached properties
func newMetadataAdapter(metadata *metadata.EntityMetadata, namespace string) *metadataAdapter {
	adapter := &metadataAdapter{
		metadata:  metadata,
		namespace: namespace,
	}

	// Pre-compute and cache all properties
	adapter.cachedProperties = make([]response.PropertyMetadata, len(metadata.Properties))
	for i, p := range metadata.Properties {
		adapter.cachedProperties[i] = response.PropertyMetadata{
			Name:              p.Name,
			JsonName:          p.JsonName,
			IsNavigationProp:  p.IsNavigationProp,
			NavigationTarget:  p.NavigationTarget,
			NavigationIsArray: p.NavigationIsArray,
		}
	}

	// Pre-compute and cache key property
	if metadata.KeyProperty != nil {
		adapter.cachedKeyProperty = &response.PropertyMetadata{
			Name:              metadata.KeyProperty.Name,
			JsonName:          metadata.KeyProperty.JsonName,
			IsNavigationProp:  metadata.KeyProperty.IsNavigationProp,
			NavigationTarget:  metadata.KeyProperty.NavigationTarget,
			NavigationIsArray: metadata.KeyProperty.NavigationIsArray,
		}
	}

	// Pre-compute and cache key properties
	adapter.cachedKeyProperties = make([]response.PropertyMetadata, len(metadata.KeyProperties))
	for i, p := range metadata.KeyProperties {
		adapter.cachedKeyProperties[i] = response.PropertyMetadata{
			Name:              p.Name,
			JsonName:          p.JsonName,
			IsNavigationProp:  p.IsNavigationProp,
			NavigationTarget:  p.NavigationTarget,
			NavigationIsArray: p.NavigationIsArray,
		}
	}

	// Pre-compute and cache ETag property
	if metadata.ETagProperty != nil {
		adapter.cachedETagProperty = &response.PropertyMetadata{
			Name:              metadata.ETagProperty.Name,
			JsonName:          metadata.ETagProperty.JsonName,
			IsNavigationProp:  metadata.ETagProperty.IsNavigationProp,
			NavigationTarget:  metadata.ETagProperty.NavigationTarget,
			NavigationIsArray: metadata.ETagProperty.NavigationIsArray,
		}
	}

	return adapter
}

func (a *metadataAdapter) GetProperties() []response.PropertyMetadata {
	// Return cached properties - no allocation needed
	return a.cachedProperties
}

func (a *metadataAdapter) GetKeyProperty() *response.PropertyMetadata {
	// Return cached key property
	return a.cachedKeyProperty
}

func (a *metadataAdapter) GetKeyProperties() []response.PropertyMetadata {
	// Return cached key properties - no allocation needed
	return a.cachedKeyProperties
}

func (a *metadataAdapter) GetEntitySetName() string {
	return a.metadata.EntitySetName
}

func (a *metadataAdapter) GetETagProperty() *response.PropertyMetadata {
	// Return cached ETag property
	return a.cachedETagProperty
}

func (a *metadataAdapter) GetNamespace() string {
	if strings.TrimSpace(a.namespace) == "" {
		return defaultNamespace
	}
	return a.namespace
}

// ValidateODataVersion checks if the OData-MaxVersion header is compatible with OData v4.0
// Returns true if the version is acceptable, false if it should be rejected
func ValidateODataVersion(r *http.Request) bool {
	maxVersion := r.Header.Get(HeaderODataMaxVersion)

	// If no OData-MaxVersion header is present, accept the request
	if maxVersion == "" {
		return true
	}

	// Parse the version string (e.g., "4.0", "3.0", "4.01")
	// We need to check if it's below 4.0
	majorVersion, _ := parseVersion(maxVersion)

	// Accept if major version is >= 4
	if majorVersion >= 4 {
		return true
	}

	// Reject if major version is < 4
	return false
}

// parseVersion parses a version string like "4.0" or "3.0" into major and minor components
func parseVersion(version string) (int, int) {
	version = strings.TrimSpace(version)
	parts := strings.Split(version, ".")

	if len(parts) == 0 {
		return 0, 0
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0
	}

	minor := 0
	if len(parts) > 1 {
		// Ignore error - if minor version can't be parsed, default to 0
		minor, _ = strconv.Atoi(parts[1]) //nolint:errcheck
	}

	return major, minor
}

// buildEntityIDFromValue builds the @odata.id value from an entity value
func (h *EntityHandler) buildEntityIDFromValue(entityValue reflect.Value, r *http.Request) string {
	if r == nil {
		return ""
	}
	baseURL := response.BuildBaseURL(r)
	keySegment := h.buildKeySegmentFromEntity(entityValue)
	if keySegment == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s(%s)", baseURL, h.metadata.EntitySetName, keySegment)
}

// buildEntityIDFromMap builds the @odata.id value from a map entity
func (h *EntityHandler) buildEntityIDFromMap(entityMap map[string]interface{}, r *http.Request) string {
	if r == nil {
		return ""
	}
	baseURL := response.BuildBaseURL(r)

	// Build key segment from map
	keyProps := h.metadata.KeyProperties
	if len(keyProps) == 0 {
		return ""
	}

	// Single key
	if len(keyProps) == 1 {
		if keyValue, exists := entityMap[keyProps[0].JsonName]; exists && keyValue != nil {
			return fmt.Sprintf("%s/%s(%v)", baseURL, h.metadata.EntitySetName, keyValue)
		}
		return ""
	}

	// Composite keys
	var parts []string
	for _, keyProp := range keyProps {
		if keyValue, exists := entityMap[keyProp.JsonName]; exists && keyValue != nil {
			// Quote string values
			if strVal, ok := keyValue.(string); ok {
				parts = append(parts, fmt.Sprintf("%s='%s'", keyProp.JsonName, strVal))
			} else {
				parts = append(parts, fmt.Sprintf("%s=%v", keyProp.JsonName, keyValue))
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("%s/%s(%s)", baseURL, h.metadata.EntitySetName, strings.Join(parts, ","))
}
