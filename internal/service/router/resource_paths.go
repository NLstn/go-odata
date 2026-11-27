package router

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/version"
)

const methodQuery = "QUERY"

// handleQueryBody rewrites the OData query-body forms to the equivalent GET.
func (r *Router) handleQueryBody(w http.ResponseWriter, req *http.Request) *http.Request {
	suffix := strings.HasSuffix(req.URL.Path, "/$query")
	if suffix && req.Method != http.MethodPost {
		if err := response.WriteMethodNotAllowed(w, req, "POST, OPTIONS", "Method not allowed",
			"resource paths ending in /$query require POST"); err != nil {
			r.logger.Error("Error writing error response", "error", err)
		}
		return nil
	}

	request402 := version.GetRequestVersion(req.Context()).Is402OrLater()
	if req.Method == methodQuery && !request402 {
		r.writeQueryBodyError(w, req, http.StatusNotImplemented, "QUERY requires an OData 4.02 request")
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || (mediaType != "text/plain" &&
		(!request402 || (mediaType != "application/x-www-form-urlencoded" && mediaType != "application/json"))) {
		r.writeQueryBodyError(w, req, http.StatusUnsupportedMediaType, "unsupported Content-Type for an OData query body")
		return nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		r.writeQueryBodyError(w, req, http.StatusBadRequest, err.Error())
		return nil
	}
	targetPath := strings.TrimSuffix(req.URL.Path, "/$query")
	lastSegment := targetPath[strings.LastIndexByte(targetPath, '/')+1:]
	resolved, _ := r.resolveBoundName(lastSegment)
	hasFunction := len(r.functions[resolved]) > 0

	var rawQuery string
	var bodyParams map[string]json.RawMessage
	if mediaType == "application/json" {
		rawQuery, bodyParams, err = parseJSONQueryBody(body, hasFunction)
	} else {
		rawQuery, err = decodeQueryBody(mediaType, body)
	}
	if err != nil {
		r.writeQueryBodyError(w, req, http.StatusBadRequest, err.Error())
		return nil
	}

	req = req.Clone(req.Context())
	if bodyParams != nil {
		// The public 4.02 resource path omits parentheses. Reuse the
		// existing function router with an internal empty argument list.
		targetPath += "()"
		req = actions.WithQueryBodyParams(req, bodyParams)
	}
	if rawQuery != "" {
		if req.URL.RawQuery != "" {
			req.URL.RawQuery += "&"
		}
		req.URL.RawQuery += rawQuery
	}
	if req.URL.Path != targetPath {
		req.URL.Path = targetPath
		req.URL.RawPath = ""
	}
	req.Body = http.NoBody
	req.ContentLength = 0
	req.Header.Del("Content-Type")
	req.Method = http.MethodGet
	return req
}

func (r *Router) writeQueryBodyError(w http.ResponseWriter, req *http.Request, status int, detail string) {
	if err := response.WriteError(w, req, status, http.StatusText(status), detail); err != nil {
		r.logger.Error("Error writing query body response", "error", err)
	}
}

func decodeQueryBody(mediaType string, body []byte) (string, error) {
	rawQuery := string(body)
	if mediaType == "text/plain" && (rawQuery == "" || strings.ContainsAny(rawQuery, " \t\r\n")) {
		return "", errors.New("text/plain query options must be percent-encoded without whitespace")
	}
	if _, err := url.QueryUnescape(rawQuery); err != nil {
		return "", fmt.Errorf("invalid percent-encoding in query options: %w", err)
	}
	return rawQuery, nil
}

func parseJSONQueryBody(body []byte, hasFunction bool) (string, map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return "", nil, errors.New("a JSON query body must be an object")
	}

	options := url.Values{}
	var params map[string]json.RawMessage
	if hasFunction {
		params = map[string]json.RawMessage{}
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return "", nil, err
		}
		key, ok := token.(string)
		if !ok || key == "" {
			return "", nil, errors.New("a JSON query option must have a name")
		}
		normalized := query.NormalizeQueryParams(url.Values{key: nil})
		isOption := strings.HasPrefix(key, "$") || strings.HasPrefix(key, "@")
		if _, unchanged := normalized[key]; !hasFunction && !unchanged && !strings.HasPrefix(key, "$") {
			return "", nil, fmt.Errorf("JSON system query option %q requires a $ prefix", key)
		}
		if _, exists := params[key]; exists || options.Has(key) {
			return "", nil, fmt.Errorf("JSON query option %q must not appear more than once", key)
		}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return "", nil, err
		}
		if hasFunction && !isOption {
			params[key] = value
			continue
		}
		switch strings.ToLower(key) {
		case "$top", "$skip":
			text := string(value)
			if text == "" || strings.Trim(text, "0123456789") != "" {
				return "", nil, fmt.Errorf("JSON query option %q must be a non-negative integer", key)
			}
			options.Set(key, text)
		default:
			var text *string
			if err := json.Unmarshal(value, &text); err != nil || text == nil {
				return "", nil, fmt.Errorf("JSON query option %q must be a string", key)
			}
			options.Set(key, *text)
		}
	}
	if _, err := decoder.Token(); err != nil {
		return "", nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return "", nil, errors.New("unexpected data after JSON query options")
	}
	return options.Encode(), params, nil
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
	writeVirtualCollection(w, req, items, "Collection(Edm.EntityType)")
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
				row[set+"@odata.navigationLink"] = id
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
	writeVirtualCollection(w, req, rows, "Collection(Edm.ComplexType)")
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

func writeVirtualCollection(w http.ResponseWriter, req *http.Request, items []json.RawMessage, contextType string) {
	w.Header().Set("Content-Type", "application/json")
	if req.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	payload := map[string]interface{}{
		"@odata.context": "$metadata#" + contextType,
		"value":          items,
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return
	}
}
