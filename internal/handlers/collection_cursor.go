package handlers

import (
	"fmt"

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
	var prefix, branches []clause.Expression
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
