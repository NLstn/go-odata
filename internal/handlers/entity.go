package handlers

import (
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/gorm"
)

// EntityHandler handles HTTP requests for entity collections
type EntityHandler struct {
	db               *gorm.DB
	metadata         *metadata.EntityMetadata
	entitiesMetadata map[string]*metadata.EntityMetadata
}

// NewEntityHandler creates a new entity handler
func NewEntityHandler(db *gorm.DB, entityMetadata *metadata.EntityMetadata) *EntityHandler {
	return &EntityHandler{
		db:       db,
		metadata: entityMetadata,
	}
}

// SetEntitiesMetadata sets the entities metadata registry for navigation property handling
func (h *EntityHandler) SetEntitiesMetadata(entitiesMetadata map[string]*metadata.EntityMetadata) {
	h.entitiesMetadata = entitiesMetadata
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
