package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// HandleNavigationProperty handles GET, HEAD, and OPTIONS requests for navigation properties (e.g., Products(1)/Descriptions)
func (h *EntityHandler) HandleNavigationProperty(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string, isRef bool) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetNavigationProperty(w, r, entityKey, navigationProperty, isRef)
	case http.MethodPut:
		if isRef {
			h.handlePutNavigationPropertyRef(w, r, entityKey, navigationProperty)
		} else {
			if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
				fmt.Sprintf("Method %s is not supported for navigation properties without $ref", r.Method)); err != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, err)
			}
		}
	case http.MethodPost:
		if isRef {
			h.handlePostNavigationPropertyRef(w, r, entityKey, navigationProperty)
		} else {
			if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
				fmt.Sprintf("Method %s is not supported for navigation properties without $ref", r.Method)); err != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, err)
			}
		}
	case http.MethodDelete:
		if isRef {
			h.handleDeleteNavigationPropertyRef(w, r, entityKey, navigationProperty)
		} else {
			if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
				fmt.Sprintf("Method %s is not supported for navigation properties without $ref", r.Method)); err != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, err)
			}
		}
	case http.MethodOptions:
		if isRef {
			h.handleOptionsNavigationPropertyRef(w)
		} else {
			h.handleOptionsNavigationProperty(w)
		}
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for navigation properties", r.Method)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
	}
}

// HandleNavigationPropertyCount handles GET, HEAD, and OPTIONS requests for navigation property count (e.g., Products(1)/Descriptions/$count)
func (h *EntityHandler) HandleNavigationPropertyCount(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetNavigationPropertyCount(w, r, entityKey, navigationProperty)
	case http.MethodOptions:
		h.handleOptionsNavigationPropertyCount(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for navigation property $count", r.Method)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
	}
}

// handleGetNavigationProperty handles GET requests for navigation properties
func (h *EntityHandler) handleGetNavigationProperty(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string, isRef bool) {
	// Parse the navigation property to extract any key (e.g., RelatedProducts(2))
	navPropName, targetKey := h.parseNavigationPropertyWithKey(navigationProperty)

	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navPropName)
	if navProp == nil {
		if err := response.WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navPropName, h.metadata.EntitySetName)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// If a target key is specified for a collection navigation property (e.g., RelatedProducts(2))
	// this means we're accessing a specific item from the collection
	if targetKey != "" && navProp.NavigationIsArray {
		h.handleNavigationCollectionItem(w, r, entityKey, navProp, targetKey, isRef)
		return
	}

	// For collection navigation properties, check if query options are present
	// If so, we need to query the collection separately to apply filters, etc.
	if navProp.NavigationIsArray && hasQueryOptions(r) {
		h.handleNavigationCollectionWithQueryOptions(w, r, entityKey, navProp, isRef)
		return
	}

	// Fetch the parent entity with the navigation property preloaded
	parent, err := h.fetchParentEntityWithNav(entityKey, navProp.Name)
	if err != nil {
		h.handleFetchError(w, err, entityKey)
		return
	}

	// Extract and write the navigation property value
	navFieldValue := h.extractNavigationField(parent, navProp.Name)
	if !navFieldValue.IsValid() {
		if err := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			"Could not access navigation property"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	if isRef {
		h.writeNavigationRefResponse(w, r, entityKey, navProp, navFieldValue)
	} else {
		h.writeNavigationResponse(w, r, entityKey, navProp, navFieldValue)
	}
}

// handleOptionsNavigationProperty handles OPTIONS requests for navigation properties (without $ref)
func (h *EntityHandler) handleOptionsNavigationProperty(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

// handleOptionsNavigationPropertyRef handles OPTIONS requests for navigation properties with $ref
func (h *EntityHandler) handleOptionsNavigationPropertyRef(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, PUT, POST, DELETE, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

// handleNavigationCollectionWithQueryOptions handles collection navigation properties with query options
// This method queries the related collection directly to properly apply filters, orderby, etc.
func (h *EntityHandler) handleNavigationCollectionWithQueryOptions(w http.ResponseWriter, r *http.Request, entityKey string, navProp *metadata.PropertyMetadata, isRef bool) {
	// Get the target entity metadata
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err)); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Parse query options using the target entity's metadata
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), targetMetadata)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// First verify that the parent entity exists and is authorized
	parentOptions := &query.QueryOptions{}
	parentScopes, parentHookErr := callBeforeReadEntity(h.metadata, r, parentOptions)
	if parentHookErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", parentHookErr.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	parent := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}
	if len(parentScopes) > 0 {
		db = db.Scopes(parentScopes...)
	}
	if err := db.First(parent).Error; err != nil {
		h.handleFetchError(w, err, entityKey)
		return
	}

	if _, _, parentAfterErr := callAfterReadEntity(h.metadata, r, parentOptions, parent); parentAfterErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", parentAfterErr.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Build a query for the related collection
	// We need to filter by the foreign key that relates to the parent entity
	relatedDB := h.db.Model(reflect.New(targetMetadata.EntityType).Interface())

	// Extract the parent entity's key values to filter the related collection
	parentValue := reflect.ValueOf(parent).Elem()

	// Build foreign key constraints
	// This assumes standard GORM conventions: ParentID field in child references ID in parent
	// For the Product -> ProductDescriptions example: ProductDescription.ProductID = Product.ID
	// GORM uses snake_case for column names, so ProductID becomes product_id
	for _, keyProp := range h.metadata.KeyProperties {
		keyFieldValue := parentValue.FieldByName(keyProp.Name)
		if keyFieldValue.IsValid() {
			// Build the foreign key column name using GORM's naming convention
			// EntityName + KeyProperty name, converted to snake_case
			foreignKeyFieldName := fmt.Sprintf("%s%s", h.metadata.EntityName, keyProp.Name)
			foreignKeyColumnName := toSnakeCase(foreignKeyFieldName)
			relatedDB = relatedDB.Where(fmt.Sprintf("%s = ?", foreignKeyColumnName), keyFieldValue.Interface())
		}
	}

	// Apply BeforeReadCollection hooks for the target entity
	targetScopes, targetHookErr := callBeforeReadCollection(targetMetadata, r, queryOptions)
	if targetHookErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", targetHookErr.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}
	if len(targetScopes) > 0 {
		relatedDB = relatedDB.Scopes(targetScopes...)
	}

	// Get total count if $count=true
	var totalCount *int64
	if queryOptions.Count {
		var count int64
		countDB := relatedDB
		// Apply filter to count query if present
		if queryOptions.Filter != nil {
			countDB = query.ApplyFilterOnly(countDB, queryOptions.Filter, targetMetadata)
		}
		if err := countDB.Count(&count).Error; err != nil {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}
		totalCount = &count
	}

	// Apply query options to the database query
	relatedDB = query.ApplyQueryOptions(relatedDB, queryOptions, targetMetadata)

	// Fetch the results
	resultsSlice := reflect.New(reflect.SliceOf(targetMetadata.EntityType)).Interface()
	if err := relatedDB.Find(resultsSlice).Error; err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Calculate next link if pagination is active
	sliceValue := reflect.ValueOf(resultsSlice).Elem()
	resultsData := sliceValue.Interface()
	var nextLink *string
	if queryOptions.Top != nil && sliceValue.Len() > *queryOptions.Top {
		// Trim to the requested top count
		trimmedSlice := reflect.MakeSlice(sliceValue.Type(), *queryOptions.Top, *queryOptions.Top)
		reflect.Copy(trimmedSlice, sliceValue.Slice(0, *queryOptions.Top))
		sliceValue = trimmedSlice
		resultsData = sliceValue.Interface()

		// Build next link
		baseURL := response.BuildBaseURL(r)
		navigationPath := fmt.Sprintf("%s(%s)/%s", h.metadata.EntitySetName, entityKey, navProp.JsonName)
		nextURL := buildNextLink(baseURL, navigationPath, queryOptions)
		nextLink = &nextURL
	}

	if override, hasOverride, afterErr := callAfterReadCollection(targetMetadata, r, queryOptions, resultsData); afterErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", afterErr.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	} else if hasOverride {
		resultsData = override
	}

	// Write response
	navigationPath := fmt.Sprintf("%s(%s)/%s", h.metadata.EntitySetName, entityKey, navProp.JsonName)

	if isRef {
		h.writeNavigationCollectionRefFromData(w, r, targetMetadata, resultsData, totalCount, nextLink)
	} else {
		if err := response.WriteODataCollection(w, r, navigationPath, resultsData, totalCount, nextLink); err != nil {
			fmt.Printf("Error writing navigation property collection: %v\n", err)
		}
	}
}

