package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/response"
)

func TestSetODataHeader(t *testing.T) {
	w := httptest.NewRecorder()

	SetODataHeader(w, "OData-Version", "4.01")

	// Verify header was set - use direct map access since OData headers have non-canonical keys
	// OData spec requires exact "OData-Version" capitalization which is non-canonical in Go
	header := w.Header()
	values := header["OData-Version"] //nolint:staticcheck // OData headers require non-canonical keys
	if len(values) == 0 || values[0] != "4.01" {
		t.Errorf("Expected OData-Version header to be 4.01, got %v", values)
	}
}

func TestSetODataVersionHeader(t *testing.T) {
	w := httptest.NewRecorder()

	SetODataVersionHeader(w)

	// Verify header was set with correct value - use direct map access
	// OData spec requires exact "OData-Version" capitalization which is non-canonical in Go
	header := w.Header()
	values := header["OData-Version"] //nolint:staticcheck // OData headers require non-canonical keys
	if len(values) == 0 {
		t.Error("Expected OData-Version header to be set")
	}
}

func TestSetODataVersionHeaderForRequest(t *testing.T) {
	tests := []struct {
		name            string
		maxVersion      string
		expectedVersion string
	}{
		{
			name:            "No MaxVersion header",
			maxVersion:      "",
			expectedVersion: "4.01",
		},
		{
			name:            "MaxVersion 4.0",
			maxVersion:      "4.0",
			expectedVersion: "4.0",
		},
		{
			name:            "MaxVersion 4.01",
			maxVersion:      "4.01",
			expectedVersion: "4.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.maxVersion != "" {
				req.Header.Set("OData-MaxVersion", tt.maxVersion)
			}

			SetODataVersionHeaderForRequest(w, req)

			// OData spec requires exact "OData-Version" capitalization which is non-canonical in Go
			values := w.Header()["OData-Version"] //nolint:staticcheck // OData headers require non-canonical keys
			if len(values) == 0 {
				t.Error("Expected OData-Version header to be set")
				return
			}
			if values[0] != tt.expectedVersion {
				t.Errorf("Expected OData-Version %s, got %s", tt.expectedVersion, values[0])
			}
		})
	}
}

func TestGetNegotiatedODataVersion(t *testing.T) {
	tests := []struct {
		name            string
		maxVersion      string
		expectedVersion string
	}{
		{
			name:            "No MaxVersion header",
			maxVersion:      "",
			expectedVersion: "4.01",
		},
		{
			name:            "MaxVersion 4.0",
			maxVersion:      "4.0",
			expectedVersion: "4.0",
		},
		{
			name:            "MaxVersion 4.01",
			maxVersion:      "4.01",
			expectedVersion: "4.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.maxVersion != "" {
				req.Header.Set("OData-MaxVersion", tt.maxVersion)
			}

			version := GetNegotiatedODataVersion(req)
			if version != tt.expectedVersion {
				t.Errorf("GetNegotiatedODataVersion() = %s, want %s", version, tt.expectedVersion)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ProductID", "product_id"},
		{"productId", "product_id"},
		{"XMLParser", "xml_parser"},
		{"SimpleTest", "simple_test"},
		{"ID", "id"},
		{"Test", "test"},
		{"test", "test"},
		{"TestXMLParser", "test_xml_parser"},
		{"HTTPResponse", "http_response"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseCompositeKey(t *testing.T) {
	tests := []struct {
		name       string
		keyPart    string
		wantKeyMap map[string]string
		wantErr    bool
	}{
		{
			name:       "Simple key",
			keyPart:    "123",
			wantKeyMap: nil,
			wantErr:    true, // Not a composite key
		},
		{
			name:    "Single composite key",
			keyPart: "ID=123",
			wantKeyMap: map[string]string{
				"ID": "123",
			},
			wantErr: false,
		},
		{
			name:    "Multiple composite keys",
			keyPart: "ID=123,Name=test",
			wantKeyMap: map[string]string{
				"ID":   "123",
				"Name": "test",
			},
			wantErr: false,
		},
		{
			name:    "Quoted string value",
			keyPart: "ID=123,Name='test value'",
			wantKeyMap: map[string]string{
				"ID":   "123",
				"Name": "test value",
			},
			wantErr: false,
		},
		{
			name:    "Double quoted value",
			keyPart: "ID=123,Name=\"test value\"",
			wantKeyMap: map[string]string{
				"ID":   "123",
				"Name": "test value",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := &response.ODataURLComponents{
				EntityKeyMap: make(map[string]string),
			}

			err := parseCompositeKey(tt.keyPart, components)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCompositeKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantKeyMap != nil {
				for k, v := range tt.wantKeyMap {
					if components.EntityKeyMap[k] != v {
						t.Errorf("parseCompositeKey() key %q = %q, want %q", k, components.EntityKeyMap[k], v)
					}
				}
			}
		})
	}
}
