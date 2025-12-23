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
		{
			name:            "Single string key with single quotes",
			path:            "UserSessions('1f8d7b3b-3271-4ce2-8ea9-a875ad35bbd6')",
			expectEntitySet: "UserSessions",
			expectKeyMap:    map[string]string{},
			expectKey:       "1f8d7b3b-3271-4ce2-8ea9-a875ad35bbd6", // Quotes should be stripped
		},
		{
			name:            "Single string key with double quotes",
			path:            `UserSessions("abc-def-ghi")`,
			expectEntitySet: "UserSessions",
			expectKeyMap:    map[string]string{},
			expectKey:       "abc-def-ghi", // Quotes should be stripped
		},
		{
			name:            "Single string key with spaces and quotes",
			path:            "Users('John Doe')",
			expectEntitySet: "Users",
			expectKeyMap:    map[string]string{},
			expectKey:       "John Doe", // Quotes should be stripped
		},
		{
			name:            "Single string key with mismatched quotes should not strip",
			path:            "Keys('value\")",
			expectEntitySet: "Keys",
			expectKeyMap:    map[string]string{},
			expectKey:       "'value\"", // Quotes should NOT be stripped (mismatched)
		},
		{
			name:            "Single string key with embedded quotes",
			path:            "Keys('a\"b')",
			expectEntitySet: "Keys",
			expectKeyMap:    map[string]string{},
			expectKey:       "a\"b", // Only outer quotes should be stripped
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

func TestParseODataURLComponentsCount(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		expectEntitySet string
		expectIsCount   bool
		expectHasKey    bool
	}{
		{
			name:            "Collection count",
			path:            "Products/$count",
			expectEntitySet: "Products",
			expectIsCount:   true,
			expectHasKey:    false,
		},
		{
			name:            "Collection without count",
			path:            "Products",
			expectEntitySet: "Products",
			expectIsCount:   false,
			expectHasKey:    false,
		},
		{
			name:            "Navigation property (not count)",
			path:            "Products(1)/Descriptions",
			expectEntitySet: "Products",
			expectIsCount:   false,
			expectHasKey:    true,
		},
		{
			name:            "Leading slash",
			path:            "/Products/$count",
			expectEntitySet: "Products",
			expectIsCount:   true,
			expectHasKey:    false,
		},
		{
			name:            "Navigation property with count",
			path:            "Products(1)/Descriptions/$count",
			expectEntitySet: "Products",
			expectIsCount:   true,
			expectHasKey:    true,
		},
		{
			name:            "Navigation property with count - leading slash",
			path:            "/Products(1)/Descriptions/$count",
			expectEntitySet: "Products",
			expectIsCount:   true,
			expectHasKey:    true,
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

			if components.IsCount != tt.expectIsCount {
				t.Errorf("Expected IsCount %v, got %v", tt.expectIsCount, components.IsCount)
			}

			hasKey := components.EntityKey != "" || len(components.EntityKeyMap) > 0
			if hasKey != tt.expectHasKey {
				t.Errorf("Expected HasKey %v, got %v", tt.expectHasKey, hasKey)
			}
		})
	}
}

func TestParseODataURLComponentsRef(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		expectEntitySet string
		expectIsRef     bool
		expectHasKey    bool
		expectNavProp   string
	}{
		{
			name:            "Entity reference",
			path:            "Products(1)/$ref",
			expectEntitySet: "Products",
			expectIsRef:     true,
			expectHasKey:    true,
			expectNavProp:   "",
		},
		{
			name:            "Navigation property reference",
			path:            "Products(1)/Descriptions/$ref",
			expectEntitySet: "Products",
			expectIsRef:     true,
			expectHasKey:    true,
			expectNavProp:   "Descriptions",
		},
		{
			name:            "Collection reference",
			path:            "Products/$ref",
			expectEntitySet: "Products",
			expectIsRef:     true,
			expectHasKey:    false,
			expectNavProp:   "",
		},
		{
			name:            "Leading slash",
			path:            "/Products(1)/Descriptions/$ref",
			expectEntitySet: "Products",
			expectIsRef:     true,
			expectHasKey:    true,
			expectNavProp:   "Descriptions",
		},
		{
			name:            "Without $ref",
			path:            "Products(1)/Descriptions",
			expectEntitySet: "Products",
			expectIsRef:     false,
			expectHasKey:    true,
			expectNavProp:   "Descriptions",
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

			if components.IsRef != tt.expectIsRef {
				t.Errorf("Expected IsRef %v, got %v", tt.expectIsRef, components.IsRef)
			}

			hasKey := components.EntityKey != "" || len(components.EntityKeyMap) > 0
			if hasKey != tt.expectHasKey {
				t.Errorf("Expected HasKey %v, got %v", tt.expectHasKey, hasKey)
			}

			if components.NavigationProperty != tt.expectNavProp {
				t.Errorf("Expected NavigationProperty %s, got %s", tt.expectNavProp, components.NavigationProperty)
			}
		})
	}
}