// handleNavigationCollectionItem handles accessing a specific item from a collection navigation property
// Example: GET Products(1)/RelatedProducts(2) or GET Products(1)/RelatedProducts(2)/$ref
func (h *EntityHandler) handleNavigationCollectionItem(w http.ResponseWriter, r *http.Request, entityKey string, navProp *metadata.PropertyMetadata, targetKey string, isRef bool) {
	// Get the target entity metadata
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err)); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// First verify that the parent entity exists
	parent := reflect.New(h.metadata.EntityType).Interface()
	parentDB, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}
	
	// Preload the navigation property to verify the relationship
	parentDB = parentDB.Preload(navProp.Name)
	if err := parentDB.First(parent).Error; err != nil {
		h.handleFetchError(w, err, entityKey)
		return
	}

	// Extract the navigation property value to verify the relationship exists
	parentValue := reflect.ValueOf(parent).Elem()
	navFieldValue := parentValue.FieldByName(navProp.Name)
	if !navFieldValue.IsValid() {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			"Could not access navigation property"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Check if the target key exists in the collection
	found := false
	var targetEntity interface{}
	
	if navFieldValue.Kind() == reflect.Slice {
		for i := 0; i < navFieldValue.Len(); i++ {
			item := navFieldValue.Index(i)
			// Extract the key from this item
			if len(targetMetadata.KeyProperties) == 1 {
				// Single key property
				keyProp := targetMetadata.KeyProperties[0]
				itemKeyValue := item.FieldByName(keyProp.Name)
				if itemKeyValue.IsValid() {
					// Convert key value to string for comparison
					itemKeyStr := fmt.Sprintf("%v", itemKeyValue.Interface())
					if itemKeyStr == targetKey {
						found = true
						targetEntity = item.Interface()
						break
					}
				}
			}
			// TODO: Handle composite keys if needed
		}
	}

	if !found {
		if writeErr := response.WriteError(w, http.StatusNotFound, "Entity not found",
			fmt.Sprintf("Entity with key '%s' is not related to the parent entity via '%s'", targetKey, navProp.JsonName)); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Write the response
	if isRef {
		// Write a single entity reference
		// WriteEntityReference expects just the entity path (e.g., "Products(2)"), not the full URL
		entityPath := fmt.Sprintf("%s(%s)", targetMetadata.EntitySetName, targetKey)
		if err := response.WriteEntityReference(w, r, entityPath); err != nil {
			fmt.Printf("Error writing entity reference: %v\n", err)
		}
	} else {
		// Write the full entity
		navigationPath := fmt.Sprintf("%s(%s)/%s(%s)", h.metadata.EntitySetName, entityKey, navProp.JsonName, targetKey)
		if err := response.WriteODataCollection(w, r, navigationPath, []interface{}{targetEntity}, nil, nil); err != nil {
			fmt.Printf("Error writing navigation property entity: %v\n", err)
		}
	}
}

// handleGetNavigationPropertyCount handles GET requests for navigation property count
func (h *EntityHandler) handleGetNavigationPropertyCount(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string) {
	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navigationProperty)
	if navProp == nil {
		if err := response.WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navigationProperty, h.metadata.EntitySetName)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// $count is only valid for collection navigation properties
	if !navProp.NavigationIsArray {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request",
			fmt.Sprintf("$count is only supported on collection navigation properties. '%s' is a single-valued navigation property.", navigationProperty)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// Fetch the parent entity with the navigation property preloaded
	parent, err := h.fetchParentEntityWithNav(entityKey, navProp.Name)
	if err != nil {
		h.handleFetchError(w, err, entityKey)
		return
	}

	// Extract the navigation property value
	navFieldValue := h.extractNavigationField(parent, navProp.Name)
	if !navFieldValue.IsValid() {
		if err := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			"Could not access navigation property"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// Get the count of the navigation collection
	var count int64
	if navFieldValue.Kind() == reflect.Slice {
		count = int64(navFieldValue.Len())
	}

	// Write the count as plain text according to OData v4 spec
	w.Header().Set(HeaderContentType, "text/plain")
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

	if _, err := fmt.Fprintf(w, "%d", count); err != nil {
		fmt.Printf("Error writing count response: %v\n", err)
	}
}

// handleOptionsNavigationPropertyCount handles OPTIONS requests for navigation property count
func (h *EntityHandler) handleOptionsNavigationPropertyCount(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

// findNavigationProperty finds a navigation property by name in the entity metadata
func (h *EntityHandler) findNavigationProperty(navigationProperty string) *metadata.PropertyMetadata {
	return h.metadata.FindNavigationProperty(navigationProperty)
}

// findStructuralProperty finds a structural (non-navigation) property by name in the entity metadata
func (h *EntityHandler) findStructuralProperty(propertyName string) *metadata.PropertyMetadata {
	return h.metadata.FindStructuralProperty(propertyName)
}

// IsNavigationProperty checks if a property name is a navigation property
func (h *EntityHandler) IsNavigationProperty(propertyName string) bool {
	// Parse out any key from the property name (e.g., "RelatedProducts(2)" -> "RelatedProducts")
	navPropName, _ := h.parseNavigationPropertyWithKey(propertyName)
	return h.findNavigationProperty(navPropName) != nil
}

// IsStructuralProperty checks if a property name is a structural property
func (h *EntityHandler) IsStructuralProperty(propertyName string) bool {
	return h.findStructuralProperty(propertyName) != nil
}

// findComplexTypeProperty finds a complex type property by name in the entity metadata
func (h *EntityHandler) findComplexTypeProperty(propertyName string) *metadata.PropertyMetadata {
	return h.metadata.FindComplexTypeProperty(propertyName)
}

// IsComplexTypeProperty checks if a property name is a complex type property
func (h *EntityHandler) IsComplexTypeProperty(propertyName string) bool {
	return h.findComplexTypeProperty(propertyName) != nil
}

// HandleComplexTypeProperty handles GET, HEAD, and OPTIONS requests for complex type properties (e.g., Products(1)/ShippingAddress)
// propertySegments represents the navigation path segments without $value/$ref/$count (e.g., ["ShippingAddress", "City"]).
func (h *EntityHandler) HandleComplexTypeProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertySegments []string, isValue bool) {
	if len(propertySegments) == 0 {
		h.writePropertyNotFoundError(w, "")
		return
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetComplexTypeProperty(w, r, entityKey, propertySegments, isValue)
	case http.MethodOptions:
		h.handleOptionsComplexTypeProperty(w)
	default:
		h.writeMethodNotAllowedError(w, r.Method, "complex property access")
	}
}

// handleGetComplexTypeProperty resolves and writes a complex type property response
func (h *EntityHandler) handleGetComplexTypeProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertySegments []string, isValue bool) {
	rootName := propertySegments[0]
	complexProp := h.findComplexTypeProperty(rootName)
	if complexProp == nil {
		h.writePropertyNotFoundError(w, rootName)
		return
	}

	if len(propertySegments) == 1 && isValue {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request",
			"$value is not supported on complex properties"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	fieldValue, err := h.fetchComplexPropertyValue(w, entityKey, complexProp)
	if err != nil {
		return
	}

	// Handle root null complex property
	if isNilPointer(fieldValue) {
		h.writeNoContentForNullComplex(w, r)
		return
	}

	// Prepare traversal state
	contextSegments := []string{complexProp.JsonName}
	currentValue := dereferenceValue(fieldValue)
	currentType := dereferenceType(complexProp.Type)

	// Traverse nested segments, if any
	for idx, segment := range propertySegments[1:] {
		resolved, resolvedType, resolvedJSONName, ok := resolveStructField(currentValue, segment)
		if !ok {
			// Support maps as well as structs
			if currentValue.Kind() == reflect.Map {
				resolved, resolvedType, resolvedJSONName, ok = resolveMapField(currentValue, segment)
			}
		}

		if !ok {
			h.writePropertyNotFoundError(w, segment)
			return
		}

		contextSegments = append(contextSegments, resolvedJSONName)
		currentValue = resolved
		currentType = resolvedType

		// If there are more segments to traverse, ensure we can continue
		if idx < len(propertySegments[1:])-1 {
			if isNilPointer(currentValue) {
				h.writeComplexSegmentNullError(w, contextSegments)
				return
			}
			currentValue = dereferenceValue(currentValue)
			currentType = dereferenceType(currentType)
		}
	}

	h.writeResolvedComplexValue(w, r, entityKey, contextSegments, currentValue, currentType, isValue)
}

// handleOptionsComplexTypeProperty handles OPTIONS requests for complex properties
func (h *EntityHandler) handleOptionsComplexTypeProperty(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

// HandleStructuralProperty handles GET and OPTIONS requests for structural properties (e.g., Products(1)/Name)
// When isValue is true, returns the raw property value without JSON wrapper (e.g., Products(1)/Name/$value)
func (h *EntityHandler) HandleStructuralProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertyName string, isValue bool) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetStructuralProperty(w, r, entityKey, propertyName, isValue)
	case http.MethodOptions:
		h.handleOptionsStructuralProperty(w)
	default:
		h.writeMethodNotAllowedError(w, r.Method, "property access")
	}
}

