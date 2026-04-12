package handlers

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/metadata"
)

type metadataOperationProduct struct {
	ID   int `odata:"key"`
	Name string
}

func TestMetadataIncludesOperationElementsXML(t *testing.T) {
	entityMeta, err := metadata.AnalyzeEntity(metadataOperationProduct{})
	if err != nil {
		t.Fatalf("failed to analyze entity: %v", err)
	}

	entities := map[string]*metadata.EntityMetadata{
		entityMeta.EntitySetName: entityMeta,
	}

	functions := map[string][]*actions.FunctionDefinition{
		"GetTopProducts": {
			{
				Name:       "GetTopProducts",
				IsBound:    false,
				Parameters: []actions.ParameterDefinition{{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true}},
				ReturnType: reflect.TypeOf([]metadataOperationProduct{}),
			},
		},
		"GetRelatedProducts": {
			{
				Name:       "GetRelatedProducts",
				IsBound:    true,
				EntitySet:  entityMeta.EntitySetName,
				ReturnType: reflect.TypeOf([]metadataOperationProduct{}),
			},
		},
	}

	actionsMap := map[string][]*actions.ActionDefinition{
		"ResetProducts": {
			{
				Name:    "ResetProducts",
				IsBound: false,
			},
		},
		"ApplyDiscount": {
			{
				Name:       "ApplyDiscount",
				IsBound:    true,
				EntitySet:  entityMeta.EntitySetName,
				Parameters: []actions.ParameterDefinition{{Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true}},
			},
		},
	}

	handler := NewMetadataHandlerWithOperations(entities, actionsMap, functions)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", w.Code)
	}

	body := w.Body.String()
	checks := []string{
		`<Function Name="GetTopProducts" IsBound="false">`,
		`<Function Name="GetRelatedProducts" IsBound="true">`,
		`<Action Name="ResetProducts" IsBound="false">`,
		`<Action Name="ApplyDiscount" IsBound="true">`,
		`<FunctionImport Name="GetTopProducts" Function="ODataService.GetTopProducts" />`,
		`<ActionImport Name="ResetProducts" Action="ODataService.ResetProducts" />`,
		`<ReturnType Type="Collection(ODataService.metadataOperationProduct)" />`,
		`<Parameter Name="bindingParameter" Type="ODataService.metadataOperationProduct" Nullable="false" />`,
	}

	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Fatalf("metadata XML missing expected fragment: %s\nBody:\n%s", want, body)
		}
	}

	if strings.Contains(body, `<FunctionImport Name="GetRelatedProducts"`) {
		t.Fatalf("bound function should not be exposed as FunctionImport")
	}
	if strings.Contains(body, `<ActionImport Name="ApplyDiscount"`) {
		t.Fatalf("bound action should not be exposed as ActionImport")
	}
}

func TestMetadataIncludesOperationElementsJSON(t *testing.T) {
	entityMeta, err := metadata.AnalyzeEntity(metadataOperationProduct{})
	if err != nil {
		t.Fatalf("failed to analyze entity: %v", err)
	}

	entities := map[string]*metadata.EntityMetadata{
		entityMeta.EntitySetName: entityMeta,
	}

	functions := map[string][]*actions.FunctionDefinition{
		"GetTopProducts": {
			{
				Name:       "GetTopProducts",
				IsBound:    false,
				Parameters: []actions.ParameterDefinition{{Name: "count", Type: reflect.TypeOf(int64(0)), Required: true}},
				ReturnType: reflect.TypeOf([]metadataOperationProduct{}),
			},
		},
	}

	actionsMap := map[string][]*actions.ActionDefinition{
		"ResetProducts": {
			{
				Name:    "ResetProducts",
				IsBound: false,
			},
		},
	}

	handler := NewMetadataHandlerWithOperations(entities, actionsMap, functions)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", w.Code)
	}

	body := w.Body.String()
	checks := []string{
		`"$Kind": "Function"`,
		`"$Kind": "Action"`,
		`"$Kind": "FunctionImport"`,
		`"$Kind": "ActionImport"`,
		`"$Function": "ODataService.GetTopProducts"`,
		`"$Action": "ODataService.ResetProducts"`,
	}

	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Fatalf("metadata JSON missing expected fragment: %s\nBody:\n%s", want, body)
		}
	}
}
