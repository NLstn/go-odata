package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// HandleNavigationProperty handles GET, HEAD, and OPTIONS requests for navigation properties (e.g., Products(1)/Descriptions)
func (h *EntityHandler) HandleNavigationProperty(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string, isRef bool) {
	propertyPath := []string{navigationProperty}
	if isRef {
		propertyPath = append(propertyPath, "$ref")
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, entityKey, propertyPath), auth.OperationRead, h.logger) {
			return
		}
		h.handleGetNavigationProperty(w, r, entityKey, navigationProperty, isRef)
	case http.MethodPut:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, entityKey, propertyPath), auth.OperationUpdate, h.logger) {
			return
		}
		if isRef {
			h.handlePutNavigationPropertyRef(w, r, entityKey, navigationProperty)
		} else {
			WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
				fmt.Sprintf("Method %s is not supported for navigation properties without $ref", r.Method))
		}
	case http.MethodPost:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, entityKey, propertyPath), auth.OperationUpdate, h.logger) {
			return
		}
		if isRef {
			h.handlePostNavigationPropertyRef(w, r, entityKey, navigationProperty)
		} else {
			WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
				fmt.Sprintf("Method %s is not supported for navigation properties without $ref", r.Method))
		}
	case http.MethodDelete:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, entityKey, propertyPath), auth.OperationUpdate, h.logger) {
			return
		}
		if isRef {
			h.handleDeleteNavigationPropertyRef(w, r, entityKey, navigationProperty)
		} else {
			WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
				fmt.Sprintf("Method %s is not supported for navigation properties without $ref", r.Method))
		}
	case http.MethodOptions:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, entityKey, propertyPath), auth.OperationRead, h.logger) {
			return
		}
		if isRef {
			h.handleOptionsNavigationPropertyRef(w)
		} else {
			h.handleOptionsNavigationProperty(w)
		}
	default:
		WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for navigation properties", r.Method))
	}
}

// HandleNavigationPropertyCount handles GET, HEAD, and OPTIONS requests for navigation property count (e.g., Products(1)/Descriptions/$count)
func (h *EntityHandler) HandleNavigationPropertyCount(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, entityKey, []string{navigationProperty, "$count"}), auth.OperationQuery, h.logger) {
			return
		}
		h.handleGetNavigationPropertyCount(w, r, entityKey, navigationProperty)
	case http.MethodOptions:
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(h.metadata, entityKey, []string{navigationProperty, "$count"}), auth.OperationRead, h.logger) {
			return
		}
		h.handleOptionsNavigationPropertyCount(w)
	default:
		WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for navigation property $count", r.Method))
	}
}

// handleGetNavigationProperty handles GET requests for navigation properties
func (h *EntityHandler) handleGetNavigationProperty(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string, isRef bool) {
	// Parse the navigation property to extract any key (e.g., RelatedProducts(2))
	navPropName, targetKey := h.parseNavigationPropertyWithKey(navigationProperty)

	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navPropName)
	if navProp == nil {
		WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navPropName, h.metadata.EntitySetName))
		return
	}

	// Authorize access to the target entity set
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err))
		return
	}

	operation := auth.OperationRead
	if navProp.NavigationIsArray && targetKey == "" {
		operation = auth.OperationQuery
	}

	if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(targetMetadata, targetKey, nil), operation, h.logger) {
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
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			"Could not access navigation property")
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
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err))
		return
	}

	parent, err := h.verifyAndFetchParentEntity(w, r, entityKey)
	if err != nil {
		return
	}

	relatedDB := h.buildNavigationRelatedQuery(parent, targetMetadata)
	navigationPath := fmt.Sprintf("%s(%s)/%s", h.metadata.EntitySetName, entityKey, navProp.JsonName)

	h.executeCollectionQuery(w, &collectionExecutionContext{
		Metadata:          targetMetadata,
		ParseQueryOptions: h.createNavParseQueryOptions(r, targetMetadata),
		BeforeRead:        h.createNavBeforeRead(r, targetMetadata),
		CountFunc:         h.createNavCountFunc(relatedDB, targetMetadata),
		FetchFunc:         h.createNavFetchFunc(relatedDB, targetMetadata),
		NextLinkFunc:      h.createNavNextLinkFunc(r, targetMetadata),
		AfterRead:         h.createNavAfterRead(r, targetMetadata),
		WriteResponse:     h.createNavWriteResponse(w, r, navigationPath, targetMetadata, isRef),
	})
}