// handleGetStructuralProperty handles GET requests for structural properties
func (h *EntityHandler) handleGetStructuralProperty(w http.ResponseWriter, r *http.Request, entityKey string, propertyName string, isValue bool) {
	// Find and validate the structural property
	prop := h.findStructuralProperty(propertyName)
	if prop == nil {
		h.writePropertyNotFoundError(w, propertyName)
		return
	}

	// Fetch property value
	fieldValue, err := h.fetchPropertyValue(w, entityKey, prop)
	if err != nil {
		return // Error already written
	}

	// Write response
	if isValue {
		h.writeRawPropertyValue(w, r, prop, fieldValue)
	} else {
		h.writePropertyResponse(w, r, entityKey, prop, fieldValue)
	}
}

// handleOptionsStructuralProperty handles OPTIONS requests for structural properties
func (h *EntityHandler) handleOptionsStructuralProperty(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

// fetchComplexPropertyValue fetches a complex property value from an entity without applying select clauses
func (h *EntityHandler) fetchComplexPropertyValue(w http.ResponseWriter, entityKey string, prop *metadata.PropertyMetadata) (reflect.Value, error) {
	entity := reflect.New(h.metadata.EntityType).Interface()
	
	var db *gorm.DB
	var err error
	
	// Handle singleton case where entityKey is empty
	if h.metadata.IsSingleton && entityKey == "" {
		// For singletons, we don't use a key query, just fetch the first (and only) record
		db = h.db
	} else {
		// For regular entities, build the key query
		db, err = h.buildKeyQuery(entityKey)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return reflect.Value{}, err
		}
	}

	if err := db.First(entity).Error; err != nil {
		h.handlePropertyFetchError(w, err, entityKey)
		return reflect.Value{}, err
	}

	entityValue := reflect.ValueOf(entity).Elem()
	fieldValue := entityValue.FieldByName(prop.Name)
	if !fieldValue.IsValid() {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			"Could not access complex property"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return reflect.Value{}, fmt.Errorf("invalid field")
	}

	return fieldValue, nil
}

// writeNoContentForNullComplex writes a 204 No Content response for null complex properties
func (h *EntityHandler) writeNoContentForNullComplex(w http.ResponseWriter, r *http.Request) {
	metadataLevel := response.GetODataMetadataLevel(r)
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusNoContent)
}

// writeComplexSegmentNullError writes an error when a nested complex segment is null
func (h *EntityHandler) writeComplexSegmentNullError(w http.ResponseWriter, contextSegments []string) {
	path := strings.Join(contextSegments, "/")
	if err := response.WriteError(w, http.StatusNotFound, "Property not found",
		fmt.Sprintf("Complex property path '%s' is null", path)); err != nil {
		fmt.Printf(LogMsgErrorWritingErrorResponse, err)
	}
}

// writeResolvedComplexValue writes the resolved complex property (or nested primitive) response
func (h *EntityHandler) writeResolvedComplexValue(w http.ResponseWriter, r *http.Request, entityKey string, contextSegments []string, value reflect.Value, valueType reflect.Type, isValue bool) {
	if isComplexValue(value, valueType) {
		if isValue {
			if err := response.WriteError(w, http.StatusBadRequest, "Invalid request",
				"$value is not supported on complex properties"); err != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, err)
			}
			return
		}

		if isNilPointer(value) {
			h.writeNoContentForNullComplex(w, r)
			return
		}

		resolvedValue := dereferenceValue(value)
		h.writeComplexValueResponse(w, r, entityKey, contextSegments, resolvedValue)
		return
	}

	// Primitive value resolution
	resolvedValue := dereferenceValue(value)

	if isValue {
		h.writeRawPrimitiveValue(w, r, resolvedValue)
		return
	}

	h.writePrimitiveComplexPropertyResponse(w, r, entityKey, contextSegments, value)
}

