package response

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestIsAcceptableFormat tests the IsAcceptableFormat function for data endpoints
func TestIsAcceptableFormat(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		formatParam  string
		want         bool
		description  string
	}{
		{
			name:         "No Accept header",
			acceptHeader: "",
			want:         true,
			description:  "Should accept when no Accept header is provided (defaults to JSON)",
		},
		{
			name:         "Accept JSON",
			acceptHeader: "application/json",
			want:         true,
			description:  "Should accept application/json",
		},
		{
			name:         "Accept XML only",
			acceptHeader: "application/xml",
			want:         false,
			description:  "Should reject application/xml (not supported for data)",
		},
		{
			name:         "Accept text/xml only",
			acceptHeader: "text/xml",
			want:         false,
			description:  "Should reject text/xml (not supported for data)",
		},
		{
			name:         "Accept atom+xml only",
			acceptHeader: "application/atom+xml",
			want:         false,
			description:  "Should reject application/atom+xml (not supported for data)",
		},
		{
			name:         "Accept wildcard",
			acceptHeader: "*/*",
			want:         true,
			description:  "Should accept wildcard (can return JSON)",
		},
		{
			name:         "Accept application wildcard",
			acceptHeader: "application/*",
			want:         true,
			description:  "Should accept application/* (can return JSON)",
		},
		{
			name:         "JSON with higher quality than XML",
			acceptHeader: "application/xml;q=0.5, application/json;q=0.9",
			want:         true,
			description:  "Should accept when JSON has higher quality",
		},
		{
			name:         "XML with higher quality but wildcard present",
			acceptHeader: "application/xml;q=0.9, application/json;q=0.5, */*;q=0.1",
			want:         true,
			description:  "Should accept when wildcard is present (can return JSON via wildcard)",
		},
		{
			name:         "XML with higher quality and no wildcard",
			acceptHeader: "application/xml;q=0.9, application/json;q=0.5",
			want:         true,
			description:  "Should accept when JSON is explicitly listed (even with lower quality)",
		},
		{
			name:         "Browser-like Accept header",
			acceptHeader: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			want:         true,
			description:  "Should accept browser-like headers with wildcard",
		},
		{
			name:        "Format parameter json",
			formatParam: "json",
			want:        true,
			description: "Should accept $format=json",
		},
		{
			name:        "Format parameter application/json",
			formatParam: "application/json",
			want:        true,
			description: "Should accept $format=application/json",
		},
		{
			name:        "Format parameter xml",
			formatParam: "xml",
			want:        false,
			description: "Should reject $format=xml for data endpoints",
		},
		{
			name:        "Format parameter application/xml",
			formatParam: "application/xml",
			want:        false,
			description: "Should reject $format=application/xml for data endpoints",
		},
		{
			name:         "Format parameter overrides Accept",
			acceptHeader: "application/json",
			formatParam:  "xml",
			want:         false,
			description:  "Format parameter should take precedence over Accept header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.formatParam != "" {
				url += "?$format=" + tt.formatParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			got := IsAcceptableFormat(req)
			if got != tt.want {
				t.Errorf("IsAcceptableFormat() = %v, want %v\nTest: %s\nAccept: %q, Format: %q",
					got, tt.want, tt.description, tt.acceptHeader, tt.formatParam)
			}
		})
	}
}

// TestDataEndpointsRejectXML tests that data endpoints properly reject XML requests
func TestDataEndpointsRejectXML(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		expectReject bool
	}{
		{"Pure XML", "application/xml", true},
		{"Pure text/xml", "text/xml", true},
		{"Pure atom+xml", "application/atom+xml", true},
		{"XML with quality", "application/xml;q=0.9", true},
		{"JSON only", "application/json", false},
		{"Wildcard only", "*/*", false},
		{"JSON and XML with wildcard", "application/json, application/xml, */*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			acceptable := IsAcceptableFormat(req)
			rejected := !acceptable

			if rejected != tt.expectReject {
				t.Errorf("Expected rejection=%v, but got rejection=%v for Accept: %q",
					tt.expectReject, rejected, tt.acceptHeader)
			}
		})
	}
}
