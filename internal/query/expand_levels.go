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
			return 0, fmt.Errorf("$levels=max requires a positive maximum expand depth")
		}
		return limit, nil
	}
	if levels < 1 {
		return 0, fmt.Errorf("$levels must be a positive integer or 'max'")
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
		clone[i].Expand = cloneExpandOptions(options[i].Expand)
	}
	return clone
}
