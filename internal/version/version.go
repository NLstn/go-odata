package version

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const negotiatedVersionKey contextKey = "odata.negotiated.version"

// Version represents an OData protocol version
type Version struct {
	Major int
	Minor int
}

// Pre-computed string representations for the two standard OData versions.
// These avoid fmt.Sprintf allocations on every response header write.
const (
	v400String = "4.0"
	v401String = "4.01"
)

// String returns the version as a string in "Major.Minor" format
// For minor version 1, returns "4.01" to match OData convention
func (v Version) String() string {
	// Fast path for the two common OData versions
	if v.Major == 4 {
		if v.Minor == 0 {
			return v400String
		}
		if v.Minor == 1 {
			return v401String
		}
	}
	if v.Minor == 0 {
		return fmt.Sprintf("%d.0", v.Major)
	}
	if v.Minor < 10 {
		return fmt.Sprintf("%d.0%d", v.Major, v.Minor)
	}
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// LessThanOrEqual compares two versions using decimal comparison
func (v Version) LessThanOrEqual(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	return v.Minor <= other.Minor
}

// Supports returns whether this version supports a specific feature
// This can be used for feature detection in version-specific handlers
func (v Version) Supports(feature string) bool {
	switch feature {
	case "in-operator":
		// The 'in' operator was added in OData 4.01
		return v.Major > 4 || (v.Major == 4 && v.Minor >= 1)
	case "case-insensitive-functions":
		// Case-insensitive string functions added in 4.01
		return v.Major > 4 || (v.Major == 4 && v.Minor >= 1)
	default:
		return false
	}
}

// parseVersion parses a version string like "4.0" or "4.01" into major and minor components.
// Returns an error if the version string is invalid.
func parseVersion(version string) (int, int, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return 0, 0, fmt.Errorf("empty version string")
	}

	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return 0, 0, fmt.Errorf("invalid version format: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid major version in %s: %w", version, err)
	}

	minor := 0
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			// Treat invalid minor version as 0 but log it
			slog.Debug("Invalid OData minor version, treating as 0", "version", version, "error", err)
			minor = 0
		}
	}

	return major, minor, nil
}

// ParseVersionString parses a version string and returns a Version struct.
// Returns a zero version (0.0) if parsing fails, with error logged.
func ParseVersionString(versionStr string) Version {
	major, minor, err := parseVersion(versionStr)
	if err != nil {
		slog.Warn("Failed to parse OData version string", "version", versionStr, "error", err)
		return Version{Major: 0, Minor: 0}
	}
	return Version{Major: major, Minor: minor}
}

// NegotiateVersion determines the OData version to use in the response
// based on the client's OData-MaxVersion header.
// It returns the highest version supported by the service that is less than
// or equal to the client's requested maximum version.
// Assumes the client's version has already been validated as >= 4.0
func NegotiateVersion(clientMaxVersion string) Version {
	// Define the service's supported versions (highest to lowest)
	serviceVersion := Version{Major: 4, Minor: 1}
	supportedVersions := []Version{
		{Major: 4, Minor: 1},
		{Major: 4, Minor: 0},
	}

	// If no max version specified, return the highest supported version
	if clientMaxVersion == "" {
		return serviceVersion
	}

	// Parse the client's maximum version
	clientMax := ParseVersionString(clientMaxVersion)

	// Find the highest supported version that is <= client's max
	for _, supported := range supportedVersions {
		if supported.LessThanOrEqual(clientMax) {
			return supported
		}
	}

	// Fallback to v4.0 (shouldn't normally reach here if validation is done first)
	return Version{Major: 4, Minor: 0}
}

// WithVersion stores the negotiated OData version in the request context
func WithVersion(ctx context.Context, version Version) context.Context {
	return context.WithValue(ctx, negotiatedVersionKey, version)
}

// GetVersion retrieves the negotiated OData version from the request context
// If no version is stored, it returns the default version (4.01)
func GetVersion(ctx context.Context) Version {
	if v, ok := ctx.Value(negotiatedVersionKey).(Version); ok {
		return v
	}
	// Default to the highest supported version
	return Version{Major: 4, Minor: 1}
}
