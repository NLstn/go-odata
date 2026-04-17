package preference

import (
	"net/http"
	"strconv"
	"strings"
)

// Preference represents parsed OData Prefer header preferences
type Preference struct {
	ReturnRepresentation  bool
	ReturnMinimal         bool
	MaxPageSize           *int // odata.maxpagesize preference
	TrackChangesRequested bool
	RespondAsyncRequested bool
	AllowEntityReferences bool    // odata.allow-entityreferences preference (OData v4.0, §8.2.8.1)
	IncludeAnnotations    *string // odata.include-annotations preference value (OData v4.0, §8.2.8.4)

	trackChangesApplied       bool
	respondAsyncApplied       bool
	allowEntityRefsApplied    bool
	includeAnnotationsApplied bool
}

// ParsePrefer parses the Prefer header from an HTTP request
// According to OData v4 spec, the Prefer header can contain:
// - return=representation: requests the service to return the created/updated entity
// - return=minimal: requests the service to return minimal or no content (default for PATCH/PUT)
// - odata.maxpagesize=n: requests the service to limit page size to n items
// - odata.allow-entityreferences: allows service to return @odata.id references (OData v4.0, §8.2.8.1)
// - odata.include-annotations: controls which instance annotations are included (OData v4.0, §8.2.8.4)
func ParsePrefer(r *http.Request) *Preference {
	pref := &Preference{}

	preferHeader := r.Header.Get("Prefer")
	if preferHeader == "" {
		return pref
	}

	// Parse comma-separated preferences, respecting quoted values that may contain commas
	// (e.g. odata.include-annotations="*,-Org.OData.Core.V1.Description")
	preferences := splitPreferTokens(preferHeader)
	for _, p := range preferences {
		p = strings.TrimSpace(p)
		pLower := strings.ToLower(p)

		switch pLower {
		case "return=representation":
			pref.ReturnRepresentation = true
		case "return=minimal":
			pref.ReturnMinimal = true
		case "respond-async":
			pref.RespondAsyncRequested = true
		case "odata.allow-entityreferences":
			pref.AllowEntityReferences = true
		default:
			// Check for odata.maxpagesize preference
			if strings.HasPrefix(pLower, "odata.maxpagesize=") {
				maxPageSizeStr := strings.TrimPrefix(p, "odata.maxpagesize=")
				maxPageSizeStr = strings.TrimPrefix(maxPageSizeStr, "odata.maxPageSize=")
				maxPageSizeStr = strings.TrimPrefix(maxPageSizeStr, "odata.MaxPageSize=")
				if maxPageSize, err := strconv.Atoi(maxPageSizeStr); err == nil && maxPageSize > 0 {
					pref.MaxPageSize = &maxPageSize
				}
				continue
			}

			if strings.HasPrefix(pLower, "odata.track-changes") {
				pref.TrackChangesRequested = true
				continue
			}

			// Check for odata.include-annotations preference
			// Value can be quoted or unquoted, e.g.:
			//   odata.include-annotations="*"
			//   odata.include-annotations=*
			//   odata.include-annotations="*,-Org.OData.Core.V1.Description"
			if strings.HasPrefix(pLower, "odata.include-annotations=") {
				val := p[len("odata.include-annotations="):]
				// Strip surrounding quotes if present
				if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
					val = val[1 : len(val)-1]
				}
				pref.IncludeAnnotations = &val
				continue
			}
		}
	}

	return pref
}

// ShouldReturnContent determines if content should be returned based on preferences and operation
// For POST: default is to return representation, unless return=minimal is specified
// For PATCH/PUT: default is to return minimal, unless return=representation is specified
func (p *Preference) ShouldReturnContent(isPostOperation bool) bool {
	if isPostOperation {
		// POST defaults to returning representation
		return !p.ReturnMinimal
	}

	// PATCH/PUT default to returning minimal (no content)
	return p.ReturnRepresentation
}

