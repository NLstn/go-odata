package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
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
func WriteODataCollectionWithNavigation(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink *string, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, fullMetadata *metadata.EntityMetadata) error {
	return writeODataCollectionWithNavigationResponse(w, r, entitySetName, data, count, nextLink, nil, metadata, expandOptions, selectedNavProps, fullMetadata)
}

// WriteODataCollectionWithNavigationAndDelta writes an OData collection response with navigation links and a delta link.
func WriteODataCollectionWithNavigationAndDelta(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, fullMetadata *metadata.EntityMetadata) error {
	return writeODataCollectionWithNavigationResponse(w, r, entitySetName, data, count, nextLink, deltaLink, metadata, expandOptions, selectedNavProps, fullMetadata)
}

func writeODataCollectionResponse(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, r, http.StatusNotAcceptable, "Not Acceptable",
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

func writeODataCollectionWithNavigationResponse(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, metadata EntityMetadataProvider, expandOptions []query.ExpandOption, selectedNavProps []string, fullMetadata *metadata.EntityMetadata) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, r, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses.")
	}

	metadataLevel := GetODataMetadataLevel(r)

	contextURL := ""
	if metadataLevel != "none" {
		contextURL = buildContextURL(r, entitySetName)
	}

	transformedData := addNavigationLinks(data, metadata, expandOptions, selectedNavProps, r, entitySetName, metadataLevel, fullMetadata)
	if transformedData == nil {
		transformedData = []interface{}{}
	}

	// Add @odata.index annotations if $index query parameter is present
	if shouldAddIndexAnnotations(r) {
		transformedData = addIndexAnnotations(transformedData)
	}

	// Build the entire JSON response into a single pooled buffer.
	// This allows us to:
	//   1. Set the Content-Length header (enables HTTP/1.1 connection reuse)
	//   2. Stream entity JSON via writeJSONTo without per-entity []byte copies
	//   3. Return pooled OrderedMaps back to the pool immediately after encoding
	buf, err := buildCollectionResponseBuffer(contextURL, transformedData, count, nextLink, deltaLink)
	if err != nil {
		return WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "Failed to serialize response to JSON.")
	}
	defer func() {
		if buf.Cap() < 65536 {
			bufferPool.Put(buf)
		}
	}()

	contentType := fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(buf.Bytes())
	return err
}

// buildCollectionResponseBuffer serialises a collection response into a single pooled
// bytes.Buffer. Entities that are *OrderedMap instances are written via writeJSONTo to
// avoid per-entity []byte allocations; after being written they are returned to the pool.
// The caller is responsible for returning the buffer to bufferPool when done.
func buildCollectionResponseBuffer(contextURL string, transformedData []interface{}, count *int64, nextLink, deltaLink *string) (*bytes.Buffer, error) {
	// Estimate capacity: ~50 bytes overhead + ~300 bytes per entity (typical)
	estimated := 64 + len(transformedData)*300
	buf := bufferPool.Get().(*bytes.Buffer) //nolint:errcheck // sync.Pool.Get() doesn't return error
	buf.Reset()
	buf.Grow(estimated)

	buf.WriteByte('{')
	needsComma := false

	if contextURL != "" {
		buf.WriteString(`"@odata.context":`)
		writeJSONStringToBuffer(buf, contextURL)
		needsComma = true
	}
	if count != nil {
		if needsComma {
			buf.WriteByte(',')
		}
		buf.WriteString(`"@odata.count":`)
		writeInt(buf, *count)
		needsComma = true
	}
	if needsComma {
		buf.WriteByte(',')
	}
	buf.WriteString(`"value":[`)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	for i, item := range transformedData {
		if i > 0 {
			buf.WriteByte(',')
		}
		switch v := item.(type) {
		case *OrderedMap:
			if err := v.writeJSONTo(buf); err != nil {
				return buf, err
			}
			// Return the ordered map to pool now that its JSON is in buf.
			v.Release()
		default:
			if err := enc.Encode(item); err != nil {
				return buf, err
			}
			// Remove trailing newline added by json.Encoder.Encode
			buf.Truncate(buf.Len() - 1)
		}
	}
	buf.WriteByte(']')
	if nextLink != nil && *nextLink != "" {
		buf.WriteString(`,"@odata.nextLink":`)
		writeJSONStringToBuffer(buf, *nextLink)
	}
	if deltaLink != nil && *deltaLink != "" {
		buf.WriteString(`,"@odata.deltaLink":`)
		writeJSONStringToBuffer(buf, *deltaLink)
	}
	buf.WriteByte('}')
	return buf, nil
}

// writeJSONStringToBuffer writes a JSON-encoded string value (with surrounding quotes) into buf.
// For strings without special characters it uses a fast direct path.
func writeJSONStringToBuffer(buf *bytes.Buffer, s string) {
	if needsEscaping(s) {
		b, _ := json.Marshal(s) //nolint:errcheck // json.Marshal of a string never errors
		buf.Write(b)
		return
	}
	buf.WriteByte('"')
	buf.WriteString(s)
	buf.WriteByte('"')
}

// WriteODataDeltaResponse writes an OData delta response containing change tracking entries.
func WriteODataDeltaResponse(w http.ResponseWriter, r *http.Request, entitySetName string, entries []map[string]interface{}, deltaLink *string) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, r, http.StatusNotAcceptable, "Not Acceptable",
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
	_, exists := query.GetOrParseParsedQuery(r.Context(), r.URL.RawQuery)["$index"]
	return exists
}

// addIndexAnnotations adds @odata.index annotations to collection items
// The index represents the zero-based ordinal position of each item in the collection
func addIndexAnnotations(data []interface{}) []interface{} {
	for i, item := range data {
		// Only add index to map items (structs are already converted to maps by this point)
		if itemMap, ok := item.(map[string]interface{}); ok {
			itemMap["@odata.index"] = i
		}
	}
	return data
}
