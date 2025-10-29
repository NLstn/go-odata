package handlers

import (
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/trackchanges"
)

func (h *EntityHandler) handleDeltaCollection(w http.ResponseWriter, r *http.Request, token string) {
	if !h.supportsTrackChanges() {
		WriteError(w, http.StatusNotImplemented, ErrMsgNotImplemented,
			"Change tracking is not enabled for this entity set")
		return
	}

	entitySet, err := h.tracker.EntitySetFromToken(token)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"Invalid $deltatoken value")
		return
	}

	if entitySet != h.metadata.EntitySetName {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"Delta token does not match the requested entity set")
		return
	}

	events, newToken, err := h.tracker.ChangesSince(token)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error())
		return
	}

	entries := h.buildDeltaEntries(r, events)
	deltaLink := response.BuildDeltaLink(r, newToken)

	if err := response.WriteODataDeltaResponse(w, r, h.metadata.EntitySetName, entries, &deltaLink); err != nil {
		fmt.Printf("Error writing delta response: %v\n", err)
	}
}

func (h *EntityHandler) buildDeltaEntries(r *http.Request, events []trackchanges.ChangeEvent) []map[string]interface{} {
	metadataLevel := response.GetODataMetadataLevel(r)
	includeMetadata := metadataLevel != "none"
	baseURL := response.BuildBaseURL(r)
	entityTypeAnnotation := ""
	if metadataLevel == "full" {
		entityTypeAnnotation = fmt.Sprintf("#ODataService.%s", h.metadata.EntityName)
	}

	entries := make([]map[string]interface{}, 0, len(events))

	for _, event := range events {
		entityID := response.BuildEntityID(h.metadata.EntitySetName, event.KeyValues)
		resourceID := baseURL + "/" + entityID

		switch event.Type {
		case trackchanges.ChangeTypeAdded, trackchanges.ChangeTypeUpdated:
			entry := make(map[string]interface{})
			for k, v := range event.Data {
				entry[k] = v
			}
			if includeMetadata {
				entry["@odata.id"] = resourceID
				if h.metadata.ETagProperty != nil {
					if etagValue := etag.Generate(event.Data, h.metadata); etagValue != "" {
						entry["@odata.etag"] = etagValue
					}
				}
				if entityTypeAnnotation != "" {
					entry["@odata.type"] = entityTypeAnnotation
				}
			}
			entries = append(entries, entry)
		case trackchanges.ChangeTypeDeleted:
			entry := make(map[string]interface{})
			if includeMetadata {
				entry["@odata.id"] = resourceID
				if entityTypeAnnotation != "" {
					entry["@odata.type"] = entityTypeAnnotation
				}
			}
			entry["@odata.removed"] = map[string]string{"reason": "deleted"}
			for k, v := range event.KeyValues {
				entry[k] = v
			}
			entries = append(entries, entry)
		}
	}

	return entries
}
