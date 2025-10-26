package odata_test

import (
	"net/http/httptest"
	"testing"
)

// TestODataVersionHeader verifies that the OData-Version header is set on all responses
// This is a centralized test to ensure the header is present, removing the need to check
// this in every individual unit test since the header is set at the ServeHTTP level.
func TestODataVersionHeader(t *testing.T) {
	service, _ := setupTestService(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"Service document", "GET", "/"},
		{"Metadata document", "GET", "/$metadata"},
		{"Collection", "GET", "/Products"},
		{"Single entity", "GET", "/Products(1)"},
		{"Count", "GET", "/Products/$count"},
		{"Not found", "GET", "/NonExistent"},
		{"Batch", "POST", "/$batch"},
		{"Error response", "GET", "/Products(99999)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			// Verify OData-Version header is present and set to 4.01
			// Access the header directly with exact casing (OData-Version with capital 'D')
			// as we intentionally use non-canonical casing per OData v4 spec
			//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
			odataVersionValues := w.Header()["OData-Version"]
			if len(odataVersionValues) == 0 || odataVersionValues[0] != "4.01" {
				t.Errorf("OData-Version = %v, want [4.01] (status: %d)", odataVersionValues, w.Code)
			}
		})
	}
}
