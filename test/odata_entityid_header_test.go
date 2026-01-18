package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// EntityIdTestProduct is a test entity for OData-EntityId header tests
type EntityIdTestProduct struct {
	ID          int     `json:"id" gorm:"primarykey;autoIncrement" odata:"key"`
	Name        string  `json:"name" odata:"required"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

// EntityIdTestCompositeKey is a test entity with composite keys for OData-EntityId header tests
type EntityIdTestCompositeKey struct {
	ProductID   int    `json:"productID" gorm:"primaryKey" odata:"key"`
	LanguageKey string `json:"languageKey" gorm:"primaryKey" odata:"key"`
	Name        string `json:"name" odata:"required"`
	Description string `json:"description"`
}

func setupEntityIdTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityIdTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(EntityIdTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func setupEntityIdCompositeKeyTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&EntityIdTestCompositeKey{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(EntityIdTestCompositeKey{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestODataEntityIdHeader_POST tests OData-EntityId header for POST with Prefer: return=minimal
func TestODataEntityIdHeader_POST(t *testing.T) {
	service, _ := setupEntityIdTestService(t)

	newProduct := map[string]interface{}{
		"name":  "Test Product",
		"price": 99.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/EntityIdTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Check OData-EntityId header is present
	// Access directly with exact casing (OData-EntityId with capital 'D')
	//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
	entityIdValues := w.Header()["OData-EntityId"]
	if len(entityIdValues) == 0 {
		t.Error("OData-EntityId header is missing")
	}
	entityId := entityIdValues[0]

	// Verify format: http://example.com/EntityIdTestProducts(1)
	expectedFormat := "http://example.com/EntityIdTestProducts(1)"
	if entityId != expectedFormat {
		t.Errorf("OData-EntityId = %v, want %v", entityId, expectedFormat)
	}

	// Verify Location header is also present (should match OData-EntityId)
	location := w.Header().Get("Location")
	if location != entityId {
		t.Errorf("Location = %v, should match OData-EntityId = %v", location, entityId)
	}
}

// TestODataEntityIdHeader_POST_CompositeKey tests OData-EntityId header for POST with composite key
func TestODataEntityIdHeader_POST_CompositeKey(t *testing.T) {
	service, _ := setupEntityIdCompositeKeyTestService(t)

	newProduct := map[string]interface{}{
		"productID":   1,
		"languageKey": "EN",
		"name":        "Test Product",
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/EntityIdTestCompositeKeys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
	}

	// Check OData-EntityId header is present
	// Access directly with exact casing (OData-EntityId with capital 'D')
	//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
	entityIdValues := w.Header()["OData-EntityId"]
	if len(entityIdValues) == 0 {
		t.Error("OData-EntityId header is missing")
	}
	entityId := entityIdValues[0]

	// Verify format: http://example.com/EntityIdTestCompositeKeys(productID=1,languageKey='EN')
	expectedFormat := "http://example.com/EntityIdTestCompositeKeys(productID=1,languageKey='EN')"
	if entityId != expectedFormat {
		t.Errorf("OData-EntityId = %v, want %v", entityId, expectedFormat)
	}
}

// TestODataEntityIdHeader_POST_WithRepresentation tests that OData-EntityId is present with 201
// Note: While the OData spec only REQUIRES OData-EntityId for 204 responses, including it
// in 201 responses is a best practice for consistency and provides the canonical entity-id to clients.
func TestODataEntityIdHeader_POST_WithRepresentation(t *testing.T) {
	service, _ := setupEntityIdTestService(t)

	newProduct := map[string]interface{}{
		"name":  "Test Product",
		"price": 99.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/EntityIdTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// Default behavior: return=representation
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 201 Created
	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	// OData-EntityId should be present (best practice, though only required for 204 by spec)
	// Access directly with exact casing (OData-EntityId with capital 'D')
	//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
	entityIdValues := w.Header()["OData-EntityId"]
	if len(entityIdValues) == 0 {
		t.Error("OData-EntityId header is missing")
	}
	entityId := entityIdValues[0]

	// Verify format: http://example.com/EntityIdTestProducts(1)
	expectedFormat := "http://example.com/EntityIdTestProducts(1)"
	if entityId != expectedFormat {
		t.Errorf("OData-EntityId = %v, want %v", entityId, expectedFormat)
	}

	// Location header should also be present and match
	location := w.Header().Get("Location")
	if location != entityId {
		t.Errorf("Location = %v, should match OData-EntityId = %v", location, entityId)
	}
}

// TestODataEntityIdHeader_PATCH tests OData-EntityId header for PATCH
func TestODataEntityIdHeader_PATCH(t *testing.T) {
	service, db := setupEntityIdTestService(t)

	// Create a product first
	product := EntityIdTestProduct{Name: "Original", Price: 100.00}
	db.Create(&product)

	updateData := map[string]interface{}{
		"price": 150.00,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/EntityIdTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content (default for PATCH)
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Check OData-EntityId header is present
	// Access directly with exact casing (OData-EntityId with capital 'D')
	//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
	entityIdValues := w.Header()["OData-EntityId"]
	if len(entityIdValues) == 0 {
		t.Error("OData-EntityId header is missing")
	}
	entityId := entityIdValues[0]

	// Verify format
	expectedFormat := "http://example.com/EntityIdTestProducts(1)"
	if entityId != expectedFormat {
		t.Errorf("OData-EntityId = %v, want %v", entityId, expectedFormat)
	}
}

// TestODataEntityIdHeader_PATCH_WithRepresentation tests OData-EntityId with 200 response
// Note: OData-EntityId is only REQUIRED by spec for 204 responses. For 200 OK responses
// (when returning representation), the spec doesn't require it, so we verify it's not included.
func TestODataEntityIdHeader_PATCH_WithRepresentation(t *testing.T) {
	service, db := setupEntityIdTestService(t)

	// Create a product first
	product := EntityIdTestProduct{Name: "Original", Price: 100.00}
	db.Create(&product)

	updateData := map[string]interface{}{
		"price": 150.00,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest(http.MethodPatch, "/EntityIdTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 200 OK with representation
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// OData-EntityId should NOT be present when returning representation (200 OK)
	// The spec only requires it for 204 No Content responses
	// Access directly with exact casing (OData-EntityId with capital 'D')
	//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
	entityIdValues := w.Header()["OData-EntityId"]
	if len(entityIdValues) > 0 {
		t.Errorf("OData-EntityId should not be present with 200 response, got %v", entityIdValues[0])
	}
}

// TestODataEntityIdHeader_PUT tests OData-EntityId header for PUT
func TestODataEntityIdHeader_PUT(t *testing.T) {
	service, db := setupEntityIdTestService(t)

	// Create a product first
	product := EntityIdTestProduct{Name: "Original", Price: 100.00}
	db.Create(&product)

	replacementData := map[string]interface{}{
		"name":  "Replaced",
		"price": 200.00,
	}
	body, _ := json.Marshal(replacementData)

	req := httptest.NewRequest(http.MethodPut, "/EntityIdTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content (default for PUT)
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Check OData-EntityId header is present
	// Access directly with exact casing (OData-EntityId with capital 'D')
	//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
	entityIdValues := w.Header()["OData-EntityId"]
	if len(entityIdValues) == 0 {
		t.Error("OData-EntityId header is missing")
	}
	entityId := entityIdValues[0]

	// Verify format
	expectedFormat := "http://example.com/EntityIdTestProducts(1)"
	if entityId != expectedFormat {
		t.Errorf("OData-EntityId = %v, want %v", entityId, expectedFormat)
	}
}

// TestODataEntityIdHeader_PUT_CompositeKey tests OData-EntityId header for PUT with composite key
func TestODataEntityIdHeader_PUT_CompositeKey(t *testing.T) {
	service, db := setupEntityIdCompositeKeyTestService(t)

	// Create a product first
	product := EntityIdTestCompositeKey{ProductID: 1, LanguageKey: "EN", Name: "Original"}
	db.Create(&product)

	replacementData := map[string]interface{}{
		"name": "Replaced",
	}
	body, _ := json.Marshal(replacementData)

	req := httptest.NewRequest(http.MethodPut, "/EntityIdTestCompositeKeys(productID=1,languageKey='EN')", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 204 No Content (default for PUT)
	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Check OData-EntityId header is present
	// Access directly with exact casing (OData-EntityId with capital 'D')
	//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
	entityIdValues := w.Header()["OData-EntityId"]
	if len(entityIdValues) == 0 {
		t.Error("OData-EntityId header is missing")
	}
	entityId := entityIdValues[0]

	// Verify format
	expectedFormat := "http://example.com/EntityIdTestCompositeKeys(productID=1,languageKey='EN')"
	if entityId != expectedFormat {
		t.Errorf("OData-EntityId = %v, want %v", entityId, expectedFormat)
	}
}
