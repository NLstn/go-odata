package query

import (
	"fmt"

	"github.com/nlstn/go-odata/internal/metadata"
)

const defaultMaxExpandLevels = 5

func applyExpandLevels(expandOptions []ExpandOption, entityMetadata *metadata.EntityMetadata, config *ParserConfig) ([]ExpandOption, error) {
	if len(expandOptions) == 0 || entityMetadata == nil {
		return expandOptions, nil
	}

	limit := maxExpandLevels(config)
	expanded := make([]ExpandOption, len(expandOptions))
	for i := range expandOptions {
		opt := expandOptions[i]

		var targetMetadata *metadata.EntityMetadata
		if opt.Levels != nil || len(opt.Expand) > 0 {
			var err error
			targetMetadata, err = entityMetadata.ResolveNavigationTarget(opt.NavigationProperty)
			if err != nil {
				return nil, err
			}
		}

		if len(opt.Expand) > 0 {
			nested, err := applyExpandLevels(opt.Expand, targetMetadata, config)
			if err != nil {
				return nil, err
			}
			opt.Expand = nested
		}

		if opt.Levels != nil {
			levels, err := resolveExpandLevels(*opt.Levels, limit)
			if err != nil {
				return nil, err
			}
			opt, err = buildLevelsExpansion(opt, opt.Expand, entityMetadata, levels)
			if err != nil {
				return nil, err
			}
		}

		expanded[i] = opt
	}

	return expanded, nil
}

func maxExpandLevels(config *ParserConfig) int {
	if config != nil && config.MaxExpandDepth > 0 {
		return config.MaxExpandDepth
	}
	return defaultMaxExpandLevels
}

func resolveExpandLevels(levels int, limit int) (int, error) {
	if levels == -1 {
		if limit < 1 {
			return 0, errLevelsMaxRequiresDepth
		}
		return limit, nil
	}
	if levels < 1 {
		return 0, errLevelsMustBeIntOrMax
	}
	if limit > 0 && levels > limit {
		return 0, fmt.Errorf("$levels value (%d) exceeds maximum allowed depth (%d)", levels, limit)
	}
	return levels, nil
}

func buildLevelsExpansion(opt ExpandOption, baseNested []ExpandOption, currentMetadata *metadata.EntityMetadata, levels int) (ExpandOption, error) {
	opt.Expand = cloneExpandOptions(baseNested)
	opt.Levels = nil

	if levels <= 1 || currentMetadata == nil {
		return opt, nil
	}

	targetMetadata, err := currentMetadata.ResolveNavigationTarget(opt.NavigationProperty)
	if err != nil {
		return opt, err
	}

	if targetMetadata.FindNavigationProperty(opt.NavigationProperty) == nil {
		return opt, nil
	}

	nextOpt, err := buildLevelsExpansion(opt, baseNested, targetMetadata, levels-1)
	if err != nil {
		return opt, err
	}

	opt.Expand = append(opt.Expand, nextOpt)
	return opt, nil
}

func cloneExpandOptions(options []ExpandOption) []ExpandOption {
	if len(options) == 0 {
		return nil
	}
	clone := make([]ExpandOption, len(options))
	for i := range options {
		clone[i] = options[i]

		// Deep copy pointer fields to ensure proper isolation
		if options[i].Top != nil {
			topValue := *options[i].Top
			clone[i].Top = &topValue
		}
		if options[i].Skip != nil {
			skipValue := *options[i].Skip
			clone[i].Skip = &skipValue
		}
		if options[i].Levels != nil {
			levelsValue := *options[i].Levels
			clone[i].Levels = &levelsValue
		}
		if options[i].Filter != nil {
			clone[i].Filter = cloneFilterExpression(options[i].Filter)
		}
		if options[i].Compute != nil {
			clone[i].Compute = cloneComputeTransformation(options[i].Compute)
		}

		// Recursively clone nested expand options
		clone[i].Expand = cloneExpandOptions(options[i].Expand)

		// Deep copy Select slice
		if len(options[i].Select) > 0 {
			clone[i].Select = make([]string, len(options[i].Select))
			copy(clone[i].Select, options[i].Select)
		}

		// Deep copy OrderBy slice
		if len(options[i].OrderBy) > 0 {
			clone[i].OrderBy = make([]OrderByItem, len(options[i].OrderBy))
			copy(clone[i].OrderBy, options[i].OrderBy)
		}
	}
	return clone
}

func cloneFilterExpression(filter *FilterExpression) *FilterExpression {
	if filter == nil {
		return nil
	}
	clone := acquireFilterExpression()
	clone.Property = filter.Property
	clone.Operator = filter.Operator
	clone.Value = filter.Value
	clone.Logical = filter.Logical
	clone.IsNot = filter.IsNot
	clone.maxInClauseSize = filter.maxInClauseSize
	if filter.Left != nil {
		clone.Left = cloneFilterExpression(filter.Left)
	}
	if filter.Right != nil {
		clone.Right = cloneFilterExpression(filter.Right)
	}
	return clone
}

func cloneComputeTransformation(compute *ComputeTransformation) *ComputeTransformation {
	if compute == nil {
		return nil
	}
	clone := &ComputeTransformation{}
	if len(compute.Expressions) > 0 {
		clone.Expressions = make([]ComputeExpression, len(compute.Expressions))
		copy(clone.Expressions, compute.Expressions)
	}
	return clone
}
