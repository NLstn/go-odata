package query

import (
	"net/url"
	"testing"
)

func TestParseIndexOption(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name        string
		queryString string
		expectError bool
		expectIndex bool
	}{
		{
			name:        "Valid $index parameter",
			queryString: "$index",
			expectError: false,
			expectIndex: true,
		},
		{
			name:        "$index with other query options",
			queryString: "$index&$top=10&$orderby=Name",
			expectError: false,
			expectIndex: true,
		},
		{
			name:        "No $index parameter",
			queryString: "$top=10",
			expectError: false,
			expectIndex: false,
		},
		{
			name:        "$index with value should fail",
			queryString: "$index=true",
			expectError: true,
			expectIndex: false,
		},
		{
			name:        "$index with filter",
			queryString: "$index&$filter=Price gt 100",
			expectError: false,
			expectIndex: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams, err := url.ParseQuery(tt.queryString)
			if err != nil {
				t.Fatalf("Failed to parse query string: %v", err)
			}

			options, err := ParseQueryOptions(queryParams, meta)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && options != nil && options.Index != tt.expectIndex {
				t.Errorf("Expected Index=%v, got %v", tt.expectIndex, options.Index)
			}
		})
	}
}

func TestParseSchemaVersionOption(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name                  string
		queryString           string
		expectError           bool
		expectedSchemaVersion *string
	}{
		{
			name:                  "Valid $schemaversion parameter",
			queryString:           "$schemaversion=1.0",
			expectError:           false,
			expectedSchemaVersion: stringPtr("1.0"),
		},
		{
			name:                  "$schemaversion with semantic version",
			queryString:           "$schemaversion=2.1.3",
			expectError:           false,
			expectedSchemaVersion: stringPtr("2.1.3"),
		},
		{
			name:                  "No $schemaversion parameter",
			queryString:           "$top=10",
			expectError:           false,
			expectedSchemaVersion: nil,
		},
		{
			name:                  "Empty $schemaversion value should fail",
			queryString:           "$schemaversion=",
			expectError:           true,
			expectedSchemaVersion: nil,
		},
		{
			name:                  "$schemaversion with whitespace only should fail",
			queryString:           "$schemaversion=%20%20%20",
			expectError:           true,
			expectedSchemaVersion: nil,
		},
		{
			name:                  "$schemaversion with other query options",
			queryString:           "$schemaversion=3.0&$top=5",
			expectError:           false,
			expectedSchemaVersion: stringPtr("3.0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams, err := url.ParseQuery(tt.queryString)
			if err != nil {
				t.Fatalf("Failed to parse query string: %v", err)
			}

			options, err := ParseQueryOptions(queryParams, meta)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && options != nil {
				if tt.expectedSchemaVersion == nil && options.SchemaVersion != nil {
					t.Errorf("Expected SchemaVersion=nil, got %v", *options.SchemaVersion)
				}
				if tt.expectedSchemaVersion != nil && options.SchemaVersion == nil {
					t.Errorf("Expected SchemaVersion=%v, got nil", *tt.expectedSchemaVersion)
				}
				if tt.expectedSchemaVersion != nil && options.SchemaVersion != nil && *options.SchemaVersion != *tt.expectedSchemaVersion {
					t.Errorf("Expected SchemaVersion=%v, got %v", *tt.expectedSchemaVersion, *options.SchemaVersion)
				}
			}
		})
	}
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}
