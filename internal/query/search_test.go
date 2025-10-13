package query

import (
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestEntity represents a test entity with searchable properties
type SearchTestEntity struct {
	ID          int     `json:"ID" odata:"key"`
	Name        string  `json:"Name" odata:"searchable"`
	Description string  `json:"Description" odata:"searchable"`
	Category    string  `json:"Category"`
	Price       float64 `json:"Price"`
}

// TestEntityWithFuzziness represents a test entity with custom fuzziness
type SearchTestEntityWithFuzziness struct {
	ID      int    `json:"ID" odata:"key"`
	Name    string `json:"Name" odata:"searchable,fuzziness=2"`
	Email   string `json:"Email" odata:"searchable,fuzziness=3"`
	Address string `json:"Address"`
}

// TestEntityNoSearchable represents a test entity with no searchable fields
type SearchTestEntityNoSearchable struct {
	ID          int     `json:"ID" odata:"key"`
	Name        string  `json:"Name"`
	Description string  `json:"Description"`
	Category    string  `json:"Category"`
	Price       float64 `json:"Price"`
}

func TestApplySearch_BasicSearch(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	entities := []SearchTestEntity{
		{ID: 1, Name: "Laptop Pro", Description: "High-performance laptop", Category: "Electronics", Price: 1200},
		{ID: 2, Name: "Desktop Computer", Description: "Powerful desktop for gaming", Category: "Electronics", Price: 1500},
		{ID: 3, Name: "Wireless Mouse", Description: "Ergonomic wireless mouse", Category: "Accessories", Price: 25},
		{ID: 4, Name: "Keyboard", Description: "Mechanical keyboard with RGB", Category: "Accessories", Price: 75},
	}

	tests := []struct {
		name          string
		searchQuery   string
		expectedCount int
		expectedIDs   []int
	}{
		{
			name:          "Search for 'laptop'",
			searchQuery:   "laptop",
			expectedCount: 1,
			expectedIDs:   []int{1},
		},
		{
			name:          "Search for 'Laptop' (case-insensitive)",
			searchQuery:   "Laptop",
			expectedCount: 1,
			expectedIDs:   []int{1},
		},
		{
			name:          "Search for 'wireless'",
			searchQuery:   "wireless",
			expectedCount: 1,
			expectedIDs:   []int{3},
		},
		{
			name:          "Search for 'gaming'",
			searchQuery:   "gaming",
			expectedCount: 1,
			expectedIDs:   []int{2},
		},
		{
			name:          "Search for 'desktop'",
			searchQuery:   "desktop",
			expectedCount: 1,
			expectedIDs:   []int{2},
		},
		{
			name:          "Search for 'keyboard'",
			searchQuery:   "keyboard",
			expectedCount: 1,
			expectedIDs:   []int{4},
		},
		{
			name:          "Search for empty string",
			searchQuery:   "",
			expectedCount: 4,
			expectedIDs:   []int{1, 2, 3, 4},
		},
		{
			name:          "Search for non-existent term",
			searchQuery:   "smartphone",
			expectedCount: 0,
			expectedIDs:   []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplySearch(entities, tt.searchQuery, meta)

			resultSlice, ok := result.([]SearchTestEntity)
			if !ok {
				t.Fatalf("Expected []SearchTestEntity, got %T", result)
			}

			if len(resultSlice) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(resultSlice))
			}

			// Check that the correct entities are returned
			for i, expectedID := range tt.expectedIDs {
				if i >= len(resultSlice) {
					t.Errorf("Expected entity with ID %d not found", expectedID)
					continue
				}
				if resultSlice[i].ID != expectedID {
					t.Errorf("Expected entity ID %d at position %d, got %d", expectedID, i, resultSlice[i].ID)
				}
			}
		})
	}
}