// GetPreferenceApplied returns the Preference-Applied header value
// Returns empty string if no preference was applied
func (p *Preference) GetPreferenceApplied() string {
	var applied []string
	if p.ReturnRepresentation {
		applied = append(applied, "return=representation")
	}
	if p.ReturnMinimal {
		applied = append(applied, "return=minimal")
	}
	if p.MaxPageSize != nil {
		applied = append(applied, "odata.maxpagesize="+strconv.Itoa(*p.MaxPageSize))
	}
	if p.trackChangesApplied {
		applied = append(applied, "odata.track-changes")
	}
	if p.respondAsyncApplied {
		applied = append(applied, "respond-async")
	}
	if p.allowEntityRefsApplied {
		applied = append(applied, "odata.allow-entityreferences")
	}
	if p.includeAnnotationsApplied && p.IncludeAnnotations != nil {
		applied = append(applied, `odata.include-annotations="`+*p.IncludeAnnotations+`"`)
	}
	return strings.Join(applied, ", ")
}

// ApplyTrackChanges marks the odata.track-changes preference as applied if it was requested.
func (p *Preference) ApplyTrackChanges() {
	if p.TrackChangesRequested {
		p.trackChangesApplied = true
	}
}

// ApplyRespondAsync marks the respond-async preference as applied if it was requested.
func (p *Preference) ApplyRespondAsync() {
	if p.RespondAsyncRequested {
		p.respondAsyncApplied = true
	}
}

// RespondAsyncApplied returns true if the respond-async preference was applied.
func (p *Preference) RespondAsyncApplied() bool {
	return p.respondAsyncApplied
}

// ApplyAllowEntityReferences marks the odata.allow-entityreferences preference as applied.
func (p *Preference) ApplyAllowEntityReferences() {
	if p.AllowEntityReferences {
		p.allowEntityRefsApplied = true
	}
}

// ApplyIncludeAnnotations marks the odata.include-annotations preference as applied.
func (p *Preference) ApplyIncludeAnnotations() {
	if p.IncludeAnnotations != nil {
		p.includeAnnotationsApplied = true
	}
}

// MatchesAnnotationFilter reports whether the given qualified annotation term name
// should be included according to the odata.include-annotations filter value.
//
// The filter is a comma-separated list of rules processed in order; later rules
// take precedence over earlier ones.  Each rule is either:
//   - "*"                   – include all annotations
//   - "-*"                  – exclude all annotations
//   - "Namespace.TermName"  – include the specific annotation
//   - "-Namespace.TermName" – exclude the specific annotation
//   - "Namespace.*"         – include all annotations in a namespace
//   - "-Namespace.*"        – exclude all annotations in a namespace
//
// If the filter is empty, false is returned (no annotations by default).
func MatchesAnnotationFilter(qualifiedTermName string, filter string) bool {
	if filter == "" {
		return false
	}

	rules := strings.Split(filter, ",")
	include := false // conservative default

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		exclude := strings.HasPrefix(rule, "-")
		if exclude {
			rule = rule[1:]
		}

		var matches bool
		switch {
		case rule == "*":
			matches = true
		case strings.HasSuffix(rule, ".*"):
			ns := strings.TrimSuffix(rule, ".*")
			matches = strings.HasPrefix(qualifiedTermName, ns+".")
		default:
			matches = strings.EqualFold(qualifiedTermName, rule)
		}

		if matches {
			include = !exclude
		}
	}

	return include
}

// SanitizeForAsyncDispatch rebuilds a Prefer header without the respond-async token.
func SanitizeForAsyncDispatch(preferHeader string) string {
	if preferHeader == "" {
		return ""
	}

	tokens := strings.Split(preferHeader, ",")
	sanitized := make([]string, 0, len(tokens))
	for _, token := range tokens {
		trimmed := strings.TrimSpace(token)
		if trimmed == "" {
			continue
		}
		if strings.EqualFold(trimmed, "respond-async") {
			continue
		}
		sanitized = append(sanitized, trimmed)
	}

	return strings.Join(sanitized, ", ")
}

// splitPreferTokens splits a Prefer header value into individual tokens, respecting
// quoted values that may contain commas (e.g. odata.include-annotations="*,-Term").
func splitPreferTokens(header string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(header); i++ {
		ch := header[i]
		switch {
		case ch == '"':
			inQuote = !inQuote
			current.WriteByte(ch)
		case ch == ',' && !inQuote:
			tokens = append(tokens, current.String())
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}
