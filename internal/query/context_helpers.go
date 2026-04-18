package query

// ContextPropertiesFromApply computes the output property names that will appear in the
// result of the supplied $apply transformations. These properties are used to build the
// OData context URL fragment, e.g. #Products(CategoryID,Count).
//
// Rules applied:
//   - groupby properties are included as-is.
//   - Aggregate/nested-aggregate aliases are included.
//   - filter/compute-only pipelines return nil (shape unchanged from base entity set).
//
// Returns nil when no deterministic reduced output shape can be derived.
func ContextPropertiesFromApply(transformations []ApplyTransformation) []string {
	if len(transformations) == 0 {
		return nil
	}

	var properties []string
	propSet := make(map[string]bool)

	hasShapeChange := false

	for _, t := range transformations {
		switch t.Type {
		case ApplyTypeGroupBy:
			hasShapeChange = true
			if t.GroupBy != nil {
				for _, p := range t.GroupBy.Properties {
					if !propSet[p] {
						propSet[p] = true
						properties = append(properties, p)
					}
				}
				// Include rollup properties in the output shape.
				if t.GroupBy.Rollup != nil {
					for _, p := range t.GroupBy.Rollup.Properties {
						if !propSet[p] {
							propSet[p] = true
							properties = append(properties, p)
						}
					}
				}
				// Collect aggregate aliases from nested transforms inside groupby
				for _, nested := range t.GroupBy.Transform {
					if nested.Type == ApplyTypeAggregate && nested.Aggregate != nil {
						for _, expr := range nested.Aggregate.Expressions {
							if expr.Alias != "" && !propSet[expr.Alias] {
								propSet[expr.Alias] = true
								properties = append(properties, expr.Alias)
							}
						}
					}
				}
			}
		case ApplyTypeAggregate:
			hasShapeChange = true
			if t.Aggregate != nil {
				for _, expr := range t.Aggregate.Expressions {
					if expr.Alias != "" && !propSet[expr.Alias] {
						propSet[expr.Alias] = true
						properties = append(properties, expr.Alias)
					}
				}
			}
		case ApplyTypeJoin, ApplyTypeOuterJoin:
			hasShapeChange = true
			if t.Join != nil && t.Join.Alias != "" {
				contextProp := t.Join.Alias + "()"
				if !propSet[contextProp] {
					propSet[contextProp] = true
					properties = append(properties, contextProp)
				}
			}
			// filter and compute do not change the output shape
		}
	}

	if !hasShapeChange {
		return nil
	}

	return properties
}
