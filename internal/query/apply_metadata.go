package query

import (
	"github.com/nlstn/go-odata/internal/metadata"
	"reflect"
)

func applyAliasProperty(alias string, expr *FilterExpression, source *metadata.EntityMetadata) metadata.PropertyMetadata {
	p := metadata.PropertyMetadata{Name: alias, JsonName: alias, FieldName: alias, ColumnName: alias, Type: reflect.TypeOf(float64(0)), EdmType: metadata.PrimitiveTypeDouble}
	if expr != nil && expr.Property != "" {
		if original := source.FindProperty(expr.Property); original != nil {
			p.Type = original.Type
			p.EdmType = original.EdmType
		}
	}
	return p
}

// applyOutputMetadata never mutates the registered model shared by requests.
func applyOutputMetadata(source *metadata.EntityMetadata, tr ApplyTransformation) *metadata.EntityMetadata {
	if source == nil {
		return nil
	}
	copy := *source
	copy.Properties = append([]metadata.PropertyMetadata(nil), source.Properties...)
	switch tr.Type {
	case ApplyTypeCompute:
		for _, e := range tr.Compute.Expressions {
			copy.Properties = append(copy.Properties, applyAliasProperty(e.Alias, e.Expression, source))
		}
	case ApplyTypeAggregate:
		copy.Properties = nil
		for _, e := range tr.Aggregate.Expressions {
			copy.Properties = append(copy.Properties, applyAliasProperty(e.Alias, &FilterExpression{Property: e.Property}, source))
		}
	case ApplyTypeGroupBy:
		if tr.GroupBy.Rollup != nil {
			return source
		}
		copy.Properties = nil
		for _, name := range tr.GroupBy.Properties {
			if p := source.FindProperty(name); p != nil {
				copy.Properties = append(copy.Properties, *p)
			}
		}
		for _, nested := range tr.GroupBy.Transform {
			if nested.Aggregate != nil {
				for _, e := range nested.Aggregate.Expressions {
					copy.Properties = append(copy.Properties, applyAliasProperty(e.Alias, &FilterExpression{Property: e.Property}, source))
				}
			}
		}
	}
	return &copy
}
