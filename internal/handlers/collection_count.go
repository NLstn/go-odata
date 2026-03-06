package handlers

import (
	"context"
	"sort"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/gorm"
)

// countEntities applies query scopes and filters to return the total number of matching entities.
func (h *EntityHandler) countEntities(ctx context.Context, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (int64, error) {
	return h.storage.CountEntities(ctx, h, queryOptions, scopes)
}

func searchAppliedAtDB(db *gorm.DB) bool {
	if val, ok := db.Get("_fts_search_applied"); ok {
		if applied, ok := val.(bool); ok && applied {
			return true
		}
	}
	return false
}

func searchableCountColumns(entityMetadata *metadata.EntityMetadata) []string {
	if entityMetadata == nil {
		return nil
	}

	columns := make(map[string]struct{})
	for _, key := range entityMetadata.KeyProperties {
		if key.ColumnName != "" {
			columns[key.ColumnName] = struct{}{}
		}
	}

	for _, prop := range query.SearchableProperties(entityMetadata) {
		if prop.ColumnName != "" {
			columns[prop.ColumnName] = struct{}{}
		}
	}

	if len(columns) == 0 {
		return nil
	}

	selectColumns := make([]string, 0, len(columns))
	for column := range columns {
		selectColumns = append(selectColumns, column)
	}
	sort.Strings(selectColumns)
	return selectColumns
}
