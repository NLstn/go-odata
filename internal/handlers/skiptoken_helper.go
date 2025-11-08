package handlers

import (
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/skiptoken"
)

// buildNextLinkWithSkipToken constructs a next link URL using $skiptoken when possible.
// The caller must ensure the result slice uses a deterministic ordering that matches the
// provided query options (e.g., explicit $orderby or stable key ordering) so that the
// generated token can be decoded reliably on subsequent requests.
func buildNextLinkWithSkipToken(
	meta *metadata.EntityMetadata,
	queryOptions *query.QueryOptions,
	sliceValue interface{},
	r *http.Request,
) *string {
	if queryOptions.Top == nil {
		return nil
	}

	v := reflect.ValueOf(sliceValue)
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return nil
	}

	lastIndex := *queryOptions.Top - 1
	if lastIndex < 0 || lastIndex >= v.Len() {
		return nil
	}

	lastEntity := v.Index(lastIndex).Interface()

	keyProps := make([]string, len(meta.KeyProperties))
	for i, kp := range meta.KeyProperties {
		keyProps[i] = kp.JsonName
	}

	orderByProps := make([]string, len(queryOptions.OrderBy))
	for i, ob := range queryOptions.OrderBy {
		orderByProps[i] = ob.Property
		if ob.Descending {
			orderByProps[i] += " desc"
		}
	}

	token, err := skiptoken.ExtractFromEntity(lastEntity, keyProps, orderByProps)
	if err != nil {
		return nil
	}

	encoded, err := skiptoken.Encode(token)
	if err != nil {
		return nil
	}

	nextURL := response.BuildNextLinkWithSkipToken(r, encoded)
	return &nextURL
}
