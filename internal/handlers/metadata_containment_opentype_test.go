package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// ----- Containment navigation property test entities -----

type ContainedItem struct {
	ID   string `json:"ID" odata:"key"`
	Name string `json:"Name"`
}

type ContainerEntity struct {
	ID    string          `json:"ID" odata:"key"`
	Items []ContainedItem `json:"Items,omitempty" gorm:"foreignKey:ContainerEntityID" odata:"containment"`
}

// ----- Open type test entities -----

type OpenEntity struct {
	ID   string `json:"ID" odata:"key"`
	Name string `json:"Name"`
}

func (OpenEntity) IsOpenType() bool {
	return true
}

// ClosedEntity has no IsOpenType method (not an open type)
type ClosedEntity struct {
	ID   string `json:"ID" odata:"key"`
	Name string `json:"Name"`
}

// TestContainmentNavigationProperty_XML verifies that ContainsTarget="true" is emitted
// in XML metadata for navigation properties tagged with odata:"containment".
func TestContainmentNavigationProperty_XML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	containerMeta, err := metadata.AnalyzeEntity(ContainerEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze ContainerEntity: %v", err)
	}
	entities[containerMeta.EntitySetName] = containerMeta

	containedMeta, err := metadata.AnalyzeEntity(ContainedItem{})
	if err != nil {
		t.Fatalf("Failed to analyze ContainedItem: %v", err)
	}
	entities[containedMeta.EntitySetName] = containedMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// ContainsTarget="true" must appear for the Items navigation property
	if !strings.Contains(body, `ContainsTarget="true"`) {
		t.Errorf("XML metadata should contain ContainsTarget=\"true\" for containment nav property.\nBody:\n%s", body)
	}
}

// TestContainmentNavigationProperty_JSON verifies that "$ContainsTarget": true is emitted
// in JSON metadata for navigation properties tagged with odata:"containment".
func TestContainmentNavigationProperty_JSON(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	containerMeta, err := metadata.AnalyzeEntity(ContainerEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze ContainerEntity: %v", err)
	}
	entities[containerMeta.EntitySetName] = containerMeta

	containedMeta, err := metadata.AnalyzeEntity(ContainedItem{})
	if err != nil {
		t.Fatalf("Failed to analyze ContainedItem: %v", err)
	}
	entities[containedMeta.EntitySetName] = containedMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON metadata: %v\nBody: %s", err, w.Body.String())
	}

	// Navigate to ContainerEntity type
	odataService, ok := result["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected ODataService in JSON metadata")
	}

	containerEntityType, ok := odataService["ContainerEntity"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected ContainerEntity in ODataService. Keys: %v", keys(odataService))
	}

	itemsNavProp, ok := containerEntityType["Items"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected Items navigation property in ContainerEntity. Keys: %v", keys(containerEntityType))
	}

	containsTarget, ok := itemsNavProp["$ContainsTarget"].(bool)
	if !ok || !containsTarget {
		t.Errorf("Expected $ContainsTarget=true in Items navigation property, got: %v", itemsNavProp["$ContainsTarget"])
	}
}

// TestNonContainmentNavigationProperty_XML verifies that ContainsTarget is NOT emitted
// for regular (non-containment) navigation properties.
func TestNonContainmentNavigationProperty_XML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	memberMeta, err := metadata.AnalyzeEntity(Member{})
	if err != nil {
		t.Fatalf("Failed to analyze Member: %v", err)
	}
	entities[memberMeta.EntitySetName] = memberMeta

	privacyMeta, err := metadata.AnalyzeEntity(MemberPrivacySettings{})
	if err != nil {
		t.Fatalf("Failed to analyze MemberPrivacySettings: %v", err)
	}
	entities[privacyMeta.EntitySetName] = privacyMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	body := w.Body.String()

	if strings.Contains(body, `ContainsTarget="true"`) {
		t.Errorf("XML metadata should NOT contain ContainsTarget=\"true\" for non-containment nav property.\nBody:\n%s", body)
	}
}

// TestOpenType_XML verifies that OpenType="true" is emitted in XML metadata
// for entity types that implement IsOpenType() bool returning true.
func TestOpenType_XML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	openMeta, err := metadata.AnalyzeEntity(OpenEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze OpenEntity: %v", err)
	}
	entities[openMeta.EntitySetName] = openMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	if !strings.Contains(body, `OpenType="true"`) {
		t.Errorf("XML metadata should contain OpenType=\"true\" for open type entity.\nBody:\n%s", body)
	}
}

// TestOpenType_JSON verifies that "$OpenType": true is emitted in JSON metadata
// for entity types that implement IsOpenType() bool returning true.
func TestOpenType_JSON(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	openMeta, err := metadata.AnalyzeEntity(OpenEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze OpenEntity: %v", err)
	}
	entities[openMeta.EntitySetName] = openMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON metadata: %v\nBody: %s", err, w.Body.String())
	}

	odataService, ok := result["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected ODataService in JSON metadata")
	}

	openEntityType, ok := odataService["OpenEntity"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected OpenEntity in ODataService. Keys: %v", keys(odataService))
	}

	openType, ok := openEntityType["$OpenType"].(bool)
	if !ok || !openType {
		t.Errorf("Expected $OpenType=true in OpenEntity, got: %v", openEntityType["$OpenType"])
	}
}

// TestClosedType_XML verifies that OpenType is NOT emitted for regular (closed) entity types.
func TestClosedType_XML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	closedMeta, err := metadata.AnalyzeEntity(ClosedEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze ClosedEntity: %v", err)
	}
	entities[closedMeta.EntitySetName] = closedMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	body := w.Body.String()

	if strings.Contains(body, `OpenType="true"`) {
		t.Errorf("XML metadata should NOT contain OpenType=\"true\" for a closed entity type.\nBody:\n%s", body)
	}
}

// keys returns map keys as a slice (helper for test error messages)
func keys(m map[string]interface{}) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
