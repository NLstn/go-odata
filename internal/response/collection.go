package response

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/metadata"
)

// WriteODataCollection writes an OData collection response.
func WriteODataCollection(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string) error {
	return writeODataCollectionResponse(w, r, entitySetName, data, count, nextLink, nil)
}

// WriteODataCollectionWithDelta writes an OData collection response that includes a delta link.
func WriteODataCollectionWithDelta(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string) error {
	return writeODataCollectionResponse(w, r, entitySetName, data, count, nextLink, deltaLink)
}

// WriteODataCollectionWithNavigation writes an OData collection response with navigation links.
func WriteODataCollectionWithNavigation(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string, metadata EntityMetadataProvider, expandedProps []string, fullMetadata *metadata.EntityMetadata) error {
	return writeODataCollectionWithNavigationResponse(w, r, entitySetName, data, count, nextLink, nil, metadata, expandedProps, fullMetadata)
}

// WriteODataCollectionWithNavigationAndDelta writes an OData collection response with navigation links and a delta link.
func WriteODataCollectionWithNavigationAndDelta(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, metadata EntityMetadataProvider, expandedProps []string, fullMetadata *metadata.EntityMetadata) error {
	return writeODataCollectionWithNavigationResponse(w, r, entitySetName, data, count, nextLink, deltaLink, metadata, expandedProps, fullMetadata)
}

func writeODataCollectionResponse(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses.")
	}

	metadataLevel := GetODataMetadataLevel(r)

	contextURL := ""
	if metadataLevel != "none" {
		contextURL = buildContextURL(r, entitySetName)
	}

	if data == nil {
		data = []interface{}{}
	}

	response := map[string]interface{}{
		"value": data,
	}

	if contextURL != "" {
		response["@odata.context"] = contextURL
	}
	if count != nil {
		response["@odata.count"] = *count
	}
	if nextLink != nil && *nextLink != "" {
		response["@odata.nextLink"] = *nextLink
	}
	if deltaLink != nil && *deltaLink != "" {
		response["@odata.deltaLink"] = *deltaLink
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return nil
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

func writeODataCollectionWithNavigationResponse(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, metadata EntityMetadataProvider, expandedProps []string, fullMetadata *metadata.EntityMetadata) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses.")
	}

	metadataLevel := GetODataMetadataLevel(r)

	contextURL := ""
	if metadataLevel != "none" {
		contextURL = buildContextURL(r, entitySetName)
	}

	transformedData := addNavigationLinks(data, metadata, expandedProps, r, entitySetName, metadataLevel, fullMetadata)
	if transformedData == nil {
		transformedData = []interface{}{}
	}

	response := map[string]interface{}{
		"value": transformedData,
	}

	if contextURL != "" {
		response["@odata.context"] = contextURL
	}
	if count != nil {
		response["@odata.count"] = *count
	}
	if nextLink != nil && *nextLink != "" {
		response["@odata.nextLink"] = *nextLink
	}
	if deltaLink != nil && *deltaLink != "" {
		response["@odata.deltaLink"] = *deltaLink
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return nil
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

// WriteODataDeltaResponse writes an OData delta response containing change tracking entries.
func WriteODataDeltaResponse(w http.ResponseWriter, r *http.Request, entitySetName string, entries []map[string]interface{}, deltaLink *string) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses.")
	}

	metadataLevel := GetODataMetadataLevel(r)

	if entries == nil {
		entries = []map[string]interface{}{}
	}

	response := map[string]interface{}{
		"value": entries,
	}

	if metadataLevel != "none" {
		response["@odata.context"] = buildDeltaContextURL(r, entitySetName)
	}
	if deltaLink != nil && *deltaLink != "" {
		response["@odata.deltaLink"] = *deltaLink
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return nil
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}
