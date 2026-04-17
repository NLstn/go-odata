package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// ----- Inheritance test entities -----

// VehicleEntity is a base entity type for inheritance tests.
type VehicleEntity struct {
	ID   string `json:"ID" odata:"key"`
	Make string `json:"Make"`
}

// CarEntity is a derived entity type that extends VehicleEntity.
// It uses ODataBaseType() to declare the base type and only contains its own properties.
type CarEntity struct {
	NumDoors int32 `json:"NumDoors"`
}

func (CarEntity) ODataBaseType() string {
	return "TestNamespace.VehicleEntity"
}

// AbstractVehicleEntity is an abstract base entity type.
type AbstractVehicleEntity struct {
	ID   string `json:"ID" odata:"key"`
	Make string `json:"Make"`
}

func (AbstractVehicleEntity) IsAbstract() bool {
	return true
}

// DerivedAbstractEntity is a derived type of an abstract base, implementing both methods.
type DerivedAbstractEntity struct {
	NumDoors int32 `json:"NumDoors"`
}

func (DerivedAbstractEntity) ODataBaseType() string {
	return "TestNamespace.AbstractVehicleEntity"
}

func (DerivedAbstractEntity) IsAbstract() bool {
	return false
}

// ----- XML tests -----

func TestBaseType_XML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	vehicleMeta, err := metadata.AnalyzeEntity(VehicleEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze VehicleEntity: %v", err)
	}
	entities[vehicleMeta.EntitySetName] = vehicleMeta

	carMeta, err := metadata.AnalyzeEntity(CarEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze CarEntity: %v", err)
	}
	entities[carMeta.EntitySetName] = carMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// CarEntity must include BaseType attribute
	if !strings.Contains(body, `BaseType="TestNamespace.VehicleEntity"`) {
		t.Errorf("XML metadata should contain BaseType attribute for CarEntity.\nBody:\n%s", body)
	}

	// CarEntity must NOT include a <Key> element (key is inherited from VehicleEntity)
	// The easiest check: the CarEntity block should not contain <Key>
	// We look for the CarEntity type block and ensure it has no Key section
	carTypeStart := strings.Index(body, `Name="CarEntity"`)
	if carTypeStart == -1 {
		t.Fatalf("CarEntity not found in metadata.\nBody:\n%s", body)
	}
	// Find end of CarEntity block
	carTypeEnd := strings.Index(body[carTypeStart:], "</EntityType>")
	if carTypeEnd == -1 {
		t.Fatalf("Could not find closing </EntityType> for CarEntity.\nBody:\n%s", body)
	}
	carBlock := body[carTypeStart : carTypeStart+carTypeEnd]
	if strings.Contains(carBlock, "<Key>") {
		t.Errorf("Derived type CarEntity should NOT contain a <Key> element.\nCarEntity block:\n%s", carBlock)
	}

	// VehicleEntity must still have its own <Key> element
	vehicleTypeStart := strings.Index(body, `Name="VehicleEntity"`)
	if vehicleTypeStart == -1 {
		t.Fatalf("VehicleEntity not found in metadata.\nBody:\n%s", body)
	}
	vehicleTypeEnd := strings.Index(body[vehicleTypeStart:], "</EntityType>")
	if vehicleTypeEnd == -1 {
		t.Fatalf("Could not find closing </EntityType> for VehicleEntity.\nBody:\n%s", body)
	}
	vehicleBlock := body[vehicleTypeStart : vehicleTypeStart+vehicleTypeEnd]
	if !strings.Contains(vehicleBlock, "<Key>") {
		t.Errorf("Base type VehicleEntity must contain a <Key> element.\nVehicleEntity block:\n%s", vehicleBlock)
	}
}

func TestAbstract_XML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	abstractMeta, err := metadata.AnalyzeEntity(AbstractVehicleEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze AbstractVehicleEntity: %v", err)
	}
	entities[abstractMeta.EntitySetName] = abstractMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	if !strings.Contains(body, `Abstract="true"`) {
		t.Errorf("XML metadata should contain Abstract=\"true\" for AbstractVehicleEntity.\nBody:\n%s", body)
	}
}

