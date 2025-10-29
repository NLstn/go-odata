package handlers

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
)

// EntityHandler handles HTTP requests for entity collections
type EntityHandler struct {
	db               *gorm.DB
	metadata         *metadata.EntityMetadata
	entitiesMetadata map[string]*metadata.EntityMetadata
	namespace        string
	tracker          *trackchanges.Tracker
}

// NewEntityHandler creates a new entity handler
func NewEntityHandler(db *gorm.DB, entityMetadata *metadata.EntityMetadata) *EntityHandler {
	return &EntityHandler{
		db:        db,
		metadata:  entityMetadata,
		namespace: defaultNamespace,
	}
}

// SetEntitiesMetadata sets the entities metadata registry for navigation property handling
func (h *EntityHandler) SetEntitiesMetadata(entitiesMetadata map[string]*metadata.EntityMetadata) {
	h.entitiesMetadata = entitiesMetadata
}

// SetNamespace updates the namespace used for generated metadata annotations.
func (h *EntityHandler) SetNamespace(namespace string) {
	trimmed := strings.TrimSpace(namespace)
	if trimmed == "" {
		trimmed = defaultNamespace
	}
	h.namespace = trimmed
}

func (h *EntityHandler) namespaceOrDefault() string {
	if strings.TrimSpace(h.namespace) == "" {
		return defaultNamespace
	}
	return h.namespace
}

func (h *EntityHandler) qualifiedTypeName(typeName string) string {
	return fmt.Sprintf("%s.%s", h.namespaceOrDefault(), typeName)
}

// SetDeltaTracker configures the change tracker used for odata.track-changes support.
func (h *EntityHandler) SetDeltaTracker(tracker *trackchanges.Tracker) {
	h.tracker = tracker
}

// EnableChangeTracking turns on change tracking for this entity handler.
// It registers the entity set with the configured tracker so that delta tokens can be issued.
func (h *EntityHandler) EnableChangeTracking() error {
	if h.metadata == nil {
		return fmt.Errorf("entity metadata is not initialized")
	}
	if h.metadata.IsSingleton {
		return fmt.Errorf("change tracking is not supported for singleton '%s'", h.metadata.EntitySetName)
	}
	if h.tracker == nil {
		return fmt.Errorf("change tracker is not configured for entity set '%s'", h.metadata.EntitySetName)
	}
	if h.metadata.ChangeTrackingEnabled {
		return nil
	}

	h.metadata.ChangeTrackingEnabled = true
	h.tracker.RegisterEntity(h.metadata.EntitySetName)
	return nil
}

// IsSingleton returns true if this handler is for a singleton
func (h *EntityHandler) IsSingleton() bool {
	return h.metadata.IsSingleton
}

// FetchEntity fetches an entity by its key string
// This is a public method that can be used by action/function handlers
func (h *EntityHandler) FetchEntity(entityKey string) (interface{}, error) {
	// Use empty query options since we just need to verify entity exists
	queryOptions := &query.QueryOptions{}
	return h.fetchEntityByKey(entityKey, queryOptions, nil)
}

// IsNotFoundError checks if an error is a "not found" error
func IsNotFoundError(err error) bool {
	return err == gorm.ErrRecordNotFound
}

// entityMatchesType checks if an entity matches a given type name
// This uses reflection to check if the entity has a "ProductType" field matching the type name
func (h *EntityHandler) entityMatchesType(entity interface{}, typeName string) bool {
	// Use reflection to check for a ProductType or similar discriminator field
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	// Look for common discriminator field names
	discriminatorFields := []string{"ProductType", "Type", "EntityType", "@odata.type"}

	for _, fieldName := range discriminatorFields {
		field := v.FieldByName(fieldName)
		if field.IsValid() && field.Kind() == reflect.String {
			fieldValue := field.String()
			// Match if the type name matches exactly or if it matches without namespace prefix
			if fieldValue == typeName {
				return true
			}
			// Also allow matching just the type name part (after the last dot)
			parts := strings.Split(typeName, ".")
			if len(parts) > 0 && fieldValue == parts[len(parts)-1] {
				return true
			}
		}
	}

	// If no discriminator field found, check if the type name matches the entity name
	// This allows base types to match their own name
	if h.metadata.EntityName == typeName {
		return true
	}

	// Also check without namespace prefix
	parts := strings.Split(typeName, ".")
	if len(parts) > 0 && h.metadata.EntityName == parts[len(parts)-1] {
		return true
	}

	return false
}
