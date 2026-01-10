package odata_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// VersionTestProduct is a test entity for version header tests
type VersionTestProduct struct {
	ID    int     `json:"id" gorm:"primarykey;autoIncrement" odata:"key"`
	Name  string  `json:"name" odata:"required"`
	Price float64 `json:"price"`
}

func setupVersionTestService(t *testing.T) *odata.Service {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&VersionTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create some test data
	products := []VersionTestProduct{
		{Name: "Laptop", Price: 999.99},
		{Name: "Mouse", Price: 29.99},
	}
	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(VersionTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service
}

// TestODataMaxVersion_NoHeader tests that requests without OData-MaxVersion header are accepted
func TestODataMaxVersion_NoHeader(t *testing.T) {
	service := setupVersionTestService(t)

	testCases := []struct {
		name string
		path string
	}{
		{"Service document", "/"},
		{"Metadata", "/$metadata"},
		{"Entity collection", "/VersionTestProducts"},
		{"Single entity", "/VersionTestProducts(1)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			// Should not be rejected (status should not be 406)
			if w.Code == http.StatusNotAcceptable {
				t.Errorf("Request rejected with 406, but no OData-MaxVersion header was provided")
			}
		})
	}
}

// TestODataMaxVersion_Version4_0 tests that OData-MaxVersion: 4.0 is accepted
func TestODataMaxVersion_Version4_0(t *testing.T) {
	service := setupVersionTestService(t)

	testCases := []struct {
		name string
		path string
	}{
		{"Service document", "/"},
		{"Metadata", "/$metadata"},
		{"Entity collection", "/VersionTestProducts"},
		{"Single entity", "/VersionTestProducts(1)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("OData-MaxVersion", "4.0")
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			// Should not be rejected (status should not be 406)
			if w.Code == http.StatusNotAcceptable {
				t.Errorf("Request with OData-MaxVersion: 4.0 was rejected with 406, but should be accepted")
			}
		})
	}
}

// TestODataMaxVersion_Version4_01 tests that OData-MaxVersion: 4.01 is accepted
func TestODataMaxVersion_Version4_01(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/VersionTestProducts", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should not be rejected (status should not be 406)
	if w.Code == http.StatusNotAcceptable {
		t.Errorf("Request with OData-MaxVersion: 4.01 was rejected with 406, but should be accepted")
	}
}

// TestODataMaxVersion_Version5 tests that OData-MaxVersion: 5.0 is accepted
func TestODataMaxVersion_Version5(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/VersionTestProducts", nil)
	req.Header.Set("OData-MaxVersion", "5.0")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should not be rejected (status should not be 406)
	if w.Code == http.StatusNotAcceptable {
		t.Errorf("Request with OData-MaxVersion: 5.0 was rejected with 406, but should be accepted")
	}
}

// TestODataMaxVersion_Version3_0_Rejected tests that OData-MaxVersion: 3.0 is rejected
func TestODataMaxVersion_Version3_0_Rejected(t *testing.T) {
	service := setupVersionTestService(t)

	testCases := []struct {
		name string
		path string
	}{
		{"Service document", "/"},
		{"Metadata", "/$metadata"},
		{"Entity collection", "/VersionTestProducts"},
		{"Single entity", "/VersionTestProducts(1)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("OData-MaxVersion", "3.0")
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			// Should be rejected with 406 Not Acceptable
			if w.Code != http.StatusNotAcceptable {
				t.Errorf("Expected status 406 Not Acceptable, got %d", w.Code)
			}

			// Verify error response structure
			var errorResponse map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			errorObj, ok := errorResponse["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Error response does not have 'error' object")
			}

			message, ok := errorObj["message"].(string)
			if !ok {
				t.Fatal("Error response does not have 'message' field")
			}

			if !strings.Contains(strings.ToLower(message), "version") {
				t.Errorf("Error message does not mention version: %s", message)
			}
		})
	}
}

// TestODataMaxVersion_Version2_0_Rejected tests that OData-MaxVersion: 2.0 is rejected
func TestODataMaxVersion_Version2_0_Rejected(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/VersionTestProducts", nil)
	req.Header.Set("OData-MaxVersion", "2.0")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should be rejected with 406 Not Acceptable
	if w.Code != http.StatusNotAcceptable {
		t.Errorf("Expected status 406 Not Acceptable, got %d", w.Code)
	}
}

// TestODataMaxVersion_Version1_0_Rejected tests that OData-MaxVersion: 1.0 is rejected
func TestODataMaxVersion_Version1_0_Rejected(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/VersionTestProducts", nil)
	req.Header.Set("OData-MaxVersion", "1.0")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should be rejected with 406 Not Acceptable
	if w.Code != http.StatusNotAcceptable {
		t.Errorf("Expected status 406 Not Acceptable, got %d", w.Code)
	}
}

