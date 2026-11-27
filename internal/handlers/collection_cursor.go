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

func collectionCursorSelect(meta *metadata.EntityMetadata, selected []string, order []query.OrderByItem) []string {
	result := make([]string, 0, len(selected)+len(order))
	result = append(result, selected...)
	for _, item := range order {
		if prop := meta.FindProperty(item.Property); prop != nil && prop.ColumnName != "" && !prop.IsNavigationProp && !prop.IsComputed && !prop.IsStream && !prop.IsComplexType {
			result = append(result, prop.JsonName)
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
	prefix := make([]clause.Expression, 0, len(order))
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
		if !ok {
			return nil, fmt.Errorf("missing skiptoken value for %q", item.Property)
		}
		column := clause.Column{Table: clause.CurrentTable, Name: property.ColumnName}
		// NULL sorts first ascending and last descending. At the last NULL
		// partition only later terms can advance the cursor.
		var after clause.Expression
		switch {
		case value == nil && !item.Descending:
			after = clause.Neq{Column: column, Value: nil}
		case value != nil && item.Descending:
			after = clause.Or(clause.Lt{Column: column, Value: value}, clause.Eq{Column: column, Value: nil})
		case value != nil:
			after = clause.Gt{Column: column, Value: value}
		}
		if after != nil {
			branch := make([]clause.Expression, 0, len(prefix)+1)
			branch = append(branch, prefix...)
			branches = append(branches, clause.And(append(branch, after)...))
		}
		prefix = append(prefix, clause.Eq{Column: column, Value: value})
	}
	if len(branches) == 0 {
		return db.Where("1 = 0"), nil
	}
	return db.Where(clause.Or(branches...)), nil
}

func plainCollectionGroupBy(apply []query.ApplyTransformation) []string {
	if len(apply) == 0 {
		return nil
	}
	last := apply[len(apply)-1]
	if last.Type != query.ApplyTypeGroupBy || last.GroupBy == nil || last.GroupBy.Rollup != nil || last.GroupBy.AllValues {
		return nil
	}
	for _, key := range last.GroupBy.Properties {
		if key == "" || strings.Contains(key, "/") {
			return nil
		}
	}
	return last.GroupBy.Properties
}

// groupedCursorOrder completes the ordering for offset paging over plain
// groupby results. Entity keys alone do not identify the cursor position.
func groupedCursorOrder(apply []query.ApplyTransformation, order []query.OrderByItem) []query.OrderByItem {
	keys := plainCollectionGroupBy(apply)
	if len(keys) == 0 {
		return order
	}
	result := make([]query.OrderByItem, 0, len(order)+len(keys))
	result = append(result, order...)
	have := make(map[string]bool, len(result))
	for _, item := range result {
		have[item.Property] = true
	}
	for _, key := range keys {
		if have[key] {
			continue
		}
		result = append(result, query.OrderByItem{Property: key})
		have[key] = true
	}
	return result
}
