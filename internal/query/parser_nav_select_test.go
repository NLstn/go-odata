package query

import "testing"

// TestMergeNavigationSelectsPlainNavProperty tests that plain navigation properties
// in $select do not trigger implicit expands.
func TestMergeNavigationSelectsPlainNavProperty(t *testing.T) {
	tests := []struct {
		name             string
		selectProps      []string
		existingExpands  []ExpandOption
		expectedExpands  int
		expectedNavProp  string
		shouldAutoExpand bool
	}{
		{
			name:             "Plain navigation property should not auto-expand",
			selectProps:      []string{"Name", "Descriptions"},
			existingExpands:  []ExpandOption{},
			expectedExpands:  0,
			expectedNavProp:  "",
			shouldAutoExpand: false,
		},
		{
			name:             "Navigation property with existing expand should not duplicate",
			selectProps:      []string{"Name", "Descriptions"},
			existingExpands:  []ExpandOption{{NavigationProperty: "Descriptions"}},
			expectedExpands:  1,
			expectedNavProp:  "Descriptions",
			shouldAutoExpand: false,
		},
		{
			name:             "Navigation path should still work",
			selectProps:      []string{"Name", "Descriptions/Text"},
			existingExpands:  []ExpandOption{},
			expectedExpands:  1,
			expectedNavProp:  "Descriptions",
			shouldAutoExpand: true,
		},
		{
			name:             "Multiple plain navigation properties do not auto-expand",
			selectProps:      []string{"Name", "Descriptions"},
			existingExpands:  []ExpandOption{},
			expectedExpands:  0,
			expectedNavProp:  "",
			shouldAutoExpand: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &QueryOptions{
				Select: tt.selectProps,
				Expand: tt.existingExpands,
			}

			mergeNavigationSelects(options)

			if len(options.Expand) != tt.expectedExpands {
				t.Errorf("Expected %d expand options, got %d", tt.expectedExpands, len(options.Expand))
			}

			if tt.shouldAutoExpand && len(options.Expand) > 0 {
				found := false
				for _, expand := range options.Expand {
					if expand.NavigationProperty == tt.expectedNavProp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected navigation property '%s' to be in expand options", tt.expectedNavProp)
				}
			}
		})
	}
}

// TestMergeNavigationSelectsRegularProperty tests that regular (non-navigation) properties
// are not added to expand options
func TestMergeNavigationSelectsRegularProperty(t *testing.T) {
	options := &QueryOptions{
		Select: []string{"Name", "Price"},
		Expand: []ExpandOption{},
	}

	mergeNavigationSelects(options)

	if len(options.Expand) != 0 {
		t.Errorf("Expected 0 expand options for regular properties, got %d", len(options.Expand))
	}
}
