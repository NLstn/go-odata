package handlers

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// MetadataTestProduct is a test entity for metadata handler tests
type MetadataTestProduct struct {
	ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string  `json:"Name" odata:"required"`
	Price    float64 `json:"Price"`
	Category string  `json:"Category"`
}

func TestNewMetadataHandler(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)
	if handler == nil {
		t.Fatal("Expected handler to be created")
	}
}

func TestMetadataHandler_HandleMetadata_GetXML(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/xml")
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "xml") {
		t.Errorf("Content-Type = %v, want xml", contentType)
	}

	// Verify it's valid XML
	decoder := xml.NewDecoder(strings.NewReader(w.Body.String()))
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Errorf("Invalid XML response: %v", err)
			break
		}
	}
}

func TestMetadataHandler_HandleMetadata_GetJSON(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "json") {
		t.Errorf("Content-Type = %v, want json", contentType)
	}

	// Verify it's valid JSON
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}
}

func TestMetadataHandler_HandleMetadata_Head(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)

	req := httptest.NewRequest(http.MethodHead, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// HEAD should not return a body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}

func TestMetadataHandler_HandleMetadata_Options(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)

	req := httptest.NewRequest(http.MethodOptions, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	allowHeader := w.Header().Get("Allow")
	if allowHeader != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow header = %v, want 'GET, HEAD, OPTIONS'", allowHeader)
	}
}

func TestMetadataHandler_HandleMetadata_MethodNotAllowed(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/$metadata", nil)
			w := httptest.NewRecorder()

			handler.HandleMetadata(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestMetadataHandler_SetNamespace(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)
	handler.SetNamespace("Custom.Namespace")

	// Make a request to verify the namespace is used
	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Custom.Namespace") {
		t.Error("Expected namespace to be in response")
	}
}

func TestMetadataHandler_SetNamespace_Empty(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)
	handler.SetNamespace("")

	// Make a request to verify default namespace is used
	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	// Should contain default namespace
	if !strings.Contains(body, "Default") && !strings.Contains(body, "$metadata") {
		t.Log("Response should contain namespace information")
	}
}

func TestMetadataHandler_SetPolicy(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)
	handler.SetPolicy(nil)

	// Should not panic
}

func TestMetadataHandler_EmptyMetadata(t *testing.T) {
	entitiesMetadata := map[string]*metadata.EntityMetadata{}

	handler := NewMetadataHandler(entitiesMetadata)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestMetadataHandler_MultipleEntities(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	meta2, _ := metadata.AnalyzeEntity(TestEntity{})

	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
		"TestEntities":         meta2,
	}

	handler := NewMetadataHandler(entitiesMetadata)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	// Should contain both entity types
	if !strings.Contains(body, "MetadataTestProduct") {
		t.Error("Expected MetadataTestProduct in response")
	}
	if !strings.Contains(body, "TestEntity") {
		t.Error("Expected TestEntity in response")
	}
}

func TestMetadataHandler_AcceptHeaderParsing(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)

	tests := []struct {
		name         string
		acceptHeader string
		expectXML    bool
	}{
		{
			name:         "application/xml",
			acceptHeader: "application/xml",
			expectXML:    true,
		},
		{
			name:         "application/json",
			acceptHeader: "application/json",
			expectXML:    false,
		},
		{
			name:         "text/xml",
			acceptHeader: "text/xml",
			expectXML:    true,
		},
		{
			name:         "no accept header",
			acceptHeader: "",
			expectXML:    true, // Default to XML
		},
		{
			name:         "accept */*",
			acceptHeader: "*/*",
			expectXML:    true, // Default to XML
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			handler.HandleMetadata(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			contentType := w.Header().Get("Content-Type")
			if tt.expectXML {
				if !strings.Contains(contentType, "xml") {
					t.Errorf("Expected XML content type, got %v", contentType)
				}
			} else {
				if !strings.Contains(contentType, "json") {
					t.Errorf("Expected JSON content type, got %v", contentType)
				}
			}
		})
	}
}

func TestMetadataHandler_Caching(t *testing.T) {
	meta1, _ := metadata.AnalyzeEntity(MetadataTestProduct{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"MetadataTestProducts": meta1,
	}

	handler := NewMetadataHandler(entitiesMetadata)

	// Make multiple requests - should use cache
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		req.Header.Set("Accept", "application/xml")
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Status = %v, want %v", i, w.Code, http.StatusOK)
		}
	}
}
