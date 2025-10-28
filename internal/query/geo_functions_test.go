package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestGeoEntity represents a test entity with geospatial properties
type TestGeoEntity struct {
	ID       int     `json:"ID" odata:"key"`
	Location string  `json:"Location"` // Would be GeographyPoint in real usage
	Route    string  `json:"Route"`    // Would be GeographyLineString
	Area     string  `json:"Area"`     // Would be GeographyPolygon
	Price    float64 `json:"Price"`
}

func getTestGeoMetadata(t *testing.T) *metadata.EntityMetadata {
	meta, err := metadata.AnalyzeEntity(TestGeoEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	return meta
}

func TestGeoDistanceFunction(t *testing.T) {
	meta := getTestGeoMetadata(t)

	tests := []struct {
		name        string
		filterStr   string
		expectError bool
	}{
		{
			name:        "geo.distance with geography literal",
			filterStr:   "geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000",
			expectError: false,
		},
		{
			name:        "geo.distance with geometry literal",
			filterStr:   "geo.distance(Location,geometry'POINT(0 0)') lt 5000",
			expectError: false,
		},
		{
			name:        "geo.distance missing second argument",
			filterStr:   "geo.distance(Location) lt 100",
			expectError: true,
		},
		{
			name:        "geo.distance with too many arguments",
			filterStr:   "geo.distance(Location,geography'POINT(0 0)',100) lt 10000",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parseFilter(tt.filterStr, meta)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if filter == nil {
					t.Errorf("Expected filter but got nil")
				}
			}
		})
	}
}

func TestGeoLengthFunction(t *testing.T) {
	meta := getTestGeoMetadata(t)

	tests := []struct {
		name        string
		filterStr   string
		expectError bool
	}{
		{
			name:        "geo.length on linestring",
			filterStr:   "geo.length(Route) gt 1000",
			expectError: false,
		},
		{
			name:        "geo.length with too many arguments",
			filterStr:   "geo.length(Route, 100) gt 1000",
			expectError: true,
		},
		{
			name:        "geo.length missing argument",
			filterStr:   "geo.length() gt 1000",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parseFilter(tt.filterStr, meta)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if filter == nil {
					t.Errorf("Expected filter but got nil")
				}
			}
		})
	}
}

func TestGeoIntersectsFunction(t *testing.T) {
	meta := getTestGeoMetadata(t)

	tests := []struct {
		name        string
		filterStr   string
		expectError bool
	}{
		{
			name:        "geo.intersects with polygon",
			filterStr:   "geo.intersects(Area,geography'SRID=4326;POLYGON((0 0,10 0,10 10,0 10,0 0))')",
			expectError: false,
		},
		{
			name:        "geo.intersects with point",
			filterStr:   "geo.intersects(Area,geometry'POINT(5 5)')",
			expectError: false,
		},
		{
			name:        "geo.intersects missing second argument",
			filterStr:   "geo.intersects(Area)",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parseFilter(tt.filterStr, meta)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if filter == nil {
					t.Errorf("Expected filter but got nil")
				}
			}
		})
	}
}

func TestGeoCombinedWithOtherFilters(t *testing.T) {
	meta := getTestGeoMetadata(t)
	filterStr := "Price gt 100 and geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000"
	
	filter, err := parseFilter(filterStr, meta)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if filter == nil {
		t.Errorf("Expected filter but got nil")
		return
	}
	
	// Verify it's a logical AND
	if filter.Logical != LogicalAnd {
		t.Errorf("Expected logical AND, got %v", filter.Logical)
	}
}

func TestGeoLiteralParsing(t *testing.T) {
	meta := getTestGeoMetadata(t)

	tests := []struct {
		name      string
		filterStr string
		wantType  string
	}{
		{
			name:      "geography literal",
			filterStr: "geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 100",
			wantType:  "geography",
		},
		{
			name:      "geometry literal",
			filterStr: "geo.distance(Location,geometry'POINT(0 0)') lt 100",
			wantType:  "geometry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parseFilter(tt.filterStr, meta)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			// The filter should have Left as a function comparison
			if filter.Left == nil {
				t.Errorf("Expected Left to be set for function comparison")
			}
		})
	}
}
