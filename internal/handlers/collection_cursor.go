package handlers

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/skiptoken"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// collectionCursorOrder keeps the requested sort and appends missing entity
// keys, in metadata order, as ascending tie-breakers.
func collectionCursorOrder(meta *metadata.EntityMetadata, order []query.OrderByItem) []query.OrderByItem {
	result := append([]query.OrderByItem(nil), order...)
	for _, key := range meta.KeyProperties {
		found := false
		for _, item := range order {
			if property := meta.FindProperty(item.Property); property != nil && property.ColumnName == key.ColumnName {
				found = true
				break
			}
		}
		if !found {
			result = append(result, query.OrderByItem{Property: key.JsonName})
		}
	}
	return result
}

func (h *EntityHandler) applyCollectionCursor(db *gorm.DB, opts *query.QueryOptions) (*gorm.DB, error) {
	token, err := skiptoken.Decode(*opts.SkipToken)
	if err != nil {
		return nil, err
	}
	order := collectionCursorOrder(h.metadata, opts.OrderBy)
	if len(order) == 0 {
		return nil, fmt.Errorf("skiptoken requires a deterministic order")
	}
	// Each branch matches the preceding components and advances the current
	// component in its declared direction.
	var prefix []clause.Expression
	branches := make([]clause.Expression, 0, len(order))
	for _, item := range order {
		property := h.metadata.FindProperty(item.Property)
		if property == nil || property.ColumnName == "" {
			return nil, fmt.Errorf("unsupported skiptoken sort property %q", item.Property)
		}
		value, ok := token.OrderByValues[item.Property]
		if !ok {
			value, ok = token.KeyValues[property.JsonName]
		}
		if !ok || value == nil {
			return nil, fmt.Errorf("missing skiptoken value for %q", item.Property)
		}
		column := clause.Column{Table: clause.CurrentTable, Name: property.ColumnName}
		var after clause.Expression = clause.Gt{Column: column, Value: value}
		if item.Descending {
			after = clause.Lt{Column: column, Value: value}
		}
		branch := append(append([]clause.Expression(nil), prefix...), after)
		branches = append(branches, clause.And(branch...))
		prefix = append(prefix, clause.Eq{Column: column, Value: value})
	}
	return db.Where(clause.Or(branches...)), nil
}

// groupedCursorOrder returns the order for a grouped ($apply ... groupby) result.
// OFFSET paging needs a total order. The requested sort may tie (for example on an
// aggregate), so the group keys are appended, in declared order, as ascending
// tie-breakers. When the client gave no sort, the group keys are the order.
// It only applies when the last transformation is a plain groupby (no rollup, no
// $all, no navigation paths); every other shape is returned unchanged.
func groupedCursorOrder(apply []query.ApplyTransformation, order []query.OrderByItem) []query.OrderByItem {
	if len(apply) == 0 {
		return order
	}
	last := apply[len(apply)-1]
	if last.Type != query.ApplyTypeGroupBy || last.GroupBy == nil || last.GroupBy.Rollup != nil || last.GroupBy.AllValues {
		return order
	}
	result := append([]query.OrderByItem(nil), order...)
	have := make(map[string]bool, len(result))
	for _, item := range result {
		have[item.Property] = true
	}
	for _, key := range last.GroupBy.Properties {
		if key == "" || strings.Contains(key, "/") || have[key] {
			return order
		}
		result = append(result, query.OrderByItem{Property: key})
		have[key] = true
	}
	return result
}
