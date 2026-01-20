package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestMetadataJSON_VocabularyReferences tests that vocabulary references are correctly
// included in JSON metadata when annotations are present
func TestMetadataJSON_VocabularyReferences(t *testing.T) {
	t.Run("WithAnnotations_IncludesReferences", func(t *testing.T) {
		// Create an entity with annotations
		meta, _ := metadata.AnalyzeEntity(AnnotatedTestProduct{})
		entitiesMetadata := map[string]*metadata.EntityMetadata{
			"AnnotatedTestProducts": meta,
		}

		handler := NewMetadataHandler(entitiesMetadata)

		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		// Verify $Reference section exists
		references, ok := response["$Reference"]
		if !ok {
			t.Fatal("Expected $Reference section to be present in JSON metadata")
		}

		referencesMap, ok := references.(map[string]interface{})
		if !ok {
			t.Fatal("Expected $Reference to be a map")
		}

		// Verify Core vocabulary reference exists
		coreURI := "https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Core.V1.xml"
		coreRef, ok := referencesMap[coreURI]
		if !ok {
			t.Errorf("Expected Core vocabulary reference at URI %s", coreURI)
		}

		// Verify $Include array structure
		coreRefMap, ok := coreRef.(map[string]interface{})
		if !ok {
			t.Fatal("Expected Core vocabulary reference to be a map")
		}

		includeArray, ok := coreRefMap["$Include"]
		if !ok {
			t.Fatal("Expected $Include array in Core vocabulary reference")
		}

		includeSlice, ok := includeArray.([]interface{})
		if !ok {
			t.Fatal("Expected $Include to be an array")
		}

		if len(includeSlice) == 0 {
			t.Fatal("Expected $Include array to have at least one element")
		}

		// Verify namespace and alias in first include element
		firstInclude, ok := includeSlice[0].(map[string]interface{})
		if !ok {
			t.Fatal("Expected first $Include element to be a map")
		}

		namespace, ok := firstInclude["$Namespace"]
		if !ok {
			t.Error("Expected $Namespace in $Include element")
		}
		if namespace != "Org.OData.Core.V1" {
			t.Errorf("Expected $Namespace = 'Org.OData.Core.V1', got %v", namespace)
		}

		alias, ok := firstInclude["$Alias"]
		if !ok {
			t.Error("Expected $Alias in $Include element")
		}
		if alias != "Core" {
			t.Errorf("Expected $Alias = 'Core', got %v", alias)
		}
	})

	t.Run("WithoutAnnotations_NoReferences", func(t *testing.T) {
		// Create an entity without annotations (not even auto-generated ones)
		meta, _ := metadata.AnalyzeEntity(PlainTestProduct{})
		entitiesMetadata := map[string]*metadata.EntityMetadata{
			"PlainTestProducts": meta,
		}

		handler := NewMetadataHandler(entitiesMetadata)

		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		// Verify $Reference section does NOT exist
		if _, ok := response["$Reference"]; ok {
			t.Error("Expected $Reference section to be absent when no annotations are present")
		}
	})

	t.Run("MultipleVocabularies_IncludesAll", func(t *testing.T) {
		// Create an entity with annotations from multiple vocabularies
		meta, _ := metadata.AnalyzeEntity(MultiVocabTestProduct{})
		entitiesMetadata := map[string]*metadata.EntityMetadata{
			"MultiVocabTestProducts": meta,
		}

		handler := NewMetadataHandler(entitiesMetadata)

		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		// Verify $Reference section exists
		references, ok := response["$Reference"]
		if !ok {
			t.Fatal("Expected $Reference section to be present")
		}

		referencesMap, ok := references.(map[string]interface{})
		if !ok {
			t.Fatal("Expected $Reference to be a map")
		}

		// Verify both Core and Capabilities vocabularies are referenced
		coreURI := "https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Core.V1.xml"
		capURI := "https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Capabilities.V1.xml"

		if _, ok := referencesMap[coreURI]; !ok {
			t.Error("Expected Core vocabulary reference")
		}

		if _, ok := referencesMap[capURI]; !ok {
			t.Error("Expected Capabilities vocabulary reference")
		}
	})

	t.Run("CustomVocabulary_UsesFallbackAlias", func(t *testing.T) {
		// Create an entity with a custom vocabulary annotation
		meta, _ := metadata.AnalyzeEntity(CustomVocabTestProduct{})
		
		// Manually add a custom vocabulary annotation (not in the standard map)
		if meta.Annotations == nil {
			meta.Annotations = metadata.NewAnnotationCollection()
		}
		meta.Annotations.AddTerm("Custom.Namespace.V1.CustomTerm", true)

		entitiesMetadata := map[string]*metadata.EntityMetadata{
			"CustomVocabTestProducts": meta,
		}

		handler := NewMetadataHandler(entitiesMetadata)

		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		// Verify $Reference section exists
		references, ok := response["$Reference"]
		if !ok {
			t.Fatal("Expected $Reference section to be present")
		}

		referencesMap, ok := references.(map[string]interface{})
		if !ok {
			t.Fatal("Expected $Reference to be a map")
		}

		// Custom vocabulary should use the URN pattern
		customURI := "urn:custom:vocabulary:Custom.Namespace.V1"
		customRef, ok := referencesMap[customURI]
		if !ok {
			t.Errorf("Expected custom vocabulary reference at URI %s", customURI)
		}

		// Verify the alias is the last segment of the namespace (fallback behavior)
		customRefMap, ok := customRef.(map[string]interface{})
		if !ok {
			t.Fatal("Expected custom vocabulary reference to be a map")
		}

		includeArray, ok := customRefMap["$Include"]
		if !ok {
			t.Fatal("Expected $Include array")
		}

		includeSlice, ok := includeArray.([]interface{})
		if !ok || len(includeSlice) == 0 {
			t.Fatal("Expected $Include array to have elements")
		}

		firstInclude, ok := includeSlice[0].(map[string]interface{})
		if !ok {
			t.Fatal("Expected first $Include element to be a map")
		}

		alias, ok := firstInclude["$Alias"]
		if !ok {
			t.Error("Expected $Alias in custom vocabulary $Include element")
		}
		// The alias should be "V1" (last segment of "Custom.Namespace.V1")
		if alias != "V1" {
			t.Errorf("Expected fallback alias 'V1', got %v", alias)
		}

		namespace, ok := firstInclude["$Namespace"]
		if !ok {
			t.Error("Expected $Namespace in custom vocabulary $Include element")
		}
		if namespace != "Custom.Namespace.V1" {
			t.Errorf("Expected $Namespace = 'Custom.Namespace.V1', got %v", namespace)
		}
	})

	t.Run("StandardVocabularies_UseMappedAliases", func(t *testing.T) {
		// Create an entity with standard vocabulary annotations
		meta, _ := metadata.AnalyzeEntity(AnnotatedTestProduct{})
		entitiesMetadata := map[string]*metadata.EntityMetadata{
			"AnnotatedTestProducts": meta,
		}

		handler := NewMetadataHandler(entitiesMetadata)

		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		references, ok := response["$Reference"]
		if !ok {
			t.Fatal("Expected $Reference section")
		}

		referencesMap := references.(map[string]interface{})
		
		// Test each standard vocabulary uses its mapped alias
		testCases := []struct {
			uri       string
			namespace string
			alias     string
		}{
			{
				uri:       "https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Core.V1.xml",
				namespace: "Org.OData.Core.V1",
				alias:     "Core",
			},
		}

		for _, tc := range testCases {
			ref, ok := referencesMap[tc.uri]
			if !ok {
				continue // Skip if this vocabulary is not present
			}

			refMap := ref.(map[string]interface{})
			includeSlice := refMap["$Include"].([]interface{})
			firstInclude := includeSlice[0].(map[string]interface{})

			if firstInclude["$Namespace"] != tc.namespace {
				t.Errorf("For URI %s: expected namespace %s, got %v", tc.uri, tc.namespace, firstInclude["$Namespace"])
			}

			if firstInclude["$Alias"] != tc.alias {
				t.Errorf("For URI %s: expected alias %s, got %v", tc.uri, tc.alias, firstInclude["$Alias"])
			}
		}
	})

	t.Run("MatchesDocumentationExample", func(t *testing.T) {
		// This test verifies the structure matches the example in annotations.md
		meta, _ := metadata.AnalyzeEntity(AnnotatedTestProduct{})
		entitiesMetadata := map[string]*metadata.EntityMetadata{
			"AnnotatedTestProducts": meta,
		}

		handler := NewMetadataHandler(entitiesMetadata)

		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		// Verify top-level structure
		if _, ok := response["$Version"]; !ok {
			t.Error("Expected $Version in response")
		}

		if _, ok := response["$EntityContainer"]; !ok {
			t.Error("Expected $EntityContainer in response")
		}

		references, ok := response["$Reference"]
		if !ok {
			t.Fatal("Expected $Reference in response")
		}

		// Verify $Reference is a map keyed by URI
		referencesMap, ok := references.(map[string]interface{})
		if !ok {
			t.Fatal("Expected $Reference to be a map keyed by URI")
		}

		// Each vocabulary reference should have the structure:
		// URI: {
		//   "$Include": [
		//     {
		//       "$Namespace": "...",
		//       "$Alias": "..."
		//     }
		//   ]
		// }
		for uri, ref := range referencesMap {
			refMap, ok := ref.(map[string]interface{})
			if !ok {
				t.Errorf("Reference at URI %s is not a map", uri)
				continue
			}

			includes, ok := refMap["$Include"]
			if !ok {
				t.Errorf("Reference at URI %s missing $Include", uri)
				continue
			}

			includeSlice, ok := includes.([]interface{})
			if !ok {
				t.Errorf("$Include at URI %s is not an array", uri)
				continue
			}

			if len(includeSlice) == 0 {
				t.Errorf("$Include at URI %s has no elements", uri)
				continue
			}

			for i, item := range includeSlice {
				includeMap, ok := item.(map[string]interface{})
				if !ok {
					t.Errorf("$Include[%d] at URI %s is not a map", i, uri)
					continue
				}

				if _, ok := includeMap["$Namespace"]; !ok {
					t.Errorf("$Include[%d] at URI %s missing $Namespace", i, uri)
				}

				if _, ok := includeMap["$Alias"]; !ok {
					t.Errorf("$Include[%d] at URI %s missing $Alias", i, uri)
				}
			}
		}
	})
}

// Test entities with annotations

// AnnotatedTestProduct has Core vocabulary annotations
type AnnotatedTestProduct struct {
	ID        uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name      string  `json:"Name" odata:"required,annotation:Core.Description=Product name"`
	CreatedAt string  `json:"CreatedAt" odata:"auto,annotation:Core.Computed"`
	Price     float64 `json:"Price"`
}

// MultiVocabTestProduct has annotations from multiple vocabularies
type MultiVocabTestProduct struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name" odata:"annotation:Core.Description=Product name"`
	IsAvailable bool    `json:"IsAvailable" odata:"annotation:Capabilities.ReadRestrictions"`
	Price       float64 `json:"Price"`
}

// CustomVocabTestProduct is used for testing custom vocabulary handling
type CustomVocabTestProduct struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name"`
}

// PlainTestProduct has no annotations whatsoever (no auto-generated ones)
type PlainTestProduct struct {
	ID    string  `json:"ID" odata:"key"` // String key, no autoIncrement
	Name  string  `json:"Name"`
	Price float64 `json:"Price"`
}
