package response

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetNegotiatedODataVersion(t *testing.T) {
	testCases := []struct {
		name            string
		maxVersion      string
		expectedVersion string
		description     string
	}{
		{
			name:            "No OData-MaxVersion header",
			maxVersion:      "",
			expectedVersion: "4.01",
			description:     "When no OData-MaxVersion is provided, return the maximum supported version",
		},
		{
			name:            "OData-MaxVersion 4.0",
			maxVersion:      "4.0",
			expectedVersion: "4.0",
			description:     "OData-MaxVersion 4.0 should return 4.0",
		},
		{
			name:            "OData-MaxVersion 4.00",
			maxVersion:      "4.00",
			expectedVersion: "4.0",
			description:     "OData-MaxVersion 4.00 should return 4.0",
		},
		{
			name:            "OData-MaxVersion 4.01",
			maxVersion:      "4.01",
			expectedVersion: "4.01",
			description:     "OData-MaxVersion 4.01 should return 4.01",
		},
		{
			name:            "OData-MaxVersion 4.1",
			maxVersion:      "4.1",
			expectedVersion: "4.01",
			description:     "OData-MaxVersion 4.1 should return 4.01 (our max)",
		},
		{
			name:            "OData-MaxVersion 4.2",
			maxVersion:      "4.2",
			expectedVersion: "4.01",
			description:     "OData-MaxVersion 4.2 should return 4.01 (our max)",
		},
		{
			name:            "OData-MaxVersion 5.0",
			maxVersion:      "5.0",
			expectedVersion: "4.01",
			description:     "OData-MaxVersion 5.0 should return 4.01 (our max)",
		},
		{
			name:            "Major version 4 only",
			maxVersion:      "4",
			expectedVersion: "4.0",
			description:     "Major version 4 only should be treated as 4.0",
		},
		{
			name:            "Version with leading/trailing spaces",
			maxVersion:      " 4.0 ",
			expectedVersion: "4.0",
			description:     "Version with spaces should be trimmed",
		},
		{
			name:            "Invalid version format returns default",
			maxVersion:      "abc",
			expectedVersion: "4.01",
			description:     "Invalid version format should return default (4.01)",
		},
		{
			name:            "Version 3.0 returns 4.0",
			maxVersion:      "3.0",
			expectedVersion: "4.0",
			description:     "Version 3.0 (below our minimum) should return 4.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.maxVersion != "" {
				req.Header.Set("OData-MaxVersion", tc.maxVersion)
			}

			version := GetNegotiatedODataVersion(req)

			if version != tc.expectedVersion {
				t.Errorf("%s: expected %q, got %q", tc.description, tc.expectedVersion, version)
			}
		})
	}
}

func TestGetNegotiatedODataVersion_NilRequest(t *testing.T) {
	version := GetNegotiatedODataVersion(nil)

	if version != "4.01" {
		t.Errorf("Expected 4.01 for nil request, got %q", version)
	}
}

func TestSetODataVersionHeaderForRequest(t *testing.T) {
	testCases := []struct {
		name            string
		maxVersion      string
		expectedVersion string
	}{
		{
			name:            "No OData-MaxVersion header",
			maxVersion:      "",
			expectedVersion: "4.01",
		},
		{
			name:            "OData-MaxVersion 4.0",
			maxVersion:      "4.0",
			expectedVersion: "4.0",
		},
		{
			name:            "OData-MaxVersion 4.01",
			maxVersion:      "4.01",
			expectedVersion: "4.01",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.maxVersion != "" {
				r.Header.Set("OData-MaxVersion", tc.maxVersion)
			}

			SetODataVersionHeaderForRequest(w, r)

			// Access the header directly using non-canonical key since OData spec requires
			// the exact capitalization "OData-Version" which is set via direct map assignment
			versionHeader := w.Header()["OData-Version"] //nolint:staticcheck
			if len(versionHeader) == 0 {
				t.Fatal("OData-Version header not set")
			}

			if versionHeader[0] != tc.expectedVersion {
				t.Errorf("Expected OData-Version %q, got %q", tc.expectedVersion, versionHeader[0])
			}
		})
	}
}