// verifyAndFetchParentEntity verifies that the parent entity exists and is authorized
func (h *EntityHandler) verifyAndFetchParentEntity(w http.ResponseWriter, r *http.Request, entityKey string) (interface{}, error) {
	parentOptions := &query.QueryOptions{}
	parentScopes, parentHookErr := callBeforeReadEntity(h.metadata, r, parentOptions)
	if parentHookErr != nil {
		h.writeHookError(w, parentHookErr, http.StatusForbidden, "Authorization failed")
		return nil, parentHookErr
	}

	parent := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(h.db, entityKey)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error())
		return nil, err
	}
	if len(parentScopes) > 0 {
		db = db.Scopes(parentScopes...)
	}
	if err := db.First(parent).Error; err != nil {
		h.handleFetchError(w, err, entityKey)
		return nil, err
	}

	if _, _, parentAfterErr := callAfterReadEntity(h.metadata, r, parentOptions, parent); parentAfterErr != nil {
		h.writeHookError(w, parentAfterErr, http.StatusForbidden, "Authorization failed")
		return nil, parentAfterErr
	}

	return parent, nil
}

// buildNavigationRelatedQuery builds a GORM query for the related collection with foreign key constraints
func (h *EntityHandler) buildNavigationRelatedQuery(parent interface{}, targetMetadata *metadata.EntityMetadata) *gorm.DB {
	relatedDB := h.db.Model(reflect.New(targetMetadata.EntityType).Interface())
	parentValue := reflect.ValueOf(parent).Elem()

	for _, keyProp := range h.metadata.KeyProperties {
		keyFieldValue := parentValue.FieldByName(keyProp.Name)
		if keyFieldValue.IsValid() {
			foreignKeyFieldName := fmt.Sprintf("%s%s", h.metadata.EntityName, keyProp.Name)
			foreignKeyColumnName := toSnakeCase(foreignKeyFieldName)
			relatedDB = relatedDB.Where(fmt.Sprintf("%s = ?", foreignKeyColumnName), keyFieldValue.Interface())
		}
	}

	return relatedDB
}

// createNavParseQueryOptions creates the ParseQueryOptions callback for navigation collections
func (h *EntityHandler) createNavParseQueryOptions(r *http.Request, targetMetadata *metadata.EntityMetadata) func() (*query.QueryOptions, error) {
	return func() (*query.QueryOptions, error) {
		queryOptions, err := query.ParseQueryOptions(r.URL.Query(), targetMetadata)
		if err != nil {
			return nil, err
		}
		if err := applyPolicyFilter(r, h.policy, buildEntityResourceDescriptor(targetMetadata, "", nil), queryOptions); err != nil {
			return nil, &collectionRequestError{
				StatusCode: http.StatusForbidden,
				ErrorCode:  "Authorization failed",
				Message:    err.Error(),
			}
		}
		return queryOptions, nil
	}
}

// createNavBeforeRead creates the BeforeRead callback for navigation collections
func (h *EntityHandler) createNavBeforeRead(r *http.Request, targetMetadata *metadata.EntityMetadata) func(*query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	return func(queryOptions *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
		return callBeforeReadCollection(targetMetadata, r, queryOptions)
	}
}

