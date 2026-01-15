package response

import (
	"context"
	"net/http"
	"testing"
)

func TestGetBasePath(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	t.Run("returns empty when missing", func(t *testing.T) {
		if got := getBasePath(req); got != "" {
			t.Fatalf("expected empty base path, got %q", got)
		}
	})

	t.Run("returns value from context", func(t *testing.T) {
		ctx := context.WithValue(req.Context(), BasePathContextKey, "/odata")
		reqWithBasePath := req.WithContext(ctx)

		if got := getBasePath(reqWithBasePath); got != "/odata" {
			t.Fatalf("expected base path %q, got %q", "/odata", got)
		}
	})
}
