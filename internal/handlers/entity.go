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

// entityMatchesType checks if an entity matches a given type name.
// It supports both struct instances and map-based results (used for $select/$expand projections).
func (h *EntityHandler) entityMatchesType(entity interface{}, typeName string) bool {
	typeCandidates := uniqueStrings(h.typeNameCandidates(typeName))
	if len(typeCandidates) == 0 {
		return false
	}

	if entityMap, ok := entity.(map[string]interface{}); ok {
		if h.mapEntityMatchesType(entityMap, typeCandidates) {
			return true
		}
		return h.entityNameMatches(typeCandidates)
	}

	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	if h.structEntityMatchesType(v, typeCandidates) {
		return true
	}

	return h.entityNameMatches(typeCandidates)
}

func (h *EntityHandler) structEntityMatchesType(v reflect.Value, typeCandidates []string) bool {
	for _, fieldName := range h.discriminatorFieldNames() {
		field := v.FieldByName(fieldName)
		if !field.IsValid() || field.Kind() != reflect.String {
			continue
		}

		if typeNameMatches(field.String(), typeCandidates) {
			return true
		}
	}

	return false
}

func (h *EntityHandler) mapEntityMatchesType(entity map[string]interface{}, typeCandidates []string) bool {
	for _, key := range h.mapDiscriminatorKeys() {
		value, ok := entity[key]
		if !ok {
			continue
		}

		if strVal, ok := value.(string); ok {
			if typeNameMatches(strVal, typeCandidates) {
				return true
			}
		}
	}

	return false
}

func (h *EntityHandler) entityNameMatches(typeCandidates []string) bool {
	entityName := h.metadata.EntityName
	if entityName == "" {
		return false
	}

	qualified := h.qualifiedTypeName(entityName)

	for _, candidate := range typeCandidates {
		if candidate == entityName || candidate == qualified {
			return true
		}
	}

	return false
}

func (h *EntityHandler) typeNameCandidates(typeName string) []string {
	trimmed := strings.TrimSpace(typeName)
	if trimmed == "" {
		return nil
	}

	candidates := []string{trimmed}

	if strings.Contains(trimmed, ".") {
		parts := strings.Split(trimmed, ".")
		if len(parts) > 0 {
			simple := parts[len(parts)-1]
			if simple != "" {
				candidates = append(candidates, simple)
			}
		}
	} else {
		qualified := h.qualifiedTypeName(trimmed)
		if qualified != "" && qualified != trimmed {
			candidates = append(candidates, qualified)
		}
	}

	return candidates
}

func (h *EntityHandler) discriminatorFieldNames() []string {
	var names []string

	if prop := h.typeDiscriminatorProperty(); prop != nil {
		if prop.FieldName != "" {
			names = append(names, prop.FieldName)
		}
		if prop.Name != "" && !strings.EqualFold(prop.Name, prop.FieldName) {
			names = append(names, prop.Name)
		}
	}

	names = append(names, "ProductType", "Type", "EntityType")

	return uniqueStrings(names)
}

func (h *EntityHandler) mapDiscriminatorKeys() []string {
	var keys []string

	if prop := h.typeDiscriminatorProperty(); prop != nil {
		if prop.JsonName != "" && prop.JsonName != "-" {
			keys = append(keys, prop.JsonName)
		}
		if prop.FieldName != "" {
			keys = append(keys, prop.FieldName)
		}
		if prop.Name != "" && !strings.EqualFold(prop.Name, prop.FieldName) {
			keys = append(keys, prop.Name)
		}
	}

	keys = append(keys, "ProductType", "Type", "EntityType", "@odata.type")

	return uniqueStrings(keys)
}

func (h *EntityHandler) typeDiscriminatorProperty() *metadata.PropertyMetadata {
	if h.metadata == nil {
		return nil
	}

	candidates := h.discriminatorPropertyNames()
	for _, candidate := range candidates {
		for i := range h.metadata.Properties {
			prop := &h.metadata.Properties[i]
			if prop.Type.Kind() != reflect.String {
				continue
			}

			if strings.EqualFold(prop.FieldName, candidate) || strings.EqualFold(prop.Name, candidate) ||
				strings.EqualFold(prop.JsonName, candidate) {
				return prop
			}
		}
	}

	return nil
}

func (h *EntityHandler) discriminatorPropertyNames() []string {
	var names []string

	if h.metadata != nil {
		entityName := strings.TrimSpace(h.metadata.EntityName)
		if entityName != "" {
			names = append(names, entityName+"Type")
		}
	}

	names = append(names, "ProductType", "Type", "EntityType")

	return uniqueStrings(names)
}

func (h *EntityHandler) typeDiscriminatorColumn() string {
	prop := h.typeDiscriminatorProperty()
	if prop == nil {
		return ""
	}

	if column := parseGORMColumn(prop.GormTag); column != "" {
		return column
	}

	name := strings.TrimSpace(prop.JsonName)
	if name == "" || name == "-" {
		name = prop.FieldName
	}
	if name == "" {
		name = prop.Name
	}
	if name == "" {
		return ""
	}

	return toSnakeCase(name)
}

func parseGORMColumn(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ""
	}

	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "column:") {
			column := strings.TrimSpace(strings.TrimPrefix(part, "column:"))
			if column != "" {
				return column
			}
		}
	}

	return ""
}

func typeNameMatches(value string, typeCandidates []string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	for _, candidate := range typeCandidates {
		if value == candidate {
			return true
		}
	}

	if idx := strings.LastIndex(value, "."); idx != -1 && idx < len(value)-1 {
		simple := value[idx+1:]
		for _, candidate := range typeCandidates {
			if candidate == simple {
				return true
			}
		}
	}

	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, exists := seen[lower]; exists {
			continue
		}
		seen[lower] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}
