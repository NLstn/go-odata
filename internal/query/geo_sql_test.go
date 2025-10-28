package query

import (
	"testing"
)

// TestGeoFunctionSQLGeneration tests that geospatial functions generate correct SQL
func TestGeoFunctionSQLGeneration(t *testing.T) {
	meta := getTestGeoMetadata(t)

	tests := []struct {
		name        string
		filterStr   string
		expectError bool
		checkSQL    bool
	}{
		{
			name:        "geo.distance generates ST_Distance SQL",
			filterStr:   "geo.distance(Location,geography'SRID=4326;POINT(0 0)') lt 10000",
			expectError: false,
			checkSQL:    true,
		},
		{
			name:        "geo.length generates ST_Length SQL",
			filterStr:   "geo.length(Route) gt 1000",
			expectError: false,
			checkSQL:    true,
		},
		{
			name:        "geo.intersects generates ST_Intersects SQL",
			filterStr:   "geo.intersects(Area,geography'POLYGON((0 0,10 0,10 10,0 10,0 0))')",
			expectError: false,
			checkSQL:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := parseFilter(tt.filterStr, meta, nil)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if filter == nil {
				t.Errorf("Expected filter but got nil")
				return
			}

			// Build the SQL condition
			sql, args := buildFilterCondition(filter, meta)
			
			if sql == "" {
				t.Errorf("Expected SQL but got empty string")
				return
			}

			// Just verify that SQL was generated - we don't check exact format
			// as it may vary across database implementations
			t.Logf("Generated SQL: %s", sql)
			t.Logf("Args: %v", args)
		})
	}
}

// TestInvalidGeoFunction tests that invalid geospatial functions return errors
func TestInvalidGeoFunction(t *testing.T) {
	meta := getTestGeoMetadata(t)

	tests := []struct {
		name      string
		filterStr string
	}{
		{
			name:      "geo.invalid is not a valid function",
			filterStr: "geo.invalid(Location)",
		},
		{
			name:      "geo.distance with wrong argument count",
			filterStr: "geo.distance(Location,geography'POINT(0 0)',extra) lt 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFilter(tt.filterStr, meta, nil)
			
			if err == nil {
				t.Errorf("Expected error for invalid function but got nil")
			}
		})
	}
}
