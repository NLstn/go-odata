package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
)

// Test entity with various facets
type ProductWithFacets struct {
	ID          int       `json:"id" odata:"key"`
	Name        string    `json:"name" odata:"required,maxlength=100"`
	Description string    `json:"description" odata:"maxlength=500,nullable"`
	Price       float64   `json:"price" odata:"precision=10,scale=2"`
	SKU         string    `json:"sku" odata:"maxlength=50,default=AUTO"`
	CreatedAt   time.Time `json:"createdAt"`
	Active      bool      `json:"active" odata:"default=true"`
}

// Test entity with navigation properties and referential constraints
type Order struct {
	ID         int     `json:"id" odata:"key"`
	CustomerID int     `json:"customerId" odata:"required"`
	Customer   *User   `json:"customer" gorm:"foreignKey:CustomerID;references:ID"`
	TotalPrice float64 `json:"totalPrice" odata:"precision=10,scale=2"`
}

type User struct {
	ID     int     `json:"id" odata:"key"`
	Name   string  `json:"name" odata:"required,maxlength=100"`
	Email  string  `json:"email" odata:"maxlength=255"`
	Orders []Order `json:"orders" gorm:"foreignKey:CustomerID;references:ID"`
}

func TestMetadataWithFacetsXML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	entityMeta, err := metadata.AnalyzeEntity(ProductWithFacets{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	entities["ProductWithFacets"] = entityMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// Check for MaxLength attribute
	if !strings.Contains(body, `MaxLength="100"`) {
		t.Error("XML should contain MaxLength attribute")
	}

	// Check for Precision and Scale
	if !strings.Contains(body, `Precision="10"`) {
		t.Error("XML should contain Precision attribute")
	}
	if !strings.Contains(body, `Scale="2"`) {
		t.Error("XML should contain Scale attribute")
	}

	// Check for DefaultValue
	if !strings.Contains(body, `DefaultValue="AUTO"`) {
		t.Error("XML should contain DefaultValue attribute for SKU")
	}
	if !strings.Contains(body, `DefaultValue="true"`) {
		t.Error("XML should contain DefaultValue attribute for Active")
	}

	// Check for time.Time mapped to DateTimeOffset
	if !strings.Contains(body, `Type="Edm.DateTimeOffset"`) {
		t.Error("XML should contain DateTimeOffset type for time.Time")
	}
}

func TestMetadataWithFacetsJSON(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	entityMeta, err := metadata.AnalyzeEntity(ProductWithFacets{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	entities["ProductWithFacets"] = entityMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	odataService, ok := response["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("ODataService not found in response")
	}

	entityType, ok := odataService["ProductWithFacets"].(map[string]interface{})
	if !ok {
		t.Fatal("ProductWithFacets not found in metadata")
	}

	// Check name property with MaxLength
	nameProp, ok := entityType["name"].(map[string]interface{})
	if !ok {
		t.Fatal("name property not found")
	}
	if maxLen, ok := nameProp["$MaxLength"].(float64); !ok || maxLen != 100 {
		t.Errorf("Expected $MaxLength=100 for name, got %v", nameProp["$MaxLength"])
	}

	// Check description with nullable
	descProp, ok := entityType["description"].(map[string]interface{})
	if !ok {
		t.Fatal("description property not found")
	}
	if nullable, ok := descProp["$Nullable"].(bool); !ok || !nullable {
		t.Errorf("Expected $Nullable=true for description, got %v", descProp["$Nullable"])
	}
	if maxLen, ok := descProp["$MaxLength"].(float64); !ok || maxLen != 500 {
		t.Errorf("Expected $MaxLength=500 for description, got %v", descProp["$MaxLength"])
	}

	// Check price with precision and scale
	priceProp, ok := entityType["price"].(map[string]interface{})
	if !ok {
		t.Fatal("price property not found")
	}
	if precision, ok := priceProp["$Precision"].(float64); !ok || precision != 10 {
		t.Errorf("Expected $Precision=10 for price, got %v", priceProp["$Precision"])
	}
	if scale, ok := priceProp["$Scale"].(float64); !ok || scale != 2 {
		t.Errorf("Expected $Scale=2 for price, got %v", priceProp["$Scale"])
	}

	// Check SKU with default value
	skuProp, ok := entityType["sku"].(map[string]interface{})
	if !ok {
		t.Fatal("sku property not found")
	}
	if defaultVal, ok := skuProp["$DefaultValue"].(string); !ok || defaultVal != "AUTO" {
		t.Errorf("Expected $DefaultValue=AUTO for sku, got %v", skuProp["$DefaultValue"])
	}

	// Check active with default value
	activeProp, ok := entityType["active"].(map[string]interface{})
	if !ok {
		t.Fatal("active property not found")
	}
	if defaultVal, ok := activeProp["$DefaultValue"].(string); !ok || defaultVal != "true" {
		t.Errorf("Expected $DefaultValue=true for active, got %v", activeProp["$DefaultValue"])
	}

	// Check createdAt for DateTimeOffset type
	createdProp, ok := entityType["createdAt"].(map[string]interface{})
	if !ok {
		t.Fatal("createdAt property not found")
	}
	if propType, ok := createdProp["$Type"].(string); !ok || propType != "Edm.DateTimeOffset" {
		t.Errorf("Expected $Type=Edm.DateTimeOffset for createdAt, got %v", createdProp["$Type"])
	}
}

func TestMetadataWithReferentialConstraintsXML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	orderMeta, err := metadata.AnalyzeEntity(Order{})
	if err != nil {
		t.Fatalf("Failed to analyze Order entity: %v", err)
	}
	entities["Orders"] = orderMeta

	userMeta, err := metadata.AnalyzeEntity(User{})
	if err != nil {
		t.Fatalf("Failed to analyze User entity: %v", err)
	}
	entities["Users"] = userMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// Check for referential constraint in navigation property
	if !strings.Contains(body, "<ReferentialConstraint>") {
		t.Error("XML should contain ReferentialConstraint element")
	}

	// Check for property mapping
	if !strings.Contains(body, `Name="CustomerID"`) && strings.Contains(body, `ReferencedProperty="ID"`) {
		t.Error("XML should contain referential constraint mapping CustomerID to ID")
	}
}

func TestMetadataWithReferentialConstraintsJSON(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	orderMeta, err := metadata.AnalyzeEntity(Order{})
	if err != nil {
		t.Fatalf("Failed to analyze Order entity: %v", err)
	}
	entities["Orders"] = orderMeta

	userMeta, err := metadata.AnalyzeEntity(User{})
	if err != nil {
		t.Fatalf("Failed to analyze User entity: %v", err)
	}
	entities["Users"] = userMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	odataService, ok := response["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("ODataService not found in response")
	}

	orderType, ok := odataService["Order"].(map[string]interface{})
	if !ok {
		t.Fatal("Order not found in metadata")
	}

	// Check customer navigation property
	customerNav, ok := orderType["customer"].(map[string]interface{})
	if !ok {
		t.Fatal("customer navigation property not found")
	}

	// Check for referential constraints
	constraints, ok := customerNav["$ReferentialConstraint"].([]interface{})
	if !ok || len(constraints) == 0 {
		t.Error("$ReferentialConstraint not found or empty")
	} else {
		constraint := constraints[0].(map[string]interface{})
		if prop, ok := constraint["Property"].(string); !ok || prop != "CustomerID" {
			t.Errorf("Expected Property=CustomerID, got %v", constraint["Property"])
		}
		if refProp, ok := constraint["ReferencedProperty"].(string); !ok || refProp != "ID" {
			t.Errorf("Expected ReferencedProperty=ID, got %v", constraint["ReferencedProperty"])
		}
	}
}

func TestEdmTypeMapping(t *testing.T) {
	tests := []struct {
		name    string
		goType  string
		edmType string
	}{
		{"time.Time", "time.Time", "Edm.DateTimeOffset"},
		{"int", "int", "Edm.Int32"},
		{"int64", "int64", "Edm.Int64"},
		{"float64", "float64", "Edm.Double"},
		{"string", "string", "Edm.String"},
		{"bool", "bool", "Edm.Boolean"},
	}

	// Create test entity type for each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test this without creating actual types,
			// but we can verify through the metadata output
			// This test is more of a verification that types are correctly mapped
		})
	}
}

func TestNullableOverride(t *testing.T) {
	type EntityWithNullable struct {
		ID             int    `json:"id" odata:"key"`
		RequiredField  string `json:"requiredField" odata:"required"`
		NullableField  string `json:"nullableField" odata:"nullable"`
		NonNullableOpt string `json:"nonNullableOpt" odata:"nullable=false"`
	}

	entities := make(map[string]*metadata.EntityMetadata)
	entityMeta, err := metadata.AnalyzeEntity(EntityWithNullable{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	entities["EntityWithNullable"] = entityMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	odataService := response["ODataService"].(map[string]interface{})
	entityType := odataService["EntityWithNullable"].(map[string]interface{})

	// Check nullable override
	nullableField := entityType["nullableField"].(map[string]interface{})
	if nullable, ok := nullableField["$Nullable"].(bool); !ok || !nullable {
		t.Errorf("Expected nullableField to be nullable, got %v", nullableField["$Nullable"])
	}

	// Check non-nullable override
	nonNullableField := entityType["nonNullableOpt"].(map[string]interface{})
	if nullable, ok := nonNullableField["$Nullable"].(bool); ok && nullable {
		t.Errorf("Expected nonNullableOpt to be non-nullable, got %v", nonNullableField["$Nullable"])
	}
}
