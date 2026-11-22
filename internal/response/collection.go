package response

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/version"
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
	return writeODataCollectionWithNavigationResponse(w, r, entitySetName, data, count, nextLink, nil, metadata, expandOptions, selectedNavProps, fullMetadata, nil, 0)
}

// WriteODataCollectionWithNavigationAndDelta writes an OData collection response with navigation links and a delta link.
func WriteODataCollectionWithNavigationAndDelta(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, fullMetadata *metadata.EntityMetadata) error {
	return writeODataCollectionWithNavigationResponse(w, r, entitySetName, data, count, nextLink, deltaLink, metadata, expandOptions, selectedNavProps, fullMetadata, nil, 0)
}

// WriteODataCollectionWithNavigationAndSelect writes an OData collection response with navigation links
// and a pre-computed list of selected/output properties used to build the @odata.context URL.
// selectedProps should contain the $select properties (or $apply output properties) so that the
// context URL is shaped as #EntitySet(prop1,prop2) per the OData spec.
// skip is the $skip offset applied to the underlying query, used so that @odata.index annotations
// (when $index is requested) reflect the item's absolute position in the full collection rather
// than its position within the current page.
func WriteODataCollectionWithNavigationAndSelect(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, md EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, fullMetadata *metadata.EntityMetadata, selectedProps []string, skip int) error {
	return writeODataCollectionWithNavigationResponse(w, r, entitySetName, data, count, nextLink, deltaLink, md, expandOptions, selectedNavProps, fullMetadata, selectedProps, skip)
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
		response["@odata.count"] = countValue(r, *count)
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
		w.Header().Set("Content-Type", BuildJSONContentType(r))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))
		w.WriteHeader(http.StatusOK)
		return nil
	}

	w.Header().Set("Content-Type", BuildJSONContentType(r))
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

func writeODataCollectionWithNavigationResponse(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, fullMetadata *metadata.EntityMetadata, selectedProps []string, skip int) error {
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

	// Fast path: when the collection is a slice of entity structs, serialize each
	// entity straight into the response buffer from a cached field plan, skipping
	// the per-entity OrderedMap/map intermediate. Expanded navigation properties are
	// handled inline. Requests that need per-item OrderedMap rewriting (Prefer:
	// omit-values, $index) keep the slow path.
	if fastSlice, ok := canFastWriteCollection(data, fullMetadata); ok {
		pref := preference.ParsePrefer(r)
		if pref.OmitValues == nil && !shouldAddIndexAnnotations(r) {
			var annotationFilter *string
			if pref.IncludeAnnotations != nil {
				annotationFilter = pref.IncludeAnnotations
			}
			ctx := &fastEntityContext{
				baseURL:           buildBaseURL(r),
				entitySetName:     entitySetName,
				metadataLevel:     metadataLevel,
				ieee754Compatible: GetIEEE754Compatible(r),
				metadata:          metadata,
				fullMetadata:      fullMetadata,
				selectedNavProps:  selectedNavProps,
				expandOptions:     expandOptions,
				annotationFilter:  annotationFilter,
				selectedSet:       buildSelectedSet(selectedProps),
				keySet:            buildKeySet(metadata),
			}
			return writeFastCollectionToResponse(w, r, fastSlice, ctx, contextURL, count, nextLink, deltaLink)
		}
	}

	transformedData := addNavigationLinks(data, metadata, expandOptions, selectedNavProps, r, entitySetName, metadataLevel, fullMetadata)
	if transformedData == nil {
		transformedData = []interface{}{}
	}

	// Honor Prefer: omit-values=nulls by removing null-valued properties from each item.
	if pref := preference.ParsePrefer(r); pref.OmitValues != nil {
		pref.ApplyOmitValues(true)
		if pref.OmitsNulls() {
			for _, item := range transformedData {
				OmitNullValues(item)
			}
		}
	}

	// Add @odata.index annotations if $index query parameter is present
	if shouldAddIndexAnnotations(r) {
		transformedData = addIndexAnnotations(transformedData, skip)
	}

	// Build the envelope as an OrderedMap to avoid reflection-based map encoding.
	// OData spec key order: @odata.context, @odata.count, @odata.nextLink, @odata.deltaLink, value.
	envelope := AcquireOrderedMapWithCapacity(5)
	if contextURL != "" {
		envelope.Set("@odata.context", contextURL)
	}
	if count != nil {
		envelope.Set("@odata.count", countValue(r, *count))
	}
	if nextLink != nil && *nextLink != "" {
		envelope.Set("@odata.nextLink", *nextLink)
	}
	if deltaLink != nil && *deltaLink != "" {
		envelope.Set("@odata.deltaLink", *deltaLink)
	}
	envelope.Set("value", transformedData)

	if r.Method == http.MethodHead {
		buf, err := envelope.marshalToPooledBuffer()
		envelope.Release()
		if err != nil {
			releaseOrderedMaps(transformedData)
			return WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to serialize response to JSON.")
		}
		w.Header().Set("Content-Type", BuildJSONContentType(r))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
		w.WriteHeader(http.StatusOK)
		releasePooledBuffer(buf)
		releaseOrderedMaps(transformedData)
		return nil
	}

	// Marshal into a pooled buffer and write its bytes directly, avoiding the copy-out
	// allocation that envelope.MarshalJSON() would otherwise need to satisfy the
	// json.Marshaler contract of returning an owned []byte.
	buf, marshalErr := envelope.marshalToPooledBuffer()
	envelope.Release()
	releaseOrderedMaps(transformedData)
	if marshalErr != nil {
		return marshalErr
	}
	w.Header().Set("Content-Type", BuildJSONContentType(r))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(buf.Bytes())
	releasePooledBuffer(buf)
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

	envelope := NewOrderedMapWithCapacity(3)
	if metadataLevel != "none" {
		envelope.Set("@odata.context", buildDeltaContextURL(r, entitySetName))
	}
	if version.GetVersion(r.Context()).Is402OrLater() {
		envelope.Set("value", orderDeltaEntries(entries))
	} else {
		envelope.Set("value", entries)
	}
	if deltaLink != nil && *deltaLink != "" {
		envelope.Set("@odata.deltaLink", *deltaLink)
	}

	contentType := BuildJSONContentType(r)
	if version.GetVersion(r.Context()).Is402OrLater() && requestsJSONStreaming(r) {
		contentType += ";streaming=true"
	}

	if r.Method == http.MethodHead {
		jsonBytes, err := json.Marshal(envelope)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))
		w.WriteHeader(http.StatusOK)
		return nil
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(envelope)
}

