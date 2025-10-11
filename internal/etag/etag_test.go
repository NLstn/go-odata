package etag

import (
	"reflect"
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
)

// Test entity with different field types
type TestEntity struct {
	ID           int
	Version      int
	LastModified time.Time
	Name         string
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name         string
		entity       TestEntity
		etagProperty *metadata.PropertyMetadata
		wantEmpty    bool
	}{
		{
			name:   "Generate ETag from integer field",
			entity: TestEntity{ID: 1, Version: 5},
			etagProperty: &metadata.PropertyMetadata{
				FieldName: "Version",
				IsETag:    true,
			},
			wantEmpty: false,
		},
		{
			name:   "Generate ETag from string field",
			entity: TestEntity{ID: 1, Name: "Test"},
			etagProperty: &metadata.PropertyMetadata{
				FieldName: "Name",
				IsETag:    true,
			},
			wantEmpty: false,
		},
		{
			name:   "Generate ETag from time field",
			entity: TestEntity{ID: 1, LastModified: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			etagProperty: &metadata.PropertyMetadata{
				FieldName: "LastModified",
				IsETag:    true,
			},
			wantEmpty: false,
		},
		{
			name:         "No ETag property defined",
			entity:       TestEntity{ID: 1, Version: 5},
			etagProperty: nil,
			wantEmpty:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &metadata.EntityMetadata{
				EntityType:   reflect.TypeOf(TestEntity{}),
				ETagProperty: tt.etagProperty,
			}

			etag := Generate(&tt.entity, meta)

			if tt.wantEmpty {
				if etag != "" {
					t.Errorf("Generate() = %v, want empty string", etag)
				}
			} else {
				if etag == "" {
					t.Error("Generate() returned empty string, want non-empty")
				}
				// Check that it starts with W/" (weak ETag format)
				if len(etag) < 3 || etag[:3] != "W/\"" {
					t.Errorf("Generate() = %v, want format W/\"...\"", etag)
				}
			}
		})
	}
}

func TestGenerate_Consistency(t *testing.T) {
	// Test that the same input produces the same ETag
	entity := TestEntity{ID: 1, Version: 42}
	meta := &metadata.EntityMetadata{
		EntityType: reflect.TypeOf(TestEntity{}),
		ETagProperty: &metadata.PropertyMetadata{
			FieldName: "Version",
			IsETag:    true,
		},
	}

	etag1 := Generate(&entity, meta)
	etag2 := Generate(&entity, meta)

	if etag1 != etag2 {
		t.Errorf("Generate() produced different ETags for same input: %v vs %v", etag1, etag2)
	}
}

func TestGenerate_DifferentValues(t *testing.T) {
	// Test that different values produce different ETags
	entity1 := TestEntity{ID: 1, Version: 1}
	entity2 := TestEntity{ID: 1, Version: 2}
	meta := &metadata.EntityMetadata{
		EntityType: reflect.TypeOf(TestEntity{}),
		ETagProperty: &metadata.PropertyMetadata{
			FieldName: "Version",
			IsETag:    true,
		},
	}

	etag1 := Generate(&entity1, meta)
	etag2 := Generate(&entity2, meta)

	if etag1 == etag2 {
		t.Errorf("Generate() produced same ETag for different values: %v", etag1)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Weak ETag",
			input: "W/\"abc123\"",
			want:  "abc123",
		},
		{
			name:  "Strong ETag",
			input: "\"abc123\"",
			want:  "abc123",
		},
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
		{
			name:  "No quotes",
			input: "abc123",
			want:  "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if got != tt.want {
				t.Errorf("Parse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name        string
		ifMatch     string
		currentETag string
		want        bool
	}{
		{
			name:        "Exact match with weak ETags",
			ifMatch:     "W/\"abc123\"",
			currentETag: "W/\"abc123\"",
			want:        true,
		},
		{
			name:        "Exact match with strong and weak ETags",
			ifMatch:     "\"abc123\"",
			currentETag: "W/\"abc123\"",
			want:        true,
		},
		{
			name:        "No match",
			ifMatch:     "W/\"abc123\"",
			currentETag: "W/\"def456\"",
			want:        false,
		},
		{
			name:        "Wildcard matches any ETag",
			ifMatch:     "*",
			currentETag: "W/\"abc123\"",
			want:        true,
		},
		{
			name:        "Wildcard with empty current ETag",
			ifMatch:     "*",
			currentETag: "",
			want:        false,
		},
		{
			name:        "Empty If-Match always matches",
			ifMatch:     "",
			currentETag: "W/\"abc123\"",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Match(tt.ifMatch, tt.currentETag)
			if got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.ifMatch, tt.currentETag, got, tt.want)
			}
		})
	}
}

func TestNoneMatch(t *testing.T) {
	tests := []struct {
		name          string
		ifNoneMatch   string
		currentETag   string
		want          bool
		description   string
	}{
		{
			name:          "ETags match - should return false (304)",
			ifNoneMatch:   "W/\"abc123\"",
			currentETag:   "W/\"abc123\"",
			want:          false,
			description:   "When ETags match, NoneMatch returns false indicating 304 should be returned",
		},
		{
			name:          "ETags don't match - should return true (200)",
			ifNoneMatch:   "W/\"abc123\"",
			currentETag:   "W/\"def456\"",
			want:          true,
			description:   "When ETags don't match, NoneMatch returns true indicating normal response",
		},
		{
			name:          "Strong and weak ETags match",
			ifNoneMatch:   "\"abc123\"",
			currentETag:   "W/\"abc123\"",
			want:          false,
			description:   "Strong and weak ETags with same value match",
		},
		{
			name:          "Empty If-None-Match always returns true",
			ifNoneMatch:   "",
			currentETag:   "W/\"abc123\"",
			want:          true,
			description:   "No If-None-Match header means no condition",
		},
		{
			name:          "Wildcard with existing entity",
			ifNoneMatch:   "*",
			currentETag:   "W/\"abc123\"",
			want:          false,
			description:   "Wildcard matches any existing entity",
		},
		{
			name:          "Wildcard with empty current ETag",
			ifNoneMatch:   "*",
			currentETag:   "",
			want:          true,
			description:   "Wildcard with no entity means none-match is true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NoneMatch(tt.ifNoneMatch, tt.currentETag)
			if got != tt.want {
				t.Errorf("NoneMatch(%q, %q) = %v, want %v\nDescription: %s", 
					tt.ifNoneMatch, tt.currentETag, got, tt.want, tt.description)
			}
		})
	}
}
