package handlers

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// processODataBindAnnotations processes @odata.bind annotations in request data
// and establishes relationships between entities according to OData v4.01 spec.
//
// The @odata.bind annotation allows clients to bind navigation properties to existing entities
// by reference (URL) rather than embedding the full entity details.
//
// Spec reference: OData v4.01 Part 1: Protocol, Section 11.4.2 and 11.4.3
// - Section 11.4.2: Binding single-valued navigation properties
// - Section 11.4.3: Binding collection-valued navigation properties
//
// Examples:
//   Single-valued: "Category@odata.bind": "Categories(1)" or "http://host/Categories(1)"
//   Collection-valued: "Orders@odata.bind": ["Orders(1)", "Orders(2)"]
func (h *EntityHandler) processODataBindAnnotations(entity interface{}, requestData map[string]interface{}, db *gorm.DB) error {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Iterate through all properties to find @odata.bind annotations
	for key, value := range requestData {
		// Check if this is an @odata.bind annotation
		if !strings.HasSuffix(key, "@odata.bind") {
			continue
		}

		// Extract the navigation property name (remove @odata.bind suffix)
		navPropName := strings.TrimSuffix(key, "@odata.bind")

		// Find the navigation property metadata using existing method
		navProp := h.findNavigationProperty(navPropName)
		if navProp == nil {
			return fmt.Errorf("navigation property '%s' not found in entity '%s'", navPropName, h.metadata.EntityName)
		}

		// Process based on whether it's a collection or single-valued navigation property
		if navProp.NavigationIsArray {
			// Collection-valued navigation property
			if err := h.bindCollectionNavigationProperty(entityValue, navProp, value, db); err != nil {
				return fmt.Errorf("failed to bind collection navigation property '%s': %w", navPropName, err)
			}
		} else {
			// Single-valued navigation property
			if err := h.bindSingleNavigationProperty(entityValue, navProp, value, db); err != nil {
				return fmt.Errorf("failed to bind single navigation property '%s': %w", navPropName, err)
			}
		}
	}

	return nil
}

// processODataBindAnnotationsForUpdate is similar to processODataBindAnnotations but also
// adds the foreign key values to the updateData map so they get persisted via Updates()
func (h *EntityHandler) processODataBindAnnotationsForUpdate(entity interface{}, requestData map[string]interface{}, db *gorm.DB) error {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Iterate through all properties to find @odata.bind annotations
	for key, value := range requestData {
		// Check if this is an @odata.bind annotation
		if !strings.HasSuffix(key, "@odata.bind") {
			continue
		}

		// Extract the navigation property name (remove @odata.bind suffix)
		navPropName := strings.TrimSuffix(key, "@odata.bind")

		// Find the navigation property metadata using existing method
		navProp := h.findNavigationProperty(navPropName)
		if navProp == nil {
			return fmt.Errorf("navigation property '%s' not found in entity '%s'", navPropName, h.metadata.EntityName)
		}

		// Process based on whether it's a collection or single-valued navigation property
		if navProp.NavigationIsArray {
			// Collection-valued navigation property
			if err := h.bindCollectionNavigationProperty(entityValue, navProp, value, db); err != nil {
				return fmt.Errorf("failed to bind collection navigation property '%s': %w", navPropName, err)
			}
		} else {
			// Single-valued navigation property
			foreignKeyValues, err := h.bindSingleNavigationPropertyForUpdate(entityValue, navProp, value, db)
			if err != nil {
				return fmt.Errorf("failed to bind single navigation property '%s': %w", navPropName, err)
			}
			// Add the foreign key values to updateData so they get persisted
			for fkName, fkValue := range foreignKeyValues {
				requestData[fkName] = fkValue
			}
		}
	}

	return nil
}

