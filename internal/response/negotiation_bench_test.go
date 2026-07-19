package response

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// benchNegotiationSequence mirrors the negotiation helper calls a single
// collection response performs: acceptability, atom check, metadata level,
// content type and the $index check.
func benchNegotiationSequence(r *http.Request) {
	_ = IsAcceptableFormat(r)
	_ = IsAtomFormat(r)
	_ = GetODataMetadataLevel(r)
	_ = BuildJSONContentType(r)
	_ = shouldAddIndexAnnotations(r)
}

// BenchmarkNegotiationFallback measures the cost when every helper recomputes
// the negotiation from the raw query and Accept header (pre-change behavior).
func BenchmarkNegotiationFallback(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/Products?$top=100&$filter=Price%20gt%20500", nil)
	req.Header.Set("Accept", "application/json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchNegotiationSequence(req)
	}
}

// BenchmarkNegotiationCached measures the cost when the negotiation is computed
// once at the entry point and cached on the request context (post-change).
func BenchmarkNegotiationCached(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/Products?$top=100&$filter=Price%20gt%20500", nil)
	req.Header.Set("Accept", "application/json")
	req = req.WithContext(WithNegotiation(req.Context(), req))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchNegotiationSequence(req)
	}
}

// BenchmarkNegotiationCachedWithInjection includes the one-time WithNegotiation
// cost per request, giving the true per-request comparison against the fallback.
func BenchmarkNegotiationCachedWithInjection(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/Products?$top=100&$filter=Price%20gt%20500", nil)
	req.Header.Set("Accept", "application/json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cr := req.WithContext(WithNegotiation(req.Context(), req))
		benchNegotiationSequence(cr)
	}
}
