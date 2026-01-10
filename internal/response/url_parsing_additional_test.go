package response

import (
	"testing"
)

func TestParseODataURLComponents_Basic(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantEntitySet string
		wantEntityKey string
		wantNavProp   string
		wantIsCount   bool
		wantIsValue   bool
		wantIsRef     bool
		wantTypeCast  string
		wantErr       bool
	}{
		{
			name:          "Simple entity set",
			path:          "/Products",
			wantEntitySet: "Products",
		},
		{
			name:          "Entity set with single key",
			path:          "/Products(1)",
			wantEntitySet: "Products",
			wantEntityKey: "1",
		},
		{
			name:          "Entity set with string key",
			path:          "/Products('abc')",
			wantEntitySet: "Products",
			wantEntityKey: "abc",
		},
		{
			name:          "Entity set with double-quoted key",
			path:          `/Products("abc")`,
			wantEntitySet: "Products",
			wantEntityKey: "abc",
		},
		{
			name:          "Entity set with GUID key",
			path:          "/Products(12345678-1234-1234-1234-123456789012)",
			wantEntitySet: "Products",
			wantEntityKey: "12345678-1234-1234-1234-123456789012",
		},
		{
			name:          "Navigation property",
			path:          "/Products(1)/Category",
			wantEntitySet: "Products",
			wantEntityKey: "1",
			wantNavProp:   "Category",
		},
		{
			name:          "Collection $count",
			path:          "/Products/$count",
			wantEntitySet: "Products",
			wantIsCount:   true,
		},
		{
			name:          "Entity $count",
			path:          "/Products(1)/Orders/$count",
			wantEntitySet: "Products",
			wantEntityKey: "1",
			wantNavProp:   "Orders",
			wantIsCount:   true,
		},
		{
			name:          "Property $value",
			path:          "/Products(1)/Name/$value",
			wantEntitySet: "Products",
			wantEntityKey: "1",
			wantNavProp:   "Name",
			wantIsValue:   true,
		},
		{
			name:          "Entity $ref",
			path:          "/Products(1)/Category/$ref",
			wantEntitySet: "Products",
			wantEntityKey: "1",
			wantNavProp:   "Category",
			wantIsRef:     true,
		},
		{
			name:          "Type cast",
			path:          "/Products(1)/ODataService.SpecialProduct",
			wantEntitySet: "Products",
			wantEntityKey: "1",
			wantTypeCast:  "ODataService.SpecialProduct",
		},
		{
			name:          "Type cast on collection",
			path:          "/Products/ODataService.SpecialProduct",
			wantEntitySet: "Products",
			wantTypeCast:  "ODataService.SpecialProduct",
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: false,
		},
		{
			name:    "Root path",
			path:    "/",
			wantErr: false,
		},
		{
			name:    "Consecutive slashes",
			path:    "/Products//Category",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			components, err := ParseODataURLComponents(tc.path)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if components.EntitySet != tc.wantEntitySet {
				t.Errorf("EntitySet = %v, want %v", components.EntitySet, tc.wantEntitySet)
			}
			if components.EntityKey != tc.wantEntityKey {
				t.Errorf("EntityKey = %v, want %v", components.EntityKey, tc.wantEntityKey)
			}
			if components.NavigationProperty != tc.wantNavProp {
				t.Errorf("NavigationProperty = %v, want %v", components.NavigationProperty, tc.wantNavProp)
			}
			if components.IsCount != tc.wantIsCount {
				t.Errorf("IsCount = %v, want %v", components.IsCount, tc.wantIsCount)
			}
			if components.IsValue != tc.wantIsValue {
				t.Errorf("IsValue = %v, want %v", components.IsValue, tc.wantIsValue)
			}
			if components.IsRef != tc.wantIsRef {
				t.Errorf("IsRef = %v, want %v", components.IsRef, tc.wantIsRef)
			}
			if components.TypeCast != tc.wantTypeCast {
				t.Errorf("TypeCast = %v, want %v", components.TypeCast, tc.wantTypeCast)
			}
		})
	}
}

