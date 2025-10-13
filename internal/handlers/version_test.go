package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateODataVersion(t *testing.T) {
	testCases := []struct {
		name           string
		maxVersion     string
		expectedResult bool
		description    string
	}{
		{
			name:           "No OData-MaxVersion header",
			maxVersion:     "",
			expectedResult: true,
			description:    "Requests without OData-MaxVersion header should be accepted",
		},
		{
			name:           "Version 4.0",
			maxVersion:     "4.0",
			expectedResult: true,
			description:    "Version 4.0 should be accepted",
		},
		{
			name:           "Version 4.01",
			maxVersion:     "4.01",
			expectedResult: true,
			description:    "Version 4.01 should be accepted",
		},
		{
			name:           "Version 4.1",
			maxVersion:     "4.1",
			expectedResult: true,
			description:    "Version 4.1 should be accepted",
		},
		{
			name:           "Version 5.0",
			maxVersion:     "5.0",
			expectedResult: true,
			description:    "Version 5.0 should be accepted",
		},
		{
			name:           "Version 10.0",
			maxVersion:     "10.0",
			expectedResult: true,
			description:    "Version 10.0 should be accepted",
		},
		{
			name:           "Version 3.0",
			maxVersion:     "3.0",
			expectedResult: false,
			description:    "Version 3.0 should be rejected",
		},
		{
			name:           "Version 2.0",
			maxVersion:     "2.0",
			expectedResult: false,
			description:    "Version 2.0 should be rejected",
		},
		{
			name:           "Version 1.0",
			maxVersion:     "1.0",
			expectedResult: false,
			description:    "Version 1.0 should be rejected",
		},
		{
			name:           "Major version 4 only",
			maxVersion:     "4",
			expectedResult: true,
			description:    "Major version 4 only should be accepted as 4.0",
		},
		{
			name:           "Major version 3 only",
			maxVersion:     "3",
			expectedResult: false,
			description:    "Major version 3 only should be rejected",
		},
		{
			name:           "Version with spaces",
			maxVersion:     " 4.0 ",
			expectedResult: true,
			description:    "Version with spaces should be trimmed and accepted",
		},
		{
			name:           "Invalid version format",
			maxVersion:     "abc",
			expectedResult: false,
			description:    "Invalid version format should be rejected (parsed as 0.0)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.maxVersion != "" {
				req.Header.Set(HeaderODataMaxVersion, tc.maxVersion)
			}

			result := ValidateODataVersion(req)

			if result != tc.expectedResult {
				t.Errorf("%s: expected %v, got %v", tc.description, tc.expectedResult, result)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	testCases := []struct {
		name          string
		version       string
		expectedMajor int
		expectedMinor int
	}{
		{
			name:          "Standard version 4.0",
			version:       "4.0",
			expectedMajor: 4,
			expectedMinor: 0,
		},
		{
			name:          "Version 4.01",
			version:       "4.01",
			expectedMajor: 4,
			expectedMinor: 1,
		},
		{
			name:          "Version 3.0",
			version:       "3.0",
			expectedMajor: 3,
			expectedMinor: 0,
		},
		{
			name:          "Major version only",
			version:       "4",
			expectedMajor: 4,
			expectedMinor: 0,
		},
		{
			name:          "Version with spaces",
			version:       " 4.0 ",
			expectedMajor: 4,
			expectedMinor: 0,
		},
		{
			name:          "Empty string",
			version:       "",
			expectedMajor: 0,
			expectedMinor: 0,
		},
		{
			name:          "Invalid format",
			version:       "abc",
			expectedMajor: 0,
			expectedMinor: 0,
		},
		{
			name:          "Version 5.2",
			version:       "5.2",
			expectedMajor: 5,
			expectedMinor: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			major, minor := parseVersion(tc.version)

			if major != tc.expectedMajor {
				t.Errorf("Expected major version %d, got %d", tc.expectedMajor, major)
			}

			if minor != tc.expectedMinor {
				t.Errorf("Expected minor version %d, got %d", tc.expectedMinor, minor)
			}
		})
	}
}
