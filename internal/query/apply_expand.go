package query

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// applyExpand applies expand (preload) options to the GORM query
func applyExpand(db *gorm.DB, expand []ExpandOption, entityMetadata *metadata.EntityMetadata) *gorm.DB {
	for _, expandOpt := range expand {
		navProp := findNavigationProperty(expandOpt.NavigationProperty, entityMetadata)
		if navProp == nil {
			continue
		}

		var targetMetadata *metadata.EntityMetadata
		if entityMetadata != nil {
			target, err := entityMetadata.ResolveNavigationTarget(expandOpt.NavigationProperty)
			if err == nil {
				targetMetadata = target
			}
		}

		if needsPreloadCallback(expandOpt) {
			db = db.Preload(navProp.Name, func(db *gorm.DB) *gorm.DB {
				return applyExpandCallback(db, expandOpt, targetMetadata)
			})
		} else {
			db = db.Preload(navProp.Name)
		}
	}
	return db
}

// needsPreloadCallback checks if an expand option requires a preload callback
func needsPreloadCallback(expandOpt ExpandOption) bool {
	return expandOpt.Select != nil || expandOpt.Filter != nil || expandOpt.OrderBy != nil ||
		expandOpt.Top != nil || expandOpt.Skip != nil || len(expandOpt.Expand) > 0 ||
		expandOpt.Compute != nil
}

// applyExpandCallback applies the expand options within a GORM preload callback
func applyExpandCallback(db *gorm.DB, expandOpt ExpandOption, targetMetadata *metadata.EntityMetadata) *gorm.DB {
	if expandOpt.Filter != nil {
		db = applyFilter(db, expandOpt.Filter, targetMetadata)
	}

	if len(expandOpt.OrderBy) > 0 {
		dialect := getDatabaseDialect(db)
		for _, item := range expandOpt.OrderBy {
			direction := "ASC"
			if item.Descending {
				direction = "DESC"
			}
			columnName := GetColumnName(item.Property, targetMetadata)
			quotedColumn := quoteColumnReference(dialect, columnName)
			db = db.Order(fmt.Sprintf("%s %s", quotedColumn, direction))
		}
	}

	if expandOpt.Skip != nil {
		db = applyOffsetWithLimit(db, *expandOpt.Skip, expandOpt.Top)
	}
	if expandOpt.Top != nil {
		db = db.Limit(*expandOpt.Top)
	}

	// Recursively apply nested expand options
	if len(expandOpt.Expand) > 0 {
		db = applyExpand(db, expandOpt.Expand, targetMetadata)
	}

	return db
}

func quoteColumnReference(dialect string, column string) string {
	if column == "" {
		return column
	}
	if strings.Contains(column, ".") {
		parts := strings.Split(column, ".")
		for i, part := range parts {
			parts[i] = quoteIdent(dialect, part)
		}
		return strings.Join(parts, ".")
	}
	return quoteIdent(dialect, column)
}
