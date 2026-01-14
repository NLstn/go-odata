package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

func TestContainsGeospatialOperations(t *testing.T) {
	// Test entity for geospatial operations
	type TestEntity struct {
		ID       int    `json:"ID" odata:"key"`
		Location string `json:"Location"`
		Name     string `json:"Name"`
	}

	meta, err := metadata.AnalyzeEntity(&TestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	tests := []struct {
		name     string
		filter   string
		expected bool
	}{
		{
			name:     "geo.distance with comparison",
			filter:   "geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000",
			expected: true,
		},
		{
			name:     "geo.length with comparison",
			filter:   "geo.length(Location) gt 100",
			expected: true,
		},
		{
			name:     "geo.intersects",
			filter:   "geo.intersects(Location,geography'POINT(0 0)')",
			expected: true,
		},
		{
			name:     "non-geospatial filter",
			filter:   "Name eq 'test'",
			expected: false,
		},
		{
			name:     "geo.distance with other conditions",
			filter:   "Name eq 'test' and geo.distance(Location,geography'POINT(0 0)') lt 100",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parseFilter(tt.filter, meta, nil, 0)
			if err != nil {
				t.Fatalf("Failed to parse filter: %v", err)
			}

			result := ContainsGeospatialOperations(filter)
			if result != tt.expected {
				t.Errorf("ContainsGeospatialOperations() = %v, expected %v", result, tt.expected)
				// Debug output
				t.Logf("Filter: %+v", filter)
			}
		})
	}
}
