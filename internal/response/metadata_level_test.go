package response

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetODataMetadataLevel tests extraction of odata.metadata parameter
func TestGetODataMetadataLevel(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		formatParam  string
		expected     string
		description  string
	}{
		{
			name:        "No headers defaults to minimal",
			expected:    "minimal",
			description: "Should default to minimal when no headers provided",
		},
		{
			name:         "Accept header with minimal",
			acceptHeader: "application/json;odata.metadata=minimal",
			expected:     "minimal",
			description:  "Should extract minimal from Accept header",
		},
		{
			name:         "Accept header with full",
			acceptHeader: "application/json;odata.metadata=full",
			expected:     "full",
			description:  "Should extract full from Accept header",
		},
		{
			name:         "Accept header with none",
			acceptHeader: "application/json;odata.metadata=none",
			expected:     "none",
			description:  "Should extract none from Accept header",
		},
		{
			name:         "Accept header without odata.metadata defaults to minimal",
			acceptHeader: "application/json",
			expected:     "minimal",
			description:  "Should default to minimal when Accept has no odata.metadata",
		},
		{
			name:        "$format parameter with minimal",
			formatParam: "application/json;odata.metadata=minimal",
			expected:    "minimal",
			description: "Should extract minimal from $format parameter",
		},
		{
			name:        "$format parameter with full",
			formatParam: "application/json;odata.metadata=full",
			expected:    "full",
			description: "Should extract full from $format parameter",
		},
		{
			name:        "$format parameter with none",
			formatParam: "application/json;odata.metadata=none",
			expected:    "none",
			description: "Should extract none from $format parameter",
		},
		{
			name:        "$format parameter without odata.metadata defaults to minimal",
			formatParam: "json",
			expected:    "minimal",
			description: "Should default to minimal when $format has no odata.metadata",
		},
		{
			name:         "$format parameter overrides Accept header",
			acceptHeader: "application/json;odata.metadata=full",
			formatParam:  "application/json;odata.metadata=none",
			expected:     "none",
			description:  "$format parameter should take precedence over Accept header",
		},
		{
			name:         "Accept header with quality parameter",
			acceptHeader: "application/json;odata.metadata=full;q=0.9",
			expected:     "full",
			description:  "Should extract metadata level even with quality parameter",
		},
		{
			name:         "Multiple Accept headers picks first JSON",
			acceptHeader: "application/json;odata.metadata=full, application/xml;q=0.8",
			expected:     "full",
			description:  "Should extract from first matching JSON media type",
		},
		{
			name:         "Wildcard Accept with metadata",
			acceptHeader: "*/*;odata.metadata=none",
			expected:     "none",
			description:  "Should extract from wildcard Accept",
		},
		{
			name:         "Accept with extra spaces",
			acceptHeader: "application/json;  odata.metadata=full",
			expected:     "full",
			description:  "Should handle extra spaces in Accept header",
		},
		{
			name:        "$format with extra spaces",
			formatParam: "application/json;  odata.metadata=none",
			expected:    "none",
			description: "Should handle extra spaces in $format parameter",
		},
		{
			name:         "Invalid metadata value defaults to minimal",
			acceptHeader: "application/json;odata.metadata=invalid",
			expected:     "minimal",
			description:  "Should default to minimal for invalid metadata value",
		},
		{
			name:         "Case-sensitive metadata values",
			acceptHeader: "application/json;odata.metadata=FULL",
			expected:     "minimal",
			description:  "Should be case-sensitive (FULL != full)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.formatParam != "" {
				// Set query parameter properly
				q := req.URL.Query()
				q.Set("$format", tt.formatParam)
				req.URL.RawQuery = q.Encode()
			}
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			got := GetODataMetadataLevel(req)
			if got != tt.expected {
				t.Errorf("GetODataMetadataLevel() = %v, want %v (test: %s)",
					got, tt.expected, tt.description)
			}
		})
	}
}

// TestExtractMetadataFromFormat tests the extractMetadataFromFormat helper
func TestExtractMetadataFromFormat(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"json", "minimal"},
		{"application/json", "minimal"},
		{"application/json;odata.metadata=minimal", "minimal"},
		{"application/json;odata.metadata=full", "full"},
		{"application/json;odata.metadata=none", "none"},
		{"application/json;charset=utf-8;odata.metadata=full", "full"},
		{"", "minimal"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got := extractMetadataFromFormat(tt.format)
			if got != tt.expected {
				t.Errorf("extractMetadataFromFormat(%q) = %v, want %v",
					tt.format, got, tt.expected)
			}
		})
	}
}

// TestExtractMetadataFromAccept tests the extractMetadataFromAccept helper
func TestExtractMetadataFromAccept(t *testing.T) {
	tests := []struct {
		accept   string
		expected string
	}{
		{"application/json", "minimal"},
		{"application/json;odata.metadata=minimal", "minimal"},
		{"application/json;odata.metadata=full", "full"},
		{"application/json;odata.metadata=none", "none"},
		{"application/json;odata.metadata=full;q=0.9", "full"},
		{"application/xml;odata.metadata=full", "minimal"}, // XML is ignored
		{"*/*;odata.metadata=none", "none"},
		{"application/*;odata.metadata=full", "full"},
		{"text/html, application/json;odata.metadata=full", "full"},
		{"", "minimal"},
	}

	for _, tt := range tests {
		t.Run(tt.accept, func(t *testing.T) {
			got := extractMetadataFromAccept(tt.accept)
			if got != tt.expected {
				t.Errorf("extractMetadataFromAccept(%q) = %v, want %v",
					tt.accept, got, tt.expected)
			}
		})
	}
}

// TestIsAcceptableFormatWithMetadata ensures IsAcceptableFormat still works with odata.metadata parameter
func TestIsAcceptableFormatWithMetadata(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		formatParam  string
		expected     bool
		description  string
	}{
		{
			name:        "$format with odata.metadata parameter - JSON",
			formatParam: "application/json;odata.metadata=full",
			expected:    true,
			description: "Should accept JSON format with metadata parameter",
		},
		{
			name:        "$format with odata.metadata parameter - json",
			formatParam: "json;odata.metadata=minimal",
			expected:    true,
			description: "Should accept json format with metadata parameter",
		},
		{
			name:         "Accept with odata.metadata parameter",
			acceptHeader: "application/json;odata.metadata=none",
			expected:     true,
			description:  "Should accept JSON in Accept with metadata parameter",
		},
		{
			name:         "Accept XML with odata.metadata parameter",
			acceptHeader: "application/xml;odata.metadata=full",
			expected:     false,
			description:  "Should reject XML even with metadata parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.formatParam != "" {
				// Set query parameter properly
				q := req.URL.Query()
				q.Set("$format", tt.formatParam)
				req.URL.RawQuery = q.Encode()
			}
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			got := IsAcceptableFormat(req)
			if got != tt.expected {
				t.Errorf("IsAcceptableFormat() = %v, want %v (test: %s)",
					got, tt.expected, tt.description)
			}
		})
	}
}