// TestODataMaxVersion_InvalidFormat tests that invalid version formats are handled gracefully
func TestODataMaxVersion_InvalidFormat(t *testing.T) {
	service := setupVersionTestService(t)

	testCases := []struct {
		name           string
		maxVersion     string
		expectedStatus int // 0 = any success (200/201), 406 = Not Acceptable
	}{
		{"Empty string", "", 0},                               // Empty should be treated as no header (success)
		{"Invalid number", "abc", 0},                          // Invalid format → ignored per OData spec, treated as no header (success)
		{"Just major version", "4", 0},                        // "4" should be accepted as 4.0 (success)
		{"Major only below 4", "3", http.StatusNotAcceptable}, // "3" → 406 Not Acceptable (valid format, unsupported version)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/VersionTestProducts", nil)
			if tc.maxVersion != "" {
				req.Header.Set("OData-MaxVersion", tc.maxVersion)
			}
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if tc.expectedStatus != 0 {
				if w.Code != tc.expectedStatus {
					t.Errorf("Expected status %d for version '%s', got %d", tc.expectedStatus, tc.maxVersion, w.Code)
				}
			} else {
				// Success cases - should not be rejected
				if w.Code >= 400 {
					t.Errorf("Version '%s' should not be rejected, got status %d", tc.maxVersion, w.Code)
				}
			}
		})
	}
}

// TestODataMaxVersion_ErrorResponseStructure tests the error response format
func TestODataMaxVersion_ErrorResponseStructure(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/VersionTestProducts", nil)
	req.Header.Set("OData-MaxVersion", "3.0")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotAcceptable {
		t.Fatalf("Expected status 406, got %d", w.Code)
	}

	// Verify OData-compliant error structure
	var errorResponse map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	// Check that it has the standard OData error structure
	errorObj, ok := errorResponse["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Error response missing 'error' object")
	}

	// Check for required fields
	if _, ok := errorObj["code"]; !ok {
		t.Error("Error object missing 'code' field")
	}

	if _, ok := errorObj["message"]; !ok {
		t.Error("Error object missing 'message' field")
	}

	// Verify details array if present
	if details, ok := errorObj["details"].([]interface{}); ok && len(details) > 0 {
		firstDetail := details[0].(map[string]interface{})
		if _, ok := firstDetail["message"]; !ok {
			t.Error("Error detail missing 'message' field")
		}
	}
}

// TestODataMaxVersion_WithPOSTRequest tests version validation with POST requests
func TestODataMaxVersion_WithPOSTRequest(t *testing.T) {
	service := setupVersionTestService(t)

	// Test that POST is also validated
	newProduct := map[string]interface{}{
		"name":  "Keyboard",
		"price": 79.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/VersionTestProducts", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OData-MaxVersion", "3.0")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should be rejected with 406 Not Acceptable
	if w.Code != http.StatusNotAcceptable {
		t.Errorf("POST request with OData-MaxVersion: 3.0 should be rejected with 406, got %d", w.Code)
	}
}

// TestODataMaxVersion_WithDELETERequest tests version validation with DELETE requests
func TestODataMaxVersion_WithDELETERequest(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodDelete, "/VersionTestProducts(1)", nil)
	req.Header.Set("OData-MaxVersion", "3.0")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should be rejected with 406 Not Acceptable
	if w.Code != http.StatusNotAcceptable {
		t.Errorf("DELETE request with OData-MaxVersion: 3.0 should be rejected with 406, got %d", w.Code)
	}
}

// TestODataVersionNegotiation_Returns4_0_WhenClientRequests4_0 verifies version negotiation
func TestODataVersionNegotiation_Returns4_0_WhenClientRequests4_0(t *testing.T) {
	service := setupVersionTestService(t)

	testCases := []struct {
		name string
		path string
	}{
		{"Service document", "/"},
		{"Metadata XML", "/$metadata"},
		{"Entity collection", "/VersionTestProducts"},
		{"Single entity", "/VersionTestProducts(1)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("OData-MaxVersion", "4.0")
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			// Verify response has OData-Version: 4.0
			//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
			odataVersionValues := w.Header()["OData-Version"]
			if len(odataVersionValues) == 0 || odataVersionValues[0] != "4.0" {
				t.Errorf("Expected OData-Version: 4.0, got: %v", odataVersionValues)
			}
		})
	}
}

// TestODataVersionNegotiation_Returns4_01_WhenClientRequests4_01 verifies version negotiation
func TestODataVersionNegotiation_Returns4_01_WhenClientRequests4_01(t *testing.T) {
	service := setupVersionTestService(t)

	testCases := []struct {
		name string
		path string
	}{
		{"Service document", "/"},
		{"Metadata XML", "/$metadata"},
		{"Entity collection", "/VersionTestProducts"},
		{"Single entity", "/VersionTestProducts(1)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("OData-MaxVersion", "4.01")
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			// Verify response has OData-Version: 4.01
			//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
			odataVersionValues := w.Header()["OData-Version"]
			if len(odataVersionValues) == 0 || odataVersionValues[0] != "4.01" {
				t.Errorf("Expected OData-Version: 4.01, got: %v", odataVersionValues)
			}
		})
	}
}

