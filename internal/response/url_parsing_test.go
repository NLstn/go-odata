package response

import (
	"testing"
)

func TestParseODataURLComponentsCompositeKey(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		expectEntitySet string
		expectKeyMap    map[string]string
		expectKey       string
	}{
		{
			name:            "Composite key with quotes",
			path:            "ProductDescriptions(ProductID=1,LanguageKey='EN')",
			expectEntitySet: "ProductDescriptions",
			expectKeyMap: map[string]string{
				"ProductID":   "1",
				"LanguageKey": "EN",
			},
			expectKey: "",
		},
		{
			name:            "Composite key with double quotes",
			path:            `ProductDescriptions(ProductID=2,LanguageKey="DE")`,
			expectEntitySet: "ProductDescriptions",
			expectKeyMap: map[string]string{
				"ProductID":   "2",
				"LanguageKey": "DE",
			},
			expectKey: "",
		},
		{
			name:            "Single key value",
			path:            "Products(5)",
			expectEntitySet: "Products",
			expectKeyMap:    map[string]string{},
			expectKey:       "5",
		},
		{
			name:            "Single key with name",
			path:            "Products(ID=5)",
			expectEntitySet: "Products",
			expectKeyMap: map[string]string{
				"ID": "5",
			},
			expectKey: "5", // Should be set for backwards compatibility
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components, err := ParseODataURLComponents(tt.path)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if components.EntitySet != tt.expectEntitySet {
				t.Errorf("Expected entity set %s, got %s", tt.expectEntitySet, components.EntitySet)
			}

			if tt.expectKey != "" && components.EntityKey != tt.expectKey {
				t.Errorf("Expected EntityKey %s, got %s", tt.expectKey, components.EntityKey)
			}

			if len(tt.expectKeyMap) > 0 {
				if len(components.EntityKeyMap) != len(tt.expectKeyMap) {
					t.Errorf("Expected %d key-value pairs, got %d", len(tt.expectKeyMap), len(components.EntityKeyMap))
				}
				for key, expectedValue := range tt.expectKeyMap {
					actualValue, ok := components.EntityKeyMap[key]
					if !ok {
						t.Errorf("Expected key %s not found in EntityKeyMap", key)
					} else if actualValue != expectedValue {
						t.Errorf("For key %s, expected value %s, got %s", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}