// createNavCountFunc creates the CountFunc callback for navigation collections
func (h *EntityHandler) createNavCountFunc(relatedDB *gorm.DB, targetMetadata *metadata.EntityMetadata) func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (*int64, error) {
	return func(queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (*int64, error) {
		if !queryOptions.Count {
			return nil, nil
		}

		countDB := relatedDB
		if len(scopes) > 0 {
			countDB = countDB.Scopes(scopes...)
		}

		if queryOptions.Filter != nil {
			countDB = query.ApplyFilterOnly(countDB, queryOptions.Filter, targetMetadata)
		}

		var count int64
		if err := countDB.Count(&count).Error; err != nil {
			return nil, err
		}

		return &count, nil
	}
}

// createNavFetchFunc creates the FetchFunc callback for navigation collections
func (h *EntityHandler) createNavFetchFunc(relatedDB *gorm.DB, targetMetadata *metadata.EntityMetadata) func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	return func(queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
		modifiedOptions := *queryOptions
		if queryOptions.Top != nil {
			topPlusOne := *queryOptions.Top + 1
			modifiedOptions.Top = &topPlusOne
		}

		db := relatedDB
		if len(scopes) > 0 {
			db = db.Scopes(scopes...)
		}

		db = query.ApplyQueryOptions(db, &modifiedOptions, targetMetadata)

		if query.ShouldUseMapResults(queryOptions) {
			var mapResults []map[string]interface{}
			if err := db.Find(&mapResults).Error; err != nil {
				return nil, err
			}
			return mapResults, nil
		}

		resultsSlice := reflect.New(reflect.SliceOf(targetMetadata.EntityType)).Interface()
		if err := db.Find(resultsSlice).Error; err != nil {
			return nil, err
		}

		results := reflect.ValueOf(resultsSlice).Elem().Interface()

		if queryOptions.Search != "" {
			results = query.ApplySearch(results, queryOptions.Search, targetMetadata)
		}

		if len(queryOptions.Select) > 0 {
			results = query.ApplySelect(results, queryOptions.Select, targetMetadata, queryOptions.Expand)
		}

		return results, nil
	}
}

// createNavNextLinkFunc creates the NextLinkFunc callback for navigation collections
func (h *EntityHandler) createNavNextLinkFunc(r *http.Request, targetMetadata *metadata.EntityMetadata) func(*query.QueryOptions, interface{}) (*string, interface{}, error) {
	return func(queryOptions *query.QueryOptions, results interface{}) (*string, interface{}, error) {
		if queryOptions.Top == nil {
			return nil, results, nil
		}

		value := reflect.ValueOf(results)
		if value.Kind() == reflect.Slice && value.Len() > *queryOptions.Top {
			trimmed := h.trimResults(results, *queryOptions.Top)

			if nextURL := buildNextLinkWithSkipToken(targetMetadata, queryOptions, results, r); nextURL != nil {
				return nextURL, trimmed, nil
			}

			currentSkip := 0
			if queryOptions.Skip != nil {
				currentSkip = *queryOptions.Skip
			}
			nextSkip := currentSkip + *queryOptions.Top
			fallbackURL := response.BuildNextLink(r, nextSkip)
			return &fallbackURL, trimmed, nil
		}

		return nil, results, nil
	}
}

// createNavAfterRead creates the AfterRead callback for navigation collections
func (h *EntityHandler) createNavAfterRead(r *http.Request, targetMetadata *metadata.EntityMetadata) func(*query.QueryOptions, interface{}) (interface{}, bool, error) {
	return func(queryOptions *query.QueryOptions, results interface{}) (interface{}, bool, error) {
		return callAfterReadCollection(targetMetadata, r, queryOptions, results)
	}
}

// createNavWriteResponse creates the WriteResponse callback for navigation collections
func (h *EntityHandler) createNavWriteResponse(w http.ResponseWriter, r *http.Request, navigationPath string, targetMetadata *metadata.EntityMetadata, isRef bool) func(*query.QueryOptions, interface{}, *int64, *string) error {
	return func(queryOptions *query.QueryOptions, results interface{}, totalCount *int64, nextLink *string) error {
		if isRef {
			h.writeNavigationCollectionRefFromData(w, r, targetMetadata, results, totalCount, nextLink)
			return nil
		}

		if err := response.WriteODataCollection(w, r, navigationPath, results, totalCount, nextLink); err != nil {
			h.logger.Error("Error writing navigation property collection", "error", err)
		}

		return nil
	}
}