// writeComplexValueResponse serializes a complex value (struct or map) with @odata.context
func (h *EntityHandler) writeComplexValueResponse(w http.ResponseWriter, r *http.Request, entityKey string, contextSegments []string, value reflect.Value) {
	metadataLevel := response.GetODataMetadataLevel(r)
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	contextPath := strings.Join(contextSegments, "/")
	responseMap := make(map[string]interface{})
	if metadataLevel != "none" {
		contextURL := fmt.Sprintf("%s/$metadata#%s(%s)/%s", response.BuildBaseURL(r), h.metadata.EntitySetName, entityKey, contextPath)
		responseMap[ODataContextProperty] = contextURL
	}

	switch value.Kind() {
	case reflect.Struct:
		structMap := structValueToMap(value)
		for k, v := range structMap {
			responseMap[k] = v
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			key := iter.Key()
			if key.Kind() == reflect.String {
				responseMap[key.String()] = iter.Value().Interface()
			}
		}
	default:
		responseMap["value"] = value.Interface()
	}

	if err := json.NewEncoder(w).Encode(responseMap); err != nil {
		fmt.Printf("Error writing complex property response: %v\n", err)
	}
}

// writePrimitiveComplexPropertyResponse writes a primitive property (nested within a complex type)
func (h *EntityHandler) writePrimitiveComplexPropertyResponse(w http.ResponseWriter, r *http.Request, entityKey string, contextSegments []string, value reflect.Value) {
	metadataLevel := response.GetODataMetadataLevel(r)
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	var valueInterface interface{}
	if !value.IsValid() {
		valueInterface = nil
	} else if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			valueInterface = nil
		} else {
			valueInterface = value.Elem().Interface()
		}
	} else {
		valueInterface = value.Interface()
	}

	responseBody := map[string]interface{}{
		"value": valueInterface,
	}

	if metadataLevel != "none" {
		contextURL := fmt.Sprintf("%s/$metadata#%s(%s)/%s", response.BuildBaseURL(r), h.metadata.EntitySetName, entityKey, strings.Join(contextSegments, "/"))
		responseBody[ODataContextProperty] = contextURL
	}

	if err := json.NewEncoder(w).Encode(responseBody); err != nil {
		fmt.Printf("Error writing complex primitive property response: %v\n", err)
	}
}

// writeRawPrimitiveValue writes a primitive value in raw form for $value requests
func (h *EntityHandler) writeRawPrimitiveValue(w http.ResponseWriter, r *http.Request, value reflect.Value) {
	if !value.IsValid() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch value.Kind() {
	case reflect.String:
		w.Header().Set(HeaderContentType, ContentTypePlainText)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		w.Header().Set(HeaderContentType, ContentTypePlainText)
	default:
		w.Header().Set(HeaderContentType, "application/octet-stream")
	}

	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	if _, err := fmt.Fprintf(w, "%v", value.Interface()); err != nil {
		fmt.Printf("Error writing raw primitive value: %v\n", err)
	}
}

// resolveStructField resolves a field from a struct by segment name, returning the reflect.Value, type, and canonical JSON name
func resolveStructField(value reflect.Value, segment string) (reflect.Value, reflect.Type, string, bool) {
	if value.Kind() != reflect.Struct {
		return reflect.Value{}, nil, "", false
	}

	valueType := value.Type()
	lowerSegment := strings.ToLower(segment)
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonName := getJSONFieldName(field)
		candidates := []string{field.Name, jsonName}
		for _, candidate := range candidates {
			if candidate == "" || candidate == "-" {
				continue
			}
			if candidate == segment || strings.ToLower(candidate) == lowerSegment {
				return value.Field(i), field.Type, jsonName, true
			}
		}
	}

	return reflect.Value{}, nil, "", false
}

// resolveMapField resolves a value from a map with string keys
func resolveMapField(value reflect.Value, segment string) (reflect.Value, reflect.Type, string, bool) {
	if value.Kind() != reflect.Map || value.Type().Key().Kind() != reflect.String {
		return reflect.Value{}, nil, "", false
	}

	mapValue := value.MapIndex(reflect.ValueOf(segment))
	if mapValue.IsValid() {
		return mapValue, mapValue.Type(), segment, true
	}

	lowerSegment := strings.ToLower(segment)
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key()
		if strings.ToLower(key.String()) == lowerSegment {
			val := iter.Value()
			return val, val.Type(), key.String(), true
		}
	}

	return reflect.Value{}, nil, "", false
}

// isNilPointer checks if a value is a nil pointer or interface
func isNilPointer(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	switch value.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// dereferenceValue dereferences pointer values until a non-pointer is reached
func dereferenceValue(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

// dereferenceType dereferences pointer types until a non-pointer is reached
func dereferenceType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// isComplexValue determines if a resolved value should be treated as a complex object
func isComplexValue(value reflect.Value, valueType reflect.Type) bool {
	if !value.IsValid() {
		if valueType == nil {
			return false
		}
		value = reflect.New(dereferenceType(valueType)).Elem()
	}

	if value.Kind() == reflect.Map {
		return true
	}

	resolvedType := valueType
	if resolvedType == nil {
		resolvedType = value.Type()
	}

	resolvedType = dereferenceType(resolvedType)
	if resolvedType == nil {
		return false
	}

	if resolvedType.Kind() != reflect.Struct {
		return false
	}

	// Treat time.Time (and similar) as primitive despite being structs
	if resolvedType.PkgPath() == "time" && resolvedType.Name() == "Time" {
		return false
	}

	return true
}

// structValueToMap converts a struct to a map using JSON tag names
func structValueToMap(value reflect.Value) map[string]interface{} {
	result := make(map[string]interface{})
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonName := getJSONFieldName(field)
		if jsonName == "-" {
			continue
		}
		if jsonName == "" {
			jsonName = field.Name
		}

		result[jsonName] = value.Field(i).Interface()
	}
	return result
}

// getJSONFieldName extracts the JSON field name from a struct field
func getJSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}

	parts := strings.Split(tag, ",")
	if len(parts) == 0 || parts[0] == "" {
		return field.Name
	}

	return parts[0]
}

// writeMethodNotAllowedError writes a method not allowed error for a specific context
func (h *EntityHandler) writeMethodNotAllowedError(w http.ResponseWriter, method, context string) {
	if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
		fmt.Sprintf("Method %s is not supported for %s", method, context)); err != nil {
		fmt.Printf(LogMsgErrorWritingErrorResponse, err)
	}
}

// writePropertyNotFoundError writes a property not found error
func (h *EntityHandler) writePropertyNotFoundError(w http.ResponseWriter, propertyName string) {
	if err := response.WriteError(w, http.StatusNotFound, "Property not found",
		fmt.Sprintf("'%s' is not a valid property for %s", propertyName, h.metadata.EntitySetName)); err != nil {
		fmt.Printf(LogMsgErrorWritingErrorResponse, err)
	}
}

