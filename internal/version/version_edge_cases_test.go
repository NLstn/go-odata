package version

import (
	"context"
	"sync"
	"testing"
)

// TestGetVersion_CorruptedContext tests GetVersion with invalid context values
func TestGetVersion_CorruptedContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected Version
	}{
		{
			name:     "empty context returns default",
			ctx:      context.Background(),
			expected: Version{Major: 4, Minor: 1},
		},
		{
			name:     "context with wrong type (string) returns default",
			ctx:      context.WithValue(context.Background(), negotiatedVersionKey, "4.01"),
			expected: Version{Major: 4, Minor: 1},
		},
		{
			name:     "context with wrong type (int) returns default",
			ctx:      context.WithValue(context.Background(), negotiatedVersionKey, 401),
			expected: Version{Major: 4, Minor: 1},
		},
		{
			name:     "context with wrong type (map) returns default",
			ctx:      context.WithValue(context.Background(), negotiatedVersionKey, map[string]int{"major": 4}),
			expected: Version{Major: 4, Minor: 1},
		},
		{
			name:     "context with nil value returns default",
			ctx:      context.WithValue(context.Background(), negotiatedVersionKey, nil),
			expected: Version{Major: 4, Minor: 1},
		},
		{
			name:     "context with valid Version works",
			ctx:      WithVersion(context.Background(), Version{Major: 4, Minor: 0}),
			expected: Version{Major: 4, Minor: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic with corrupted context
			result := GetVersion(tt.ctx)

			if result != tt.expected {
				t.Errorf("GetVersion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestWithVersion_Concurrent tests concurrent WithVersion calls
func TestWithVersion_Concurrent(t *testing.T) {
	versions := []Version{
		{Major: 4, Minor: 0},
		{Major: 4, Minor: 1},
		{Major: 5, Minor: 0},
	}

	var wg sync.WaitGroup
	// Run concurrent WithVersion calls
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ctx := context.Background()
			ver := versions[idx%len(versions)]
			ctx = WithVersion(ctx, ver)

			retrieved := GetVersion(ctx)
			if retrieved != ver {
				t.Errorf("Concurrent WithVersion/GetVersion mismatch: got %v, want %v", retrieved, ver)
			}
		}(i)
	}
	wg.Wait()
}

// TestParseVersionString_EdgeCases tests edge cases in version parsing
func TestParseVersionString_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Version
	}{
		{"very long version string", "4.01234567890", Version{4, 1234567890}},
		{"negative major", "-4.0", Version{-4, 0}}, // strconv.Atoi parses negative numbers
		{"negative minor", "4.-1", Version{4, -1}}, // strconv.Atoi parses negative numbers
		{"multiple dots", "4.0.1.2", Version{4, 0}},
		{"leading zeros", "04.01", Version{4, 1}},
		{"only whitespace", "   ", Version{0, 0}},
		{"special characters", "4.0!@#", Version{4, 0}},
		{"unicode characters", "4.0😀", Version{4, 0}},
		{"very large number", "999999999.0", Version{999999999, 0}},
		{"scientific notation", "4e2.0", Version{0, 0}},
		{"hex notation", "0x4.0", Version{0, 0}},
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

// TestNegotiateVersion_Concurrent tests concurrent version negotiation
func TestNegotiateVersion_Concurrent(t *testing.T) {
	versions := []string{"4.0", "4.01", "5.0", "", "invalid", "10.0"}
	results := make(chan Version, 100)

	// Run concurrent negotiations
	for i := 0; i < 100; i++ {
		go func(idx int) {
			ver := NegotiateVersion(versions[idx%len(versions)])
			results <- ver
		}(i)
	}

	// Collect results
	for i := 0; i < 100; i++ {
		ver := <-results
		// All results should be valid versions (4.0 or 4.1)
		if ver.Major != 4 || (ver.Minor != 0 && ver.Minor != 1) {
			t.Errorf("Concurrent NegotiateVersion returned unexpected version: %v", ver)
		}
	}
}

// TestVersion_Supports_Concurrent tests concurrent feature support checks
func TestVersion_Supports_Concurrent(t *testing.T) {
	versions := []Version{
		{Major: 4, Minor: 0},
		{Major: 4, Minor: 1},
		{Major: 5, Minor: 0},
	}
	features := []string{"in-operator", "case-insensitive-functions", "unknown-feature"}

	results := make(chan bool, 300)

	for i := 0; i < 100; i++ {
		go func(idx int) {
			ver := versions[idx%len(versions)]
			feat := features[idx%len(features)]
			result := ver.Supports(feat)
			results <- result
		}(i)
	}

	// Collect results (just verify no panics)
	for i := 0; i < 100; i++ {
		<-results
	}
}