// handleNavigationCollectionItem handles accessing a specific item from a collection navigation property
// Example: GET Products(1)/RelatedProducts(2) or GET Products(1)/RelatedProducts(2)/$ref
func (h *EntityHandler) handleNavigationCollectionItem(w http.ResponseWriter, r *http.Request, entityKey string, navProp *metadata.PropertyMetadata, targetKey string, isRef bool) {
	// Get the target entity metadata
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err))
		return
	}

	// First verify that the parent entity exists
	parent := reflect.New(h.metadata.EntityType).Interface()
	parentDB, err := h.buildKeyQuery(h.db, entityKey)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error())
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
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			"Could not access navigation property")
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
			} else {
				// Composite key: parse the target key string and compare all key values
				// Format: key1=value1,key2=value2
				targetKeyMap, err := h.parseCompositeKeyString(targetKey)
				if err != nil {
					// If parsing fails, the key format is invalid
					continue
				}

				// Check if all key values match
				allMatch := true
				for _, keyProp := range targetMetadata.KeyProperties {
					itemKeyValue := item.FieldByName(keyProp.Name)
					if !itemKeyValue.IsValid() {
						allMatch = false
						break
					}

					// Get the expected value from the target key map
					expectedValue, ok := targetKeyMap[keyProp.JsonName]
					if !ok {
						// Try with the field name if JsonName not found
						expectedValue, ok = targetKeyMap[keyProp.Name]
						if !ok {
							allMatch = false
							break
						}
					}

					// Convert item key value to string for comparison
					itemKeyStr := fmt.Sprintf("%v", itemKeyValue.Interface())
					if itemKeyStr != expectedValue {
						allMatch = false
						break
					}
				}

				if allMatch {
					found = true
					targetEntity = item.Interface()
					break
				}
			}
		}
	}

	if !found {
		WriteError(w, http.StatusNotFound, "Entity not found",
			fmt.Sprintf("Entity with key '%s' is not related to the parent entity via '%s'", targetKey, navProp.JsonName))
		return
	}

	// Write the response
	if isRef {
		// Write a single entity reference
		// WriteEntityReference expects just the entity path (e.g., "Products(2)"), not the full URL
		entityPath := fmt.Sprintf("%s(%s)", targetMetadata.EntitySetName, targetKey)
		if err := response.WriteEntityReference(w, r, entityPath); err != nil {
			h.logger.Error("Error writing entity reference", "error", err)
		}
	} else {
		// Write the full entity
		navigationPath := fmt.Sprintf("%s(%s)/%s(%s)", h.metadata.EntitySetName, entityKey, navProp.JsonName, targetKey)
		if err := response.WriteODataCollection(w, r, navigationPath, []interface{}{targetEntity}, nil, nil); err != nil {
			h.logger.Error("Error writing navigation property entity", "error", err)
		}
	}
}

// handleGetNavigationPropertyCount handles GET requests for navigation property count
func (h *EntityHandler) handleGetNavigationPropertyCount(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string) {
	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navigationProperty)
	if navProp == nil {
		WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navigationProperty, h.metadata.EntitySetName))
		return
	}

	// Authorize access to the target entity set
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err))
		return
	}

	if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(targetMetadata, "", nil), auth.OperationQuery, h.logger) {
		return
	}

	// $count is only valid for collection navigation properties
	if !navProp.NavigationIsArray {
		WriteError(w, http.StatusBadRequest, "Invalid request",
			fmt.Sprintf("$count is only supported on collection navigation properties. '%s' is a single-valued navigation property.", navigationProperty))
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
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			"Could not access navigation property")
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
		h.logger.Error("Error writing count response", "error", err)
	}
}

