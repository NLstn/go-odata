package query

import (
	"testing"
)

// TestDateFunctions_ComputeParsing tests parsing of date functions in compute expressions
func TestDateFunctions_ComputeParsing(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		compute   string
		expectErr bool
	}{
		{
			name:      "year in compute",
			compute:   "year(CreatedAt) as Year",
			expectErr: false,
		},
		{
			name:      "month in compute",
			compute:   "month(CreatedAt) as Month",
			expectErr: false,
		},
		{
			name:      "day in compute",
			compute:   "day(CreatedAt) as Day",
			expectErr: false,
		},
		{
			name:      "hour in compute",
			compute:   "hour(CreatedAt) as Hour",
			expectErr: false,
		},
		{
			name:      "minute in compute",
			compute:   "minute(CreatedAt) as Minute",
			expectErr: false,
		},
		{
			name:      "second in compute",
			compute:   "second(CreatedAt) as Second",
			expectErr: false,
		},
		{
			name:      "date in compute",
			compute:   "date(CreatedAt) as DateOnly",
			expectErr: false,
		},
		{
			name:      "time in compute",
			compute:   "time(CreatedAt) as TimeOnly",
			expectErr: false,
		},
		{
			name:      "multiple date functions",
			compute:   "year(CreatedAt) as Year,month(CreatedAt) as Month",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the compute transformation
			result, err := parseCompute("compute("+tt.compute+")", meta, 0)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil || result.Compute == nil {
				t.Error("Expected non-nil compute transformation")
				return
			}

			if len(result.Compute.Expressions) == 0 {
				t.Error("Expected at least one compute expression")
			}
		})
	}
}

// TestDateFunctions_ComputeSQL tests SQL generation for date functions in compute
func TestDateFunctions_ComputeSQL(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name           string
		compute        string
		expectErr      bool
		expectedFields int // Expected number of computed fields
	}{
		{
			name:           "year extraction",
			compute:        "year(CreatedAt) as Year",
			expectErr:      false,
			expectedFields: 1,
		},
		{
			name:           "month extraction",
			compute:        "month(CreatedAt) as Month",
			expectErr:      false,
			expectedFields: 1,
		},
		{
			name:           "multiple extractions",
			compute:        "year(CreatedAt) as Year,month(CreatedAt) as Month,day(CreatedAt) as Day",
			expectErr:      false,
			expectedFields: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCompute("compute("+tt.compute+")", meta, 0)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil || result.Compute == nil {
				t.Error("Expected non-nil compute transformation")
				return
			}

			if len(result.Compute.Expressions) != tt.expectedFields {
				t.Errorf("Expected %d expressions, got %d",
					tt.expectedFields, len(result.Compute.Expressions))
			}
		})
	}
}