// bindSingleNavigationProperty binds a single-valued navigation property
func (h *EntityHandler) bindSingleNavigationProperty(entityValue reflect.Value, navProp *metadata.PropertyMetadata, value interface{}, db *gorm.DB) error {
	// Value should be a string containing the entity reference
	refURL, ok := value.(string)
	if !ok {
		return fmt.Errorf("@odata.bind value for single-valued navigation property must be a string, got %T", value)
	}

	// Parse the entity reference to get the entity set and key
	entitySetName, entityKey, err := parseEntityReference(refURL)
	if err != nil {
		return fmt.Errorf("invalid entity reference '%s': %w", refURL, err)
	}

	// Get the target entity metadata
	targetMetadata, exists := h.entitiesMetadata[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' not found", entitySetName)
	}

	// Verify that the navigation target matches
	if targetMetadata.EntityName != navProp.NavigationTarget {
		return fmt.Errorf("entity set '%s' does not match navigation target '%s'", entitySetName, navProp.NavigationTarget)
	}

	// If we have referential constraints (foreign key), set the foreign key value directly
	if len(navProp.ReferentialConstraints) > 0 {
		// Build a query to fetch the target entity to get its key value(s)
		targetHandler := NewEntityHandler(db, targetMetadata)
		targetHandler.SetEntitiesMetadata(h.entitiesMetadata)
		
		targetDB, err := targetHandler.buildKeyQuery(entityKey)
		if err != nil {
			return fmt.Errorf("invalid entity key '%s': %w", entityKey, err)
		}

		// Create an instance to fetch the target entity
		targetEntity := reflect.New(targetMetadata.EntityType).Interface()
		if err := targetDB.First(targetEntity).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("referenced entity '%s(%s)' not found", entitySetName, entityKey)
			}
			return fmt.Errorf("failed to fetch referenced entity: %w", err)
		}

		// Get the target entity's key value(s) and set the foreign key on our entity
		targetValue := reflect.ValueOf(targetEntity).Elem()
		for dependentProp, principalProp := range navProp.ReferentialConstraints {
			// Find the principal property in the target entity
			var principalField reflect.Value
			for _, prop := range targetMetadata.Properties {
				if prop.Name == principalProp || prop.JsonName == principalProp {
					principalField = targetValue.FieldByName(prop.Name)
					break
				}
			}
			if !principalField.IsValid() {
				return fmt.Errorf("principal property '%s' not found in target entity", principalProp)
			}

			// Find the dependent property in our entity
			dependentField := entityValue.FieldByName(dependentProp)
			if !dependentField.IsValid() {
				return fmt.Errorf("dependent property '%s' not found in entity", dependentProp)
			}

			// Set the foreign key value
			if dependentField.CanSet() {
				// Handle pointer types - if the dependent field is a pointer but principal field is not,
				// we need to create a pointer to the value
				if dependentField.Kind() == reflect.Ptr && principalField.Kind() != reflect.Ptr {
					// Create a new pointer of the correct type
					newPtr := reflect.New(dependentField.Type().Elem())
					newPtr.Elem().Set(principalField)
					dependentField.Set(newPtr)
				} else if dependentField.Kind() != reflect.Ptr && principalField.Kind() == reflect.Ptr {
					// If dependent is not a pointer but principal is, dereference the principal
					dependentField.Set(principalField.Elem())
				} else {
					// Types match (both pointers or both values)
					dependentField.Set(principalField)
				}
			} else {
				return fmt.Errorf("cannot set dependent property '%s'", dependentProp)
			}
		}
	}
	// Note: If there are no referential constraints, GORM will handle the association
	// when we use Association().Append or Replace after the entity is created/saved

	return nil
}