func TestDerivedAbstract_XML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	abstractMeta, err := metadata.AnalyzeEntity(AbstractVehicleEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze AbstractVehicleEntity: %v", err)
	}
	entities[abstractMeta.EntitySetName] = abstractMeta

	derivedMeta, err := metadata.AnalyzeEntity(DerivedAbstractEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze DerivedAbstractEntity: %v", err)
	}
	entities[derivedMeta.EntitySetName] = derivedMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// AbstractVehicleEntity must have Abstract="true"
	if !strings.Contains(body, `Abstract="true"`) {
		t.Errorf("XML metadata should contain Abstract=\"true\" for AbstractVehicleEntity.\nBody:\n%s", body)
	}

	// DerivedAbstractEntity must have BaseType set
	if !strings.Contains(body, `BaseType="TestNamespace.AbstractVehicleEntity"`) {
		t.Errorf("XML metadata should contain BaseType for DerivedAbstractEntity.\nBody:\n%s", body)
	}

	// DerivedAbstractEntity returns IsAbstract()=false, so should NOT have Abstract="true"
	// (AbstractVehicleEntity has it, but DerivedAbstractEntity should not)
	derivedStart := strings.Index(body, `Name="DerivedAbstractEntity"`)
	if derivedStart == -1 {
		t.Fatalf("DerivedAbstractEntity not found in metadata.\nBody:\n%s", body)
	}
	derivedEnd := strings.Index(body[derivedStart:], "</EntityType>")
	if derivedEnd == -1 {
		t.Fatalf("Could not find closing </EntityType> for DerivedAbstractEntity.\nBody:\n%s", body)
	}
	derivedBlock := body[derivedStart : derivedStart+derivedEnd]
	if strings.Contains(derivedBlock, `Abstract="true"`) {
		t.Errorf("DerivedAbstractEntity (IsAbstract=false) should NOT have Abstract=\"true\".\nDerived block:\n%s", derivedBlock)
	}
}

// ----- JSON tests -----

func TestBaseType_JSON(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	vehicleMeta, err := metadata.AnalyzeEntity(VehicleEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze VehicleEntity: %v", err)
	}
	entities[vehicleMeta.EntitySetName] = vehicleMeta

	carMeta, err := metadata.AnalyzeEntity(CarEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze CarEntity: %v", err)
	}
	entities[carMeta.EntitySetName] = carMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var jsonDoc map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &jsonDoc); err != nil {
		t.Fatalf("Response is not valid JSON: %v\nBody: %s", err, w.Body.String())
	}

	// Navigate into the schema
	schema, ok := findJSONSchema(jsonDoc)
	if !ok {
		t.Fatalf("Could not find schema in JSON metadata.\nBody:\n%s", w.Body.String())
	}

	carType, ok := schema["CarEntity"].(map[string]interface{})
	if !ok {
		t.Fatalf("CarEntity not found in JSON metadata schema.\nBody:\n%s", w.Body.String())
	}

	// Must have $BaseType
	if baseType, ok := carType["$BaseType"]; !ok || baseType != "TestNamespace.VehicleEntity" {
		t.Errorf("CarEntity JSON metadata should have $BaseType=\"TestNamespace.VehicleEntity\", got: %v", baseType)
	}

	// Derived type must NOT have $Key
	if _, ok := carType["$Key"]; ok {
		t.Errorf("Derived type CarEntity JSON metadata should NOT have $Key (key is inherited).\nCarEntity: %v", carType)
	}

	// VehicleEntity must still have $Key
	vehicleType, ok := schema["VehicleEntity"].(map[string]interface{})
	if !ok {
		t.Fatalf("VehicleEntity not found in JSON metadata schema.\nBody:\n%s", w.Body.String())
	}
	if _, ok := vehicleType["$Key"]; !ok {
		t.Errorf("Base type VehicleEntity JSON metadata must have $Key.\nVehicleEntity: %v", vehicleType)
	}
}

func TestAbstract_JSON(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	abstractMeta, err := metadata.AnalyzeEntity(AbstractVehicleEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze AbstractVehicleEntity: %v", err)
	}
	entities[abstractMeta.EntitySetName] = abstractMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var jsonDoc map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &jsonDoc); err != nil {
		t.Fatalf("Response is not valid JSON: %v\nBody: %s", err, w.Body.String())
	}

	schema, ok := findJSONSchema(jsonDoc)
	if !ok {
		t.Fatalf("Could not find schema in JSON metadata.\nBody:\n%s", w.Body.String())
	}

	abstractType, ok := schema["AbstractVehicleEntity"].(map[string]interface{})
	if !ok {
		t.Fatalf("AbstractVehicleEntity not found in JSON metadata schema.\nBody:\n%s", w.Body.String())
	}

	if abstract, ok := abstractType["$Abstract"]; !ok || abstract != true {
		t.Errorf("AbstractVehicleEntity JSON metadata should have $Abstract=true, got: %v", abstract)
	}
}

// findJSONSchema navigates the CSDL JSON document to locate the first schema map.
func findJSONSchema(jsonDoc map[string]interface{}) (map[string]interface{}, bool) {
	// CSDL JSON: {"$Version":"4.0","$EntityContainer":"...","Namespace.":{"EntityType":...}}
	for k, v := range jsonDoc {
		if k == "$Version" || k == "$EntityContainer" || k == "$Reference" {
			continue
		}
		if schema, ok := v.(map[string]interface{}); ok {
			return schema, true
		}
	}
	return nil, false
}
