package response

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestEntity struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestWriteODataCollection(t *testing.T) {
	data := []TestEntity{
		{ID: 1, Name: "Test 1"},
		{ID: 2, Name: "Test 2"},
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/Products", nil)
	w := httptest.NewRecorder()

	err := WriteODataCollection(w, req, "Products", data)
	if err != nil {
		t.Fatalf("WriteODataCollection() error = %v", err)
	}

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Check headers
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}

	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", odataVersion)
	}

	// Check response body
	var response ODataResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Context != "http://localhost:8080/$metadata#Products" {
		t.Errorf("Context = %v, want http://localhost:8080/$metadata#Products", response.Context)
	}

	// Check value array
	valueArray, ok := response.Value.([]interface{})
	if !ok {
		t.Fatal("Value is not an array")
	}

	if len(valueArray) != 2 {
		t.Errorf("len(Value) = %v, want 2", len(valueArray))
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	err := WriteError(w, http.StatusNotFound, "Not found", "Entity not found")
	if err != nil {
		t.Fatalf("WriteError() error = %v", err)
	}

	// Check status code
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	// Check headers
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorData, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error field is not an object")
	}

	if errorData["message"] != "Not found" {
		t.Errorf("error.message = %v, want Not found", errorData["message"])
	}

	if errorData["details"] != "Entity not found" {
		t.Errorf("error.details = %v, want Entity not found", errorData["details"])
	}
}

func TestWriteServiceDocument(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	w := httptest.NewRecorder()

	entitySets := []string{"Products", "Categories"}

	err := WriteServiceDocument(w, req, entitySets)
	if err != nil {
		t.Fatalf("WriteServiceDocument() error = %v", err)
	}

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Check response body
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["@odata.context"] != "http://localhost:8080/$metadata" {
		t.Errorf("@odata.context = %v, want http://localhost:8080/$metadata", response["@odata.context"])
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value) != 2 {
		t.Errorf("len(value) = %v, want 2", len(value))
	}
}

func TestBuildBaseURL(t *testing.T) {
	tests := []struct {
		name   string
		req    *http.Request
		want   string
	}{
		{
			name: "http request",
			req: &http.Request{
				Host: "localhost:8080",
				Header: http.Header{},
			},
			want: "http://localhost:8080",
		},
		{
			name: "https request",
			req: &http.Request{
				Host: "example.com",
				TLS:  &tls.ConnectionState{},
				Header: http.Header{},
			},
			want: "https://example.com",
		},
		{
			name: "request with X-Forwarded-Proto",
			req: &http.Request{
				Host: "example.com",
				Header: http.Header{
					"X-Forwarded-Proto": []string{"https"},
				},
			},
			want: "https://example.com",
		},
		{
			name: "request without host",
			req: &http.Request{
				Header: http.Header{},
			},
			want: "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildBaseURL(tt.req)
			if got != tt.want {
				t.Errorf("BuildBaseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseODataURL(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantEntitySet string
		wantEntityKey string
		wantErr       bool
	}{
		{
			name:          "collection",
			path:          "Products",
			wantEntitySet: "Products",
			wantEntityKey: "",
			wantErr:       false,
		},
		{
			name:          "entity with key",
			path:          "Products(1)",
			wantEntitySet: "Products",
			wantEntityKey: "1",
			wantErr:       false,
		},
		{
			name:          "entity with string key",
			path:          "Products('ABC')",
			wantEntitySet: "Products",
			wantEntityKey: "'ABC'",
			wantErr:       false,
		},
		{
			name:          "path with leading slash",
			path:          "/Products",
			wantEntitySet: "Products",
			wantEntityKey: "",
			wantErr:       false,
		},
		{
			name:          "empty path",
			path:          "",
			wantEntitySet: "",
			wantEntityKey: "",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEntitySet, gotEntityKey, err := ParseODataURL(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseODataURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotEntitySet != tt.wantEntitySet {
				t.Errorf("ParseODataURL() entitySet = %v, want %v", gotEntitySet, tt.wantEntitySet)
			}
			if gotEntityKey != tt.wantEntityKey {
				t.Errorf("ParseODataURL() entityKey = %v, want %v", gotEntityKey, tt.wantEntityKey)
			}
		})
	}
}
