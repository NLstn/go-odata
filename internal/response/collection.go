package response

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
)

// WriteODataCollection writes an OData collection response.
func WriteODataCollection(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string) error {
	return writeODataCollectionResponse(w, r, entitySetName, data, count, nextLink, nil, nil)
}

// WriteODataCollectionWithSelect writes an OData collection response with a pre-computed list of
// selected/output properties used to build the @odata.context URL.
// selectedProps should contain the $select properties so that the context URL is shaped as
// #EntitySet(prop1,prop2) per the OData spec.
func WriteODataCollectionWithSelect(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string, selectedProps []string) error {
	return writeODataCollectionResponse(w, r, entitySetName, data, count, nextLink, nil, selectedProps)
}

// WriteODataCollectionWithDelta writes an OData collection response that includes a delta link.
func WriteODataCollectionWithDelta(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string) error {
	return writeODataCollectionResponse(w, r, entitySetName, data, count, nextLink, deltaLink, nil)
}

// WriteODataCollectionWithNavigation writes an OData collection response with navigation links.
func WriteODataCollectionWithNavigation(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, fullMetadata *metadata.EntityMetadata) error {
	return writeODataCollectionWithNavigationResponse(w, r, entitySetName, data, count, nextLink, nil, metadata, expandOptions, selectedNavProps, fullMetadata, nil)
}

// WriteODataCollectionWithNavigationAndDelta writes an OData collection response with navigation links and a delta link.
func WriteODataCollectionWithNavigationAndDelta(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, fullMetadata *metadata.EntityMetadata) error {
	return writeODataCollectionWithNavigationResponse(w, r, entitySetName, data, count, nextLink, deltaLink, metadata, expandOptions, selectedNavProps, fullMetadata, nil)
}

// WriteODataCollectionWithNavigationAndSelect writes an OData collection response with navigation links
// and a pre-computed list of selected/output properties used to build the @odata.context URL.
// selectedProps should contain the $select properties (or $apply output properties) so that the
// context URL is shaped as #EntitySet(prop1,prop2) per the OData spec.
func WriteODataCollectionWithNavigationAndSelect(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, md EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, fullMetadata *metadata.EntityMetadata, selectedProps []string) error {
	return writeODataCollectionWithNavigationResponse(w, r, entitySetName, data, count, nextLink, deltaLink, md, expandOptions, selectedNavProps, fullMetadata, selectedProps)
}

func writeODataCollectionResponse(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, selectedProps []string) error {
	if IsAtomFormat(r) {
		return WriteAtomCollection(w, r, entitySetName, data, count, nextLink, deltaLink, nil)
	}

	if !IsAcceptableFormat(r) {
		return WriteError(w, r, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json and application/atom+xml are supported for data responses.")
	}

	metadataLevel := GetODataMetadataLevel(r)

	contextURL := ""
	if metadataLevel != "none" {
		contextURL = buildContextURLWithSelect(r, entitySetName, selectedProps)
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

	if r.Method == http.MethodHead {
		jsonBytes, err := json.Marshal(response)
		if err != nil {
			return WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to marshal response to JSON")
		}
		w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))
		w.WriteHeader(http.StatusOK)
		return nil
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

func writeODataCollectionWithNavigationResponse(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, fullMetadata *metadata.EntityMetadata, selectedProps []string) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, r, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json and application/atom+xml are supported for data responses.")
	}

	if IsAtomFormat(r) {
		// For Atom format, transform data with minimal metadata to populate @odata.id per entry.
		transformedData := addNavigationLinks(data, metadata, expandOptions, selectedNavProps, r, entitySetName, MetadataMinimal, fullMetadata)
		if transformedData == nil {
			transformedData = []interface{}{}
		}
		keyProps := metadata.GetKeyProperties()
		err := WriteAtomCollection(w, r, entitySetName, transformedData, count, nextLink, deltaLink, keyProps)
		releaseOrderedMaps(transformedData)
		return err
	}

	metadataLevel := GetODataMetadataLevel(r)

	contextURL := ""
	if metadataLevel != "none" {
		contextURL = buildContextURLWithSelect(r, entitySetName, selectedProps)
	}

	transformedData := addNavigationLinks(data, metadata, expandOptions, selectedNavProps, r, entitySetName, metadataLevel, fullMetadata)
	if transformedData == nil {
		transformedData = []interface{}{}
	}

	// Add @odata.index annotations if $index query parameter is present
	if shouldAddIndexAnnotations(r) {
		transformedData = addIndexAnnotations(transformedData)
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

	if r.Method == http.MethodHead {
		jsonBytes, err := json.Marshal(response)
		if err != nil {
			return WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to serialize response to JSON.")
		}
		w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))
		w.WriteHeader(http.StatusOK)
		releaseOrderedMaps(transformedData)
		return nil
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(response)
	// Always release the OrderedMap objects regardless of encoding outcome.
	// MarshalJSON is called during Encode and produces its own copy of the bytes, so the
	// maps are safe to return to the pool even when a subsequent write-to-network error occurs.
	releaseOrderedMaps(transformedData)
	return err
}

// WriteODataDeltaResponse writes an OData delta response containing change tracking entries.
func WriteODataDeltaResponse(w http.ResponseWriter, r *http.Request, entitySetName string, entries []map[string]interface{}, deltaLink *string) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, r, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json and application/atom+xml are supported for data responses.")
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

	if r.Method == http.MethodHead {
		jsonBytes, err := json.Marshal(response)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))
		w.WriteHeader(http.StatusOK)
		return nil
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

// shouldAddIndexAnnotations checks if the $index query parameter is present in the request
func shouldAddIndexAnnotations(r *http.Request) bool {
	_, exists := r.URL.Query()["$index"]
	return exists
}

// addIndexAnnotations adds @odata.index annotations to collection items
// The index represents the zero-based ordinal position of each item in the collection
func addIndexAnnotations(data []interface{}) []interface{} {
	for i, item := range data {
		switch value := item.(type) {
		case map[string]interface{}:
			value["@odata.index"] = i
		case *OrderedMap:
			value.Set("@odata.index", i)
		}
	}
	return data
}

// releaseOrderedMaps returns any *OrderedMap elements in data back to the pool.
// This must only be called after the data has been fully serialized to JSON.
func releaseOrderedMaps(data []interface{}) {
	for _, item := range data {
		if om, ok := item.(*OrderedMap); ok {
			om.Release()
		}
	}
}
