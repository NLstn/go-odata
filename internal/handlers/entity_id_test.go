package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

func TestBuildEntityIDFromResultSupportsStructAndMap(t *testing.T) {
	t.Parallel()

	type entity struct {
		ID string
	}
	type row map[string]interface{}

	handler := &EntityHandler{metadata: &metadata.EntityMetadata{
		EntitySetName: "Products",
		KeyProperties: []metadata.PropertyMetadata{
			{Name: "ID", FieldName: "ID", JsonName: "id"},
		},
	}}
	request := httptest.NewRequest("GET", "http://example.com/Products", nil)
	want := "http://example.com/Products('O''Brien')"

	for _, test := range []struct {
		name   string
		result interface{}
	}{
		{name: "struct", result: entity{ID: "O'Brien"}},
		{name: "map", result: map[string]interface{}{"id": "O'Brien"}},
		{name: "named map", result: row{"id": "O'Brien"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := handler.buildEntityIDFromResult(test.result, request); got != want {
				t.Fatalf("buildEntityIDFromResult() = %q, want %q", got, want)
			}
			if got := handler.buildEntityLocation(request, test.result); got != want {
				t.Fatalf("buildEntityLocation() = %q, want %q", got, want)
			}
		})
	}
}

func TestBuildEntityIDFromResultRequiresAllKeys(t *testing.T) {
	t.Parallel()

	handler := &EntityHandler{metadata: &metadata.EntityMetadata{
		EntitySetName: "Products",
		KeyProperties: []metadata.PropertyMetadata{
			{Name: "ID", JsonName: "id"},
			{Name: "Locale", JsonName: "locale"},
		},
	}}
	request := httptest.NewRequest("GET", "http://example.com/Products", nil)

	if got := handler.buildEntityIDFromResult(map[string]interface{}{"id": 1}, request); got != "" {
		t.Fatalf("buildEntityIDFromResult() = %q, want empty ID", got)
	}
}
