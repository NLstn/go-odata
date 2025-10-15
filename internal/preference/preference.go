package preference

import (
	"net/http"
	"strconv"
	"strings"
)

// Preference represents parsed OData Prefer header preferences
type Preference struct {
	ReturnRepresentation bool
	ReturnMinimal        bool
	MaxPageSize          *int // odata.maxpagesize preference
}

// ParsePrefer parses the Prefer header from an HTTP request
// According to OData v4 spec, the Prefer header can contain:
// - return=representation: requests the service to return the created/updated entity
// - return=minimal: requests the service to return minimal or no content (default for PATCH/PUT)
// - odata.maxpagesize=n: requests the service to limit page size to n items
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
		pLower := strings.ToLower(p)

		switch pLower {
		case "return=representation":
			pref.ReturnRepresentation = true
		case "return=minimal":
			pref.ReturnMinimal = true
		default:
			// Check for odata.maxpagesize preference
			if strings.HasPrefix(pLower, "odata.maxpagesize=") {
				maxPageSizeStr := strings.TrimPrefix(p, "odata.maxpagesize=")
				maxPageSizeStr = strings.TrimPrefix(maxPageSizeStr, "odata.maxPageSize=")
				maxPageSizeStr = strings.TrimPrefix(maxPageSizeStr, "odata.MaxPageSize=")
				if maxPageSize, err := strconv.Atoi(maxPageSizeStr); err == nil && maxPageSize > 0 {
					pref.MaxPageSize = &maxPageSize
				}
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
	return strings.Join(applied, ", ")
}
