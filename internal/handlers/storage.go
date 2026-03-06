package handlers

import (
	"context"
	"reflect"

	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/gorm"
)

// Storage defines the persistence primitives used by entity handlers.
//
// The default implementation (DBStorage) preserves existing GORM behavior and
// provides a seam for cache/write-behind implementations.
type Storage interface {
	FetchEntityByKey(ctx context.Context, h *EntityHandler, entityKey string, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error)
	FetchCollection(ctx context.Context, h *EntityHandler, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error)
	CountEntities(ctx context.Context, h *EntityHandler, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (int64, error)
	Create(tx *gorm.DB, entity interface{}) error
	UpdatePartial(tx *gorm.DB, entity interface{}, updateData map[string]interface{}) error
	UpdateFull(tx *gorm.DB, entity interface{}, replacement interface{}) error
	Delete(tx *gorm.DB, entity interface{}) error
	Refresh(tx *gorm.DB, entity interface{}) error
}

// DBStorage is the default Storage implementation backed by GORM only.
type DBStorage struct{}

// NewDBStorage returns the default DB-backed storage implementation.
func NewDBStorage() Storage {
	return &DBStorage{}
}

func (s *DBStorage) FetchEntityByKey(ctx context.Context, h *EntityHandler, entityKey string, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	result := reflect.New(h.metadata.EntityType).Interface()

	db := h.db.WithContext(ctx)
	if len(scopes) > 0 {
		db = db.Scopes(scopes...)
	}
	baseDB := db

	db, err := h.buildKeyQuery(db, entityKey)
	if err != nil {
		return nil, err
	}

	if queryOptions != nil && len(queryOptions.Expand) > 0 {
		db = query.ApplyExpandOnly(db, queryOptions.Expand, h.metadata, h.logger)
	}

	if err := db.First(result).Error; err != nil {
		return nil, err
	}

	if queryOptions != nil && len(queryOptions.Expand) > 0 {
		if err := query.ApplyPerParentExpand(baseDB, result, queryOptions.Expand, h.metadata); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *DBStorage) FetchCollection(ctx context.Context, h *EntityHandler, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	modifiedOptions := *queryOptions
	if queryOptions.Top != nil {
		topPlusOne := *queryOptions.Top + 1
		modifiedOptions.Top = &topPlusOne
	}

	db := h.db.WithContext(ctx)
	if len(scopes) > 0 {
		db = db.Scopes(scopes...)
	}
	baseDB := db

	if queryOptions.SkipToken != nil {
		db = h.applySkipTokenFilter(db, queryOptions)
	}

	tableName := h.metadata.TableName
	db = query.ApplyQueryOptionsWithFTS(db, &modifiedOptions, h.metadata, h.ftsManager, tableName, h.logger)

	searchAppliedAtDB := false
	if val, ok := db.Get("_fts_search_applied"); ok {
		if applied, ok := val.(bool); ok && applied {
			searchAppliedAtDB = true
		}
	}

	if query.ShouldUseMapResults(queryOptions) {
		var results []map[string]interface{}
		if err := db.Find(&results).Error; err != nil {
			return nil, err
		}
		if len(queryOptions.Select) > 0 && queryOptions.Compute != nil {
			computedAliases := make(map[string]bool)
			for _, expr := range queryOptions.Compute.Expressions {
				computedAliases[expr.Alias] = true
			}
			results = query.ApplySelectToMapResults(results, queryOptions.Select, h.metadata, computedAliases)
		}
		return results, nil
	}

	sliceType := reflect.SliceOf(h.metadata.EntityType)
	slicePtr := reflect.New(sliceType)
	if modifiedOptions.Top != nil && *modifiedOptions.Top > 0 {
		slicePtr.Elem().Set(reflect.MakeSlice(sliceType, 0, *modifiedOptions.Top))
	}
	results := slicePtr.Interface()

	if err := db.Find(results).Error; err != nil {
		return nil, err
	}

	if len(queryOptions.Expand) > 0 {
		if err := query.ApplyPerParentExpand(baseDB, results, queryOptions.Expand, h.metadata); err != nil {
			return nil, err
		}
	}

	sliceValue := reflect.ValueOf(results).Elem().Interface()

	if queryOptions.Search != "" && !searchAppliedAtDB {
		sliceValue = query.ApplySearch(sliceValue, queryOptions.Search, h.metadata)
	}

	if len(queryOptions.Expand) > 0 {
		sliceValue = query.ApplyExpandComputeToResults(sliceValue, queryOptions.Expand)
	}

	if len(queryOptions.Select) > 0 {
		sliceValue = query.ApplySelect(sliceValue, queryOptions.Select, h.metadata, queryOptions.Expand)
	}

	return sliceValue, nil
}

func (s *DBStorage) CountEntities(ctx context.Context, h *EntityHandler, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (int64, error) {
	baseDB := h.db.WithContext(ctx).Model(reflect.New(h.metadata.EntityType).Interface())
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
	if err := countDB.Find(results).Error; err != nil {
		return 0, err
	}

	sliceValue := reflect.ValueOf(results).Elem().Interface()
	filtered := query.ApplySearch(sliceValue, search, h.metadata)
	count := int64(reflect.ValueOf(filtered).Len())

	return count, nil
}

func (s *DBStorage) Create(tx *gorm.DB, entity interface{}) error {
	return tx.Create(entity).Error
}

func (s *DBStorage) UpdatePartial(tx *gorm.DB, entity interface{}, updateData map[string]interface{}) error {
	return tx.Model(entity).Updates(updateData).Error
}

func (s *DBStorage) UpdateFull(tx *gorm.DB, entity interface{}, replacement interface{}) error {
	return tx.Model(entity).Select("*").Updates(replacement).Error
}

func (s *DBStorage) Delete(tx *gorm.DB, entity interface{}) error {
	return tx.Delete(entity).Error
}

func (s *DBStorage) Refresh(tx *gorm.DB, entity interface{}) error {
	return tx.First(entity).Error
}