// fetchPropertyValue fetches a property value from an entity
func (h *EntityHandler) fetchPropertyValue(w http.ResponseWriter, entityKey string, prop *metadata.PropertyMetadata) (reflect.Value, error) {
	entity := reflect.New(h.metadata.EntityType).Interface()
	
	var db *gorm.DB
	var err error
	
	// Handle singleton case where entityKey is empty
	if h.metadata.IsSingleton && entityKey == "" {
		// For singletons, we don't use a key query, just fetch the first (and only) record
		db = h.db
	} else {
		// For regular entities, build the key query
		db, err = h.buildKeyQuery(entityKey)
		if err != nil {
			if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return reflect.Value{}, err
		}
	}

	db = h.applyStructuralPropertySelect(db, prop)

	if err := db.First(entity).Error; err != nil {
		h.handlePropertyFetchError(w, err, entityKey)
		return reflect.Value{}, err
	}

	entityValue := reflect.ValueOf(entity).Elem()
	fieldValue := entityValue.FieldByName(prop.Name)
	if !fieldValue.IsValid() {
		if err := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			"Could not access property"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return reflect.Value{}, fmt.Errorf("invalid field")
	}

	return fieldValue, nil
}

// handlePropertyFetchError handles errors when fetching a property
func (h *EntityHandler) handlePropertyFetchError(w http.ResponseWriter, err error, entityKey string) {
	if err == gorm.ErrRecordNotFound {
		var errorMessage string
		if h.metadata.IsSingleton {
			errorMessage = fmt.Sprintf("Singleton '%s' not found", h.metadata.SingletonName)
		} else {
			errorMessage = fmt.Sprintf("Entity with key '%s' not found", entityKey)
		}
		
		if writeErr := response.WriteError(w, http.StatusNotFound, ErrMsgEntityNotFound, errorMessage); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
	} else {
		h.writeDatabaseError(w, err)
	}
}

// writePropertyResponse writes a property response with OData context
func (h *EntityHandler) writePropertyResponse(w http.ResponseWriter, r *http.Request, entityKey string, prop *metadata.PropertyMetadata, fieldValue reflect.Value) {
	// Get metadata level to determine which fields to include
	metadataLevel := response.GetODataMetadataLevel(r)

	odataResponse := map[string]interface{}{
		"value": fieldValue.Interface(),
	}

	// Only include @odata.context for minimal and full metadata (not for none)
	if metadataLevel != "none" {
		contextURL := fmt.Sprintf("%s/$metadata#%s(%s)/%s", response.BuildBaseURL(r), h.metadata.EntitySetName, entityKey, prop.JsonName)
		odataResponse[ODataContextProperty] = contextURL
	}

	// Set Content-Type with dynamic metadata level
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		fmt.Printf("Error writing property response: %v\n", err)
	}
}

// writeRawPropertyValue writes a property value in raw format for /$value requests
func (h *EntityHandler) writeRawPropertyValue(w http.ResponseWriter, r *http.Request, prop *metadata.PropertyMetadata, fieldValue reflect.Value) {
	// Set appropriate content type based on the value type
	valueInterface := fieldValue.Interface()

	// Check for binary data ([]byte) first
	if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.Uint8 {
		// Binary data - set appropriate content type and write raw bytes
		// Use custom content type if specified, otherwise default to application/octet-stream
		contentType := "application/octet-stream"
		if prop.ContentType != "" {
			contentType = prop.ContentType
		}
		w.Header().Set(HeaderContentType, contentType)
		w.WriteHeader(http.StatusOK)

		// For HEAD requests, don't write the body
		if r.Method == http.MethodHead {
			return
		}

		// Write raw binary data
		if byteData, ok := valueInterface.([]byte); ok {
			if _, err := w.Write(byteData); err != nil {
				fmt.Printf("Error writing binary value: %v\n", err)
			}
		}
		return
	}

	// Determine content type based on the property type
	switch fieldValue.Kind() {
	case reflect.String:
		w.Header().Set(HeaderContentType, ContentTypePlainText)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		w.Header().Set(HeaderContentType, ContentTypePlainText)
	case reflect.Bool:
		w.Header().Set(HeaderContentType, ContentTypePlainText)
	default:
		// For other types, use application/octet-stream
		w.Header().Set(HeaderContentType, "application/octet-stream")
	}

	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

	// Write the raw value
	if _, err := fmt.Fprintf(w, "%v", valueInterface); err != nil {
		fmt.Printf("Error writing raw value: %v\n", err)
	}
}

// applyStructuralPropertySelect applies SELECT clause to fetch only the structural property and key columns
func (h *EntityHandler) applyStructuralPropertySelect(db *gorm.DB, prop *metadata.PropertyMetadata) *gorm.DB {
	// Build select columns list: property + all key properties
	// Use struct field names - GORM will handle column name conversion
	selectColumns := []string{prop.Name}
	for _, keyProp := range h.metadata.KeyProperties {
		// Avoid duplicates if the property itself is a key
		if keyProp.Name != prop.Name {
			selectColumns = append(selectColumns, keyProp.Name)
		}
	}
	return db.Select(selectColumns)
}

// fetchParentEntityWithNav fetches the parent entity and preloads the specified navigation property
func (h *EntityHandler) fetchParentEntityWithNav(entityKey, navPropertyName string) (interface{}, error) {
	parent := reflect.New(h.metadata.EntityType).Interface()

	var db *gorm.DB
	var err error
	
	// Handle singleton case where entityKey is empty
	if h.metadata.IsSingleton && entityKey == "" {
		// For singletons, we don't use a key query, just fetch the first (and only) record
		db = h.db
	} else {
		// For regular entities, build the key query
		db, err = h.buildKeyQuery(entityKey)
		if err != nil {
			return nil, err
		}
	}

	db = db.Preload(navPropertyName)
	return parent, db.First(parent).Error
}

// extractNavigationField extracts the navigation property field value from the parent entity
func (h *EntityHandler) extractNavigationField(parent interface{}, navPropertyName string) reflect.Value {
	parentValue := reflect.ValueOf(parent).Elem()
	return parentValue.FieldByName(navPropertyName)
}

// writeNavigationResponse writes the navigation property response (collection or single entity)
func (h *EntityHandler) writeNavigationResponse(w http.ResponseWriter, r *http.Request, entityKey string, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	if navProp.NavigationIsArray {
		h.writeNavigationCollection(w, r, entityKey, navProp, navFieldValue)
	} else {
		h.writeSingleNavigationEntity(w, r, entityKey, navProp, navFieldValue)
	}
}

// writeNavigationCollection writes a collection navigation property response
func (h *EntityHandler) writeNavigationCollection(w http.ResponseWriter, r *http.Request, entityKey string, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	navData := navFieldValue.Interface()
	// Build the navigation path according to OData V4 spec: EntitySet(key)/NavigationProperty
	navigationPath := fmt.Sprintf(ODataEntityKeyFormat, h.metadata.EntitySetName, entityKey)
	navigationPath = fmt.Sprintf("%s/%s", navigationPath, navProp.JsonName)
	if err := response.WriteODataCollection(w, r, navigationPath, navData, nil, nil); err != nil {
		fmt.Printf("Error writing navigation property collection: %v\n", err)
	}
}

