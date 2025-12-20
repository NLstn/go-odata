package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// Test entities for navigation property binding with custom EntitySetName

// MemberPrivacySettings has a custom EntitySetName that doesn't follow standard pluralization
type MemberPrivacySettings struct {
	ID             string `json:"ID" odata:"key"`
	MemberID       string `json:"MemberID" odata:"required"`
	ShareBirthDate bool   `json:"ShareBirthDate"`
}

func (MemberPrivacySettings) EntitySetName() string {
	return "MemberPrivacySettings" // Without extra "es" suffix
}

// Member has a navigation property to MemberPrivacySettings
type Member struct {
	ID              string                  `json:"ID" odata:"key"`
	Name            string                  `json:"Name"`
	PrivacySettings *MemberPrivacySettings `json:"PrivacySettings,omitempty" gorm:"foreignKey:MemberID;references:ID"`
}

// TestNavigationBindingWithCustomEntitySetName_XML tests that navigation property bindings
// in XML metadata respect custom EntitySetName() methods
func TestNavigationBindingWithCustomEntitySetName_XML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	// Analyze and register both entities
	memberMeta, err := metadata.AnalyzeEntity(Member{})
	if err != nil {
		t.Fatalf("Failed to analyze Member entity: %v", err)
	}
	entities[memberMeta.EntitySetName] = memberMeta

	privacyMeta, err := metadata.AnalyzeEntity(MemberPrivacySettings{})
	if err != nil {
		t.Fatalf("Failed to analyze MemberPrivacySettings entity: %v", err)
	}
	entities[privacyMeta.EntitySetName] = privacyMeta

	// Verify the custom entity set name was detected
	if privacyMeta.EntitySetName != "MemberPrivacySettings" {
		t.Errorf("Expected EntitySetName='MemberPrivacySettings', got '%s'", privacyMeta.EntitySetName)
	}

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// Check that the navigation property binding uses the correct entity set name
	// It should be "MemberPrivacySettings", NOT "MemberPrivacySettingses"
	expectedBinding := `<NavigationPropertyBinding Path="PrivacySettings" Target="MemberPrivacySettings" />`
	if !strings.Contains(body, expectedBinding) {
		t.Errorf("XML metadata should contain correct navigation binding: %s\nGot body:\n%s", expectedBinding, body)
	}

	// Make sure it's NOT using the incorrectly pluralized version
	incorrectBinding := `Target="MemberPrivacySettingses"`
	if strings.Contains(body, incorrectBinding) {
		t.Errorf("XML metadata should NOT contain incorrectly pluralized binding: %s", incorrectBinding)
	}
}

// TestNavigationBindingWithCustomEntitySetName_JSON tests that navigation property bindings
// in JSON metadata respect custom EntitySetName() methods
func TestNavigationBindingWithCustomEntitySetName_JSON(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	// Analyze and register both entities
	memberMeta, err := metadata.AnalyzeEntity(Member{})
	if err != nil {
		t.Fatalf("Failed to analyze Member entity: %v", err)
	}
	entities[memberMeta.EntitySetName] = memberMeta

	privacyMeta, err := metadata.AnalyzeEntity(MemberPrivacySettings{})
	if err != nil {
		t.Fatalf("Failed to analyze MemberPrivacySettings entity: %v", err)
	}
	entities[privacyMeta.EntitySetName] = privacyMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Parse the JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON metadata: %v", err)
	}

	// Navigate to the Members entity set
	odataService, ok := result["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected ODataService in JSON metadata")
	}

	container, ok := odataService["Container"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Container in ODataService")
	}

	members, ok := container["Members"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Members entity set in Container")
	}

	// Check for navigation property binding
	navBindings, ok := members["$NavigationPropertyBinding"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected $NavigationPropertyBinding in Members entity set")
	}

	// Verify the navigation binding uses the correct entity set name
	privacySettingsTarget, ok := navBindings["PrivacySettings"].(string)
	if !ok {
		t.Fatal("Expected PrivacySettings navigation binding")
	}

	expectedTarget := "MemberPrivacySettings"
	if privacySettingsTarget != expectedTarget {
		t.Errorf("Expected navigation binding target to be '%s', got '%s'", expectedTarget, privacySettingsTarget)
	}

	// Make sure it's NOT the incorrectly pluralized version
	incorrectTarget := "MemberPrivacySettingses"
	if privacySettingsTarget == incorrectTarget {
		t.Errorf("Navigation binding should NOT use incorrectly pluralized target: %s", incorrectTarget)
	}
}

