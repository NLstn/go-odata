package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/version"
)

func TestWriteError_BasicError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Set default version in context
	ctx := version.WithVersion(req.Context(), version.Version{Major: 4, Minor: 1})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	err := WriteError(w, req, http.StatusBadRequest, "Bad Request", "Invalid input")
	if err != nil {
		t.Fatalf("WriteError failed: %v", err)
	}

	// Verify status code
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	// Verify Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}

	// Verify OData-Version header
	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.01" {
		t.Errorf("OData-Version = %v, want 4.01", odataVersion)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify error structure
	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error field missing or not an object")
	}

	if errorObj["code"] != "400" {
		t.Errorf("error.code = %v, want 400", errorObj["code"])
	}
	if errorObj["message"] != "Bad Request" {
		t.Errorf("error.message = %v, want Bad Request", errorObj["message"])
	}

	// Verify details
	details, ok := errorObj["details"].([]interface{})
	if !ok || len(details) == 0 {
		t.Fatal("error.details missing or empty")
	}

	firstDetail, ok := details[0].(map[string]interface{})
	if !ok {
		t.Fatal("error.details[0] is not an object")
	}
	if firstDetail["message"] != "Invalid input" {
		t.Errorf("error.details[0].message = %v, want Invalid input", firstDetail["message"])
	}
}

func TestWriteError_NoDetails(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err := WriteError(w, req, http.StatusNotFound, "Not Found", "")
	if err != nil {
		t.Fatalf("WriteError failed: %v", err)
	}

	// Verify status code
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorObj := response["error"].(map[string]interface{})

	// Verify no details when empty string passed
	if _, hasDetails := errorObj["details"]; hasDetails {
		t.Error("error.details should be omitted when empty")
	}
}

func TestWriteError_VariousStatusCodes(t *testing.T) {
	tests := []struct {
		code    int
		codeStr string
		message string
	}{
		{http.StatusBadRequest, "400", "Bad Request"},
		{http.StatusUnauthorized, "401", "Unauthorized"},
		{http.StatusForbidden, "403", "Forbidden"},
		{http.StatusNotFound, "404", "Not Found"},
		{http.StatusMethodNotAllowed, "405", "Method Not Allowed"},
		{http.StatusInternalServerError, "500", "Internal Server Error"},
	}

	for _, tc := range tests {
		t.Run(tc.message, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			err := WriteError(w, req, tc.code, tc.message, "")
			if err != nil {
				t.Fatalf("WriteError failed: %v", err)
			}

			if w.Code != tc.code {
				t.Errorf("Status = %v, want %v", w.Code, tc.code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			errorObj := response["error"].(map[string]interface{})
			if errorObj["code"] != tc.codeStr {
				t.Errorf("error.code = %v, want %v", errorObj["code"], tc.codeStr)
			}
			if errorObj["message"] != tc.message {
				t.Errorf("error.message = %v, want %v", errorObj["message"], tc.message)
			}
		})
	}
}

func TestWriteODataError_FullStructure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	odataErr := &ODataError{
		Code:    "CustomCode",
		Message: "Custom Message",
		Target:  "entity.field",
		Details: []ODataErrorDetail{
			{Code: "DetailCode1", Message: "Detail 1", Target: "field1"},
			{Code: "DetailCode2", Message: "Detail 2", Target: "field2"},
		},
		InnerError: &ODataInnerError{
			Message:    "Inner error message",
			TypeName:   "CustomException",
			StackTrace: "at line 1\nat line 2",
		},
	}

	err := WriteODataError(w, req, http.StatusBadRequest, odataErr)
	if err != nil {
		t.Fatalf("WriteODataError failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error field missing")
	}

	if errorObj["code"] != "CustomCode" {
		t.Errorf("error.code = %v, want CustomCode", errorObj["code"])
	}
	if errorObj["message"] != "Custom Message" {
		t.Errorf("error.message = %v, want Custom Message", errorObj["message"])
	}
	if errorObj["target"] != "entity.field" {
		t.Errorf("error.target = %v, want entity.field", errorObj["target"])
	}

	// Verify details
	details, ok := errorObj["details"].([]interface{})
	if !ok || len(details) != 2 {
		t.Fatalf("error.details should have 2 items, got %v", details)
	}

	// Verify inner error
	innerError, ok := errorObj["innererror"].(map[string]interface{})
	if !ok {
		t.Fatal("error.innererror missing")
	}
	if innerError["message"] != "Inner error message" {
		t.Errorf("error.innererror.message = %v, want 'Inner error message'", innerError["message"])
	}
	if innerError["type"] != "CustomException" {
		t.Errorf("error.innererror.type = %v, want CustomException", innerError["type"])
	}
}

func TestWriteErrorWithTarget_Basic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err := WriteErrorWithTarget(w, req, http.StatusBadRequest, "Validation Error", "Products(1)/Name", "Name is required")
	if err != nil {
		t.Fatalf("WriteErrorWithTarget failed: %v", err)
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorObj := response["error"].(map[string]interface{})
	if errorObj["target"] != "Products(1)/Name" {
		t.Errorf("error.target = %v, want Products(1)/Name", errorObj["target"])
	}
	if errorObj["message"] != "Validation Error" {
		t.Errorf("error.message = %v, want Validation Error", errorObj["message"])
	}

	details := errorObj["details"].([]interface{})
	if len(details) != 1 {
		t.Fatalf("Expected 1 detail, got %d", len(details))
	}

	detail := details[0].(map[string]interface{})
	if detail["target"] != "Products(1)/Name" {
		t.Errorf("detail.target = %v, want Products(1)/Name", detail["target"])
	}
	if detail["message"] != "Name is required" {
		t.Errorf("detail.message = %v, want 'Name is required'", detail["message"])
	}
}