// writeSingleNavigationEntity writes a single navigation property entity response
func (h *EntityHandler) writeSingleNavigationEntity(w http.ResponseWriter, r *http.Request, entityKey string, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	navData := navFieldValue.Interface()
	navValue := reflect.ValueOf(navData)

	// Handle pointer and check for nil
	if navValue.Kind() == reflect.Ptr {
		if navValue.IsNil() {
			// Set Content-Type with dynamic metadata level even for 204 responses
			metadataLevel := response.GetODataMetadataLevel(r)
			w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		navValue = navValue.Elem()
	}

	// Get metadata level
	metadataLevel := response.GetODataMetadataLevel(r)

	// Build the OData response with navigation path according to OData V4 spec: EntitySet(key)/NavigationProperty/$entity
	navigationPath := fmt.Sprintf(ODataEntityKeyFormat, h.metadata.EntitySetName, entityKey)
	navigationPath = fmt.Sprintf("%s/%s", navigationPath, navProp.JsonName)
	contextURL := fmt.Sprintf("%s/$metadata#%s/$entity", response.BuildBaseURL(r), navigationPath)
	odataResponse := h.buildEntityResponseWithMetadata(navValue, contextURL, metadataLevel)

	// Set Content-Type with dynamic metadata level
	w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

	if err := json.NewEncoder(w).Encode(odataResponse); err != nil {
		fmt.Printf("Error writing navigation property response: %v\n", err)
	}
}

// writeNavigationRefResponse writes entity reference(s) for navigation properties
func (h *EntityHandler) writeNavigationRefResponse(w http.ResponseWriter, r *http.Request, entityKey string, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	if navProp.NavigationIsArray {
		h.writeNavigationCollectionRef(w, r, navProp, navFieldValue)
	} else {
		h.writeSingleNavigationRef(w, r, entityKey, navProp, navFieldValue)
	}
}

// writeNavigationCollectionRef writes entity references for a collection navigation property
func (h *EntityHandler) writeNavigationCollectionRef(w http.ResponseWriter, r *http.Request, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	// Get the target entity metadata to extract keys
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err)); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Build entity IDs for each entity in the collection
	var entityIDs []string
	if navFieldValue.Kind() == reflect.Slice {
		for i := 0; i < navFieldValue.Len(); i++ {
			entity := navFieldValue.Index(i).Interface()
			keyValues := response.ExtractEntityKeys(entity, targetMetadata.KeyProperties)
			entityID := response.BuildEntityID(targetMetadata.EntitySetName, keyValues)
			entityIDs = append(entityIDs, entityID)
		}
	}

	if err := response.WriteEntityReferenceCollection(w, r, entityIDs, nil, nil); err != nil {
		fmt.Printf("Error writing entity reference collection: %v\n", err)
	}
}

// writeSingleNavigationRef writes an entity reference for a single navigation property
func (h *EntityHandler) writeSingleNavigationRef(w http.ResponseWriter, r *http.Request, _ string, navProp *metadata.PropertyMetadata, navFieldValue reflect.Value) {
	navData := navFieldValue.Interface()
	navValue := reflect.ValueOf(navData)

	// Handle pointer and check for nil
	if navValue.Kind() == reflect.Ptr {
		if navValue.IsNil() {
			// Set Content-Type with dynamic metadata level even for 204 responses
			metadataLevel := response.GetODataMetadataLevel(r)
			w.Header().Set(HeaderContentType, fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
			SetODataVersionHeader(w)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		navValue = navValue.Elem()
	}

	// Get the target entity metadata to extract keys
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err)); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Extract key values and build entity ID
	keyValues := response.ExtractEntityKeys(navValue.Interface(), targetMetadata.KeyProperties)
	entityID := response.BuildEntityID(targetMetadata.EntitySetName, keyValues)

	if err := response.WriteEntityReference(w, r, entityID); err != nil {
		fmt.Printf("Error writing entity reference: %v\n", err)
	}
}

// getTargetMetadata retrieves metadata for a navigation target entity type
func (h *EntityHandler) getTargetMetadata(targetName string) (*metadata.EntityMetadata, error) {
	if h.entitiesMetadata == nil {
		return nil, fmt.Errorf("entities metadata not available")
	}

	// Try with the target name as-is (entity set name)
	if meta, ok := h.entitiesMetadata[targetName]; ok {
		return meta, nil
	}

	// Try to find by entity name
	for _, meta := range h.entitiesMetadata {
		if meta.EntityName == targetName {
			return meta, nil
		}
	}

	return nil, fmt.Errorf("metadata for target '%s' not found", targetName)
}

// hasQueryOptions checks if the request has any OData query options
func hasQueryOptions(r *http.Request) bool {
	query := r.URL.Query()
	odataOptions := []string{"$filter", "$select", "$orderby", "$top", "$skip", "$count", "$expand", "$search", "$skiptoken"}
	for _, option := range odataOptions {
		if query.Has(option) {
			return true
		}
	}
	return false
}

// buildNextLink builds a next link URL for pagination
func buildNextLink(baseURL, path string, options *query.QueryOptions) string {
	// Simple implementation - in production this should handle skiptoken properly
	if options.Skip == nil {
		skip := 0
		options.Skip = &skip
	}
	if options.Top == nil {
		top := 20
		options.Top = &top
	}
	nextSkip := *options.Skip + *options.Top
	return fmt.Sprintf("%s/%s?$skip=%d&$top=%d", baseURL, path, nextSkip, *options.Top)
}

// writeNavigationCollectionRefFromData writes entity references for a navigation collection from data
func (h *EntityHandler) writeNavigationCollectionRefFromData(w http.ResponseWriter, r *http.Request, targetMetadata *metadata.EntityMetadata, data interface{}, count *int64, nextLink *string) {
	// Build entity IDs for each entity in the collection
	var entityIDs []string

	sliceValue := reflect.ValueOf(data)
	if sliceValue.Kind() == reflect.Slice {
		for i := 0; i < sliceValue.Len(); i++ {
			entity := sliceValue.Index(i).Interface()
			keyValues := response.ExtractEntityKeys(entity, targetMetadata.KeyProperties)
			entityID := response.BuildEntityID(targetMetadata.EntitySetName, keyValues)
			entityIDs = append(entityIDs, entityID)
		}
	}

	if err := response.WriteEntityReferenceCollection(w, r, entityIDs, count, nextLink); err != nil {
		fmt.Printf("Error writing entity reference collection: %v\n", err)
	}
}