// bindSingleNavigationPropertyForUpdate binds a single-valued navigation property
// and returns the foreign key values that should be added to updateData
func (h *EntityHandler) bindSingleNavigationPropertyForUpdate(entityValue reflect.Value, navProp *metadata.PropertyMetadata, value interface{}, db *gorm.DB) (map[string]interface{}, error) {
	foreignKeyValues := make(map[string]interface{})
	
	// Value should be a string containing the entity reference
	refURL, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("@odata.bind value for single-valued navigation property must be a string, got %T", value)
	}

	// Parse the entity reference to get the entity set and key
	entitySetName, entityKey, err := parseEntityReference(refURL)
	if err != nil {
		return nil, fmt.Errorf("invalid entity reference '%s': %w", refURL, err)
	}

	// Get the target entity metadata
	targetMetadata, exists := h.entitiesMetadata[entitySetName]
	if !exists {
		return nil, fmt.Errorf("entity set '%s' not found", entitySetName)
	}

	// Verify that the navigation target matches
	if targetMetadata.EntityName != navProp.NavigationTarget {
		return nil, fmt.Errorf("entity set '%s' does not match navigation target '%s'", entitySetName, navProp.NavigationTarget)
	}

	// If we have referential constraints (foreign key), set the foreign key value directly
	if len(navProp.ReferentialConstraints) > 0 {
		// Build a query to fetch the target entity to get its key value(s)
		targetHandler := NewEntityHandler(db, targetMetadata)
		targetHandler.SetEntitiesMetadata(h.entitiesMetadata)
		
		targetDB, err := targetHandler.buildKeyQuery(entityKey)
		if err != nil {
			return nil, fmt.Errorf("invalid entity key '%s': %w", entityKey, err)
		}

		// Create an instance to fetch the target entity
		targetEntity := reflect.New(targetMetadata.EntityType).Interface()
		if err := targetDB.First(targetEntity).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("referenced entity '%s(%s)' not found", entitySetName, entityKey)
			}
			return nil, fmt.Errorf("failed to fetch referenced entity: %w", err)
		}

		// Get the target entity's key value(s) and set the foreign key on our entity
		targetValue := reflect.ValueOf(targetEntity).Elem()
		for dependentProp, principalProp := range navProp.ReferentialConstraints {
			// Find the principal property in the target entity
			var principalField reflect.Value
			for _, prop := range targetMetadata.Properties {
				if prop.Name == principalProp || prop.JsonName == principalProp {
					principalField = targetValue.FieldByName(prop.Name)
					break
				}
			}
			if !principalField.IsValid() {
				return nil, fmt.Errorf("principal property '%s' not found in target entity", principalProp)
			}

			// Find the dependent property in our entity
			dependentField := entityValue.FieldByName(dependentProp)
			if !dependentField.IsValid() {
				return nil, fmt.Errorf("dependent property '%s' not found in entity", dependentProp)
			}

			// Get the actual value to store
			var valueToStore interface{}
			if principalField.Kind() == reflect.Ptr && !principalField.IsNil() {
				valueToStore = principalField.Elem().Interface()
			} else {
				valueToStore = principalField.Interface()
			}

			// Set the foreign key value in the entity (for in-memory use)
			if dependentField.CanSet() {
				// Handle pointer types - if the dependent field is a pointer but principal field is not,
				// we need to create a pointer to the value
				if dependentField.Kind() == reflect.Ptr && principalField.Kind() != reflect.Ptr {
					// Create a new pointer of the correct type
					newPtr := reflect.New(dependentField.Type().Elem())
					newPtr.Elem().Set(principalField)
					dependentField.Set(newPtr)
				} else if dependentField.Kind() != reflect.Ptr && principalField.Kind() == reflect.Ptr {
					// If dependent is not a pointer but principal is, dereference the principal
					dependentField.Set(principalField.Elem())
				} else {
					// Types match (both pointers or both values)
					dependentField.Set(principalField)
				}
			}

			// Add to foreignKeyValues map using the JSON name for the updateData map
			// Find the JSON name for this dependent property
			for _, prop := range h.metadata.Properties {
				if prop.Name == dependentProp {
					foreignKeyValues[prop.JsonName] = valueToStore
					break
				}
			}
		}
	}
	// Note: If there are no referential constraints, GORM will handle the association
	// when we use Association().Append or Replace after the entity is created/saved

	return foreignKeyValues, nil
}