func TestApplySearch_NoSearchableFields(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntityNoSearchable{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	entities := []SearchTestEntityNoSearchable{
		{ID: 1, Name: "Laptop Pro", Description: "High-performance laptop", Category: "Electronics", Price: 1200},
		{ID: 2, Name: "Desktop Computer", Description: "Powerful desktop for gaming", Category: "Electronics", Price: 1500},
		{ID: 3, Name: "Wireless Mouse", Description: "Ergonomic wireless mouse", Category: "Accessories", Price: 25},
	}

	tests := []struct {
		name          string
		searchQuery   string
		expectedCount int
		expectedIDs   []int
	}{
		{
			name:          "Search all string fields when no searchable defined",
			searchQuery:   "laptop",
			expectedCount: 1,
			expectedIDs:   []int{1},
		},
		{
			name:          "Search in Description field",
			searchQuery:   "gaming",
			expectedCount: 1,
			expectedIDs:   []int{2},
		},
		{
			name:          "Search in Category field",
			searchQuery:   "Accessories",
			expectedCount: 1,
			expectedIDs:   []int{3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplySearch(entities, tt.searchQuery, meta)

			resultSlice, ok := result.([]SearchTestEntityNoSearchable)
			if !ok {
				t.Fatalf("Expected []SearchTestEntityNoSearchable, got %T", result)
			}

			if len(resultSlice) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(resultSlice))
			}

			for i, expectedID := range tt.expectedIDs {
				if i >= len(resultSlice) {
					t.Errorf("Expected entity with ID %d not found", expectedID)
					continue
				}
				if resultSlice[i].ID != expectedID {
					t.Errorf("Expected entity ID %d at position %d, got %d", expectedID, i, resultSlice[i].ID)
				}
			}
		})
	}
}

func TestApplySearch_WithFuzziness(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntityWithFuzziness{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	entities := []SearchTestEntityWithFuzziness{
		{ID: 1, Name: "John Doe", Email: "john.doe@example.com", Address: "123 Main St"},
		{ID: 2, Name: "Jane Smith", Email: "jane.smith@example.com", Address: "456 Oak Ave"},
		{ID: 3, Name: "Bob Johnson", Email: "bob.johnson@example.com", Address: "789 Pine Rd"},
	}

	tests := []struct {
		name          string
		searchQuery   string
		expectedCount int
		description   string
	}{
		{
			name:          "Exact match",
			searchQuery:   "John",
			expectedCount: 2, // John Doe and Bob Johnson
			description:   "Should find exact substring matches",
		},
		{
			name:          "With one character difference (fuzziness=2)",
			searchQuery:   "Jahn", // One char different from "John"
			expectedCount: 3,      // Should match all with fuzziness (Jane has similar pattern)
			description:   "Should match with one character difference",
		},
		{
			name:          "Email search",
			searchQuery:   "example.com",
			expectedCount: 3, // All emails contain example.com
			description:   "Should search in email field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplySearch(entities, tt.searchQuery, meta)

			resultSlice, ok := result.([]SearchTestEntityWithFuzziness)
			if !ok {
				t.Fatalf("Expected []SearchTestEntityWithFuzziness, got %T", result)
			}

			if len(resultSlice) != tt.expectedCount {
				t.Errorf("%s: Expected %d results, got %d", tt.description, tt.expectedCount, len(resultSlice))
			}
		})
	}
}

func TestApplySearch_PartialMatch(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	entities := []SearchTestEntity{
		{ID: 1, Name: "Smartphone", Description: "Latest smartphone model", Category: "Electronics", Price: 800},
		{ID: 2, Name: "Smart TV", Description: "4K Ultra HD Smart Television", Category: "Electronics", Price: 1200},
		{ID: 3, Name: "Smartwatch", Description: "Fitness smartwatch", Category: "Wearables", Price: 300},
	}

	result := ApplySearch(entities, "smart", meta)
	resultSlice, ok := result.([]SearchTestEntity)
	if !ok {
		t.Fatalf("Expected []SearchTestEntity, got %T", result)
	}

	// All three entities should match as they all contain "smart"
	if len(resultSlice) != 3 {
		t.Errorf("Expected 3 results for partial match 'smart', got %d", len(resultSlice))
	}
}

