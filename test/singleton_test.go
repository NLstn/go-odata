package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// CompanyInfo represents a singleton entity for company information
type CompanyInfo struct {
	ID          uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string `json:"Name" gorm:"not null" odata:"required"`
	CEO         string `json:"CEO" gorm:"not null"`
	Founded     int    `json:"Founded"`
	HeadQuarter string `json:"HeadQuarter"`
	Version     int    `json:"Version" gorm:"default:1" odata:"etag"`
}

// setupSingletonTestDB creates a test database with singleton data
func setupSingletonTestDB(t *testing.T) (*gorm.DB, *odata.Service) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&CompanyInfo{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed with singleton data
	company := CompanyInfo{
		ID:          1,
		Name:        "Contoso Corporation",
		CEO:         "John Doe",
		Founded:     1990,
		HeadQuarter: "Seattle, WA",
		Version:     1,
	}
	if err := db.Create(&company).Error; err != nil {
		t.Fatalf("Failed to seed database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterSingleton(&CompanyInfo{}, "Company"); err != nil {
		t.Fatalf("Failed to register singleton: %v", err)
	}

	return db, service
}

// TestSingletonRegistration tests that singleton registration works correctly
func TestSingletonRegistration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&CompanyInfo{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	err = service.RegisterSingleton(&CompanyInfo{}, "Company")
	if err != nil {
		t.Fatalf("Failed to register singleton: %v", err)
	}
}

// TestSingletonGet tests GET request for singleton
func TestSingletonGet(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/Company", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the @odata.context
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("Expected @odata.context in response")
	}
	expectedContext := "http://example.com/$metadata#Company/$entity"
	if context != expectedContext {
		t.Errorf("Expected context '%s', got '%s'", expectedContext, context)
	}

	// Verify entity data
	if response["Name"].(string) != "Contoso Corporation" {
		t.Errorf("Expected Name 'Contoso Corporation', got %v", response["Name"])
	}
	if response["CEO"].(string) != "John Doe" {
		t.Errorf("Expected CEO 'John Doe', got %v", response["CEO"])
	}
}

