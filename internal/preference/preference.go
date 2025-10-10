package preference

import (
	"net/http"
	"strings"
)

// Preference represents parsed OData Prefer header preferences
type Preference struct {
	ReturnRepresentation bool
	ReturnMinimal        bool
}

// ParsePrefer parses the Prefer header from an HTTP request
// According to OData v4 spec, the Prefer header can contain:
// - return=representation: requests the service to return the created/updated entity
// - return=minimal: requests the service to return minimal or no content (default for PATCH/PUT)
func ParsePrefer(r *http.Request) *Preference {
	pref := &Preference{}

	preferHeader := r.Header.Get("Prefer")
	if preferHeader == "" {
		return pref
	}

	// Parse comma-separated preferences
	preferences := strings.Split(preferHeader, ",")
	for _, p := range preferences {
		p = strings.TrimSpace(p)
		p = strings.ToLower(p)

		switch p {
		case "return=representation":
			pref.ReturnRepresentation = true
		case "return=minimal":
			pref.ReturnMinimal = true
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
	if p.ReturnRepresentation {
		return "return=representation"
	}
	if p.ReturnMinimal {
		return "return=minimal"
	}
	return ""
}