func TestWriteErrorWithTarget_NoDetails(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	err := WriteErrorWithTarget(w, req, http.StatusNotFound, "Not Found", "Products(999)", "")
	if err != nil {
		t.Fatalf("WriteErrorWithTarget failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorObj := response["error"].(map[string]interface{})
	if errorObj["target"] != "Products(999)" {
		t.Errorf("error.target = %v, want Products(999)", errorObj["target"])
	}

	// Verify no details when empty string passed
	if _, hasDetails := errorObj["details"]; hasDetails {
		t.Error("error.details should be omitted when empty")
	}
}

func TestWriteServiceDocument_Basic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	w := httptest.NewRecorder()

	entitySets := []string{"Products", "Categories"}
	singletons := []string{"Config"}

	err := WriteServiceDocument(w, req, entitySets, singletons)
	if err != nil {
		t.Fatalf("WriteServiceDocument failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify @odata.context
	context := response["@odata.context"].(string)
	if context != "http://example.com/$metadata" {
		t.Errorf("@odata.context = %v, want http://example.com/$metadata", context)
	}

	// Verify value array
	value := response["value"].([]interface{})
	if len(value) != 3 {
		t.Errorf("Expected 3 items in value, got %d", len(value))
	}

	// Verify entity sets and singleton
	foundProducts := false
	foundCategories := false
	foundConfig := false

	for _, item := range value {
		entity := item.(map[string]interface{})
		name := entity["name"].(string)
		kind := entity["kind"].(string)

		switch name {
		case "Products":
			foundProducts = true
			if kind != "EntitySet" {
				t.Errorf("Products kind = %v, want EntitySet", kind)
			}
		case "Categories":
			foundCategories = true
			if kind != "EntitySet" {
				t.Errorf("Categories kind = %v, want EntitySet", kind)
			}
		case "Config":
			foundConfig = true
			if kind != "Singleton" {
				t.Errorf("Config kind = %v, want Singleton", kind)
			}
		}
	}

	if !foundProducts {
		t.Error("Products not found in service document")
	}
	if !foundCategories {
		t.Error("Categories not found in service document")
	}
	if !foundConfig {
		t.Error("Config singleton not found in service document")
	}
}

func TestWriteServiceDocument_HEAD(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, "http://example.com/", nil)
	w := httptest.NewRecorder()

	err := WriteServiceDocument(w, req, []string{"Products"}, nil)
	if err != nil {
		t.Fatalf("WriteServiceDocument failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// For HEAD, body should be empty
	if w.Body.Len() != 0 {
		t.Errorf("Body should be empty for HEAD, got %d bytes", w.Body.Len())
	}

	// But Content-Length should be set
	contentLength := w.Header().Get("Content-Length")
	if contentLength == "" || contentLength == "0" {
		t.Error("Content-Length should be set for HEAD request")
	}
}

func TestWriteServiceDocument_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	w := httptest.NewRecorder()

	err := WriteServiceDocument(w, req, nil, nil)
	if err != nil {
		t.Fatalf("WriteServiceDocument failed: %v", err)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response body as JSON: %v", err)
	}

	value := response["value"].([]interface{})
	if len(value) != 0 {
		t.Errorf("Expected empty value array, got %d items", len(value))
	}
}

func TestWriteServiceDocument_InvalidAcceptHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Accept", "application/xml")
	w := httptest.NewRecorder()

	err := WriteServiceDocument(w, req, []string{"Products"}, nil)
	if err != nil {
		t.Fatalf("WriteServiceDocument failed: %v", err)
	}

	// Should return 406 Not Acceptable
	if w.Code != http.StatusNotAcceptable {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotAcceptable)
	}
}

func TestWriteError_RespectsVersionNegotiation_40(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("OData-MaxVersion", "4.0")
	
	// Simulate version negotiation middleware
	ctx := req.Context()
	ctx = version.WithVersion(ctx, version.Version{Major: 4, Minor: 0})
	req = req.WithContext(ctx)
	
	w := httptest.NewRecorder()

	err := WriteError(w, req, http.StatusBadRequest, "Bad Request", "Test error")
	if err != nil {
		t.Fatalf("WriteError failed: %v", err)
	}

	// Verify OData-Version header matches negotiated version
	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", odataVersion)
	}
}

func TestWriteError_RespectsVersionNegotiation_401(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	
	// Simulate version negotiation middleware
	ctx := req.Context()
	ctx = version.WithVersion(ctx, version.Version{Major: 4, Minor: 1})
	req = req.WithContext(ctx)
	
	w := httptest.NewRecorder()

	err := WriteError(w, req, http.StatusNotFound, "Not Found", "Test error")
	if err != nil {
		t.Fatalf("WriteError failed: %v", err)
	}

	// Verify OData-Version header matches negotiated version
	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.01" {
		t.Errorf("OData-Version = %v, want 4.01", odataVersion)
	}
}

func TestWriteODataError_RespectsVersionNegotiation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("OData-MaxVersion", "4.0")
	
	// Simulate version negotiation middleware
	ctx := req.Context()
	ctx = version.WithVersion(ctx, version.Version{Major: 4, Minor: 0})
	req = req.WithContext(ctx)
	
	w := httptest.NewRecorder()

	odataErr := &ODataError{
		Code:    "TestCode",
		Message: "Test Message",
	}

	err := WriteODataError(w, req, http.StatusBadRequest, odataErr)
	if err != nil {
		t.Fatalf("WriteODataError failed: %v", err)
	}

	// Verify OData-Version header matches negotiated version
	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", odataVersion)
	}
}
