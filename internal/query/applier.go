package query

import (
	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// applyOffsetWithLimit applies OFFSET to the query and adds a LIMIT for MySQL/MariaDB compatibility
// when no explicit top limit is provided
func applyOffsetWithLimit(db *gorm.DB, skip int, top *int) *gorm.DB {
	db = db.Offset(skip)
	// If no explicit top is set, use a very large limit for MySQL/MariaDB compatibility
	if top == nil {
		dialect := getDatabaseDialect(db)
		if dialect == "mysql" {
			// MySQL/MariaDB require LIMIT when OFFSET is used
			// Use max int32 value which is effectively unlimited
			maxLimit := 2147483647
			db = db.Limit(maxLimit)
		}
	}
	return db
}

// ShouldUseMapResults returns true if the query options require map results instead of entity results
func ShouldUseMapResults(options *QueryOptions) bool {
	return options != nil && (len(options.Apply) > 0 || options.Compute != nil)
}

// ApplyQueryOptions applies parsed query options to a GORM database query
func ApplyQueryOptions(db *gorm.DB, options *QueryOptions, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	return ApplyQueryOptionsWithFTS(db, options, entityMetadata, nil, "")
}

// ApplyQueryOptionsWithFTS applies parsed query options to a GORM database query with optional FTS support
func ApplyQueryOptionsWithFTS(db *gorm.DB, options *QueryOptions, entityMetadata *metadata.EntityMetadata, ftsManager *FTSManager, tableName string) *gorm.DB {
	if options == nil {
		return db
	}

	dialect := getDatabaseDialect(db)

	// Try to apply search at database level using FTS if available
	searchAppliedAtDB := false
	if options.Search != "" && ftsManager != nil && ftsManager.IsFTSAvailable() && tableName != "" {
		var err error
		db, err = ftsManager.ApplyFTSSearch(db, tableName, options.Search, entityMetadata)
		if err == nil {
			searchAppliedAtDB = true
		}
		// If FTS fails, we'll fall back to in-memory search
	}

	// Store whether search was applied at DB level for later use
	if searchAppliedAtDB {
		// Mark that search was already applied so it doesn't get applied again in-memory
		db = db.Set("_fts_search_applied", true)
	}

	// Apply transformations (if present, they take precedence over other query options)
	if len(options.Apply) > 0 {
		db = applyTransformations(db, options.Apply, entityMetadata)

		// After applying transformations, still apply orderby, top, and skip
		// These are standalone query options that work with $apply
		if len(options.OrderBy) > 0 {
			db = applyOrderBy(db, options.OrderBy, entityMetadata)
		}

		if options.Skip != nil {
			db = applyOffsetWithLimit(db, *options.Skip, options.Top)
		}

		if options.Top != nil {
			db = db.Limit(*options.Top)
		}

		return db
	}

	// Apply standalone compute transformation (before select)
	if options.Compute != nil {
		// Reset alias expressions map for compute aliases used in $filter
		resetAliasExprs()
		db = applyCompute(db, dialect, options.Compute, entityMetadata)
	}

	// Apply select at database level to fetch only needed columns
	// Skip this if compute is present, as compute handles the select clause
	if len(options.Select) > 0 && options.Compute == nil {
		db = applySelect(db, options.Select, entityMetadata)
	}

	// Apply filter
	if options.Filter != nil {
		db = applyFilter(db, options.Filter, entityMetadata)
	}

	// Apply expand (preload navigation properties)
	if len(options.Expand) > 0 {
		db = applyExpand(db, options.Expand, entityMetadata)
	}

	// Apply orderby
	if len(options.OrderBy) > 0 {
		db = applyOrderBy(db, options.OrderBy, entityMetadata)
	}

	// Apply top and skip (for pagination)
	// MySQL/MariaDB require LIMIT when using OFFSET, so we ensure a LIMIT is always present
	if options.Skip != nil {
		db = applyOffsetWithLimit(db, *options.Skip, options.Top)
	}

	if options.Top != nil {
		db = db.Limit(*options.Top)
	}

	return db
}

// ApplyFilterOnly applies only the filter expression to a GORM database query
// This is useful for getting counts without applying pagination
func ApplyFilterOnly(db *gorm.DB, filter *FilterExpression, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if filter == nil {
		return db
	}
	return applyFilter(db, filter, entityMetadata)
}

// ApplyExpandOnly applies only the expand (preload) options to a GORM database query
// This is useful for loading related entities without applying other query options
func ApplyExpandOnly(db *gorm.DB, expand []ExpandOption, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if len(expand) == 0 {
		return db
	}
	return applyExpand(db, expand, entityMetadata)
}