// handlePutNavigationPropertyRef handles PUT requests to update a single-valued navigation property reference
// Example: PUT Products(1)/Category/$ref with body {"@odata.id":"http://localhost:8080/Categories(2)"}
func (h *EntityHandler) handlePutNavigationPropertyRef(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string) {
	// Parse the navigation property to extract any key (though PUT shouldn't have a key in the navigation property)
	navPropName, targetKey := h.parseNavigationPropertyWithKey(navigationProperty)
	
	// If a key was provided in the navigation property name for PUT, that's an error
	if targetKey != "" {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request",
			"PUT $ref does not support specifying a key in the navigation property. Use PUT /EntitySet(key)/NavigationProperty/$ref"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navPropName)
	if navProp == nil {
		if err := response.WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navPropName, h.metadata.EntitySetName)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// PUT is only valid for single-valued navigation properties
	if navProp.NavigationIsArray {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request",
			"PUT $ref is only supported on single-valued navigation properties. Use POST to add to collection navigation properties."); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// Parse the request body to extract @odata.id
	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request body",
			"Failed to parse JSON request body"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	odataID, ok := requestBody["@odata.id"]
	if !ok {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request body",
			"Request body must contain '@odata.id' property"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	odataIDStr, ok := odataID.(string)
	if !ok {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request body",
			"'@odata.id' must be a string"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// Validate and extract the target entity key from @odata.id
	targetKey, err := h.validateAndExtractEntityKey(odataIDStr, navProp.NavigationTarget)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid @odata.id",
			err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Update the navigation property reference
	if err := h.updateNavigationPropertyReference(entityKey, navProp, targetKey); err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError,
			fmt.Sprintf("Failed to update navigation property: %v", err)); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Success - return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// handlePostNavigationPropertyRef handles POST requests to add a reference to a collection navigation property
// Example: POST Products(1)/RelatedProducts/$ref with body {"@odata.id":"http://localhost:8080/Products(2)"}
func (h *EntityHandler) handlePostNavigationPropertyRef(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string) {
	// Parse the navigation property to extract any key (though POST shouldn't have a key in the navigation property)
	navPropName, targetKey := h.parseNavigationPropertyWithKey(navigationProperty)
	
	// If a key was provided in the navigation property name for POST, that's an error
	if targetKey != "" {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request",
			"POST $ref does not support specifying a key in the navigation property. Use POST /EntitySet(key)/NavigationProperty/$ref"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navPropName)
	if navProp == nil {
		if err := response.WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navPropName, h.metadata.EntitySetName)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// POST is only valid for collection navigation properties
	if !navProp.NavigationIsArray {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request",
			"POST $ref is only supported on collection navigation properties. Use PUT for single-valued navigation properties."); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// Parse the request body to extract @odata.id
	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid request body",
			"Failed to parse JSON request body"); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	odataID, ok := requestBody["@odata.id"]
	if !ok {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request body",
			"Request body must contain '@odata.id' property"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	odataIDStr, ok := odataID.(string)
	if !ok {
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request body",
			"'@odata.id' must be a string"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// Validate and extract the target entity key from @odata.id
	targetKey, err := h.validateAndExtractEntityKey(odataIDStr, navProp.NavigationTarget)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, "Invalid @odata.id",
			err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Add the reference to the collection navigation property
	if err := h.addNavigationPropertyReference(entityKey, navProp, targetKey); err != nil {
		if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError,
			fmt.Sprintf("Failed to add navigation property reference: %v", err)); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return
	}

	// Success - return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteNavigationPropertyRef handles DELETE requests to remove a navigation property reference
// Example: DELETE Products(1)/Category/$ref (single-valued)
// Example: DELETE Products(1)/RelatedProducts(2)/$ref (collection - handled here by extracting key from navigation property)
func (h *EntityHandler) handleDeleteNavigationPropertyRef(w http.ResponseWriter, _ *http.Request, entityKey string, navigationProperty string) {
	// Check if the navigation property contains a key (e.g., RelatedProducts(2))
	navPropName, targetKey := h.parseNavigationPropertyWithKey(navigationProperty)

	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navPropName)
	if navProp == nil {
		if err := response.WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navPropName, h.metadata.EntitySetName)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	}

	// If this is a collection navigation property with a target key specified
	if navProp.NavigationIsArray && targetKey != "" {
		// DELETE specific reference from collection: EntitySet(key)/NavProp(targetKey)/$ref
		if err := h.deleteCollectionNavigationPropertyReference(entityKey, navProp, targetKey); err != nil {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError,
				fmt.Sprintf("Failed to delete navigation property reference: %v", err)); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}
	} else if navProp.NavigationIsArray && targetKey == "" {
		// Collection navigation property without target key specified
		if err := response.WriteError(w, http.StatusBadRequest, "Invalid request",
			"DELETE $ref on collection navigation properties requires specifying the target entity key. Use EntitySet(key)/NavigationProperty(targetKey)/$ref"); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
		return
	} else {
		// Single-valued navigation property
		// Remove the reference by setting the navigation property to null
		if err := h.deleteNavigationPropertyReference(entityKey, navProp); err != nil {
			if writeErr := response.WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError,
				fmt.Sprintf("Failed to delete navigation property reference: %v", err)); writeErr != nil {
				fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
			}
			return
		}
	}

	// Success - return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// parseNavigationPropertyWithKey parses a navigation property that may contain a key
// Example: "RelatedProducts(2)" returns ("RelatedProducts", "2")
// Example: "Category" returns ("Category", "")
func (h *EntityHandler) parseNavigationPropertyWithKey(navigationProperty string) (string, string) {
	if idx := strings.Index(navigationProperty, "("); idx != -1 {
		if strings.HasSuffix(navigationProperty, ")") {
			navPropName := navigationProperty[:idx]
			targetKey := navigationProperty[idx+1 : len(navigationProperty)-1]
			return navPropName, targetKey
		}
	}
	return navigationProperty, ""
}

// validateAndExtractEntityKey validates an @odata.id URL and extracts the entity key
// Example: http://localhost:8080/Products(1) -> "1"
// The targetEntityType is the entity type name (e.g., "Product"), but we need to find the entity set name
func (h *EntityHandler) validateAndExtractEntityKey(odataID string, targetEntityType string) (string, error) {
	// Get the target entity metadata to find the correct entity set name
	targetMetadata, err := h.getTargetMetadata(targetEntityType)
	if err != nil {
		return "", fmt.Errorf("invalid target entity type '%s': %w", targetEntityType, err)
	}

	targetEntitySet := targetMetadata.EntitySetName

	// Parse the @odata.id URL to extract entity set and key
	// The URL format should be: http://server/EntitySet(key) or http://server/EntitySet(key1=value1,key2=value2)

	// Find the entity set name in the URL
	entitySetIndex := strings.LastIndex(odataID, "/"+targetEntitySet+"(")
	if entitySetIndex == -1 {
		return "", fmt.Errorf("invalid @odata.id: expected entity set '%s' (for entity type '%s')", targetEntitySet, targetEntityType)
	}

	// Extract the key portion after EntitySet(
	keyStart := entitySetIndex + len("/"+targetEntitySet+"(")
	keyEnd := strings.Index(odataID[keyStart:], ")")
	if keyEnd == -1 {
		return "", fmt.Errorf("invalid @odata.id: missing closing parenthesis")
	}

	key := odataID[keyStart : keyStart+keyEnd]
	if key == "" {
		return "", fmt.Errorf("invalid @odata.id: empty key")
	}

	return key, nil
}

