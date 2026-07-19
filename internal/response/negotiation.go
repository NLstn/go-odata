package response

import (
	"context"
	"net/http"

	oquery "github.com/nlstn/go-odata/internal/query"
)

// negotiationContextKey is the context key under which a *negotiation computed
// once at the start of a request is stored.
const negotiationContextKey ContextKey = "odata.negotiation"

// negotiation holds the result of content negotiation for a single request.
//
// Metadata level, format acceptability, the Atom/JSON decision, IEEE754
// compatibility and the presence of $index all derive solely from the request's
// $format query parameter and Accept header, both of which are immutable for the
// lifetime of a request. Computing them once and reusing the result avoids
// re-parsing the raw query string (two url.Values allocations plus percent
// decoding) on each of the four to eight negotiation helper calls per request.
type negotiation struct {
	metadataLevel string
	isAtom        bool
	acceptable    bool
	ieee754       bool
	hasIndex      bool
	metadataErr   error
}

// computeNegotiation performs all format/Accept parsing for a request exactly
// once. The raw query string is parsed a single time to derive both the $format
// value and the presence of $index, and that result is threaded through the
// per-field helpers instead of each of them re-parsing the query.
func computeNegotiation(r *http.Request) *negotiation {
	format, hasIndex := parseQueryNegotiation(r.URL.RawQuery)
	accept := r.Header.Get("Accept")

	n := &negotiation{
		metadataLevel: metadataLevelFrom(format, accept),
		isAtom:        isAtomFrom(format, accept),
		acceptable:    isAcceptableFrom(format, accept),
		ieee754:       ieee754From(format, accept),
		hasIndex:      hasIndex,
		metadataErr:   validateMetadataFrom(format, accept),
	}
	return n
}

// parseQueryNegotiation parses the raw query string once and reports the $format
// value (empty when absent) and whether the $index system query option is
// present in either its prefixed or normalized form.
func parseQueryNegotiation(rawQuery string) (format string, hasIndex bool) {
	raw := oquery.ParseRawQuery(rawQuery)
	if _, ok := raw["$index"]; ok {
		hasIndex = true
	}

	normalized := oquery.NormalizeQueryParams(raw)
	if !hasIndex {
		_, hasIndex = normalized["$index"]
	}

	return normalized.Get("$format"), hasIndex
}

// WithNegotiation computes the request's content negotiation once and returns a
// context carrying the result. Call it at the start of request handling so that
// the response helpers reuse the cached decision instead of re-parsing.
func WithNegotiation(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, negotiationContextKey, computeNegotiation(r))
}

// getNegotiation returns the negotiation cached on the request context, computing
// it on demand (without caching) when absent. The fallback keeps direct callers
// of the exported helpers — tests and alternate entry points that never went
// through WithNegotiation — correct.
func getNegotiation(r *http.Request) *negotiation {
	if n, ok := r.Context().Value(negotiationContextKey).(*negotiation); ok {
		return n
	}
	return computeNegotiation(r)
}
