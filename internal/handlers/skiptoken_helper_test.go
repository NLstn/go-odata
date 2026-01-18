package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/skiptoken"
)

type skipTokenEntity struct {
	ID   int    `json:"id"`
	Rank int    `json:"rank"`
	Name string `json:"name"`
}

func TestBuildNextLinkWithSkipTokenReturnsNil(t *testing.T) {
	top := 1
	tooLargeTop := 2
	meta := &metadata.EntityMetadata{}
	request := httptest.NewRequest(http.MethodGet, "http://example.test/Entities", nil)

	tests := []struct {
		name         string
		queryOptions *query.QueryOptions
		sliceValue   interface{}
	}{
		{
			name:         "Top is nil",
			queryOptions: &query.QueryOptions{},
			sliceValue:   []skipTokenEntity{{ID: 1}},
		},
		{
			name:         "Input is not a slice",
			queryOptions: &query.QueryOptions{Top: &top},
			sliceValue:   skipTokenEntity{ID: 1},
		},
		{
			name:         "Top exceeds slice length",
			queryOptions: &query.QueryOptions{Top: &tooLargeTop},
			sliceValue:   []skipTokenEntity{{ID: 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildNextLinkWithSkipToken(meta, tt.queryOptions, tt.sliceValue, request)
			if result != nil {
				t.Fatalf("expected nil next link, got %q", *result)
			}
		})
	}
}

func TestBuildNextLinkWithSkipTokenSuccess(t *testing.T) {
	top := 2
	entities := []skipTokenEntity{{ID: 1, Rank: 10, Name: "First"}, {ID: 2, Rank: 20, Name: "Second"}}
	meta := &metadata.EntityMetadata{
		KeyProperties: []metadata.PropertyMetadata{{JsonName: "id"}},
	}
	queryOptions := &query.QueryOptions{
		Top: &top,
		OrderBy: []query.OrderByItem{{
			Property:   "rank",
			Descending: true,
		}},
	}
	request := httptest.NewRequest(http.MethodGet, "http://example.test/Entities", nil)

	nextLink := buildNextLinkWithSkipToken(meta, queryOptions, entities, request)
	if nextLink == nil {
		t.Fatal("expected next link, got nil")
	}

	parsedURL, err := url.Parse(*nextLink)
	if err != nil {
		t.Fatalf("failed to parse next link: %v", err)
	}

	encodedToken := parsedURL.Query().Get("$skiptoken")
	if encodedToken == "" {
		t.Fatalf("expected $skiptoken query parameter, got %q", *nextLink)
	}

	decoded, err := skiptoken.Decode(encodedToken)
	if err != nil {
		t.Fatalf("failed to decode skiptoken: %v", err)
	}

	if decoded.KeyValues["id"] != float64(2) {
		t.Fatalf("expected key id 2, got %v", decoded.KeyValues["id"])
	}
	if decoded.OrderByValues["rank"] != float64(20) {
		t.Fatalf("expected orderby rank 20, got %v", decoded.OrderByValues["rank"])
	}
}
