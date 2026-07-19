package response

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNegotiationCacheMatchesFallback verifies that the negotiation cached on the
// request context produces exactly the same results as computing it on demand,
// so injecting it at the entry point never changes observable behavior.
func TestNegotiationCacheMatchesFallback(t *testing.T) {
	cases := []struct {
		name   string
		rawURL string
		accept string
	}{
		{"plain json", "/Products?$top=100", "application/json"},
		{"metadata full via accept", "/Products", "application/json;odata.metadata=full"},
		{"metadata none via format", "/Products?$format=application/json;odata.metadata=none", ""},
		{"ieee754 via accept", "/Products", "application/json;IEEE754Compatible=true"},
		{"atom via format", "/Products?$format=atom", ""},
		{"atom via accept", "/Products", "application/atom+xml"},
		{"xml unacceptable", "/Products", "application/xml"},
		{"index present", "/Products?$index=true", "application/json"},
		{"no accept no format", "/Products", ""},
		{"invalid metadata", "/Products?$format=application/json;odata.metadata=bogus", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Fallback path: no negotiation on the context.
			plain := httptest.NewRequest(http.MethodGet, tc.rawURL, nil)
			if tc.accept != "" {
				plain.Header.Set("Accept", tc.accept)
			}

			// Cached path: negotiation injected on the context.
			cached := httptest.NewRequest(http.MethodGet, tc.rawURL, nil)
			if tc.accept != "" {
				cached.Header.Set("Accept", tc.accept)
			}
			cached = cached.WithContext(WithNegotiation(cached.Context(), cached))

			assertSame := func(label string, a, b any) {
				t.Helper()
				if a != b {
					t.Errorf("%s: cached=%v fallback=%v", label, a, b)
				}
			}

			assertSame("metadataLevel", GetODataMetadataLevel(cached), GetODataMetadataLevel(plain))
			assertSame("isAtom", IsAtomFormat(cached), IsAtomFormat(plain))
			assertSame("acceptable", IsAcceptableFormat(cached), IsAcceptableFormat(plain))
			assertSame("ieee754", GetIEEE754Compatible(cached), GetIEEE754Compatible(plain))
			assertSame("contentType", BuildJSONContentType(cached), BuildJSONContentType(plain))
			assertSame("index", shouldAddIndexAnnotations(cached), shouldAddIndexAnnotations(plain))

			cachedErr := ValidateODataMetadata(cached)
			plainErr := ValidateODataMetadata(plain)
			if (cachedErr == nil) != (plainErr == nil) {
				t.Errorf("metadataErr mismatch: cached=%v fallback=%v", cachedErr, plainErr)
			}
		})
	}
}

// TestNegotiationComputedOnce verifies that once a negotiation is cached on the
// context, repeated helper calls reuse the same instance rather than recomputing.
func TestNegotiationComputedOnce(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/Products?$top=100", nil)
	req.Header.Set("Accept", "application/json")
	req = req.WithContext(WithNegotiation(req.Context(), req))

	first := getNegotiation(req)
	second := getNegotiation(req)
	if first != second {
		t.Fatalf("expected the cached negotiation to be reused, got distinct instances")
	}
}
