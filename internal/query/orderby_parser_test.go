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
