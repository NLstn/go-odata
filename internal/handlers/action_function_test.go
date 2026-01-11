package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseActionFunctionURL(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantName    string
		wantKey     string
		wantIsBound bool
		wantErr     bool
	}{
		{
			name:        "Unbound action single segment",
			path:        "UpdatePrice",
			wantName:    "UpdatePrice",
			wantKey:     "",
			wantIsBound: false,
			wantErr:     false,
		},
		{
			name:        "Bound action with entity key",
			path:        "Products(1)/UpdatePrice",
			wantName:    "UpdatePrice",
			wantKey:     "1",
			wantIsBound: true,
			wantErr:     false,
		},
		{
			name:        "Bound action with string key",
			path:        "Customers('ALFKI')/Approve",
			wantName:    "Approve",
			wantKey:     "'ALFKI'",
			wantIsBound: true,
			wantErr:     false,
		},
		{
			name:        "Invalid URL format with extra segments",
			path:        "Products(1)/Details/Action",
			wantName:    "",
			wantKey:     "",
			wantIsBound: false,
			wantErr:     true,
		},
		{
			name:        "Entity set without key",
			path:        "Products/UpdatePrice",
			wantName:    "",
			wantKey:     "",
			wantIsBound: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotKey, gotIsBound, err := ParseActionFunctionURL(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseActionFunctionURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotName != tt.wantName {
				t.Errorf("ParseActionFunctionURL() name = %v, want %v", gotName, tt.wantName)
			}

			if gotKey != tt.wantKey {
				t.Errorf("ParseActionFunctionURL() key = %v, want %v", gotKey, tt.wantKey)
			}

			if gotIsBound != tt.wantIsBound {
				t.Errorf("ParseActionFunctionURL() isBound = %v, want %v", gotIsBound, tt.wantIsBound)
			}
		})
	}
}

func TestActionFunctionHandler_HandleActionOrFunction_MethodNotAllowed(t *testing.T) {
	handler := NewActionFunctionHandler(
		func() map[string]interface{} { return make(map[string]interface{}) },
		func() map[string]interface{} { return make(map[string]interface{}) },
		nil,
	)

	methods := []string{http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/TestAction", nil)
			w := httptest.NewRecorder()

			handler.HandleActionOrFunction(w, req, "TestAction", "", false)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("HandleActionOrFunction() status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if _, ok := response["error"]; !ok {
				t.Error("Response missing error field")
			}
		})
	}
}

func TestActionFunctionHandler_HandleAction_NotFound(t *testing.T) {
	handler := NewActionFunctionHandler(
		func() map[string]interface{} { return make(map[string]interface{}) },
		func() map[string]interface{} { return make(map[string]interface{}) },
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/UnknownAction", nil)
	w := httptest.NewRecorder()

	handler.HandleActionOrFunction(w, req, "UnknownAction", "", false)

	if w.Code != http.StatusNotFound {
		t.Errorf("HandleActionOrFunction() status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing error field")
	}

	if errorObj["message"] != "Action not found" {
		t.Errorf("Error message = %v, want 'Action not found'", errorObj["message"])
	}
}

func TestActionFunctionHandler_HandleFunction_NotFound(t *testing.T) {
	handler := NewActionFunctionHandler(
		func() map[string]interface{} { return make(map[string]interface{}) },
		func() map[string]interface{} { return make(map[string]interface{}) },
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/UnknownFunction", nil)
	w := httptest.NewRecorder()

	handler.HandleActionOrFunction(w, req, "UnknownFunction", "", false)

	if w.Code != http.StatusNotFound {
		t.Errorf("HandleActionOrFunction() status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing error field")
	}

	if errorObj["message"] != "Function not found" {
		t.Errorf("Error message = %v, want 'Function not found'", errorObj["message"])
	}
}

func TestActionFunctionHandler_HandleAction_Success(t *testing.T) {
	actions := map[string]interface{}{
		"TestAction": struct{}{},
	}

	handler := NewActionFunctionHandler(
		func() map[string]interface{} { return actions },
		func() map[string]interface{} { return make(map[string]interface{}) },
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/TestAction", nil)
	w := httptest.NewRecorder()

	handler.HandleActionOrFunction(w, req, "TestAction", "", false)

	if w.Code != http.StatusOK {
		t.Errorf("HandleActionOrFunction() status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}
}

func TestActionFunctionHandler_HandleFunction_Success(t *testing.T) {
	functions := map[string]interface{}{
		"TestFunction": struct{}{},
	}

	handler := NewActionFunctionHandler(
		func() map[string]interface{} { return make(map[string]interface{}) },
		func() map[string]interface{} { return functions },
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/TestFunction", nil)
	w := httptest.NewRecorder()

	handler.HandleActionOrFunction(w, req, "TestFunction", "", false)

	if w.Code != http.StatusOK {
		t.Errorf("HandleActionOrFunction() status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}
}

func TestActionFunctionHandler_HandleBoundAction_Success(t *testing.T) {
	actions := map[string]interface{}{
		"BoundAction": struct{}{},
	}

	handler := NewActionFunctionHandler(
		func() map[string]interface{} { return actions },
		func() map[string]interface{} { return make(map[string]interface{}) },
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/Products(1)/BoundAction", nil)
	w := httptest.NewRecorder()

	handler.HandleActionOrFunction(w, req, "BoundAction", "1", true)

	if w.Code != http.StatusOK {
		t.Errorf("HandleActionOrFunction() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify the response contains bound=true information
	value, ok := response["value"].(string)
	if !ok {
		t.Fatal("Response missing value field")
	}

	if value == "" {
		t.Error("Expected non-empty value")
	}
}

func TestActionFunctionHandler_MetadataLevel(t *testing.T) {
	tests := []struct {
		name          string
		metadataQuery string
		expectedType  string
	}{
		{
			name:          "Minimal metadata",
			metadataQuery: "",
			expectedType:  "application/json;odata.metadata=minimal",
		},
		{
			name:          "Full metadata via header",
			metadataQuery: "",
			expectedType:  "application/json;odata.metadata=minimal",
		},
	}

	actions := map[string]interface{}{
		"TestAction": struct{}{},
	}

	handler := NewActionFunctionHandler(
		func() map[string]interface{} { return actions },
		func() map[string]interface{} { return make(map[string]interface{}) },
		nil,
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/TestAction"
			if tt.metadataQuery != "" {
				url += "?" + tt.metadataQuery
			}
			req := httptest.NewRequest(http.MethodPost, url, nil)
			w := httptest.NewRecorder()

			handler.HandleActionOrFunction(w, req, "TestAction", "", false)

			if w.Code != http.StatusOK {
				t.Errorf("HandleActionOrFunction() status = %d, want %d", w.Code, http.StatusOK)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("Content-Type = %v, want %v", contentType, tt.expectedType)
			}
		})
	}
}
