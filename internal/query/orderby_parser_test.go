package query

import (
	"net/url"
	"strings"
	"testing"
)

func TestParseOrderByRejectsExtraTokens(t *testing.T) {
	meta := getTestMetadata(t)

	params := url.Values{}
	params.Set("$orderby", "Name desc extra")

	_, err := ParseQueryOptionsWithConfig(params, meta, nil)
	if err == nil {
		t.Fatal("expected error for extra orderby tokens")
	}

	if !strings.Contains(err.Error(), "Name") {
		t.Fatalf("expected error to reference property 'Name', got %v", err)
	}

	if !strings.Contains(err.Error(), "extra") {
		t.Fatalf("expected error to reference offending token 'extra', got %v", err)
	}
}

func TestParseOrderByWithoutMetadata(t *testing.T) {
	t.Run("valid single property ascending", func(t *testing.T) {
		result, err := parseOrderByWithoutMetadata("Name")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result))
		}
		if result[0].Property != "Name" {
			t.Errorf("expected property 'Name', got %s", result[0].Property)
		}
		if result[0].Descending {
			t.Error("expected ascending order")
		}
	})

	t.Run("valid single property descending", func(t *testing.T) {
		result, err := parseOrderByWithoutMetadata("Name desc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result))
		}
		if result[0].Property != "Name" {
			t.Errorf("expected property 'Name', got %s", result[0].Property)
		}
		if !result[0].Descending {
			t.Error("expected descending order")
		}
	})

	t.Run("valid multiple properties", func(t *testing.T) {
		result, err := parseOrderByWithoutMetadata("Name asc, Price desc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 items, got %d", len(result))
		}
		if result[0].Property != "Name" {
			t.Errorf("expected property 'Name', got %s", result[0].Property)
		}
		if result[0].Descending {
			t.Error("expected ascending order for first property")
		}
		if result[1].Property != "Price" {
			t.Errorf("expected property 'Price', got %s", result[1].Property)
		}
		if !result[1].Descending {
			t.Error("expected descending order for second property")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		result, err := parseOrderByWithoutMetadata("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d items", len(result))
		}
	})

	t.Run("extra tokens error", func(t *testing.T) {
		_, err := parseOrderByWithoutMetadata("Name desc extra")
		if err == nil {
			t.Fatal("expected error for extra tokens")
		}
	})
}
