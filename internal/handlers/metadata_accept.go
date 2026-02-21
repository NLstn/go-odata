package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/query"
)

// shouldReturnJSON determines if JSON format should be returned based on request
func shouldReturnJSON(r *http.Request) bool {
	format := query.GetOrParseParsedQuery(r.Context(), r.URL.RawQuery).Get("$format")
	if format == "json" || format == "application/json" {
		return true
	}
	if format == "xml" || format == "application/xml" {
		return false
	}

	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}

	type mediaType struct {
		mimeType string
		quality  float64
	}

	parts := strings.Split(accept, ",")
	mediaTypes := make([]mediaType, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		subparts := strings.Split(part, ";")
		mimeType := strings.TrimSpace(subparts[0])
		quality := 1.0

		for _, param := range subparts[1:] {
			param = strings.TrimSpace(param)
			if strings.HasPrefix(param, "q=") {
				if q, err := parseQuality(param[2:]); err == nil {
					quality = q
				}
			}
		}

		mediaTypes = append(mediaTypes, mediaType{mimeType: mimeType, quality: quality})
	}

	var bestJSON, bestXML, bestWildcard float64
	for _, mt := range mediaTypes {
		switch mt.mimeType {
		case "application/json":
			if mt.quality > bestJSON {
				bestJSON = mt.quality
			}
		case "application/xml", "text/xml":
			if mt.quality > bestXML {
				bestXML = mt.quality
			}
		case "*/*", "application/*":
			if mt.quality > bestWildcard {
				bestWildcard = mt.quality
			}
		}
	}

	if bestJSON > 0 && bestXML > 0 {
		return bestJSON >= bestXML
	}

	if bestJSON > 0 {
		return true
	}

	if bestXML > 0 {
		return false
	}

	if bestWildcard > 0 {
		return false
	}

	return false
}

// parseQuality parses a quality value from Accept header
func parseQuality(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 1.0, nil
	}

	switch s {
	case "1", "1.0", "1.00", "1.000":
		return 1.0, nil
	case "0.9":
		return 0.9, nil
	case "0.8":
		return 0.8, nil
	case "0.7":
		return 0.7, nil
	case "0.6":
		return 0.6, nil
	case "0.5":
		return 0.5, nil
	case "0":
		return 0.0, nil
	default:
		var q float64
		_, err := fmt.Sscanf(s, "%f", &q)
		if err != nil {
			return 1.0, err
		}
		if q < 0 {
			q = 0
		}
		if q > 1 {
			q = 1
		}
		return q, nil
	}
}
