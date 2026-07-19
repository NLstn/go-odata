package handlers

import (
	"context"
	"reflect"
	"sort"

	"github.com/nlstn/go-odata/internal/fastscan"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/gorm"
)

// countEntities applies query scopes and filters to return the total number of matching entities.
func (h *EntityHandler) countEntities(ctx context.Context, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (int64, error) {
	// Fast path: count directly from the in-memory snapshot cache when it is warm
	// and the query is within the supported subset.
	if len(scopes) == 0 && h.entityCache != nil && h.snapshotSupportsCount(queryOptions) {
		if snap, ok := h.cacheSnapshot(ctx); ok {
			var filter *query.FilterExpression
			if queryOptions != nil {
				filter = queryOptions.Filter
			}
			return h.countSnapshot(snap, filter), nil
		}
	}

	db := h.db.WithContext(ctx)

	baseDB := db.Model(reflect.New(h.metadata.EntityType).Interface())
	if len(scopes) > 0 {
		baseDB = baseDB.Scopes(scopes...)
	}

	if queryOptions == nil {
		var count int64
		if err := baseDB.Count(&count).Error; err != nil {
			return 0, err
		}

		return count, nil
	}

	// $apply transforms the collection (groupby/aggregate/filter/etc.) before any other
	// system query option is evaluated, so the count must reflect the transformed result
	// set rather than the original collection's row count.
	if len(queryOptions.Apply) > 0 {
		return h.countTransformedEntities(ctx, queryOptions, baseDB)
	}

	filter := queryOptions.Filter
	search := queryOptions.Search

	if search != "" && h.ftsManager != nil {
		countOptions := &query.QueryOptions{Filter: filter, Search: search}
		countDB := query.ApplyQueryOptionsWithFTS(baseDB, countOptions, h.metadata, h.ftsManager, h.metadata.TableName, h.logger)
		if searchAppliedAtDB(countDB) {
			var count int64
			if err := countDB.Count(&count).Error; err != nil {
				return 0, err
			}

			return count, nil
		}
	}

	countDB := baseDB
	if filter != nil {
		countDB = query.ApplyFilterOnly(countDB, filter, h.metadata, h.logger)
	}

	if search == "" {
		var count int64
		if err := countDB.Count(&count).Error; err != nil {
			return 0, err
		}

		return count, nil
	}

	selectColumns := searchableCountColumns(h.metadata)
	if len(selectColumns) > 0 {
		countDB = countDB.Select(selectColumns)
	}

	sliceType := reflect.SliceOf(h.metadata.EntityType)
	results := reflect.New(sliceType).Interface()

	if err := fastscan.Find(countDB, results); err != nil {
		return 0, err
	}

	sliceValue := reflect.ValueOf(results).Elem().Interface()
	filtered := query.ApplySearch(sliceValue, search, h.metadata)
	count := int64(reflect.ValueOf(filtered).Len())

	return count, nil
}

// countTransformedEntities returns the number of rows produced by a $apply pipeline
// (honoring groupby/aggregate/filter transformations and any top-level $filter applied
// against the transformed result), ignoring $top/$skip since a count reflects the total
// matching set rather than a single page.
func (h *EntityHandler) countTransformedEntities(ctx context.Context, queryOptions *query.QueryOptions, baseDB *gorm.DB) (int64, error) {
	countOptions := &query.QueryOptions{
		Apply:  queryOptions.Apply,
		Filter: queryOptions.Filter,
	}
	countDB := query.ApplyQueryOptionsWithFTS(baseDB.WithContext(ctx), countOptions, h.metadata, nil, "", h.logger)

	var results []map[string]interface{}
	if err := countDB.Find(&results).Error; err != nil {
		return 0, err
	}

	if rollupGroupBy, ok := query.GetRollupGroupByFromDB(countDB); ok {
		rolled, err := applyMapGroupByRollup(results, rollupGroupBy)
		if err != nil {
			return 0, err
		}
		return int64(len(rolled)), nil
	}

	return int64(len(results)), nil
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