func TestParseODataURLComponentsTypeCast(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		expectEntitySet string
		expectTypeCast  string
		expectHasKey    bool
		expectKey       string
		expectNavProp   string
	}{
		{
			name:            "Type cast on collection",
			path:            "Products/Namespace.SpecialProduct",
			expectEntitySet: "Products",
			expectTypeCast:  "Namespace.SpecialProduct",
			expectHasKey:    false,
		},
		{
			name:            "Type cast on single entity",
			path:            "Products(1)/Namespace.SpecialProduct",
			expectEntitySet: "Products",
			expectTypeCast:  "Namespace.SpecialProduct",
			expectHasKey:    true,
			expectKey:       "1",
		},
		{
			name:            "Type cast with property access",
			path:            "Products(1)/Namespace.SpecialProduct/SpecialProperty",
			expectEntitySet: "Products",
			expectTypeCast:  "Namespace.SpecialProduct",
			expectHasKey:    true,
			expectKey:       "1",
			expectNavProp:   "SpecialProperty",
		},
		{
			name:            "Type cast with navigation property",
			path:            "Products(1)/Namespace.SpecialProduct/Category",
			expectEntitySet: "Products",
			expectTypeCast:  "Namespace.SpecialProduct",
			expectHasKey:    true,
			expectKey:       "1",
			expectNavProp:   "Category",
		},
		{
			name:            "Type cast with fully qualified namespace",
			path:            "Products/ComplianceService.SpecialProduct",
			expectEntitySet: "Products",
			expectTypeCast:  "ComplianceService.SpecialProduct",
			expectHasKey:    false,
		},
		{
			name:            "No type cast - regular path",
			path:            "Products(1)/Category",
			expectEntitySet: "Products",
			expectTypeCast:  "",
			expectHasKey:    true,
			expectKey:       "1",
			expectNavProp:   "Category",
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

			if components.TypeCast != tt.expectTypeCast {
				t.Errorf("Expected TypeCast %s, got %s", tt.expectTypeCast, components.TypeCast)
			}

			hasKey := components.EntityKey != "" || len(components.EntityKeyMap) > 0
			if hasKey != tt.expectHasKey {
				t.Errorf("Expected HasKey %v, got %v", tt.expectHasKey, hasKey)
			}

			if tt.expectKey != "" && components.EntityKey != tt.expectKey {
				t.Errorf("Expected EntityKey %s, got %s", tt.expectKey, components.EntityKey)
			}

			if components.NavigationProperty != tt.expectNavProp {
				t.Errorf("Expected NavigationProperty %s, got %s", tt.expectNavProp, components.NavigationProperty)
			}
		})
	}
}

func TestParseODataURLComponentsEmptyPathSegments(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "Consecutive slashes in path",
			path:        "/Products//",
			expectError: true,
		},
		{
			name:        "Consecutive slashes with entity key",
			path:        "/Products//1",
			expectError: true,
		},
		{
			name:        "Multiple consecutive slashes at start",
			path:        "///Products",
			expectError: true, // After stripping leading slash, //Products splits to ["", "", "Products"] - consecutive empty segments
		},
		{
			name:        "Empty segment in middle",
			path:        "/Products//Details",
			expectError: true,
		},
		{
			name:        "Valid path with trailing slash",
			path:        "/Products/",
			expectError: false,
		},
		{
			name:        "Valid path with leading slash",
			path:        "/Products",
			expectError: false,
		},
		{
			name:        "Valid path without slashes",
			path:        "Products",
			expectError: false,
		},
		{
			name:        "Valid path with entity key",
			path:        "/Products(1)",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseODataURLComponents(tt.path)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for path %s, but got none", tt.path)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for path %s, but got: %v", tt.path, err)
			}
		})
	}
}