// bindCollectionNavigationProperty binds a collection-valued navigation property
func (h *EntityHandler) bindCollectionNavigationProperty(entityValue reflect.Value, navProp *metadata.PropertyMetadata, value interface{}, db *gorm.DB) error {
	_ = entityValue // Reserved for future implementation of collection-valued navigation properties
	
	// Value should be an array of strings containing entity references
	refArray, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("@odata.bind value for collection navigation property must be an array, got %T", value)
	}

	// Get the target entity metadata
	// We need to get the entity set name from the first reference
	if len(refArray) == 0 {
		// Empty array means clear the collection
		return nil
	}

	// Parse the first reference to determine the entity set
	firstRef, ok := refArray[0].(string)
	if !ok {
		return fmt.Errorf("@odata.bind array elements must be strings, got %T", refArray[0])
	}

	entitySetName, _, err := parseEntityReference(firstRef)
	if err != nil {
		return fmt.Errorf("invalid entity reference '%s': %w", firstRef, err)
	}

	targetMetadata, exists := h.entitiesMetadata[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' not found", entitySetName)
	}

	// Verify that the navigation target matches
	if targetMetadata.EntityName != navProp.NavigationTarget {
		return fmt.Errorf("entity set '%s' does not match navigation target '%s'", entitySetName, navProp.NavigationTarget)
	}

	// Fetch all referenced entities
	targetHandler := NewEntityHandler(db, targetMetadata)
	targetHandler.SetEntitiesMetadata(h.entitiesMetadata)

	// Pre-allocate slice for target entities
	targetEntities := make([]interface{}, 0, len(refArray))
	for _, refValue := range refArray {
		refURL, ok := refValue.(string)
		if !ok {
			return fmt.Errorf("@odata.bind array elements must be strings, got %T", refValue)
		}

		refEntitySetName, entityKey, err := parseEntityReference(refURL)
		if err != nil {
			return fmt.Errorf("invalid entity reference '%s': %w", refURL, err)
		}

		if refEntitySetName != entitySetName {
			return fmt.Errorf("all references in collection must be from the same entity set")
		}

		targetDB, err := targetHandler.buildKeyQuery(entityKey)
		if err != nil {
			return fmt.Errorf("invalid entity key '%s': %w", entityKey, err)
		}

		targetEntity := reflect.New(targetMetadata.EntityType).Interface()
		if err := targetDB.First(targetEntity).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("referenced entity '%s(%s)' not found", entitySetName, entityKey)
			}
			return fmt.Errorf("failed to fetch referenced entity: %w", err)
		}

		targetEntities = append(targetEntities, targetEntity)
	}

	// Store the target entities for later association after the main entity is saved
	// We'll use GORM's Association API to handle the many-to-many or one-to-many relationship
	// This is done by storing a special marker in the entity that we can process after save
	// Note: This will be handled by the caller after db.Create/Updates is called
	// TODO: Implement full collection-valued navigation property binding
	_ = targetEntities // Mark as intentionally unused until fully implemented

	return nil
}

// parseEntityReference parses an entity reference URL and extracts the entity set and key
// Supports both absolute and relative URLs according to OData v4.01 spec
// Examples:
//   - "Categories(1)" -> ("Categories", "1", nil)
//   - "Products(ProductID=1,LanguageKey='en')" -> ("Products", "ProductID=1,LanguageKey='en'", nil)
//   - "http://host/service/Categories(1)" -> ("Categories", "1", nil)
//   - "/service/Categories(1)" -> ("Categories", "1", nil)
func parseEntityReference(refURL string) (entitySetName string, entityKey string, err error) {
	// Trim whitespace
	refURL = strings.TrimSpace(refURL)

	// Handle absolute URLs - extract just the path portion
	if strings.HasPrefix(refURL, "http://") || strings.HasPrefix(refURL, "https://") {
		parsedURL, err := url.Parse(refURL)
		if err != nil {
			return "", "", fmt.Errorf("invalid URL: %w", err)
		}
		refURL = strings.TrimPrefix(parsedURL.Path, "/")
	} else if strings.HasPrefix(refURL, "/") {
		// Handle root-relative URLs
		refURL = strings.TrimPrefix(refURL, "/")
	}

	// Now we should have something like "Categories(1)" or "Products(ProductID=1,LanguageKey='en')"
	// Find the opening parenthesis
	openParen := strings.Index(refURL, "(")
	if openParen == -1 {
		return "", "", fmt.Errorf("entity reference must include key in parentheses: %s", refURL)
	}

	// Extract entity set name
	entitySetName = refURL[:openParen]

	// Find the closing parenthesis
	closeParen := strings.LastIndex(refURL, ")")
	if closeParen == -1 || closeParen <= openParen {
		return "", "", fmt.Errorf("entity reference has invalid key format: %s", refURL)
	}

	// Extract the key portion (everything between parentheses)
	entityKey = refURL[openParen+1 : closeParen]

	return entitySetName, entityKey, nil
}