// handleOptionsNavigationPropertyCount handles OPTIONS requests for navigation property count
func (h *EntityHandler) handleOptionsNavigationPropertyCount(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
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
		db, err = h.buildKeyQuery(h.db, entityKey)
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
		h.logger.Error("Error writing navigation property collection", "error", err)
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
		h.logger.Error("Error writing navigation property response", "error", err)
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
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err))
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
		h.logger.Error("Error writing entity reference collection", "error", err)
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
			SetODataVersionHeaderForRequest(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		navValue = navValue.Elem()
	}

	// Get the target entity metadata to extract keys
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err))
		return
	}

	// Extract key values and build entity ID
	keyValues := response.ExtractEntityKeys(navValue.Interface(), targetMetadata.KeyProperties)
	entityID := response.BuildEntityID(targetMetadata.EntitySetName, keyValues)

	if err := response.WriteEntityReference(w, r, entityID); err != nil {
		h.logger.Error("Error writing entity reference", "error", err)
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
		h.logger.Error("Error writing entity reference collection", "error", err)
	}
}

// handlePutNavigationPropertyRef handles PUT requests to update a single-valued navigation property reference
// Example: PUT Products(1)/Category/$ref with body {"@odata.id":"http://localhost:8080/Categories(2)"}
func (h *EntityHandler) handlePutNavigationPropertyRef(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string) {
	// Parse the navigation property to extract any key (though PUT shouldn't have a key in the navigation property)
	navPropName, targetKey := h.parseNavigationPropertyWithKey(navigationProperty)

	// If a key was provided in the navigation property name for PUT, that's an error
	if targetKey != "" {
		WriteError(w, http.StatusBadRequest, "Invalid request",
			"PUT $ref does not support specifying a key in the navigation property. Use PUT /EntitySet(key)/NavigationProperty/$ref")
		return
	}

	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navPropName)
	if navProp == nil {
		WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navPropName, h.metadata.EntitySetName))
		return
	}

	// PUT is only valid for single-valued navigation properties
	if navProp.NavigationIsArray {
		WriteError(w, http.StatusBadRequest, "Invalid request",
			"PUT $ref is only supported on single-valued navigation properties. Use POST to add to collection navigation properties.")
		return
	}

	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err))
		return
	}

	if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(targetMetadata, "", nil), auth.OperationUpdate, h.logger) {
		return
	}

	// Parse the request body to extract @odata.id
	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body",
			"Failed to parse JSON request body")
		return
	}

	odataID, ok := requestBody["@odata.id"]
	if !ok {
		WriteError(w, http.StatusBadRequest, "Invalid request body",
			"Request body must contain '@odata.id' property")
		return
	}

	odataIDStr, ok := odataID.(string)
	if !ok {
		WriteError(w, http.StatusBadRequest, "Invalid request body",
			"'@odata.id' must be a string")
		return
	}

	// Validate and extract the target entity key from @odata.id
	targetKey, err = h.validateAndExtractEntityKey(odataIDStr, navProp.NavigationTarget)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid @odata.id",
			err.Error())
		return
	}

	// Update the navigation property reference
	if err := h.updateNavigationPropertyReference(entityKey, navProp, targetKey); err != nil {
		h.logger.Error("Failed to update navigation property reference", "error", err, "entityKey", entityKey, "navProp", navProp.Name, "targetKey", targetKey)
		WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError,
			fmt.Sprintf("Failed to update navigation property: %v", err))
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
		WriteError(w, http.StatusBadRequest, "Invalid request",
			"POST $ref does not support specifying a key in the navigation property. Use POST /EntitySet(key)/NavigationProperty/$ref")
		return
	}

	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navPropName)
	if navProp == nil {
		WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navPropName, h.metadata.EntitySetName))
		return
	}

	// POST is only valid for collection navigation properties
	if !navProp.NavigationIsArray {
		WriteError(w, http.StatusBadRequest, "Invalid request",
			"POST $ref is only supported on collection navigation properties. Use PUT for single-valued navigation properties.")
		return
	}

	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
			fmt.Sprintf("Failed to get target metadata: %v", err))
		return
	}

	if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(targetMetadata, "", nil), auth.OperationUpdate, h.logger) {
		return
	}

	// Parse the request body to extract @odata.id
	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body",
			"Failed to parse JSON request body")
		return
	}

	odataID, ok := requestBody["@odata.id"]
	if !ok {
		WriteError(w, http.StatusBadRequest, "Invalid request body",
			"Request body must contain '@odata.id' property")
		return
	}

	odataIDStr, ok := odataID.(string)
	if !ok {
		WriteError(w, http.StatusBadRequest, "Invalid request body",
			"'@odata.id' must be a string")
		return
	}

	// Validate and extract the target entity key from @odata.id
	targetKey, err = h.validateAndExtractEntityKey(odataIDStr, navProp.NavigationTarget)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid @odata.id",
			err.Error())
		return
	}

	// Add the reference to the collection navigation property
	if err := h.addNavigationPropertyReference(entityKey, navProp, targetKey); err != nil {
		h.logger.Error("Failed to add navigation property reference", "error", err, "entityKey", entityKey, "navProp", navProp.Name, "targetKey", targetKey)
		WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError,
			fmt.Sprintf("Failed to add navigation property reference: %v", err))
		return
	}

	// Success - return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteNavigationPropertyRef handles DELETE requests to remove a navigation property reference
