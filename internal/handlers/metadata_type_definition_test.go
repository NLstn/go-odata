package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// typeDefWeight is a named float64 type representing a weight value (→ Edm.Double)
type typeDefWeight float64

// typeDefScore is a named int32 type representing a score (→ Edm.Int32)
type typeDefScore int32

// typeDefLabel is a named string type representing a short label (→ Edm.String)
type typeDefLabel string

// typeDefEntity is a test entity with TypeDefinition properties
type typeDefEntity struct {
	ID     int           `json:"ID" odata:"key"`
	Weight typeDefWeight `json:"Weight"`
	Score  typeDefScore  `json:"Score"`
	Label  typeDefLabel  `json:"Label"`
}

func setupTypeDefMetadataEntities(t *testing.T) map[string]*metadata.EntityMetadata {
	t.Helper()

	// Register TypeDefinitions
	err := metadata.RegisterTypeDefinition(reflect.TypeOf(typeDefWeight(0)), metadata.TypeDefinitionInfo{
		Name:           "Weight",
		UnderlyingType: "Edm.Double",
	})
	if err != nil {
		t.Fatalf("Failed to register Weight TypeDefinition: %v", err)
	}

	err = metadata.RegisterTypeDefinition(reflect.TypeOf(typeDefScore(0)), metadata.TypeDefinitionInfo{
		Name:           "Score",
		UnderlyingType: "Edm.Int32",
	})
	if err != nil {
		t.Fatalf("Failed to register Score TypeDefinition: %v", err)
	}

	err = metadata.RegisterTypeDefinition(reflect.TypeOf(typeDefLabel("")), metadata.TypeDefinitionInfo{
		Name:           "Label",
		UnderlyingType: "Edm.String",
		MaxLength:      50,
	})
	if err != nil {
		t.Fatalf("Failed to register Label TypeDefinition: %v", err)
	}

	// Register a Decimal-based TypeDefinition by directly providing the underlying type
	type typeDefDecimalAmount float64 // underlying will be inferred as Edm.Double, so we set it manually
	_ = typeDefDecimalAmount(0)

	entities := make(map[string]*metadata.EntityMetadata)
	entityMeta, err := metadata.AnalyzeEntity(typeDefEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze typeDefEntity: %v", err)
	}
	entities["typeDefEntity"] = entityMeta

	return entities
}

func TestTypeDefinitionXMLMetadata(t *testing.T) {
	entities := setupTypeDefMetadataEntities(t)
	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// TypeDefinition elements should be present
	if !strings.Contains(body, `<TypeDefinition Name="Weight" UnderlyingType="Edm.Double"`) {
		t.Errorf("Expected Weight TypeDefinition in XML, got:\n%s", body)
	}
	if !strings.Contains(body, `<TypeDefinition Name="Score" UnderlyingType="Edm.Int32"`) {
		t.Errorf("Expected Score TypeDefinition in XML, got:\n%s", body)
	}
	if !strings.Contains(body, `<TypeDefinition Name="Label" UnderlyingType="Edm.String"`) {
		t.Errorf("Expected Label TypeDefinition in XML, got:\n%s", body)
	}
	if !strings.Contains(body, `MaxLength="50"`) {
		t.Errorf("Expected MaxLength facet on Label TypeDefinition")
	}

	// Properties should reference the TypeDefinition types
	if !strings.Contains(body, `Type="ODataService.Weight"`) {
		t.Errorf("Expected Weight property to reference TypeDefinition type")
	}
	if !strings.Contains(body, `Type="ODataService.Score"`) {
		t.Errorf("Expected Score property to reference TypeDefinition type")
	}
	if !strings.Contains(body, `Type="ODataService.Label"`) {
		t.Errorf("Expected Label property to reference TypeDefinition type")
	}
}

func TestTypeDefinitionXMLMetadataDecimalFacets(t *testing.T) {
	type typeDefDecimalPrice float64
	err := metadata.RegisterTypeDefinition(reflect.TypeOf(typeDefDecimalPrice(0)), metadata.TypeDefinitionInfo{
		Name:           "Price",
		UnderlyingType: "Edm.Decimal",
		Precision:      10,
		Scale:          2,
	})
	if err != nil {
		t.Fatalf("Failed to register Price TypeDefinition: %v", err)
	}

	type productWithPrice struct {
		ID    int                 `json:"ID" odata:"key"`
		Price typeDefDecimalPrice `json:"Price"`
	}

	entities := make(map[string]*metadata.EntityMetadata)
	entityMeta, err := metadata.AnalyzeEntity(productWithPrice{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	entities["productWithPrice"] = entityMeta

	handler := NewMetadataHandler(entities)
	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `<TypeDefinition Name="Price" UnderlyingType="Edm.Decimal"`) {
		t.Errorf("Expected Price TypeDefinition with Edm.Decimal, got:\n%s", body)
	}
	if !strings.Contains(body, `Precision="10"`) {
		t.Errorf("Expected Precision=10 on Price TypeDefinition")
	}
	if !strings.Contains(body, `Scale="2"`) {
		t.Errorf("Expected Scale=2 on Price TypeDefinition")
	}
	if !strings.Contains(body, `Type="ODataService.Price"`) {
		t.Errorf("Expected Price property to reference TypeDefinition type")
	}
}

func TestTypeDefinitionJSONMetadata(t *testing.T) {
	entities := setupTypeDefMetadataEntities(t)
	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var csdl map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &csdl); err != nil {
		t.Fatalf("Failed to parse JSON metadata: %v", err)
	}

	odataService, ok := csdl["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected ODataService namespace")
	}

	// Weight TypeDefinition
	weightDef, ok := odataService["Weight"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Weight TypeDefinition in JSON metadata")
	}
	if kind := weightDef["$Kind"]; kind != "TypeDefinition" {
		t.Errorf("Expected Weight.$Kind=TypeDefinition, got %v", kind)
	}
	if ut := weightDef["$UnderlyingType"]; ut != "Edm.Double" {
		t.Errorf("Expected Weight.$UnderlyingType=Edm.Double, got %v", ut)
	}

	// Label TypeDefinition with MaxLength
	labelDef, ok := odataService["Label"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Label TypeDefinition in JSON metadata")
	}
	if kind := labelDef["$Kind"]; kind != "TypeDefinition" {
		t.Errorf("Expected Label.$Kind=TypeDefinition, got %v", kind)
	}
	if ml := labelDef["$MaxLength"]; ml != float64(50) {
		t.Errorf("Expected Label.$MaxLength=50, got %v", ml)
	}

	// Entity Weight property should reference TypeDefinition
	entityType, ok := odataService["typeDefEntity"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected typeDefEntity in JSON metadata")
	}
	weightProp, ok := entityType["Weight"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Weight property in typeDefEntity")
	}
	if wt := weightProp["$Type"]; wt != "ODataService.Weight" {
		t.Errorf("Expected Weight property $Type=ODataService.Weight, got %v", wt)
	}
}
