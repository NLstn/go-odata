package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/response"
)

// handleQueryBody implements OData 4.01 Part 2 §4.17. It moves the
// percent-encoded query options from a text/plain POST body into the request URL
// and then lets normal entity-set routing and query execution handle them.
func (r *Router) handleQueryBody(w http.ResponseWriter, req *http.Request) bool {
	path := strings.TrimPrefix(req.URL.Path, "/")
	if !strings.HasSuffix(path, "/$query") {
		return false
	}

	if req.Method != http.MethodPost {
		if err := response.WriteMethodNotAllowed(w, req, "POST, OPTIONS", "Method not allowed",
			"resource paths ending in /$query require POST"); err != nil {
			r.logger.Error("Error writing error response", "error", err)
		}
		return true
	}

	mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "text/plain") {
		if err := response.WriteError(w, req, http.StatusUnsupportedMediaType, "Unsupported Media Type",
			"the /$query request body must use Content-Type text/plain"); err != nil {
			r.logger.Error("Error writing error response", "error", err)
		}
		return true
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		if writeErr := response.WriteError(w, req, http.StatusBadRequest, "Invalid query options", err.Error()); writeErr != nil {
			r.logger.Error("Error writing error response", "error", writeErr)
		}
		return true
	}
	bodyText := string(body)
	if bodyText == "" || strings.ContainsAny(bodyText, " \t\r\n") {
		if err := response.WriteError(w, req, http.StatusBadRequest, "Invalid query options",
			"the /$query request body must contain percent-encoded query options without whitespace"); err != nil {
			r.logger.Error("Error writing error response", "error", err)
		}
		return true
	}

	req.URL.Path = strings.TrimSuffix(req.URL.Path, "/$query")
	if req.URL.RawQuery == "" {
		req.URL.RawQuery = bodyText
	} else {
		req.URL.RawQuery += "&" + bodyText
	}
	req.Body = io.NopCloser(bytes.NewReader(nil))
	// The query body has been consumed; keep the downstream collection handler from
	// treating this request as a JSON data-modification payload.
	req.Header.Set("Content-Type", "application/json")
	// /$query is a read operation even though the protocol uses POST for its
	// query-options body; dispatch the consumed request through collection GET logic.
	req.Method = http.MethodGet
	return false
}