func TestApplySearch_MultipleWords(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	entities := []SearchTestEntity{
		{ID: 1, Name: "High Performance Laptop", Description: "Gaming laptop with high specs", Category: "Electronics", Price: 1500},
		{ID: 2, Name: "Budget Laptop", Description: "Affordable laptop for students", Category: "Electronics", Price: 500},
	}

	// Search for phrase "High Performance"
	result := ApplySearch(entities, "High Performance", meta)
	resultSlice, ok := result.([]SearchTestEntity)
	if !ok {
		t.Fatalf("Expected []SearchTestEntity, got %T", result)
	}

	if len(resultSlice) != 1 {
		t.Errorf("Expected 1 result for phrase 'High Performance', got %d", len(resultSlice))
	}
	if len(resultSlice) > 0 && resultSlice[0].ID != 1 {
		t.Errorf("Expected entity with ID 1, got %d", resultSlice[0].ID)
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"hello", "hello", 0},
		{"hello", "helo", 1},
		{"hello", "hallo", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"", "hello", 5},
		{"hello", "", 5},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			result := levenshteinDistance(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, expected %d", tt.s1, tt.s2, result, tt.expected)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		pattern   string
		fuzziness int
		expected  bool
	}{
		{
			name:      "Exact match with fuzziness=1",
			text:      "hello world",
			pattern:   "world",
			fuzziness: 1,
			expected:  true,
		},
		{
			name:      "No match with fuzziness=1",
			text:      "hello world",
			pattern:   "xyz",
			fuzziness: 1,
			expected:  false,
		},
		{
			name:      "Fuzzy match with fuzziness=2",
			text:      "hello world",
			pattern:   "wrld",
			fuzziness: 2,
			expected:  true,
		},
		{
			name:      "Fuzzy match with fuzziness=3",
			text:      "hello world",
			pattern:   "warld",
			fuzziness: 3,
			expected:  true,
		},
		{
			name:      "Case sensitivity handled elsewhere",
			text:      "hello",
			pattern:   "hello",
			fuzziness: 1,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fuzzyMatch(tt.text, tt.pattern, tt.fuzziness)
			if result != tt.expected {
				t.Errorf("fuzzyMatch(%q, %q, %d) = %v, expected %v", tt.text, tt.pattern, tt.fuzziness, result, tt.expected)
			}
		})
	}
}

func TestGetSearchableProperties(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	searchable := getSearchableProperties(meta)

	// Should have 2 searchable properties: Name and Description
	if len(searchable) != 2 {
		t.Errorf("Expected 2 searchable properties, got %d", len(searchable))
	}

	// Check that the correct properties are marked as searchable
	searchableNames := make(map[string]bool)
	for _, prop := range searchable {
		searchableNames[prop.Name] = true
	}

	if !searchableNames["Name"] {
		t.Error("Expected Name to be searchable")
	}
	if !searchableNames["Description"] {
		t.Error("Expected Description to be searchable")
	}
	if searchableNames["Category"] {
		t.Error("Category should not be searchable")
	}
}

func TestGetAllStringProperties(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntityNoSearchable{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	stringProps := getAllStringProperties(meta)

	// Should have 3 string properties: Name, Description, Category
	if len(stringProps) != 3 {
		t.Errorf("Expected 3 string properties, got %d", len(stringProps))
	}

	// Check that all string properties are included
	propNames := make(map[string]bool)
	for _, prop := range stringProps {
		propNames[prop.Name] = true
		// Check that default fuzziness is set
		if prop.SearchFuzziness != 1 {
			t.Errorf("Expected default fuzziness of 1 for property %s, got %d", prop.Name, prop.SearchFuzziness)
		}
	}

	if !propNames["Name"] {
		t.Error("Expected Name to be included in string properties")
	}
	if !propNames["Description"] {
		t.Error("Expected Description to be included in string properties")
	}
	if !propNames["Category"] {
		t.Error("Expected Category to be included in string properties")
	}
}

func TestApplySearch_EmptyResults(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	entities := []SearchTestEntity{}

	result := ApplySearch(entities, "laptop", meta)
	resultSlice, ok := result.([]SearchTestEntity)
	if !ok {
		t.Fatalf("Expected []SearchTestEntity, got %T", result)
	}

	if len(resultSlice) != 0 {
		t.Errorf("Expected 0 results for empty input, got %d", len(resultSlice))
	}
}

func TestApplySearch_NonSliceInput(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	// Pass a single entity instead of a slice
	entity := SearchTestEntity{ID: 1, Name: "Laptop", Description: "Test", Category: "Electronics", Price: 1000}

	result := ApplySearch(entity, "laptop", meta)

	// Should return the input unchanged if it's not a slice
	if !reflect.DeepEqual(result, entity) {
		t.Error("Expected non-slice input to be returned unchanged")
	}
}

func TestApplySearch_PointerEntities(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(SearchTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	entities := []*SearchTestEntity{
		{ID: 1, Name: "Laptop Pro", Description: "High-performance laptop", Category: "Electronics", Price: 1200},
		{ID: 2, Name: "Desktop Computer", Description: "Powerful desktop for gaming", Category: "Electronics", Price: 1500},
	}

	result := ApplySearch(entities, "laptop", meta)
	resultSlice, ok := result.([]*SearchTestEntity)
	if !ok {
		t.Fatalf("Expected []*SearchTestEntity, got %T", result)
	}

	if len(resultSlice) != 1 {
		t.Errorf("Expected 1 result, got %d", len(resultSlice))
	}
	if len(resultSlice) > 0 && resultSlice[0].ID != 1 {
		t.Errorf("Expected entity with ID 1, got %d", resultSlice[0].ID)
	}
}
