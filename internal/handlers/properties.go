package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// HandleNavigationProperty handles GET, HEAD, and OPTIONS requests for navigation properties (e.g., Products(1)/Descriptions)
func (h *EntityHandler) HandleNavigationProperty(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string, isRef bool) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetNavigationProperty(w, r, entityKey, navigationProperty, isRef)
	case http.MethodOptions:
		h.handleOptionsNavigationProperty(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for navigation properties", r.Method)); err != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, err)
		}
	}
}

// handleGetNavigationProperty handles GET requests for navigation properties
func (h *EntityHandler) handleGetNavigationProperty(w http.ResponseWriter, r *http.Request, entityKey string, navigationProperty string, isRef bool) {
	// Find and validate the navigation property
	navProp := h.findNavigationProperty(navigationProperty)
	if navProp == nil {
		if err := response.WriteError(w, http.StatusNotFound, "Navigation property not found",
			fmt.Sprintf("'%s' is not a valid navigation property for %s", navigationProperty, h.metadata.EntitySetName)); err != nil {
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

// handleOptionsNavigationProperty handles OPTIONS requests for navigation properties
func (h *EntityHandler) handleOptionsNavigationProperty(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

// findNavigationProperty finds a navigation property by name in the entity metadata
func (h *EntityHandler) findNavigationProperty(navigationProperty string) *metadata.PropertyMetadata {
	for _, prop := range h.metadata.Properties {
		if (prop.JsonName == navigationProperty || prop.Name == navigationProperty) && prop.IsNavigationProp {
			return &prop
		}
	}
	return nil
}

// findStructuralProperty finds a structural (non-navigation) property by name in the entity metadata
func (h *EntityHandler) findStructuralProperty(propertyName string) *metadata.PropertyMetadata {
	for _, prop := range h.metadata.Properties {
		if (prop.JsonName == propertyName || prop.Name == propertyName) && !prop.IsNavigationProp {
			return &prop
		}
	}
	return nil
}

// IsNavigationProperty checks if a property name is a navigation property
func (h *EntityHandler) IsNavigationProperty(propertyName string) bool {
	return h.findNavigationProperty(propertyName) != nil
}

// IsStructuralProperty checks if a property name is a structural property
func (h *EntityHandler) IsStructuralProperty(propertyName string) bool {
	return h.findStructuralProperty(propertyName) != nil
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
		h.writeRawPropertyValue(w, r, fieldValue)
	} else {
		h.writePropertyResponse(w, r, entityKey, prop, fieldValue)
	}
}

// handleOptionsStructuralProperty handles OPTIONS requests for structural properties
func (h *EntityHandler) handleOptionsStructuralProperty(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
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
	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			fmt.Printf(LogMsgErrorWritingErrorResponse, writeErr)
		}
		return reflect.Value{}, err
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
		if writeErr := response.WriteError(w, http.StatusNotFound, ErrMsgEntityNotFound,
			fmt.Sprintf("Entity with key '%s' not found", entityKey)); writeErr != nil {
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
func (h *EntityHandler) writeRawPropertyValue(w http.ResponseWriter, r *http.Request, fieldValue reflect.Value) {
	// Set appropriate content type based on the value type
	valueInterface := fieldValue.Interface()

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

	db, err := h.buildKeyQuery(entityKey)
	if err != nil {
		return nil, err
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
			w.Header().Set(HeaderODataVersion, "4.0")
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
