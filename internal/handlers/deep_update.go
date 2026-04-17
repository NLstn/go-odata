package handlers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// processDeepUpdateNavigationProperties processes navigation properties with inline entity data
// for deep update in PATCH operations. Related single-valued navigation property entities are
// updated with the provided inline data map (PATCH semantics applied to the related entity).
//
// Returns the list of navigation property keys that were processed and should be removed from
// updateData before the main entity is saved. Collection-valued navigation properties are also
// added to the removal list to prevent GORM from trying to save them as columns.
func (h *EntityHandler) processDeepUpdateNavigationProperties(ctx context.Context, entity interface{}, updateData map[string]interface{}, tx *gorm.DB) ([]string, error) {
	entityValue := reflect.ValueOf(entity).Elem()
	keysToRemove := make([]string, 0, len(updateData))

	for key, value := range updateData {
		// Skip annotations (both entity-level "@odata.type" and property-level "prop@odata.bind")
		if strings.HasPrefix(key, "@") || strings.Contains(key, "@") {
			continue
		}

		// Only process navigation properties
		navProp := h.findNavigationProperty(key)
		if navProp == nil {
			continue
		}

		// Always remove navigation property keys from updateData to prevent GORM column errors
		keysToRemove = append(keysToRemove, key)

		// Collection-valued navigation properties are not supported for deep update
		if navProp.NavigationIsArray {
			continue
		}

		// Only proceed if the value is a map (inline entity data)
		inlineData, ok := value.(map[string]interface{})
		if !ok || len(inlineData) == 0 {
			continue
		}

		// Find the target entity metadata
		targetMeta := h.findTargetEntityMetadataByType(navProp.NavigationTarget)
		if targetMeta == nil {
			return nil, fmt.Errorf("entity type '%s' for navigation property '%s' is not registered", navProp.NavigationTarget, key)
		}

		// Perform the deep update on the related entity
		if err := h.performDeepUpdateSingleNavProp(ctx, entityValue, navProp, targetMeta, inlineData, tx); err != nil {
			return nil, fmt.Errorf("failed to deep update navigation property '%s': %w", key, err)
		}
	}

	return keysToRemove, nil
}

// findTargetEntityMetadataByType finds the metadata for an entity by its type name (EntityName).
func (h *EntityHandler) findTargetEntityMetadataByType(entityTypeName string) *metadata.EntityMetadata {
	if h.entitiesMetadata == nil {
		return nil
	}
	for _, meta := range h.entitiesMetadata {
		if meta.EntityName == entityTypeName {
			return meta
		}
	}
	return nil
}

// performDeepUpdateSingleNavProp applies a partial update (PATCH semantics) to the entity
// referenced by a single-valued navigation property.
//
// Two relationship patterns are supported:
//   - BelongsTo (FK on current entity): e.g. Product.CategoryID references Category.ID
//   - HasOne (FK on related entity):    e.g. Order has one Address where Address.OrderID = Order.ID
func (h *EntityHandler) performDeepUpdateSingleNavProp(
	ctx context.Context,
	entityValue reflect.Value,
	navProp *metadata.PropertyMetadata,
	targetMeta *metadata.EntityMetadata,
	inlineData map[string]interface{},
	tx *gorm.DB,
) error {
	ctxDB := tx.WithContext(ctx)

	if len(navProp.ReferentialConstraints) > 0 {
		for dependentProp, principalProp := range navProp.ReferentialConstraints {
			// Check if the FK field exists on the current entity (BelongsTo relationship)
			fkField := entityValue.FieldByName(dependentProp)
			if fkField.IsValid() {
				// BelongsTo: FK is on the current entity; find the related entity by the FK value
				fkValue := extractFieldValue(fkField)
				if fkValue == nil {
					return fmt.Errorf("foreign key '%s' is nil, cannot deep update related entity '%s'", dependentProp, targetMeta.EntityName)
				}

				// Find the principal property's column name in the target entity
				principalColumnName := findEntityPropertyColumnName(targetMeta, principalProp)
				if principalColumnName == "" {
					return fmt.Errorf("referenced property '%s' not found in target entity '%s'", principalProp, targetMeta.EntityName)
				}

				// Fetch the related entity
				targetEntity := reflect.New(targetMeta.EntityType).Interface()
				if err := ctxDB.Where(fmt.Sprintf("%s = ?", principalColumnName), fkValue).First(targetEntity).Error; err != nil {
					if err == gorm.ErrRecordNotFound {
						return fmt.Errorf("related entity '%s' with %s=%v not found", targetMeta.EntityName, principalColumnName, fkValue)
					}
					return fmt.Errorf("failed to fetch related entity '%s': %w", targetMeta.EntityName, err)
				}

				return ctxDB.Model(targetEntity).Updates(inlineData).Error
			}
		}
	}

	// HasOne: FK is on the related entity; use the current entity's referenced key value
	// to find the related entity via the FK column stored in navProp.ForeignKeyColumnName.
	currentKeyValue := h.getNavPropPrincipalValue(entityValue, navProp)
	if currentKeyValue == nil {
		return fmt.Errorf("cannot determine key value for navigation property '%s'", navProp.Name)
	}

	fkColumnName := navProp.ForeignKeyColumnName
	if fkColumnName == "" {
		return fmt.Errorf("cannot determine foreign key column for navigation property '%s'", navProp.Name)
	}

	targetEntity := reflect.New(targetMeta.EntityType).Interface()
	if err := ctxDB.Where(fmt.Sprintf("%s = ?", fkColumnName), currentKeyValue).First(targetEntity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("related entity '%s' with %s=%v not found", targetMeta.EntityName, fkColumnName, currentKeyValue)
		}
		return fmt.Errorf("failed to fetch related entity '%s': %w", targetMeta.EntityName, err)
	}

	return ctxDB.Model(targetEntity).Updates(inlineData).Error
}

// extractFieldValue returns the underlying Go value for a reflect.Value, dereferencing pointers.
func extractFieldValue(field reflect.Value) interface{} {
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return nil
		}
		return field.Elem().Interface()
	}
	return field.Interface()
}

// findEntityPropertyColumnName returns the database column name for the named property in meta.
func findEntityPropertyColumnName(meta *metadata.EntityMetadata, propName string) string {
	for _, prop := range meta.Properties {
		if prop.Name == propName || prop.JsonName == propName {
			if prop.ColumnName != "" {
				return prop.ColumnName
			}
			return toSnakeCase(prop.Name)
		}
	}
	return ""
}

// getNavPropPrincipalValue returns the value of the principal (referenced) property on the
// current entity for HasOne relationships, where the FK lives on the related entity.
func (h *EntityHandler) getNavPropPrincipalValue(entityValue reflect.Value, navProp *metadata.PropertyMetadata) interface{} {
	if len(navProp.ReferentialConstraints) > 0 {
		// In HasOne, ReferentialConstraints maps "FK on related" -> "referenced field on current entity"
		for _, principalProp := range navProp.ReferentialConstraints {
			field := entityValue.FieldByName(principalProp)
			if field.IsValid() {
				return extractFieldValue(field)
			}
		}
	}

	// Fall back to the first key property
	if len(h.metadata.KeyProperties) > 0 {
		field := entityValue.FieldByName(h.metadata.KeyProperties[0].Name)
		if field.IsValid() {
			return extractFieldValue(field)
		}
	}
	return nil
}
