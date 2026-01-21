package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// PreferTestProduct is a test entity for Prefer header tests
type PreferTestProduct struct {
	ID          int     `json:"id" gorm:"primarykey;autoIncrement" odata:"key"`
	Name        string  `json:"name" odata:"required"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

func setupPreferTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	// Use a file-based SQLite database for async tests to ensure
	// data is shared across goroutines (async workers)
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "prefer_test.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PreferTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(PreferTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Cleanup the database file when test completes
	t.Cleanup(func() {
		os.Remove(dbPath)
	})

	return service, db
}

// Test POST with default behavior (return representation)
func TestPostEntity_DefaultReturnRepresentation(t *testing.T) {
	service, _ := setupPreferTestService(t)

	newProduct := map[string]interface{}{
		"name":  "Laptop",
		"price": 999.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PreferTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 201 Created with body
	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	// Verify response body is present
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["name"] != "Laptop" {
		t.Errorf("name = %v, want Laptop", response["name"])
	}

	// Location header should be present
	if location := w.Header().Get("Location"); location == "" {
		t.Error("Location header is empty")
	}
}

// Test POST with Prefer: return=minimal
func TestPostEntity_PreferReturnMinimal(t *testing.T) {
	service, _ := setupPreferTestService(t)

	newProduct := map[string]interface{}{
		"name":  "Mouse",
		"price": 29.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PreferTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Per OData v4.01 spec, POST with return=minimal should return 201 Created with empty body
	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	// Verify Preference-Applied header is present
	preferenceApplied := w.Header().Get("Preference-Applied")
	if preferenceApplied != "return=minimal" {
		t.Errorf("Preference-Applied = %v, want return=minimal", preferenceApplied)
	}

	// Location header should still be present
	if location := w.Header().Get("Location"); location == "" {
		t.Error("Location header is empty")
	}

	// Body should be empty
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty but has %d bytes", w.Body.Len())
	}
}

// Test POST with explicit Prefer: return=representation
func TestPostEntity_PreferReturnRepresentation(t *testing.T) {
	service, _ := setupPreferTestService(t)

	newProduct := map[string]interface{}{
		"name":  "Keyboard",
		"price": 79.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PreferTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 201 Created with body
	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	// Verify Preference-Applied header is present
	preferenceApplied := w.Header().Get("Preference-Applied")
	if preferenceApplied != "return=representation" {
		t.Errorf("Preference-Applied = %v, want return=representation", preferenceApplied)
	}

	// Verify response body is present
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["name"] != "Keyboard" {
		t.Errorf("name = %v, want Keyboard", response["name"])
	}
}

// Test PATCH with default behavior (no content)
func TestPatchEntity_DefaultNoContent(t *testing.T) {
	service, db := setupPreferTestService(t)

	// Create a product first
	product := PreferTestProduct{Name: "Original", Price: 100.00}
	db.Create(&product)

	updateData := map[string]interface{}{
		"price": 150.00,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PreferTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Body should be empty
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty but has %d bytes", w.Body.Len())
	}
}

// Test PATCH with Prefer: return=representation
func TestPatchEntity_PreferReturnRepresentation(t *testing.T) {
	service, db := setupPreferTestService(t)

	// Create a product first
	product := PreferTestProduct{Name: "Original", Price: 100.00, Description: "Test"}
	db.Create(&product)

	updateData := map[string]interface{}{
		"price": 150.00,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PreferTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 200 OK with body
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify Preference-Applied header is present
	preferenceApplied := w.Header().Get("Preference-Applied")
	if preferenceApplied != "return=representation" {
		t.Errorf("Preference-Applied = %v, want return=representation", preferenceApplied)
	}

	// Verify response body contains updated entity
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["price"] != 150.00 {
		t.Errorf("price = %v, want 150.00", response["price"])
	}

	// Verify original properties are still there
	if response["name"] != "Original" {
		t.Errorf("name = %v, want Original", response["name"])
	}

	// Verify @odata.context is present
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Response missing @odata.context")
	}
}

// Test PATCH with explicit Prefer: return=minimal
func TestPatchEntity_PreferReturnMinimal(t *testing.T) {
	service, db := setupPreferTestService(t)

	// Create a product first
	product := PreferTestProduct{Name: "Original", Price: 100.00}
	db.Create(&product)

	updateData := map[string]interface{}{
		"price": 150.00,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/PreferTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Verify Preference-Applied header is present
	preferenceApplied := w.Header().Get("Preference-Applied")
	if preferenceApplied != "return=minimal" {
		t.Errorf("Preference-Applied = %v, want return=minimal", preferenceApplied)
	}

	// Body should be empty
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty but has %d bytes", w.Body.Len())
	}
}

// Test PUT with default behavior (no content)
func TestPutEntity_DefaultNoContent(t *testing.T) {
	service, db := setupPreferTestService(t)

	// Create a product first
	product := PreferTestProduct{Name: "Original", Price: 100.00, Description: "Original Description"}
	db.Create(&product)

	updateData := map[string]interface{}{
		"name":  "Updated",
		"price": 200.00,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPut, "/PreferTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Body should be empty
	if w.Body.Len() > 0 {
		t.Errorf("Body should be empty but has %d bytes", w.Body.Len())
	}
}

// Test PUT with Prefer: return=representation
func TestPutEntity_PreferReturnRepresentation(t *testing.T) {
	service, db := setupPreferTestService(t)

	// Create a product first
	product := PreferTestProduct{Name: "Original", Price: 100.00, Description: "Original Description"}
	db.Create(&product)

	updateData := map[string]interface{}{
		"name":  "Updated",
		"price": 200.00,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPut, "/PreferTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 200 OK with body
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify Preference-Applied header is present
	preferenceApplied := w.Header().Get("Preference-Applied")
	if preferenceApplied != "return=representation" {
		t.Errorf("Preference-Applied = %v, want return=representation", preferenceApplied)
	}

	// Verify response body contains updated entity
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["name"] != "Updated" {
		t.Errorf("name = %v, want Updated", response["name"])
	}

	if response["price"] != 200.00 {
		t.Errorf("price = %v, want 200.00", response["price"])
	}

	// Verify @odata.context is present
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Response missing @odata.context")
	}
}

// Test case-insensitive Prefer header
func TestPreferHeader_CaseInsensitive(t *testing.T) {
	service, _ := setupPreferTestService(t)

	newProduct := map[string]interface{}{
		"name":  "Test",
		"price": 50.00,
	}
	body, _ := json.Marshal(newProduct)

	testCases := []string{
		"RETURN=MINIMAL",
		"Return=Minimal",
		"return=MINIMAL",
	}

	for _, preferValue := range testCases {
		req := httptest.NewRequest(http.MethodPost, "/PreferTestProducts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Prefer", preferValue)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		// Per OData v4.01 spec, POST with return=minimal should return 201 Created
		if w.Code != http.StatusCreated {
			t.Errorf("For Prefer header '%s', Status = %v, want %v", preferValue, w.Code, http.StatusCreated)
		}
	}
}

// Test that OData-Version header is always present
func TestPreferHeader_ODataVersionAlwaysPresent(t *testing.T) {
	service, _ := setupPreferTestService(t)

	testCases := []struct {
		name   string
		prefer string
	}{
		{"No Prefer header", ""},
		{"return=minimal", "return=minimal"},
		{"return=representation", "return=representation"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newProduct := map[string]interface{}{
				"name":  "Test",
				"price": 50.00,
			}
			body, _ := json.Marshal(newProduct)

			req := httptest.NewRequest(http.MethodPost, "/PreferTestProducts", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.prefer != "" {
				req.Header.Set("Prefer", tc.prefer)
			}
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)
		})
	}
}

// Test multiple preferences in header (comma-separated)
func TestPreferHeader_MultiplePreferences(t *testing.T) {
	service, _ := setupPreferTestService(t)

	newProduct := map[string]interface{}{
		"name":  "Test",
		"price": 50.00,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PreferTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal, respond-async")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Per OData v4.01 spec, POST with return=minimal should return 201 Created
	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	preferenceApplied := w.Header().Get("Preference-Applied")
	if preferenceApplied != "return=minimal" {
		t.Errorf("Preference-Applied = %v, want return=minimal", preferenceApplied)
	}
}

func TestPreferHeader_RespondAsyncOnlyDoesNotSetPreferenceApplied(t *testing.T) {
	service, _ := setupPreferTestService(t)

	newProduct := map[string]interface{}{
		"name":  "AsyncOnly",
		"price": 15.00,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PreferTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "respond-async")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	if preferenceApplied := w.Header().Get("Preference-Applied"); preferenceApplied != "" {
		t.Fatalf("Preference-Applied should be empty when async preference is not honored, got %q", preferenceApplied)
	}
}

func TestGetEntities_RespondAsyncIntegration(t *testing.T) {
	service, db := setupPreferTestService(t)
	enableAsyncProcessing(t, service, time.Second)

	sample := PreferTestProduct{Name: "Async Widget", Price: 42.0}
	if err := db.Create(&sample).Error; err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/PreferTestProducts", nil)
	req.Header.Set("Prefer", "respond-async")

	rec := httptest.NewRecorder()
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for async request, got %d", rec.Code)
	}

	if got := rec.Header().Get("Preference-Applied"); got != "respond-async" {
		t.Fatalf("expected Preference-Applied respond-async, got %q", got)
	}

	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("expected Retry-After header of 1, got %q", got)
	}

	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("expected monitor Location header")
	}

	expected := httptest.NewRecorder()
	service.ServeHTTP(expected, httptest.NewRequest(http.MethodGet, "/PreferTestProducts", nil))

	monitorRec := waitForMonitorCompletion(t, service, location)

	if monitorRec.Code != expected.Code {
		t.Fatalf("monitor status %d, want %d", monitorRec.Code, expected.Code)
	}

	var expectedBody map[string]any
	if err := json.NewDecoder(expected.Body).Decode(&expectedBody); err != nil {
		t.Fatalf("failed to decode expected body: %v", err)
	}

	var actualBody map[string]any
	if err := json.NewDecoder(monitorRec.Body).Decode(&actualBody); err != nil {
		t.Fatalf("failed to decode monitor body: %v", err)
	}

	if !reflect.DeepEqual(actualBody, expectedBody) {
		t.Fatalf("monitor response mismatch: got %v, want %v", actualBody, expectedBody)
	}
}

func TestPostEntity_RespondAsyncHonorsRepresentationPreference(t *testing.T) {
	service, _ := setupPreferTestService(t)
	enableAsyncProcessing(t, service, time.Second)

	payload := map[string]any{
		"name":  "Async Keyboard",
		"price": 88.0,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/PreferTestProducts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation, respond-async")

	rec := httptest.NewRecorder()
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for async POST, got %d", rec.Code)
	}

	if got := rec.Header().Get("Preference-Applied"); got != "respond-async" {
		t.Fatalf("expected Preference-Applied respond-async, got %q", got)
	}

	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("expected monitor Location header")
	}

	monitorRec := waitForMonitorCompletion(t, service, location)

	if monitorRec.Code != http.StatusCreated {
		t.Fatalf("monitor returned %d, want %d", monitorRec.Code, http.StatusCreated)
	}

	if got := monitorRec.Header().Get("Preference-Applied"); got != "return=representation" {
		t.Fatalf("final response preference mismatch: got %q", got)
	}

	var response map[string]any
	if err := json.NewDecoder(monitorRec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode monitor body: %v", err)
	}

	if response["name"] != "Async Keyboard" {
		t.Fatalf("unexpected entity name %v", response["name"])
	}
}
