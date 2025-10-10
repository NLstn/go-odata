package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

type CompositeKeyEntity struct {
	ID1  int    `json:"id1" odata:"key"`
	ID2  int    `json:"id2" odata:"key"`
	Name string `json:"name"`
}

func TestMetadataHandlerJSONWithCompositeKey(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)

	entityMeta, err := metadata.AnalyzeEntity(CompositeKeyEntity{})
	if err != nil {
		t.Fatalf("Error analyzing entity: %v", err)
	}

	entities["CompositeKeyEntities"] = entityMeta

	handler := NewMetadataHandler(entities)

	t.Run("JSON format with composite key via query parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", contentType)
		}

		// Parse JSON response
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		// Validate JSON structure
		if version, ok := response["$Version"].(string); !ok || version != "4.0" {
			t.Errorf("Expected $Version to be 4.0, got %v", response["$Version"])
		}

		odataService, ok := response["ODataService"].(map[string]interface{})
		if !ok {
			t.Fatal("ODataService not found in response")
		}

		// Check for entity type
		entityType, ok := odataService["CompositeKeyEntity"].(map[string]interface{})
		if !ok {
			t.Fatal("CompositeKeyEntity not found in metadata")
		}

		// Validate entity type structure
		if kind, ok := entityType["$Kind"].(string); !ok || kind != "EntityType" {
			t.Errorf("Expected $Kind to be EntityType, got %v", entityType["$Kind"])
		}

		// Check for composite key property - should have both id1 and id2
		key, ok := entityType["$Key"].([]interface{})
		if !ok {
			t.Fatal("$Key not found or not an array")
		}

		if len(key) != 2 {
			t.Errorf("Expected 2 key properties, got %d", len(key))
		}

		// Verify both keys are present
		keyNames := make(map[string]bool)
		for _, k := range key {
			if keyName, ok := k.(string); ok {
				keyNames[keyName] = true
			}
		}

		if !keyNames["id1"] {
			t.Error("id1 not found in $Key")
		}
		if !keyNames["id2"] {
			t.Error("id2 not found in $Key")
		}

		// Check for properties
		if _, ok := entityType["id1"]; !ok {
			t.Error("id1 property not found in entity type")
		}
		if _, ok := entityType["id2"]; !ok {
			t.Error("id2 property not found in entity type")
		}
		if _, ok := entityType["name"]; !ok {
			t.Error("name property not found in entity type")
		}

		// Check for container
		container, ok := odataService["Container"].(map[string]interface{})
		if !ok {
			t.Fatal("Container not found in metadata")
		}

		// Validate container structure
		if kind, ok := container["$Kind"].(string); !ok || kind != "EntityContainer" {
			t.Errorf("Expected $Kind to be EntityContainer, got %v", container["$Kind"])
		}

		// Check for entity set
		if _, ok := container["CompositeKeyEntities"]; !ok {
			t.Error("CompositeKeyEntities entity set not found in container")
		}
	})

	t.Run("JSON format with composite key via Accept header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", contentType)
		}

		// Parse JSON response
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		// Basic validation
		if _, ok := response["ODataService"]; !ok {
			t.Error("ODataService not found in response")
		}
	})
}
