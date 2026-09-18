package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

func TestEntityHandlerQueryRestrictionsRejectAdvertisedOptions(t *testing.T) {
	tests := []struct {
		name  string
		term  string
		field string
		path  string
	}{
		{"filter", metadata.CapFilterRestrictions, "Filterable", "/HandlerTestProducts?$filter=ID%20eq%201"},
		{"orderby", metadata.CapSortRestrictions, "Sortable", "/HandlerTestProducts?$orderby=Name"},
		{"expand", metadata.CapExpandRestrictions, "Expandable", "/HandlerTestProducts?$expand=*"},
		{"count", metadata.CapCountRestrictions, "Countable", "/HandlerTestProducts?$count=true"},
		{"search", metadata.CapSearchRestrictions, "Searchable", "/HandlerTestProducts?$search=test"},
		{"select", metadata.CapSelectSupport, "Supported", "/HandlerTestProducts?$select=Name"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, _ := setupEntityTestHandler(t)
			handler.metadata.EntitySetAnnotations = metadata.NewAnnotationCollection()
			handler.metadata.EntitySetAnnotations.AddTerm(test.term, map[string]interface{}{test.field: false})

			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			res := httptest.NewRecorder()
			handler.HandleCollection(res, req)

			if res.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusBadRequest, res.Body.String())
			}
			var payload map[string]interface{}
			if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
				t.Fatalf("response is not JSON: %v", err)
			}
			if _, ok := payload["error"]; !ok {
				t.Fatalf("response is not an OData error: %s", res.Body.String())
			}
			if !strings.Contains(res.Body.String(), test.name) && !strings.Contains(res.Body.String(), "$"+test.name) {
				t.Logf("error does not include the short option name; body = %s", res.Body.String())
			}
		})
	}
}

func TestEntityHandlerCountSegmentRestrictionRejectsRequest(t *testing.T) {
	handler, _ := setupEntityTestHandler(t)
	handler.metadata.EntitySetAnnotations = metadata.NewAnnotationCollection()
	handler.metadata.EntitySetAnnotations.AddTerm(metadata.CapCountRestrictions, map[string]interface{}{"Countable": false})

	req := httptest.NewRequest(http.MethodGet, "/HandlerTestProducts/$count", nil)
	res := httptest.NewRecorder()
	handler.HandleCount(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusBadRequest, res.Body.String())
	}
}

func TestEntityHandlerQueryRestrictionsAllowUnadvertisedOptions(t *testing.T) {
	handler, db := setupEntityTestHandler(t)
	if err := db.Create(&HandlerTestProduct{ID: 1, Name: "Test", Price: 10}).Error; err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/HandlerTestProducts?$filter=ID%20eq%201&$orderby=Name&$count=true&$select=Name", nil)
	res := httptest.NewRecorder()
	handler.HandleCollection(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}
}
