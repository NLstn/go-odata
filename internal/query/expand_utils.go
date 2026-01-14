package query

import "strings"

// FindExpandOption returns the expand option matching the provided property names.
func FindExpandOption(expandOptions []ExpandOption, propName, jsonName string) *ExpandOption {
	if len(expandOptions) == 0 {
		return nil
	}

	for i := range expandOptions {
		nav := expandOptions[i].NavigationProperty
		if strings.EqualFold(nav, propName) || (jsonName != "" && strings.EqualFold(nav, jsonName)) {
			return &expandOptions[i]
		}
	}

	return nil
}