// Example: DELETE Products(1)/Category/$ref (single-valued)
// Example: DELETE Products(1)/RelatedProducts(2)/$ref (collection - handled here by extracting key from navigation property)
func (h *EntityHandler) handleDeleteNavigationPropertyRef(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string) {
	// Check if the navigation property contains a key (e.g., RelatedProducts(2))
	navPropName, targetKey := h.parseNavigationPropertyWithKey(navigationProperty)

	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navPropName)
	if navProp == nil {
		WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navPropName, h.metadata.EntitySetName))
		return
	}

	// If this is a collection navigation property with a target key specified
	if navProp.NavigationIsArray && targetKey != "" {
		targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
				fmt.Sprintf("Failed to get target metadata: %v", err))
			return
		}
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(targetMetadata, targetKey, nil), auth.OperationUpdate, h.logger) {
			return
		}
		// DELETE specific reference from collection: EntitySet(key)/NavProp(targetKey)/$ref
		if err := h.deleteCollectionNavigationPropertyReference(entityKey, navProp, targetKey); err != nil {
			h.logger.Error("Failed to delete collection navigation property reference", "error", err, "entityKey", entityKey, "navProp", navProp.Name, "targetKey", targetKey)
			WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError,
				fmt.Sprintf("Failed to delete navigation property reference: %v", err))
			return
		}
	} else if navProp.NavigationIsArray && targetKey == "" {
		// Collection navigation property without target key specified
		WriteError(w, http.StatusBadRequest, "Invalid request",
			"DELETE $ref on collection navigation properties requires specifying the target entity key. Use EntitySet(key)/NavigationProperty(targetKey)/$ref")
		return
	} else {
		targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, ErrMsgInternalError,
				fmt.Sprintf("Failed to get target metadata: %v", err))
			return
		}
		if !authorizeRequest(w, r, h.policy, buildEntityResourceDescriptor(targetMetadata, "", nil), auth.OperationUpdate, h.logger) {
			return
		}
		// Single-valued navigation property
		// Remove the reference by setting the navigation property to null
		if err := h.deleteNavigationPropertyReference(entityKey, navProp); err != nil {
			h.logger.Error("Failed to delete single navigation property reference", "error", err, "entityKey", entityKey, "navProp", navProp.Name)
			WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError,
				fmt.Sprintf("Failed to delete navigation property reference: %v", err))
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
	db, err := h.buildKeyQuery(h.db, entityKey)
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
	db, err := h.buildKeyQuery(h.db, entityKey)
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
	db, err := h.buildKeyQuery(h.db, entityKey)
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
	db, err := h.buildKeyQuery(h.db, entityKey)
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

// parseCompositeKeyString parses a composite key string into a map of key names to values
// Example: "ProductID=1,LanguageKey='EN'" -> map[string]string{"ProductID": "1", "LanguageKey": "EN"}
func (h *EntityHandler) parseCompositeKeyString(keyString string) (map[string]string, error) {
	keyMap := make(map[string]string)

	// Check if this looks like a composite key
	if !strings.Contains(keyString, "=") {
		return nil, fmt.Errorf("not a composite key format")
	}

	// Parse composite key format: key1=value1,key2=value2
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

		keyMap[keyName] = keyValue
	}

	return keyMap, nil
}
