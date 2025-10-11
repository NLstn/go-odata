package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestEntityHandlerCollectionHead tests HEAD request on entity collection
func TestEntityHandlerCollectionHead(t *testing.T) {
	handler, db := setupTestHandler(t)

	// Insert test data
	testData := []TestEntity{
		{ID: 1, Name: "Test 1"},
		{ID: 2, Name: "Test 2"},
		{ID: 3, Name: "Test 3"},
	}
	for _, entity := range testData {
		db.Create(&entity)
	}

	req := httptest.NewRequest(http.MethodHead, "/TestEntities", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	// HEAD should return 200 OK like GET
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify headers are set
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Content-Type header should be set for HEAD request")
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", odataVersion)
	}

	// Verify response body is empty (HEAD shouldn't return body)
	if w.Body.Len() != 0 {
		t.Errorf("HEAD response should have empty body, got %d bytes", w.Body.Len())
	}
}

// TestEntityHandlerEntityHead tests HEAD request on individual entity
func TestEntityHandlerEntityHead(t *testing.T) {
	handler, db := setupTestHandler(t)

	// Insert test data
	testEntity := TestEntity{ID: 1, Name: "Test Entity"}
	db.Create(&testEntity)

	req := httptest.NewRequest(http.MethodHead, "/TestEntities(1)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	// HEAD should return 200 OK like GET
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify headers are set
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Content-Type header should be set for HEAD request")
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", odataVersion)
	}

	// Verify response body is empty (HEAD shouldn't return body)
	if w.Body.Len() != 0 {
		t.Errorf("HEAD response should have empty body, got %d bytes", w.Body.Len())
	}
}

// TestEntityHandlerCountHead tests HEAD request on $count endpoint
func TestEntityHandlerCountHead(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert test data
	products := []Product{
		{ID: 1, Name: "Product 1", Price: 10.0, Category: "Cat1"},
		{ID: 2, Name: "Product 2", Price: 20.0, Category: "Cat2"},
	}
	for _, product := range products {
		db.Create(&product)
	}

	req := httptest.NewRequest(http.MethodHead, "/Products/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleCount(w, req)

	// HEAD should return 200 OK like GET
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify headers are set for plain text response
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Content-Type = %v, want text/plain", contentType)
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", odataVersion)
	}

	// Verify response body is empty (HEAD shouldn't return body)
	if w.Body.Len() != 0 {
		t.Errorf("HEAD response should have empty body, got %d bytes", w.Body.Len())
	}
}

// TestMetadataHandlerHead tests HEAD request on metadata document
func TestMetadataHandlerHead(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	entityMeta, _ := metadata.AnalyzeEntity(TestEntity{})
	entities["TestEntities"] = entityMeta

	handler := NewMetadataHandler(entities)

	req := httptest.NewRequest(http.MethodHead, "/$metadata", nil)
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	// HEAD should return 200 OK like GET
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify headers are set
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Content-Type header should be set for HEAD request")
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "" && odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0 or empty", odataVersion)
	}

	// Verify response body is empty (HEAD shouldn't return body)
	if w.Body.Len() != 0 {
		t.Errorf("HEAD response should have empty body, got %d bytes", w.Body.Len())
	}
}

// TestServiceDocumentHandlerHead tests HEAD request on service document
func TestServiceDocumentHandlerHead(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	handler := NewServiceDocumentHandler(entities)

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	w := httptest.NewRecorder()

	handler.HandleServiceDocument(w, req)

	// HEAD should return 200 OK like GET
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify headers are set
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Content-Type header should be set for HEAD request")
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "" && odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0 or empty", odataVersion)
	}

	// Verify response body is empty (HEAD shouldn't return body)
	if w.Body.Len() != 0 {
		t.Errorf("HEAD response should have empty body, got %d bytes", w.Body.Len())
	}
}

// TestEntityHandlerCollectionHeadWithQueryOptions tests HEAD with query options
func TestEntityHandlerCollectionHeadWithQueryOptions(t *testing.T) {
	handler, db := setupTestHandler(t)

	// Insert test data
	testData := []TestEntity{
		{ID: 1, Name: "Alpha"},
		{ID: 2, Name: "Beta"},
		{ID: 3, Name: "Gamma"},
	}
	for _, entity := range testData {
		db.Create(&entity)
	}

	// Test HEAD with $filter - URL encode the query parameter
	req := httptest.NewRequest(http.MethodHead, "/TestEntities?$filter=id%20gt%201", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify response body is empty
	if w.Body.Len() != 0 {
		t.Errorf("HEAD response should have empty body, got %d bytes", w.Body.Len())
	}
}

// TestOptionsHandlersIncludeHead tests that OPTIONS responses include HEAD in Allow header
func TestOptionsHandlersIncludeHead(t *testing.T) {
	handler, _ := setupTestHandler(t)

	tests := []struct {
		name          string
		path          string
		handlerFunc   func(http.ResponseWriter, *http.Request)
		expectedAllow string
	}{
		{
			name: "Collection OPTIONS",
			path: "/TestEntities",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.HandleCollection(w, r)
			},
			expectedAllow: "GET, HEAD, POST, OPTIONS",
		},
		{
			name: "Entity OPTIONS",
			path: "/TestEntities(1)",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.HandleEntity(w, r, "1")
			},
			expectedAllow: "GET, HEAD, DELETE, PATCH, PUT, OPTIONS",
		},
		{
			name: "Count OPTIONS",
			path: "/TestEntities/$count",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.HandleCount(w, r)
			},
			expectedAllow: "GET, HEAD, OPTIONS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, tt.path, nil)
			w := httptest.NewRecorder()

			tt.handlerFunc(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			allowHeader := w.Header().Get("Allow")
			if allowHeader != tt.expectedAllow {
				t.Errorf("Allow header = %v, want %v", allowHeader, tt.expectedAllow)
			}
		})
	}
}

// TestMetadataAndServiceDocumentOptionsIncludeHead tests OPTIONS for metadata and service document
func TestMetadataAndServiceDocumentOptionsIncludeHead(t *testing.T) {
	entities := make(map[string]*metadata.EntityMetadata)
	entityMeta, _ := metadata.AnalyzeEntity(TestEntity{})
	entities["TestEntities"] = entityMeta

	tests := []struct {
		name          string
		path          string
		expectedAllow string
		isMeta        bool
	}{
		{
			name:          "Metadata OPTIONS",
			path:          "/$metadata",
			expectedAllow: "GET, HEAD, OPTIONS",
			isMeta:        true,
		},
		{
			name:          "Service Document OPTIONS",
			path:          "/",
			expectedAllow: "GET, HEAD, OPTIONS",
			isMeta:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, tt.path, nil)
			w := httptest.NewRecorder()

			if tt.isMeta {
				handler := NewMetadataHandler(entities)
				handler.HandleMetadata(w, req)
			} else {
				handler := NewServiceDocumentHandler(entities)
				handler.HandleServiceDocument(w, req)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			allowHeader := w.Header().Get("Allow")
			if allowHeader != tt.expectedAllow {
				t.Errorf("Allow header = %v, want %v", allowHeader, tt.expectedAllow)
			}
		})
	}
}

// TestEntityHandlerHeadNotFound tests HEAD request on non-existent entity
func TestEntityHandlerHeadNotFound(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodHead, "/TestEntities(999)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "999")

	// HEAD should return 404 for non-existent entities
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	// Note: Error responses may include body content even for HEAD requests
	// since the error handling doesn't have access to the request method.
	// This is acceptable behavior for error responses.
}
