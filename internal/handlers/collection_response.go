package handlers

import (
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
)

func (h *EntityHandler) collectionResponseWriter(w http.ResponseWriter, r *http.Request, pref *preference.Preference) func(*query.QueryOptions, interface{}, *int64, *string) error {
	return func(queryOptions *query.QueryOptions, results interface{}, totalCount *int64, nextLink *string) error {
		// Record result count in metrics
		if h.observability != nil {
			ctx := r.Context()
			count := int64(0)
			if results != nil {
				rv := reflect.ValueOf(results)
				if rv.Kind() == reflect.Slice {
					count = int64(rv.Len())
				}
			}
			h.observability.Metrics().RecordResultCount(ctx, h.metadata.EntitySetName, count)
		}

		var deltaLink *string
		if pref.TrackChangesRequested {
			if !h.supportsTrackChanges() {
				return &collectionRequestError{
					StatusCode: http.StatusNotImplemented,
					ErrorCode:  ErrMsgNotImplemented,
					Message:    "Change tracking is not enabled for this entity set",
				}
			}

			token, err := h.tracker.CurrentToken(h.metadata.EntitySetName)
			if err != nil {
				return &collectionRequestError{
					StatusCode: http.StatusInternalServerError,
					ErrorCode:  ErrMsgInternalError,
					Message:    err.Error(),
				}
			}

			link := response.BuildDeltaLink(r, token)
			deltaLink = &link
			pref.ApplyTrackChanges()
		}

		if applied := pref.GetPreferenceApplied(); applied != "" {
			w.Header().Set(HeaderPreferenceApplied, applied)
		}

		expandedProps := make([]string, len(queryOptions.Expand))
		for i, exp := range queryOptions.Expand {
			expandedProps[i] = exp.NavigationProperty
		}

		metadataProvider := newMetadataAdapter(h.metadata, h.namespace)
		if err := response.WriteODataCollectionWithNavigationAndDelta(w, r, h.metadata.EntitySetName, results, totalCount, nextLink, deltaLink, metadataProvider, expandedProps, h.metadata); err != nil {
			h.logger.Error("Error writing OData response", "error", err)
		}

		return nil
	}
}
