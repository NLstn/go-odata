package query

import (
	"testing"
)

func TestFindExpandOption(t *testing.T) {
	expandOpts := []ExpandOption{
		{NavigationProperty: "category"},
		{NavigationProperty: "tags"},
		{NavigationProperty: "supplier"},
	}

	t.Run("find by exact property name", func(t *testing.T) {
		result := FindExpandOption(expandOpts, "category", "")
		if result == nil {
			t.Fatal("expected to find expand option")
		}
		if result.NavigationProperty != "category" {
			t.Errorf("expected 'category', got %s", result.NavigationProperty)
		}
	})

	t.Run("find by case insensitive property name", func(t *testing.T) {
		result := FindExpandOption(expandOpts, "CATEGORY", "")
		if result == nil {
			t.Fatal("expected to find expand option (case insensitive)")
		}
		if result.NavigationProperty != "category" {
			t.Errorf("expected 'category', got %s", result.NavigationProperty)
		}
	})

	t.Run("find by json name", func(t *testing.T) {
		result := FindExpandOption(expandOpts, "", "tags")
		if result == nil {
			t.Fatal("expected to find expand option by json name")
		}
		if result.NavigationProperty != "tags" {
			t.Errorf("expected 'tags', got %s", result.NavigationProperty)
		}
	})

	t.Run("find by case insensitive json name", func(t *testing.T) {
		result := FindExpandOption(expandOpts, "", "TAGS")
		if result == nil {
			t.Fatal("expected to find expand option by json name (case insensitive)")
		}
		if result.NavigationProperty != "tags" {
			t.Errorf("expected 'tags', got %s", result.NavigationProperty)
		}
	})

	t.Run("not found returns nil", func(t *testing.T) {
		result := FindExpandOption(expandOpts, "nonExistent", "")
		if result != nil {
			t.Error("expected nil for non-existent property")
		}
	})

	t.Run("empty expand options returns nil", func(t *testing.T) {
		result := FindExpandOption([]ExpandOption{}, "category", "")
		if result != nil {
			t.Error("expected nil for empty expand options")
		}
	})

	t.Run("nil expand options returns nil", func(t *testing.T) {
		result := FindExpandOption(nil, "category", "")
		if result != nil {
			t.Error("expected nil for nil expand options")
		}
	})
}
