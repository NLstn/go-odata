package handlers

import (
	"net/http"

	"github.com/nlstn/go-odata/internal/query"
)

// OverwriteContext provides context information for overwrite handlers.
// It contains the parsed OData query options and entity key (for single entity operations).
type OverwriteContext struct {
	// QueryOptions contains the parsed OData query options ($filter, $select, $expand, etc.)
	QueryOptions *query.QueryOptions
	// EntityKey is the key value for single entity operations (empty for collection operations)
	EntityKey string
	// EntityKeyValues contains parsed key-value pairs for composite keys
	// For single keys: map with one entry (key name -> key value)
	// For composite keys: map with multiple entries (e.g., {"OrderID": 1, "ProductID": 5})
	// Empty for collection operations
	EntityKeyValues map[string]interface{}
	// Request is the original HTTP request
	Request *http.Request
}

// CollectionResult represents the result from a GetCollection overwrite handler.
type CollectionResult struct {
	// Items contains the collection of entities
	Items interface{}
	// Count is the total count of entities (only needed if $count=true was requested)
	Count *int64
}

// GetCollectionHandler is the function signature for overwriting the GetCollection operation.
// It receives the overwrite context and should return the collection result or an error.
type GetCollectionHandler func(ctx *OverwriteContext) (*CollectionResult, error)

// GetEntityHandler is the function signature for overwriting the GetEntity operation.
// It receives the overwrite context (including EntityKey) and should return the entity or an error.
type GetEntityHandler func(ctx *OverwriteContext) (interface{}, error)

// CreateHandler is the function signature for overwriting the Create operation.
// It receives the overwrite context and the parsed entity data, and should return the created entity or an error.
type CreateHandler func(ctx *OverwriteContext, entity interface{}) (interface{}, error)

// UpdateHandler is the function signature for overwriting the Update operation.
// It receives the overwrite context (including EntityKey), update data, and whether it's a full replacement (PUT).
// It should return the updated entity or an error.
type UpdateHandler func(ctx *OverwriteContext, updateData map[string]interface{}, isFullReplace bool) (interface{}, error)

// DeleteHandler is the function signature for overwriting the Delete operation.
// It receives the overwrite context (including EntityKey) and should return an error if deletion fails.
type DeleteHandler func(ctx *OverwriteContext) error

// GetCountHandler is the function signature for overwriting the GetCount operation.
// It receives the overwrite context and should return the count or an error.
type GetCountHandler func(ctx *OverwriteContext) (int64, error)

// EntityOverwrite contains all overwrite handlers for an entity set.
// Any handler that is nil will use the default GORM-based implementation.
type EntityOverwrite struct {
	// GetCollection overrides the collection retrieval operation (GET /EntitySet)
	GetCollection GetCollectionHandler
	// GetEntity overrides the single entity retrieval operation (GET /EntitySet(key))
	GetEntity GetEntityHandler
	// Create overrides the entity creation operation (POST /EntitySet)
	Create CreateHandler
	// Update overrides the entity update operation (PATCH/PUT /EntitySet(key))
	Update UpdateHandler
	// Delete overrides the entity deletion operation (DELETE /EntitySet(key))
	Delete DeleteHandler
	// GetCount overrides the count operation (GET /EntitySet/$count)
	GetCount GetCountHandler
}

// entityOverwriteHandlers stores the overwrite handlers for an EntityHandler.
type entityOverwriteHandlers struct {
	getCollection GetCollectionHandler
	getEntity     GetEntityHandler
	create        CreateHandler
	update        UpdateHandler
	delete        DeleteHandler
	getCount      GetCountHandler
}

// hasGetCollection returns true if a GetCollection overwrite is registered.
func (o *entityOverwriteHandlers) hasGetCollection() bool {
	return o != nil && o.getCollection != nil
}

// hasGetEntity returns true if a GetEntity overwrite is registered.
func (o *entityOverwriteHandlers) hasGetEntity() bool {
	return o != nil && o.getEntity != nil
}

// hasCreate returns true if a Create overwrite is registered.
func (o *entityOverwriteHandlers) hasCreate() bool {
	return o != nil && o.create != nil
}

// hasUpdate returns true if an Update overwrite is registered.
func (o *entityOverwriteHandlers) hasUpdate() bool {
	return o != nil && o.update != nil
}

// hasDelete returns true if a Delete overwrite is registered.
func (o *entityOverwriteHandlers) hasDelete() bool {
	return o != nil && o.delete != nil
}

// hasGetCount returns true if a GetCount overwrite is registered.
func (o *entityOverwriteHandlers) hasGetCount() bool {
	return o != nil && o.getCount != nil
}