// handleAllResource serves the symbolic service-root $all collection.
func (r *Router) handleAllResource(w http.ResponseWriter, req *http.Request) {
	if r.entitySetNames == nil {
		if err := response.WriteError(w, req, http.StatusNotImplemented, "Not Implemented", "$all is not configured"); err != nil {
			r.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	var items []json.RawMessage
	for _, name := range r.entitySetNames() {
		handler, ok := r.resolveHandler(name)
		if !ok {
			continue
		}
		child := req.Clone(req.Context())
		child.URL.Path = "/" + name
		child.URL.RawQuery = withoutPagingOptions(req.URL.RawQuery)
		recorder := httptest.NewRecorder()
		if handler.IsSingleton() {
			handler.HandleSingleton(recorder, child)
		} else {
			handler.HandleCollection(recorder, child)
		}
		if recorder.Code >= http.StatusBadRequest {
			continue
		}

		var document map[string]json.RawMessage
		if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
			continue
		}
		if value, ok := document["value"]; ok {
			var collection []json.RawMessage
			if json.Unmarshal(value, &collection) == nil {
				items = append(items, collection...)
			}
		} else if handler.IsSingleton() {
			if raw, err := json.Marshal(document); err == nil {
				items = append(items, raw)
			}
		}
	}

	items = applyPaging(items, req.URL.Query())
	writeVirtualCollection(w, req, items)
}

// handleCrossJoin serves the service-root $crossjoin(E1,E2,...) collection.
func (r *Router) handleCrossJoin(w http.ResponseWriter, req *http.Request, path string) {
	open := strings.Index(path, "(")
	if open < 0 || !strings.HasSuffix(path, ")") {
		if err := response.WriteError(w, req, http.StatusBadRequest, "Invalid $crossjoin",
			"entity sets must be supplied in parentheses"); err != nil {
			r.logger.Error("Error writing error response", "error", err)
		}
		return
	}
	sets := strings.Split(path[open+1:len(path)-1], ",")
	if len(sets) < 2 {
		if err := response.WriteError(w, req, http.StatusBadRequest, "Invalid $crossjoin",
			"at least two entity sets are required"); err != nil {
			r.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	collections := make([][]json.RawMessage, len(sets))
	for i, set := range sets {
		set = strings.TrimSpace(set)
		handler, ok := r.resolveHandler(set)
		if !ok || handler.IsSingleton() {
			if err := response.WriteError(w, req, http.StatusNotFound, "Entity set not found",
				fmt.Sprintf("entity set %q is not registered", set)); err != nil {
				r.logger.Error("Error writing error response", "error", err)
			}
			return
		}
		child := req.Clone(req.Context())
		child.URL.Path = "/" + set
		child.URL.RawQuery = withoutPagingOptions(req.URL.RawQuery)
		recorder := httptest.NewRecorder()
		handler.HandleCollection(recorder, child)
		if recorder.Code >= http.StatusBadRequest {
			if err := response.WriteError(w, req, recorder.Code, "Unable to read entity set",
				fmt.Sprintf("entity set %q returned status %d", set, recorder.Code)); err != nil {
				r.logger.Error("Error writing error response", "error", err)
			}
			return
		}

		var document map[string]json.RawMessage
		if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
			if writeErr := response.WriteError(w, req, http.StatusInternalServerError, "Invalid entity response", err.Error()); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
		if err := json.Unmarshal(document["value"], &collections[i]); err != nil {
			if writeErr := response.WriteError(w, req, http.StatusInternalServerError, "Invalid entity response", err.Error()); writeErr != nil {
				r.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
	}

	rows := []json.RawMessage{{}}
	for i, set := range sets {
		set = strings.TrimSpace(set)
		next := make([]json.RawMessage, 0)
		for _, prefix := range rows {
			var base map[string]json.RawMessage
			if err := json.Unmarshal(prefix, &base); err != nil {
				base = make(map[string]json.RawMessage)
			}
			for _, entity := range collections[i] {
				var value map[string]json.RawMessage
				if err := json.Unmarshal(entity, &value); err != nil {
					continue
				}
				id := value["@odata.id"]
				if len(id) == 0 {
					if rawID, ok := value["ID"]; ok {
						id = json.RawMessage(strconv.Quote("/" + set + "(" + string(rawID) + ")"))
					}
				}
				if len(id) == 0 {
					continue
				}
				row := make(map[string]json.RawMessage, len(base)+1)
				for key, raw := range base {
					row[key] = raw
				}
				row[set+"@odata.id"] = id
				encoded, err := json.Marshal(row)
				if err != nil {
					continue
				}
				next = append(next, encoded)
			}
		}
		rows = next
	}
	rows = applyPaging(rows, req.URL.Query())
	writeVirtualCollection(w, req, rows)
}

func withoutPagingOptions(rawQuery string) string {
	values := strings.Split(rawQuery, "&")
	kept := values[:0]
	for _, value := range values {
		name := value
		if idx := strings.IndexByte(value, '='); idx >= 0 {
			name = value[:idx]
		}
		if name != "$top" && name != "$skip" {
			kept = append(kept, value)
		}
	}
	return strings.Join(kept, "&")
}

func applyPaging(items []json.RawMessage, values url.Values) []json.RawMessage {
	skip, skipErr := strconv.Atoi(values.Get("$skip"))
	if skipErr != nil {
		skip = 0
	}
	top, topErr := strconv.Atoi(values.Get("$top"))
	topSet := topErr == nil && values.Get("$top") != ""
	if skip < 0 {
		skip = 0
	}
	if skip >= len(items) {
		return []json.RawMessage{}
	}
	items = items[skip:]
	if topSet && top >= 0 && top < len(items) {
		items = items[:top]
	}
	return items
}

func writeVirtualCollection(w http.ResponseWriter, req *http.Request, items []json.RawMessage) {
	w.Header().Set("Content-Type", "application/json")
	if req.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	payload := map[string]interface{}{
		"@odata.context": "$metadata#Collection(Edm.EntityType)",
		"value":         items,
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return
	}
}
