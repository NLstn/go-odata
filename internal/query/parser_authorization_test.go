package query

import (
	"net/url"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
)

type denyPropertyPolicy struct {
	denied map[string]bool
}

func (p denyPropertyPolicy) Authorize(_ auth.AuthContext, resource auth.ResourceDescriptor, _ auth.Operation) auth.Decision {
	path := strings.Join(resource.PropertyPath, "/")
	if p.denied[path] {
		return auth.Deny("blocked")
	}
	return auth.Allow()
}

func TestParseQueryOptionsFiltersUnauthorizedSelect(t *testing.T) {
	meta := getTestMetadata(t)
	params := url.Values{}
	params.Set("$select", "Name,Price")

	policy := denyPropertyPolicy{denied: map[string]bool{"Price": true}}
	options, err := ParseQueryOptions(params, meta, policy, auth.AuthContext{})
	if err != nil {
		t.Fatalf("ParseQueryOptions failed: %v", err)
	}

	if !options.SelectSpecified {
		t.Fatal("expected SelectSpecified to be true")
	}

	if len(options.Select) != 1 || options.Select[0] != "Name" {
		t.Fatalf("expected only Name to be selected, got %v", options.Select)
	}
}

func TestParseQueryOptionsFiltersUnauthorizedExpand(t *testing.T) {
	authorMeta, err := metadata.AnalyzeEntity(&TestAuthor{})
	if err != nil {
		t.Fatalf("AnalyzeEntity failed: %v", err)
	}

	params := url.Values{}
	params.Set("$expand", "Books($select=Title,AuthorID)")

	policy := denyPropertyPolicy{denied: map[string]bool{"Books/Title": true}}
	options, err := ParseQueryOptions(params, authorMeta, policy, auth.AuthContext{})
	if err != nil {
		t.Fatalf("ParseQueryOptions failed: %v", err)
	}

	if len(options.Expand) != 1 {
		t.Fatalf("expected 1 expand option, got %d", len(options.Expand))
	}

	expand := options.Expand[0]
	if !expand.SelectSpecified {
		t.Fatal("expected SelectSpecified to be true for expand")
	}

	if len(expand.Select) != 1 || expand.Select[0] != "AuthorID" {
		t.Fatalf("expected AuthorID to remain in select, got %v", expand.Select)
	}
}

func TestParseQueryOptionsRejectsUnauthorizedExpand(t *testing.T) {
	authorMeta, err := metadata.AnalyzeEntity(&TestAuthor{})
	if err != nil {
		t.Fatalf("AnalyzeEntity failed: %v", err)
	}

	params := url.Values{}
	params.Set("$expand", "Books")

	policy := denyPropertyPolicy{denied: map[string]bool{"Books": true}}
	options, err := ParseQueryOptions(params, authorMeta, policy, auth.AuthContext{})
	if err != nil {
		t.Fatalf("ParseQueryOptions failed: %v", err)
	}

	if len(options.Expand) != 0 {
		t.Fatalf("expected expand to be filtered, got %d options", len(options.Expand))
	}
}
