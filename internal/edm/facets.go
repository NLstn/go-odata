package edm

import (
	"fmt"
	"strconv"
	"strings"
)

// Facets contains metadata attributes that constrain EDM type values
type Facets struct {
	Precision *int  // For Decimal: total number of digits
	Scale     *int  // For Decimal: digits after decimal point
	MaxLength *int  // For String, Binary: maximum length
	Unicode   *bool // For String: whether Unicode is supported
	SRID      *int  // For Geography/Geometry: spatial reference ID
	Nullable  bool  // Whether null values are allowed
}

// ParseTypeFromTag extracts EDM type name and facets from an odata struct tag
// Example tags:
//   - "type=Edm.Decimal,precision=18,scale=4"
//   - "nullable,type=Edm.Date"
//   - "maxLength=50"
func ParseTypeFromTag(tag string) (typeName string, facets Facets, err error) {
	if tag == "" {
		return "", Facets{}, nil
	}

	// Split by comma
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle boolean flags
		if part == "nullable" {
			facets.Nullable = true
			continue
		}

		// Handle key=value pairs
		if !strings.Contains(part, "=") {
			// Unknown flag, skip it (could be 'key', 'searchable', etc.)
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "type":
			typeName = value

		case "precision":
			precision, parseErr := strconv.Atoi(value)
			if parseErr != nil {
				return "", Facets{}, fmt.Errorf("invalid precision value: %s", value)
			}
			facets.Precision = &precision

		case "scale":
			scale, parseErr := strconv.Atoi(value)
			if parseErr != nil {
				return "", Facets{}, fmt.Errorf("invalid scale value: %s", value)
			}
			facets.Scale = &scale

		case "maxLength":
			maxLength, parseErr := strconv.Atoi(value)
			if parseErr != nil {
				return "", Facets{}, fmt.Errorf("invalid maxLength value: %s", value)
			}
			facets.MaxLength = &maxLength

		case "unicode":
			unicode, parseErr := strconv.ParseBool(value)
			if parseErr != nil {
				return "", Facets{}, fmt.Errorf("invalid unicode value: %s", value)
			}
			facets.Unicode = &unicode

		case "srid":
			srid, parseErr := strconv.Atoi(value)
			if parseErr != nil {
				return "", Facets{}, fmt.Errorf("invalid srid value: %s", value)
			}
			facets.SRID = &srid
		}
	}

	return typeName, facets, nil
}

// ValidateDecimalFacets validates that a decimal value conforms to precision and scale facets
func ValidateDecimalFacets(valueStr string, facets Facets) error {
	if facets.Precision == nil && facets.Scale == nil {
		return nil // No constraints
	}

	// Remove sign if present
	absValue := strings.TrimPrefix(valueStr, "-")
	absValue = strings.TrimPrefix(absValue, "+")

	// Split into integer and fractional parts
	parts := strings.Split(absValue, ".")
	fractionalPart := ""
	if len(parts) > 1 {
		fractionalPart = parts[1]
	}

	// Count total digits (excluding decimal point)
	totalDigits := len(strings.ReplaceAll(absValue, ".", ""))

	// Validate precision (total digits)
	if facets.Precision != nil {
		if totalDigits > *facets.Precision {
			return fmt.Errorf("value exceeds precision: %d digits (max %d)", totalDigits, *facets.Precision)
		}
	}

	// Validate scale (fractional digits)
	if facets.Scale != nil {
		if len(fractionalPart) > *facets.Scale {
			return fmt.Errorf("value exceeds scale: %d fractional digits (max %d)", len(fractionalPart), *facets.Scale)
		}
	}

	return nil
}

// ValidateLengthFacet validates that a value conforms to maxLength facet
func ValidateLengthFacet(length int, facets Facets) error {
	if facets.MaxLength != nil && length > *facets.MaxLength {
		return fmt.Errorf("value exceeds maxLength: %d (max %d)", length, *facets.MaxLength)
	}
	return nil
}
