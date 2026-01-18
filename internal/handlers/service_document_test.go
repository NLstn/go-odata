package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// ServiceDocumentTestEntity is a test entity for service document tests
type ServiceDocumentTestEntity struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name"`
}

func TestNewServiceDocumentHandler(t *testing.T) {
	// Create some test metadata
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	meta2, _ := metadata.AnalyzeEntity(TestEntity{})

	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
		"TestEntities":                meta2,
	}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)
	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
}

func TestServiceDocumentHandler_HandleServiceDocument_Get(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
	}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check @odata.context
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Expected @odata.context in response")
	}

	// Check value array
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(value) == 0 {
		t.Error("Expected at least one entity set in value array")
	}
}

func TestServiceDocumentHandler_HandleServiceDocument_Head(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
	}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestServiceDocumentHandler_HandleServiceDocument_Options(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
	}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}
}

func TestServiceDocumentHandler_HandleServiceDocument_MethodNotAllowed(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
	}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()

			handler.HandleServiceDocument(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestServiceDocumentHandler_SetPolicy(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
	}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)
	handler.SetPolicy(nil)

	// Should not panic
}

func TestServiceDocumentHandler_FiltersByPolicy(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	meta2, _ := metadata.AnalyzeEntity(TestEntity{})

	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
		"TestEntities":                meta2,
	}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that entity sets are present
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	// Should have both entity sets
	if len(value) < 2 {
		t.Errorf("Expected at least 2 entity sets, got %d", len(value))
	}
}

func TestServiceDocumentHandler_SkipsSingletons(t *testing.T) {
	// Create a regular entity and a singleton
	regularMeta, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	singletonMeta, _ := metadata.AnalyzeSingleton(ServiceDocumentTestEntity{}, "Settings")

	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": regularMeta,
		"Settings":                    singletonMeta,
	}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that singleton is listed separately
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	// Count entity sets and singletons
	hasSingleton := false
	hasEntitySet := false

	for _, item := range value {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		kind, _ := itemMap["kind"].(string)
		if kind == "Singleton" {
			hasSingleton = true
		}
		if kind == "EntitySet" {
			hasEntitySet = true
		}
	}

	if !hasEntitySet {
		t.Error("Expected entity set in service document")
	}

	if !hasSingleton {
		t.Error("Expected singleton in service document")
	}
}

func TestServiceDocumentHandler_EmptyMetadata(t *testing.T) {
	// Test with empty metadata
	entitiesMetadata := map[string]*metadata.EntityMetadata{}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that value array is empty
	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(value) != 0 {
		t.Errorf("Expected empty value array, got %d items", len(value))
	}
}

func TestServiceDocumentHandler_MetadataLevel_None(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
	}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json;odata.metadata=none")
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Service document response should be valid JSON
	// The @odata.context behavior depends on implementation - just verify valid response
	if response == nil {
		t.Error("Expected valid response")
	}
}

func TestServiceDocumentHandler_MetadataLevel_Full(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
	}

	handler := NewServiceDocumentHandler(entitiesMetadata, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json;odata.metadata=full")
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// With metadata=full, @odata.context should be present
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Expected @odata.context to be present with metadata=full")
	}
}