// updateNavigationPropertyReference updates a single-valued navigation property reference
func (h *EntityHandler) updateNavigationPropertyReference(entityKey string, navProp *metadata.PropertyMetadata, targetKey string) error {
	// Get the target entity metadata to find the foreign key fields
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		return fmt.Errorf("failed to get target metadata: %w", err)
	}

	// Fetch the parent entity
	parent := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		return fmt.Errorf("invalid entity key: %w", err)
	}
	if err := db.First(parent).Error; err != nil {
		return fmt.Errorf("parent entity not found: %w", err)
	}

	// Fetch the target entity to verify it exists and get its key value
	target := reflect.New(targetMetadata.EntityType).Interface()
	targetDB, err := h.buildTargetKeyQuery(targetKey, targetMetadata)
	if err != nil {
		return fmt.Errorf("invalid target key: %w", err)
	}
	if err := targetDB.First(target).Error; err != nil {
		return fmt.Errorf("target entity not found: %w", err)
	}

	// Extract the target entity's key value(s)
	targetValue := reflect.ValueOf(target).Elem()

	// Update the foreign key field(s) in the parent entity
	// This assumes GORM convention: NavigationPropertyID field in parent references ID in target
	parentValue := reflect.ValueOf(parent).Elem()

	// For each key property in the target, find the corresponding foreign key field in the parent
	for _, keyProp := range targetMetadata.KeyProperties {
		targetKeyValue := targetValue.FieldByName(keyProp.Name)
		if !targetKeyValue.IsValid() {
			continue
		}

		// Build foreign key field name: NavigationPropertyName + KeyPropertyName
		foreignKeyFieldName := navProp.Name + keyProp.Name
		foreignKeyField := parentValue.FieldByName(foreignKeyFieldName)

		if foreignKeyField.IsValid() && foreignKeyField.CanSet() {
			// Handle type conversion if the foreign key field is a pointer
			if foreignKeyField.Kind() == reflect.Ptr {
				// Create a new pointer of the correct type and set it
				if targetKeyValue.CanAddr() {
					foreignKeyField.Set(targetKeyValue.Addr())
				} else {
					// Create a new value and copy the data
					newValue := reflect.New(foreignKeyField.Type().Elem())
					newValue.Elem().Set(targetKeyValue)
					foreignKeyField.Set(newValue)
				}
			} else {
				// Direct assignment for non-pointer fields
				foreignKeyField.Set(targetKeyValue)
			}
		}
	}

	// Save the updated parent entity
	if err := h.db.Save(parent).Error; err != nil {
		return fmt.Errorf("failed to save entity: %w", err)
	}

	return nil
}

// addNavigationPropertyReference adds a reference to a collection navigation property
func (h *EntityHandler) addNavigationPropertyReference(entityKey string, navProp *metadata.PropertyMetadata, targetKey string) error {
	// Get the target entity metadata
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		return fmt.Errorf("failed to get target metadata: %w", err)
	}

	// Fetch the parent entity
	parent := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		return fmt.Errorf("invalid entity key: %w", err)
	}
	if err := db.First(parent).Error; err != nil {
		return fmt.Errorf("parent entity not found: %w", err)
	}

	// Fetch the target entity to verify it exists
	target := reflect.New(targetMetadata.EntityType).Interface()
	targetDB, err := h.buildTargetKeyQuery(targetKey, targetMetadata)
	if err != nil {
		return fmt.Errorf("invalid target key: %w", err)
	}
	if err := targetDB.First(target).Error; err != nil {
		return fmt.Errorf("target entity not found: %w", err)
	}

	// Use GORM's association API to add the relationship
	parentValue := reflect.ValueOf(parent).Elem()
	navField := parentValue.FieldByName(navProp.Name)

	if !navField.IsValid() {
		return fmt.Errorf("navigation property field not found")
	}

	// Use GORM Model().Association() to append the target entity
	if err := h.db.Model(parent).Association(navProp.Name).Append(target); err != nil {
		return fmt.Errorf("failed to add association: %w", err)
	}

	return nil
}

// deleteNavigationPropertyReference removes a single-valued navigation property reference
func (h *EntityHandler) deleteNavigationPropertyReference(entityKey string, navProp *metadata.PropertyMetadata) error {
	// Fetch the parent entity
	parent := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		return fmt.Errorf("invalid entity key: %w", err)
	}
	if err := db.First(parent).Error; err != nil {
		return fmt.Errorf("parent entity not found: %w", err)
	}

	// Get the target entity metadata to find the foreign key fields
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		return fmt.Errorf("failed to get target metadata: %w", err)
	}

	// Set the foreign key field(s) to null/zero value
	parentValue := reflect.ValueOf(parent).Elem()

	for _, keyProp := range targetMetadata.KeyProperties {
		// Build foreign key field name: NavigationPropertyName + KeyPropertyName
		foreignKeyFieldName := navProp.Name + keyProp.Name
		foreignKeyField := parentValue.FieldByName(foreignKeyFieldName)

		if foreignKeyField.IsValid() && foreignKeyField.CanSet() {
			// Set to zero value (null for nullable types, 0 for numeric types)
			foreignKeyField.Set(reflect.Zero(foreignKeyField.Type()))
		}
	}

	// Save the updated parent entity
	if err := h.db.Save(parent).Error; err != nil {
		return fmt.Errorf("failed to save entity: %w", err)
	}

	return nil
}

// deleteCollectionNavigationPropertyReference removes a specific reference from a collection navigation property
func (h *EntityHandler) deleteCollectionNavigationPropertyReference(entityKey string, navProp *metadata.PropertyMetadata, targetKey string) error {
	// Get the target entity metadata
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		return fmt.Errorf("failed to get target metadata: %w", err)
	}

	// Fetch the parent entity
	parent := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		return fmt.Errorf("invalid entity key: %w", err)
	}
	if err := db.First(parent).Error; err != nil {
		return fmt.Errorf("parent entity not found: %w", err)
	}

	// Fetch the target entity to verify it exists
	target := reflect.New(targetMetadata.EntityType).Interface()
	targetDB, err := h.buildTargetKeyQuery(targetKey, targetMetadata)
	if err != nil {
		return fmt.Errorf("invalid target key: %w", err)
	}
	if err := targetDB.First(target).Error; err != nil {
		return fmt.Errorf("target entity not found: %w", err)
	}

	// Use GORM's association API to delete the relationship
	if err := h.db.Model(parent).Association(navProp.Name).Delete(target); err != nil {
		return fmt.Errorf("failed to delete association: %w", err)
	}

	return nil
}

// buildTargetKeyQuery builds a database query to find an entity by key in a different entity set
func (h *EntityHandler) buildTargetKeyQuery(keyString string, targetMetadata *metadata.EntityMetadata) (*gorm.DB, error) {
	// Parse the key string and build query conditions
	// This reuses the logic from buildKeyQuery but with target metadata

	db := h.db.Model(reflect.New(targetMetadata.EntityType).Interface())

	// Check if this is a composite key (contains '=' or ',')
	if strings.Contains(keyString, "=") || strings.Contains(keyString, ",") {
		// Composite key: ProductID=1,LanguageKey='EN'
		keyPairs := strings.Split(keyString, ",")
		for _, pair := range keyPairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid composite key format: %s", keyString)
			}

			keyName := strings.TrimSpace(parts[0])
			keyValue := strings.TrimSpace(parts[1])

			// Remove quotes if present
			keyValue = strings.Trim(keyValue, "'\"")

			// Find the key property in metadata
			var keyProp *metadata.PropertyMetadata
			for i := range targetMetadata.KeyProperties {
				if targetMetadata.KeyProperties[i].Name == keyName || targetMetadata.KeyProperties[i].JsonName == keyName {
					keyProp = &targetMetadata.KeyProperties[i]
					break
				}
			}

			if keyProp == nil {
				return nil, fmt.Errorf("key property '%s' not found", keyName)
			}

			// Add where condition using GORM column name
			db = db.Where(fmt.Sprintf("%s = ?", keyProp.Name), keyValue)
		}
	} else {
		// Single key
		if len(targetMetadata.KeyProperties) != 1 {
			return nil, fmt.Errorf("entity requires composite key, but single key provided")
		}

		keyProp := targetMetadata.KeyProperties[0]
		keyValue := strings.Trim(keyString, "'\"")
		db = db.Where(fmt.Sprintf("%s = ?", keyProp.Name), keyValue)
	}

	return db, nil
}
