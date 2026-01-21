package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

func entitySetRestrictionDisabled(annotations *metadata.AnnotationCollection, term, field string) bool {
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
	if !entitySetRestrictionDisabled(h.metadata.EntitySetAnnotations, metadata.CapInsertRestrictions, "Insertable") {
		return true
	}

	h.writeMethodNotAllowedError(w, r, "POST", fmt.Sprintf("entity set '%s'", h.metadata.EntitySetName))
	return false
}

func (h *EntityHandler) enforceUpdateRestrictions(w http.ResponseWriter, r *http.Request, method string) bool {
	if !entitySetRestrictionDisabled(h.metadata.EntitySetAnnotations, metadata.CapUpdateRestrictions, "Updatable") {
		return true
	}

	h.writeMethodNotAllowedError(w, r, method, fmt.Sprintf("entity set '%s'", h.metadata.EntitySetName))
	return false
}

func (h *EntityHandler) enforceDeleteRestrictions(w http.ResponseWriter, r *http.Request) bool {
	if !entitySetRestrictionDisabled(h.metadata.EntitySetAnnotations, metadata.CapDeleteRestrictions, "Deletable") {
		return true
	}

	h.writeMethodNotAllowedError(w, r, "DELETE", fmt.Sprintf("entity set '%s'", h.metadata.EntitySetName))
	return false
}
