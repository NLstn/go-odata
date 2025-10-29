package query

import (
	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// ShouldUseMapResults returns true if the query options require map results instead of entity results
func ShouldUseMapResults(options *QueryOptions) bool {
	return options != nil && (len(options.Apply) > 0 || options.Compute != nil)
}

// ApplyQueryOptions applies parsed query options to a GORM database query
func ApplyQueryOptions(db *gorm.DB, options *QueryOptions, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	if options == nil {
		return db
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
			db = db.Offset(*options.Skip)
		}

		if options.Top != nil {
			db = db.Limit(*options.Top)
		}

		return db
	}

	// Apply standalone compute transformation (before select)
	if options.Compute != nil {
		db = applyCompute(db, options.Compute, entityMetadata)
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
	if options.Skip != nil {
		db = db.Offset(*options.Skip)
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
