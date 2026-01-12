package response

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// EdmDecimalProvider defines the interface for types that provide
// high-precision decimal values for OData Edm.Decimal fields.
//
// Types implementing this interface are responsible for:
// - Managing their own precision and scale
// - Applying rounding according to metadata constraints
// - Returning a valid decimal string representation
type EdmDecimalProvider interface {
	// EdmDecimalString returns the decimal value as a string suitable for OData JSON.
	// Format: optional minus sign, digits, optional decimal point and digits
	// Examples: "123.45", "-0.001", "1000", "0.1"
	//
	// The implementation MUST:
	// - Return a string parseable as a float64
	// - Honor precision/scale constraints if specified in struct tags
	// - NOT include quotes, spaces, or other formatting
	EdmDecimalString() string
}

// ConvertToEdmType converts a Go value to the appropriate representation for the given EDM type.
// Returns the converted value or error if conversion fails.
func ConvertToEdmType(value interface{}, edmType string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch edmType {
	case "Edm.Date":
		return convertToEdmDate(value)
	case "Edm.TimeOfDay":
		return convertToEdmTimeOfDay(value)
	case "Edm.Duration":
		return convertToEdmDuration(value)
	case "Edm.Decimal":
		return convertToEdmDecimal(value)
	default:
		// For all other EDM types, use standard JSON marshaling
		return value, nil
	}
}

func convertToEdmDate(value interface{}) (interface{}, error) {
	// Handle empty string as nil (common in database results)
	if s, ok := value.(string); ok && s == "" {
		return nil, nil
	}

	t, ok := value.(time.Time)
	if !ok {
		return nil, fmt.Errorf("Edm.Date requires time.Time, got %T", value)
	}

	// Check for zero time (uninitialized time.Time)
	if t.IsZero() {
		return nil, nil
	}

	// Always use UTC for date extraction to ensure consistency
	return t.UTC().Format("2006-01-02"), nil
}

func convertToEdmTimeOfDay(value interface{}) (interface{}, error) {
	// Handle empty string as nil (common in database results)
	if s, ok := value.(string); ok && s == "" {
		return nil, nil
	}

	t, ok := value.(time.Time)
	if !ok {
		return nil, fmt.Errorf("Edm.TimeOfDay requires time.Time, got %T", value)
	}

	// Check for zero time (uninitialized time.Time)
	if t.IsZero() {
		return nil, nil
	}

	return t.Format("15:04:05.000000"), nil
}

func convertToEdmDuration(value interface{}) (interface{}, error) {
	// Handle empty string as nil (common in database results)
	if s, ok := value.(string); ok && s == "" {
		return nil, nil
	}

	d, ok := value.(time.Duration)
	if !ok {
		return nil, fmt.Errorf("Edm.Duration requires time.Duration, got %T", value)
	}

	// ISO 8601 duration format: PnDTnHnMnS
	// This implementation supports days, hours, minutes, and seconds.
	//
	// Limitations:
	// - No support for years (P) or months (M) - time.Duration doesn't have these concepts
	// - Fractional seconds are truncated to whole seconds
	// - Negative durations format with minus sign: -PT1H30M
	//
	// For nanosecond precision or more complex duration formats, the calling code
	// should use a custom type implementing EdmDecimalProvider or similar interface.

	isNegative := d < 0
	if isNegative {
		d = -d
	}

	// Calculate components
	totalSeconds := int(d.Seconds())
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	// Build ISO 8601 duration string
	var result string
	if isNegative {
		result = "-P"
	} else {
		result = "P"
	}

	// Add days if present
	if days > 0 {
		result += fmt.Sprintf("%dD", days)
	}

	// Time component (always include T if we have hours/minutes/seconds)
	if hours > 0 || minutes > 0 || seconds > 0 || days == 0 {
		result += "T"
		if hours > 0 {
			result += fmt.Sprintf("%dH", hours)
		}
		if minutes > 0 {
			result += fmt.Sprintf("%dM", minutes)
		}
		if seconds > 0 || (hours == 0 && minutes == 0 && days == 0) {
			result += fmt.Sprintf("%dS", seconds)
		}
	}

	return result, nil
}

func convertToEdmDecimal(value interface{}) (interface{}, error) {
	// Handle string values (from any source that marshals to string)
	if s, ok := value.(string); ok {
		if s == "" {
			return nil, nil
		}
		// Validate that the string is a valid numeric value
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return nil, fmt.Errorf("Edm.Decimal requires valid numeric string, got %q: %w", s, err)
		}
		// Return as json.Number to preserve precision
		return json.Number(s), nil
	}

	// Check if type implements EdmDecimalProvider interface
	if provider, ok := value.(EdmDecimalProvider); ok {
		str := provider.EdmDecimalString()
		// Handle empty string from provider
		if str == "" {
			return nil, nil
		}
		// Return as json.Number to preserve precision
		return json.Number(str), nil
	}

	// Fall back to primitive types
	switch v := value.(type) {
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return v, nil
	default:
		return nil, fmt.Errorf("Edm.Decimal requires EdmDecimalProvider interface or numeric type, got %T", value)
	}
}
