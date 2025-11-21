package response

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// WriteEntityReference writes an OData entity reference response for a single entity.
func WriteEntityReference(w http.ResponseWriter, r *http.Request, entityID string) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses.")
	}

	baseURL := buildBaseURL(r)
	contextURL := baseURL + "/$metadata#$ref"

	response := map[string]interface{}{
		"@odata.context": contextURL,
		"@odata.id":      baseURL + "/" + entityID,
	}

	metadataLevel := GetODataMetadataLevel(r)
	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	SetODataVersionHeader(w)
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		jsonBytes, err := json.Marshal(response)
		if err == nil {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))
		}
		return nil
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

// WriteEntityReferenceCollection writes an OData entity reference collection response.
func WriteEntityReferenceCollection(w http.ResponseWriter, r *http.Request, entityIDs []string, count *int64, nextLink *string) error {
	if !IsAcceptableFormat(r) {
		return WriteError(w, http.StatusNotAcceptable, "Not Acceptable",
			"The requested format is not supported. Only application/json is supported for data responses.")
	}

	baseURL := buildBaseURL(r)
	contextURL := baseURL + "/$metadata#Collection($ref)"

	refs := make([]map[string]string, len(entityIDs))
	for i, entityID := range entityIDs {
		refs[i] = map[string]string{
			"@odata.id": baseURL + "/" + entityID,
		}
	}

	response := map[string]interface{}{
		"@odata.context": contextURL,
		"value":          refs,
	}

	if count != nil {
		response["@odata.count"] = *count
	}
	if nextLink != nil && *nextLink != "" {
		response["@odata.nextLink"] = *nextLink
	}

	metadataLevel := GetODataMetadataLevel(r)
	w.Header().Set("Content-Type", fmt.Sprintf("application/json;odata.metadata=%s", metadataLevel))
	SetODataVersionHeader(w)
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		jsonBytes, err := json.Marshal(response)
		if err == nil {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonBytes)))
		}
		return nil
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}