// TestNavigationBindingWithStandardPluralization_XML tests that entities without
// custom EntitySetName still work with standard pluralization
func TestNavigationBindingWithStandardPluralization_XML(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	// Define simple entities without custom EntitySetName
	type Category struct {
		ID   string `json:"id" odata:"key"`
		Name string `json:"name"`
	}

	type Product struct {
		ID         string    `json:"id" odata:"key"`
		Name       string    `json:"name"`
		CategoryID string    `json:"categoryId"`
		Category   *Category `json:"category" gorm:"foreignKey:CategoryID;references:ID"`
	}

	// Analyze and register both entities
	productMeta, err := metadata.AnalyzeEntity(Product{})
	if err != nil {
		t.Fatalf("Failed to analyze Product entity: %v", err)
	}
	entities[productMeta.EntitySetName] = productMeta

	categoryMeta, err := metadata.AnalyzeEntity(Category{})
	if err != nil {
		t.Fatalf("Failed to analyze Category entity: %v", err)
	}
	entities[categoryMeta.EntitySetName] = categoryMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()

	// Check that standard pluralization still works (Category -> Categories)
	expectedBinding := `<NavigationPropertyBinding Path="category" Target="Categories" />`
	if !strings.Contains(body, expectedBinding) {
		t.Errorf("XML metadata should contain navigation binding with standard pluralization: %s\nGot body:\n%s", expectedBinding, body)
	}
}

// TestNavigationBindingWithStandardPluralization_JSON tests that entities without
// custom EntitySetName still work with standard pluralization
func TestNavigationBindingWithStandardPluralization_JSON(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	// Define simple entities without custom EntitySetName
	type Category struct {
		ID   string `json:"id" odata:"key"`
		Name string `json:"name"`
	}

	type Product struct {
		ID         string    `json:"id" odata:"key"`
		Name       string    `json:"name"`
		CategoryID string    `json:"categoryId"`
		Category   *Category `json:"category" gorm:"foreignKey:CategoryID;references:ID"`
	}

	// Analyze and register both entities
	productMeta, err := metadata.AnalyzeEntity(Product{})
	if err != nil {
		t.Fatalf("Failed to analyze Product entity: %v", err)
	}
	entities[productMeta.EntitySetName] = productMeta

	categoryMeta, err := metadata.AnalyzeEntity(Category{})
	if err != nil {
		t.Fatalf("Failed to analyze Category entity: %v", err)
	}
	entities[categoryMeta.EntitySetName] = categoryMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Parse the JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON metadata: %v", err)
	}

	// Navigate to the Products entity set
	odataService, ok := result["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected ODataService in JSON metadata")
	}

	container, ok := odataService["Container"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Container in ODataService")
	}

	products, ok := container["Products"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Products entity set in Container")
	}

	// Check for navigation property binding
	navBindings, ok := products["$NavigationPropertyBinding"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected $NavigationPropertyBinding in Products entity set")
	}

	// Verify standard pluralization works (Category -> Categories)
	categoryTarget, ok := navBindings["category"].(string)
	if !ok {
		t.Fatal("Expected category navigation binding")
	}

	expectedTarget := "Categories"
	if categoryTarget != expectedTarget {
		t.Errorf("Expected navigation binding target to be '%s', got '%s'", expectedTarget, categoryTarget)
	}
}