// TestODataVersionNegotiation_Returns4_01_WhenNoMaxVersion verifies default behavior
func TestODataVersionNegotiation_Returns4_01_WhenNoMaxVersion(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/VersionTestProducts", nil)
	// No OData-MaxVersion header
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify response has OData-Version: 4.01 (latest supported)
	//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
	odataVersionValues := w.Header()["OData-Version"]
	if len(odataVersionValues) == 0 || odataVersionValues[0] != "4.01" {
		t.Errorf("Expected OData-Version: 4.01 (default), got: %v", odataVersionValues)
	}
}

// TestODataVersionNegotiation_Returns4_01_WhenClientRequests5_0 verifies version negotiation
func TestODataVersionNegotiation_Returns4_01_WhenClientRequests5_0(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/VersionTestProducts", nil)
	req.Header.Set("OData-MaxVersion", "5.0")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify response has OData-Version: 4.01 (highest supported <= 5.0)
	//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
	odataVersionValues := w.Header()["OData-Version"]
	if len(odataVersionValues) == 0 || odataVersionValues[0] != "4.01" {
		t.Errorf("Expected OData-Version: 4.01 (highest supported), got: %v", odataVersionValues)
	}
}

// TestMetadataXML_VersionAttribute_4_0 verifies metadata XML contains correct version attribute
func TestMetadataXML_VersionAttribute_4_0(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("OData-MaxVersion", "4.0")
	req.Header.Set("Accept", "application/xml")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `Version="4.0"`) {
		t.Errorf("Metadata XML should contain Version=\"4.0\", got: %s", body)
	}
}

// TestMetadataXML_VersionAttribute_4_01 verifies metadata XML contains correct version attribute
func TestMetadataXML_VersionAttribute_4_01(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	req.Header.Set("Accept", "application/xml")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `Version="4.01"`) {
		t.Errorf("Metadata XML should contain Version=\"4.01\", got: %s", body)
	}
}

// TestMetadataJSON_VersionProperty_4_0 verifies metadata JSON contains correct version property
func TestMetadataJSON_VersionProperty_4_0(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("OData-MaxVersion", "4.0")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &metadata); err != nil {
		t.Fatalf("Failed to parse JSON metadata: %v", err)
	}

	version, ok := metadata["$Version"]
	if !ok {
		t.Errorf("Metadata JSON should contain $Version property")
	} else if version != "4.0" {
		t.Errorf("Expected $Version: \"4.0\", got: %v", version)
	}
}

// TestMetadataJSON_VersionProperty_4_01 verifies metadata JSON contains correct version property
func TestMetadataJSON_VersionProperty_4_01(t *testing.T) {
	service := setupVersionTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &metadata); err != nil {
		t.Fatalf("Failed to parse JSON metadata: %v", err)
	}

	version, ok := metadata["$Version"]
	if !ok {
		t.Errorf("Metadata JSON should contain $Version property")
	} else if version != "4.01" {
		t.Errorf("Expected $Version: \"4.01\", got: %v", version)
	}
}

// TestMetadataHandler_ConcurrentAccessWithNamespaceChange tests concurrent metadata requests
// with different OData-MaxVersion headers while namespace is being changed.
// This verifies that the locking in MetadataHandler is correct and prevents race conditions.
func TestMetadataHandler_ConcurrentAccessWithNamespaceChange(t *testing.T) {
	service := setupVersionTestService(t)

	// Channel to synchronize goroutines
	done := make(chan struct{})
	errors := make(chan error, 21)

	// Launch multiple goroutines requesting metadata with different versions
	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- struct{}{} }()

			version := "4.0"
			if idx%2 == 1 {
				version = "4.01"
			}

			req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
			req.Header.Set("OData-MaxVersion", version)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("goroutine %d: expected 200, got %d", idx, w.Code)
			}
		}(i)
	}

	// Launch goroutines requesting JSON metadata
	for i := 0; i < 10; i++ {
		go func(idx int) {
			defer func() { done <- struct{}{} }()

			req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
			req.Header.Set("OData-MaxVersion", "4.01")
			req.Header.Set("Accept", "application/json")
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("json goroutine %d: expected 200, got %d", idx, w.Code)
			}
		}(i)
	}

	// Concurrently change namespace (defensive test for misuse case)
	go func() {
		defer func() { done <- struct{}{} }()
		service.SetNamespace("TestNamespace")
		service.SetNamespace("AnotherNamespace")
		service.SetNamespace("ODataService") // Reset to default
	}()

	// Wait for all goroutines
	for i := 0; i < 21; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}
