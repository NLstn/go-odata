package version

import (
	"context"
	"testing"
)

func TestVersion_String(t *testing.T) {
	tests := []struct {
		name     string
		version  Version
		expected string
	}{
		{"4.0", Version{4, 0}, "4.0"},
		{"4.01", Version{4, 1}, "4.01"},
		{"4.12", Version{4, 12}, "4.12"},
		{"5.0", Version{5, 0}, "5.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.String()
			if result != tt.expected {
				t.Errorf("Version.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestVersion_LessThanOrEqual(t *testing.T) {
	tests := []struct {
		name     string
		v1       Version
		v2       Version
		expected bool
	}{
		{"4.0 <= 4.0", Version{4, 0}, Version{4, 0}, true},
		{"4.0 <= 4.01", Version{4, 0}, Version{4, 1}, true},
		{"4.01 <= 4.0", Version{4, 1}, Version{4, 0}, false},
		{"4.12 <= 5.0", Version{4, 12}, Version{5, 0}, true},
		{"5.0 <= 4.12", Version{5, 0}, Version{4, 12}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v1.LessThanOrEqual(tt.v2)
			if result != tt.expected {
				t.Errorf("%v.LessThanOrEqual(%v) = %v, want %v", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestVersion_Supports(t *testing.T) {
	tests := []struct {
		name     string
		version  Version
		feature  string
		expected bool
	}{
		{"4.0 in-operator", Version{4, 0}, "in-operator", false},
		{"4.01 in-operator", Version{4, 1}, "in-operator", true},
		{"4.0 case-insensitive-functions", Version{4, 0}, "case-insensitive-functions", false},
		{"4.01 case-insensitive-functions", Version{4, 1}, "case-insensitive-functions", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.Supports(tt.feature)
			if result != tt.expected {
				t.Errorf("%v.Supports(%q) = %v, want %v", tt.version, tt.feature, result, tt.expected)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedMajor int
		expectedMinor int
		expectError   bool
	}{
		{"4.0", "4.0", 4, 0, false},
		{"4.01", "4.01", 4, 1, false},
		{"4", "4", 4, 0, false},
		{"invalid minor", "4.x", 4, 0, false},
		{"with spaces", "  4.0  ", 4, 0, false},
		{"empty string", "", 0, 0, true},
		{"invalid", "abc", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, err := parseVersion(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("parseVersion(%q) expected error but got none", tt.input)
				}
			} else if err != nil {
				t.Errorf("parseVersion(%q) unexpected error: %v", tt.input, err)
			}
			if major != tt.expectedMajor || minor != tt.expectedMinor {
				t.Errorf("parseVersion(%q) = (%d, %d), want (%d, %d)",
					tt.input, major, minor, tt.expectedMajor, tt.expectedMinor)
			}
		})
	}
}

func TestParseVersionString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Version
	}{
		{"empty string", "", Version{0, 0}},
		{"invalid", "abc", Version{0, 0}},
		{"invalid minor", "4.x", Version{4, 0}},
		{"valid", "4.01", Version{4, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseVersionString(tt.input)
			if result != tt.expected {
				t.Errorf("ParseVersionString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNegotiateVersion(t *testing.T) {
	tests := []struct {
		name             string
		clientMaxVersion string
		expected         Version
	}{
		{"no header", "", Version{4, 1}},
		{"client max 4.0", "4.0", Version{4, 0}},
		{"client max 4.01", "4.01", Version{4, 1}},
		{"client max 5.0", "5.0", Version{4, 1}},
		{"client max 10.0", "10.0", Version{4, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NegotiateVersion(tt.clientMaxVersion)
			if result != tt.expected {
				t.Errorf("NegotiateVersion(%q) = %v, want %v", tt.clientMaxVersion, result, tt.expected)
			}
		})
	}
}

func TestWithVersion_GetVersion(t *testing.T) {
	ctx := context.Background()
	version := Version{4, 0}

	ctx = WithVersion(ctx, version)

	retrieved := GetVersion(ctx)

	if retrieved != version {
		t.Errorf("GetVersion() = %v, want %v", retrieved, version)
	}
}

func TestGetVersion_Default(t *testing.T) {
	ctx := context.Background()
	retrieved := GetVersion(ctx)
	expected := Version{4, 1}

	if retrieved != expected {
		t.Errorf("GetVersion() without context value = %v, want %v", retrieved, expected)
	}
}
