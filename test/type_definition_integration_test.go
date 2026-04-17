package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Weight is a named Go type used as an OData TypeDefinition (float64 → Edm.Double)
type Weight float64

// Distance is a named Go type used as an OData TypeDefinition (float64 → Edm.Double)
type Distance float64

// ShortDescription is a named Go type for a string TypeDefinition with MaxLength
type ShortDescription string

// Package is an entity that uses TypeDefinition properties
type Package struct {
	ID          string           `json:"ID" gorm:"primaryKey" odata:"key"`
	Weight      Weight           `json:"Weight" gorm:"not null"`
	Distance    Distance         `json:"Distance"`
	Description ShortDescription `json:"Description"`
}

func init() {
	// Register TypeDefinitions before tests run.
	// Weight and Distance are float64-based → Edm.Double (Precision/Scale not valid for Edm.Double per OData spec)
	if err := odata.RegisterTypeDefinition(Weight(0), "Weight", odata.TypeDefinitionFacets{}); err != nil {
		panic("failed to register Weight TypeDefinition: " + err.Error())
	}
	if err := odata.RegisterTypeDefinition(Distance(0), "Distance", odata.TypeDefinitionFacets{}); err != nil {
		panic("failed to register Distance TypeDefinition: " + err.Error())
	}
	// ShortDescription is string-based → Edm.String (MaxLength is valid)
	if err := odata.RegisterTypeDefinition(ShortDescription(""), "ShortDescription", odata.TypeDefinitionFacets{MaxLength: 100}); err != nil {
		panic("failed to register ShortDescription TypeDefinition: " + err.Error())
	}
}

func setupTypeDefinitionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&Package{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	packages := []Package{
		{ID: "pkg-1", Weight: 1.5, Distance: 10.2, Description: "Small package"},
		{ID: "pkg-2", Weight: 3.75, Distance: 25.0, Description: "Medium package"},
	}
	if err := db.Create(&packages).Error; err != nil {
		t.Fatalf("Failed to seed packages: %v", err)
	}
	return db
}

func TestTypeDefinitionMetadata(t *testing.T) {
	db := setupTypeDefinitionTestDB(t)

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&Package{}); err != nil {
		t.Fatalf("Failed to register Package entity: %v", err)
	}

	tests := []struct {
		name         string
		acceptJSON   bool
		checkContent func(t *testing.T, body string)
		description  string
	}{
		{
			name:       "XML metadata contains TypeDefinition elements",
			acceptJSON: false,
			checkContent: func(t *testing.T, body string) {
				// Weight TypeDefinition (float64 → Edm.Double, no facets)
				if !strings.Contains(body, `<TypeDefinition Name="Weight" UnderlyingType="Edm.Double"`) {
					t.Errorf("Expected TypeDefinition for Weight in XML metadata, got:\n%s", body)
				}

				// Distance TypeDefinition (float64 → Edm.Double)
				if !strings.Contains(body, `<TypeDefinition Name="Distance" UnderlyingType="Edm.Double"`) {
					t.Errorf("Expected TypeDefinition for Distance in XML metadata, got:\n%s", body)
				}

				// ShortDescription TypeDefinition with MaxLength (string → Edm.String)
				if !strings.Contains(body, `<TypeDefinition Name="ShortDescription" UnderlyingType="Edm.String"`) {
					t.Errorf("Expected TypeDefinition for ShortDescription in XML metadata, got:\n%s", body)
				}
				if !strings.Contains(body, `MaxLength="100"`) {
					t.Errorf("Expected MaxLength facet on ShortDescription TypeDefinition")
				}

				// Weight property references the TypeDefinition type
				if !strings.Contains(body, `Type="ODataService.Weight"`) {
					t.Errorf("Expected Weight property to reference ODataService.Weight type, got:\n%s", body)
				}

				// Distance property references the TypeDefinition type
				if !strings.Contains(body, `Type="ODataService.Distance"`) {
					t.Errorf("Expected Distance property to reference ODataService.Distance type, got:\n%s", body)
				}

				// Description property references the TypeDefinition type
				if !strings.Contains(body, `Type="ODataService.ShortDescription"`) {
					t.Errorf("Expected Description property to reference ODataService.ShortDescription type, got:\n%s", body)
				}
			},
			description: "XML metadata should contain TypeDefinition elements with facets",
		},
		{
			name:       "JSON metadata contains TypeDefinition elements",
			acceptJSON: true,
			checkContent: func(t *testing.T, body string) {
				var csdl map[string]interface{}
				if err := json.Unmarshal([]byte(body), &csdl); err != nil {
					t.Fatalf("Failed to parse JSON metadata: %v", err)
				}

				odataService, ok := csdl["ODataService"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected ODataService namespace in JSON metadata")
				}

				// Weight TypeDefinition
				weightDef, ok := odataService["Weight"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected Weight TypeDefinition in JSON metadata")
				}
				if kind, _ := weightDef["$Kind"].(string); kind != "TypeDefinition" {
					t.Errorf("Expected Weight.$Kind to be TypeDefinition, got %s", kind)
				}
				if ut, _ := weightDef["$UnderlyingType"].(string); ut != "Edm.Double" {
					t.Errorf("Expected Weight.$UnderlyingType to be Edm.Double, got %s", ut)
				}

				// Distance TypeDefinition
				distanceDef, ok := odataService["Distance"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected Distance TypeDefinition in JSON metadata")
				}
				if kind, _ := distanceDef["$Kind"].(string); kind != "TypeDefinition" {
					t.Errorf("Expected Distance.$Kind to be TypeDefinition, got %s", kind)
				}

				// ShortDescription TypeDefinition with MaxLength
				shortDescDef, ok := odataService["ShortDescription"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected ShortDescription TypeDefinition in JSON metadata")
				}
				if kind, _ := shortDescDef["$Kind"].(string); kind != "TypeDefinition" {
					t.Errorf("Expected ShortDescription.$Kind to be TypeDefinition, got %s", kind)
				}
				if ut, _ := shortDescDef["$UnderlyingType"].(string); ut != "Edm.String" {
					t.Errorf("Expected ShortDescription.$UnderlyingType to be Edm.String, got %s", ut)
				}
				if ml, ok := shortDescDef["$MaxLength"]; !ok || ml.(float64) != 100 {
					t.Errorf("Expected ShortDescription.$MaxLength to be 100, got %v", ml)
				}

				// Package entity: Weight property uses TypeDefinition
				packageEntity, ok := odataService["Package"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected Package entity type in JSON metadata")
				}
				weightProp, ok := packageEntity["Weight"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected Weight property in Package entity")
				}
				if weightType, _ := weightProp["$Type"].(string); weightType != "ODataService.Weight" {
					t.Errorf("Expected Weight property type to be ODataService.Weight, got %s", weightType)
				}
			},
			description: "JSON metadata should contain TypeDefinition elements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
			if tt.acceptJSON {
				req.Header.Set("Accept", "application/json")
			}

			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected 200 OK, got %d", w.Code)
			}

			tt.checkContent(t, w.Body.String())
		})
	}
}

func TestTypeDefinitionQueryWorks(t *testing.T) {
	db := setupTypeDefinitionTestDB(t)

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&Package{}); err != nil {
		t.Fatalf("Failed to register Package entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/Packages", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	values, ok := result["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(values) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(values))
	}
}
