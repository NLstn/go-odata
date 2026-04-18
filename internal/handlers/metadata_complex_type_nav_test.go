package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// ---- test entities for complex type navigation property tests ----

// NavCountry is an entity that a complex type references via a navigation property.
type NavCountry struct {
	ID   string `json:"ID" odata:"key"`
	Name string `json:"Name"`
}

// NavAddress is a complex type that contains a navigation property to NavCountry.
type NavAddress struct {
	Street  string      `json:"Street"`
	City    string      `json:"City"`
	Country *NavCountry `json:"Country,omitempty" gorm:"foreignKey:CountryID;references:ID"`
}

// NavOrder is an entity that embeds NavAddress as a complex type.
type NavOrder struct {
	ID              string     `json:"ID" odata:"key"`
	CountryID       string     `json:"CountryID"`
	ShippingAddress NavAddress `json:"ShippingAddress,omitempty" gorm:"embedded;embeddedPrefix:shipping_"`
}

func buildComplexNavTestEntities(t *testing.T) map[string]*metadata.EntityMetadata {
	t.Helper()
	entities := make(map[string]*metadata.EntityMetadata)

	orderMeta, err := metadata.AnalyzeEntity(NavOrder{})
	if err != nil {
		t.Fatalf("Failed to analyze NavOrder: %v", err)
	}
	entities[orderMeta.EntitySetName] = orderMeta

	countryMeta, err := metadata.AnalyzeEntity(NavCountry{})
	if err != nil {
		t.Fatalf("Failed to analyze NavCountry: %v", err)
	}
	entities[countryMeta.EntitySetName] = countryMeta

	return entities
}

// TestComplexTypeNavigation_XML verifies that:
//  1. A <ComplexType> element is emitted for complex types.
//  2. Navigation properties inside the complex type appear as
//     <NavigationProperty> elements inside the <ComplexType>.
//  3. The entity container includes a NavigationPropertyBinding for the nested
//     navigation property using a "ComplexProp/NavProp" path.
func TestComplexTypeNavigation_XML(t *testing.T) {
	entities := buildComplexNavTestEntities(t)
	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	// 1. A ComplexType element must exist for NavAddress
	if !strings.Contains(body, `<ComplexType Name="NavAddress">`) {
		t.Errorf("XML metadata should contain <ComplexType Name=\"NavAddress\">\nBody:\n%s", body)
	}

	// 2. The navigation property must appear inside the ComplexType
	if !strings.Contains(body, `<NavigationProperty Name="Country"`) {
		t.Errorf("XML metadata should contain NavigationProperty for Country inside ComplexType\nBody:\n%s", body)
	}

	// 3. The entity container must include a binding with path ShippingAddress/Country
	if !strings.Contains(body, `Path="ShippingAddress/Country"`) {
		t.Errorf("XML metadata should contain NavigationPropertyBinding Path=\"ShippingAddress/Country\"\nBody:\n%s", body)
	}
}

// TestComplexTypeNavigation_JSON verifies that:
//  1. A ComplexType entry exists in the CSDL JSON.
//  2. The navigation property inside the complex type has $Kind=NavigationProperty.
//  3. The entity set includes a $NavigationPropertyBinding for the nested path.
func TestComplexTypeNavigation_JSON(t *testing.T) {
	entities := buildComplexNavTestEntities(t)
	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON metadata: %v\nBody: %s", err, w.Body.String())
	}

	odataService, ok := result["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected ODataService in JSON metadata")
	}

	// 1. ComplexType entry must exist
	navAddress, ok := odataService["NavAddress"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected NavAddress ComplexType in ODataService. Keys: %v", keys(odataService))
	}

	kind, _ := navAddress["$Kind"].(string)
	if kind != "ComplexType" {
		t.Errorf("Expected $Kind=ComplexType for NavAddress, got %q", kind)
	}

	// 2. Navigation property inside the complex type must be present
	countryNav, ok := navAddress["Country"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected Country navigation property inside NavAddress. Keys: %v", keys(navAddress))
	}

	navKind, _ := countryNav["$Kind"].(string)
	if navKind != "NavigationProperty" {
		t.Errorf("Expected $Kind=NavigationProperty for Country, got %q", navKind)
	}

	navType, _ := countryNav["$Type"].(string)
	if !strings.Contains(navType, "NavCountry") {
		t.Errorf("Expected $Type to reference NavCountry, got %q", navType)
	}

	// 3. Entity set must include $NavigationPropertyBinding for nested path
	container, ok := odataService["Container"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Container in ODataService")
	}

	navOrders, ok := container["NavOrders"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected NavOrders entity set in Container. Keys: %v", keys(container))
	}

	navBindings, ok := navOrders["$NavigationPropertyBinding"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected $NavigationPropertyBinding in NavOrders. Keys: %v", keys(navOrders))
	}

	target, ok := navBindings["ShippingAddress/Country"].(string)
	if !ok {
		t.Fatalf("Expected ShippingAddress/Country binding. Bindings: %v", navBindings)
	}

	if !strings.Contains(target, "NavCountr") {
		t.Errorf("Expected ShippingAddress/Country to target NavCountr..., got %q", target)
	}
}

// TestComplexTypeWithoutNavigation_XML verifies that plain complex types (no nav props)
// still emit a <ComplexType> element in the XML metadata.
func TestComplexTypeWithoutNavigation_XML(t *testing.T) {
	type SimpleAddress struct {
		Street string `json:"Street"`
		City   string `json:"City"`
	}

	type Invoice struct {
		ID      string        `json:"ID" odata:"key"`
		Address SimpleAddress `json:"Address,omitempty" gorm:"embedded"`
	}

	entities := make(map[string]*metadata.EntityMetadata)
	invoiceMeta, err := metadata.AnalyzeEntity(Invoice{})
	if err != nil {
		t.Fatalf("Failed to analyze Invoice: %v", err)
	}
	entities[invoiceMeta.EntitySetName] = invoiceMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	body := w.Body.String()

	if !strings.Contains(body, `<ComplexType Name="SimpleAddress">`) {
		t.Errorf("XML metadata should contain <ComplexType Name=\"SimpleAddress\">\nBody:\n%s", body)
	}

	// No NavigationPropertyBinding should be added for plain complex types
	if strings.Contains(body, `Path="Address/`) {
		t.Errorf("XML should not contain a NavigationPropertyBinding path for plain complex type\nBody:\n%s", body)
	}
}

// TestComplexTypeWithoutNavigation_JSON verifies that plain complex types emit
// a $Kind=ComplexType entry in the JSON metadata.
func TestComplexTypeWithoutNavigation_JSON(t *testing.T) {
	type SimpleAddress struct {
		Street string `json:"Street"`
		City   string `json:"City"`
	}

	type Invoice struct {
		ID      string        `json:"ID" odata:"key"`
		Address SimpleAddress `json:"Address,omitempty" gorm:"embedded"`
	}

	entities := make(map[string]*metadata.EntityMetadata)
	invoiceMeta, err := metadata.AnalyzeEntity(Invoice{})
	if err != nil {
		t.Fatalf("Failed to analyze Invoice: %v", err)
	}
	entities[invoiceMeta.EntitySetName] = invoiceMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON metadata: %v", err)
	}

	odataService, ok := result["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected ODataService in JSON metadata")
	}

	simpleAddr, ok := odataService["SimpleAddress"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected SimpleAddress in ODataService. Keys: %v", keys(odataService))
	}

	kind, _ := simpleAddr["$Kind"].(string)
	if kind != "ComplexType" {
		t.Errorf("Expected $Kind=ComplexType for SimpleAddress, got %q", kind)
	}
}