// TestSingletonGetNotFound tests GET request when singleton doesn't exist
func TestSingletonGetNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&CompanyInfo{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterSingleton(&CompanyInfo{}, "Company"); err != nil {
		t.Fatalf("Failed to register singleton: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/Company", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSingletonPatch tests PATCH request for singleton
func TestSingletonPatch(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	patchData := map[string]interface{}{
		"CEO": "Jane Smith",
	}
	body, _ := json.Marshal(patchData)

	req := httptest.NewRequest(http.MethodPatch, "/Company", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the CEO was updated
	if response["CEO"].(string) != "Jane Smith" {
		t.Errorf("Expected CEO 'Jane Smith', got %v", response["CEO"])
	}
	// Verify other fields remain unchanged
	if response["Name"].(string) != "Contoso Corporation" {
		t.Errorf("Expected Name 'Contoso Corporation', got %v", response["Name"])
	}
}

// TestSingletonPatchMinimal tests PATCH with Prefer: return=minimal
func TestSingletonPatchMinimal(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	patchData := map[string]interface{}{
		"CEO": "Bob Johnson",
	}
	body, _ := json.Marshal(patchData)

	req := httptest.NewRequest(http.MethodPatch, "/Company", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify Preference-Applied header
	preferenceApplied := w.Header().Get("Preference-Applied")
	if preferenceApplied != "return=minimal" {
		t.Errorf("Expected Preference-Applied 'return=minimal', got '%s'", preferenceApplied)
	}
}

// TestSingletonPut tests PUT request for singleton
func TestSingletonPut(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	putData := CompanyInfo{
		Name:        "Contoso Ltd",
		CEO:         "Alice Brown",
		Founded:     1995,
		HeadQuarter: "New York, NY",
	}
	body, _ := json.Marshal(putData)

	req := httptest.NewRequest(http.MethodPut, "/Company", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify all fields were updated
	if response["Name"].(string) != "Contoso Ltd" {
		t.Errorf("Expected Name 'Contoso Ltd', got %v", response["Name"])
	}
	if response["CEO"].(string) != "Alice Brown" {
		t.Errorf("Expected CEO 'Alice Brown', got %v", response["CEO"])
	}
	if int(response["Founded"].(float64)) != 1995 {
		t.Errorf("Expected Founded 1995, got %v", response["Founded"])
	}
}

// TestSingletonETag tests ETag support for singleton
func TestSingletonETag(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	// First, GET the entity to get its ETag
	req := httptest.NewRequest(http.MethodGet, "/Company", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET failed: %d", w.Code)
	}

	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("Expected ETag header")
	}

	// Try to update with correct ETag
	patchData := map[string]interface{}{
		"CEO": "Charlie Davis",
	}
	body, _ := json.Marshal(patchData)

	req = httptest.NewRequest(http.MethodPatch, "/Company", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", etag)
	req.Header.Set("Prefer", "return=minimal")
	w = httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 with correct ETag, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSingletonETagMismatch tests ETag mismatch for singleton
func TestSingletonETagMismatch(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	patchData := map[string]interface{}{
		"CEO": "Wrong Update",
	}
	body, _ := json.Marshal(patchData)

	req := httptest.NewRequest(http.MethodPatch, "/Company", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", `W/"wrong-etag"`)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusPreconditionFailed {
		t.Errorf("Expected status 412 with wrong ETag, got %d", w.Code)
	}
}

// TestSingletonInServiceDocument tests that singleton appears in service document
func TestSingletonInServiceDocument(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected 'value' array in service document")
	}

	// Find the singleton entry
	foundSingleton := false
	for _, item := range value {
		entry := item.(map[string]interface{})
		if entry["name"] == "Company" && entry["kind"] == "Singleton" {
			foundSingleton = true
			if entry["url"] != "Company" {
				t.Errorf("Expected url 'Company', got %v", entry["url"])
			}
			break
		}
	}

	if !foundSingleton {
		t.Error("Singleton 'Company' not found in service document")
	}
}

// TestSingletonInMetadataXML tests that singleton appears in XML metadata
func TestSingletonInMetadataXML(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	metadata := w.Body.String()

	// Check for singleton definition
	if !containsStr(metadata, `<Singleton Name="Company"`) {
		t.Error("Singleton definition not found in XML metadata")
	}
	if !containsStr(metadata, `Type="ODataService.CompanyInfo"`) {
		t.Error("Singleton type not found in XML metadata")
	}
}

// TestSingletonInMetadataJSON tests that singleton appears in JSON metadata
func TestSingletonInMetadataJSON(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &metadata); err != nil {
		t.Fatalf("Failed to parse JSON metadata: %v", err)
	}

	// Navigate to ODataService namespace then Container
	odataService, ok := metadata["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatal("ODataService not found in metadata")
	}

	container, ok := odataService["Container"].(map[string]interface{})
	if !ok {
		t.Fatal("Container not found in ODataService")
	}

	// Check for singleton
	company, ok := container["Company"].(map[string]interface{})
	if !ok {
		t.Fatal("Company singleton not found in Container")
	}

	// Verify it's a singleton (no $Collection property)
	if _, hasCollection := company["$Collection"]; hasCollection {
		t.Error("Singleton should not have $Collection property")
	}

	// Verify Type
	if company["$Type"] != "ODataService.CompanyInfo" {
		t.Errorf("Expected $Type 'ODataService.CompanyInfo', got %v", company["$Type"])
	}
}

// TestSingletonMethodNotAllowed tests unsupported HTTP methods
func TestSingletonMethodNotAllowed(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	tests := []struct {
		name   string
		method string
	}{
		{"POST not allowed", http.MethodPost},
		{"DELETE not allowed", http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/Company", nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", tt.method, w.Code)
			}
		})
	}
}

// TestSingletonOptions tests OPTIONS request
func TestSingletonOptions(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	req := httptest.NewRequest(http.MethodOptions, "/Company", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	allow := w.Header().Get("Allow")
	if !containsStr(allow, "GET") || !containsStr(allow, "PATCH") || !containsStr(allow, "PUT") {
		t.Errorf("Expected Allow header to contain GET, PATCH, PUT, got '%s'", allow)
	}
	if containsStr(allow, "DELETE") || containsStr(allow, "POST") {
		t.Errorf("Allow header should not contain DELETE or POST for singleton, got '%s'", allow)
	}
}

// TestSingletonHead tests HEAD request
func TestSingletonHead(t *testing.T) {
	_, service := setupSingletonTestDB(t)

	req := httptest.NewRequest(http.MethodHead, "/Company", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// HEAD should not return body
	if w.Body.Len() > 0 {
		t.Error("HEAD request should not return a body")
	}
}
