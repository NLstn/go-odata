package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestMetadataAcceptHeader tests that the metadata endpoint respects Accept header for XML/JSON
func TestMetadataAcceptHeader(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	entityMeta, _ := metadata.AnalyzeEntity(TestEntity{})
	entities["TestEntities"] = entityMeta

	handler := NewMetadataHandler(entities)

	tests := []struct {
		name           string
		acceptHeader   string
		formatParam    string
		expectedFormat string // "xml" or "json"
		description    string
	}{
		{
			name:           "Explicit XML via Accept header",
			acceptHeader:   "application/xml",
			expectedFormat: "xml",
			description:    "Should return XML when Accept: application/xml",
		},
		{
			name:           "Explicit JSON via Accept header",
			acceptHeader:   "application/json",
			expectedFormat: "json",
			description:    "Should return JSON when Accept: application/json",
		},
		{
			name:           "No Accept header defaults to XML",
			acceptHeader:   "",
			expectedFormat: "xml",
			description:    "Should default to XML when no Accept header provided",
		},
		{
			name:           "Wildcard Accept header defaults to XML",
			acceptHeader:   "*/*",
			expectedFormat: "xml",
			description:    "Should default to XML for wildcard Accept",
		},
		{
			name:           "XML preferred over JSON via quality",
			acceptHeader:   "application/json;q=0.8, application/xml;q=0.9",
			expectedFormat: "xml",
			description:    "Should return XML when it has higher quality value",
		},
		{
			name:           "JSON preferred over XML via quality",
			acceptHeader:   "application/xml;q=0.8, application/json;q=0.9",
			expectedFormat: "json",
			description:    "Should return JSON when it has higher quality value",
		},
		{
			name:           "JSON with equal quality prefers JSON",
			acceptHeader:   "application/xml;q=0.9, application/json;q=0.9",
			expectedFormat: "json",
			description:    "Should prefer JSON when qualities are equal (based on order or tie-breaking)",
		},
		{
			name:           "Complex Accept with multiple types",
			acceptHeader:   "text/html, application/xml;q=0.9, */*;q=0.8",
			expectedFormat: "xml",
			description:    "Should pick XML from complex Accept header",
		},
		{
			name:           "Format parameter overrides Accept header - JSON",
			acceptHeader:   "application/xml",
			formatParam:    "json",
			expectedFormat: "json",
			description:    "$format parameter should take precedence over Accept header",
		},
		{
			name:           "Format parameter overrides Accept header - XML",
			acceptHeader:   "application/json",
			formatParam:    "xml",
			expectedFormat: "xml",
			description:    "$format parameter should take precedence over Accept header",
		},
		{
			name:           "text/xml is treated as XML",
			acceptHeader:   "text/xml",
			expectedFormat: "xml",
			description:    "Should recognize text/xml as XML format",
		},
		{
			name:           "Browser-like Accept header",
			acceptHeader:   "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			expectedFormat: "xml",
			description:    "Should handle browser-style Accept headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/$metadata"
			if tt.formatParam != "" {
				url += "?$format=" + tt.formatParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			handler.HandleMetadata(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			contentType := w.Header().Get("Content-Type")
			body := w.Body.String()

			if tt.expectedFormat == "json" {
				if contentType != "application/json" {
					t.Errorf("Content-Type = %v, want application/json (test: %s)", contentType, tt.description)
				}
				// Verify it's actually JSON
				if !strings.Contains(body, "{") || !strings.Contains(body, "$Version") {
					t.Errorf("Response doesn't look like JSON (test: %s)", tt.description)
				}
			} else { // xml
				if contentType != "application/xml" {
					t.Errorf("Content-Type = %v, want application/xml (test: %s)", contentType, tt.description)
				}
				// Verify it's actually XML
				if !strings.Contains(body, "<?xml") && !strings.Contains(body, "<edmx:") {
					t.Errorf("Response doesn't look like XML (test: %s)", tt.description)
				}
			}
		})
	}
}

// TestServiceDocumentAcceptHeader tests that the service document respects Accept header
func TestServiceDocumentAcceptHeader(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	entityMeta, _ := metadata.AnalyzeEntity(TestEntity{})
	entities["TestEntities"] = entityMeta

	handler := NewServiceDocumentHandler(entities)

	tests := []struct {
		name         string
		acceptHeader string
		description  string
	}{
		{
			name:         "JSON via Accept header",
			acceptHeader: "application/json",
			description:  "Service document should return JSON",
		},
		{
			name:         "No Accept header",
			acceptHeader: "",
			description:  "Service document should default to JSON",
		},
		{
			name:         "Wildcard Accept header",
			acceptHeader: "*/*",
			description:  "Service document should return JSON for wildcard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			handler.HandleServiceDocument(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Content-Type = %v, want application/json (test: %s)", contentType, tt.description)
			}
		})
	}
}

// TestShouldReturnJSON tests the shouldReturnJSON function directly
func TestShouldReturnJSON(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		formatParam  string
		want         bool
	}{
		{"No headers", "", "", false},
		{"Accept JSON", "application/json", "", true},
		{"Accept XML", "application/xml", "", false},
		{"Accept text/xml", "text/xml", "", false},
		{"Accept wildcard", "*/*", "", false},
		{"Format param json", "", "json", true},
		{"Format param xml", "", "xml", false},
		{"Format param overrides Accept", "application/xml", "json", true},
		{"JSON quality higher", "application/json;q=0.9, application/xml;q=0.8", "", true},
		{"XML quality higher", "application/json;q=0.8, application/xml;q=0.9", "", false},
		{"Complex browser Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/$metadata"
			if tt.formatParam != "" {
				url += "?$format=" + tt.formatParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			got := shouldReturnJSON(req)
			if got != tt.want {
				t.Errorf("shouldReturnJSON() = %v, want %v (Accept: %q, format: %q)",
					got, tt.want, tt.acceptHeader, tt.formatParam)
			}
		})
	}
}
