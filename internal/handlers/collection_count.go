package handlers

import (
	"context"
	"reflect"

	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/gorm"
)

// countEntities applies query scopes and filters to return the total number of matching entities.
func (h *EntityHandler) countEntities(ctx context.Context, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (int64, error) {
	countDB := h.db.WithContext(ctx).Model(reflect.New(h.metadata.EntityType).Interface())
	if len(scopes) > 0 {
		countDB = countDB.Scopes(scopes...)
	}

	if queryOptions != nil && queryOptions.Filter != nil {
		countDB = query.ApplyFilterOnly(countDB, queryOptions.Filter, h.metadata)
	}

	var count int64
	if err := countDB.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