func TestParseODataURLComponents_CompositeKeys(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantEntitySet string
		wantKeyMap    map[string]string
	}{
		{
			name:          "Simple composite key",
			path:          "/Products(productID=1,languageKey='EN')",
			wantEntitySet: "Products",
			wantKeyMap:    map[string]string{"productID": "1", "languageKey": "EN"},
		},
		{
			name:          "Composite key with double quotes",
			path:          `/Products(productID=1,languageKey="EN")`,
			wantEntitySet: "Products",
			wantKeyMap:    map[string]string{"productID": "1", "languageKey": "EN"},
		},
		{
			name:          "Composite key with spaces around equals",
			path:          "/Products(productID = 1, languageKey = 'EN')",
			wantEntitySet: "Products",
			wantKeyMap:    map[string]string{"productID": "1", "languageKey": "EN"},
		},
		{
			name:          "Three-part composite key",
			path:          "/Products(a=1,b=2,c=3)",
			wantEntitySet: "Products",
			wantKeyMap:    map[string]string{"a": "1", "b": "2", "c": "3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			components, err := ParseODataURLComponents(tc.path)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if components.EntitySet != tc.wantEntitySet {
				t.Errorf("EntitySet = %v, want %v", components.EntitySet, tc.wantEntitySet)
			}

			if len(components.EntityKeyMap) != len(tc.wantKeyMap) {
				t.Errorf("EntityKeyMap length = %d, want %d", len(components.EntityKeyMap), len(tc.wantKeyMap))
			}

			for key, want := range tc.wantKeyMap {
				if got := components.EntityKeyMap[key]; got != want {
					t.Errorf("EntityKeyMap[%s] = %v, want %v", key, got, want)
				}
			}
		})
	}
}

func TestParseODataURLComponents_PropertySegments(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		wantPropertyPath string
		wantSegments     []string
	}{
		{
			name:             "Single property",
			path:             "/Products(1)/Name",
			wantPropertyPath: "Name",
			wantSegments:     []string{"Name"},
		},
		{
			name:             "Nested property",
			path:             "/Products(1)/Address/City",
			wantPropertyPath: "Address/City",
			wantSegments:     []string{"Address", "City"},
		},
		{
			name:             "Deep nested property",
			path:             "/Products(1)/Address/City/Name",
			wantPropertyPath: "Address/City/Name",
			wantSegments:     []string{"Address", "City", "Name"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			components, err := ParseODataURLComponents(tc.path)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if components.PropertyPath != tc.wantPropertyPath {
				t.Errorf("PropertyPath = %v, want %v", components.PropertyPath, tc.wantPropertyPath)
			}

			if len(components.PropertySegments) != len(tc.wantSegments) {
				t.Errorf("PropertySegments length = %d, want %d", len(components.PropertySegments), len(tc.wantSegments))
			}

			for i, want := range tc.wantSegments {
				if i < len(components.PropertySegments) {
					if got := components.PropertySegments[i]; got != want {
						t.Errorf("PropertySegments[%d] = %v, want %v", i, got, want)
					}
				}
			}
		})
	}
}

func TestParseODataURL_Simple(t *testing.T) {
	entitySet, entityKey, err := ParseODataURL("/Products(1)")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if entitySet != "Products" {
		t.Errorf("entitySet = %v, want Products", entitySet)
	}
	if entityKey != "1" {
		t.Errorf("entityKey = %v, want 1", entityKey)
	}
}

func TestIsTypeCastSegment(t *testing.T) {
	tests := []struct {
		name    string
		segment string
		want    bool
	}{
		{"Valid type cast", "ODataService.Product", true},
		{"Multi-level namespace", "Company.Api.Models.Product", true},
		{"No dot", "Product", false},
		{"Starts with $", "$metadata", false},
		{"Contains parentheses", "Function(param=1)", false},
		{"Empty string", "", false},
		{"Just dot", ".", false},
		{"Lowercase type name", "ODataService.product", false},
		{"Just namespace", "ODataService.", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isTypeCastSegment(tc.segment)
			if got != tc.want {
				t.Errorf("isTypeCastSegment(%q) = %v, want %v", tc.segment, got, tc.want)
			}
		})
	}
}

func TestSplitKeyPairs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "Single pair",
			input: "id=1",
			want:  []string{"id=1"},
		},
		{
			name:  "Two pairs",
			input: "productID=1,languageKey='EN'",
			want:  []string{"productID=1", "languageKey='EN'"},
		},
		{
			name:  "Comma in quoted value",
			input: "name='Hello, World',id=1",
			want:  []string{"name='Hello, World'", "id=1"},
		},
		{
			name:    "Unclosed quote",
			input:   "name='Hello",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := splitKeyPairs(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(got) != len(tc.want) {
				t.Errorf("splitKeyPairs(%q) returned %d pairs, want %d", tc.input, len(got), len(tc.want))
				return
			}

			for i, want := range tc.want {
				if got[i] != want {
					t.Errorf("splitKeyPairs(%q)[%d] = %q, want %q", tc.input, i, got[i], want)
				}
			}
		})
	}
}
