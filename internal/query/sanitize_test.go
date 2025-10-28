package query

import (
	"testing"
)

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid simple identifier",
			input:    "MyProperty",
			expected: "MyProperty",
		},
		{
			name:     "valid identifier with underscores",
			input:    "my_property_123",
			expected: "my_property_123",
		},
		{
			name:     "valid identifier starting with underscore",
			input:    "_private",
			expected: "_private",
		},
		{
			name:     "OData reserved identifier $it",
			input:    "$it",
			expected: "$it",
		},
		{
			name:     "SQL injection attempt with semicolon",
			input:    "Property; DROP TABLE Users--",
			expected: "",
		},
		{
			name:     "SQL injection attempt with single quote",
			input:    "Property' OR '1'='1",
			expected: "",
		},
		{
			name:     "SQL injection attempt with comment",
			input:    "Property--comment",
			expected: "",
		},
		{
			name:     "SQL injection with space",
			input:    "Property Name",
			expected: "",
		},
		{
			name:     "SQL reserved keyword SELECT",
			input:    "SELECT",
			expected: "",
		},
		{
			name:     "SQL reserved keyword DROP",
			input:    "DROP",
			expected: "",
		},
		{
			name:     "SQL reserved keyword UNION",
			input:    "UNION",
			expected: "",
		},
		{
			name:     "identifier starting with number",
			input:    "123Property",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "identifier with special characters",
			input:    "Property$Name",
			expected: "",
		},
		{
			name:     "identifier with parentheses",
			input:    "Property()",
			expected: "",
		},
		{
			name:     "valid computed property name",
			input:    "TaxedPrice",
			expected: "TaxedPrice",
		},
		{
			name:     "valid computed property with numbers",
			input:    "Price2",
			expected: "Price2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeIdentifier(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeIdentifier_CaseInsensitiveKeywords(t *testing.T) {
	// Test that keywords are rejected regardless of case
	keywords := []string{
		"select", "SELECT", "Select",
		"drop", "DROP", "Drop",
		"insert", "INSERT", "Insert",
	}

	for _, keyword := range keywords {
		t.Run(keyword, func(t *testing.T) {
			result := sanitizeIdentifier(keyword)
			if result != "" {
				t.Errorf("sanitizeIdentifier(%q) should reject SQL keyword, got %q", keyword, result)
			}
		})
	}
}
