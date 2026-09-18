package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
)

func isOperationProhibited(annotations *metadata.AnnotationCollection, term, field string) bool {
	if annotations == nil {
		return false
	}

	for _, annotation := range annotations.GetByTerm(term) {
		record, ok := annotation.Value.(map[string]interface{})
		if !ok {
			continue
		}
		rawValue, ok := record[field]
		if !ok {
			continue
		}
		if boolValue, ok := boolFromAnnotationValue(rawValue); ok {
			return !boolValue
		}
	}

	return false
}

func boolFromAnnotationValue(value interface{}) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed := strings.EqualFold(typed, "true")
		if strings.EqualFold(typed, "true") || strings.EqualFold(typed, "false") {
			return parsed, true
		}
	}
	return false, false
}

func (h *EntityHandler) enforceInsertRestrictions(w http.ResponseWriter, r *http.Request) bool {
	if isOperationProhibited(h.metadata.EntitySetAnnotations, metadata.CapInsertRestrictions, "Insertable") {
		h.writeMethodNotAllowedError(w, r, "POST", fmt.Sprintf("entity set '%s'", h.metadata.EntitySetName), "GET, HEAD, OPTIONS")
		return false
	}

	return true
}

func (h *EntityHandler) enforceUpdateRestrictions(w http.ResponseWriter, r *http.Request, method string) bool {
	if isOperationProhibited(h.metadata.EntitySetAnnotations, metadata.CapUpdateRestrictions, "Updatable") {
		h.writeMethodNotAllowedError(w, r, method, fmt.Sprintf("entity set '%s'", h.metadata.EntitySetName), "GET, HEAD, DELETE, OPTIONS")
		return false
	}

	return true
}

func (h *EntityHandler) enforceDeleteRestrictions(w http.ResponseWriter, r *http.Request) bool {
	if isOperationProhibited(h.metadata.EntitySetAnnotations, metadata.CapDeleteRestrictions, "Deletable") {
		h.writeMethodNotAllowedError(w, r, "DELETE", fmt.Sprintf("entity set '%s'", h.metadata.EntitySetName), "GET, HEAD, PUT, PATCH, OPTIONS")
		return false
	}

	return true
}

// queryRestrictionError validates query options against the entity-set
// capability annotations. Capability metadata is a contract with clients: when
// a service advertises an option as unsupported, it must reject that option
// instead of silently executing it.
func (h *EntityHandler) queryRestrictionError(r *http.Request, queryOptions *query.QueryOptions) error {
	if queryOptions == nil {
		return nil
	}
	expandUsed := len(queryOptions.Expand) > 0
	if r != nil {
		expandUsed = expandUsed || query.ParseRawQuery(r.URL.RawQuery).Get("$expand") != ""
	}

	checks := []struct {
		used       bool
		term       string
		field      string
		optionName string
	}{
		{queryOptions.Filter != nil, metadata.CapFilterRestrictions, "Filterable", "$filter"},
		{len(queryOptions.OrderBy) > 0, metadata.CapSortRestrictions, "Sortable", "$orderby"},
		{expandUsed, metadata.CapExpandRestrictions, "Expandable", "$expand"},
		{queryOptions.Count, metadata.CapCountRestrictions, "Countable", "$count"},
		{queryOptions.Search != "", metadata.CapSearchRestrictions, "Searchable", "$search"},
		{len(queryOptions.Select) > 0, metadata.CapSelectSupport, "Supported", "$select"},
	}

	for _, check := range checks {
		if check.used && isOperationProhibited(h.metadata.EntitySetAnnotations, check.term, check.field) {
			return &requestError{
				StatusCode: http.StatusBadRequest,
				ErrorCode:  ErrMsgInvalidQueryOptions,
				Message:    fmt.Sprintf("%s is not supported for entity set '%s'", check.optionName, h.metadata.EntitySetName),
			}
		}
	}

	return nil
}

func (h *EntityHandler) countRestrictionError() error {
	if isOperationProhibited(h.metadata.EntitySetAnnotations, metadata.CapCountRestrictions, "Countable") {
		return &requestError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  ErrMsgInvalidQueryOptions,
			Message:    fmt.Sprintf("$count is not supported for entity set '%s'", h.metadata.EntitySetName),
		}
	}
	return nil
}