// requestsJSONStreaming recognizes the format parameter for ordered responses.
func requestsJSONStreaming(r *http.Request) bool {
	format, _ := parseQueryNegotiation(r.URL.RawQuery)
	if format == "" {
		format = r.Header.Get("Accept")
	}
	for _, mediaRange := range strings.Split(format, ",") {
		mediaType, params, err := mime.ParseMediaType(mediaRange)
		if err != nil || mediaType != "application/json" {
			continue
		}
		if quality, exists := params["q"]; exists {
			if q, err := strconv.ParseFloat(quality, 64); err != nil || q <= 0 || q > 1 {
				continue
			}
		}
		if strings.EqualFold(params["streaming"], "true") || strings.EqualFold(params["odata.streaming"], "true") {
			return true
		}
	}
	return false
}

// orderDeltaEntries puts 4.02 control information before properties, with
// type immediately following removed for deleted entities.
func orderDeltaEntries(entries []map[string]interface{}) []*OrderedMap {
	ordered := make([]*OrderedMap, 0, len(entries))
	control := [...]string{"@odata.context", "@odata.removed", "@odata.type", "@odata.id", "@odata.etag"}
	for _, entry := range entries {
		value := NewOrderedMapWithCapacity(len(entry))
		for _, key := range control {
			if data, exists := entry[key]; exists {
				value.Set(key, data)
			}
		}
		for key, data := range entry {
			if _, exists := value.values[key]; !exists {
				value.Set(key, data)
			}
		}
		ordered = append(ordered, value)
	}
	return ordered
}

// shouldAddIndexAnnotations checks if the $index query parameter is present in the request
func shouldAddIndexAnnotations(r *http.Request) bool {
	return getNegotiation(r).hasIndex
}

// addIndexAnnotations adds @odata.index annotations to collection items.
// The index represents the zero-based ordinal position of each item in the total collection,
// so skip (the $skip offset of the current page) is added to the item's position within the page.
func addIndexAnnotations(data []interface{}, skip int) []interface{} {
	for i, item := range data {
		index := skip + i
		switch value := item.(type) {
		case map[string]interface{}:
			value["@odata.index"] = index
		case *OrderedMap:
			value.Set("@odata.index", index)
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

// countValue preserves Int64 precision when IEEE754Compatible is negotiated.
func countValue(r *http.Request, count int64) interface{} {
	if GetIEEE754Compatible(r) {
		return strconv.FormatInt(count, 10)
	}
	return count
}
